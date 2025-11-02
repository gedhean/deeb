package models

import (
	"time"

	"github.com/google/uuid"
)

// PolicyFinding represents a SQL Review finding (violation) from policy evaluation
type PolicyFinding struct {
	ID            uuid.UUID `json:"id"                    db:"id"`
	ChangePlanID  uuid.UUID `json:"change_plan_id"        db:"change_plan_id"`
	ValidatorType string    `json:"validator_type"        db:"validator_type"`
	Severity      string    `json:"severity"               db:"severity"`
	RuleName      string    `json:"rule_name"              db:"rule_name"`
	Message       string    `json:"message"               db:"message"`
	LineNumber    *int      `json:"line_number,omitempty" db:"line_number"`
	ColumnNumber  *int      `json:"column_number,omitempty" db:"column_number"`
	CreatedAt     time.Time `json:"created_at"            db:"created_at"`
}

// Validate performs basic validation on the PolicyFinding
func (p *PolicyFinding) Validate() error {
	if p.ChangePlanID == uuid.Nil {
		return ErrInvalidPolicyFinding("change_plan_id is required")
	}
	if p.ValidatorType == "" {
		return ErrInvalidPolicyFinding("validator_type is required")
	}
	if p.Severity == "" {
		return ErrInvalidPolicyFinding("severity is required")
	}
	if p.RuleName == "" {
		return ErrInvalidPolicyFinding("rule_name is required")
	}
	if p.Message == "" {
		return ErrInvalidPolicyFinding("message is required")
	}
	return nil
}

// ErrInvalidPolicyFinding represents a validation error for PolicyFinding
type ErrInvalidPolicyFinding string

func (e ErrInvalidPolicyFinding) Error() string {
	return string(e)
}
