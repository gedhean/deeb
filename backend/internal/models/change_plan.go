package models

import (
	"time"

	"github.com/google/uuid"
)

// ChangePlanStatus represents the current state of a change plan
type ChangePlanStatus string

const (
	ChangePlanStatusDraft    ChangePlanStatus = "Draft"
	ChangePlanStatusOpen     ChangePlanStatus = "Open"
	ChangePlanStatusApproved ChangePlanStatus = "Approved"
	ChangePlanStatusClosed   ChangePlanStatus = "Closed"
	ChangePlanStatusExecuted ChangePlanStatus = "Executed"
	ChangePlanStatusFailed   ChangePlanStatus = "Failed"
)

// String returns the string representation of ChangePlanStatus
func (s ChangePlanStatus) String() string {
	return string(s)
}

// IsValid checks if the status is one of the defined values
func (s ChangePlanStatus) IsValid() bool {
	switch s {
	case ChangePlanStatusDraft, ChangePlanStatusOpen, ChangePlanStatusApproved,
		ChangePlanStatusClosed, ChangePlanStatusExecuted, ChangePlanStatusFailed:
		return true
	}
	return false
}

// ChangeType represents the type of database change
type ChangeType string

const (
	ChangeTypeDDL ChangeType = "DDL"
	ChangeTypeDML ChangeType = "DML"
)

// RiskLevel represents the risk level of a database change
type RiskLevel string

const (
	RiskLevelLow    RiskLevel = "Low"
	RiskLevelMedium RiskLevel = "Medium"
	RiskLevelHigh   RiskLevel = "High"
)

// ChangePlan represents a database change request
type ChangePlan struct {
	ID                  uuid.UUID        `json:"id"                              db:"id"`
	ProjectID           uuid.UUID        `json:"project_id"                      db:"project_id"`
	TargetDatabaseID    uuid.UUID        `json:"target_database_id"              db:"target_database_id"`
	Title               string           `json:"title"                           db:"title"`
	Description         string           `json:"description"                     db:"description"`
	ChangeType          ChangeType       `json:"change_type"                     db:"change_type"`
	SQLContent          string           `json:"sql_content"                     db:"sql_content"`
	RollbackSQL         *string          `json:"rollback_sql,omitempty"          db:"rollback_sql"`
	Status              ChangePlanStatus `json:"status"                          db:"status"`
	RiskLevel           RiskLevel        `json:"risk_level"                      db:"risk_level"`
	AuthorID            uuid.UUID        `json:"author_id"                       db:"author_id"`
	AssignedReviewerID  *uuid.UUID       `json:"assigned_reviewer_id,omitempty"  db:"assigned_reviewer_id"`
	ScheduledFor        *time.Time       `json:"scheduled_for,omitempty"         db:"scheduled_for"`
	ExecutedAt          *time.Time       `json:"executed_at,omitempty"           db:"executed_at"`
	ExecutionDurationMs *int             `json:"execution_duration_ms,omitempty" db:"execution_duration_ms"`
	AffectedRows        *int             `json:"affected_rows,omitempty"         db:"affected_rows"`
	ErrorMessage        *string          `json:"error_message,omitempty"         db:"error_message"`
	CreatedAt           time.Time        `json:"created_at"                      db:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"                      db:"updated_at"`
}

// CreateChangePlanRequest represents the request to create a new change plan
type CreateChangePlanRequest struct {
	ProjectID        uuid.UUID `json:"project_id"         validate:"required"`
	TargetDatabaseID uuid.UUID `json:"target_database_id"  validate:"required"`
	Title            string    `json:"title"               validate:"required,max=255"`
	Description      string    `json:"description"         validate:"required"`
	ChangeType       ChangeType `json:"change_type"        validate:"required,oneof=DDL DML"`
	SQLContent       string    `json:"sql_content"         validate:"required"`
	RollbackSQL      *string   `json:"rollback_sql,omitempty"`
}

// UpdateChangePlanRequest represents the request to update a change plan
type UpdateChangePlanRequest struct {
	Title          *string          `json:"title,omitempty"          validate:"omitempty,max=255"`
	Description    *string          `json:"description,omitempty"`
	SQLContent     *string          `json:"sql_content,omitempty"`
	RollbackSQL    *string          `json:"rollback_sql,omitempty"`
	Status         *ChangePlanStatus `json:"status,omitempty"        validate:"omitempty,oneof=Draft Open Approved Closed Executed Failed"`
	RiskLevel      *RiskLevel        `json:"risk_level,omitempty"    validate:"omitempty,oneof=Low Medium High"`
	AssignedReviewerID *uuid.UUID    `json:"assigned_reviewer_id,omitempty"`
}

// ChangePlanResponse represents a change plan response
type ChangePlanResponse struct {
	ID                  uuid.UUID        `json:"id"`
	ProjectID           uuid.UUID        `json:"project_id"`
	TargetDatabaseID    uuid.UUID        `json:"target_database_id"`
	Title               string           `json:"title"`
	Description         string           `json:"description"`
	ChangeType          ChangeType       `json:"change_type"`
	SQLContent          string           `json:"sql_content"`
	RollbackSQL         *string          `json:"rollback_sql,omitempty"`
	Status              ChangePlanStatus `json:"status"`
	RiskLevel           RiskLevel        `json:"risk_level"`
	AuthorID            uuid.UUID        `json:"author_id"`
	AssignedReviewerID  *uuid.UUID       `json:"assigned_reviewer_id,omitempty"`
	ScheduledFor        *time.Time       `json:"scheduled_for,omitempty"`
	ExecutedAt          *time.Time       `json:"executed_at,omitempty"`
	ExecutionDurationMs *int             `json:"execution_duration_ms,omitempty"`
	AffectedRows        *int             `json:"affected_rows,omitempty"`
	ErrorMessage        *string          `json:"error_message,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

// ToResponse converts a ChangePlan to ChangePlanResponse
func (cp *ChangePlan) ToResponse() *ChangePlanResponse {
	return &ChangePlanResponse{
		ID:                  cp.ID,
		ProjectID:           cp.ProjectID,
		TargetDatabaseID:    cp.TargetDatabaseID,
		Title:               cp.Title,
		Description:         cp.Description,
		ChangeType:          cp.ChangeType,
		SQLContent:          cp.SQLContent,
		RollbackSQL:         cp.RollbackSQL,
		Status:              cp.Status,
		RiskLevel:           cp.RiskLevel,
		AuthorID:            cp.AuthorID,
		AssignedReviewerID:  cp.AssignedReviewerID,
		ScheduledFor:        cp.ScheduledFor,
		ExecutedAt:          cp.ExecutedAt,
		ExecutionDurationMs: cp.ExecutionDurationMs,
		AffectedRows:        cp.AffectedRows,
		ErrorMessage:        cp.ErrorMessage,
		CreatedAt:           cp.CreatedAt,
		UpdatedAt:           cp.UpdatedAt,
	}
}

