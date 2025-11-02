package validators

import (
	"context"
	"testing"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/review"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformanceValidator(t *testing.T) {
	validator := NewPerformanceValidator()
	ctx := context.Background()

	t.Run("GetName returns validator name", func(t *testing.T) {
		assert.Equal(t, "performance", validator.GetName())
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
		assert.Equal(t, "performance", validator.GetName())
		assert.Equal(t, review.SeverityWarning, validator.GetSeverity())
		assert.Implements(t, (*review.Validator)(nil), validator)
	})

	t.Run("detects missing indexes", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT, name VARCHAR(100))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 2) // Missing indexes and missing primary key

		rules := make(map[string]bool)
		for _, v := range result.Violations {
			rules[v.Rule] = true
		}
		assert.True(t, rules["missing-indexes"])
		assert.True(t, rules["missing-primary-key"])
	})

	t.Run("detects missing primary key", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT, name VARCHAR(100), INDEX idx_name (name))"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "missing-primary-key", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("accepts table with primary key", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		// Should still warn about missing indexes (non-primary key indexes)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "missing-indexes", result.Violations[0].Rule)
	})

	t.Run("accepts table with primary key and indexes", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100), INDEX idx_name (name))"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("detects missing foreign key index", func(t *testing.T) {
		sql := "CREATE TABLE orders (id INT PRIMARY KEY, user_id INT, FOREIGN KEY (user_id) REFERENCES users(id))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		// Should warn about missing foreign key index
		require.GreaterOrEqual(t, len(result.Violations), 1)

		rules := make(map[string]bool)
		for _, v := range result.Violations {
			rules[v.Rule] = true
		}
		assert.True(t, rules["missing-foreign-key-index"])
	})

	t.Run("accepts foreign key with index", func(t *testing.T) {
		sql := "CREATE TABLE orders (id INT PRIMARY KEY, user_id INT, INDEX idx_user_id (user_id), FOREIGN KEY (user_id) REFERENCES users(id))"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		// Should not warn about missing foreign key index since index exists
		// May still warn about missing indexes on other columns
		rules := make(map[string]bool)
		for _, v := range result.Violations {
			rules[v.Rule] = true
		}
		assert.False(t, rules["missing-foreign-key-index"])
	})

	t.Run("detects multiple performance issues", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT, name VARCHAR(100), email VARCHAR(255), FOREIGN KEY (email) REFERENCES emails(id))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Violations), 3)

		rules := make(map[string]bool)
		for _, v := range result.Violations {
			rules[v.Rule] = true
		}
		assert.True(t, rules["missing-indexes"])
		assert.True(t, rules["missing-primary-key"])
		assert.True(t, rules["missing-foreign-key-index"])
	})

	t.Run("works with both MySQL and PostgreSQL", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT, name VARCHAR(100))"

		resultMySQL := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)
		resultPostgreSQL := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, resultMySQL.Valid)
		assert.False(t, resultPostgreSQL.Valid)
		assert.Len(t, resultMySQL.Violations, 2)
		assert.Len(t, resultPostgreSQL.Violations, 2)
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		sql := "create table users (id int, name varchar(100))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 2)
	})

	t.Run("allows ALTER TABLE ADD INDEX", func(t *testing.T) {
		sql := "ALTER TABLE users ADD INDEX idx_name (name)"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("allows ALTER TABLE ADD PRIMARY KEY", func(t *testing.T) {
		sql := "ALTER TABLE users ADD PRIMARY KEY (id)"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})
}
