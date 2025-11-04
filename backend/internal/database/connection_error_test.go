package database

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestNewConnectionError_NetworkError(t *testing.T) {
	netErr := &net.OpError{
		Op:  "dial",
		Err: errors.New("connection refused"),
	}

	connErr := NewConnectionError(
		"postgresql",
		"testdb",
		"db-123",
		"test-instance",
		"localhost",
		5432,
		"testuser",
		false,
		10*time.Second,
		netErr,
	)

	// Verify error classification
	assert.Equal(t, "network", connErr.ErrorType)
	assert.Equal(t, "postgresql", connErr.DatabaseType)
	assert.Equal(t, "testdb", connErr.DatabaseName)
	assert.Equal(t, "localhost", connErr.Host)
	assert.Equal(t, 5432, connErr.Port)

	// Verify message contains helpful information
	assert.Contains(t, connErr.Message, "Database connection failed")
	assert.Contains(t, connErr.Message, "Unable to reach database server")
	assert.Contains(t, connErr.Message, "localhost:5432")
	assert.Contains(t, connErr.Message, "Database server is running")

	// Verify stack trace is captured
	assert.NotEmpty(t, connErr.StackTrace)

	// Verify underlying error
	assert.Equal(t, netErr, connErr.Underlying)
}

func TestNewConnectionError_TimeoutError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	<-ctx.Done()

	connErr := NewConnectionError(
		"mysql",
		"testdb",
		"db-123",
		"test-instance",
		"localhost",
		3306,
		"testuser",
		false,
		5*time.Second,
		ctx.Err(),
	)

	// Verify error classification
	assert.Equal(t, "timeout", connErr.ErrorType)

	// Verify message contains helpful information
	assert.Contains(t, connErr.Message, "Connection timeout")
	assert.Contains(t, connErr.Message, "Database server is responding")
}

func TestNewConnectionError_PostgreSQLError(t *testing.T) {
	pqErr := &pq.Error{
		Code:    "28P01",
		Message: "password authentication failed for user",
	}

	connErr := NewConnectionError(
		"postgresql",
		"testdb",
		"db-123",
		"test-instance",
		"localhost",
		5432,
		"testuser",
		false,
		10*time.Second,
		pqErr,
	)

	// Verify error classification
	assert.Equal(t, "authentication", connErr.ErrorType)
	assert.Equal(t, "28P01", connErr.ErrorCode)

	// Verify message contains helpful information
	assert.Contains(t, connErr.Message, "Authentication failed")
	assert.Contains(t, connErr.Message, "Username and password are correct")
}

func TestNewConnectionError_MySQLError(t *testing.T) {
	mysqlErr := &mysql.MySQLError{
		Number:  1045,
		Message: "Access denied for user",
	}

	connErr := NewConnectionError(
		"mysql",
		"testdb",
		"db-123",
		"test-instance",
		"localhost",
		3306,
		"testuser",
		false,
		10*time.Second,
		mysqlErr,
	)

	// Verify error classification
	assert.Equal(t, "authentication", connErr.ErrorType)
	assert.Equal(t, "1045", connErr.ErrorCode)

	// Verify message contains helpful information
	assert.Contains(t, connErr.Message, "Authentication failed")
}

func TestNewConnectionError_DatabaseNotFound(t *testing.T) {
	pqErr := &pq.Error{
		Code:    "3D000",
		Message: "database \"nonexistent\" does not exist",
	}

	connErr := NewConnectionError(
		"postgresql",
		"nonexistent",
		"db-123",
		"test-instance",
		"localhost",
		5432,
		"testuser",
		false,
		10*time.Second,
		pqErr,
	)

	// Verify error classification
	assert.Equal(t, "database_not_found", connErr.ErrorType)
	assert.Equal(t, "3D000", connErr.ErrorCode)

	// Verify message contains helpful information
	assert.Contains(t, connErr.Message, "Database does not exist")
	assert.Contains(t, connErr.Message, "Database name is correct")
}

func TestNewConnectionError_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	<-ctx.Done()

	connErr := NewConnectionError(
		"postgresql",
		"testdb",
		"db-123",
		"test-instance",
		"localhost",
		5432,
		"testuser",
		false,
		1*time.Nanosecond,
		ctx.Err(),
	)

	// Verify error classification
	assert.Equal(t, "timeout", connErr.ErrorType)
}

func TestNewConnectionError_GenericError(t *testing.T) {
	genericErr := errors.New("unexpected connection error")

	connErr := NewConnectionError(
		"postgresql",
		"testdb",
		"db-123",
		"test-instance",
		"localhost",
		5432,
		"testuser",
		false,
		10*time.Second,
		genericErr,
	)

	// Verify error classification
	assert.Equal(t, "other", connErr.ErrorType)

	// Verify message
	assert.Contains(t, connErr.Message, "Database connection failed")
}

