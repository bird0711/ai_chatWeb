package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai_chat/internal/ai"
	"ai_chat/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrValidation = errors.New("validation error")
	ErrMVPBlocked = errors.New("mvp blocked")
)

type Services struct {
	store Store
	ai    ai.Client
}

type Store interface {
	CreateChat(ctx context.Context, name string) (domain.Chat, error)
	CreateUser(ctx context.Context, email, passwordHash string) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUser(ctx context.Context, userID int64) (domain.User, error)
	CreateSession(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	GetSessionUser(ctx context.Context, tokenHash string, now time.Time) (domain.User, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	ListChats(ctx context.Context) ([]domain.Chat, error)
	GetChat(ctx context.Context, chatID int64) (domain.Chat, error)
	UpdateChatAIReview(ctx context.Context, chatID int64, enabled bool) (domain.Chat, error)
	UpdateChatTopic(ctx context.Context, chatID int64, topic string) (domain.Chat, error)
	CreateRole(ctx context.Context, role domain.Role) (domain.Role, error)
	ListRoles(ctx context.Context, chatID int64) ([]domain.Role, error)
	GetRole(ctx context.Context, chatID, roleID int64) (domain.Role, error)
	UpdateRole(ctx context.Context, role domain.Role) (domain.Role, error)
	DeleteRole(ctx context.Context, chatID, roleID int64) error
	SaveModelConfig(ctx context.Context, config domain.ModelConfig) (domain.ModelConfig, error)
	ListModelConfigs(ctx context.Context) ([]domain.ModelConfig, error)
	GetModelConfig(ctx context.Context) (domain.ModelConfig, error)
	GetModelConfigByID(ctx context.Context, configID int64) (domain.ModelConfig, error)
	DeleteModelConfig(ctx context.Context, configID int64) error
	CountRolesByModelConfig(ctx context.Context, configID int64) (int, error)
	Ping(ctx context.Context) error
	CreateMessage(ctx context.Context, message domain.Message) (domain.Message, error)
	ListMessages(ctx context.Context, chatID int64) ([]domain.Message, error)
	ListMessagesAfter(ctx context.Context, chatID, afterID int64) ([]domain.Message, error)
	CreateTokenUsage(ctx context.Context, usage domain.TokenUsage) (domain.TokenUsage, error)
	TokenUsageStats(ctx context.Context, now time.Time) (domain.TokenUsageStats, error)
	DeleteChat(ctx context.Context, chatID int64) error
}

func NewServices(st Store, aiClient ai.Client) *Services {
	return &Services{store: st, ai: aiClient}
}

func (s *Services) Register(ctx context.Context, email, password string) (domain.User, string, time.Time, error) {
	email = normalizeEmail(email)
	if err := validateCredentials(email, password); err != nil {
		return domain.User{}, "", time.Time{}, err
	}
	_, err := s.store.GetUserByEmail(ctx, email)
	if err == nil {
		return domain.User{}, "", time.Time{}, fmt.Errorf("%w: email already exists", ErrValidation)
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, "", time.Time{}, err
	}
	passwordHash, err := hashPassword(password)
	if err != nil {
		return domain.User{}, "", time.Time{}, err
	}
	user, err := s.store.CreateUser(ctx, email, passwordHash)
	if err != nil {
		return domain.User{}, "", time.Time{}, err
	}
	token, expiresAt, err := s.createSession(ctx, user.ID)
	if err != nil {
		return domain.User{}, "", time.Time{}, err
	}
	return user, token, expiresAt, nil
}

func (s *Services) Login(ctx context.Context, email, password string) (domain.User, string, time.Time, error) {
	email = normalizeEmail(email)
	if email == "" || strings.TrimSpace(password) == "" {
		return domain.User{}, "", time.Time{}, fmt.Errorf("%w: email and password are required", ErrValidation)
	}
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, "", time.Time{}, fmt.Errorf("%w: invalid email or password", ErrValidation)
		}
		return domain.User{}, "", time.Time{}, err
	}
	if !verifyPassword(user.PasswordHash, password) {
		return domain.User{}, "", time.Time{}, fmt.Errorf("%w: invalid email or password", ErrValidation)
	}
	token, expiresAt, err := s.createSession(ctx, user.ID)
	if err != nil {
		return domain.User{}, "", time.Time{}, err
	}
	return user, token, expiresAt, nil
}

