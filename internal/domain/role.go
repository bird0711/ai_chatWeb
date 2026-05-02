package domain

import "time"

type Role struct {
	ID              int64
	ChatID          int64
	ModelConfigID   int64
	Name            string
	Avatar          string
	Persona         string
	ReplyStyle      string
	Model           string
	ReasoningEffort string
	CanSpeak        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
