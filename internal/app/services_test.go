package app

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"ai_chat/internal/ai"
	"ai_chat/internal/domain"
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

func testUserContext() context.Context {
	return domain.WithUserID(context.Background(), 1)
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
			for idx := range f.messages {
				if f.messages[idx].RoleID != nil && *f.messages[idx].RoleID == roleID {
					f.messages[idx].RoleID = nil
				}
			}
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
	return []string{"model", "backup-model"}, nil
}

type capturingAI struct {
	mu           sync.Mutex
	replyInputs  []ai.ReplyInput
	reviewInputs []ai.ReviewInput
}

func (c *capturingAI) GenerateReply(ctx context.Context, input ai.ReplyInput) (ai.Reply, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.replyInputs = append(c.replyInputs, input)
	return ai.Reply{Content: "reply from " + input.Role.Name}, nil
}

func (c *capturingAI) GenerateReview(ctx context.Context, input ai.ReviewInput) (ai.Reply, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reviewInputs = append(c.reviewInputs, input)
	return ai.Reply{Content: "review from " + input.Role.Name}, nil
}

func (c *capturingAI) ListModels(ctx context.Context, config domain.ModelConfig) ([]string, error) {
	return []string{"model"}, nil
}

type slowAI struct {
	started chan string
	release chan struct{}
}

func newSlowAI() *slowAI {
	return &slowAI{started: make(chan string, 2), release: make(chan struct{})}
}

func (s *slowAI) GenerateReply(ctx context.Context, input ai.ReplyInput) (ai.Reply, error) {
	s.started <- input.Role.Name
	select {
	case <-s.release:
		return ai.Reply{Content: "reply from " + input.Role.Name}, nil
	case <-ctx.Done():
		return ai.Reply{}, ctx.Err()
	}
}

func (s *slowAI) GenerateReview(ctx context.Context, input ai.ReviewInput) (ai.Reply, error) {
	return ai.Reply{Content: "review from " + input.Role.Name}, nil
}

func (s *slowAI) ListModels(ctx context.Context, config domain.ModelConfig) ([]string, error) {
	return []string{"model"}, nil
}

func saveTestConfig(t *testing.T, ctx context.Context, svc *Services, models string) domain.ModelConfig {
	t.Helper()
	config, err := svc.SaveModelConfig(ctx, "Test API", "openai-compatible", "https://example.test/v1", "key", "model", models)
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestRegisterLoginAndSession(t *testing.T) {
	ctx := context.Background()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	user, token, expiresAt, err := svc.Register(ctx, "USER@example.test", "secret1")
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "user@example.test" || token == "" || !expiresAt.After(time.Now()) {
		t.Fatalf("unexpected register result: %#v token=%q expires=%s", user, token, expiresAt)
	}
	if _, _, _, err := svc.Register(ctx, "user@example.test", "secret1"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected duplicate email validation, got %v", err)
	}
	loggedIn, loginToken, _, err := svc.Login(ctx, "user@example.test", "secret1")
	if err != nil {
		t.Fatal(err)
	}
	if loggedIn.ID != user.ID || loginToken == "" {
		t.Fatalf("unexpected login result: %#v token=%q", loggedIn, loginToken)
	}
	sessionUser, err := svc.UserBySession(ctx, loginToken)
	if err != nil {
		t.Fatal(err)
	}
	if sessionUser.ID != user.ID {
		t.Fatalf("expected session user %d, got %d", user.ID, sessionUser.ID)
	}
	if err := svc.Logout(ctx, loginToken); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UserBySession(ctx, loginToken); err == nil {
		t.Fatal("expected logged out session to be invalid")
	}
}

func TestSendUserMessageRequiresTwoRoles(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "MVP")
	if err != nil {
		t.Fatal(err)
	}
	config := saveTestConfig(t, ctx, svc, "model")
	if _, err := svc.AddRole(ctx, chat.ID, config.ID, "A", "", "persona", "style", "model", ""); err != nil {
		t.Fatal(err)
	}
	_, err = svc.SendUserMessage(ctx, chat.ID, "hello")
	if err == nil || !strings.Contains(err.Error(), "at least two AI roles") {
		t.Fatalf("expected two-role MVP block, got %v", err)
	}
}