func (s *Services) UserBySession(ctx context.Context, token string) (domain.User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return domain.User{}, domain.ErrNotFound
	}
	return s.store.GetSessionUser(ctx, hashSessionToken(token), time.Now())
}

func (s *Services) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return s.store.DeleteSession(ctx, hashSessionToken(token))
}

func (s *Services) CreateChat(ctx context.Context, name string) (domain.Chat, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Chat{}, fmt.Errorf("%w: chat name is required", ErrValidation)
	}
	return s.store.CreateChat(ctx, name)
}

func (s *Services) createSession(ctx context.Context, userID int64) (string, time.Time, error) {
	token, err := randomToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	if err := s.store.CreateSession(ctx, userID, hashSessionToken(token), expiresAt); err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func validateCredentials(email, password string) error {
	if email == "" || !strings.Contains(email, "@") {
		return fmt.Errorf("%w: valid email is required", ErrValidation)
	}
	if len(password) < 6 {
		return fmt.Errorf("%w: password must be at least 6 characters", ErrValidation)
	}
	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func verifyPassword(passwordHash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}

func randomToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Services) ListChats(ctx context.Context) ([]domain.Chat, error) {
	return s.store.ListChats(ctx)
}

func (s *Services) DeleteChat(ctx context.Context, chatID int64) error {
	if _, err := s.store.GetChat(ctx, chatID); err != nil {
		return err
	}
	return s.store.DeleteChat(ctx, chatID)
}

func (s *Services) TokenUsageStats(ctx context.Context) (domain.TokenUsageStats, error) {
	return s.store.TokenUsageStats(ctx, time.Now())
}

func (s *Services) SetChatAIReview(ctx context.Context, chatID int64, enabled bool) (domain.Chat, error) {
	if _, err := s.store.GetChat(ctx, chatID); err != nil {
		return domain.Chat{}, err
	}
	return s.store.UpdateChatAIReview(ctx, chatID, enabled)
}

func (s *Services) SetChatTopic(ctx context.Context, chatID int64, topic string) (domain.Chat, error) {
	if _, err := s.store.GetChat(ctx, chatID); err != nil {
		return domain.Chat{}, err
	}
	topic = strings.TrimSpace(topic)
	if len([]rune(topic)) > 500 {
		return domain.Chat{}, fmt.Errorf("%w: chat topic must be at most 500 characters", ErrValidation)
	}
	return s.store.UpdateChatTopic(ctx, chatID, topic)
}

func (s *Services) GetChat(ctx context.Context, chatID int64) (domain.ChatDetail, error) {
	chat, err := s.store.GetChat(ctx, chatID)
	if err != nil {
		return domain.ChatDetail{}, err
	}
	roles, err := s.store.ListRoles(ctx, chatID)
	if err != nil {
		return domain.ChatDetail{}, err
	}
	messages, err := s.store.ListMessages(ctx, chatID)
	if err != nil {
		return domain.ChatDetail{}, err
	}
	return domain.ChatDetail{Chat: chat, Roles: roles, Messages: messages}, nil
}

func (s *Services) AddRole(ctx context.Context, chatID int64, modelConfigID int64, name, avatar, persona, style, model, reasoningEffort string) (domain.Role, error) {
	if _, err := s.store.GetChat(ctx, chatID); err != nil {
		return domain.Role{}, err
	}
	role := domain.Role{
		ChatID:          chatID,
		ModelConfigID:   modelConfigID,
		Name:            strings.TrimSpace(name),
		Avatar:          strings.TrimSpace(avatar),
		Persona:         strings.TrimSpace(persona),
		ReplyStyle:      strings.TrimSpace(style),
		Model:           strings.TrimSpace(model),
		ReasoningEffort: cleanReasoningEffort(reasoningEffort),
		CanSpeak:        true,
	}
	if err := s.validateRole(ctx, role, "adding roles"); err != nil {
		return domain.Role{}, err
	}
	return s.store.CreateRole(ctx, role)
}

func (s *Services) GetRole(ctx context.Context, chatID, roleID int64) (domain.Role, error) {
	if _, err := s.store.GetChat(ctx, chatID); err != nil {
		return domain.Role{}, err
	}
	return s.store.GetRole(ctx, chatID, roleID)
}

func (s *Services) UpdateRole(ctx context.Context, chatID, roleID, modelConfigID int64, name, avatar, persona, style, model, reasoningEffort string, canSpeak bool) (domain.Role, error) {
	if _, err := s.store.GetChat(ctx, chatID); err != nil {
		return domain.Role{}, err
	}
	current, err := s.store.GetRole(ctx, chatID, roleID)
	if err != nil {
		return domain.Role{}, err
	}
	role := domain.Role{
		ID:              current.ID,
		ChatID:          current.ChatID,
		ModelConfigID:   modelConfigID,
		Name:            strings.TrimSpace(name),
		Avatar:          strings.TrimSpace(avatar),
		Persona:         strings.TrimSpace(persona),
		ReplyStyle:      strings.TrimSpace(style),
		Model:           strings.TrimSpace(model),
		ReasoningEffort: cleanReasoningEffort(reasoningEffort),
		CanSpeak:        canSpeak,
	}
	if err := s.validateRole(ctx, role, "editing roles"); err != nil {
		return domain.Role{}, err
	}
	return s.store.UpdateRole(ctx, role)
}

func (s *Services) ToggleRoleSpeaking(ctx context.Context, chatID, roleID int64) (domain.Role, error) {
	if _, err := s.store.GetChat(ctx, chatID); err != nil {
		return domain.Role{}, err
	}
	role, err := s.store.GetRole(ctx, chatID, roleID)
	if err != nil {
		return domain.Role{}, err
	}
	role.CanSpeak = !role.CanSpeak
	return s.store.UpdateRole(ctx, role)
}

func (s *Services) DeleteRole(ctx context.Context, chatID, roleID int64) error {
	if _, err := s.store.GetChat(ctx, chatID); err != nil {
		return err
	}
	return s.store.DeleteRole(ctx, chatID, roleID)
}

func (s *Services) SaveModelConfig(ctx context.Context, name, provider, baseURL, apiKey, defaultModel, modelsText string) (domain.ModelConfig, error) {
	models := parseModels(modelsText)
	config := domain.ModelConfig{
		Name:         strings.TrimSpace(name),
		Provider:     strings.TrimSpace(provider),
		BaseURL:      strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		APIKey:       strings.TrimSpace(apiKey),
		DefaultModel: strings.TrimSpace(defaultModel),
		Models:       models,
	}
	if config.Provider == "" {
		config.Provider = "openai-compatible"
	}
	if config.Name == "" {
		return domain.ModelConfig{}, fmt.Errorf("%w: API config name is required", ErrValidation)
	}
	if config.BaseURL == "" {
		return domain.ModelConfig{}, fmt.Errorf("%w: base URL is required", ErrValidation)
	}
	if config.APIKey == "" {
		return domain.ModelConfig{}, fmt.Errorf("%w: API key is required", ErrValidation)
	}
	if config.DefaultModel == "" {
		return domain.ModelConfig{}, fmt.Errorf("%w: default model is required", ErrValidation)
	}
	if len(config.Models) == 0 {
		return domain.ModelConfig{}, fmt.Errorf("%w: supported models are required", ErrValidation)
	}
	if !modelAllowed(config.DefaultModel, config.Models) {
		return domain.ModelConfig{}, fmt.Errorf("%w: default model must be in supported models", ErrValidation)
	}
	return s.store.SaveModelConfig(ctx, config)
}

func (s *Services) CheckModelConfig(ctx context.Context, provider, baseURL, apiKey string) (domain.ModelConfig, error) {
	config := domain.ModelConfig{
		Provider: strings.TrimSpace(provider),
		BaseURL:  strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		APIKey:   strings.TrimSpace(apiKey),
	}
	if config.Provider == "" {
		config.Provider = "openai-compatible"
	}
	if config.BaseURL == "" {
		return domain.ModelConfig{}, fmt.Errorf("%w: base URL is required", ErrValidation)
	}
	if config.APIKey == "" {
		return domain.ModelConfig{}, fmt.Errorf("%w: API key is required", ErrValidation)
	}
	models, err := s.ai.ListModels(ctx, config)
	if err != nil {
		return config, err
	}
	config.Models = models
	config.DefaultModel = models[0]
	return config, nil
}

func (s *Services) ListModelConfigs(ctx context.Context) ([]domain.ModelConfig, error) {
	return s.store.ListModelConfigs(ctx)
}

func (s *Services) GetModelConfig(ctx context.Context) (domain.ModelConfig, error) {
	return s.store.GetModelConfig(ctx)
}

func (s *Services) DeleteModelConfig(ctx context.Context, configID int64) error {
	if _, err := s.store.GetModelConfigByID(ctx, configID); err != nil {
		return err
	}
	count, err := s.store.CountRolesByModelConfig(ctx, configID)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: model API config is used by %d role(s)", ErrValidation, count)
	}
	return s.store.DeleteModelConfig(ctx, configID)
}

