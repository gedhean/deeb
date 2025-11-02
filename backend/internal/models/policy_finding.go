package models

import (
	"time"

	"github.com/google/uuid"
)

// ValidatorType represents the type of validator that found a policy finding
type ValidatorType string

const (
	ValidatorTypeSyntaxError           ValidatorType = "syntax_error"
	ValidatorTypeSecurityRisk          ValidatorType = "security_risk"
	ValidatorTypePerformanceIssue      ValidatorType = "performance_issue"
	ValidatorTypeBackwardsIncompatible ValidatorType = "backwards_incompatible"
	ValidatorTypeBestPracticeViolation ValidatorType = "best_practice_violation"
)

// FindingSeverity represents the severity of a policy finding
type FindingSeverity string

const (
	FindingSeverityError   FindingSeverity = "Error"
	FindingSeverityWarning FindingSeverity = "Warning"
	FindingSeverityInfo    FindingSeverity = "Info"
)

// PolicyFinding represents a SQL Review finding (result) from policy evaluation
type PolicyFinding struct {
	ID            uuid.UUID       `json:"id"                      db:"id"`
	ChangePlanID  uuid.UUID       `json:"change_plan_id"          db:"change_plan_id"`
	ValidatorType ValidatorType   `json:"validator_type"          db:"validator_type"`
	Severity      FindingSeverity `json:"severity"                db:"severity"`
	RuleName      string          `json:"rule_name"               db:"rule_name"`
	Message       string          `json:"message"                 db:"message"`
	LineNumber    *int            `json:"line_number,omitempty"   db:"line_number"`
	ColumnNumber  *int            `json:"column_number,omitempty" db:"column_number"`
	CreatedAt     time.Time       `json:"created_at"              db:"created_at"`
}

// CreatePolicyFindingRequest represents the request to create a new policy finding
type CreatePolicyFindingRequest struct {
	ChangePlanID  uuid.UUID       `json:"change_plan_id"          validate:"required"`
	ValidatorType ValidatorType   `json:"validator_type"          validate:"required"`
	Severity      FindingSeverity `json:"severity"                validate:"required,oneof=Error Warning Info"`
	RuleName      string          `json:"rule_name"               validate:"required,max=255"`
	Message       string          `json:"message"                 validate:"required"`
	LineNumber    *int            `json:"line_number,omitempty"`
	ColumnNumber  *int            `json:"column_number,omitempty"`
}

// PolicyFindingResponse represents a policy finding response
type PolicyFindingResponse struct {
	ID            uuid.UUID       `json:"id"`
	ChangePlanID  uuid.UUID       `json:"change_plan_id"`
	ValidatorType ValidatorType   `json:"validator_type"`
	Severity      FindingSeverity `json:"severity"`
	RuleName      string          `json:"rule_name"`
	Message       string          `json:"message"`
	LineNumber    *int            `json:"line_number,omitempty"`
	ColumnNumber  *int            `json:"column_number,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// ToResponse converts a PolicyFinding to PolicyFindingResponse
func (p *PolicyFinding) ToResponse() *PolicyFindingResponse {
	return &PolicyFindingResponse{
		ID:            p.ID,
		ChangePlanID:  p.ChangePlanID,
		ValidatorType: p.ValidatorType,
		Severity:      p.Severity,
		RuleName:      p.RuleName,
		Message:       p.Message,
		LineNumber:    p.LineNumber,
		ColumnNumber:  p.ColumnNumber,
		CreatedAt:     p.CreatedAt,
	}
}
