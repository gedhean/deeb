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
	ID                   uuid.UUID          `json:"id"                               db:"id"`
	ChangePlanID         uuid.UUID          `json:"change_plan_id"                   db:"change_plan_id"`
	ExecutorID           uuid.UUID          `json:"executor_id"                      db:"executor_id"`
	Status               ExecutionLogStatus `json:"status"                           db:"status"`
	ExecutionDurationMs  int                `json:"execution_duration_ms"            db:"execution_duration_ms"`
	RowsAffected         *int               `json:"rows_affected,omitempty"          db:"rows_affected"`
	ErrorMessage         *string            `json:"error_message,omitempty"          db:"error_message"`
	SQLContent           *string            `json:"sql_content,omitempty"            db:"sql_content"`
	DatabaseID           *uuid.UUID         `json:"database_id,omitempty"            db:"database_id"`
	DatabaseName         *string            `json:"database_name,omitempty"          db:"database_name"`
	DatabaseInstanceName *string            `json:"database_instance_name,omitempty" db:"database_instance_name"`
	DatabaseType         *string            `json:"database_type,omitempty"          db:"database_type"`
	StartedAt            time.Time          `json:"started_at"                       db:"started_at"`
	CompletedAt          time.Time          `json:"completed_at"                     db:"completed_at"`
	CreatedAt            time.Time          `json:"created_at"                       db:"created_at"`
}

// ExecutionLogResponse represents an execution log response
type ExecutionLogResponse struct {
	ID                   uuid.UUID          `json:"id"`
	ChangePlanID         uuid.UUID          `json:"change_plan_id"`
	ExecutorID           uuid.UUID          `json:"executor_id"`
	Status               ExecutionLogStatus `json:"status"`
	ExecutionDurationMs  int                `json:"execution_duration_ms"`
	RowsAffected         *int               `json:"rows_affected,omitempty"`
	ErrorMessage         *string            `json:"error_message,omitempty"`
	SQLContent           *string            `json:"sql_content,omitempty"`
	DatabaseID           *uuid.UUID         `json:"database_id,omitempty"`
	DatabaseName         *string            `json:"database_name,omitempty"`
	DatabaseInstanceName *string            `json:"database_instance_name,omitempty"`
	DatabaseType         *string            `json:"database_type,omitempty"`
	StartedAt            time.Time          `json:"started_at"`
	CompletedAt          time.Time          `json:"completed_at"`
	CreatedAt            time.Time          `json:"created_at"`
}

// ToResponse converts ExecutionLog to ExecutionLogResponse
func (l *ExecutionLog) ToResponse() *ExecutionLogResponse {
	return &ExecutionLogResponse{
		ID:                   l.ID,
		ChangePlanID:         l.ChangePlanID,
		ExecutorID:           l.ExecutorID,
		Status:               l.Status,
		ExecutionDurationMs:  l.ExecutionDurationMs,
		RowsAffected:         l.RowsAffected,
		ErrorMessage:         l.ErrorMessage,
		SQLContent:           l.SQLContent,
		DatabaseID:           l.DatabaseID,
		DatabaseName:         l.DatabaseName,
		DatabaseInstanceName: l.DatabaseInstanceName,
		DatabaseType:         l.DatabaseType,
		StartedAt:            l.StartedAt,
		CompletedAt:          l.CompletedAt,
		CreatedAt:            l.CreatedAt,
	}
}