func (s *Services) Health(ctx context.Context) error {
	return s.store.Ping(ctx)
}

func (s *Services) SendUserMessage(ctx context.Context, chatID int64, content string) (domain.MessageResult, error) {
	chat, roles, config, userMessage, history, err := s.prepareUserMessage(ctx, chatID, content)
	if err != nil {
		return domain.MessageResult{}, err
	}
	result := s.generateAIReplies(ctx, chat, roles, config, userMessage, history)
	s.appendAIReviews(ctx, chat, roles, config, userMessage, history, &result)
	if len(result.AIMessages) < 2 {
		if len(result.Errors) > 0 {
			return result, fmt.Errorf("%w: fewer than two AI replies succeeded: %s", ErrMVPBlocked, strings.Join(result.Errors, "; "))
		}
		return result, fmt.Errorf("%w: fewer than two AI replies succeeded", ErrMVPBlocked)
	}
	return result, nil
}

func (s *Services) SendUserMessageAsync(ctx context.Context, chatID int64, content string) (domain.Message, error) {
	chat, roles, config, userMessage, history, err := s.prepareUserMessage(ctx, chatID, content)
	if err != nil {
		return domain.Message{}, err
	}
	go func() {
		replyCtx, cancel := backgroundUserContext(ctx, 2*time.Minute)
		defer cancel()
		result := s.generateAIReplies(replyCtx, chat, roles, config, userMessage, history)
		s.appendAIReviews(replyCtx, chat, roles, config, userMessage, history, &result)
		if len(result.Errors) > 0 || len(result.AIMessages) < 2 {
			message := "AI 回复未完成"
			if len(result.Errors) > 0 {
				message += "：" + strings.Join(result.Errors, "；")
			}
			if len(result.AIMessages) < 2 {
				message += "。本轮少于两个 AI 回复成功。"
			}
			_, _ = s.store.CreateMessage(replyCtx, domain.Message{
				ChatID:     chatID,
				SenderType: domain.SenderSystem,
				SenderName: "系统",
				Content:    message,
			})
		}
	}()
	return userMessage, nil
}

func backgroundUserContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	base := context.Background()
	if userID, ok := domain.UserIDFromContext(ctx); ok {
		base = domain.WithUserID(base, userID)
	}
	return context.WithTimeout(base, timeout)
}

func (s *Services) ListMessagesAfter(ctx context.Context, chatID, afterID int64) ([]domain.Message, error) {
	if afterID < 0 {
		return nil, fmt.Errorf("%w: after id must not be negative", ErrValidation)
	}
	if _, err := s.store.GetChat(ctx, chatID); err != nil {
		return nil, err
	}
	return s.store.ListMessagesAfter(ctx, chatID, afterID)
}

func (s *Services) prepareUserMessage(ctx context.Context, chatID int64, content string) (domain.Chat, []domain.Role, map[int64]domain.ModelConfig, domain.Message, []domain.Message, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return domain.Chat{}, nil, nil, domain.Message{}, nil, fmt.Errorf("%w: message content is required", ErrValidation)
	}
	chat, err := s.store.GetChat(ctx, chatID)
	if err != nil {
		return domain.Chat{}, nil, nil, domain.Message{}, nil, err
	}
	roles, err := s.store.ListRoles(ctx, chatID)
	if err != nil {
		return domain.Chat{}, nil, nil, domain.Message{}, nil, err
	}
	roles = speakingRoles(roles)
	if len(roles) < 2 {
		return domain.Chat{}, nil, nil, domain.Message{}, nil, fmt.Errorf("%w: at least two AI roles with speaking permission are required before sending", ErrMVPBlocked)
	}
	configs, err := s.roleModelConfigs(ctx, roles)
	if err != nil {
		return domain.Chat{}, nil, nil, domain.Message{}, nil, err
	}

	userMessage, err := s.store.CreateMessage(ctx, domain.Message{
		ChatID:     chatID,
		SenderType: domain.SenderUser,
		SenderName: "User",
		Content:    content,
	})
	if err != nil {
		return domain.Chat{}, nil, nil, domain.Message{}, nil, err
	}
	history, err := s.store.ListMessages(ctx, chatID)
	if err != nil {
		return domain.Chat{}, nil, nil, domain.Message{}, nil, err
	}
	return chat, roles, configs, userMessage, history, nil
}

