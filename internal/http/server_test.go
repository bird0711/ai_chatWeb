package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"ai_chat/internal/ai"
	"ai_chat/internal/app"
	"ai_chat/internal/domain"

	"github.com/gin-gonic/gin"
)

type fakeStore struct {
	mu       sync.Mutex
	chats    []domain.Chat
	roles    []domain.Role
	messages []domain.Message
	files    []domain.ChatFile
	tools    []domain.ToolExecution
	usages   []domain.TokenUsage
	users    []domain.User
	sessions map[string]int64
	configs  []domain.ModelConfig
	nextID   int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{nextID: 1, sessions: map[string]int64{}}
}

func (f *fakeStore) CreateUser(ctx context.Context, email, passwordHash string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	user := domain.User{ID: int64(len(f.users) + 1), Email: email, PasswordHash: passwordHash}
	f.users = append(f.users, user)
	return user, nil
}

func (f *fakeStore) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, user := range f.users {
		if user.Email == email {
			return user, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (f *fakeStore) GetUser(ctx context.Context, userID int64) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, user := range f.users {
		if user.ID == userID {
			return user, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (f *fakeStore) CreateSession(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sessions[tokenHash] = userID
	return nil
}

func (f *fakeStore) GetSessionUser(ctx context.Context, tokenHash string, now time.Time) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, ok := f.sessions[tokenHash]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	for _, user := range f.users {
		if user.ID == userID {
			return user, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (f *fakeStore) DeleteSession(ctx context.Context, tokenHash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, tokenHash)
	return nil
}

func (f *fakeStore) CreateChat(ctx context.Context, name string) (domain.Chat, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	chat := domain.Chat{ID: f.next(), UserID: userID, Name: name}
	f.chats = append(f.chats, chat)
	return chat, nil
}

func (f *fakeStore) ListChats(ctx context.Context) ([]domain.Chat, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	var chats []domain.Chat
	for _, chat := range f.chats {
		if chat.UserID == userID {
			chats = append(chats, chat)
		}
	}
	return chats, nil
}

func (f *fakeStore) GetChat(ctx context.Context, chatID int64) (domain.Chat, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	for _, chat := range f.chats {
		if chat.ID == chatID && chat.UserID == userID {
			return chat, nil
		}
	}
	return domain.Chat{}, domain.ErrNotFound
}

func (f *fakeStore) UpdateChatAIReview(ctx context.Context, chatID int64, enabled bool) (domain.Chat, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	for i, chat := range f.chats {
		if chat.ID == chatID && chat.UserID == userID {
			f.chats[i].AIReviewEnabled = enabled
			return f.chats[i], nil
		}
	}
	return domain.Chat{}, domain.ErrNotFound
}

func (f *fakeStore) UpdateChatTopic(ctx context.Context, chatID int64, topic string) (domain.Chat, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	for i, chat := range f.chats {
		if chat.ID == chatID && chat.UserID == userID {
			f.chats[i].Topic = topic
			return f.chats[i], nil
		}
	}
	return domain.Chat{}, domain.ErrNotFound
}

func (f *fakeStore) DeleteChat(ctx context.Context, chatID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	for i, chat := range f.chats {
		if chat.ID == chatID && chat.UserID == userID {
			f.chats = append(f.chats[:i], f.chats[i+1:]...)
			var roles []domain.Role
			for _, role := range f.roles {
				if role.ChatID != chatID {
					roles = append(roles, role)
				}
			}
			f.roles = roles
			var messages []domain.Message
			for _, message := range f.messages {
				if message.ChatID != chatID {
					messages = append(messages, message)
				}
			}
			f.messages = messages
			var files []domain.ChatFile
			for _, file := range f.files {
				if file.ChatID != chatID {
					files = append(files, file)
				}
			}
			f.files = files
			var tools []domain.ToolExecution
			for _, tool := range f.tools {
				if tool.ChatID != chatID {
					tools = append(tools, tool)
				}
			}
			f.tools = tools
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeStore) CreateChatFile(ctx context.Context, file domain.ChatFile) (domain.ChatFile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	file.ID = f.next()
	file.UserID = userID
	file.CreatedAt = time.Now()
	f.files = append(f.files, file)
	return file, nil
}

func (f *fakeStore) ListChatFiles(ctx context.Context, chatID int64) ([]domain.ChatFile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	var files []domain.ChatFile
	for _, file := range f.files {
		if file.ChatID == chatID && file.UserID == userID {
			files = append(files, file)
		}
	}
	return files, nil
}

func (f *fakeStore) CreateToolExecution(ctx context.Context, execution domain.ToolExecution) (domain.ToolExecution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	execution.ID = f.next()
	execution.UserID = userID
	execution.CreatedAt = time.Now()
	f.tools = append(f.tools, execution)
	return execution, nil
}

func (f *fakeStore) ListToolExecutions(ctx context.Context, chatID int64) ([]domain.ToolExecution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	var tools []domain.ToolExecution
	for _, tool := range f.tools {
		if tool.ChatID == chatID && tool.UserID == userID {
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

func (f *fakeStore) CreateRole(ctx context.Context, role domain.Role) (domain.Role, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	role.ID = f.next()
	f.roles = append(f.roles, role)
	return role, nil
}

func (f *fakeStore) ListRoles(ctx context.Context, chatID int64) ([]domain.Role, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var roles []domain.Role
	for _, role := range f.roles {
		if role.ChatID == chatID {
			roles = append(roles, role)
		}
	}
	return roles, nil
}

func (f *fakeStore) GetRole(ctx context.Context, chatID, roleID int64) (domain.Role, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, role := range f.roles {
		if role.ChatID == chatID && role.ID == roleID {
			return role, nil
		}
	}
	return domain.Role{}, domain.ErrNotFound
}

func (f *fakeStore) UpdateRole(ctx context.Context, role domain.Role) (domain.Role, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, existing := range f.roles {
		if existing.ChatID == role.ChatID && existing.ID == role.ID {
			f.roles[i] = role
			return role, nil
		}
	}
	return domain.Role{}, domain.ErrNotFound
}

func (f *fakeStore) DeleteRole(ctx context.Context, chatID, roleID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, role := range f.roles {
		if role.ChatID == chatID && role.ID == roleID {
			f.roles = append(f.roles[:i], f.roles[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeStore) SaveModelConfig(ctx context.Context, config domain.ModelConfig) (domain.ModelConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	config.UserID = userID
	config.ID = f.next()
	f.configs = append(f.configs, config)
	return config, nil
}

func (f *fakeStore) ListModelConfigs(ctx context.Context) ([]domain.ModelConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	var configs []domain.ModelConfig
	for _, config := range f.configs {
		if config.UserID == userID {
			configs = append(configs, config)
		}
	}
	return configs, nil
}

func (f *fakeStore) GetModelConfig(ctx context.Context) (domain.ModelConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	for i := len(f.configs) - 1; i >= 0; i-- {
		if f.configs[i].UserID == userID {
			return f.configs[i], nil
		}
	}
	return domain.ModelConfig{}, domain.ErrNotFound
}

func (f *fakeStore) GetModelConfigByID(ctx context.Context, configID int64) (domain.ModelConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	for _, config := range f.configs {
		if config.ID == configID && config.UserID == userID {
			return config, nil
		}
	}
	return domain.ModelConfig{}, domain.ErrNotFound
}

func (f *fakeStore) DeleteModelConfig(ctx context.Context, configID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	userID, _ := domain.UserIDFromContext(ctx)
	for i, config := range f.configs {
		if config.ID == configID && config.UserID == userID {
			f.configs = append(f.configs[:i], f.configs[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeStore) CountRolesByModelConfig(ctx context.Context, configID int64) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, role := range f.roles {
		if role.ModelConfigID == configID {
			count++
		}
	}
	return count, nil
}

func (f *fakeStore) Ping(ctx context.Context) error {
	return nil
}

func (f *fakeStore) CreateMessage(ctx context.Context, message domain.Message) (domain.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	message.ID = f.next()
	f.messages = append(f.messages, message)
	return message, nil
}

func (f *fakeStore) ListMessages(ctx context.Context, chatID int64) ([]domain.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var messages []domain.Message
	for _, message := range f.messages {
		if message.ChatID == chatID {
			messages = append(messages, message)
		}
	}
	return messages, nil
}

func (f *fakeStore) ListMessagesAfter(ctx context.Context, chatID, afterID int64) ([]domain.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var messages []domain.Message
	for _, message := range f.messages {
		if message.ChatID == chatID && message.ID > afterID {
			messages = append(messages, message)
		}
	}
	return messages, nil
}

func (f *fakeStore) CreateTokenUsage(ctx context.Context, usage domain.TokenUsage) (domain.TokenUsage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	usage.ID = f.next()
	usage.UserID = 1
	usage.CreatedAt = time.Now()
	f.usages = append(f.usages, usage)
	return usage, nil
}

func (f *fakeStore) TokenUsageStats(ctx context.Context, now time.Time) (domain.TokenUsageStats, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stats := domain.TokenUsageStats{
		Today:   domain.TokenUsageSummary{Label: "今天"},
		Recent7: domain.TokenUsageSummary{Label: "最近 7 天"},
	}
	byModel := map[string]*domain.TokenUsageSummary{}
	for _, usage := range f.usages {
		stats.Today.PromptTokens += usage.PromptTokens
		stats.Today.CompletionTokens += usage.CompletionTokens
		stats.Today.TotalTokens += usage.TotalTokens
		stats.Recent7.PromptTokens += usage.PromptTokens
		stats.Recent7.CompletionTokens += usage.CompletionTokens
		stats.Recent7.TotalTokens += usage.TotalTokens
		summary := byModel[usage.Model]
		if summary == nil {
			summary = &domain.TokenUsageSummary{Label: usage.Model}
			byModel[usage.Model] = summary
		}
		summary.PromptTokens += usage.PromptTokens
		summary.CompletionTokens += usage.CompletionTokens
		summary.TotalTokens += usage.TotalTokens
	}
	for _, summary := range byModel {
		stats.ByModel = append(stats.ByModel, *summary)
	}
	return stats, nil
}

func (f *fakeStore) next() int64 {
	id := f.nextID
	f.nextID++
	return id
}

type fakeAI struct{}

func (fakeAI) GenerateReply(ctx context.Context, input ai.ReplyInput) (ai.Reply, error) {
	return ai.Reply{Content: "reply from " + input.Role.Name + ": " + input.UserMessage.Content, Usage: ai.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}}, nil
}

func (fakeAI) GenerateReview(ctx context.Context, input ai.ReviewInput) (ai.Reply, error) {
	return ai.Reply{Content: "review from " + input.Role.Name + ": supplement or rebut " + input.FirstRoundReplies[0].SenderName, Usage: ai.Usage{PromptTokens: 8, CompletionTokens: 4, TotalTokens: 12}}, nil
}

func (fakeAI) ListModels(ctx context.Context, config domain.ModelConfig) ([]string, error) {
	return []string{"test-model", "backup-model"}, nil
}

type fakeRedis struct{}

func (fakeRedis) Ping(ctx context.Context) error {
	return nil
}

func TestMVPWebOperationPath(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodGet, "/chats", nil, http.StatusFound)
	assertStatus(t, router, client, http.MethodGet, "/login", nil, http.StatusOK)
	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"mvp@example.test"}, "password": {"secret1"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodGet, "/chats", nil, http.StatusOK)
	location := assertStatus(t, router, client, http.MethodPost, "/chats", url.Values{"name": {"MVP Chat"}}, http.StatusFound).Header().Get("Location")
	if location != "/chats/1" {
		t.Fatalf("expected redirect to /chats/1, got %q", location)
	}
	assertStatus(t, router, client, http.MethodGet, location, nil, http.StatusOK)
	checkBody := assertStatus(t, router, client, http.MethodPost, "/settings/model/check", url.Values{
		"provider": {"openai-compatible"},
		"base_url": {"https://example.test/v1"},
		"api_key":  {"test-key"},
	}, http.StatusOK).Body.String()
	for _, expected := range []string{"连接成功", "test-model", "backup-model"} {
		if !strings.Contains(checkBody, expected) {
			t.Fatalf("expected check settings page to contain %q, body: %s", expected, checkBody)
		}
	}
	assertStatus(t, router, client, http.MethodPost, "/settings/model", url.Values{
		"name":          {"Test API"},
		"provider":      {"openai-compatible"},
		"base_url":      {"https://example.test/v1"},
		"api_key":       {"test-key"},
		"default_model": {"test-model"},
		"models":        {"test-model\nbackup-model"},
	}, http.StatusOK)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles", roleForm("Architect"), http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles", roleForm("Reviewer"), http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/messages", url.Values{"content": {"How should we scope MVP?"}}, http.StatusFound)

	body := assertStatus(t, router, client, http.MethodGet, "/chats/1", nil, http.StatusOK).Body.String()
	for _, expected := range []string{"User", "Architect", "Reviewer", "How should we scope MVP?", "reply from Architect", "reply from Reviewer"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected chat page to contain %q, body: %s", expected, body)
		}
	}
	for _, expected := range []string{"message-row user", "message-row ai", "message-bubble", "message-meta", "message-content", "message-avatar", "avatar-chip"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected chat-like message markup %q, body: %s", expected, body)
		}
	}
	for _, expected := range []string{"chat-workspace", "chat-sidebar-left", "chat-main", "chat-sidebar-right", "settings-menu", "side-menu", "本地头像图片", `type="file" name="avatar_file"`, `enctype="multipart/form-data"`, `data-enter-submit="true"`, "按 Enter 发送，Shift + Enter 换行。", `data-theme-toggle`, `/static/theme.js?v=20260504a`, `/static/chat.js?v=20260504a`, ">发送</button>"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected centered chat workspace markup %q, body: %s", expected, body)
		}
	}
	for _, expected := range []string{"文件资料", "/chats/1/files", `name="chat_file"`, `data-chat-file-form`, `data-chat-file-input`, `data-chat-file-submit`, ".docx", ".pdf", "从本机上传文件", "点击选择本地文件，或把文件拖到这里", "支持 txt、md、json、csv、log、docx、pdf", "请直接选择你电脑本地的文件", "扫描版 PDF 暂不支持", "上传后 AI 回复和互评会参考文件文本内容"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected chat file upload markup %q, body: %s", expected, body)
		}
	}
	for _, expected := range []string{`name="reasoning_effort"`, "思考强度", "默认", "低", "中", "高", "仅在兼容的模型 API 中生效"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected reasoning effort markup %q, body: %s", expected, body)
		}
	}
	for _, expected := range []string{"搜索历史消息", `data-message-search`, `data-message-sender-filter`, "只看用户", "只看 AI", "只看系统", `data-message-search-clear`, `data-message-search-empty`, `data-sender-type="user"`, `data-sender-type="ai"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected message history search markup %q, body: %s", expected, body)
		}
	}
	for _, expected := range []string{"讨论主题", "/chats/1/topic", "保存主题"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected topic guide markup %q, body: %s", expected, body)
		}
	}
	for _, unexpected := range []string{"头像文本或图片 URL", "粘贴图片 URL"} {
		if strings.Contains(body, unexpected) {
			t.Fatalf("expected local avatar upload instead of URL copy, found %q in body: %s", unexpected, body)
		}
	}
}

func TestChatResponsiveStyles(t *testing.T) {
	css, err := os.ReadFile("../../web/static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	body := string(css)
	for _, expected := range []string{
		"overflow-x: hidden",
		"minmax(0, 880px)",
		"overscroll-behavior: contain",
		"@media (max-width: 520px)",
		"position: sticky",
		"bottom: 8px",
		".message-avatar",
		"min-height: 44px",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected responsive CSS %q, body: %s", expected, body)
		}
	}
}

func TestAuthRegisterLoginLogoutAndIsolation(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	alice := newTestClient()
	bob := newTestClient()

	assertStatus(t, router, alice, http.MethodGet, "/chats", nil, http.StatusFound)
	assertStatus(t, router, alice, http.MethodPost, "/register", url.Values{"email": {"alice@example.test"}, "password": {"secret1"}}, http.StatusFound)
	assertStatus(t, router, alice, http.MethodPost, "/chats", url.Values{"name": {"Alice Chat"}}, http.StatusFound)
	aliceBody := assertStatus(t, router, alice, http.MethodGet, "/chats", nil, http.StatusOK).Body.String()
	if !strings.Contains(aliceBody, "Alice Chat") || !strings.Contains(aliceBody, "alice@example.test") {
		t.Fatalf("expected alice to see own chat and email, body: %s", aliceBody)
	}

	assertStatus(t, router, bob, http.MethodPost, "/register", url.Values{"email": {"bob@example.test"}, "password": {"secret1"}}, http.StatusFound)
	bobBody := assertStatus(t, router, bob, http.MethodGet, "/chats", nil, http.StatusOK).Body.String()
	if strings.Contains(bobBody, "Alice Chat") {
		t.Fatalf("expected bob not to see alice chat, body: %s", bobBody)
	}
	assertStatus(t, router, bob, http.MethodGet, "/chats/1", nil, http.StatusNotFound)

	assertStatus(t, router, alice, http.MethodPost, "/logout", nil, http.StatusFound)
	assertStatus(t, router, alice, http.MethodGet, "/chats", nil, http.StatusFound)
	assertStatus(t, router, alice, http.MethodPost, "/login", url.Values{"email": {"alice@example.test"}, "password": {"secret1"}}, http.StatusFound)
	aliceBody = assertStatus(t, router, alice, http.MethodGet, "/chats", nil, http.StatusOK).Body.String()
	if !strings.Contains(aliceBody, "Alice Chat") {
		t.Fatalf("expected alice chat after login, body: %s", aliceBody)
	}
}

func TestRenderedChatIndexFormsIncludeCSRFToken(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"csrf@example.test"}, "password": {"secret1"}}, http.StatusFound)
	body := assertStatus(t, router, client, http.MethodGet, "/chats", nil, http.StatusOK).Body.String()
	token := csrfCookieValue(client)
	if token == "" {
		t.Fatal("expected csrf cookie to be set")
	}
	tokenField := `name="csrf_token" value="` + token + `"`
	for _, expected := range []string{`action="/logout"`, `action="/chats"`, tokenField} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected chat index to contain %q, body: %s", expected, body)
		}
	}
	if strings.Contains(body, `name="csrf_token" value=""`) {
		t.Fatalf("expected chat index csrf fields to be non-empty, body: %s", body)
	}
}

func TestV02AsyncMessagesAndDeletionRoutes(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	st := newFakeStore()
	router := NewRouter(app.NewServices(st, fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"async@example.test"}, "password": {"secret1"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats", url.Values{"name": {"Usable Chat"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/settings/model", url.Values{
		"name":          {"Test API"},
		"provider":      {"openai-compatible"},
		"base_url":      {"https://example.test/v1"},
		"api_key":       {"test-key"},
		"default_model": {"test-model"},
		"models":        {"test-model\nbackup-model"},
	}, http.StatusOK)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles", roleForm("Architect"), http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles", roleForm("Reviewer"), http.StatusFound)

	rec := assertStatus(t, router, client, http.MethodPost, "/chats/1/messages/async", url.Values{"content": {"How should v0.2 work?"}}, http.StatusAccepted)
	var sendBody struct {
		Message struct {
			ID      int64  `json:"id"`
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sendBody); err != nil {
		t.Fatal(err)
	}
	if sendBody.Message.ID == 0 || sendBody.Message.Content != "How should v0.2 work?" {
		t.Fatalf("unexpected async send response: %s", rec.Body.String())
	}

	waitForMessages(t, router, client, "/chats/1/messages/updates?after_id=0", []string{"reply from Architect", "reply from Reviewer"})

	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles/4/delete", nil, http.StatusFound)
	body := assertStatus(t, router, client, http.MethodGet, "/chats/1", nil, http.StatusOK).Body.String()
	if !strings.Contains(body, "1 个 AI 角色") || strings.Contains(body, "/chats/1/roles/4/delete") || !strings.Contains(body, "reply from Reviewer") {
		t.Fatalf("expected deleted role to be absent from role controls while history remains, body: %s", body)
	}

	assertStatus(t, router, client, http.MethodPost, "/chats/1/delete", nil, http.StatusFound)
	body = assertStatus(t, router, client, http.MethodGet, "/chats", nil, http.StatusOK).Body.String()
	if strings.Contains(body, "Usable Chat") {
		t.Fatalf("expected deleted chat to be absent from list, body: %s", body)
	}
	assertStatus(t, router, client, http.MethodGet, "/chats/1", nil, http.StatusNotFound)
}

func TestV02RoleEditAndSpeakingPermissionRoutes(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"roles@example.test"}, "password": {"secret1"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats", url.Values{"name": {"Role Chat"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/settings/model", url.Values{
		"name":          {"Test API"},
		"provider":      {"openai-compatible"},
		"base_url":      {"https://example.test/v1"},
		"api_key":       {"test-key"},
		"default_model": {"test-model"},
		"models":        {"test-model\nbackup-model"},
	}, http.StatusOK)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles", roleForm("Architect"), http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles", roleForm("Reviewer"), http.StatusFound)

	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles/4", url.Values{
		"name":         {"Strategist"},
		"persona":      {"Updated persona"},
		"reply_style":  {"short"},
		"model_choice": {"2::backup-model"},
	}, http.StatusFound)
	body := assertStatus(t, router, client, http.MethodGet, "/chats/1", nil, http.StatusOK).Body.String()
	for _, expected := range []string{"Strategist", "St", "Updated persona", "short · 思考强度：默认", "路由 #2 · Test API · openai-compatible · backup-model", "已禁言"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected role page to contain %q, body: %s", expected, body)
		}
	}
	rec := assertStatus(t, router, client, http.MethodPost, "/chats/1/messages/async", url.Values{"content": {"blocked"}}, http.StatusBadRequest)
	if !strings.Contains(rec.Body.String(), "speaking permission") {
		t.Fatalf("expected speaking permission error, got %s", rec.Body.String())
	}

	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles/4/toggle-speaking", nil, http.StatusFound)
	body = assertStatus(t, router, client, http.MethodGet, "/chats/1", nil, http.StatusOK).Body.String()
	if !strings.Contains(body, "允许发言") {
		t.Fatalf("expected role to be allowed after toggle, body: %s", body)
	}
	assertStatus(t, router, client, http.MethodPost, "/chats/1/messages/async", url.Values{"content": {"allowed"}}, http.StatusAccepted)
	waitForMessages(t, router, client, "/chats/1/messages/updates?after_id=0", []string{"reply from Architect", "reply from Strategist"})
}

func TestChatActionErrorsAreLogged(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"logs@example.test"}, "password": {"secret1"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats", url.Values{"name": {"Log Chat"}}, http.StatusFound)

	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)

	assertStatus(t, router, client, http.MethodPost, "/chats/1/messages/async", url.Values{"content": {"blocked"}}, http.StatusBadRequest)
	for _, expected := range []string{"chat_action_error", "send_message_async", "chat_id=1", "status=400", "request_id="} {
		if !strings.Contains(logs.String(), expected) {
			t.Fatalf("expected log output to contain %q, got %s", expected, logs.String())
		}
	}
}

func TestRequestIDHeaderOnSuccessfulRequest(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})

	rec := assertStatus(t, router, nil, http.MethodGet, "/login", nil, http.StatusOK)
	if strings.TrimSpace(rec.Header().Get(requestIDHeaderName)) == "" {
		t.Fatalf("expected %s header on successful request", requestIDHeaderName)
	}
}

func TestRequestIDAppearsInHTMLErrorPage(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"requestid-html@example.test"}, "password": {"secret1"}}, http.StatusFound)
	rec := assertStatus(t, router, client, http.MethodGet, "/chats/not-a-number", nil, http.StatusBadRequest)

	requestID := strings.TrimSpace(rec.Header().Get(requestIDHeaderName))
	if requestID == "" {
		t.Fatalf("expected %s header on HTML error response", requestIDHeaderName)
	}
	if !strings.Contains(rec.Body.String(), requestID) {
		t.Fatalf("expected error page to contain request ID %q, body: %s", requestID, rec.Body.String())
	}
}

func TestRequestIDAppearsInJSONErrorResponse(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"requestid-json@example.test"}, "password": {"secret1"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats", url.Values{"name": {"Request ID Chat"}}, http.StatusFound)

	rec := assertStatus(t, router, client, http.MethodPost, "/chats/1/messages/async", url.Values{"content": {"blocked"}}, http.StatusBadRequest)
	requestID := strings.TrimSpace(rec.Header().Get(requestIDHeaderName))
	if requestID == "" {
		t.Fatalf("expected %s header on JSON error response", requestIDHeaderName)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body, got %s: %v", rec.Body.String(), err)
	}
	if body["request_id"] != requestID {
		t.Fatalf("expected request_id %q in JSON body, got %#v", requestID, body["request_id"])
	}
}

func TestPanicRecoveryIncludesRequestID(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"panic@example.test"}, "password": {"secret1"}}, http.StatusFound)
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	rec := assertStatus(t, router, client, http.MethodGet, "/panic", nil, http.StatusInternalServerError)
	requestID := strings.TrimSpace(rec.Header().Get(requestIDHeaderName))
	if requestID == "" {
		t.Fatalf("expected %s header on panic response", requestIDHeaderName)
	}
	if !strings.Contains(rec.Body.String(), requestID) {
		t.Fatalf("expected panic error page to contain request ID %q, body: %s", requestID, rec.Body.String())
	}
}

func TestV02AIReviewRoutes(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"review@example.test"}, "password": {"secret1"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats", url.Values{"name": {"Review Chat"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/settings/model", url.Values{
		"name":          {"Test API"},
		"provider":      {"openai-compatible"},
		"base_url":      {"https://example.test/v1"},
		"api_key":       {"test-key"},
		"default_model": {"test-model"},
		"models":        {"test-model\nbackup-model"},
	}, http.StatusOK)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles", roleForm("Architect"), http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles", roleForm("Reviewer"), http.StatusFound)

	body := assertStatus(t, router, client, http.MethodGet, "/chats/1", nil, http.StatusOK).Body.String()
	if !strings.Contains(body, "AI 互评") || !strings.Contains(body, "已关闭") {
		t.Fatalf("expected AI review disabled state, body: %s", body)
	}
	if !strings.Contains(body, `data-ai-review-form`) || !strings.Contains(body, `data-ai-review-page-status`) {
		t.Fatalf("expected async AI review toggle markup, body: %s", body)
	}
	assertStatus(t, router, client, http.MethodPost, "/chats/1/ai-review", url.Values{"enabled": {"1"}}, http.StatusFound)
	body = assertStatus(t, router, client, http.MethodGet, "/chats/1", nil, http.StatusOK).Body.String()
	if !strings.Contains(body, "已开启") {
		t.Fatalf("expected AI review enabled state, body: %s", body)
	}
	rec := assertStatusWithHeaders(t, router, client, http.MethodPost, "/chats/1/ai-review", url.Values{"enabled": {"0"}}, map[string]string{"Accept": "application/json"}, http.StatusOK)
	var toggleBody struct {
		ChatID          int64  `json:"chat_id"`
		AIReviewEnabled bool   `json:"ai_review_enabled"`
		Status          string `json:"ai_review_status"`
		Headline        string `json:"ai_review_headline"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &toggleBody); err != nil {
		t.Fatalf("expected JSON AI review response, got %s: %v", rec.Body.String(), err)
	}
	if toggleBody.ChatID != 1 || toggleBody.AIReviewEnabled || toggleBody.Status != "已关闭" || toggleBody.Headline != "AI 互评已关闭" {
		t.Fatalf("unexpected AI review JSON response: %#v", toggleBody)
	}
	assertStatusWithHeaders(t, router, client, http.MethodPost, "/chats/1/ai-review", url.Values{"enabled": {"1"}}, map[string]string{"Accept": "application/json"}, http.StatusOK)

	assertStatus(t, router, client, http.MethodPost, "/chats/1/messages/async", url.Values{"content": {"with review"}}, http.StatusAccepted)
	waitForMessages(t, router, client, "/chats/1/messages/updates?after_id=0", []string{"reply from Architect", "reply from Reviewer", "review from"})

	assertStatus(t, router, client, http.MethodPost, "/chats/1/messages/async", url.Values{"content": {"with review, please compare these options and identify the main risk"}}, http.StatusAccepted)
	waitForMessages(t, router, client, "/chats/1/messages/updates?after_id=0", []string{"review from"})
}

func TestV03ChatTopicRoutes(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"topic@example.test"}, "password": {"secret1"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats", url.Values{"name": {"Topic Chat"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/topic", url.Values{"topic": {"  v0.3 用户体验优化  "}}, http.StatusFound)
	body := assertStatus(t, router, client, http.MethodGet, "/chats/1", nil, http.StatusOK).Body.String()
	for _, expected := range []string{"讨论主题", "已设置", "主题：v0.3 用户体验优化", "v0.3 用户体验优化"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected topic page to contain %q, body: %s", expected, body)
		}
	}
	tooLong := strings.Repeat("好", 501)
	rec := assertStatus(t, router, client, http.MethodPost, "/chats/1/topic", url.Values{"topic": {tooLong}}, http.StatusBadRequest)
	if !strings.Contains(rec.Body.String(), "at most 500") {
		t.Fatalf("expected topic validation error, body: %s", rec.Body.String())
	}
}

func TestV10ToolRoutes(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"tools@example.test"}, "password": {"secret1"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats", url.Values{"name": {"Tool Chat"}}, http.StatusFound)
	body := assertStatus(t, router, client, http.MethodGet, "/chats/1", nil, http.StatusOK).Body.String()
	for _, expected := range []string{"受控工具", "/chats/1/tools", "current_time", "text_stats", "calculator", "不执行系统命令"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected tool UI to contain %q, body: %s", expected, body)
		}
	}

	assertStatus(t, router, client, http.MethodPost, "/chats/1/tools", url.Values{"tool_name": {"calculator"}, "tool_input": {"12 * (3 + 4)"}}, http.StatusFound)
	body = assertStatus(t, router, client, http.MethodGet, "/chats/1", nil, http.StatusOK).Body.String()
	for _, expected := range []string{"工具 calculator 执行成功", "结果：84", "calculator · success"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected tool result %q, body: %s", expected, body)
		}
	}
}

func TestV02MultipleModelConfigsRoutes(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"multi-api@example.test"}, "password": {"secret1"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats", url.Values{"name": {"Multi API Chat"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/settings/model", url.Values{
		"name":          {"Primary API"},
		"provider":      {"openai-compatible"},
		"base_url":      {"https://primary.example.test/v1"},
		"api_key":       {"test-key"},
		"default_model": {"test-model"},
		"models":        {"test-model"},
	}, http.StatusOK)
	assertStatus(t, router, client, http.MethodPost, "/settings/model", url.Values{
		"name":          {"Backup API"},
		"provider":      {"openai-compatible"},
		"base_url":      {"https://backup.example.test/v1"},
		"api_key":       {"test-key"},
		"default_model": {"backup-model"},
		"models":        {"backup-model"},
	}, http.StatusOK)

	body := assertStatus(t, router, client, http.MethodGet, "/settings/model", nil, http.StatusOK).Body.String()
	for _, expected := range []string{"Primary API", "Backup API", "test-model", "backup-model", "路由 #2", "路由 #3", "openai-compatible", "https://primary.example.test/v1"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected settings page to contain %q, body: %s", expected, body)
		}
	}
	chatBody := assertStatus(t, router, client, http.MethodGet, "/chats/1", nil, http.StatusOK).Body.String()
	for _, expected := range []string{"路由 #2 · Primary API / test-model", "路由 #3 · Backup API / backup-model"} {
		if !strings.Contains(chatBody, expected) {
			t.Fatalf("expected chat page to contain model choice %q, body: %s", expected, chatBody)
		}
	}

	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles", url.Values{
		"model_choice": {"2::test-model"},
		"name":         {"Primary"},
		"persona":      {"persona"},
		"reply_style":  {"concise"},
	}, http.StatusFound)
	chatBody = assertStatus(t, router, client, http.MethodGet, "/chats/1", nil, http.StatusOK).Body.String()
	for _, expected := range []string{"路由 #2 · Primary API · openai-compatible · test-model", "route-chip"} {
		if !strings.Contains(chatBody, expected) {
			t.Fatalf("expected role route metadata %q, body: %s", expected, chatBody)
		}
	}
	assertStatus(t, router, client, http.MethodPost, "/settings/model/3/delete", nil, http.StatusFound)
	body = assertStatus(t, router, client, http.MethodGet, "/settings/model", nil, http.StatusOK).Body.String()
	if strings.Contains(body, "Backup API") {
		t.Fatalf("expected unused backup config to be deleted, body: %s", body)
	}
	rec := assertStatus(t, router, client, http.MethodPost, "/settings/model/2/delete", nil, http.StatusBadRequest)
	if !strings.Contains(rec.Body.String(), "used by") {
		t.Fatalf("expected used config delete to be blocked, body: %s", rec.Body.String())
	}
}

func TestThemeToggleMarkupOnPrimaryPages(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	router := NewRouter(app.NewServices(newFakeStore(), fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"theme@example.test"}, "password": {"secret1"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats", url.Values{"name": {"Theme Chat"}}, http.StatusFound)

	for _, path := range []string{"/chats", "/chats/1", "/settings/model", "/health"} {
		body := assertStatus(t, router, client, http.MethodGet, path, nil, http.StatusOK).Body.String()
		for _, expected := range []string{`data-theme-toggle`, "黑夜模式", `/static/theme.js`} {
			if !strings.Contains(body, expected) {
				t.Fatalf("expected %s to contain theme markup %q, body: %s", path, expected, body)
			}
		}
	}
}

func TestTokenUsagePage(t *testing.T) {
	t.Setenv("TEMPLATE_GLOB", "../../web/templates/*.html")
	t.Setenv("STATIC_DIR", "../../web/static")
	st := newFakeStore()
	router := NewRouter(app.NewServices(st, fakeAI{}), fakeRedis{})
	client := newTestClient()

	assertStatus(t, router, client, http.MethodPost, "/register", url.Values{"email": {"usage@example.test"}, "password": {"secret1"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats", url.Values{"name": {"Usage Chat"}}, http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/settings/model", url.Values{
		"name":          {"Usage API"},
		"provider":      {"openai-compatible"},
		"base_url":      {"https://example.test/v1"},
		"api_key":       {"test-key"},
		"default_model": {"test-model"},
		"models":        {"test-model"},
	}, http.StatusOK)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles", roleForm("Architect"), http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/roles", roleForm("Reviewer"), http.StatusFound)
	assertStatus(t, router, client, http.MethodPost, "/chats/1/messages", url.Values{"content": {"hello"}}, http.StatusFound)

	body := assertStatus(t, router, client, http.MethodGet, "/usage", nil, http.StatusOK).Body.String()
	for _, expected := range []string{"Token 统计", "今天", "最近 7 天", "Prompt", "Completion", "Total", "test-model", "30", "只统计 Token 数"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected usage page to contain %q, body: %s", expected, body)
		}
	}
}

func roleForm(name string) url.Values {
	return url.Values{
		"model_choice":     {"2::test-model"},
		"name":             {name},
		"avatar":           {"/uploads/avatars/test.png"},
		"persona":          {"MVP persona"},
		"reply_style":      {"concise"},
		"reasoning_effort": {""},
	}
}

type testClient struct {
	cookies []*http.Cookie
}

func newTestClient() *testClient {
	return &testClient{}
}

func assertStatus(t *testing.T, handler http.Handler, client *testClient, method, path string, form url.Values, status int) *httptest.ResponseRecorder {
	t.Helper()
	return assertStatusWithHeaders(t, handler, client, method, path, form, nil, status)
}

func assertStatusWithHeaders(t *testing.T, handler http.Handler, client *testClient, method, path string, form url.Values, headers map[string]string, status int) *httptest.ResponseRecorder {
	t.Helper()
	if client != nil && requiresCSRFFromClient(method, path) {
		ensureCSRFCookieForClient(t, handler, client)
		if form == nil {
			form = url.Values{}
		}
		if token := csrfCookieValue(client); token != "" {
			form.Set("csrf_token", token)
		}
	}
	var body *strings.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if client != nil {
		for _, cookie := range client.cookies {
			req.AddCookie(cookie)
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != status {
		t.Fatalf("%s %s expected status %d, got %d; body: %s", method, path, status, rec.Code, rec.Body.String())
	}
	if client != nil {
		client.cookies = mergeCookies(client.cookies, rec.Result().Cookies())
	}
	return rec
}

func ensureCSRFCookieForClient(t *testing.T, handler http.Handler, client *testClient) {
	t.Helper()
	if csrfCookieValue(client) != "" {
		return
	}
	req := httptest.NewRequest(http.MethodGet, "/login", strings.NewReader(""))
	for _, cookie := range client.cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	client.cookies = mergeCookies(client.cookies, rec.Result().Cookies())
}

func csrfCookieValue(client *testClient) string {
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

func requiresCSRFFromClient(method, path string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}
	return !strings.HasPrefix(path, "/login") && !strings.HasPrefix(path, "/register")
}

func mergeCookies(existing, updates []*http.Cookie) []*http.Cookie {
	out := append([]*http.Cookie(nil), existing...)
	for _, update := range updates {
		replaced := false
		for i, cookie := range out {
			if cookie.Name == update.Name {
				if update.MaxAge < 0 {
					out = append(out[:i], out[i+1:]...)
				} else {
					out[i] = update
				}
				replaced = true
				break
			}
		}
		if !replaced && update.MaxAge >= 0 {
			out = append(out, update)
		}
	}
	return out
}

func waitForMessages(t *testing.T, handler http.Handler, client *testClient, path string, expected []string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		rec := assertStatus(t, handler, client, http.MethodGet, path, nil, http.StatusOK)
		body := rec.Body.String()
		ok := true
		for _, item := range expected {
			if !strings.Contains(body, item) {
				ok = false
				break
			}
		}
		if ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for messages %v, last body: %s", expected, body)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
