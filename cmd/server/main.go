package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai_chat/internal/ai"
	"ai_chat/internal/app"
	"ai_chat/internal/config"
	apphttp "ai_chat/internal/http"
	"ai_chat/internal/store"
	"github.com/gin-gonic/gin"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	mysqlStore, err := openMySQL(ctx, cfg)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer func() {
		if err := mysqlStore.Close(); err != nil {
			log.Printf("error closing mysqlStore: %v", err)
		}
	}()

	redisClient, err := openRedis(ctx, cfg)
	if err != nil {
		log.Fatalf("%v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("error closing redisClient: %v", err)
		}
	}()

	router := newRouter(mysqlStore, redisClient)

	if err := runServer(cfg, router); err != nil {
		log.Fatalf("run server: %v", err)
	}

}

func openMySQL(ctx context.Context, cfg config.Config) (*store.MySQLStore, error) {
	mysqlDSN := cfg.MySQLDSN
	if mysqlDSN == "" {
		mysqlConfig := store.MySQLConfig{
			User:     cfg.MySQLUser,
			Password: cfg.MySQLPassword,
			Host:     cfg.MySQLHost,
			Port:     cfg.MySQLPort,
			Database: cfg.MySQLDatabase,
		}
		log.Printf("ensuring mysql database at %s", mysqlConfig.SafeAddr())
		if err := store.EnsureMySQLDatabase(ctx, mysqlConfig); err != nil {
			return nil, fmt.Errorf("ensure mysql database: %w", err)
		}
		mysqlDSN = mysqlConfig.DSN()
	} else {
		log.Printf("using MYSQL_DSN from environment")
	}

	mysqlStore, err := store.OpenMySQL(mysqlDSN)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	if err := mysqlStore.Migrate(ctx); err != nil {
		_ = mysqlStore.Close()
		return nil, fmt.Errorf("migrate mysql: %w", err)
	}
	return mysqlStore, nil
}

func openRedis(ctx context.Context, cfg config.Config) (*redis.Client, error) {
	log.Printf("checking redis at %s db %d", cfg.RedisAddr, cfg.RedisDB)
	redisClient, err := store.OpenRedis(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		return nil, fmt.Errorf("open redis: %w", err)
	}
	return redisClient, nil
}

func newRouter(mysqlStore *store.MySQLStore, redisClient *redis.Client) *gin.Engine {
	services := app.NewServices(mysqlStore, ai.NewOpenAICompatibleClient())
	return apphttp.NewRouter(services, redisPinger{client: redisClient})
}

func runServer(cfg config.Config, handler http.Handler) error {
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", cfg.Addr)
		errCh <- server.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	case sig := <-sigCh:
		log.Printf("shutdown signal received: %s", sig)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		err := <-errCh
		log.Printf("server exit result: %v", err)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server stopped with error: %w", err)
		}
		return nil

	}
}

type redisPinger struct {
	client *redis.Client
}

func (p redisPinger) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}
