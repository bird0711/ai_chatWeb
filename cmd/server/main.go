package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"ai_chat/internal/ai"
	"ai_chat/internal/app"
	apphttp "ai_chat/internal/http"
	"ai_chat/internal/store"

	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mysqlDSN := getenv("MYSQL_DSN", "")
	if mysqlDSN == "" {
		mysqlConfig := store.MySQLConfig{
			User:     getenv("MYSQL_USER", "root"),
			Password: getenv("MYSQL_PASSWORD", "4399"),
			Host:     getenv("MYSQL_HOST", "127.0.0.1"),
			Port:     getenv("MYSQL_PORT", "3306"),
			Database: getenv("MYSQL_DATABASE", "ai_chat"),
		}
		log.Printf("ensuring mysql database at %s", mysqlConfig.SafeAddr())
		if err := store.EnsureMySQLDatabase(ctx, mysqlConfig); err != nil {
			log.Fatalf("ensure mysql database: %v", err)
		}
		mysqlDSN = mysqlConfig.DSN()
	} else {
		log.Printf("using MYSQL_DSN from environment")
	}
	mysqlStore, err := store.OpenMySQL(mysqlDSN)
	if err != nil {
		log.Fatalf("open mysql: %v", err)
	}
	defer func() {
		if err := mysqlStore.Close(); err != nil {
			log.Printf("error closing mysqlStore: %v", err)
		}
	}()
	if err := mysqlStore.Migrate(ctx); err != nil {
		log.Fatalf("migrate mysql: %v", err)
	}

	redisDB, err := strconv.Atoi(getenv("REDIS_DB", "0"))
	if err != nil {
		log.Fatalf("parse REDIS_DB: %v", err)
	}
	redisAddr := getenv("REDIS_ADDR", "127.0.0.1:6379")
	log.Printf("checking redis at %s db %d", redisAddr, redisDB)
	redisClient, err := store.OpenRedis(ctx, redisAddr, getenv("REDIS_PASSWORD", "4399"), redisDB)
	if err != nil {
		log.Fatalf("open redis: %v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("error closing redisClient: %v", err)
		}
	}()

	services := app.NewServices(mysqlStore, ai.NewOpenAICompatibleClient())
	router := apphttp.NewRouter(services, redisPinger{client: redisClient})

	addr := getenv("ADDR", ":8080")
	log.Printf("listening on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("run server: %v", err)
	}
}

type redisPinger struct {
	client *redis.Client
}

func (p redisPinger) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
