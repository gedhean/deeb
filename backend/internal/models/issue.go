package models

import (
	"time"

	"github.com/google/uuid"
)

// IssueStatus represents the current state of a database change issue
type IssueStatus string

const (
	IssueStatusDraft           IssueStatus = "Draft"
	IssueStatusPendingReview   IssueStatus = "PendingReview"
	IssueStatusPendingApproval IssueStatus = "PendingApproval"
	IssueStatusApproved        IssueStatus = "Approved"
	IssueStatusScheduled       IssueStatus = "Scheduled"
	IssueStatusInProgress      IssueStatus = "InProgress"
	IssueStatusCompleted       IssueStatus = "Completed"
	IssueStatusFailed          IssueStatus = "Failed"
	IssueStatusCancelled       IssueStatus = "Cancelled"
)

// String returns the string representation of IssueStatus
func (s IssueStatus) String() string {
	return string(s)
}

// IsValid checks if the status is one of the defined values
func (s IssueStatus) IsValid() bool {
	switch s {
	case IssueStatusDraft, IssueStatusPendingReview, IssueStatusPendingApproval,
		IssueStatusApproved, IssueStatusScheduled, IssueStatusInProgress,
		IssueStatusCompleted, IssueStatusFailed, IssueStatusCancelled:
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
	RiskLevelLow      RiskLevel = "Low"
	RiskLevelMedium   RiskLevel = "Medium"
	RiskLevelHigh     RiskLevel = "High"
	RiskLevelCritical RiskLevel = "Critical"
)

// Issue represents a database change request
type Issue struct {
	ID                  uuid.UUID   `json:"id"                              db:"id"`
	ProjectID           uuid.UUID   `json:"project_id"                      db:"project_id"`
	TargetDatabaseID    uuid.UUID   `json:"target_database_id"              db:"target_database_id"`
	Title               string      `json:"title"                           db:"title"`
	Description         string      `json:"description"                     db:"description"`
	ChangeType          ChangeType  `json:"change_type"                     db:"change_type"`
	SQLContent          string      `json:"sql_content"                     db:"sql_content"`
	RollbackSQL         *string     `json:"rollback_sql,omitempty"          db:"rollback_sql"`
	Status              IssueStatus `json:"status"                          db:"status"`
	RiskLevel           RiskLevel   `json:"risk_level"                      db:"risk_level"`
	AuthorID            uuid.UUID   `json:"author_id"                       db:"author_id"`
	AssignedReviewerID  *uuid.UUID  `json:"assigned_reviewer_id,omitempty"  db:"assigned_reviewer_id"`
	ScheduledFor        *time.Time  `json:"scheduled_for,omitempty"         db:"scheduled_for"`
	ExecutedAt          *time.Time  `json:"executed_at,omitempty"           db:"executed_at"`
	ExecutionDurationMs *int        `json:"execution_duration_ms,omitempty" db:"execution_duration_ms"`
	AffectedRows        *int        `json:"affected_rows,omitempty"         db:"affected_rows"`
	ErrorMessage        *string     `json:"error_message,omitempty"         db:"error_message"`
	CreatedAt           time.Time   `json:"created_at"                      db:"created_at"`
	UpdatedAt           time.Time   `json:"updated_at"                      db:"updated_at"`
}

// Validate performs basic validation on the Issue
func (i *Issue) Validate() error {
	if i.Title == "" {
		return ErrInvalidIssue("title is required")
	}
	if i.Description == "" {
		return ErrInvalidIssue("description is required")
	}
	if i.SQLContent == "" {
		return ErrInvalidIssue("sql_content is required")
	}
	if i.ChangeType != ChangeTypeDDL && i.ChangeType != ChangeTypeDML {
		return ErrInvalidIssue("change_type must be DDL or DML")
	}
	if i.AuthorID == uuid.Nil {
		return ErrInvalidIssue("author_id is required")
	}
	if i.TargetDatabaseID == uuid.Nil {
		return ErrInvalidIssue("target_database_id is required")
	}
	if i.ProjectID == uuid.Nil {
		return ErrInvalidIssue("project_id is required")
	}
	if !i.Status.IsValid() {
		return ErrInvalidIssue("invalid status")
	}
	return nil
}
