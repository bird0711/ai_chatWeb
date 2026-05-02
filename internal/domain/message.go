package domain

import "time"

type SenderType string

const (
	SenderUser   SenderType = "user"
	SenderAI     SenderType = "ai"
	SenderSystem SenderType = "system"
)

type Message struct {
	ID           int64
	ChatID       int64
	SenderType   SenderType
	SenderName   string
	SenderAvatar string
	RoleID       *int64
	Content      string
	CreatedAt    time.Time
}

type MessageResult struct {
	UserMessage Message
	AIMessages  []Message
	Errors      []string
}
