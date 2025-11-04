package services

import (
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestNewSQLExecutionError_PostgreSQL(t *testing.T) {
	// Create a PostgreSQL error
	pqErr := &pq.Error{
		Code:    "23505",
		Message: "duplicate key value violates unique constraint",
		Detail:  "Key (id)=(123) already exists.",
		Hint:    "Try updating the existing row instead.",
		Position: "100",
	}

	sqlContent := "INSERT INTO users (id, name) VALUES (123, 'John');"
	dbType := "postgresql"
	dbName := "testdb"
	dbID := "db-123"
	instanceName := "test-instance"

	// Create SQLExecutionError
	sqlErr := NewSQLExecutionError(dbType, dbName, dbID, instanceName, sqlContent, pqErr)

	// Verify database-specific fields
	assert.Equal(t, dbType, sqlErr.DatabaseType)
	assert.Equal(t, dbName, sqlErr.DatabaseName)
	assert.Equal(t, dbID, sqlErr.DatabaseID)
	assert.Equal(t, instanceName, sqlErr.InstanceName)
	assert.Equal(t, sqlContent, sqlErr.SQLContent)

	// Verify PostgreSQL-specific error details
	assert.Equal(t, "23505", sqlErr.DatabaseErrorCode)
	assert.Equal(t, "duplicate key value violates unique constraint", sqlErr.DatabaseMessage)
	assert.Equal(t, "Key (id)=(123) already exists.", sqlErr.DatabaseDetail)
	assert.Equal(t, "Try updating the existing row instead.", sqlErr.DatabaseHint)
	assert.Equal(t, "100", sqlErr.DatabasePosition)

	// Verify message contains helpful information
	assert.Contains(t, sqlErr.Message, "SQL execution failed")
	assert.Contains(t, sqlErr.Message, "Unique constraint violation")
	assert.Contains(t, sqlErr.Message, "Key (id)=(123) already exists.")
	assert.Contains(t, sqlErr.Message, "Try updating the existing row instead.")

	// Verify stack trace is captured
	assert.NotEmpty(t, sqlErr.StackTrace)

	// Verify underlying error
	assert.Equal(t, pqErr, sqlErr.Underlying)
}

func TestNewSQLExecutionError_MySQL(t *testing.T) {
	// Create a MySQL error
	mysqlErr := &mysql.MySQLError{
		Number:  1062,
		Message: "Duplicate entry '123' for key 'PRIMARY'",
	}

	sqlContent := "INSERT INTO users (id, name) VALUES (123, 'John');"
	dbType := "mysql"
	dbName := "testdb"
	dbID := "db-123"
	instanceName := "test-instance"

	// Create SQLExecutionError
	sqlErr := NewSQLExecutionError(dbType, dbName, dbID, instanceName, sqlContent, mysqlErr)

	// Verify database-specific fields
	assert.Equal(t, dbType, sqlErr.DatabaseType)
	assert.Equal(t, dbName, sqlErr.DatabaseName)
	assert.Equal(t, dbID, sqlErr.DatabaseID)
	assert.Equal(t, instanceName, sqlErr.InstanceName)
	assert.Equal(t, sqlContent, sqlErr.SQLContent)

	// Verify MySQL-specific error details
	assert.Equal(t, "1062", sqlErr.DatabaseErrorCode)
	assert.Equal(t, "Duplicate entry '123' for key 'PRIMARY'", sqlErr.DatabaseMessage)

	// Verify message contains helpful information
	assert.Contains(t, sqlErr.Message, "SQL execution failed")
	assert.Contains(t, sqlErr.Message, "Duplicate entry")

	// Verify stack trace is captured
	assert.NotEmpty(t, sqlErr.StackTrace)

	// Verify underlying error
	assert.Equal(t, mysqlErr, sqlErr.Underlying)
}

func TestNewSQLExecutionError_GenericError(t *testing.T) {
	// Create a generic error
	genericErr := errors.New("connection timeout")

	sqlContent := "SELECT * FROM users;"
	dbType := "postgresql"
	dbName := "testdb"
	dbID := "db-123"
	instanceName := "test-instance"

	// Create SQLExecutionError
	sqlErr := NewSQLExecutionError(dbType, dbName, dbID, instanceName, sqlContent, genericErr)

	// Verify database-specific fields
	assert.Equal(t, dbType, sqlErr.DatabaseType)
	assert.Equal(t, dbName, sqlErr.DatabaseName)
	assert.Equal(t, dbID, sqlErr.DatabaseID)
	assert.Equal(t, instanceName, sqlErr.InstanceName)
	assert.Equal(t, sqlContent, sqlErr.SQLContent)

	// Verify generic error handling
	assert.Empty(t, sqlErr.DatabaseErrorCode)
	assert.Equal(t, "connection timeout", sqlErr.DatabaseMessage)

	// Verify message
	assert.Contains(t, sqlErr.Message, "SQL execution failed")
	assert.Contains(t, sqlErr.Message, "connection timeout")

	// Verify stack trace is captured
	assert.NotEmpty(t, sqlErr.StackTrace)

	// Verify underlying error
	assert.Equal(t, genericErr, sqlErr.Underlying)
}

func TestSQLExecutionError_ErrorInterface(t *testing.T) {
	pqErr := &pq.Error{
		Code:    "42601",
		Message: "syntax error at or near \"INVALID\"",
		Position: "50",
	}

	sqlErr := NewSQLExecutionError("postgresql", "testdb", "db-123", "test-instance", "SELECT * FROM INVALID;", pqErr)

	// Test Error() method
	errorMsg := sqlErr.Error()
	assert.NotEmpty(t, errorMsg)
	assert.Contains(t, errorMsg, "SQL execution failed")
}

func TestSQLExecutionError_Unwrap(t *testing.T) {
	pqErr := &pq.Error{
		Code:    "23505",
		Message: "duplicate key",
	}

	sqlErr := NewSQLExecutionError("postgresql", "testdb", "db-123", "test-instance", "INSERT...", pqErr)

	// Test Unwrap() method
	unwrapped := sqlErr.Unwrap()
	assert.Equal(t, pqErr, unwrapped)
}

func TestGenerateUserMessage_PostgreSQLErrorCodes(t *testing.T) {
	testCases := []struct {
		name           string
		code           string
		message        string
		detail         string
		hint           string
		position       string
		expectedPhrase string
	}{
		{
			name:           "unique_violation",
			code:           "23505",
			message:        "duplicate key",
			detail:         "Key (id)=(123) already exists.",
			expectedPhrase: "Unique constraint violation",
		},
		{
			name:           "foreign_key_violation",
			code:           "23503",
			message:        "foreign key violation",
			detail:         "Key (user_id)=(999) is not present in table users.",
			expectedPhrase: "Foreign key constraint violation",
		},
		{
			name:           "not_null_violation",
			code:           "23502",
			message:        "null value",
			detail:         "null value in column \"email\" violates not-null constraint",
			expectedPhrase: "Not null constraint violation",
		},
		{
			name:           "syntax_error",
			code:           "42601",
			message:        "syntax error",
			position:       "50",
			expectedPhrase: "SQL syntax error",
		},
		{
			name:           "undefined_table",
			code:           "42P01",
			message:        "relation \"users\" does not exist",
			expectedPhrase: "Table does not exist",
		},
		{
			name:           "undefined_column",
			code:           "42703",
			message:        "column \"email\" does not exist",
			expectedPhrase: "Column does not exist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pqErr := &pq.Error{
				Code:     pq.ErrorCode(tc.code),
				Message:  tc.message,
				Detail:   tc.detail,
				Hint:     tc.hint,
				Position: tc.position,
			}

			sqlErr := NewSQLExecutionError("postgresql", "testdb", "db-123", "test-instance", "SELECT...", pqErr)
			assert.Contains(t, sqlErr.Message, tc.expectedPhrase)
		})
	}
}

