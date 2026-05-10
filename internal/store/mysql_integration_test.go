//go:build integration
// +build integration

package store

import (
	"context"
	"os"
	"testing"
	"time"

	"ai_chat/internal/domain"
)

// getTestDSN returns a DSN for MySQL integration tests.
// Expects environment variables or uses defaults for local MySQL.
func getTestDSN() string {
	user := os.Getenv("MYSQL_USER")
	if user == "" {
		user = "root"
	}
	password := os.Getenv("MYSQL_PASSWORD")
	if password == "" {
		password = "4399"
	}
	host := os.Getenv("MYSQL_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("MYSQL_PORT")
	if port == "" {
		port = "3307"
	}
	database := "ai_chat_test"
	return MySQLDSN(user, password, host, port, database)
}

// requireMySQLStore opens a real MySQL store for integration tests.
// It recreates a clean schema before each test.
func requireMySQLStore(t *testing.T) *MySQLStore {
	t.Helper()
	ctx := context.Background()
	cfg := MySQLConfig{
		User:     "root",
		Password: "4399",
		Host:     "localhost",
		Port:     "3307",
		Database: "ai_chat_test",
	}
	if err := EnsureMySQLDatabase(ctx, cfg); err != nil {
		t.Fatalf("MySQL not available: %v", err)
	}
	store, err := OpenMySQL(getTestDSN())
	if err != nil {
		t.Fatalf("Failed to open MySQL: %v", err)
	}

	// Ensure tables exist before cleanup on a fresh test database.
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Initial migrate failed: %v", err)
	}

	// Clear all tables to ensure clean state
	if err := store.ClearAllTables(ctx); err != nil {
		t.Fatalf("Failed to clear tables: %v", err)
	}

	// Run migrations
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func TestMySQLOpenAndPing(t *testing.T) {
	store := requireMySQLStore(t)
	ctx := context.Background()
	if err := store.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}

func TestMySQLMigrate(t *testing.T) {
	_ = requireMySQLStore(t)
	// Migration already done in requireMySQLStore
}

func TestMySQLCreateAndGetUser(t *testing.T) {
	store := requireMySQLStore(t)
	ctx := context.Background()

	// Create user
	user, err := store.CreateUser(ctx, "test@example.com", "hashed_password")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.Email != "test@example.com" || user.PasswordHash != "hashed_password" || user.ID == 0 {
		t.Fatalf("unexpected user: %#v", user)
	}

	// Get user by email
	retrieved, err := store.GetUserByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if retrieved.ID != user.ID || retrieved.Email != user.Email {
		t.Fatalf("unexpected retrieved user: %#v", retrieved)
	}

	// Get user by ID
	byID, err := store.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if byID.ID != user.ID {
		t.Fatalf("unexpected user by ID: %#v", byID)
	}

	// Non-existent user
	if _, err := store.GetUserByEmail(ctx, "nonexistent@example.com"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMySQLSessionOps(t *testing.T) {
	store := requireMySQLStore(t)
	ctx := context.Background()

	// Create user
	user, err := store.CreateUser(ctx, "session@example.com", "pass")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Create session
	tokenHash := "test_token_hash_123"
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := store.CreateSession(ctx, user.ID, tokenHash, expiresAt); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Get session user
	sessionUser, err := store.GetSessionUser(ctx, tokenHash, time.Now())
	if err != nil {
		t.Fatalf("GetSessionUser failed: %v", err)
	}
	if sessionUser.ID != user.ID || sessionUser.Email != user.Email {
		t.Fatalf("unexpected session user: %#v", sessionUser)
	}

	// Expired session
	if _, err := store.GetSessionUser(ctx, tokenHash, time.Now().Add(25*time.Hour)); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for expired session, got %v", err)
	}

	// Delete session
	if err := store.DeleteSession(ctx, tokenHash); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Session should no longer exist
	if _, err := store.GetSessionUser(ctx, tokenHash, time.Now()); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMySQLCreateAndGetChat(t *testing.T) {
	store := requireMySQLStore(t)
	ctx := context.Background()

	// Create user
	user, err := store.CreateUser(ctx, "chat@example.com", "pass")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	userCtx := domain.WithUserID(ctx, user.ID)

	// Create chat
	chat, err := store.CreateChat(userCtx, "Test Chat")
	if err != nil {
		t.Fatalf("CreateChat failed: %v", err)
	}
	if chat.Name != "Test Chat" || chat.UserID != user.ID || chat.ID == 0 {
		t.Fatalf("unexpected chat: %#v", chat)
	}

	// Get chat
	retrieved, err := store.GetChat(userCtx, chat.ID)
	if err != nil {
		t.Fatalf("GetChat failed: %v", err)
	}
	if retrieved.ID != chat.ID || retrieved.Name != chat.Name {
		t.Fatalf("unexpected retrieved chat: %#v", retrieved)
	}

	// List chats
	chats, err := store.ListChats(userCtx)
	if err != nil {
		t.Fatalf("ListChats failed: %v", err)
	}
	if len(chats) != 1 || chats[0].ID != chat.ID {
		t.Fatalf("unexpected chats list: %#v", chats)
	}

	// Update chat topic
	updated, err := store.UpdateChatTopic(userCtx, chat.ID, "New Topic")
	if err != nil {
		t.Fatalf("UpdateChatTopic failed: %v", err)
	}
	if updated.Topic != "New Topic" {
		t.Fatalf("unexpected updated topic: %q", updated.Topic)
	}

	// Update AI review
	reviewed, err := store.UpdateChatAIReview(userCtx, chat.ID, true)
	if err != nil {
		t.Fatalf("UpdateChatAIReview failed: %v", err)
	}
	if !reviewed.AIReviewEnabled {
		t.Fatalf("expected AIReviewEnabled=true, got %v", reviewed.AIReviewEnabled)
	}
}

func TestMySQLCreateAndListRoles(t *testing.T) {
	store := requireMySQLStore(t)
	ctx := context.Background()

	// Create user
	user, err := store.CreateUser(ctx, "role@example.com", "pass")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	userCtx := domain.WithUserID(ctx, user.ID)

	// Create chat
	chat, err := store.CreateChat(userCtx, "Role Chat")
	if err != nil {
		t.Fatalf("CreateChat failed: %v", err)
	}

	// Create role
	role := domain.Role{
		ChatID:          chat.ID,
		Name:            "Assistant",
		Avatar:          "/avatars/assistant.png",
		Persona:         "Helpful assistant",
		ReplyStyle:      "Concise",
		Model:           "gpt-4",
		ReasoningEffort: "high",
		CanSpeak:        true,
	}
	created, err := store.CreateRole(userCtx, role)
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}
	if created.ID == 0 || created.Name != "Assistant" {
		t.Fatalf("unexpected created role: %#v", created)
	}

	// Get role
	retrieved, err := store.GetRole(userCtx, chat.ID, created.ID)
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}
	if retrieved.ID != created.ID || retrieved.Name != "Assistant" {
		t.Fatalf("unexpected retrieved role: %#v", retrieved)
	}

	// List roles
	roles, err := store.ListRoles(userCtx, chat.ID)
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	if len(roles) != 1 || roles[0].ID != created.ID {
		t.Fatalf("unexpected roles list: %#v", roles)
	}

	// Update role
	retrieved.ReplyStyle = "Detailed"
	retrieved.CanSpeak = false
	updated, err := store.UpdateRole(userCtx, retrieved)
	if err != nil {
		t.Fatalf("UpdateRole failed: %v", err)
	}
	if updated.ReplyStyle != "Detailed" || updated.CanSpeak {
		t.Fatalf("unexpected updated role: %#v", updated)
	}

	// Delete role
	if err := store.DeleteRole(userCtx, chat.ID, created.ID); err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}

	// Verify deletion
	if _, err := store.GetRole(userCtx, chat.ID, created.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMySQLModelConfigs(t *testing.T) {
	store := requireMySQLStore(t)
	ctx := context.Background()

	// Create user
	user, err := store.CreateUser(ctx, "config@example.com", "pass")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	userCtx := domain.WithUserID(ctx, user.ID)

	// Save model config
	config := domain.ModelConfig{
		Name:         "Main API",
		Provider:     "openai-compatible",
		BaseURL:      "https://api.example.com/v1",
		APIKey:       "sk-test-key",
		DefaultModel: "gpt-4",
		Models:       []string{"gpt-4", "gpt-3.5-turbo"},
	}
	saved, err := store.SaveModelConfig(userCtx, config)
	if err != nil {
		t.Fatalf("SaveModelConfig failed: %v", err)
	}
	if saved.ID == 0 || saved.Name != "Main API" {
		t.Fatalf("unexpected saved config: %#v", saved)
	}

	// Get model config by ID
	retrieved, err := store.GetModelConfigByID(userCtx, saved.ID)
	if err != nil {
		t.Fatalf("GetModelConfigByID failed: %v", err)
	}
	if retrieved.ID != saved.ID || retrieved.Provider != "openai-compatible" {
		t.Fatalf("unexpected retrieved config: %#v", retrieved)
	}

	// List model configs
	configs, err := store.ListModelConfigs(userCtx)
	if err != nil {
		t.Fatalf("ListModelConfigs failed: %v", err)
	}
	if len(configs) < 1 {
		t.Fatalf("expected at least one config, got %#v", configs)
	}

	// Get default config
	defaultCfg, err := store.GetModelConfig(userCtx)
	if err != nil {
		t.Fatalf("GetModelConfig failed: %v", err)
	}
	if defaultCfg.ID == 0 {
		t.Fatalf("unexpected default config: %#v", defaultCfg)
	}

	// Delete model config
	if err := store.DeleteModelConfig(userCtx, saved.ID); err != nil {
		t.Fatalf("DeleteModelConfig failed: %v", err)
	}

	// Verify deletion
	if _, err := store.GetModelConfigByID(userCtx, saved.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMySQLCreateAndListMessages(t *testing.T) {
	store := requireMySQLStore(t)
	ctx := context.Background()

	// Create user
	user, err := store.CreateUser(ctx, "msg@example.com", "pass")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	userCtx := domain.WithUserID(ctx, user.ID)

	// Create chat
	chat, err := store.CreateChat(userCtx, "Message Chat")
	if err != nil {
		t.Fatalf("CreateChat failed: %v", err)
	}

	// Create message
	msg := domain.Message{
		ChatID:     chat.ID,
		SenderType: domain.SenderUser,
		SenderName: "User",
		Content:    "Hello, world!",
	}
	created, err := store.CreateMessage(userCtx, msg)
	if err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}
	if created.ID == 0 || created.Content != "Hello, world!" {
		t.Fatalf("unexpected created message: %#v", created)
	}

	// List messages
	messages, err := store.ListMessages(userCtx, chat.ID)
	if err != nil {
		t.Fatalf("ListMessages failed: %v", err)
	}
	if len(messages) != 1 || messages[0].ID != created.ID {
		t.Fatalf("unexpected messages list: %#v", messages)
	}

	// Create another message
	msg2 := domain.Message{
		ChatID:     chat.ID,
		SenderType: domain.SenderAI,
		SenderName: "Assistant",
		Content:    "Hi there!",
	}
	created2, err := store.CreateMessage(userCtx, msg2)
	if err != nil {
		t.Fatalf("CreateMessage #2 failed: %v", err)
	}

	// List messages after first message
	messagesAfter, err := store.ListMessagesAfter(userCtx, chat.ID, created.ID)
	if err != nil {
		t.Fatalf("ListMessagesAfter failed: %v", err)
	}
	if len(messagesAfter) != 1 || messagesAfter[0].ID != created2.ID {
		t.Fatalf("unexpected messages after: %#v", messagesAfter)
	}
}

func TestMySQLChatFileOps(t *testing.T) {
	store := requireMySQLStore(t)
	ctx := context.Background()

	// Create user
	user, err := store.CreateUser(ctx, "file@example.com", "pass")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	userCtx := domain.WithUserID(ctx, user.ID)

	// Create chat
	chat, err := store.CreateChat(userCtx, "File Chat")
	if err != nil {
		t.Fatalf("CreateChat failed: %v", err)
	}

	// Create chat file
	file := domain.ChatFile{
		ChatID:        chat.ID,
		OriginalName:  "document.txt",
		StoragePath:   "/uploads/doc123.txt",
		ContentType:   "text/plain",
		SizeBytes:     1024,
		ExtractedText: "Document content here",
	}
	created, err := store.CreateChatFile(userCtx, file)
	if err != nil {
		t.Fatalf("CreateChatFile failed: %v", err)
	}
	if created.ID == 0 || created.OriginalName != "document.txt" {
		t.Fatalf("unexpected created file: %#v", created)
	}

	// List chat files
	files, err := store.ListChatFiles(userCtx, chat.ID)
	if err != nil {
		t.Fatalf("ListChatFiles failed: %v", err)
	}
	if len(files) != 1 || files[0].ID != created.ID {
		t.Fatalf("unexpected files list: %#v", files)
	}
}

func TestMySQLTokenUsage(t *testing.T) {
	store := requireMySQLStore(t)
	ctx := context.Background()

	// Create user
	user, err := store.CreateUser(ctx, "token@example.com", "pass")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	userCtx := domain.WithUserID(ctx, user.ID)

	// Create chat and message for token usage
	chat, err := store.CreateChat(userCtx, "Token Chat")
	if err != nil {
		t.Fatalf("CreateChat failed: %v", err)
	}

	msg, err := store.CreateMessage(userCtx, domain.Message{
		ChatID:     chat.ID,
		SenderType: domain.SenderUser,
		SenderName: "User",
		Content:    "Test",
	})
	if err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	// Create token usage
	usage := domain.TokenUsage{
		ChatID:           chat.ID,
		MessageID:        msg.ID,
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
	}
	created, err := store.CreateTokenUsage(userCtx, usage)
	if err != nil {
		t.Fatalf("CreateTokenUsage failed: %v", err)
	}
	if created.ID == 0 || created.TotalTokens != 150 {
		t.Fatalf("unexpected created usage: %#v", created)
	}

	// Get token usage stats
	stats, err := store.TokenUsageStats(userCtx, time.Now())
	if err != nil {
		t.Fatalf("TokenUsageStats failed: %v", err)
	}
	if stats.Today.TotalTokens != 150 {
		t.Fatalf("unexpected token stats: %#v", stats)
	}
}

func TestMySQLToolExecution(t *testing.T) {
	store := requireMySQLStore(t)
	ctx := context.Background()

	// Create user
	user, err := store.CreateUser(ctx, "tool@example.com", "pass")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	userCtx := domain.WithUserID(ctx, user.ID)

	// Create chat and message
	chat, err := store.CreateChat(userCtx, "Tool Chat")
	if err != nil {
		t.Fatalf("CreateChat failed: %v", err)
	}

	msg, err := store.CreateMessage(userCtx, domain.Message{
		ChatID:     chat.ID,
		SenderType: domain.SenderUser,
		SenderName: "User",
		Content:    "Run calculator",
	})
	if err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	// Create tool execution
	execution := domain.ToolExecution{
		ChatID:    chat.ID,
		MessageID: msg.ID,
		ToolName:  "calculator",
		Input:     "2 + 2",
		Status:    domain.ToolExecutionSuccess,
		Result:    "4",
	}
	created, err := store.CreateToolExecution(userCtx, execution)
	if err != nil {
		t.Fatalf("CreateToolExecution failed: %v", err)
	}
	if created.ID == 0 || created.Result != "4" {
		t.Fatalf("unexpected created execution: %#v", created)
	}

	// List tool executions
	executions, err := store.ListToolExecutions(userCtx, chat.ID)
	if err != nil {
		t.Fatalf("ListToolExecutions failed: %v", err)
	}
	if len(executions) < 1 {
		t.Fatalf("unexpected executions list: %#v", executions)
	}
}

func TestMySQLDeleteChat(t *testing.T) {
	store := requireMySQLStore(t)
	ctx := context.Background()

	// Create user
	user, err := store.CreateUser(ctx, "delete@example.com", "pass")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	userCtx := domain.WithUserID(ctx, user.ID)

	// Create chat
	chat, err := store.CreateChat(userCtx, "To Delete")
	if err != nil {
		t.Fatalf("CreateChat failed: %v", err)
	}

	// Create role in chat
	_, err = store.CreateRole(userCtx, domain.Role{
		ChatID: chat.ID,
		Name:   "Role in Chat",
		Model:  "gpt-4",
	})
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	// Create message in chat
	_, err = store.CreateMessage(userCtx, domain.Message{
		ChatID:     chat.ID,
		SenderType: domain.SenderUser,
		SenderName: "User",
		Content:    "Test",
	})
	if err != nil {
		t.Fatalf("CreateMessage failed: %v", err)
	}

	// Delete chat should cascade delete related data
	if err := store.DeleteChat(userCtx, chat.ID); err != nil {
		t.Fatalf("DeleteChat failed: %v", err)
	}

	// Verify chat is deleted
	if _, err := store.GetChat(userCtx, chat.ID); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	// Verify roles are deleted
	roles, err := store.ListRoles(userCtx, chat.ID)
	if err != nil {
		t.Fatalf("ListRoles after delete failed: %v", err)
	}
	if len(roles) != 0 {
		t.Fatalf("expected no roles after chat delete, got %#v", roles)
	}
}
