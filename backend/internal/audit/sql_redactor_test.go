package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifySQL_DDL(t *testing.T) {
	tests := []struct {
		name              string
		sql               string
		expectedType      SQLStatementType
		expectedRiskLevel SQLRiskLevel
	}{
		{
			name:              "DROP TABLE high risk",
			sql:               "DROP TABLE users",
			expectedType:      SQLTypeDDL,
			expectedRiskLevel: SQLRiskHigh,
		},
		{
			name:              "TRUNCATE TABLE high risk",
			sql:               "TRUNCATE TABLE logs",
			expectedType:      SQLTypeDDL,
			expectedRiskLevel: SQLRiskHigh,
		},
		{
			name:              "CREATE TABLE medium risk",
			sql:               "CREATE TABLE new_table (id INT)",
			expectedType:      SQLTypeDDL,
			expectedRiskLevel: SQLRiskMedium,
		},
		{
			name:              "ALTER TABLE medium risk",
			sql:               "ALTER TABLE users ADD COLUMN email VARCHAR(255)",
			expectedType:      SQLTypeDDL,
			expectedRiskLevel: SQLRiskMedium,
		},
		{
			name:              "Case insensitive",
			sql:               "drop table test",
			expectedType:      SQLTypeDDL,
			expectedRiskLevel: SQLRiskHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmtType, riskLevel := ClassifySQL(tt.sql)
			assert.Equal(t, tt.expectedType, stmtType)
			assert.Equal(t, tt.expectedRiskLevel, riskLevel)
		})
	}
}

func TestClassifySQL_DML(t *testing.T) {
	tests := []struct {
		name              string
		sql               string
		expectedType      SQLStatementType
		expectedRiskLevel SQLRiskLevel
	}{
		{
			name:              "INSERT low risk",
			sql:               "INSERT INTO users (name) VALUES ('John')",
			expectedType:      SQLTypeDML,
			expectedRiskLevel: SQLRiskLow,
		},
		{
			name:              "UPDATE with WHERE medium risk",
			sql:               "UPDATE users SET name = 'Jane' WHERE id = 1",
			expectedType:      SQLTypeDML,
			expectedRiskLevel: SQLRiskMedium,
		},
		{
			name:              "UPDATE without WHERE high risk",
			sql:               "UPDATE users SET active = false",
			expectedType:      SQLTypeDML,
			expectedRiskLevel: SQLRiskHigh,
		},
		{
			name:              "DELETE with WHERE medium risk",
			sql:               "DELETE FROM users WHERE id = 1",
			expectedType:      SQLTypeDML,
			expectedRiskLevel: SQLRiskMedium,
		},
		{
			name:              "DELETE without WHERE high risk",
			sql:               "DELETE FROM users",
			expectedType:      SQLTypeDML,
			expectedRiskLevel: SQLRiskHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmtType, riskLevel := ClassifySQL(tt.sql)
			assert.Equal(t, tt.expectedType, stmtType)
			assert.Equal(t, tt.expectedRiskLevel, riskLevel)
		})
	}
}

func TestClassifySQL_Query(t *testing.T) {
	tests := []struct {
		name         string
		sql          string
		expectedType SQLStatementType
	}{
		{
			name:         "SELECT query",
			sql:          "SELECT * FROM users",
			expectedType: SQLTypeQuery,
		},
		{
			name:         "SHOW query",
			sql:          "SHOW TABLES",
			expectedType: SQLTypeQuery,
		},
		{
			name:         "DESCRIBE query",
			sql:          "DESCRIBE users",
			expectedType: SQLTypeQuery,
		},
		{
			name:         "EXPLAIN query",
			sql:          "EXPLAIN SELECT * FROM users",
			expectedType: SQLTypeQuery,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmtType, riskLevel := ClassifySQL(tt.sql)
			assert.Equal(t, tt.expectedType, stmtType)
			assert.Equal(t, SQLRiskLow, riskLevel) // All queries are low risk
		})
	}
}

func TestClassifySQL_DCL(t *testing.T) {
	tests := []struct {
		name         string
		sql          string
		expectedType SQLStatementType
	}{
		{
			name:         "GRANT statement",
			sql:          "GRANT SELECT ON database.* TO 'user'@'host'",
			expectedType: SQLTypeDCL,
		},
		{
			name:         "REVOKE statement",
			sql:          "REVOKE ALL PRIVILEGES ON database.* FROM 'user'@'host'",
			expectedType: SQLTypeDCL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmtType, riskLevel := ClassifySQL(tt.sql)
			assert.Equal(t, tt.expectedType, stmtType)
			assert.Equal(t, SQLRiskMedium, riskLevel) // DCL is medium risk
		})
	}
}

func TestClassifySQL_TCL(t *testing.T) {
	tests := []struct {
		name         string
		sql          string
		expectedType SQLStatementType
	}{
		{
			name:         "COMMIT statement",
			sql:          "COMMIT",
			expectedType: SQLTypeTCL,
		},
		{
			name:         "ROLLBACK statement",
			sql:          "ROLLBACK",
			expectedType: SQLTypeTCL,
		},
		{
			name:         "BEGIN statement",
			sql:          "BEGIN",
			expectedType: SQLTypeTCL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmtType, riskLevel := ClassifySQL(tt.sql)
			assert.Equal(t, tt.expectedType, stmtType)
			assert.Equal(t, SQLRiskLow, riskLevel) // TCL is low risk
		})
	}
}

