package validators

import (
	"context"
	"regexp"
	"strings"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/review"
)

// BackwardsCompatibilityValidator validates SQL for backwards compatibility issues
type BackwardsCompatibilityValidator struct {
	review.BaseValidator
}

// NewBackwardsCompatibilityValidator creates a new backwards compatibility validator
func NewBackwardsCompatibilityValidator() *BackwardsCompatibilityValidator {
	return &BackwardsCompatibilityValidator{
		BaseValidator: review.BaseValidator{
			Name:            "backwards-compatibility",
			DefaultSeverity: review.SeverityWarning,
		},
	}
}

// Validate validates SQL for backwards compatibility issues
func (v *BackwardsCompatibilityValidator) Validate(
	ctx context.Context,
	sqlContent string,
	dbType models.DatabaseType,
) review.Result {
	// Skip empty SQL
	sqlContent = strings.TrimSpace(sqlContent)
	if sqlContent == "" {
		return review.NewResult(v.GetName(), true, []review.Violation{})
	}

	var violations []review.Violation

	// Normalize SQL for pattern matching (case-insensitive)
	sqlUpper := strings.ToUpper(sqlContent)

	// Detect breaking changes
	violations = append(violations, v.detectBreakingChanges(sqlUpper, sqlContent)...)

	if len(violations) == 0 {
		return review.NewResult(v.GetName(), true, []review.Violation{})
	}

	return review.NewResult(v.GetName(), false, violations)
}

// detectBreakingChanges detects breaking schema changes
func (v *BackwardsCompatibilityValidator) detectBreakingChanges(sqlUpper, sqlContent string) []review.Violation {
	var violations []review.Violation

	// DROP COLUMN - Breaking change
	// Count occurrences to detect multiple DROP COLUMN operations
	dropColumnRegex := regexp.MustCompile(`(?i)\bDROP\s+COLUMN\b`)
	dropColumnMatches := dropColumnRegex.FindAllString(sqlUpper, -1)
	if len(dropColumnMatches) > 0 {
		// Report one violation per occurrence
		for i := 0; i < len(dropColumnMatches); i++ {
			violations = append(violations, review.NewViolation(
				"drop-column",
				"DROP COLUMN operation detected. This is a breaking change that will cause applications using this column to fail. Ensure all dependent applications are updated before proceeding.",
				review.SeverityWarning,
			))
		}
	}

	// RENAME TABLE - Breaking change
	// Match "RENAME TABLE" or "RENAME TO" (both are table renames)
	// Match "RENAME TABLE" but not "RENAME COLUMN"
	if matched, _ := regexp.MatchString(`(?i)\bRENAME\s+TABLE\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"rename-table",
			"RENAME TABLE operation detected. This is a breaking change that will cause applications referencing the old table name to fail. Ensure all dependent applications are updated before proceeding.",
			review.SeverityWarning,
		))
	}
	// RENAME TO (PostgreSQL/MySQL syntax for table rename)
	if matched, _ := regexp.MatchString(`(?i)\bALTER\s+TABLE\b.*\bRENAME\s+TO\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"rename-table",
			"RENAME TABLE operation detected. This is a breaking change that will cause applications referencing the old table name to fail. Ensure all dependent applications are updated before proceeding.",
			review.SeverityWarning,
		))
	}

	// RENAME COLUMN - Breaking change
	if matched, _ := regexp.MatchString(`(?i)\bRENAME\s+COLUMN\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"rename-column",
			"RENAME COLUMN operation detected. This is a breaking change that will cause applications using the old column name to fail. Ensure all dependent applications are updated before proceeding.",
			review.SeverityWarning,
		))
	}

	// ALTER COLUMN with data type change - Breaking change
	// Pattern: ALTER TABLE ... ALTER COLUMN ... TYPE ... or ALTER COLUMN ... MODIFY ...
	if matched, _ := regexp.MatchString(`(?i)\bALTER\s+COLUMN\b.*\b(TYPE|MODIFY)\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"alter-column-type",
			"ALTER COLUMN with data type change detected. This is a breaking change that may cause data loss or application failures. Ensure data migration is planned and all dependent applications are updated.",
			review.SeverityWarning,
		))
	}

	// MODIFY COLUMN (MySQL syntax) - Breaking change
	// Matches both "MODIFY COLUMN" and "MODIFY" (without COLUMN keyword)
	if matched, _ := regexp.MatchString(`(?i)\bALTER\s+TABLE\b.*\bMODIFY\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"modify-column",
			"MODIFY COLUMN operation detected. This may change data types and is a breaking change. Ensure data migration is planned and all dependent applications are updated.",
			review.SeverityWarning,
		))
	}

	// CHANGE COLUMN (MySQL syntax) - Breaking change (can rename or change type)
	if matched, _ := regexp.MatchString(`(?i)\bCHANGE\s+COLUMN\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"change-column",
			"CHANGE COLUMN operation detected. This may rename or change data types and is a breaking change. Ensure all dependent applications are updated before proceeding.",
			review.SeverityWarning,
		))
	}

	// DROP CONSTRAINT (including NOT NULL) - Breaking change
	if matched, _ := regexp.MatchString(`(?i)\bDROP\s+CONSTRAINT\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"drop-constraint",
			"DROP CONSTRAINT operation detected. Removing constraints (especially NOT NULL) may cause application failures. Ensure all dependent applications are updated.",
			review.SeverityWarning,
		))
	}

	// ALTER COLUMN SET NOT NULL - Breaking change (may fail if nulls exist)
	// Also matches MODIFY COLUMN ... NOT NULL (MySQL syntax)
	if matched, _ := regexp.MatchString(`(?i)\bALTER\s+COLUMN\b.*\bSET\s+NOT\s+NULL\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"set-not-null",
			"SET NOT NULL constraint detected. This may fail if existing rows contain NULL values. Ensure data is cleaned before applying this change.",
			review.SeverityWarning,
		))
	}
	// MODIFY COLUMN ... NOT NULL (MySQL syntax) - Breaking change
	if matched, _ := regexp.MatchString(`(?i)\bMODIFY\s+COLUMN\b.*\bNOT\s+NULL\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"set-not-null",
			"SET NOT NULL constraint detected. This may fail if existing rows contain NULL values. Ensure data is cleaned before applying this change.",
			review.SeverityWarning,
		))
	}

	// DROP DEFAULT - Breaking change (may cause application failures)
	if matched, _ := regexp.MatchString(`(?i)\bDROP\s+DEFAULT\b`, sqlUpper); matched {
		violations = append(violations, review.NewViolation(
			"drop-default",
			"DROP DEFAULT operation detected. Applications relying on default values may fail. Ensure all dependent applications are updated.",
			review.SeverityWarning,
		))
	}

	return violations
}
