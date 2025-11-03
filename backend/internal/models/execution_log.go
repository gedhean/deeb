package models

import (
	"time"

	"github.com/google/uuid"
)

// ExecutionLogStatus represents the status of an execution log entry
type ExecutionLogStatus string

const (
	ExecutionLogStatusCompleted ExecutionLogStatus = "Completed"
	ExecutionLogStatusFailed    ExecutionLogStatus = "Failed"
)

// ExecutionLog represents a single execution attempt log entry
type ExecutionLog struct {
	ID                  uuid.UUID          `json:"id"                   db:"id"`
	ChangePlanID        uuid.UUID          `json:"change_plan_id"       db:"change_plan_id"`
	ExecutorID          uuid.UUID          `json:"executor_id"          db:"executor_id"`
	Status              ExecutionLogStatus `json:"status"               db:"status"`
	ExecutionDurationMs int                `json:"execution_duration_ms" db:"execution_duration_ms"`
	RowsAffected        *int               `json:"rows_affected,omitempty" db:"rows_affected"`
	ErrorMessage        *string           `json:"error_message,omitempty" db:"error_message"`
	StartedAt           time.Time          `json:"started_at"           db:"started_at"`
	CompletedAt         time.Time          `json:"completed_at"         db:"completed_at"`
	CreatedAt           time.Time          `json:"created_at"           db:"created_at"`
}

// ExecutionLogResponse represents an execution log response
type ExecutionLogResponse struct {
	ID                  uuid.UUID          `json:"id"`
	ChangePlanID        uuid.UUID          `json:"change_plan_id"`
	ExecutorID          uuid.UUID          `json:"executor_id"`
	Status              ExecutionLogStatus `json:"status"`
	ExecutionDurationMs int                `json:"execution_duration_ms"`
	RowsAffected        *int               `json:"rows_affected,omitempty"`
	ErrorMessage        *string           `json:"error_message,omitempty"`
	StartedAt           time.Time          `json:"started_at"`
	CompletedAt         time.Time          `json:"completed_at"`
	CreatedAt           time.Time          `json:"created_at"`
}

// ToResponse converts ExecutionLog to ExecutionLogResponse
func (l *ExecutionLog) ToResponse() *ExecutionLogResponse {
	return &ExecutionLogResponse{
		ID:                  l.ID,
		ChangePlanID:        l.ChangePlanID,
		ExecutorID:          l.ExecutorID,
		Status:              l.Status,
		ExecutionDurationMs: l.ExecutionDurationMs,
		RowsAffected:        l.RowsAffected,
		ErrorMessage:        l.ErrorMessage,
		StartedAt:           l.StartedAt,
		CompletedAt:         l.CompletedAt,
		CreatedAt:           l.CreatedAt,
	}
}

