package domain

import "time"

type ModelConfig struct {
	ID           int64
	UserID       int64
	Name         string
	Provider     string
	BaseURL      string
	APIKey       string
	DefaultModel string
	Models       []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
