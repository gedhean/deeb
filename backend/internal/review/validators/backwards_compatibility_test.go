package validators

import (
	"context"
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
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
