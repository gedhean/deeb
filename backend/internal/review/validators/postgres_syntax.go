//go:build ignore

package validators

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/review"
	pg_query "github.com/pganalyze/pg_query_go/v6"
)

// PostgreSQLSyntaxValidator validates PostgreSQL SQL syntax
type PostgreSQLSyntaxValidator struct {
	review.BaseValidator
}

// postgresLineRegex matches PostgreSQL error messages with line numbers
// Format: "LINE X: ..."
var postgresLineRegex = regexp.MustCompile(`(?i)LINE\s+(\d+)`)

// postgresCharRegex matches PostgreSQL error messages with character positions
// Format: "at character X"
var postgresCharRegex = regexp.MustCompile(`(?i)at character\s+(\d+)`)

// NewPostgreSQLSyntaxValidator creates a new PostgreSQL syntax validator
func NewPostgreSQLSyntaxValidator() *PostgreSQLSyntaxValidator {
	return &PostgreSQLSyntaxValidator{
		BaseValidator: review.BaseValidator{
			Name:            "postgresql-syntax",
			DefaultSeverity: review.SeverityError,
		},
	}
}

// Validate validates PostgreSQL SQL syntax
func (v *PostgreSQLSyntaxValidator) Validate(ctx context.Context, sqlContent string, dbType models.DatabaseType) review.Result {
	// Only validate PostgreSQL database types
	if dbType != models.DatabaseTypePostgreSQL {
		return review.NewResult(v.GetName(), true, []review.Violation{})
	}

	// Skip empty SQL
	sqlContent = strings.TrimSpace(sqlContent)
	if sqlContent == "" {
		return review.NewResult(v.GetName(), true, []review.Violation{})
	}

	// Parse SQL using pg_query_go
	_, err := pg_query.Parse(sqlContent)
	if err != nil {
		// Extract line number from error message
		lineNum := v.extractLineNumber(err.Error())

		// Create violation with error message
		violations := []review.Violation{
			review.NewViolationWithLocation(
				"syntax-error",
				err.Error(),
				review.SeverityError,
				lineNum,
				0, // Column position not available from parser
			),
		}

		return review.NewResult(v.GetName(), false, violations)
	}

	// Syntax is valid
	return review.NewResult(v.GetName(), true, []review.Violation{})
}

// extractLineNumber extracts line number from PostgreSQL error message
func (v *PostgreSQLSyntaxValidator) extractLineNumber(errorMsg string) int {
	// Prioritize LINE over character position
	lineMatches := postgresLineRegex.FindStringSubmatch(errorMsg)
	if len(lineMatches) >= 2 {
		if lineNum, err := strconv.Atoi(lineMatches[1]); err == nil {
			return lineNum
		}
	}

	// If no LINE found, check for character position (fallback for backwards compatibility)
	charMatches := postgresCharRegex.FindStringSubmatch(errorMsg)
	if len(charMatches) >= 2 {
		if charNum, err := strconv.Atoi(charMatches[1]); err == nil {
			return charNum
		}
	}

	return 0 // Line number not found
}