func TestSendUserMessageSavesUserAndTwoAIReplies(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "MVP")
	if err != nil {
		t.Fatal(err)
	}
	config := saveTestConfig(t, ctx, svc, "model")
	if _, err := svc.AddRole(ctx, chat.ID, config.ID, "Architect", "", "persona", "style", "model", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRole(ctx, chat.ID, config.ID, "Reviewer", "", "persona", "style", "model", ""); err != nil {
		t.Fatal(err)
	}
	result, err := svc.SendUserMessage(ctx, chat.ID, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if result.UserMessage.SenderType != domain.SenderUser {
		t.Fatalf("expected user sender, got %s", result.UserMessage.SenderType)
	}
	if len(result.AIMessages) != 2 {
		t.Fatalf("expected two AI messages, got %d", len(result.AIMessages))
	}
	if len(st.messages) != 3 {
		t.Fatalf("expected three stored messages, got %d", len(st.messages))
	}
}

func TestSendUserMessageGeneratesFirstRoundRepliesConcurrently(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	aiClient := newSlowAI()
	svc := NewServices(st, aiClient)
	chat, err := svc.CreateChat(ctx, "Fast Chat")
	if err != nil {
		t.Fatal(err)
	}
	config := saveTestConfig(t, ctx, svc, "model")
	for _, name := range []string{"Architect", "Reviewer"} {
		if _, err := svc.AddRole(ctx, chat.ID, config.ID, name, "", "persona", "style", "model", ""); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan error, 1)
	go func() {
		_, err := svc.SendUserMessage(ctx, chat.ID, "hello")
		done <- err
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-aiClient.started:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("expected both first-round AI calls to start before either one is released")
		}
	}
	close(aiClient.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestSetChatTopic(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "Topic Chat")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.SetChatTopic(ctx, chat.ID, "  v0.3 用户体验优化  ")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Topic != "v0.3 用户体验优化" {
		t.Fatalf("expected trimmed topic, got %q", updated.Topic)
	}
	detail, err := svc.GetChat(ctx, chat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Chat.Topic != "v0.3 用户体验优化" {
		t.Fatalf("expected topic on detail, got %q", detail.Chat.Topic)
	}
	tooLong := strings.Repeat("好", 501)
	if _, err := svc.SetChatTopic(ctx, chat.ID, tooLong); err == nil || !strings.Contains(err.Error(), "at most 500") {
		t.Fatalf("expected topic length validation, got %v", err)
	}
}

func TestAddChatFileAndPassesFilesToAI(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	aiClient := &capturingAI{}
	svc := NewServices(st, aiClient)
	chat, err := svc.CreateChat(ctx, "File Chat")
	if err != nil {
		t.Fatal(err)
	}
	config := saveTestConfig(t, ctx, svc, "model")
	if _, err := svc.AddRole(ctx, chat.ID, config.ID, "Architect", "", "persona", "style", "model", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRole(ctx, chat.ID, config.ID, "Reviewer", "", "persona", "style", "model", ""); err != nil {
		t.Fatal(err)
	}
	file, err := svc.AddChatFile(ctx, chat.ID, "brief.md", "/tmp/brief.md", "text/markdown", 24, "项目目标：降低延迟")
	if err != nil {
		t.Fatal(err)
	}
	if file.UserID != 1 || file.OriginalName != "brief.md" {
		t.Fatalf("unexpected file: %#v", file)
	}
	detail, err := svc.GetChat(ctx, chat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Files) != 1 || detail.Files[0].ExtractedText != "项目目标：降低延迟" {
		t.Fatalf("expected file on chat detail, got %#v", detail.Files)
	}
	if _, err := svc.SendUserMessage(ctx, chat.ID, "请分析文件"); err != nil {
		t.Fatal(err)
	}
	if len(aiClient.replyInputs) != 2 {
		t.Fatalf("expected two reply inputs, got %d", len(aiClient.replyInputs))
	}
	for _, input := range aiClient.replyInputs {
		if len(input.Files) != 1 || input.Files[0].OriginalName != "brief.md" {
			t.Fatalf("expected uploaded file to be passed to AI, got %#v", input.Files)
		}
	}
}

func TestRunToolCreatesSystemMessageAndExecution(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "Tool Chat")
	if err != nil {
		t.Fatal(err)
	}
	execution, err := svc.RunTool(ctx, chat.ID, "calculator", "12 * (3 + 4)")
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != domain.ToolExecutionSuccess || execution.Result != "84" {
		t.Fatalf("unexpected execution: %#v", execution)
	}
	if len(st.messages) != 1 || st.messages[0].SenderType != domain.SenderSystem || !strings.Contains(st.messages[0].Content, "84") {
		t.Fatalf("expected system tool message, got %#v", st.messages)
	}
	detail, err := svc.GetChat(ctx, chat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Tools) != 1 || detail.Tools[0].ToolName != "calculator" {
		t.Fatalf("expected tool execution on detail, got %#v", detail.Tools)
	}
}

func TestRunToolRecordsFailures(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "Tool Chat")
	if err != nil {
		t.Fatal(err)
	}
	execution, err := svc.RunTool(ctx, chat.ID, "calculator", "1 / 0")
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != domain.ToolExecutionFailed || !strings.Contains(execution.Error, "division by zero") {
		t.Fatalf("expected failed execution, got %#v", execution)
	}
	if len(st.messages) != 1 || !strings.Contains(st.messages[0].Content, "执行失败") {
		t.Fatalf("expected failed system message, got %#v", st.messages)
	}
}

func TestRolesUseTheirSelectedModelConfigs(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	capture := &capturingAI{}
	svc := NewServices(st, capture)
	chat, err := svc.CreateChat(ctx, "Multi API")
	if err != nil {
		t.Fatal(err)
	}
	fast, err := svc.SaveModelConfig(ctx, "Fast API", "openai-compatible", "https://fast.test/v1", "key", "fast-model", "fast-model")
	if err != nil {
		t.Fatal(err)
	}
	deep, err := svc.SaveModelConfig(ctx, "Deep API", "openai-compatible", "https://deep.test/v1", "key", "deep-model", "deep-model")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRole(ctx, chat.ID, fast.ID, "Fast", "", "persona", "style", "fast-model", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRole(ctx, chat.ID, deep.ID, "Deep", "", "persona", "style", "deep-model", ""); err != nil {
		t.Fatal(err)
	}
	roles, err := st.ListRoles(ctx, chat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if roles[0].ModelConfigID != fast.ID || roles[1].ModelConfigID != deep.ID {
		t.Fatalf("expected roles to bind different configs, got %#v", roles)
	}
	if _, err := svc.SendUserMessage(ctx, chat.ID, "hello"); err != nil {
		t.Fatal(err)
	}
	if len(capture.replyInputs) != 2 {
		t.Fatalf("expected two routed reply calls, got %d", len(capture.replyInputs))
	}
	routes := map[string]string{}
	for _, input := range capture.replyInputs {
		routes[input.Role.Name] = input.ModelConfig.Name + "|" + input.ModelConfig.Provider + "|" + input.ModelConfig.BaseURL + "|" + input.Role.Model
	}
	if routes["Fast"] != "Fast API|openai-compatible|https://fast.test/v1|fast-model" {
		t.Fatalf("unexpected fast route: %q", routes["Fast"])
	}
	if routes["Deep"] != "Deep API|openai-compatible|https://deep.test/v1|deep-model" {
		t.Fatalf("unexpected deep route: %q", routes["Deep"])
	}
}

func TestAIReviewUsesSelectedModelRoutes(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	capture := &capturingAI{}
	svc := NewServices(st, capture)
	chat, err := svc.CreateChat(ctx, "Review Routes")
	if err != nil {
		t.Fatal(err)
	}
	fast, err := svc.SaveModelConfig(ctx, "Fast API", "openai-compatible", "https://fast.test/v1", "key", "fast-model", "fast-model")
	if err != nil {
		t.Fatal(err)
	}
	deep, err := svc.SaveModelConfig(ctx, "Deep API", "openai-compatible", "https://deep.test/v1", "key", "deep-model", "deep-model")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRole(ctx, chat.ID, fast.ID, "Fast", "", "persona", "style", "fast-model", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRole(ctx, chat.ID, deep.ID, "Deep", "", "persona", "style", "deep-model", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetChatAIReview(ctx, chat.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendUserMessage(ctx, chat.ID, "hello"); err != nil {
		t.Fatal(err)
	}
	if len(capture.reviewInputs) != 1 {
		t.Fatalf("expected enabled AI review to run for short message, got %d review calls", len(capture.reviewInputs))
	}
	if _, err := svc.SendUserMessage(ctx, chat.ID, "please compare the tradeoffs in this plan and call out the main risk"); err != nil {
		t.Fatal(err)
	}
	if len(capture.reviewInputs) != 2 {
		t.Fatalf("expected one routed review call per message, got %d", len(capture.reviewInputs))
	}
	routes := map[string]string{}
	for _, input := range capture.reviewInputs {
		routes[input.Role.Name] = input.ModelConfig.Name + "|" + input.ModelConfig.BaseURL + "|" + input.Role.Model
	}
	for name, route := range routes {
		switch name {
		case "Fast":
			if route != "Fast API|https://fast.test/v1|fast-model" {
				t.Fatalf("unexpected fast review route: %q", route)
			}
		case "Deep":
			if route != "Deep API|https://deep.test/v1|deep-model" {
				t.Fatalf("unexpected deep review route: %q", route)
			}
		default:
			t.Fatalf("unexpected review role %q with route %q", name, route)
		}
	}
}

func TestDeleteModelConfigBlocksWhenUsedByRole(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "Delete Config")
	if err != nil {
		t.Fatal(err)
	}
	config := saveTestConfig(t, ctx, svc, "model")
	if _, err := svc.AddRole(ctx, chat.ID, config.ID, "A", "", "persona", "style", "model", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteModelConfig(ctx, config.ID); err == nil || !strings.Contains(err.Error(), "used by") {
		t.Fatalf("expected delete to be blocked for used config, got %v", err)
	}
}

func TestCheckModelConfigFetchesModels(t *testing.T) {
	ctx := testUserContext()
	svc := NewServices(newFakeStore(), fakeAI{})
	config, err := svc.CheckModelConfig(ctx, "openai-compatible", "https://example.test/v1", "key")
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Models) != 2 {
		t.Fatalf("expected two models, got %v", config.Models)
	}
	if config.DefaultModel != "model" {
		t.Fatalf("expected first model as default, got %q", config.DefaultModel)
	}
}

func TestDeleteRoleRemovesRoleFromFutureReplies(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "MVP")
	if err != nil {
		t.Fatal(err)
	}
	config := saveTestConfig(t, ctx, svc, "model")
	roleA, err := svc.AddRole(ctx, chat.ID, config.ID, "Architect", "", "persona", "style", "model", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRole(ctx, chat.ID, config.ID, "Reviewer", "", "persona", "style", "model", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteRole(ctx, chat.ID, roleA.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendUserMessage(ctx, chat.ID, "hello"); err == nil || !strings.Contains(err.Error(), "at least two AI roles") {
		t.Fatalf("expected two-role block after deleting role, got %v", err)
	}
}

func TestListMessagesAfterReturnsNewerMessages(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "MVP")
	if err != nil {
		t.Fatal(err)
	}
	first, err := st.CreateMessage(ctx, domain.Message{ChatID: chat.ID, SenderType: domain.SenderUser, SenderName: "User", Content: "old"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateMessage(ctx, domain.Message{ChatID: chat.ID, SenderType: domain.SenderAI, SenderName: "AI", Content: "new"}); err != nil {
		t.Fatal(err)
	}
	messages, err := svc.ListMessagesAfter(ctx, chat.ID, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].Content != "new" {
		t.Fatalf("expected one newer message, got %#v", messages)
	}
}

func TestUpdateRoleAndSpeakingPermissionAffectReplies(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "MVP")
	if err != nil {
		t.Fatal(err)
	}
	config := saveTestConfig(t, ctx, svc, "model\nbackup-model")
	roleA, err := svc.AddRole(ctx, chat.ID, config.ID, "Architect", "/uploads/avatars/test.png", "persona", "style", "model", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRole(ctx, chat.ID, config.ID, "Reviewer", "", "persona", "style", "model", ""); err != nil {
		t.Fatal(err)
	}
	roleC, err := svc.AddRole(ctx, chat.ID, config.ID, "Editor", "", "persona", "style", "model", "")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.UpdateRole(ctx, chat.ID, roleA.ID, config.ID, "Strategist", "ST", "new persona", "direct", "backup-model", "high", false)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Strategist" || updated.Avatar != "ST" || updated.Model != "backup-model" || updated.ReasoningEffort != "high" || updated.CanSpeak {
		t.Fatalf("unexpected updated role: %#v", updated)
	}
	result, err := svc.SendUserMessage(ctx, chat.ID, "hello")
	if err != nil {
		t.Fatal(err)
	}
	for _, msg := range result.AIMessages {
		if msg.SenderName == "Strategist" {
			t.Fatalf("muted role should not reply: %#v", result.AIMessages)
		}
	}
	if len(result.AIMessages) != 2 {
		t.Fatalf("expected two replies from remaining speaking roles, got %d", len(result.AIMessages))
	}
	if _, err := svc.ToggleRoleSpeaking(ctx, chat.ID, roleA.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ToggleRoleSpeaking(ctx, chat.ID, roleC.ID); err != nil {
		t.Fatal(err)
	}
	result, err = svc.SendUserMessage(ctx, chat.ID, "again")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AIMessages) != 2 {
		t.Fatalf("expected two replies after toggling, got %d", len(result.AIMessages))
	}
	if _, err := svc.ToggleRoleSpeaking(ctx, chat.ID, roleA.ID); err != nil {
		t.Fatal(err)
	}
	_, err = svc.SendUserMessage(ctx, chat.ID, "blocked")
	if err == nil || !strings.Contains(err.Error(), "speaking permission") {
		t.Fatalf("expected speaking permission block, got %v", err)
	}
}

func TestRoleReasoningEffortValidation(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "MVP")
	if err != nil {
		t.Fatal(err)
	}
	config := saveTestConfig(t, ctx, svc, "model")
	role, err := svc.AddRole(ctx, chat.ID, config.ID, "Architect", "", "persona", "style", "model", "medium")
	if err != nil {
		t.Fatal(err)
	}
	if role.ReasoningEffort != "medium" {
		t.Fatalf("expected medium reasoning effort, got %#v", role)
	}
	if _, err := svc.UpdateRole(ctx, chat.ID, role.ID, config.ID, "Architect", "", "persona", "style", "model", "invalid", true); err == nil || !strings.Contains(err.Error(), "reasoning effort") {
		t.Fatalf("expected reasoning effort validation error, got %v", err)
	}
}

func TestSendUserMessageRecordsTokenUsage(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "Usage")
	if err != nil {
		t.Fatal(err)
	}
	config := saveTestConfig(t, ctx, svc, "model")
	if _, err := svc.AddRole(ctx, chat.ID, config.ID, "Architect", "", "persona", "style", "model", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRole(ctx, chat.ID, config.ID, "Reviewer", "", "persona", "style", "model", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendUserMessage(ctx, chat.ID, "hello"); err != nil {
		t.Fatal(err)
	}
	stats, err := svc.TokenUsageStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Today.PromptTokens != 20 || stats.Today.CompletionTokens != 10 || stats.Today.TotalTokens != 30 {
		t.Fatalf("unexpected token usage stats: %#v", stats)
	}
}

func TestSendUserMessageAsyncRecordsTokenUsageWithUserContext(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "Async Usage")
	if err != nil {
		t.Fatal(err)
	}
	config := saveTestConfig(t, ctx, svc, "model")
	if _, err := svc.AddRole(ctx, chat.ID, config.ID, "Architect", "", "persona", "style", "model", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddRole(ctx, chat.ID, config.ID, "Reviewer", "", "persona", "style", "model", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SendUserMessageAsync(ctx, chat.ID, "hello"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		stats, err := svc.TokenUsageStats(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if stats.Today.TotalTokens == 30 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected async token usage to be recorded, got %#v", stats)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestAIReviewAddsSelectiveReviewReplyWhenEnabled(t *testing.T) {
	ctx := testUserContext()
	st := newFakeStore()
	svc := NewServices(st, fakeAI{})
	chat, err := svc.CreateChat(ctx, "Review Chat")
	if err != nil {
		t.Fatal(err)
	}
	config := saveTestConfig(t, ctx, svc, "model")
	for _, name := range []string{"Architect", "Reviewer", "Editor"} {
		if _, err := svc.AddRole(ctx, chat.ID, config.ID, name, "", "persona", "style", "model", ""); err != nil {
			t.Fatal(err)
		}
	}
	result, err := svc.SendUserMessage(ctx, chat.ID, "without review")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AIMessages) != 3 {
		t.Fatalf("expected all speaking roles to reply while review disabled, got %d", len(result.AIMessages))
	}
	if _, err := svc.SetChatAIReview(ctx, chat.ID, true); err != nil {
		t.Fatal(err)
	}
	result, err = svc.SendUserMessage(ctx, chat.ID, "short")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AIMessages) != 4 {
		t.Fatalf("expected short message to include review when enabled, got %d AI messages", len(result.AIMessages))
	}
	if !strings.Contains(result.AIMessages[3].Content, "review from") {
		t.Fatalf("expected short message review reply at the end, got %#v", result.AIMessages)
	}
	result, err = svc.SendUserMessage(ctx, chat.ID, "with review, please compare these options and identify the main risk")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AIMessages) != 4 {
		t.Fatalf("expected three first-round replies plus one review reply, got %d", len(result.AIMessages))
	}
	if !strings.Contains(result.AIMessages[3].Content, "review from") {
		t.Fatalf("expected one review reply at the end, got %#v", result.AIMessages)
	}
}