func TestGenerateUserMessage_MySQLErrorCodes(t *testing.T) {
	testCases := []struct {
		name           string
		number         uint16
		message        string
		expectedPhrase string
	}{
		{
			name:           "ER_DUP_ENTRY",
			number:         1062,
			message:        "Duplicate entry '123' for key 'PRIMARY'",
			expectedPhrase: "Duplicate entry",
		},
		{
			name:           "ER_NO_REFERENCED_ROW_2",
			number:         1452,
			message:        "Cannot add or update a child row: a foreign key constraint fails",
			expectedPhrase: "Foreign key constraint violation",
		},
		{
			name:           "ER_BAD_NULL_ERROR",
			number:         1048,
			message:        "Column 'email' cannot be null",
			expectedPhrase: "Column cannot be null",
		},
		{
			name:           "ER_NO_SUCH_TABLE",
			number:         1146,
			message:        "Table 'testdb.users' doesn't exist",
			expectedPhrase: "Table does not exist",
		},
		{
			name:           "ER_PARSE_ERROR",
			number:         1064,
			message:        "You have an error in your SQL syntax",
			expectedPhrase: "SQL syntax error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mysqlErr := &mysql.MySQLError{
				Number:  tc.number,
				Message: tc.message,
			}

			sqlErr := NewSQLExecutionError("mysql", "testdb", "db-123", "test-instance", "SELECT...", mysqlErr)
			assert.Contains(t, sqlErr.Message, tc.expectedPhrase)
		})
	}
}

func TestMaskSQLContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // We'll check that sensitive info is masked
	}{
		{
			name:     "password in PASSWORD clause",
			input:    "CREATE USER test PASSWORD='secret123';",
			expected: "password", // Should contain masked password
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no sensitive data",
			input:    "SELECT * FROM users WHERE id = 1;",
			expected: "SELECT * FROM users WHERE id = 1;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSQLContent(tt.input)
			if tt.input == "" {
				assert.Equal(t, "", result)
			} else if tt.name == "no sensitive data" {
				assert.Equal(t, tt.input, result)
			} else {
				// For sensitive data, just verify it doesn't contain the original sensitive value
				assert.NotContains(t, result, "secret123")
			}
		})
	}
}

func TestMaskSQLContent_PasswordPatterns(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		shouldBeMasked bool
	}{
		{
			name:           "PASSWORD='value'",
			input:          "CREATE USER test PASSWORD='secret123';",
			shouldBeMasked: true,
		},
		{
			name:           "IDENTIFIED BY 'value'",
			input:          "CREATE USER test IDENTIFIED BY 'secret123';",
			shouldBeMasked: true,
		},
		{
			name:           "normal SQL",
			input:          "SELECT * FROM users WHERE id = 1;",
			shouldBeMasked: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := MaskSQLContent(tc.input)
			if tc.shouldBeMasked {
				// Verify that sensitive values are masked
				assert.NotContains(t, result, "secret123")
				assert.Contains(t, result, "***")
			} else {
				// Verify that normal SQL is unchanged
				assert.Equal(t, tc.input, result)
			}
		})
	}
}

