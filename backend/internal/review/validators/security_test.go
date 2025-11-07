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

func TestSecurityValidator(t *testing.T) {
	t.Skip("Skipping review tests temporarily")
	validator := NewSecurityValidator()
	ctx := context.Background()

	t.Run("GetName returns validator name", func(t *testing.T) {
		assert.Equal(t, "security", validator.GetName())
	})

	t.Run("GetSeverity returns Error severity", func(t *testing.T) {
		assert.Equal(t, review.SeverityError, validator.GetSeverity())
	})

	t.Run("skips empty SQL", func(t *testing.T) {
		result := validator.Validate(ctx, "", models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("skips whitespace-only SQL", func(t *testing.T) {
		result := validator.Validate(ctx, "   \n\t  ", models.DatabaseTypeMySQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("validator structure is correct", func(t *testing.T) {
		assert.NotNil(t, validator)
		assert.Equal(t, "security", validator.GetName())
		assert.Equal(t, review.SeverityError, validator.GetSeverity())
		assert.Implements(t, (*review.Validator)(nil), validator)
	})

	t.Run("detects DROP DATABASE", func(t *testing.T) {
		result := validator.Validate(ctx, "DROP DATABASE mydb", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-database", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityError, result.Violations[0].Severity)
		assert.Contains(t, result.Violations[0].Message, "DROP DATABASE")
	})

	t.Run("detects DROP TABLE", func(t *testing.T) {
		result := validator.Validate(ctx, "DROP TABLE users", models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-table", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityError, result.Violations[0].Severity)
		assert.Contains(t, result.Violations[0].Message, "DROP TABLE")
	})

	t.Run("detects DROP TABLE case insensitive", func(t *testing.T) {
		result := validator.Validate(ctx, "drop table users", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-table", result.Violations[0].Rule)
	})

	t.Run("detects TRUNCATE TABLE", func(t *testing.T) {
		result := validator.Validate(ctx, "TRUNCATE TABLE users", models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "truncate-table", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityError, result.Violations[0].Severity)
		assert.Contains(t, result.Violations[0].Message, "TRUNCATE")
	})

	t.Run("detects TRUNCATE without TABLE keyword", func(t *testing.T) {
		result := validator.Validate(ctx, "TRUNCATE users", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "truncate-table", result.Violations[0].Rule)
	})

	t.Run("detects DROP COLUMN", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users DROP COLUMN email", models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-column", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
		assert.Contains(t, result.Violations[0].Message, "DROP COLUMN")
	})

	t.Run("detects DROP INDEX", func(t *testing.T) {
		result := validator.Validate(ctx, "DROP INDEX idx_users_email ON users", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-index", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects DROP CONSTRAINT", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users DROP CONSTRAINT pk_users", models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-constraint", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects DROP VIEW", func(t *testing.T) {
		result := validator.Validate(ctx, "DROP VIEW user_view", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-view", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects DROP FUNCTION", func(t *testing.T) {
		result := validator.Validate(ctx, "DROP FUNCTION calculate_total", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-function", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects DROP PROCEDURE", func(t *testing.T) {
		result := validator.Validate(ctx, "DROP PROCEDURE sp_update_user", models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-function", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects multiple destructive operations", func(t *testing.T) {
		sql := `
			DROP TABLE users;
			TRUNCATE TABLE orders;
		`

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 2)

		// Check that both violations are present
		rules := make(map[string]bool)
		for _, v := range result.Violations {
			rules[v.Rule] = true
		}
		assert.True(t, rules["drop-table"])
		assert.True(t, rules["truncate-table"])
	})

	t.Run("allows safe operations", func(t *testing.T) {
		safeSQL := []string{
			"SELECT * FROM users",
			"INSERT INTO users (name) VALUES ('test')",
			"UPDATE users SET name = 'test' WHERE id = 1",
			"DELETE FROM users WHERE id = 1",
			"CREATE TABLE users (id INT PRIMARY KEY)",
			"ALTER TABLE users ADD COLUMN email VARCHAR(255)",
			"CREATE INDEX idx_users_email ON users(email)",
		}

		for _, sql := range safeSQL {
			t.Run("safe SQL: "+sql[:securityMin(30, len(sql))], func(t *testing.T) {
				result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

				assert.True(t, result.Valid, "SQL should be safe: %s", sql)
				assert.Empty(t, result.Violations)
			})
		}
	})

	t.Run("works with both MySQL and PostgreSQL", func(t *testing.T) {
		sql := "DROP TABLE users"

		resultMySQL := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)
		resultPostgreSQL := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, resultMySQL.Valid)
		assert.False(t, resultPostgreSQL.Valid)
		assert.Len(t, resultMySQL.Violations, 1)
		assert.Len(t, resultPostgreSQL.Violations, 1)
	})

	t.Run("detects DROP DATABASE case variations", func(t *testing.T) {
		variations := []string{
			"DROP DATABASE mydb",
			"drop database mydb",
			"Drop Database mydb",
			"DROP   DATABASE   mydb",
			"DROP\nDATABASE\nmydb",
		}

		for _, sql := range variations {
			t.Run("variation: "+sql[:securityMin(20, len(sql))], func(t *testing.T) {
				result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)
				assert.False(t, result.Valid, "Should detect DROP DATABASE: %s", sql)
				require.Len(t, result.Violations, 1)
				assert.Equal(t, "drop-database", result.Violations[0].Rule)
				assert.Equal(t, review.SeverityError, result.Violations[0].Severity)
			})
		}
	})

	t.Run("detects DROP TABLE with IF EXISTS", func(t *testing.T) {
		result := validator.Validate(ctx, "DROP TABLE IF EXISTS users", models.DatabaseTypeMySQL)
		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-table", result.Violations[0].Rule)
	})

	t.Run("detects TRUNCATE with CASCADE", func(t *testing.T) {
		result := validator.Validate(ctx, "TRUNCATE TABLE users CASCADE", models.DatabaseTypePostgreSQL)
		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "truncate-table", result.Violations[0].Rule)
	})

	t.Run("detects DROP COLUMN with CASCADE", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users DROP COLUMN email CASCADE", models.DatabaseTypePostgreSQL)
		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-column", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects DROP INDEX with IF EXISTS", func(t *testing.T) {
		result := validator.Validate(ctx, "DROP INDEX IF EXISTS idx_users_email", models.DatabaseTypePostgreSQL)
		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-index", result.Violations[0].Rule)
	})

	t.Run("detects DROP FUNCTION with parameters", func(t *testing.T) {
		result := validator.Validate(ctx, "DROP FUNCTION calculate_total(INT, INT)", models.DatabaseTypePostgreSQL)
		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-function", result.Violations[0].Rule)
	})

	t.Run("detects DROP PROCEDURE with IF EXISTS", func(t *testing.T) {
		result := validator.Validate(ctx, "DROP PROCEDURE IF EXISTS sp_update_user", models.DatabaseTypeMySQL)
		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-function", result.Violations[0].Rule)
	})

	t.Run("detects DROP VIEW with CASCADE", func(t *testing.T) {
		result := validator.Validate(ctx, "DROP VIEW user_view CASCADE", models.DatabaseTypePostgreSQL)
		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-view", result.Violations[0].Rule)
	})

	t.Run("handles SQL with comments", func(t *testing.T) {
		sql := `
			-- This is a comment
			DROP TABLE users;
			/* Another comment */
			TRUNCATE TABLE orders;
		`

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)
		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 2)
	})

	t.Run("handles SQL with line breaks and whitespace", func(t *testing.T) {
		sql := `
			DROP
			TABLE
			users
		`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)
		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-table", result.Violations[0].Rule)
	})

	t.Run("handles complex ALTER TABLE with DROP COLUMN", func(t *testing.T) {
		sql := "ALTER TABLE users DROP COLUMN email, ADD COLUMN phone VARCHAR(20)"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)
		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-column", result.Violations[0].Rule)
	})

	t.Run("detects DROP CONSTRAINT by name", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users DROP CONSTRAINT pk_users", models.DatabaseTypePostgreSQL)
		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-constraint", result.Violations[0].Rule)
	})

	t.Run("handles empty SQL variations", func(t *testing.T) {
		emptyInputs := []string{
			"",
			"   ",
			"\n",
			"\t",
			"   \n\t   ",
			"-- comment only",
			"/* comment */",
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
		assert.Equal(t, "security", validator.GetName())
		assert.Equal(t, review.SeverityError, validator.GetSeverity())
	})

	t.Run("violation messages are descriptive", func(t *testing.T) {
		result := validator.Validate(ctx, "DROP TABLE users", models.DatabaseTypeMySQL)
		require.Len(t, result.Violations, 1)

		message := result.Violations[0].Message
		assert.Contains(t, message, "DROP TABLE")
		assert.Contains(t, message, "permanently delete")
		assert.Contains(t, message, "backups")
	})

	t.Run("severity levels are correct", func(t *testing.T) {
		testCases := []struct {
			sql              string
			expectedSeverity review.Severity
		}{
			{"DROP DATABASE mydb", review.SeverityError},
			{"DROP TABLE users", review.SeverityError},
			{"TRUNCATE TABLE users", review.SeverityError},
			{"ALTER TABLE users DROP COLUMN email", review.SeverityWarning},
			{"DROP INDEX idx_users_email", review.SeverityWarning},
			{"ALTER TABLE users DROP CONSTRAINT pk_users", review.SeverityWarning},
			{"DROP VIEW user_view", review.SeverityWarning},
			{"DROP FUNCTION calculate_total", review.SeverityWarning},
			{"DROP PROCEDURE sp_update_user", review.SeverityWarning},
		}

		for _, tc := range testCases {
			t.Run("severity check: "+tc.sql[:securityMin(30, len(tc.sql))], func(t *testing.T) {
				result := validator.Validate(ctx, tc.sql, models.DatabaseTypeMySQL)
				assert.False(t, result.Valid)
				require.Len(t, result.Violations, 1)
				assert.Equal(t, tc.expectedSeverity, result.Violations[0].Severity)
			})
		}
	})
}

// Helper function for securityMin
func securityMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