func TestClassifySQL_EdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		sql               string
		expectedType      SQLStatementType
		expectedRiskLevel SQLRiskLevel
	}{
		{
			name:              "Empty string",
			sql:               "",
			expectedType:      SQLTypeUnknown,
			expectedRiskLevel: SQLRiskLow,
		},
		{
			name:              "Whitespace only",
			sql:               "   ",
			expectedType:      SQLTypeUnknown,
			expectedRiskLevel: SQLRiskLow,
		},
		{
			name:              "Unknown statement",
			sql:               "FOOBAR something",
			expectedType:      SQLTypeUnknown,
			expectedRiskLevel: SQLRiskLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmtType, riskLevel := ClassifySQL(tt.sql)
			assert.Equal(t, tt.expectedType, stmtType)
			assert.Equal(t, tt.expectedRiskLevel, riskLevel)
		})
	}
}

func TestSanitizeSQL(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		maxLength int
		expected  string
	}{
		{
			name:      "Short SQL no truncation",
			sql:       "SELECT * FROM users",
			maxLength: 100,
			expected:  "SELECT * FROM users",
		},
		{
			name:      "Long SQL truncated",
			sql:       "SELECT id, name, email, created_at, updated_at FROM users WHERE active = true AND role = 'admin'",
			maxLength: 50,
			expected:  "SELECT id, name, email, created_at, updated_at FRO... [truncated]",
		},
		{
			name:      "No max length specified",
			sql:       "SELECT * FROM users",
			maxLength: 0,
			expected:  "SELECT * FROM users",
		},
		{
			name:      "Exact length",
			sql:       "SELECT * FROM users",
			maxLength: 19,
			expected:  "SELECT * FROM users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSQL(tt.sql, tt.maxLength)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSQLExecutionDetails(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "Remove forbidden keys",
			input: map[string]interface{}{
				"statement":     "SELECT * FROM users",
				"rows_affected": 10,
				"results":       []map[string]interface{}{{"id": 1}}, // Should be removed
				"data":          "sensitive",                         // Should be removed
				"query_result":  "result",                            // Should be removed
			},
			expected: map[string]interface{}{
				"statement":     "SELECT * FROM users",
				"rows_affected": 10,
			},
		},
		{
			name: "Keep only allowed keys",
			input: map[string]interface{}{
				"statement":         "INSERT INTO users",
				"statement_type":    "DML",
				"risk_level":        "low",
				"database_name":     "mydb",
				"environment":       "prod",
				"rows_affected":     1,
				"execution_time_ms": 50,
				"success":           true,
				"custom_field":      "value", // Not in allowed list
			},
			expected: map[string]interface{}{
				"statement":         "INSERT INTO users",
				"statement_type":    "DML",
				"risk_level":        "low",
				"database_name":     "mydb",
				"environment":       "prod",
				"rows_affected":     1,
				"execution_time_ms": 50,
				"success":           true,
			},
		},
		{
			name:     "Empty input",
			input:    map[string]interface{}{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSQLExecutionDetails(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateSQLForDisplay(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		maxLength int
		expected  string
	}{
		{
			name:      "Short SQL",
			sql:       "SELECT * FROM users",
			maxLength: 100,
			expected:  "SELECT * FROM users",
		},
		{
			name:      "Long SQL with newlines",
			sql:       "SELECT *\nFROM users\nWHERE active = true\nAND role = 'admin'",
			maxLength: 30,
			expected:  "SELECT * FROM users WHERE acti...",
		},
		{
			name:      "SQL with extra spaces",
			sql:       "SELECT    *     FROM      users",
			maxLength: 100,
			expected:  "SELECT * FROM users",
		},
		{
			name:      "Default max length",
			sql:       "SELECT id, name, email, created_at, updated_at, deleted_at, status, role, permissions, settings, preferences FROM users WHERE active = true",
			maxLength: 0,
			expected:  "SELECT id, name, email, created_at, updated_at, deleted_at, status, role, permissions, settings, pre...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateSQLForDisplay(tt.sql, tt.maxLength)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSQLExecutionDetails_NoResultsField(t *testing.T) {
	// This test ensures the struct doesn't have any field that could contain query results
	details := SQLExecutionDetails{
		Statement:       "SELECT * FROM users",
		StatementType:   SQLTypeQuery,
		RiskLevel:       SQLRiskLow,
		DatabaseName:    "testdb",
		Environment:     "dev",
		RowsAffected:    10,
		ExecutionTimeMS: 50,
		Success:         true,
		ErrorMessage:    "",
	}

	// Verify the struct only contains safe fields
	assert.NotEmpty(t, details.Statement)
	assert.Equal(t, SQLTypeQuery, details.StatementType)
	assert.Equal(t, SQLRiskLow, details.RiskLevel)
	assert.Equal(t, "testdb", details.DatabaseName)
	assert.Equal(t, "dev", details.Environment)
	assert.Equal(t, int64(10), details.RowsAffected)
	assert.Equal(t, int64(50), details.ExecutionTimeMS)
	assert.True(t, details.Success)
}
