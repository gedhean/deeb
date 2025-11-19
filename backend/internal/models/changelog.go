package models

import (
	"time"

	"github.com/google/uuid"
)

// ChangelogStatus represents the execution status of a change plan
type ChangelogStatus string

const (
	// ChangelogStatusPending indicates the change plan execution is pending
	ChangelogStatusPending ChangelogStatus = "pending"
	// ChangelogStatusDone indicates the change plan execution completed successfully
	ChangelogStatusDone ChangelogStatus = "done"
	// ChangelogStatusFailed indicates the change plan execution failed
	ChangelogStatusFailed ChangelogStatus = "failed"
)

// String returns the string representation of ChangelogStatus
func (s ChangelogStatus) String() string {
	return string(s)
}

// IsValid checks if the status is one of the defined values
func (s ChangelogStatus) IsValid() bool {
	switch s {
	case ChangelogStatusPending, ChangelogStatusDone, ChangelogStatusFailed:
		return true
	}
	return false
}

// Changelog represents a record of a change plan execution
// It tracks the execution status and links to schema versions before and after the change
type Changelog struct {
	ID                  uuid.UUID       `json:"id"                     db:"id"`
	DatabaseInstanceID  uuid.UUID       `json:"database_instance_id"   db:"database_instance_id"`
	DatabaseID          uuid.UUID       `json:"database_id"            db:"database_id"`
	ChangePlanID        uuid.UUID       `json:"change_plan_id"         db:"change_plan_id"`
	Status              ChangelogStatus `json:"status"                 db:"status"`
	PrevSchemaHistoryID *uuid.UUID      `json:"prev_schema_history_id" db:"prev_schema_history_id"`
	SchemaHistoryID     *uuid.UUID      `json:"schema_history_id"      db:"schema_history_id"`
	CreatedAt           time.Time       `json:"created_at"             db:"created_at"`
}

// CreateChangelogRequest represents the request to create a new changelog entry
type CreateChangelogRequest struct {
	DatabaseInstanceID  uuid.UUID       `json:"database_instance_id"             validate:"required"`
	DatabaseID          uuid.UUID       `json:"database_id"                      validate:"required"`
	ChangePlanID        uuid.UUID       `json:"change_plan_id"                   validate:"required"`
	Status              ChangelogStatus `json:"status"                           validate:"required"`
	PrevSchemaHistoryID *uuid.UUID      `json:"prev_schema_history_id,omitempty"`
}

// ChangelogResponse represents the API response for a changelog entry
type ChangelogResponse struct {
	ID                  uuid.UUID       `json:"id"`
	DatabaseInstanceID  uuid.UUID       `json:"database_instance_id"`
	DatabaseID          uuid.UUID       `json:"database_id"`
	ChangePlanID        uuid.UUID       `json:"change_plan_id"`
	Status              ChangelogStatus `json:"status"`
	PrevSchemaHistoryID *uuid.UUID      `json:"prev_schema_history_id,omitempty"`
	SchemaHistoryID     *uuid.UUID      `json:"schema_history_id,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
}

// ToResponse converts a Changelog model to a ChangelogResponse
func (c *Changelog) ToResponse() *ChangelogResponse {
	return &ChangelogResponse{
		ID:                  c.ID,
		DatabaseInstanceID:  c.DatabaseInstanceID,
		DatabaseID:          c.DatabaseID,
		ChangePlanID:        c.ChangePlanID,
		Status:              c.Status,
		PrevSchemaHistoryID: c.PrevSchemaHistoryID,
		SchemaHistoryID:     c.SchemaHistoryID,
		CreatedAt:           c.CreatedAt,
	}
}
