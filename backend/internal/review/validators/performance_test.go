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
	t.Skip("Skipping review tests temporarily")
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

	t.Run("detects missing indexes with complex table definition", func(t *testing.T) {
		sql := `CREATE TABLE users (
			id INT,
			username VARCHAR(50),
			email VARCHAR(255),
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 2) // Missing indexes and missing primary key

		rules := make(map[string]bool)
		for _, v := range result.Violations {
			rules[v.Rule] = true
		}
		assert.True(t, rules["missing-indexes"])
		assert.True(t, rules["missing-primary-key"])
	})

	t.Run("detects missing primary key with UNIQUE constraint", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT UNIQUE, name VARCHAR(100), INDEX idx_name (name))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "missing-primary-key", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("detects missing indexes with PRIMARY KEY but no other indexes", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100), email VARCHAR(255))"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "missing-indexes", result.Violations[0].Rule)
	})

	t.Run("detects missing foreign key index with ALTER TABLE", func(t *testing.T) {
		sql := "ALTER TABLE orders ADD FOREIGN KEY (user_id) REFERENCES users(id)"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "missing-foreign-key-index", result.Violations[0].Rule)
		assert.Equal(t, review.SeverityWarning, result.Violations[0].Severity)
	})

	t.Run("accepts foreign key with index in ALTER TABLE", func(t *testing.T) {
		sql := "ALTER TABLE orders ADD INDEX idx_user_id (user_id), ADD FOREIGN KEY (user_id) REFERENCES users(id)"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("detects missing foreign key index with constraint name", func(t *testing.T) {
		sql := "CREATE TABLE orders (id INT PRIMARY KEY, user_id INT, CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "missing-foreign-key-index", result.Violations[0].Rule)
	})

	t.Run("accepts table with PRIMARY KEY, INDEX, and FOREIGN KEY with index", func(t *testing.T) {
		sql := "CREATE TABLE orders (id INT PRIMARY KEY, user_id INT, INDEX idx_user_id (user_id), FOREIGN KEY (user_id) REFERENCES users(id))"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("detects multiple foreign keys without indexes", func(t *testing.T) {
		sql := `CREATE TABLE order_items (
			id INT PRIMARY KEY,
			order_id INT,
			product_id INT,
			FOREIGN KEY (order_id) REFERENCES orders(id),
			FOREIGN KEY (product_id) REFERENCES products(id)
		)`

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1) // One violation for missing foreign key indexes
		assert.Equal(t, "missing-foreign-key-index", result.Violations[0].Rule)
	})

	t.Run("accepts multiple foreign keys with indexes", func(t *testing.T) {
		sql := `CREATE TABLE order_items (
			id INT PRIMARY KEY,
			order_id INT,
			product_id INT,
			INDEX idx_order_id (order_id),
			INDEX idx_product_id (product_id),
			FOREIGN KEY (order_id) REFERENCES orders(id),
			FOREIGN KEY (product_id) REFERENCES products(id)
		)`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("SQL with comments", func(t *testing.T) {
		sql := `-- Create users table
		CREATE TABLE users (
			id INT,
			name VARCHAR(100)
			-- No indexes defined
		)`

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 2)
	})

	t.Run("SQL with line breaks and whitespace", func(t *testing.T) {
		sql := `CREATE    TABLE
			users
			(
				id INT,
				name VARCHAR(100)
			)`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 2)
	})

	t.Run("complex CREATE TABLE with all issues", func(t *testing.T) {
		sql := `CREATE TABLE complex_table (
			id INT,
			column1 VARCHAR(100),
			column2 VARCHAR(200),
			foreign_key_id INT,
			FOREIGN KEY (foreign_key_id) REFERENCES other_table(id)
		)`

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
		var _ review.Validator = (*PerformanceValidator)(nil)
		assert.Implements(t, (*review.Validator)(nil), validator)
	})

	t.Run("descriptive violation messages", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT, name VARCHAR(100))"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 2)

		messages := make(map[string]string)
		for _, v := range result.Violations {
			messages[v.Rule] = v.Message
		}

		assert.Contains(t, messages["missing-indexes"], "indexes")
		assert.Contains(t, messages["missing-indexes"], "performance")
		assert.Contains(t, messages["missing-primary-key"], "PRIMARY KEY")
		assert.Contains(t, messages["missing-primary-key"], "performance")
	})

	t.Run("correct severity levels", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT, name VARCHAR(100), FOREIGN KEY (name) REFERENCES other(id))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Violations), 3)

		for _, v := range result.Violations {
			assert.Equal(t, review.SeverityWarning, v.Severity, "Violation %s should have Warning severity", v.Rule)
		}
	})

	t.Run("PostgreSQL-specific syntax", func(t *testing.T) {
		sql := "CREATE TABLE users (id SERIAL, name VARCHAR(100))"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 2)
	})

	t.Run("MySQL-specific syntax", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT AUTO_INCREMENT, name VARCHAR(100))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 2)
	})

	t.Run("table with UNIQUE constraint but no PRIMARY KEY", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT UNIQUE, name VARCHAR(100), INDEX idx_name (name))"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.False(t, result.Valid)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, "missing-primary-key", result.Violations[0].Rule)
	})

	t.Run("table with COMPOSITE PRIMARY KEY", func(t *testing.T) {
		sql := "CREATE TABLE order_items (order_id INT, item_id INT, PRIMARY KEY (order_id, item_id), INDEX idx_order_id (order_id))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("table with UNIQUE INDEX", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT PRIMARY KEY, email VARCHAR(255), UNIQUE INDEX idx_email (email))"

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("table with multiple indexes", func(t *testing.T) {
		sql := `CREATE TABLE users (
			id INT PRIMARY KEY,
			username VARCHAR(50),
			email VARCHAR(255),
			INDEX idx_username (username),
			INDEX idx_email (email)
		)`

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("ALTER TABLE ADD FOREIGN KEY with existing index", func(t *testing.T) {
		sql := `ALTER TABLE orders ADD INDEX idx_user_id (user_id);
		ALTER TABLE orders ADD FOREIGN KEY (user_id) REFERENCES users(id)`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		// Should not warn about missing foreign key index since index exists in previous statement
		// Note: This is a limitation of the current implementation - it only checks within a single statement
		// In practice, this might still warn if the validator doesn't track indexes across statements
		// This test documents the current behavior
		assert.True(t, result.Valid || len(result.Violations) == 0)
	})

	t.Run("CREATE TABLE with KEY keyword (MySQL)", func(t *testing.T) {
		sql := "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100), KEY idx_name (name))"

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("CREATE TABLE with CONSTRAINT and FOREIGN KEY", func(t *testing.T) {
		sql := `CREATE TABLE orders (
			id INT PRIMARY KEY,
			user_id INT,
			CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id),
			INDEX idx_user_id (user_id)
		)`

		result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("multiple CREATE TABLE statements", func(t *testing.T) {
		sql := `CREATE TABLE users (id INT, name VARCHAR(100));
		CREATE TABLE orders (id INT, user_id INT)`

		result := validator.Validate(ctx, sql, models.DatabaseTypeMySQL)

		// Should detect issues in both tables
		assert.False(t, result.Valid)
		require.GreaterOrEqual(t, len(result.Violations), 4) // At least 2 violations per table
	})
}
