package validators

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/review"
	pg_query "github.com/pganalyze/pg_query_go/v5"
)

// PostgreSQLSyntaxValidator validates PostgreSQL SQL syntax
type PostgreSQLSyntaxValidator struct {
	review.BaseValidator
}

// postgresErrorLineRegex matches PostgreSQL error messages with line numbers
// Format: "syntax error at or near \"...\" at character X" or "LINE X: ..."
var postgresErrorLineRegex = regexp.MustCompile(`(?i)(?:line|at character)\s+(\d+)`)

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
	matches := postgresErrorLineRegex.FindStringSubmatch(errorMsg)
	if len(matches) >= 2 {
		if lineNum, err := strconv.Atoi(matches[1]); err == nil {
			return lineNum
		}
	}
	return 0 // Line number not found
}
