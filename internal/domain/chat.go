package domain

import "time"

type Chat struct {
	ID              int64
	UserID          int64
	Name            string
	Topic           string
	AIReviewEnabled bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ChatDetail struct {
	Chat     Chat
	Roles    []Role
	Messages []Message
	Files    []ChatFile
	Tools    []ToolExecution
}

type ChatFile struct {
	ID            int64
	UserID        int64
	ChatID        int64
	OriginalName  string
	StoragePath   string
	ContentType   string
	SizeBytes     int64
	ExtractedText string
	CreatedAt     time.Time
}
