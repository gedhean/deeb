package validators

import (
	"context"
	"strings"
	"testing"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/review"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackwardsCompatibilityValidator(t *testing.T) {
	validator := NewBackwardsCompatibilityValidator()
	ctx := context.Background()

	t.Run("GetName returns validator name", func(t *testing.T) {
		assert.Equal(t, "backwards-compatibility", validator.GetName())
	})

	t.Run("GetSeverity returns Warning severity", func(t *testing.T) {
		assert.Equal(t, review.SeverityWarning, validator.GetSeverity())
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
		assert.Equal(t, "backwards-compatibility", validator.GetName())
		assert.Equal(t, review.SeverityWarning, validator.GetSeverity())
		assert.Implements(t, (*review.Validator)(nil), validator)
	})

	t.Run("detects DROP COLUMN", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users DROP COLUMN email", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-column", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
		assert.Contains(t, result.Violations[0].Message, "DROP COLUMN")
	})

	t.Run("detects RENAME TABLE", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users RENAME TO accounts", models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "rename-table", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects RENAME TABLE PostgreSQL syntax", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users RENAME TO accounts", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "rename-table", result.Violations[0].Rule)
	})

	t.Run("detects RENAME COLUMN", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users RENAME COLUMN email TO email_address", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "rename-column", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects ALTER COLUMN TYPE", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users ALTER COLUMN email TYPE VARCHAR(255)", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "alter-column-type", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects MODIFY COLUMN", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users MODIFY COLUMN email VARCHAR(255)", models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "modify-column", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects CHANGE COLUMN", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users CHANGE COLUMN email email_address VARCHAR(255)", models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "change-column", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects DROP CONSTRAINT", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users DROP CONSTRAINT pk_users", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-constraint", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects SET NOT NULL", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users ALTER COLUMN email SET NOT NULL", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "set-not-null", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects DROP DEFAULT", func(t *testing.T) {
		result := validator.Validate(ctx, "ALTER TABLE users ALTER COLUMN email DROP DEFAULT", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-default", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects multiple breaking changes", func(t *testing.T) {
		sql := `
			ALTER TABLE users DROP COLUMN email;
			ALTER TABLE users RENAME COLUMN name TO full_name;
		`

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Violations), 2)

		rules := make(map[string]bool)
		for _, v := range result.Violations {
			rules[v.Rule] = true
		}
		assert.True(t, rules["drop-column"])
		assert.True(t, rules["rename-column"])
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		result := validator.Validate(ctx, "alter table users drop column email", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-column", result.Violations[0].Rule)
	})

	t.Run("allows safe ALTER TABLE operations", func(t *testing.T) {
		safeSQL := []string{
			"ALTER TABLE users ADD COLUMN email VARCHAR(255)",
			"ALTER TABLE users ADD CONSTRAINT pk_users PRIMARY KEY (id)",
			"ALTER TABLE users ADD INDEX idx_email (email)",
			"ALTER TABLE users ALTER COLUMN email SET DEFAULT 'test@example.com'",
		}

		for _, sql := range safeSQL {
			t.Run("safe SQL: "+sql[:min(40, len(sql))], func(t *testing.T) {
				result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

				// Some operations might still trigger warnings (e.g., SET DEFAULT might be detected)
				// But ADD operations should be safe
				if strings.Contains(sql, "ADD") {
					assert.True(t, result.Valid, "SQL should be safe: %s", sql)
					assert.Empty(t, result.Violations)
				}
			})
		}
	})

	t.Run("works with both MySQL and PostgreSQL", func(t *testing.T) {
		sql := "ALTER TABLE users DROP COLUMN email"

		resultMySQL := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)
		resultPostgreSQL := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, resultMySQL.Valid)
		assert.False(t, resultPostgreSQL.Valid)
		assert.Len(t, resultMySQL.Violations, 1)
		assert.Len(t, resultPostgreSQL.Violations, 1)
	})

	t.Run("detects DROP COLUMN with IF EXISTS", func(t *testing.T) {
		sql := "ALTER TABLE users DROP COLUMN IF EXISTS email"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-column", result.Violations[0].Rule)
	})

	t.Run("detects DROP COLUMN with CASCADE", func(t *testing.T) {
		sql := "ALTER TABLE users DROP COLUMN email CASCADE"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-column", result.Violations[0].Rule)
	})

	t.Run("detects DROP COLUMN with RESTRICT", func(t *testing.T) {
		sql := "ALTER TABLE users DROP COLUMN email RESTRICT"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-column", result.Violations[0].Rule)
	})

	t.Run("detects multiple DROP COLUMN in one statement", func(t *testing.T) {
		sql := "ALTER TABLE users DROP COLUMN email, DROP COLUMN phone"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 2)
		assert.Equal(t, "drop-column", result.Violations[0].Rule)
		assert.Equal(t, "drop-column", result.Violations[1].Rule)
	})

	t.Run("detects RENAME TABLE MySQL syntax", func(t *testing.T) {
		sql := "RENAME TABLE users TO accounts"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "rename-table", result.Violations[0].Rule)
	})

	t.Run("detects RENAME TABLE with multiple tables", func(t *testing.T) {
		sql := "RENAME TABLE users TO accounts, orders TO transactions"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Violations), 1)
		assert.Equal(t, "rename-table", result.Violations[0].Rule)
	})

	t.Run("detects RENAME COLUMN PostgreSQL syntax", func(t *testing.T) {
		sql := "ALTER TABLE users RENAME COLUMN email TO email_address"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "rename-column", result.Violations[0].Rule)
	})

	t.Run("detects ALTER COLUMN TYPE with USING clause", func(t *testing.T) {
		sql := "ALTER TABLE users ALTER COLUMN email TYPE VARCHAR(255) USING email::VARCHAR"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "alter-column-type", result.Violations[0].Rule)
	})

	t.Run("detects ALTER COLUMN TYPE with COLLATE", func(t *testing.T) {
		sql := "ALTER TABLE users ALTER COLUMN email TYPE VARCHAR(255) COLLATE utf8mb4_unicode_ci"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "alter-column-type", result.Violations[0].Rule)
	})

	t.Run("detects MODIFY COLUMN with NOT NULL", func(t *testing.T) {
		sql := "ALTER TABLE users MODIFY COLUMN email VARCHAR(255) NOT NULL"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 2) // MODIFY COLUMN and SET NOT NULL
		rules := make(map[string]bool)
		for _, v := range result.Violations {
			rules[v.Rule] = true
		}
		assert.True(t, rules["modify-column"])
		assert.True(t, rules["set-not-null"])
	})

	t.Run("detects MODIFY COLUMN with DEFAULT", func(t *testing.T) {
		sql := "ALTER TABLE users MODIFY COLUMN email VARCHAR(255) DEFAULT 'test@example.com'"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "modify-column", result.Violations[0].Rule)
	})

	t.Run("detects CHANGE COLUMN with only name change", func(t *testing.T) {
		sql := "ALTER TABLE users CHANGE COLUMN email email_address VARCHAR(255)"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "change-column", result.Violations[0].Rule)
	})

	t.Run("detects CHANGE COLUMN with type change", func(t *testing.T) {
		sql := "ALTER TABLE users CHANGE COLUMN email email TEXT"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "change-column", result.Violations[0].Rule)
	})

	t.Run("detects DROP CONSTRAINT with specific name", func(t *testing.T) {
		sql := "ALTER TABLE users DROP CONSTRAINT chk_email_format"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-constraint", result.Violations[0].Rule)
	})

	t.Run("detects DROP CONSTRAINT with CASCADE", func(t *testing.T) {
		sql := "ALTER TABLE users DROP CONSTRAINT pk_users CASCADE"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-constraint", result.Violations[0].Rule)
	})

	t.Run("detects SET NOT NULL with existing data check", func(t *testing.T) {
		sql := "ALTER TABLE users ALTER COLUMN email SET NOT NULL"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "set-not-null", result.Violations[0].Rule)
		assert.Contains(t, result.Violations[0].Message, "NULL")
	})

	t.Run("detects DROP DEFAULT with column name", func(t *testing.T) {
		sql := "ALTER TABLE users ALTER COLUMN email DROP DEFAULT"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-default", result.Violations[0].Rule)
	})

	t.Run("SQL with comments", func(t *testing.T) {
		sql := `-- Drop email column
		ALTER TABLE users DROP COLUMN email
		-- This is a breaking change`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-column", result.Violations[0].Rule)
	})

	t.Run("SQL with line breaks and whitespace", func(t *testing.T) {
		sql := `ALTER    TABLE
			users
			DROP
			COLUMN
			email`

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "drop-column", result.Violations[0].Rule)
	})

	t.Run("complex ALTER TABLE with multiple breaking changes", func(t *testing.T) {
		sql := `ALTER TABLE users
			DROP COLUMN email,
			RENAME COLUMN name TO full_name,
			ALTER COLUMN age TYPE INTEGER`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Violations), 3)

		rules := make(map[string]bool)
		for _, v := range result.Violations {
			rules[v.Rule] = true
		}
		assert.True(t, rules["drop-column"])
		assert.True(t, rules["rename-column"])
		assert.True(t, rules["alter-column-type"])
	})

	t.Run("empty SQL variations", func(t *testing.T) {
		testCases := []string{
			"",
			"   ",
			"\n\t",
			"  \n  \t  ",
		}

		for _, sql := range testCases {
			result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)
			assert.True(t, result.Valid, "SQL: %q", sql)
			assert.Empty(t, result.Violations, "SQL: %q", sql)
		}
	})

	t.Run("interface implementation", func(t *testing.T) {
		var _ review.Validator = (*BackwardsCompatibilityValidator)(nil)
		assert.Implements(t, (*review.Validator)(nil), validator)
	})

	t.Run("descriptive violation messages", func(t *testing.T) {
		testCases := []struct {
			sql         string
			rule        string
			expectedMsg string
		}{
			{"ALTER TABLE users DROP COLUMN email", "drop-column", "DROP COLUMN"},
			{"ALTER TABLE users RENAME TO accounts", "rename-table", "RENAME TABLE"},
			{"ALTER TABLE users RENAME COLUMN email TO email_address", "rename-column", "RENAME COLUMN"},
			{"ALTER TABLE users ALTER COLUMN email TYPE VARCHAR(255)", "alter-column-type", "data type change"},
			{"ALTER TABLE users ALTER COLUMN email SET NOT NULL", "set-not-null", "SET NOT NULL"},
			{"ALTER TABLE users ALTER COLUMN email DROP DEFAULT", "drop-default", "DROP DEFAULT"},
		}

		for _, tc := range testCases {
			result := validator.Validate(ctx, tc.sql, models.DatabaseTypePostgreSQL)
			assert.False(t, result.Valid, "SQL: %s", tc.sql)
			require.GreaterOrEqual(t, len(result.Violations), 1, "SQL: %s", tc.sql)

			found := false
			for _, v := range result.Violations {
				if v.Rule == tc.rule {
					assert.Contains(t, v.Message, tc.expectedMsg, "SQL: %s, Rule: %s", tc.sql, tc.rule)
					found = true
					break
				}
			}
			assert.True(t, found, "Expected violation for rule %s in SQL: %s", tc.rule, tc.sql)
		}
	})

	t.Run("correct severity levels", func(t *testing.T) {
		sql := `ALTER TABLE users
			DROP COLUMN email,
			RENAME COLUMN name TO full_name,
			ALTER COLUMN age TYPE INTEGER,
			ALTER COLUMN status SET NOT NULL`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Violations), 4)

		for _, v := range result.Violations {
			assert.Equal(t, review.SeverityWarning, v.Severity, "Violation %s should have Warning severity", v.Rule)
		}
	})

	t.Run("PostgreSQL-specific ALTER COLUMN syntax", func(t *testing.T) {
		sql := "ALTER TABLE users ALTER COLUMN email TYPE TEXT USING email::TEXT"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "alter-column-type", result.Violations[0].Rule)
	})

	t.Run("MySQL-specific MODIFY syntax variations", func(t *testing.T) {
		sql := "ALTER TABLE users MODIFY email VARCHAR(255)"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "modify-column", result.Violations[0].Rule)
	})

	t.Run("multiple statements with breaking changes", func(t *testing.T) {
		sql := `ALTER TABLE users DROP COLUMN email;
		ALTER TABLE orders RENAME TO transactions;
		ALTER TABLE products RENAME COLUMN price TO cost`

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Violations), 3)

		rules := make(map[string]bool)
		for _, v := range result.Violations {
			rules[v.Rule] = true
		}
		assert.True(t, rules["drop-column"])
		assert.True(t, rules["rename-table"])
		assert.True(t, rules["rename-column"])
	})

	t.Run("allows safe operations", func(t *testing.T) {
		safeSQL := []string{
			"ALTER TABLE users ADD COLUMN email VARCHAR(255)",
			"ALTER TABLE users ADD CONSTRAINT pk_users PRIMARY KEY (id)",
			"ALTER TABLE users ADD INDEX idx_email (email)",
			"ALTER TABLE users ADD FOREIGN KEY (user_id) REFERENCES accounts(id)",
			"CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100))",
			"CREATE INDEX idx_name ON users (name)",
		}

		for _, sql := range safeSQL {
			result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)
			assert.True(t, result.Valid, "SQL should be safe: %s", sql)
			assert.Empty(t, result.Violations, "SQL should have no violations: %s", sql)
		}
	})

	t.Run("case insensitive detection", func(t *testing.T) {
		testCases := []struct {
			sql  string
			rule string
		}{
			{"alter table users drop column email", "drop-column"},
			{"ALTER TABLE users RENAME TO accounts", "rename-table"},
			{"alter table users rename column email to email_address", "rename-column"},
			{"ALTER TABLE users ALTER COLUMN email TYPE VARCHAR(255)", "alter-column-type"},
			{"alter table users modify column email varchar(255)", "modify-column"},
			{"ALTER TABLE users CHANGE COLUMN email email_address VARCHAR(255)", "change-column"},
			{"alter table users drop constraint pk_users", "drop-constraint"},
			{"ALTER TABLE users ALTER COLUMN email SET NOT NULL", "set-not-null"},
			{"alter table users alter column email drop default", "drop-default"},
		}

		for _, tc := range testCases {
			result := validator.Validate(ctx, tc.sql, models.DatabaseTypePostgreSQL)
			assert.False(t, result.Valid, "SQL: %s", tc.sql)
			require.GreaterOrEqual(t, len(result.Violations), 1, "SQL: %s", tc.sql)

			found := false
			for _, v := range result.Violations {
				if v.Rule == tc.rule {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected violation for rule %s in SQL: %s", tc.rule, tc.sql)
		}
	})
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
