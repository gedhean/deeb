package validators

import (
	"context"
	"regexp"
	"strings"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/review"
)

// PerformanceValidator validates SQL for performance issues
type PerformanceValidator struct {
	review.BaseValidator
}

// NewPerformanceValidator creates a new performance validator
func NewPerformanceValidator() *PerformanceValidator {
	return &PerformanceValidator{
		BaseValidator: review.BaseValidator{
			Name:            "performance",
			DefaultSeverity: review.SeverityWarning,
		},
	}
}

// Validate validates SQL for performance issues
func (v *PerformanceValidator) Validate(ctx context.Context, sqlContent string, dbType models.DatabaseType) review.Result {
	// Skip empty SQL
	sqlContent = strings.TrimSpace(sqlContent)
	if sqlContent == "" {
		return review.NewResult(v.GetName(), true, []review.Violation{})
	}

	var violations []review.Violation

	// Normalize SQL for pattern matching (case-insensitive)
	sqlUpper := strings.ToUpper(sqlContent)

	// Detect performance issues
	violations = append(violations, v.detectMissingIndexes(sqlUpper, sqlContent)...)
	violations = append(violations, v.detectMissingPrimaryKeys(sqlUpper, sqlContent)...)
	violations = append(violations, v.detectMissingForeignKeyIndexes(sqlUpper, sqlContent)...)

	if len(violations) == 0 {
		return review.NewResult(v.GetName(), true, []review.Violation{})
	}

	return review.NewResult(v.GetName(), false, violations)
}

// detectMissingIndexes detects CREATE TABLE statements without indexes
func (v *PerformanceValidator) detectMissingIndexes(sqlUpper, sqlContent string) []review.Violation {
	var violations []review.Violation

	// Check for CREATE TABLE statements
	if matched, _ := regexp.MatchString(`(?i)\bCREATE\s+TABLE\b`, sqlUpper); matched {
		// Check if table has any indexes defined
		// Look for INDEX, KEY, PRIMARY KEY, UNIQUE keywords
		hasIndex := regexp.MustCompile(`(?i)\b(INDEX|KEY|PRIMARY\s+KEY|UNIQUE)\b`).MatchString(sqlContent)

		if !hasIndex {
			violations = append(violations, review.NewViolation(
				"missing-indexes",
				"CREATE TABLE statement without indexes detected. Consider adding indexes on frequently queried columns to improve query performance.",
				review.SeverityWarning,
			))
		}
	}

	return violations
}

// detectMissingPrimaryKeys detects CREATE TABLE statements without primary keys
func (v *PerformanceValidator) detectMissingPrimaryKeys(sqlUpper, sqlContent string) []review.Violation {
	var violations []review.Violation

	// Check for CREATE TABLE statements
	if matched, _ := regexp.MatchString(`(?i)\bCREATE\s+TABLE\b`, sqlUpper); matched {
		// Check if table has PRIMARY KEY defined
		hasPrimaryKey := regexp.MustCompile(`(?i)\bPRIMARY\s+KEY\b`).MatchString(sqlContent)

		if !hasPrimaryKey {
			violations = append(violations, review.NewViolation(
				"missing-primary-key",
				"CREATE TABLE statement without PRIMARY KEY detected. Tables without primary keys may have performance issues and make row identification difficult.",
				review.SeverityWarning,
			))
		}
	}

	return violations
}

// detectMissingForeignKeyIndexes detects foreign keys without indexes
func (v *PerformanceValidator) detectMissingForeignKeyIndexes(sqlUpper, sqlContent string) []review.Violation {
	var violations []review.Violation

	// Check for ALTER TABLE ADD FOREIGN KEY or CREATE TABLE with FOREIGN KEY
	if matched, _ := regexp.MatchString(`(?i)\bFOREIGN\s+KEY\b`, sqlUpper); matched {
		// Extract foreign key column names
		// Pattern: FOREIGN KEY (column_name) or CONSTRAINT ... FOREIGN KEY (column_name)
		fkPattern := regexp.MustCompile(`(?i)FOREIGN\s+KEY\s*\(([^)]+)\)`)
		matches := fkPattern.FindAllStringSubmatch(sqlContent, -1)

		if len(matches) > 0 {
			// Check if there's an INDEX or KEY on the foreign key column
			// This is a simplified check - in practice, we'd need to parse the full SQL
			// For now, we check if any INDEX/KEY exists in the statement
			hasIndex := regexp.MustCompile(`(?i)\b(INDEX|KEY)\b`).MatchString(sqlContent)

			// If FOREIGN KEY exists but no INDEX/KEY found, warn
			// Note: This is a heuristic - the index might be defined elsewhere
			// In a production system, we'd need full SQL parsing to accurately detect this
			if !hasIndex {
				violations = append(violations, review.NewViolation(
					"missing-foreign-key-index",
					"FOREIGN KEY constraint detected without explicit index. Foreign key columns should be indexed to improve JOIN performance and referential integrity checks.",
					review.SeverityWarning,
				))
			}
		}
	}

	return violations
}