func (s *Services) validateRole(ctx context.Context, role domain.Role, action string) error {
	if role.Name == "" {
		return fmt.Errorf("%w: role name is required", ErrValidation)
	}
	if role.Persona == "" {
		return fmt.Errorf("%w: role persona is required", ErrValidation)
	}
	if role.ReplyStyle == "" {
		return fmt.Errorf("%w: role reply style is required", ErrValidation)
	}
	if role.Model == "" {
		return fmt.Errorf("%w: role model is required", ErrValidation)
	}
	if role.ReasoningEffort != "" && !validReasoningEffort(role.ReasoningEffort) {
		return fmt.Errorf("%w: reasoning effort must be default, low, medium, or high", ErrValidation)
	}
	if role.ModelConfigID <= 0 {
		return fmt.Errorf("%w: model API config is required", ErrValidation)
	}
	config, err := s.store.GetModelConfigByID(ctx, role.ModelConfigID)
	if errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("%w: model API config is required before %s", ErrMVPBlocked, action)
	}
	if err != nil {
		return err
	}
	if !modelAllowed(role.Model, config.Models) {
		return fmt.Errorf("%w: selected model is not in model API settings", ErrValidation)
	}
	return nil
}

func cleanReasoningEffort(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validReasoningEffort(value string) bool {
	switch value {
	case "low", "medium", "high":
		return true
	default:
		return false
	}
}

func speakingRoles(roles []domain.Role) []domain.Role {
	out := make([]domain.Role, 0, len(roles))
	for _, role := range roles {
		if role.CanSpeak {
			out = append(out, role)
		}
	}
	return out
}

func (s *Services) roleModelConfigs(ctx context.Context, roles []domain.Role) (map[int64]domain.ModelConfig, error) {
	configs := map[int64]domain.ModelConfig{}
	for _, role := range roles {
		if role.ModelConfigID <= 0 {
			return nil, fmt.Errorf("%w: role %s has no model API config", ErrMVPBlocked, role.Name)
		}
		if _, ok := configs[role.ModelConfigID]; ok {
			continue
		}
		config, err := s.store.GetModelConfigByID(ctx, role.ModelConfigID)
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: role %s model API config was not found", ErrMVPBlocked, role.Name)
		}
		if err != nil {
			return nil, err
		}
		configs[role.ModelConfigID] = config
	}
	return configs, nil
}

