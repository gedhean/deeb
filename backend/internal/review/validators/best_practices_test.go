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

func TestBestPracticesValidator(t *testing.T) {
	validator := NewBestPracticesValidator()
	ctx := context.Background()

	t.Run("GetName returns validator name", func(t *testing.T) {
		assert.Equal(t, "best-practices", validator.GetName())
	})

	t.Run("GetSeverity returns Info severity", func(t *testing.T) {
		assert.Equal(t, review.SeverityInfo, validator.GetSeverity())
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
		assert.Equal(t, "best-practices", validator.GetName())
		assert.Equal(t, review.SeverityInfo, validator.GetSeverity())
		assert.Implements(t, (*review.Validator)(nil), validator)
	})

	t.Run("detects table naming violations", func(t *testing.T) {
		tests := []struct {
			name            string
			sql             string
			expectViolation bool
			rule            string
		}{
			{
				name:            "camelCase table name",
				sql:             "CREATE TABLE UserAccounts (id INT)",
				expectViolation: true,
				rule:            "table-naming-convention",
			},
			{
				name:            "PascalCase table name",
				sql:             "CREATE TABLE UserAccounts (id INT)",
				expectViolation: true,
				rule:            "table-naming-convention",
			},
			{
				name:            "valid snake_case plural table",
				sql:             "CREATE TABLE user_accounts (id INT)",
				expectViolation: false,
			},
			{
				name:            "valid snake_case plural table ending with s",
				sql:             "CREATE TABLE users (id INT)",
				expectViolation: false,
			},
			{
				name:            "singular table name",
				sql:             "CREATE TABLE user_account (id INT)",
				expectViolation: true,
				rule:            "table-plural-form",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := validator.Validate(ctx, tt.sql, models.DatabaseTypeMySQL)

				if tt.expectViolation {
					assert.False(t, result.Valid)
					require.NotEmpty(t, result.Violations)
					assert.Equal(t, tt.rule, result.Violations[0].Rule)
					assert.Equal(t, review.SeverityInfo, result.Violations[0].Severity)
				} else {
					// May still have other violations (e.g., column naming)
					// But should not have table naming violations
					found := false
					for _, v := range result.Violations {
						if v.Rule == tt.rule {
							found = true
							break
						}
					}
					assert.False(t, found, "Should not have violation with rule: %s", tt.rule)
				}
			})
		}
	})

	t.Run("detects column naming violations", func(t *testing.T) {
		tests := []struct {
			name            string
			sql             string
			expectViolation bool
			rule            string
		}{
			{
				name:            "camelCase column name",
				sql:             "CREATE TABLE users (userId INT)",
				expectViolation: true,
				rule:            "column-naming-convention",
			},
			{
				name:            "valid snake_case column",
				sql:             "CREATE TABLE users (user_id INT)",
				expectViolation: false,
			},
			{
				name:            "plural column name",
				sql:             "CREATE TABLE users (users_id INT)",
				expectViolation: true,
				rule:            "column-singular-form",
			},
			{
				name:            "valid singular column",
				sql:             "CREATE TABLE users (id INT, name VARCHAR(100))",
				expectViolation: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := validator.Validate(ctx, tt.sql, models.DatabaseTypePostgreSQL)

				if tt.expectViolation {
					assert.False(t, result.Valid)
					found := false
					for _, v := range result.Violations {
						if v.Rule == tt.rule {
							found = true
							assert.Equal(t, review.SeverityInfo, v.Severity)
							break
						}
					}
					assert.True(t, found, "Should have violation with rule: %s", tt.rule)
				}
			})
		}
	})

	t.Run("detects index naming violations", func(t *testing.T) {
		tests := []struct {
			name            string
			sql             string
			expectViolation bool
			rule            string
		}{
			{
				name:            "index without idx_ prefix",
				sql:             "CREATE INDEX users_email ON users(email)",
				expectViolation: true,
				rule:            "index-naming-pattern",
			},
			{
				name:            "valid index name",
				sql:             "CREATE INDEX idx_users_email ON users(email)",
				expectViolation: false,
			},
			{
				name:            "camelCase index name",
				sql:             "CREATE INDEX idxUsersEmail ON users(email)",
				expectViolation: true,
				rule:            "index-naming-convention",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := validator.Validate(ctx, tt.sql, models.DatabaseTypeMySQL)

				if tt.expectViolation {
					assert.False(t, result.Valid)
					found := false
					for _, v := range result.Violations {
						if v.Rule == tt.rule {
							found = true
							assert.Equal(t, review.SeverityInfo, v.Severity)
							break
						}
					}
					assert.True(t, found, "Should have violation with rule: %s", tt.rule)
				}
			})
		}
	})

	t.Run("detects TEXT type usage", func(t *testing.T) {
		result := validator.Validate(ctx, "CREATE TABLE posts (id INT, content TEXT)", models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		found := false
		for _, v := range result.Violations {
			if v.Rule == "text-type-usage" {
				found = true
				assert.Equal(t, review.SeverityInfo, v.Severity)
				assert.Contains(t, v.Message, "TEXT")
				break
			}
		}
		assert.True(t, found, "Should detect TEXT type usage")
	})

	t.Run("allows valid SQL following best practices", func(t *testing.T) {
		sql := `
			CREATE TABLE user_accounts (
				id INT PRIMARY KEY,
				user_id INT NOT NULL,
				email VARCHAR(255),
				created_at TIMESTAMP,
				INDEX idx_user_accounts_user_id (user_id)
			)
		`

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		// Should not have naming convention violations
		namingViolations := 0
		for _, v := range result.Violations {
			if strings.Contains(v.Rule, "naming") || strings.Contains(v.Rule, "plural") || strings.Contains(v.Rule, "singular") {
				namingViolations++
			}
		}
		assert.Equal(t, 0, namingViolations, "Should not have naming convention violations for valid SQL")
	})

	t.Run("works with both MySQL and PostgreSQL", func(t *testing.T) {
		sql := "CREATE TABLE users (userId INT)"

		resultMySQL := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)
		resultPostgreSQL := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, resultMySQL.Valid)
		assert.False(t, resultPostgreSQL.Valid)
		assert.NotEmpty(t, resultMySQL.Violations)
		assert.NotEmpty(t, resultPostgreSQL.Violations)
	})

	t.Run("handles ALTER TABLE ADD COLUMN", func(t *testing.T) {
		sql := "ALTER TABLE users ADD COLUMN userEmail VARCHAR(255)"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		found := false
		for _, v := range result.Violations {
			if v.Rule == "column-naming-convention" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect column naming violation in ALTER TABLE")
	})
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