func TestConnectionError_ErrorInterface(t *testing.T) {
	err := errors.New("connection error")
	connErr := NewConnectionError("postgresql", "testdb", "db-123", "test-instance", "localhost", 5432, "testuser", false, 10*time.Second, err)

	// Test Error() method
	errorMsg := connErr.Error()
	assert.NotEmpty(t, errorMsg)
	assert.Contains(t, errorMsg, "Database connection failed")
}

func TestConnectionError_Unwrap(t *testing.T) {
	err := errors.New("connection error")
	connErr := NewConnectionError("postgresql", "testdb", "db-123", "test-instance", "localhost", 5432, "testuser", false, 10*time.Second, err)

	// Test Unwrap() method
	unwrapped := connErr.Unwrap()
	assert.Equal(t, err, unwrapped)
}

func TestClassifyPostgreSQLError(t *testing.T) {
	testCases := []struct {
		name       string
		code       pq.ErrorCode
		expectedType string
	}{
		{"invalid_password", "28P01", "authentication"},
		{"invalid_catalog_name", "3D000", "database_not_found"},
		{"connection_failure", "08006", "network"},
		{"unable_to_establish", "08001", "network"},
		{"connection_does_not_exist", "08003", "network"},
		{"too_many_connections", "53300", "other"},
		{"unknown_code", "99999", "other"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := classifyPostgreSQLError(tc.code)
			assert.Equal(t, tc.expectedType, result)
		})
	}
}

func TestClassifyMySQLError(t *testing.T) {
	testCases := []struct {
		name       string
		number     uint16
		expectedType string
	}{
		{"ER_ACCESS_DENIED_ERROR", 1045, "authentication"},
		{"ER_BAD_DB_ERROR", 1049, "database_not_found"},
		{"CR_CONN_HOST_ERROR", 2002, "network"},
		{"CR_CONNECTION_ERROR", 2003, "network"},
		{"CR_SERVER_GONE_ERROR", 2006, "network"},
		{"CR_SERVER_LOST", 2013, "network"},
		{"ER_CON_COUNT_ERROR", 1040, "other"},
		{"unknown_error", 9999, "other"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := classifyMySQLError(tc.number)
			assert.Equal(t, tc.expectedType, result)
		})
	}
}

func TestGenerateUserMessage_ErrorTypes(t *testing.T) {
	testCases := []struct {
		name           string
		errorType      string
		expectedPhrase string
	}{
		{"network", "network", "Unable to reach database server"},
		{"authentication", "authentication", "Authentication failed"},
		{"timeout", "timeout", "Connection timeout"},
		{"database_not_found", "database_not_found", "Database does not exist"},
		{"permission", "permission", "Permission denied"},
		{"other", "other", "Database connection failed"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			connErr := &ConnectionError{
				ErrorType:   tc.errorType,
				Host:        "localhost",
				Port:        5432,
				DatabaseName: "testdb",
			}
			msg := connErr.generateUserMessage()
			assert.Contains(t, msg, tc.expectedPhrase)
		})
	}
}

func TestGenerateUserMessage_WithRetryAttempts(t *testing.T) {
	connErr := &ConnectionError{
		ErrorType:     "network",
		Host:          "localhost",
		Port:          5432,
		DatabaseName:  "testdb",
		RetryAttempts: 3,
	}

	msg := connErr.generateUserMessage()
	assert.Contains(t, msg, "Retried 3 time(s)")
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, config.InitialDelay)
	assert.Equal(t, 5*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.BackoffMultiplier)
}

func TestConnectionError_ErrorTypeClassification(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		expectedType string
	}{
		{
			name:        "connection refused",
			err:         &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			expectedType: "network",
		},
		{
			name:        "password auth failed",
			err:         errors.New("password authentication failed for user"),
			expectedType: "authentication",
		},
		{
			name:        "timeout",
			err:         errors.New("connection timeout"),
			expectedType: "timeout",
		},
		{
			name:        "database does not exist",
			err:         errors.New("database does not exist"),
			expectedType: "database_not_found",
		},
		{
			name:        "unknown database",
			err:         errors.New("unknown database"),
			expectedType: "database_not_found",
		},
		{
			name:        "permission denied",
			err:         errors.New("permission denied"),
			expectedType: "permission",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connErr := NewConnectionError(
				"postgresql",
				"testdb",
				"db-123",
				"test-instance",
				"localhost",
				5432,
				"testuser",
				false,
				10*time.Second,
				tt.err,
			)
			assert.Equal(t, tt.expectedType, connErr.ErrorType)
		})
	}
}

func TestConnectionError_RetryInformation(t *testing.T) {
	// Test that retry attempts are tracked
	connErr := &ConnectionError{
		ErrorType:     "network",
		RetryAttempts: 2,
		RetryDelay:    200 * time.Millisecond,
	}

	assert.Equal(t, 2, connErr.RetryAttempts)
	assert.Equal(t, 200*time.Millisecond, connErr.RetryDelay)

	// Verify retry info is included in message
	msg := connErr.generateUserMessage()
	assert.Contains(t, msg, "Retried 2 time(s)")
}

