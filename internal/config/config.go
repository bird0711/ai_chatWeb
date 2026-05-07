package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr         string
	MySQLDSN     string
	MySQLUser    string
	MySQLPassword string
	MySQLHost    string
	MySQLPort    string
	MySQLDatabase string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	ChatFileDir string
	UploadDir   string
	StaticDir   string
	TemplateGlob string

	ModelAPITimeout            time.Duration
	ModelAPITLSHandshakeTimeout time.Duration
	ModelAPIRetryAttempts      int
	ModelAPIRetryBackoff       time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Addr:          getenv("ADDR", ":8080"),
		MySQLDSN:      strings.TrimSpace(os.Getenv("MYSQL_DSN")),
		MySQLUser:     getenv("MYSQL_USER", "root"),
		MySQLPassword: getenv("MYSQL_PASSWORD", "4399"),
		MySQLHost:     getenv("MYSQL_HOST", "127.0.0.1"),
		MySQLPort:     getenv("MYSQL_PORT", "3306"),
		MySQLDatabase: getenv("MYSQL_DATABASE", "ai_chat"),

		RedisAddr:     getenv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: getenv("REDIS_PASSWORD", "4399"),

		ChatFileDir:  getenv("CHAT_FILE_DIR", "data/chat-files"),
		UploadDir:    getenv("UPLOAD_DIR", "uploads"),
		StaticDir:    getenv("STATIC_DIR", "web/static"),
		TemplateGlob: getenv("TEMPLATE_GLOB", "web/templates/*.html"),
	}

	redisDB, err := atoiEnv("REDIS_DB", 0)
	if err != nil {
		return Config{}, fmt.Errorf("parse REDIS_DB: %w", err)
	}
	cfg.RedisDB = redisDB

	modelTimeoutSeconds, err := atoiEnv("MODEL_API_TIMEOUT_SECONDS", 90)
	if err != nil {
		return Config{}, fmt.Errorf("parse MODEL_API_TIMEOUT_SECONDS: %w", err)
	}
	cfg.ModelAPITimeout = time.Duration(modelTimeoutSeconds) * time.Second

	tlsHandshakeSeconds, err := atoiEnv("MODEL_API_TLS_HANDSHAKE_TIMEOUT_SECONDS", 30)
	if err != nil {
		return Config{}, fmt.Errorf("parse MODEL_API_TLS_HANDSHAKE_TIMEOUT_SECONDS: %w", err)
	}
	cfg.ModelAPITLSHandshakeTimeout = time.Duration(tlsHandshakeSeconds) * time.Second

	retryAttempts, err := atoiEnv("MODEL_API_RETRY_ATTEMPTS", 2)
	if err != nil {
		return Config{}, fmt.Errorf("parse MODEL_API_RETRY_ATTEMPTS: %w", err)
	}
	cfg.ModelAPIRetryAttempts = retryAttempts

	retryBackoffMS, err := atoiEnv("MODEL_API_RETRY_BACKOFF_MS", 800)
	if err != nil {
		return Config{}, fmt.Errorf("parse MODEL_API_RETRY_BACKOFF_MS: %w", err)
	}
	cfg.ModelAPIRetryBackoff = time.Duration(retryBackoffMS) * time.Millisecond

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
func (c Config) Validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return fmt.Errorf("ADDR is required")
	}
	if c.MySQLDSN == "" {
		if strings.TrimSpace(c.MySQLUser) == "" {
			return fmt.Errorf("MYSQL_USER is required when MYSQL_DSN is empty")
		}
		if strings.TrimSpace(c.MySQLHost) == "" {
			return fmt.Errorf("MYSQL_HOST is required when MYSQL_DSN is empty")
		}
		if strings.TrimSpace(c.MySQLPort) == "" {
			return fmt.Errorf("MYSQL_PORT is required when MYSQL_DSN is empty")
		}
		if strings.TrimSpace(c.MySQLDatabase) == "" {
			return fmt.Errorf("MYSQL_DATABASE is required when MYSQL_DSN is empty")
		}
	}
	if strings.TrimSpace(c.RedisAddr) == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}
	if strings.TrimSpace(c.ChatFileDir) == "" {
		return fmt.Errorf("CHAT_FILE_DIR is required")
	}
	if strings.TrimSpace(c.UploadDir) == "" {
		return fmt.Errorf("UPLOAD_DIR is required")
	}
	if strings.TrimSpace(c.StaticDir) == "" {
		return fmt.Errorf("STATIC_DIR is required")
	}
	if strings.TrimSpace(c.TemplateGlob) == "" {
		return fmt.Errorf("TEMPLATE_GLOB is required")
	}
	return nil
}
func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func atoiEnv(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	return strconv.Atoi(raw)
}
