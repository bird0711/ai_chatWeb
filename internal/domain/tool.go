package domain

import "time"

type ToolExecutionStatus string

const (
	ToolExecutionSuccess ToolExecutionStatus = "success"
	ToolExecutionFailed  ToolExecutionStatus = "failed"
)

type ToolExecution struct {
	ID        int64
	UserID    int64
	ChatID    int64
	MessageID int64
	ToolName  string
	Input     string
	Status    ToolExecutionStatus
	Result    string
	Error     string
	CreatedAt time.Time
}
