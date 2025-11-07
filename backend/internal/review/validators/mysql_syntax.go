//go:build ignore

package validators

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/review"
	"github.com/xwb1989/sqlparser"
)

// MySQLSyntaxValidator validates MySQL SQL syntax
type MySQLSyntaxValidator struct {
	review.BaseValidator
}

// mysqlErrorLineRegex matches MySQL error messages with line numbers
// Format: "syntax error at line X" or "near '...' at line X"
var mysqlErrorLineRegex = regexp.MustCompile(`(?i)(?:line|at line)\s+(\d+)`)

// NewMySQLSyntaxValidator creates a new MySQL syntax validator
func NewMySQLSyntaxValidator() *MySQLSyntaxValidator {
	return &MySQLSyntaxValidator{
		BaseValidator: review.BaseValidator{
			Name:            "mysql-syntax",
			DefaultSeverity: review.SeverityError,
		},
	}
}

// Validate validates MySQL SQL syntax
func (v *MySQLSyntaxValidator) Validate(ctx context.Context, sqlContent string, dbType models.DatabaseType) review.Result {
	// Only validate MySQL database types
	if dbType != models.DatabaseTypeMySQL {
		return review.NewResult(v.GetName(), true, []review.Violation{})
	}

	// Skip empty SQL
	sqlContent = strings.TrimSpace(sqlContent)
	if sqlContent == "" {
		return review.NewResult(v.GetName(), true, []review.Violation{})
	}

	// Parse SQL using sqlparser
	_, err := sqlparser.Parse(sqlContent)
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

// extractLineNumber extracts line number from MySQL error message
func (v *MySQLSyntaxValidator) extractLineNumber(errorMsg string) int {
	matches := mysqlErrorLineRegex.FindStringSubmatch(errorMsg)
	if len(matches) >= 2 {
		if lineNum, err := strconv.Atoi(matches[1]); err == nil {
			return lineNum
		}
	}
	return 0 // Line number not found
}
