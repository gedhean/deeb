package review

import (
	"context"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/sql-reviewer/pkg/types"
)

// Severity represents the severity level of a validation violation
// Deprecated: Use types.Advice_Status instead
type Severity string

const (
	// SeverityError indicates a critical issue that must be fixed
	SeverityError Severity = "Error"
	// SeverityWarning indicates a potential issue that should be reviewed
	SeverityWarning Severity = "Warning"
	// SeverityInfo indicates a suggestion or best practice recommendation
	SeverityInfo Severity = "Info"
)

// Violation is deprecated - use types.Advice instead
// Deprecated: Use types.Advice from sql-reviewer package
type Violation = types.Advice

// Result represents the result of running a validator on SQL content
type Result struct {
	// ValidatorName is the name of the validator that produced this result
	ValidatorName string `json:"validator_name"`
	// Valid indicates whether the validation passed (true) or failed (false)
	Valid bool `json:"valid"`
	// Violations is a list of advice items found by this validator
	// Using Violations name for backward compatibility, but contains types.Advice
	Violations []*types.Advice `json:"violations"`
	// Error is set if the validator encountered an error during validation
	Error string `json:"error,omitempty"`
}

// ReviewResult represents the aggregated result of running all validators
type ReviewResult struct {
	// Valid indicates whether all validators passed (true) or any failed (false)
	Valid bool `json:"valid"`
	// Results is a list of results from each validator
	Results []Result `json:"results"`
	// TotalViolations is the total number of violations found across all validators
	TotalViolations int `json:"total_violations"`
	// Errors is the number of violations with Error severity
	Errors int `json:"errors"`
	// Warnings is the number of violations with Warning severity
	Warnings int `json:"warnings"`
	// Infos is the number of violations with Info severity
	Infos int `json:"infos"`
}

// Validator is the interface that all SQL validators must implement
type Validator interface {
	// Validate validates SQL content and returns validation results
	// ctx: context for cancellation and timeout
	// sqlContent: the SQL content to validate
	// dbType: the database type (MySQL or PostgreSQL)
	// Returns a Result containing validation findings
	Validate(ctx context.Context, sqlContent string, dbType models.DatabaseType) Result

	// GetName returns the unique name of this validator
	GetName() string

	// GetSeverity returns the default severity level for violations from this validator
	// This can be used to configure the validator's behavior
	GetSeverity() Severity
}

// ValidatorRegistry is the interface for registering and managing validators
type ValidatorRegistry interface {
	// Register adds a validator to the registry
	Register(validator Validator) error

	// Get retrieves a validator by name
	Get(name string) (Validator, error)

	// GetAll returns all registered validators
	GetAll() []Validator

	// Unregister removes a validator from the registry
	Unregister(name string) error
}

// Engine is the interface for the SQL Review engine that orchestrates validators
type Engine interface {
	// Review validates SQL content using all registered validators
	// ctx: context for cancellation and timeout
	// sqlContent: the SQL content to validate
	// dbType: the database type (MySQL or PostgreSQL)
	// Returns a ReviewResult containing aggregated results from all validators
	Review(ctx context.Context, sqlContent string, dbType models.DatabaseType) ReviewResult

	// RegisterValidator adds a validator to the engine
	RegisterValidator(validator Validator) error

	// GetRegistry returns the validator registry
	GetRegistry() ValidatorRegistry
}
