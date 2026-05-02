package domain

import "time"

type TokenUsage struct {
	ID               int64
	UserID           int64
	ChatID           int64
	MessageID        int64
	RoleID           int64
	ModelConfigID    int64
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CreatedAt        time.Time
}

type TokenUsageSummary struct {
	Label            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type TokenUsageStats struct {
	Today   TokenUsageSummary
	Recent7 TokenUsageSummary
	ByModel []TokenUsageSummary
}
