package validators

import (
	"context"
	"regexp"
	"strings"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/review"
)

// BestPracticesValidator validates SQL for best practices and naming conventions
type BestPracticesValidator struct {
	review.BaseValidator
}

// NewBestPracticesValidator creates a new best practices validator
func NewBestPracticesValidator() *BestPracticesValidator {
	return &BestPracticesValidator{
		BaseValidator: review.BaseValidator{
			Name:            "best-practices",
			DefaultSeverity: review.SeverityInfo,
		},
	}
}

// Validate validates SQL for best practices and naming conventions
func (v *BestPracticesValidator) Validate(ctx context.Context, sqlContent string, dbType models.DatabaseType) review.Result {
	// Skip empty SQL
	sqlContent = strings.TrimSpace(sqlContent)
	if sqlContent == "" {
		return review.NewResult(v.GetName(), true, []review.Violation{})
	}

	var violations []review.Violation

	// Normalize SQL for pattern matching (case-insensitive)
	sqlUpper := strings.ToUpper(sqlContent)

	// Detect naming convention violations
	violations = append(violations, v.detectNamingConventionViolations(sqlContent, sqlUpper)...)
	violations = append(violations, v.detectDataTypeViolations(sqlContent, sqlUpper)...)

	if len(violations) == 0 {
		return review.NewResult(v.GetName(), true, []review.Violation{})
	}

	return review.NewResult(v.GetName(), false, violations)
}

