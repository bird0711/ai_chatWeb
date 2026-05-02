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
}
