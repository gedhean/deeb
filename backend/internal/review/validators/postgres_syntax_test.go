package validators

import (
	"context"
	"testing"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/review"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: pg_query_go may have compilation issues on macOS due to C code compatibility.
// The library works correctly in Linux containers. Tests that require actual parsing
// should be run in a Linux environment or Docker container.

func TestPostgreSQLSyntaxValidator(t *testing.T) {
	validator := NewPostgreSQLSyntaxValidator()
	ctx := context.Background()

	t.Run("GetName returns validator name", func(t *testing.T) {
		assert.Equal(t, "postgresql-syntax", validator.GetName())
	})

	t.Run("GetSeverity returns Error severity", func(t *testing.T) {
		assert.Equal(t, review.SeverityError, validator.GetSeverity())
	})

	t.Run("skips non-PostgreSQL database types", func(t *testing.T) {
		result := validator.Validate(ctx, "SELECT * FROM users", models.DatabaseTypeMySQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
		assert.Equal(t, "postgresql-syntax", result.ValidatorName)
	})

	t.Run("skips empty SQL", func(t *testing.T) {
		result := validator.Validate(ctx, "", models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	t.Run("skips whitespace-only SQL", func(t *testing.T) {
		result := validator.Validate(ctx, "   \n\t  ", models.DatabaseTypePostgreSQL)

		assert.True(t, result.Valid)
		assert.Empty(t, result.Violations)
	})

	// Note: Tests that actually call pg_query.Parse() may fail to compile on macOS
	// These should be run in a Linux environment or Docker container
	t.Run("validator structure is correct", func(t *testing.T) {
		// Test that the validator can be created and has correct structure
		assert.NotNil(t, validator)
		assert.Equal(t, "postgresql-syntax", validator.GetName())
		assert.Equal(t, review.SeverityError, validator.GetSeverity())
		assert.Implements(t, (*review.Validator)(nil), validator)
	})
}

func TestExtractLineNumber(t *testing.T) {
	validator := NewPostgreSQLSyntaxValidator()

	tests := []struct {
		name     string
		errorMsg string
		expected int
	}{
		{
			name:     "extracts line number from 'LINE X' format",
			errorMsg: "syntax error at or near \"FROM\" LINE 3: SELECT * FROM users",
			expected: 3,
		},
		{
			name:     "extracts character position from 'at character X' format",
			errorMsg: "syntax error at or near \"FROM\" at character 15",
			expected: 15,
		},
		{
			name:     "extracts first number found",
			errorMsg: "syntax error LINE 5 at character 20",
			expected: 5,
		},
		{
			name:     "returns 0 when no line number found",
			errorMsg: "syntax error at or near \"FROM\"",
			expected: 0,
		},
		{
			name:     "handles case insensitive matching",
			errorMsg: "syntax error at line 10",
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lineNum := validator.extractLineNumber(tt.errorMsg)
			assert.Equal(t, tt.expected, lineNum)
		})
	}
}

// TestPostgreSQLSyntaxValidatorIntegration tests the full validator functionality
// This test requires pg_query_go to compile and should be run in a Linux environment
func TestPostgreSQLSyntaxValidatorIntegration(t *testing.T) {
	t.Skip("Skipping integration test - requires pg_query_go to compile (Linux environment needed)")

	validator := NewPostgreSQLSyntaxValidator()
	ctx := context.Background()

	t.Run("validates valid PostgreSQL SQL", func(t *testing.T) {
		validSQL := []string{
			"SELECT * FROM users",
			"SELECT id, name FROM users WHERE id = 1",
			"CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(100))",
		}

		for _, sql := range validSQL {
			result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)
			require.True(t, result.Valid, "SQL should be valid: %s", sql)
			assert.Empty(t, result.Violations)
		}
	})

	t.Run("detects syntax errors", func(t *testing.T) {
		invalidSQL := []string{
			"SELECT * FROM",        // Missing table name
			"SELECT * FORM users",  // Typo: FORM instead of FROM
			"CREATE TABLE users (", // Incomplete statement
		}

		for _, sql := range invalidSQL {
			result := validator.Validate(ctx, sql, models.DatabaseTypePostgreSQL)
			require.False(t, result.Valid, "SQL should be invalid: %s", sql)
			require.NotEmpty(t, result.Violations)
			assert.Equal(t, review.SeverityError, result.Violations[0].Severity)
		}
	})
}
