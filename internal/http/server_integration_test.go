//go:build integration
// +build integration

package http

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ai_chat/internal/app"
	"ai_chat/internal/domain"
	"ai_chat/internal/store"

	"github.com/redis/go-redis/v9"
)

type integrationEnv struct {
	router      http.Handler
	store       *store.MySQLStore
	redisClient *redis.Client
	baseCtx     context.Context
}

type redisPingerAdapter struct {
	client *redis.Client
}

func (p redisPingerAdapter) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

func requireHTTPIntegrationEnv(t *testing.T) integrationEnv {
	t.Helper()

	ctx := context.Background()
	mysqlCfg := store.MySQLConfig{
		User:     getenv("MYSQL_USER", "root"),
		Password: getenv("MYSQL_PASSWORD", "4399"),
		Host:     getenv("MYSQL_HOST", "localhost"),
		Port:     getenv("MYSQL_PORT", "3307"),
		Database: "ai_chat_http_test",
	}
	if err := store.EnsureMySQLDatabase(ctx, mysqlCfg); err != nil {
		t.Fatalf("ensure mysql database: %v", err)
	}
	mysqlStore, err := store.OpenMySQL(mysqlCfg.DSN())
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	if err := mysqlStore.Migrate(ctx); err != nil {
		_ = mysqlStore.Close()
		t.Fatalf("initial migrate mysql: %v", err)
	}
	if err := mysqlStore.ClearAllTables(ctx); err != nil {
		_ = mysqlStore.Close()
		t.Fatalf("clear mysql tables: %v", err)
	}
	if err := mysqlStore.Migrate(ctx); err != nil {
		_ = mysqlStore.Close()
		t.Fatalf("migrate mysql: %v", err)
	}

	redisAddr := getenv("REDIS_ADDR", "localhost:6380")
	redisPassword := getenv("REDIS_PASSWORD", "4399")
	redisDB := 1
	redisRawDB := strings.TrimSpace(os.Getenv("REDIS_DB"))
	if redisRawDB != "" {
		var parsed int
		_, err := fmt.Sscanf(redisRawDB, "%d", &parsed)
		if err != nil {
			_ = mysqlStore.Close()
			t.Fatalf("parse REDIS_DB: %v", err)
		}
		redisDB = parsed
	}
	redisClient, err := store.OpenRedis(ctx, redisAddr, redisPassword, redisDB)
	if err != nil {
		_ = mysqlStore.Close()
		t.Fatalf("open redis: %v", err)
	}
	if err := redisClient.FlushDB(ctx).Err(); err != nil {
		_ = redisClient.Close()
		_ = mysqlStore.Close()
		t.Fatalf("clear redis: %v", err)
	}

	tempDir := t.TempDir()
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	t.Setenv("UPLOAD_DIR", filepath.Join(tempDir, "uploads"))
	t.Setenv("CHAT_FILE_DIR", filepath.Join(tempDir, "chat-files"))
	t.Setenv("COOKIE_SECURE", "false")

	router := NewRouter(app.NewServices(mysqlStore, fakeAI{}), redisPingerAdapter{client: redisClient})

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = redisClient.FlushDB(cleanupCtx).Err()
		_ = redisClient.Close()
		_ = mysqlStore.ClearAllTables(cleanupCtx)
		_ = mysqlStore.Close()
	})

	return integrationEnv{
		router:      router,
		store:       mysqlStore,
		redisClient: redisClient,
		baseCtx:     ctx,
	}
}

