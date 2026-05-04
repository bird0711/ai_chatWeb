package ai

import (
	"context"

	"ai_chat/internal/domain"
)

type Reply struct {
	Content string
	Usage   Usage
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type Client interface {
	GenerateReply(ctx context.Context, input ReplyInput) (Reply, error)
	GenerateReview(ctx context.Context, input ReviewInput) (Reply, error)
	ListModels(ctx context.Context, config domain.ModelConfig) ([]string, error)
}

type ReplyInput struct {
	Chat        domain.Chat
	Role        domain.Role
	Messages    []domain.Message
	Files       []domain.ChatFile
	ModelConfig domain.ModelConfig
	UserMessage domain.Message
}

type ReviewInput struct {
	Chat              domain.Chat
	Role              domain.Role
	Messages          []domain.Message
	Files             []domain.ChatFile
	ModelConfig       domain.ModelConfig
	UserMessage       domain.Message
	FirstRoundReplies []domain.Message
}
