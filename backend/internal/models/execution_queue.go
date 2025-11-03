package models

import (
	"time"

	"github.com/google/uuid"
)

// QueueStatus represents the status of a queued execution
type QueueStatus string

const (
	QueueStatusPending    QueueStatus = "Pending"
	QueueStatusInProgress QueueStatus = "InProgress"
	QueueStatusCompleted  QueueStatus = "Completed"
	QueueStatusFailed     QueueStatus = "Failed"
	QueueStatusCancelled  QueueStatus = "Cancelled"
)

// ExecutionQueue represents a queued Change Plan execution
type ExecutionQueue struct {
	ID               uuid.UUID   `json:"id"                      db:"id"`
	ChangePlanID     uuid.UUID   `json:"change_plan_id"          db:"change_plan_id"`
	TargetDatabaseID uuid.UUID   `json:"target_database_id"      db:"target_database_id"`
	ExecutorID       uuid.UUID   `json:"executor_id"             db:"executor_id"`
	Status           QueueStatus `json:"status"                  db:"status"`
	Position         int         `json:"position"                db:"position"`
	Priority         int         `json:"priority"                db:"priority"`
	ErrorMessage     *string     `json:"error_message,omitempty" db:"error_message"`
	QueuedAt         time.Time   `json:"queued_at"               db:"queued_at"`
	StartedAt        *time.Time  `json:"started_at,omitempty"    db:"started_at"`
	CompletedAt      *time.Time  `json:"completed_at,omitempty"  db:"completed_at"`
	CreatedAt        time.Time   `json:"created_at"              db:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"              db:"updated_at"`
}

// ExecutionQueueResponse represents a queue item response
type ExecutionQueueResponse struct {
	ID               uuid.UUID   `json:"id"`
	ChangePlanID     uuid.UUID   `json:"change_plan_id"`
	TargetDatabaseID uuid.UUID   `json:"target_database_id"`
	ExecutorID       uuid.UUID   `json:"executor_id"`
	Status           QueueStatus `json:"status"`
	Position         int         `json:"position"`
	Priority         int         `json:"priority"`
	ErrorMessage     *string     `json:"error_message,omitempty"`
	QueuedAt         time.Time   `json:"queued_at"`
	StartedAt        *time.Time  `json:"started_at,omitempty"`
	CompletedAt      *time.Time  `json:"completed_at,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// ToResponse converts ExecutionQueue to ExecutionQueueResponse
func (q *ExecutionQueue) ToResponse() *ExecutionQueueResponse {
	return &ExecutionQueueResponse{
		ID:               q.ID,
		ChangePlanID:     q.ChangePlanID,
		TargetDatabaseID: q.TargetDatabaseID,
		ExecutorID:       q.ExecutorID,
		Status:           q.Status,
		Position:         q.Position,
		Priority:         q.Priority,
		ErrorMessage:     q.ErrorMessage,
		QueuedAt:         q.QueuedAt,
		StartedAt:        q.StartedAt,
		CompletedAt:      q.CompletedAt,
		CreatedAt:        q.CreatedAt,
		UpdatedAt:        q.UpdatedAt,
	}
}
