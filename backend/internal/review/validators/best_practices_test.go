//go:build ignore

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
	t.Skip("Skipping review tests temporarily")
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

	t.Run("detects multiple naming violations", func(t *testing.T) {
		sql := "CREATE TABLE UserAccounts (userId INT, usersEmail VARCHAR(255), INDEX usersEmailIndex ON UserAccounts(usersEmail))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Violations), 3)

		rules := make(map[string]bool)
		for _, v := range result.Violations {
			rules[v.Rule] = true
		}
		assert.True(t, rules["table-naming-convention"])
		assert.True(t, rules["column-naming-convention"])
		assert.True(t, rules["index-naming-pattern"])
	})

	t.Run("detects table naming violations with IF NOT EXISTS", func(t *testing.T) {
		sql := "CREATE TABLE IF NOT EXISTS UserAccounts (id INT)"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		found := false
		for _, v := range result.Violations {
			if v.Rule == "table-naming-convention" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect table naming violation with IF NOT EXISTS")
	})

	t.Run("accepts plural table names ending with s", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT)"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		// Should not have table plural form violation
		found := false
		for _, v := range result.Violations {
			if v.Rule == "table-plural-form" {
				found = true
				break
			}
		}
		assert.False(t, found, "Should not flag plural table names ending with 's'")
	})

	t.Run("accepts plural table names ending with ies", func(t *testing.T) {
		sql := "CREATE TABLE categories (id INT)"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		// Should not have table plural form violation
		found := false
		for _, v := range result.Violations {
			if v.Rule == "table-plural-form" {
				found = true
				break
			}
		}
		assert.False(t, found, "Should not flag plural table names ending with 'ies'")
	})

	t.Run("accepts plural table names ending with es", func(t *testing.T) {
		sql := "CREATE TABLE addresses (id INT)"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		// Should not have table plural form violation
		found := false
		for _, v := range result.Violations {
			if v.Rule == "table-plural-form" {
				found = true
				break
			}
		}
		assert.False(t, found, "Should not flag plural table names ending with 'es'")
	})

	t.Run("accepts common singular exceptions", func(t *testing.T) {
		exceptions := []string{"data", "metadata", "schema", "config", "settings", "info"}

		for _, name := range exceptions {
			sql := "CREATE TABLE " + name + " (id INT)"
			result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

			// Should not have table plural form violation for exceptions
			found := false
			for _, v := range result.Violations {
				if v.Rule == "table-plural-form" {
					found = true
					break
				}
			}
			assert.False(t, found, "Should not flag singular exception: %s", name)
		}
	})

	t.Run("detects column naming violations with various data types", func(t *testing.T) {
		testCases := []string{
			"CREATE TABLE users (userId INT)",
			"CREATE TABLE users (userEmail VARCHAR(255))",
			"CREATE TABLE users (createdAt TIMESTAMP)",
			"CREATE TABLE users (isActive BOOLEAN)",
			"CREATE TABLE users (userUuid UUID)",
		}

		for _, sql := range testCases {
			result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)
			found := false
			for _, v := range result.Violations {
				if v.Rule == "column-naming-convention" {
					found = true
					break
				}
			}
			assert.True(t, found, "Should detect column naming violation in: %s", sql)
		}
	})

	t.Run("accepts common plural column exceptions", func(t *testing.T) {
		exceptions := []string{"settings", "credentials", "permissions", "preferences"}

		for _, name := range exceptions {
			sql := "CREATE TABLE users (" + name + " VARCHAR(255))"
			result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

			// Should not have column singular form violation for exceptions
			found := false
			for _, v := range result.Violations {
				if v.Rule == "column-singular-form" {
					found = true
					break
				}
			}
			assert.False(t, found, "Should not flag plural exception: %s", name)
		}
	})

	t.Run("detects index naming violations in CREATE INDEX", func(t *testing.T) {
		sql := "CREATE INDEX users_email ON users(email)"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		found := false
		for _, v := range result.Violations {
			if v.Rule == "index-naming-pattern" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect index naming pattern violation")
	})

	t.Run("detects index naming violations in CREATE UNIQUE INDEX", func(t *testing.T) {
		sql := "CREATE UNIQUE INDEX users_email_unique ON users(email)"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		found := false
		for _, v := range result.Violations {
			if v.Rule == "index-naming-pattern" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect index naming pattern violation in UNIQUE INDEX")
	})

	t.Run("detects index naming violations in ALTER TABLE ADD INDEX", func(t *testing.T) {
		sql := "ALTER TABLE users ADD INDEX users_email (email)"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		found := false
		for _, v := range result.Violations {
			if v.Rule == "index-naming-pattern" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect index naming pattern violation in ALTER TABLE")
	})

	t.Run("detects index naming violations in ALTER TABLE ADD KEY", func(t *testing.T) {
		sql := "ALTER TABLE users ADD KEY users_email (email)"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		found := false
		for _, v := range result.Violations {
			if v.Rule == "index-naming-pattern" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect index naming pattern violation in ALTER TABLE ADD KEY")
	})

	t.Run("accepts valid index names with idx_ prefix", func(t *testing.T) {
		sql := "CREATE INDEX idx_users_email ON users(email)"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		// Should not have index naming pattern violation
		found := false
		for _, v := range result.Violations {
			if v.Rule == "index-naming-pattern" {
				found = true
				break
			}
		}
		assert.False(t, found, "Should not flag valid index names with idx_ prefix")
	})

	t.Run("detects TEXT type usage in CREATE TABLE", func(t *testing.T) {
		sql := "CREATE TABLE posts (id INT, content TEXT)"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		found := false
		for _, v := range result.Violations {
			if v.Rule == "text-type-usage" {
				found = true
				assert.Contains(t, v.Message, "TEXT")
				assert.Contains(t, v.Message, "VARCHAR")
				break
			}
		}
		assert.True(t, found, "Should detect TEXT type usage")
	})

	t.Run("detects TEXT type usage in ALTER TABLE", func(t *testing.T) {
		sql := "ALTER TABLE posts ADD COLUMN description TEXT"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		found := false
		for _, v := range result.Violations {
			if v.Rule == "text-type-usage" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect TEXT type usage in ALTER TABLE")
	})

	t.Run("detects multiple TEXT columns", func(t *testing.T) {
		sql := "CREATE TABLE posts (id INT, content TEXT, description TEXT)"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		textViolations := 0
		for _, v := range result.Violations {
			if v.Rule == "text-type-usage" {
				textViolations++
			}
		}
		assert.GreaterOrEqual(t, textViolations, 2, "Should detect multiple TEXT type violations")
	})

	t.Run("SQL with comments", func(t *testing.T) {
		sql := `-- Create users table
		CREATE TABLE UserAccounts (
			userId INT
		)`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		found := false
		for _, v := range result.Violations {
			if v.Rule == "table-naming-convention" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect violations even with comments")
	})

	t.Run("SQL with line breaks and whitespace", func(t *testing.T) {
		sql := `CREATE    TABLE
			UserAccounts
			(
				userId INT
			)`

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		found := false
		for _, v := range result.Violations {
			if v.Rule == "table-naming-convention" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should detect violations with line breaks")
	})

	t.Run("complex CREATE TABLE with multiple violations", func(t *testing.T) {
		sql := `CREATE TABLE UserAccounts (
			userId INT PRIMARY KEY,
			usersEmail VARCHAR(255),
			content TEXT,
			INDEX usersEmailIndex (usersEmail)
		)`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Violations), 4)

		rules := make(map[string]bool)
		for _, v := range result.Violations {
			rules[v.Rule] = true
		}
		assert.True(t, rules["table-naming-convention"])
		assert.True(t, rules["column-naming-convention"])
		assert.True(t, rules["column-singular-form"])
		assert.True(t, rules["text-type-usage"])
		assert.True(t, rules["index-naming-pattern"])
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
		var _ review.Validator = (*BestPracticesValidator)(nil)
		assert.Implements(t, (*review.Validator)(nil), validator)
	})

	t.Run("descriptive violation messages", func(t *testing.T) {
		testCases := []struct {
			sql         string
			rule        string
			expectedMsg string
		}{
			{"CREATE TABLE UserAccounts (id INT)", "table-naming-convention", "snake_case"},
			{"CREATE TABLE users (userId INT)", "column-naming-convention", "snake_case"},
			{"CREATE INDEX users_email ON users(email)", "index-naming-pattern", "idx_"},
			{"CREATE TABLE posts (content TEXT)", "text-type-usage", "TEXT"},
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
		sql := `CREATE TABLE UserAccounts (
			userId INT,
			usersEmail VARCHAR(255),
			content TEXT,
			INDEX usersEmailIndex (usersEmail)
		)`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Violations), 4)

		for _, v := range result.Violations {
			assert.Equal(t, review.SeverityInfo, v.Severity, "Violation %s should have Info severity", v.Rule)
		}
	})

	t.Run("PostgreSQL-specific syntax", func(t *testing.T) {
		sql := "CREATE TABLE users (id SERIAL, user_email VARCHAR(255))"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		// Should not have naming convention violations for valid SQL
		namingViolations := 0
		for _, v := range result.Violations {
			if strings.Contains(v.Rule, "naming") || strings.Contains(v.Rule, "plural") || strings.Contains(v.Rule, "singular") {
				namingViolations++
			}
		}
		assert.Equal(t, 0, namingViolations, "Should not have naming violations for valid PostgreSQL syntax")
	})

	t.Run("MySQL-specific syntax", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT AUTO_INCREMENT, user_email VARCHAR(255))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		// Should not have naming convention violations for valid SQL
		namingViolations := 0
		for _, v := range result.Violations {
			if strings.Contains(v.Rule, "naming") || strings.Contains(v.Rule, "plural") || strings.Contains(v.Rule, "singular") {
				namingViolations++
			}
		}
		assert.Equal(t, 0, namingViolations, "Should not have naming violations for valid MySQL syntax")
	})

	t.Run("multiple statements with violations", func(t *testing.T) {
		sql := `CREATE TABLE UserAccounts (userId INT);
		CREATE INDEX usersEmailIndex ON UserAccounts(userId);
		ALTER TABLE UserAccounts ADD COLUMN usersEmail TEXT`

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Violations), 3)
	})

	t.Run("case insensitive detection", func(t *testing.T) {
		testCases := []struct {
			sql  string
			rule string
		}{
			{"create table UserAccounts (id int)", "table-naming-convention"},
			{"CREATE TABLE users (userId INT)", "column-naming-convention"},
			{"CREATE INDEX usersEmailIndex ON users(email)", "index-naming-pattern"},
			{"CREATE TABLE posts (content text)", "text-type-usage"},
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

	t.Run("valid SQL following all best practices", func(t *testing.T) {
		sql := `CREATE TABLE user_accounts (
			id INT PRIMARY KEY,
			user_id INT NOT NULL,
			email VARCHAR(255),
			created_at TIMESTAMP,
			INDEX idx_user_accounts_user_id (user_id)
		)`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		// Should not have any naming convention violations
		namingViolations := 0
		for _, v := range result.Violations {
			if strings.Contains(v.Rule, "naming") || strings.Contains(v.Rule, "plural") || strings.Contains(v.Rule, "singular") {
				namingViolations++
			}
		}
		assert.Equal(t, 0, namingViolations, "Should not have naming violations for valid best practices SQL")
	})
}