func (s *Services) generateAIReplies(ctx context.Context, chat domain.Chat, roles []domain.Role, configs map[int64]domain.ModelConfig, userMessage domain.Message, history []domain.Message) domain.MessageResult {
	result := domain.MessageResult{UserMessage: userMessage}
	for _, role := range roles {
		config := configs[role.ModelConfigID]
		reply, err := s.ai.GenerateReply(ctx, ai.ReplyInput{
			Chat:        chat,
			Role:        role,
			Messages:    history,
			ModelConfig: config,
			UserMessage: userMessage,
		})
		if err != nil {
			result.Errors = append(result.Errors, role.Name+": "+err.Error())
			continue
		}
		roleID := role.ID
		msg, err := s.store.CreateMessage(ctx, domain.Message{
			ChatID:       chat.ID,
			SenderType:   domain.SenderAI,
			SenderName:   role.Name,
			SenderAvatar: role.Avatar,
			RoleID:       &roleID,
			Content:      reply.Content,
		})
		if err != nil {
			result.Errors = append(result.Errors, role.Name+": save reply: "+err.Error())
			continue
		}
		s.recordTokenUsage(ctx, chat.ID, msg.ID, role, config, reply, &result)
		result.AIMessages = append(result.AIMessages, msg)
	}
	return result
}

func (s *Services) appendAIReviews(ctx context.Context, chat domain.Chat, roles []domain.Role, configs map[int64]domain.ModelConfig, userMessage domain.Message, history []domain.Message, result *domain.MessageResult) {
	if !chat.AIReviewEnabled || len(result.AIMessages) < 2 {
		return
	}
	limit := 2
	if len(roles) < limit {
		limit = len(roles)
	}
	for _, role := range roles[:limit] {
		config := configs[role.ModelConfigID]
		reply, err := s.ai.GenerateReview(ctx, ai.ReviewInput{
			Chat:              chat,
			Role:              role,
			Messages:          history,
			ModelConfig:       config,
			UserMessage:       userMessage,
			FirstRoundReplies: result.AIMessages,
		})
		if err != nil {
			result.Errors = append(result.Errors, role.Name+" review: "+err.Error())
			continue
		}
		roleID := role.ID
		msg, err := s.store.CreateMessage(ctx, domain.Message{
			ChatID:       chat.ID,
			SenderType:   domain.SenderAI,
			SenderName:   role.Name,
			SenderAvatar: role.Avatar,
			RoleID:       &roleID,
			Content:      reply.Content,
		})
		if err != nil {
			result.Errors = append(result.Errors, role.Name+" save review: "+err.Error())
			continue
		}
		s.recordTokenUsage(ctx, chat.ID, msg.ID, role, config, reply, result)
		result.AIMessages = append(result.AIMessages, msg)
	}
}

func (s *Services) recordTokenUsage(ctx context.Context, chatID, messageID int64, role domain.Role, config domain.ModelConfig, reply ai.Reply, result *domain.MessageResult) {
	if reply.Usage.PromptTokens == 0 && reply.Usage.CompletionTokens == 0 && reply.Usage.TotalTokens == 0 {
		return
	}
	total := reply.Usage.TotalTokens
	if total == 0 {
		total = reply.Usage.PromptTokens + reply.Usage.CompletionTokens
	}
	_, err := s.store.CreateTokenUsage(ctx, domain.TokenUsage{
		ChatID:           chatID,
		MessageID:        messageID,
		RoleID:           role.ID,
		ModelConfigID:    config.ID,
		Model:            role.Model,
		PromptTokens:     reply.Usage.PromptTokens,
		CompletionTokens: reply.Usage.CompletionTokens,
		TotalTokens:      total,
	})
	if err != nil {
		result.Errors = append(result.Errors, role.Name+": save token usage: "+err.Error())
	}
}

func parseModels(modelsText string) []string {
	parts := strings.FieldsFunc(modelsText, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ','
	})
	models := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		model := strings.TrimSpace(part)
		if model == "" || seen[model] {
			continue
		}
		seen[model] = true
		models = append(models, model)
	}
	return models
}

func modelAllowed(model string, models []string) bool {
	for _, allowed := range models {
		if model == allowed {
			return true
		}
	}
	return false
}
