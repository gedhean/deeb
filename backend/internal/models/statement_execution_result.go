package models

import (
	"time"

	"github.com/google/uuid"
)

// StatementExecutionStatus represents the status of a statement execution
type StatementExecutionStatus string

const (
	StatementStatusCompleted StatementExecutionStatus = "Completed"
	StatementStatusFailed    StatementExecutionStatus = "Failed"
	StatementStatusSkipped   StatementExecutionStatus = "Skipped"
)

// StatementExecutionResult represents a single statement execution result
type StatementExecutionResult struct {
	ID                  uuid.UUID                `json:"id"                      db:"id"`
	ExecutionLogID      uuid.UUID                `json:"execution_log_id"        db:"execution_log_id"`
	StatementIndex      int                      `json:"statement_index"         db:"statement_index"`
	StatementSQL        string                   `json:"statement_sql"           db:"statement_sql"`
	Status              StatementExecutionStatus `json:"status"                  db:"status"`
	RowsAffected        *int                     `json:"rows_affected,omitempty" db:"rows_affected"`
	ExecutionDurationMs int                      `json:"execution_duration_ms"   db:"execution_duration_ms"`
	ErrorMessage        *string                  `json:"error_message,omitempty" db:"error_message"`
	StartLine           *int                     `json:"start_line,omitempty"    db:"start_line"`
	StartColumn         *int                     `json:"start_column,omitempty"  db:"start_column"`
	EndLine             *int                     `json:"end_line,omitempty"      db:"end_line"`
	EndColumn           *int                     `json:"end_column,omitempty"    db:"end_column"`
	CreatedAt           time.Time                `json:"created_at"              db:"created_at"`
}

// StatementExecutionResultResponse represents a statement execution result API response
type StatementExecutionResultResponse struct {
	ID                  uuid.UUID                `json:"id"`
	ExecutionLogID      uuid.UUID                `json:"execution_log_id"`
	StatementIndex      int                      `json:"statement_index"`
	StatementSQL        string                   `json:"statement_sql"`
	Status              StatementExecutionStatus `json:"status"`
	RowsAffected        *int                     `json:"rows_affected,omitempty"`
	ExecutionDurationMs int                      `json:"execution_duration_ms"`
	ErrorMessage        *string                  `json:"error_message,omitempty"`
	StartLine           *int                     `json:"start_line,omitempty"`
	StartColumn         *int                     `json:"start_column,omitempty"`
	EndLine             *int                     `json:"end_line,omitempty"`
	EndColumn           *int                     `json:"end_column,omitempty"`
	CreatedAt           time.Time                `json:"created_at"`
}

// ToResponse converts StatementExecutionResult to StatementExecutionResultResponse
func (s *StatementExecutionResult) ToResponse() *StatementExecutionResultResponse {
	return &StatementExecutionResultResponse{
		ID:                  s.ID,
		ExecutionLogID:      s.ExecutionLogID,
		StatementIndex:      s.StatementIndex,
		StatementSQL:        s.StatementSQL,
		Status:              s.Status,
		RowsAffected:        s.RowsAffected,
		ExecutionDurationMs: s.ExecutionDurationMs,
		ErrorMessage:        s.ErrorMessage,
		StartLine:           s.StartLine,
		StartColumn:         s.StartColumn,
		EndLine:             s.EndLine,
		EndColumn:           s.EndColumn,
		CreatedAt:           s.CreatedAt,
	}
}

// ExecutionLogWithStatements combines an execution log with its statement results
type ExecutionLogWithStatements struct {
	ExecutionLog
	StatementResults []StatementExecutionResult `json:"statement_results,omitempty"`
}

// ExecutionLogDetailResponse represents an execution log with statement details
type ExecutionLogDetailResponse struct {
	ExecutionLog     *ExecutionLogResponse               `json:"execution_log"`
	StatementResults []*StatementExecutionResultResponse `json:"statement_results"`
}
