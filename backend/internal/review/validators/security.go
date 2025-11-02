package validators

import (
	"context"
	"regexp"
	"strings"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/review"
)

// SecurityValidator validates SQL for security risks and destructive operations
type SecurityValidator struct {
	review.BaseValidator
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator() *SecurityValidator {
	return &SecurityValidator{
		BaseValidator: review.BaseValidator{
			Name:            "security",
			DefaultSeverity: review.SeverityError,
		},
	}
}

// Validate validates SQL for security risks and destructive operations
func (v *SecurityValidator) Validate(ctx context.Context, sqlContent string, dbType models.DatabaseType) review.Result {
	// Skip empty SQL
	sqlContent = strings.TrimSpace(sqlContent)
	if sqlContent == "" {
		return review.NewResult(v.GetName(), true, []review.Violation{})
	}

	var violations []review.Violation

	// Normalize SQL for pattern matching (case-insensitive)
	sqlUpper := strings.ToUpper(sqlContent)

	// Detect destructive operations
	violations = append(violations, v.detectDestructiveOperations(sqlUpper, sqlContent)...)

	if len(violations) == 0 {
		return review.NewResult(v.GetName(), true, []review.Violation{})
	}

	return review.NewResult(v.GetName(), false, violations)
}

// detectDestructiveOperations detects destructive SQL operations
func (v *SecurityValidator) detectDestructiveOperations(sqlUpper, sqlContent string) []review.Violation {
	var violations []review.Violation

	// DROP DATABASE - Critical, always Error
	if matched, _ := regexp.MatchString(`(?i)\bDROP\s+DATABASE\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"drop-database",
			"DROP DATABASE operation detected. This will delete the entire database and all its data. Ensure proper backups exist before proceeding.",
			review.SeverityError,
		))
	}

	// DROP TABLE - High risk, Error severity
	if matched, _ := regexp.MatchString(`(?i)\bDROP\s+TABLE\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"drop-table",
			"DROP TABLE operation detected. This will permanently delete the table and all its data. Ensure proper backups exist before proceeding.",
			review.SeverityError,
		))
	}

	// TRUNCATE TABLE - High risk, Error severity
	if matched, _ := regexp.MatchString(`(?i)\bTRUNCATE\s+(TABLE\s+)?`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"truncate-table",
			"TRUNCATE TABLE operation detected. This will permanently delete all data from the table. Ensure proper backups exist before proceeding.",
			review.SeverityError,
		))
	}

	// DROP COLUMN - Medium risk, Warning severity
	if matched, _ := regexp.MatchString(`(?i)\bDROP\s+COLUMN\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"drop-column",
			"DROP COLUMN operation detected. This will permanently delete the column and all its data. Ensure proper backups exist and verify no application code depends on this column.",
			review.SeverityWarning,
		))
	}

	// DROP INDEX - Lower risk, Warning severity
	if matched, _ := regexp.MatchString(`(?i)\bDROP\s+INDEX\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"drop-index",
			"DROP INDEX operation detected. This may impact query performance. Verify that the index is no longer needed.",
			review.SeverityWarning,
		))
	}

	// DROP CONSTRAINT - Lower risk, Warning severity
	if matched, _ := regexp.MatchString(`(?i)\bDROP\s+CONSTRAINT\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"drop-constraint",
			"DROP CONSTRAINT operation detected. This may impact data integrity. Verify that the constraint is no longer needed.",
			review.SeverityWarning,
		))
	}

	// DROP VIEW - Lower risk, Warning severity
	if matched, _ := regexp.MatchString(`(?i)\bDROP\s+VIEW\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"drop-view",
			"DROP VIEW operation detected. This will delete the view definition. Ensure no applications depend on this view.",
			review.SeverityWarning,
		))
	}

	// DROP FUNCTION/PROCEDURE - Medium risk, Warning severity
	if matched, _ := regexp.MatchString(`(?i)\bDROP\s+(FUNCTION|PROCEDURE)\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"drop-function",
			"DROP FUNCTION/PROCEDURE operation detected. This will delete the stored procedure. Ensure no applications depend on this function/procedure.",
			review.SeverityWarning,
		))
	}

	return violations
}
