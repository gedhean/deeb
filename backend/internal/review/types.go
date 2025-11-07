package review

import (
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/sql-reviewer/pkg/types"
)

// ReviewResultV2 represents enhanced review results with execution metadata
type ReviewResultV2 struct {
	// Valid indicates if the SQL passed all checks
	Valid bool `json:"valid"`

	// Violations contains all findings from the review
	Violations []ViolationV2 `json:"violations"`

	// Warnings contains warnings (non-blocking)
	Warnings []ViolationV2 `json:"warnings"`

	// Summary statistics
	TotalViolations int `json:"total_violations"`
	ErrorCount      int `json:"error_count"`
	WarningCount    int `json:"warning_count"`

	// Policy information
	PolicyID   *uuid.UUID `json:"policy_id,omitempty"`
	PolicyName string     `json:"policy_name,omitempty"`

	// Rule execution info
	EnabledRules  []string      `json:"enabled_rules"`
	SkippedRules  []SkippedRule `json:"skipped_rules"`
	ExecutedRules int           `json:"executed_rules"`

	// Execution metadata
	ExecutionTime time.Duration `json:"execution_time_ms"`
	ReviewedAt    time.Time     `json:"reviewed_at"`
}

// ViolationV2 represents a single SQL review violation
// Note: Named ViolationV2 to avoid conflict with legacy Violation type alias
type ViolationV2 struct {
	// RuleType is the rule that was violated
	RuleType string `json:"rule_type"`

	// Severity level (error, warning)
	Severity string `json:"severity"`

	// Title of the violation
	Title string `json:"title"`

	// Message describing the violation
	Message string `json:"message"`

	// Position in the SQL where violation occurred
	Line   int32 `json:"line,omitempty"`
	Column int32 `json:"column,omitempty"`

	// Error code
	Code int32 `json:"code,omitempty"`
}

// SkippedRule represents a rule that failed to execute
type SkippedRule struct {
	// RuleType is the rule that was skipped
	RuleType string `json:"rule_type"`

	// Reason why the rule was skipped
	Reason string `json:"reason"`

	// Error message if available
	Error string `json:"error,omitempty"`
}

// ConvertAdviceToViolation converts sql-reviewer Advice to ViolationV2
func ConvertAdviceToViolation(advice *types.Advice) ViolationV2 {
	severity := "info"
	switch advice.Status {
	case types.Advice_ERROR:
		severity = "error"
	case types.Advice_WARNING:
		severity = "warning"
	case types.Advice_SUCCESS:
		severity = "success"
	}

	violation := ViolationV2{
		Severity: severity,
		Title:    advice.Title,
		Message:  advice.Content,
		Code:     advice.Code,
	}

	if advice.StartPosition != nil {
		violation.Line = advice.StartPosition.Line
		violation.Column = advice.StartPosition.Column
	}

	return violation
}

// FilterByStatus filters violations by severity
func (r *ReviewResultV2) FilterByStatus(status string) []ViolationV2 {
	filtered := []ViolationV2{}
	for _, v := range r.Violations {
		if v.Severity == status {
			filtered = append(filtered, v)
		}
	}
	for _, v := range r.Warnings {
		if v.Severity == status {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// FilterByRuleType filters violations by rule type
func (r *ReviewResultV2) FilterByRuleType(ruleType string) []ViolationV2 {
	filtered := []ViolationV2{}
	for _, v := range r.Violations {
		if v.RuleType == ruleType {
			filtered = append(filtered, v)
		}
	}
	for _, v := range r.Warnings {
		if v.RuleType == ruleType {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// HasErrors returns true if there are any error-level violations
func (r *ReviewResultV2) HasErrors() bool {
	return r.ErrorCount > 0
}

// HasWarnings returns true if there are any warning-level violations
func (r *ReviewResultV2) HasWarnings() bool {
	return r.WarningCount > 0
}

// IsClean returns true if there are no violations
func (r *ReviewResultV2) IsClean() bool {
	return r.TotalViolations == 0
}
