package store

import (
	"context"
	"time"

	"ai_chat/internal/domain"
)

type Store interface {
	Close() error
	Migrate(ctx context.Context) error

	CreateUser(ctx context.Context, email, passwordHash string) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetUser(ctx context.Context, userID int64) (domain.User, error)
	CreateSession(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	GetSessionUser(ctx context.Context, tokenHash string, now time.Time) (domain.User, error)
	DeleteSession(ctx context.Context, tokenHash string) error

	CreateChat(ctx context.Context, name string) (domain.Chat, error)
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
