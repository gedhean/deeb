package validators

import (
	"context"
	"testing"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/review"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMySQLSyntaxValidator(t *testing.T) {
	validator := NewMySQLSyntaxValidator()
	ctx := context.Background()

	t.Run("GetName returns validator name", func(t *testing.T) {
		assert.Equal(t, "mysql-syntax", validator.GetName())
	})

	t.Run("GetSeverity returns Error severity", func(t *testing.T) {
		assert.Equal(t, review.SeverityError, validator.GetSeverity())
	})

	t.Run("skips non-MySQL database types", func(t *testing.T) {
		result := validator.Validate(ctx, "SELECT * FROM users", models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
		assert.Equal(t, "mysql-syntax", result.ValidatorName)
	})

	t.Run("skips empty SQL", func(t *testing.T) {
		result := validator.Validate(ctx, "", models.DatabaseTypeMySQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("skips whitespace-only SQL", func(t *testing.T) {
		result := validator.Validate(ctx, "   \n\t  ", models.DatabaseTypeMySQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("validator structure is correct", func(t *testing.T) {
		// Test that the validator can be created and has correct structure
		assert.NotNil(t, validator)
		assert.Equal(t, "mysql-syntax", validator.GetName())
		assert.Equal(t, review.SeverityError, validator.GetSeverity())
		assert.Implements(t, (*review.Validator)(nil), validator)
	})

	t.Run("validates valid MySQL SQL", func(t *testing.T) {
		validSQL := []string{
			"SELECT * FROM users",
			"SELECT id, name FROM users WHERE id = 1",
			"CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100))",
			"INSERT INTO users (name) VALUES ('test')",
			"UPDATE users SET name = 'test' WHERE id = 1",
			"DELETE FROM users WHERE id = 1",
			"ALTER TABLE users ADD COLUMN email VARCHAR(255)",
		}

		for _, sql := range validSQL {
			t.Run("valid SQL: "+sql[:min(20, len(sql))], func(t *testing.T) {
				result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

				assert.NotNil(t, result)
				assert.Equal(t, "mysql-syntax", result.ValidatorName)
				// sqlparser should successfully parse valid SQL
				assert.True(t, result.Valid, "SQL should be valid: %s", sql)
				assert.Empty(t, result.Violations)
			})
		}
	})

	t.Run("detects syntax errors", func(t *testing.T) {
		invalidSQL := []string{
			"SELECT * FROM",            // Missing table name
			"SELECT * FORM users",      // Typo: FORM instead of FROM
			"CREATE TABLE users (",     // Incomplete statement
			"INSERT INTO users VALUES", // Missing values
		}

		for _, sql := range invalidSQL {
			t.Run("invalid SQL: "+sql[:min(20, len(sql))], func(t *testing.T) {
				result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

				assert.NotNil(t, result)
				assert.Equal(t, "mysql-syntax", result.ValidatorName)
				assert.False(t, result.Valid, "SQL should be invalid: %s", sql)
				require.NotEmpty(t, result.Violations)
				assert.Equal(t, review.SeverityError, result.Violations[0].Severity)
			})
		}
	})
}

func TestMySQLExtractLineNumber(t *testing.T) {
	validator := NewMySQLSyntaxValidator()

	tests := []struct {
		name     string
		errorMsg string
		expected int
	}{
		{
			name:     "extracts line number from 'line X' format",
			errorMsg: "syntax error at line 3 near 'FROM'",
			expected: 3,
		},
		{
			name:     "extracts line number from 'at line X' format",
			errorMsg: "syntax error at line 5",
			expected: 5,
		},
		{
			name:     "extracts first number found",
			errorMsg: "syntax error at line 10 near line 20",
			expected: 10,
		},
		{
			name:     "returns 0 when no line number found",
			errorMsg: "syntax error near 'FROM'",
			expected: 0,
		},
		{
			name:     "handles case insensitive matching",
			errorMsg: "syntax error at LINE 15",
			expected: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lineNum := validator.extractLineNumber(tt.errorMsg)
			assert.Equal(t, tt.expected, lineNum)
		})
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
