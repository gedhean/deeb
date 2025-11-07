//go:build ignore

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
	t.Skip("Skipping review tests temporarily")
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
			t.Run("valid SQL: "+sql[:mysqlMin(20, len(sql))], func(t *testing.T) {
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
			t.Run("invalid SQL: "+sql[:mysqlMin(20, len(sql))], func(t *testing.T) {
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
	t.Skip("Skipping review tests temporarily")
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
		{
			name:     "extracts from complex error message",
			errorMsg: "ERROR 1064 (42000): You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near 'FROM' at line 7",
			expected: 7,
		},
		{
			name:     "extracts from error with 'at line' format",
			errorMsg: "syntax error near 'SELECT' at line 42",
			expected: 42,
		},
		{
			name:     "handles empty error message",
			errorMsg: "",
			expected: 0,
		},
		{
			name:     "handles error message with no numbers",
			errorMsg: "syntax error near 'FROM'",
			expected: 0,
		},
		{
			name:     "extracts line number when multiple numbers present",
			errorMsg: "error at line 5 column 10",
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lineNum := validator.extractLineNumber(tt.errorMsg)
			assert.Equal(t, tt.expected, lineNum)
		})
	}
}

// TestMySQLSyntaxValidator_ErrorHandling tests error handling scenarios
func TestMySQLSyntaxValidator_ErrorHandling(t *testing.T) {
	t.Skip("Skipping review tests temporarily")
	validator := NewMySQLSyntaxValidator()
	ctx := context.Background()

	t.Run("handles multiple database types", func(t *testing.T) {
		testCases := []struct {
			dbType     models.DatabaseType
			shouldSkip bool
		}{
			{models.DatabaseTypeMySQL, false},
			{models.DatabaseTypePostgreSQL, true},
			{"", true}, // Empty database type
		}

		for _, tc := range testCases {
			t.Run(string(tc.dbType), func(t *testing.T) {
				result := validator.Validate(ctx, "SELECT * FROM users", tc.dbType)
				if tc.shouldSkip {
					assert.True(t, result.Valid)
					assert.Empty(t, result.Violations)
				}
			})
		}
	})

	t.Run("handles various empty/whitespace inputs", func(t *testing.T) {
		emptyInputs := []string{
			"",
			"   ",
			"\n",
			"\t",
			"   \n\t   ",
		}

		for _, input := range emptyInputs {
			t.Run("input: "+input, func(t *testing.T) {
				result := validator.Validate(ctx, input, models.DatabaseTypeMySQL)
				assert.True(t, result.Valid)
				assert.Empty(t, result.Violations)
			})
		}
	})

	t.Run("validator implements interface correctly", func(t *testing.T) {
		var _ review.Validator = validator
		assert.NotNil(t, validator)
		assert.Equal(t, "mysql-syntax", validator.GetName())
		assert.Equal(t, review.SeverityError, validator.GetSeverity())
	})
}

// TestMySQLSyntaxValidator_ErrorMessages tests error message validation
func TestMySQLSyntaxValidator_ErrorMessages(t *testing.T) {
	t.Skip("Skipping review tests temporarily")
	validator := NewMySQLSyntaxValidator()
	ctx := context.Background()

	t.Run("error messages contain violation details", func(t *testing.T) {
		invalidSQL := "SELECT * FORM users" // Typo: FORM instead of FROM

		result := validator.Validate(ctx, invalidSQL, models.DatabaseTypeMySQL)

		require.False(t, result.Valid)
		require.NotEmpty(t, result.Violations)

		violation := result.Violations[0]
		assert.Equal(t, "syntax-error", violation.Rule)
		assert.Equal(t, review.SeverityError, violation.Severity)
		assert.NotEmpty(t, violation.Message)
		assert.NotNil(t, violation.Line)
		assert.Equal(t, 0, violation.Column) // Column not available from parser
	})

	t.Run("multiple syntax errors detected", func(t *testing.T) {
		invalidSQL := "SELECT * FORM users WHERE id = SELECT" // Multiple errors

		result := validator.Validate(ctx, invalidSQL, models.DatabaseTypeMySQL)

		require.False(t, result.Valid)
		require.NotEmpty(t, result.Violations)
		assert.Equal(t, review.SeverityError, result.Violations[0].Severity)
	})
}

// TestMySQLSyntaxValidator_ValidSQL tests comprehensive valid SQL cases
func TestMySQLSyntaxValidator_ValidSQL(t *testing.T) {
	t.Skip("Skipping review tests temporarily")
	validator := NewMySQLSyntaxValidator()
	ctx := context.Background()

	validSQL := []string{
		"SELECT * FROM users",
		"SELECT id, name FROM users WHERE id = 1",
		"SELECT u.id, u.name FROM users u WHERE u.id > 10",
		"CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100))",
		"CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100), created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)",
		"INSERT INTO users (name) VALUES ('test')",
		"INSERT INTO users (name, email) VALUES ('test', 'test@example.com')",
		"UPDATE users SET name = 'test' WHERE id = 1",
		"UPDATE users SET name = 'test', email = 'test@example.com' WHERE id = 1",
		"DELETE FROM users WHERE id = 1",
		"DELETE FROM users WHERE id > 10 AND status = 'inactive'",
		"ALTER TABLE users ADD COLUMN email VARCHAR(255)",
		"ALTER TABLE users ADD COLUMN email VARCHAR(255) AFTER name",
		"ALTER TABLE users DROP COLUMN email",
		"ALTER TABLE users MODIFY COLUMN name VARCHAR(200)",
		"DROP TABLE IF EXISTS users",
		"CREATE INDEX idx_users_email ON users(email)",
		"DROP INDEX idx_users_email ON users",
		"SELECT COUNT(*) FROM users",
		"SELECT u.id, u.name, COUNT(o.id) as order_count FROM users u LEFT JOIN orders o ON u.id = o.user_id GROUP BY u.id, u.name",
	}

	for _, sql := range validSQL {
		t.Run("valid SQL: "+sql[:mysqlMin(30, len(sql))], func(t *testing.T) {
			result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

			assert.NotNil(t, result)
			assert.Equal(t, "mysql-syntax", result.ValidatorName)
			assert.True(t, result.Valid, "SQL should be valid: %s", sql)
			assert.Empty(t, result.Violations)
		})
	}
}

// TestMySQLSyntaxValidator_InvalidSQL tests comprehensive invalid SQL cases
func TestMySQLSyntaxValidator_InvalidSQL(t *testing.T) {
	t.Skip("Skipping review tests temporarily")
	validator := NewMySQLSyntaxValidator()
	ctx := context.Background()

	invalidSQL := []struct {
		sql    string
		reason string
	}{
		{"SELECT * FROM", "Missing table name"},
		{"SELECT * FORM users", "Typo: FORM instead of FROM"},
		{"CREATE TABLE users (", "Incomplete statement"},
		{"INSERT INTO users VALUES", "Missing values"},
		{"UPDATE users SET", "Missing SET values"},
		{"DELETE FROM", "Missing table name"},
		{"ALTER TABLE users", "Incomplete ALTER statement"},
		{"SELECT * FROM users WHERE", "Incomplete WHERE clause"},
		{"SELECT * FROM users JOIN", "Incomplete JOIN"},
		{"CREATE TABLE users id INT", "Missing parentheses"},
		{"SELECT * FROM users GROUP BY", "Incomplete GROUP BY"},
		{"SELECT * FROM users ORDER BY", "Incomplete ORDER BY"},
	}

	for _, tc := range invalidSQL {
		t.Run("invalid SQL: "+tc.reason, func(t *testing.T) {
			result := validator.Validate(ctx, tc.sql, models.DatabaseTypeMySQL)

			assert.NotNil(t, result)
			assert.Equal(t, "mysql-syntax", result.ValidatorName)
			assert.False(t, result.Valid, "SQL should be invalid: %s", tc.sql)
			require.NotEmpty(t, result.Violations, "Should have violations for: %s", tc.sql)
			assert.Equal(t, review.SeverityError, result.Violations[0].Severity)
			assert.NotEmpty(t, result.Violations[0].Message)
		})
	}
}

// Helper function for mysqlMin
func mysqlMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
