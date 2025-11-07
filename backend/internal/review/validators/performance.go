//go:build ignore

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

	// Split SQL by semicolons to handle multiple statements
	statements := strings.Split(sqlContent, ";")

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		// Normalize SQL for pattern matching (case-insensitive)
		stmtUpper := strings.ToUpper(stmt)

		// Detect performance issues per statement
		violations = append(violations, v.detectMissingIndexes(stmtUpper, stmt)...)
		violations = append(violations, v.detectMissingPrimaryKeys(stmtUpper, stmt)...)
		// Check foreign keys against the entire SQL content (to catch indexes from previous statements)
		fkViolations := v.detectMissingForeignKeyIndexes(stmtUpper, stmt, sqlContent)
		violations = append(violations, fkViolations...)
	}

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
		// Check if table has any indexes defined (excluding PRIMARY KEY and FOREIGN KEY)
		// Look for INDEX, KEY (non-primary, non-foreign), UNIQUE keywords
		// Remove PRIMARY KEY and FOREIGN KEY from content to check for other indexes
		sqlWithoutPK := regexp.MustCompile(`(?i)\bPRIMARY\s+KEY\b`).ReplaceAllString(sqlContent, "")
		sqlWithoutFK := regexp.MustCompile(`(?i)\bFOREIGN\s+KEY\b`).ReplaceAllString(sqlWithoutPK, "")
		hasIndex := regexp.MustCompile(`(?i)\b(INDEX|KEY|UNIQUE)\b`).MatchString(sqlWithoutFK)

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
// fullSQLContent is used to check for indexes across multiple statements
func (v *PerformanceValidator) detectMissingForeignKeyIndexes(sqlUpper, sqlContent, fullSQLContent string) []review.Violation {
	var violations []review.Violation

	// Check for ALTER TABLE ADD FOREIGN KEY or CREATE TABLE with FOREIGN KEY
	if matched, _ := regexp.MatchString(`(?i)\bFOREIGN\s+KEY\b`, sqlUpper); matched {
		// Extract foreign key column names
		// Pattern: FOREIGN KEY (column_name) or CONSTRAINT ... FOREIGN KEY (column_name)
		fkPattern := regexp.MustCompile(`(?i)FOREIGN\s+KEY\s*\(([^)]+)\)`)
		matches := fkPattern.FindAllStringSubmatch(sqlContent, -1)

		if len(matches) > 0 {
			// Check if there's an INDEX or KEY on the foreign key column
			// Use fullSQLContent to check for indexes across statements
			// Exclude PRIMARY KEY and FOREIGN KEY from this check - we're looking for indexes on FK columns
			// Count INDEX/KEY occurrences excluding PRIMARY KEY and FOREIGN KEY
			// First, remove FOREIGN KEY and PRIMARY KEY patterns to avoid false positives
			sqlWithoutFK := regexp.MustCompile(`(?i)\bFOREIGN\s+KEY\b`).ReplaceAllString(fullSQLContent, "")
			indexKeyMatches := regexp.MustCompile(`(?i)\b(INDEX|KEY)\b`).FindAllString(sqlWithoutFK, -1)
			primaryKeyMatches := regexp.MustCompile(`(?i)\bPRIMARY\s+KEY\b`).FindAllString(fullSQLContent, -1)
			// If there are more INDEX/KEY occurrences than PRIMARY KEY occurrences,
			// then there are indexes other than PRIMARY KEY
			hasIndex := len(indexKeyMatches) > len(primaryKeyMatches)

			// If FOREIGN KEY exists but no INDEX/KEY found (excluding PRIMARY KEY), warn
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
