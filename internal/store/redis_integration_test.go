//go:build integration
// +build integration

package store

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"
)

func getTestRedisConfig() (string, string, int) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6380"
	}
	password := os.Getenv("REDIS_PASSWORD")
	if password == "" {
		password = "4399"
	}
	db := 1
	if raw := os.Getenv("REDIS_DB"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			db = parsed
		}
	}
	return addr, password, db
}

func requireRedis(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	addr, password, db := getTestRedisConfig()
	client, err := OpenRedis(ctx, addr, password, db)
	if err != nil {
		t.Fatalf("Redis not available: %v", err)
	}

	if err := client.FlushDB(ctx).Err(); err != nil {
		_ = client.Close()
		t.Fatalf("Failed to clear Redis test db: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cleanupCancel()
		_ = client.FlushDB(cleanupCtx).Err()
		_ = client.Close()
	})
}

func TestRedisOpenAndPing(t *testing.T) {
	requireRedis(t)
}

func TestRedisSetAndGet(t *testing.T) {
	requireRedis(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	addr, password, db := getTestRedisConfig()
	client, err := OpenRedis(ctx, addr, password, db)
	if err != nil {
		t.Fatalf("OpenRedis failed: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	if err := client.Set(ctx, "integration:test:key", "hello", time.Minute).Err(); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := client.Get(ctx, "integration:test:key").Result()
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != "hello" {
		t.Fatalf("unexpected Redis value: got %q want %q", got, "hello")
	}
}