func TestHTTPIntegrationMainFlowWithRealDeps(t *testing.T) {
	env := requireHTTPIntegrationEnv(t)
	client := newTestClient()

	assertStatus(t, env.router, client, http.MethodGet, "/login", nil, http.StatusOK)
	assertStatusWithCSRF(t, env.router, client, http.MethodPost, "/register", url.Values{
		"email":    {"integration@example.test"},
		"password": {"secret1"},
	}, http.StatusFound)

	assertStatusWithCSRF(t, env.router, client, http.MethodPost, "/chats", url.Values{
		"name": {"Integration Chat"},
	}, http.StatusFound)

	assertStatusWithCSRF(t, env.router, client, http.MethodPost, "/settings/model", url.Values{
		"name":          {"Integration API"},
		"provider":      {"openai-compatible"},
		"base_url":      {"https://example.test/v1"},
		"api_key":       {"test-key"},
		"default_model": {"test-model"},
		"models":        {"test-model\nbackup-model"},
	}, http.StatusOK)

	user, err := env.store.GetUserByEmail(env.baseCtx, "integration@example.test")
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	userCtx := domain.WithUserID(env.baseCtx, user.ID)
	config, err := env.store.GetModelConfig(userCtx)
	if err != nil {
		t.Fatalf("GetModelConfig failed: %v", err)
	}
	modelChoice := fmt.Sprintf("%d::%s", config.ID, config.DefaultModel)

	assertStatusWithCSRF(t, env.router, client, http.MethodPost, "/chats/1/roles", url.Values{
		"model_choice":     {modelChoice},
		"name":             {"Architect"},
		"persona":          {"Designs the system"},
		"reply_style":      {"concise"},
		"reasoning_effort": {""},
	}, http.StatusFound)
	assertStatusWithCSRF(t, env.router, client, http.MethodPost, "/chats/1/roles", url.Values{
		"model_choice":     {modelChoice},
		"name":             {"Reviewer"},
		"persona":          {"Challenges assumptions"},
		"reply_style":      {"critical"},
		"reasoning_effort": {""},
	}, http.StatusFound)

	assertStatusWithCSRF(t, env.router, client, http.MethodPost, "/chats/1/messages", url.Values{
		"content": {"How should we ship v1?"},
	}, http.StatusFound)

	assertMultipartUploadWithCSRF(t, env.router, client, "/chats/1/files", "chat_file", "notes.txt", "project brief\nmilestone 1", http.StatusFound)

	assertStatus(t, env.router, client, http.MethodGet, "/usage", nil, http.StatusOK)
	assertStatus(t, env.router, client, http.MethodGet, "/health", nil, http.StatusOK)

	chats, err := env.store.ListChats(userCtx)
	if err != nil {
		t.Fatalf("ListChats failed: %v", err)
	}
	if len(chats) != 1 || chats[0].Name != "Integration Chat" {
		t.Fatalf("unexpected chats: %#v", chats)
	}

	roles, err := env.store.ListRoles(userCtx, chats[0].ID)
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("expected 2 roles, got %#v", roles)
	}

	messages, err := env.store.ListMessages(userCtx, chats[0].ID)
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(messages) < 3 {
		t.Fatalf("expected at least 3 messages, got %#v", messages)
	}

	files, err := env.store.ListChatFiles(userCtx, chats[0].ID)
	if err != nil {
		t.Fatalf("ListChatFiles failed: %v", err)
	}
	if len(files) != 1 || files[0].OriginalName != "notes.txt" {
		t.Fatalf("unexpected files: %#v", files)
	}

	stats, err := env.store.TokenUsageStats(userCtx, time.Now())
	if err != nil {
		t.Fatalf("TokenUsageStats failed: %v", err)
	}
	if stats.Today.TotalTokens <= 0 {
		t.Fatalf("expected token usage to be recorded, got %#v", stats)
	}
}

func TestHTTPIntegrationLoginWithRealDeps(t *testing.T) {
	env := requireHTTPIntegrationEnv(t)

	_, token, expiresAt, err := app.NewServices(env.store, fakeAI{}).Register(env.baseCtx, "login@example.test", "secret1")
	if err != nil {
		t.Fatalf("Register service setup failed: %v", err)
	}
	if token == "" || expiresAt.IsZero() {
		t.Fatalf("expected register setup to create session metadata")
	}

	client := newTestClient()
	assertStatus(t, env.router, client, http.MethodGet, "/login", nil, http.StatusOK)
	assertStatusWithCSRF(t, env.router, client, http.MethodPost, "/login", url.Values{
		"email":    {"login@example.test"},
		"password": {"secret1"},
	}, http.StatusFound)
	assertStatus(t, env.router, client, http.MethodGet, "/chats", nil, http.StatusOK)
}

func assertStatusWithCSRF(t *testing.T, handler http.Handler, client *testClient, method, path string, form url.Values, status int) *httptest.ResponseRecorder {
	t.Helper()
	if client != nil && csrfCookieValueFromClient(client) == "" {
		assertStatus(t, handler, client, http.MethodGet, "/login", nil, http.StatusOK)
	}
	if form == nil {
		form = url.Values{}
	}
	if token := csrfCookieValueFromClient(client); token != "" {
		form.Set("csrf_token", token)
	}
	return assertStatus(t, handler, client, method, path, form, status)
}

func assertMultipartUploadWithCSRF(t *testing.T, handler http.Handler, client *testClient, path, fieldName, filename, content string, status int) *httptest.ResponseRecorder {
	t.Helper()
	if csrfCookieValueFromClient(client) == "" {
		assertStatus(t, handler, client, http.MethodGet, "/login", nil, http.StatusOK)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if token := csrfCookieValueFromClient(client); token != "" {
		if err := writer.WriteField("csrf_token", token); err != nil {
			t.Fatalf("WriteField csrf_token failed: %v", err)
		}
	}
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("CreateFormFile failed: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write multipart content failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if client != nil {
		for _, cookie := range client.cookies {
			req.AddCookie(cookie)
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != status {
		t.Fatalf("POST %s expected status %d, got %d; body: %s", path, status, rec.Code, rec.Body.String())
	}
	if client != nil {
		client.cookies = mergeCookies(client.cookies, rec.Result().Cookies())
	}
	return rec
}

func csrfCookieValueFromClient(client *testClient) string {
	if client == nil {
		return ""
	}
	for _, cookie := range client.cookies {
		if cookie.Name == csrfCookieName {
			return cookie.Value
		}
	}
	return ""
}