// detectNamingConventionViolations detects violations of naming conventions
// Rules: snake_case for everything, tables plural, columns singular, indexes idx_<table>_<column>
func (v *BestPracticesValidator) detectNamingConventionViolations(sqlContent, sqlUpper string) []review.Violation {
	var violations []review.Violation

	// snake_case pattern: lowercase letters, numbers, and underscores only
	snakeCasePattern := regexp.MustCompile(`^[a-z][a-z0-9_]*(_[a-z0-9]+)*$`)

	// Check CREATE TABLE statements for table names
	if matched, _ := regexp.MatchString(`(?i)\bCREATE\s+TABLE\b`, sqlUpper); matched {
		// Extract table name: CREATE TABLE table_name
		tableNamePattern := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_]+)`)
		matches := tableNamePattern.FindStringSubmatch(sqlContent)
		if len(matches) >= 2 {
			tableName := matches[1]

			// Check snake_case
			if !snakeCasePattern.MatchString(strings.ToLower(tableName)) {
				violations = append(violations, review.NewViolation(
					"table-naming-convention",
					"Table name '"+tableName+"' should use snake_case naming convention (e.g., 'user_accounts').",
					review.SeverityInfo,
				))
			} else {
				// Check plural form (simple heuristic: should end with 's' or common plural patterns)
				lowerName := strings.ToLower(tableName)
				if !strings.HasSuffix(lowerName, "s") &&
					!strings.HasSuffix(lowerName, "ies") &&
					!strings.HasSuffix(lowerName, "es") &&
					lowerName != "users" && lowerName != "accounts" && lowerName != "products" {
					// Common exceptions: data, metadata, schema, etc.
					if !v.isCommonSingularException(lowerName) {
						violations = append(violations, review.NewViolation(
							"table-plural-form",
							"Table name '"+tableName+"' should be plural (e.g., 'users', 'orders', 'products').",
							review.SeverityInfo,
						))
					}
				}
			}
		}
	}

	// Check column names in CREATE TABLE or ALTER TABLE ADD COLUMN
	if matched, _ := regexp.MatchString(`(?i)\b(CREATE\s+TABLE|ALTER\s+TABLE.*ADD\s+COLUMN)\b`, sqlUpper); matched {
		// Extract column definitions: column_name TYPE
		columnPattern := regexp.MustCompile(
			`(?i)\b([a-zA-Z0-9_]+)\s+(INT|VARCHAR|TEXT|CHAR|BIGINT|SMALLINT|DECIMAL|NUMERIC|DATE|TIMESTAMP|BOOLEAN|UUID|SERIAL)`,
		)
		matches := columnPattern.FindAllStringSubmatch(sqlContent, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				columnName := match[1]

				// Skip keywords that might match
				if strings.ToUpper(columnName) == "PRIMARY" || strings.ToUpper(columnName) == "FOREIGN" ||
					strings.ToUpper(columnName) == "KEY" || strings.ToUpper(columnName) == "CONSTRAINT" {
					continue
				}

				// Check snake_case
				if !snakeCasePattern.MatchString(strings.ToLower(columnName)) {
					violations = append(violations, review.NewViolation(
						"column-naming-convention",
						"Column name '"+columnName+"' should use snake_case naming convention (e.g., 'user_id', 'created_at').",
						review.SeverityInfo,
					))
				} else {
					// Check singular form (should not end with 's' typically)
					lowerName := strings.ToLower(columnName)
					if strings.HasSuffix(lowerName, "s") && !v.isCommonPluralColumnException(lowerName) {
						violations = append(violations, review.NewViolation(
							"column-singular-form",
							"Column name '"+columnName+"' should be singular (e.g., 'user_id' not 'users_id').",
							review.SeverityInfo,
						))
					}
				}
			}
		}
	}

	// Check index names: should follow idx_<table>_<column> pattern
	if matched, _ := regexp.MatchString(`(?i)\b(CREATE\s+INDEX|CREATE\s+UNIQUE\s+INDEX|ADD\s+INDEX|ADD\s+KEY)\b`, sqlUpper); matched {
		// Extract index name: CREATE INDEX index_name or CREATE INDEX idx_name ON table
		indexNamePattern := regexp.MustCompile(
			`(?i)(?:CREATE\s+(?:UNIQUE\s+)?INDEX|ADD\s+(?:UNIQUE\s+)?INDEX|ADD\s+KEY)\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_]+)`,
		)
		matches := indexNamePattern.FindStringSubmatch(sqlContent)
		if len(matches) >= 2 {
			indexName := matches[1]

			// Check snake_case
			if !snakeCasePattern.MatchString(strings.ToLower(indexName)) {
				violations = append(violations, review.NewViolation(
					"index-naming-convention",
					"Index name '"+indexName+"' should use snake_case naming convention.",
					review.SeverityInfo,
				))
			} else {
				// Check idx_ prefix pattern
				lowerName := strings.ToLower(indexName)
				if !strings.HasPrefix(lowerName, "idx_") {
					violations = append(violations, review.NewViolation(
						"index-naming-pattern",
						"Index name '"+indexName+"' should follow pattern 'idx_<table>_<column>' (e.g., 'idx_users_email').",
						review.SeverityInfo,
					))
				}
			}
		}
	}

	return violations
}

// detectDataTypeViolations detects data type best practice violations
func (v *BestPracticesValidator) detectDataTypeViolations(sqlContent, sqlUpper string) []review.Violation {
	var violations []review.Violation

	// Check for TEXT type usage (should use VARCHAR for small strings)
	if matched, _ := regexp.MatchString(`(?i)\bTEXT\b`, sqlUpper); matched {
		// Check if VARCHAR would be more appropriate
		// Pattern: column_name TEXT (without size limit)
		textPattern := regexp.MustCompile(`(?i)\b([a-zA-Z0-9_]+)\s+TEXT\b`)
		matches := textPattern.FindAllStringSubmatch(sqlContent, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				columnName := match[1]
				// Skip if it's a comment or part of another word
				if strings.ToUpper(columnName) == "TEXT" {
					continue
				}
				violations = append(violations, review.NewViolation(
					"text-type-usage",
					"Column '"+columnName+"' uses TEXT type. Consider using VARCHAR(n) with appropriate size limit if the content length is known and small (< 255 characters).",
					review.SeverityInfo,
				))
			}
		}
	}

	return violations
}

// isCommonSingularException checks if a table name is a common exception (data, metadata, etc.)
func (v *BestPracticesValidator) isCommonSingularException(name string) bool {
	exceptions := []string{"data", "metadata", "schema", "config", "settings", "info"}
	for _, ex := range exceptions {
		if name == ex {
			return true
		}
	}
	return false
}

// isCommonPluralColumnException checks if a column name is a common plural exception
func (v *BestPracticesValidator) isCommonPluralColumnException(name string) bool {
	exceptions := []string{"settings", "credentials", "permissions", "preferences"}
	for _, ex := range exceptions {
		if name == ex {
			return true
		}
	}
	return false
}
