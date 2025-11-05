// Package database provides database connection management for target databases (MySQL and PostgreSQL).
package database

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// ConnectionError represents a structured error from database connection attempts
type ConnectionError struct {
	// Database information
	DatabaseType string
	DatabaseName string
	DatabaseID   string
	InstanceName string
	Host         string
	Port         int
	Username     string

	// Connection details
	SSLRequired       bool
	ConnectionTimeout time.Duration

	// Error classification
	ErrorType string // "network", "authentication", "timeout", "database_not_found", "permission", "other"
	ErrorCode string // Database-specific error code if available

	// Standard error information
	Message    string
	Underlying error

	// Stack trace
	StackTrace string

	// Retry information (if retries were attempted)
	RetryAttempts int
	RetryDelay    time.Duration
}

// Error implements the error interface
func (e *ConnectionError) Error() string {
	return e.Message
}

// Unwrap returns the underlying error
func (e *ConnectionError) Unwrap() error {
	return e.Underlying
}

// captureStackTrace captures the current stack trace
func captureStackTrace(skip int) string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	lines := strings.Split(string(buf[:n]), "\n")
	// Skip the first line (goroutine info) and the caller's line
	if len(lines) > skip*2+1 {
		return strings.Join(lines[skip*2+1:], "\n")
	}
	return string(buf[:n])
}

// NewConnectionError creates a new ConnectionError from a connection error
func NewConnectionError(
	dbType string,
	dbName string,
	dbID string,
	instanceName string,
	host string,
	port int,
	username string,
	sslRequired bool,
	timeout time.Duration,
	err error,
) *ConnectionError {
	connErr := &ConnectionError{
		DatabaseType:      dbType,
		DatabaseName:      dbName,
		DatabaseID:        dbID,
		InstanceName:      instanceName,
		Host:              host,
		Port:              port,
		Username:          username,
		SSLRequired:       sslRequired,
		ConnectionTimeout: timeout,
		Underlying:        err,
		StackTrace:        captureStackTrace(3), // Skip NewConnectionError, caller, and their caller
	}

	// Classify error type and extract details
	connErr.classifyError(err)

	// Generate user-friendly message
	connErr.Message = connErr.generateUserMessage()

	return connErr
}

// classifyError classifies the error type and extracts error codes
func (e *ConnectionError) classifyError(err error) {
	if err == nil {
		e.ErrorType = "other"
		return
	}

	// Check for network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			e.ErrorType = "timeout"
			e.Message = fmt.Sprintf("Connection timeout: %v", netErr)
		} else {
			e.ErrorType = "network"
		}
		return
	}

	// Check for context timeout/cancel
	if errors.Is(err, context.DeadlineExceeded) {
		e.ErrorType = "timeout"
		return
	}

	// Check for PostgreSQL errors
	if pqErr, ok := err.(*pq.Error); ok {
		e.ErrorCode = string(pqErr.Code)
		e.ErrorType = classifyPostgreSQLError(pqErr.Code)
		return
	}

	// Check for MySQL errors
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		e.ErrorCode = fmt.Sprintf("%d", mysqlErr.Number)
		e.ErrorType = classifyMySQLError(mysqlErr.Number)
		return
	}

	// Check error message for common patterns
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no route to host") ||
		strings.Contains(errMsg, "network is unreachable") {
		e.ErrorType = "network"
	} else if strings.Contains(errMsg, "password authentication failed") ||
		strings.Contains(errMsg, "access denied") ||
		strings.Contains(errMsg, "invalid password") {
		e.ErrorType = "authentication"
	} else if strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "timed out") {
		e.ErrorType = "timeout"
	} else if strings.Contains(errMsg, "database") && strings.Contains(errMsg, "does not exist") ||
		strings.Contains(errMsg, "unknown database") {
		e.ErrorType = "database_not_found"
	} else if strings.Contains(errMsg, "permission denied") ||
		strings.Contains(errMsg, "access denied") {
		e.ErrorType = "permission"
	} else {
		e.ErrorType = "other"
	}
}

// classifyPostgreSQLError classifies PostgreSQL error codes
func classifyPostgreSQLError(code pq.ErrorCode) string {
	switch code {
	case "28P01": // invalid_password
		return "authentication"
	case "3D000": // invalid_catalog_name
		return "database_not_found"
	case "08006": // connection_failure
		return "network"
	case "08001": // sqlclient_unable_to_establish_sqlconnection
		return "network"
	case "08003": // connection_does_not_exist
		return "network"
	case "08004": // sqlserver_rejected_establishment_of_sqlconnection
		return "network"
	case "08007": // connection_failure
		return "network"
	case "53300": // too_many_connections
		return "other"
	default:
		return "other"
	}
}

// classifyMySQLError classifies MySQL error numbers
func classifyMySQLError(number uint16) string {
	switch number {
	case 1045: // ER_ACCESS_DENIED_ERROR
		return "authentication"
	case 1049: // ER_BAD_DB_ERROR
		return "database_not_found"
	case 2002: // CR_CONN_HOST_ERROR
		return "network"
	case 2003: // CR_CONNECTION_ERROR
		return "network"
	case 2006: // CR_SERVER_GONE_ERROR
		return "network"
	case 2013: // CR_SERVER_LOST
		return "network"
	case 1040: // ER_CON_COUNT_ERROR
		return "other"
	default:
		return "other"
	}
}

// generateUserMessage generates a user-friendly error message
func (e *ConnectionError) generateUserMessage() string {
	var msg strings.Builder

	// Base message
	switch e.ErrorType {
	case "network":
		msg.WriteString("Database connection failed: Unable to reach database server")
	case "authentication":
		msg.WriteString("Database connection failed: Authentication failed")
	case "timeout":
		msg.WriteString("Database connection failed: Connection timeout")
	case "database_not_found":
		msg.WriteString("Database connection failed: Database does not exist")
	case "permission":
		msg.WriteString("Database connection failed: Permission denied")
	default:
		msg.WriteString("Database connection failed")
	}

	// Add actionable details
	msg.WriteString(fmt.Sprintf(" (Host: %s:%d, Database: %s)", e.Host, e.Port, e.DatabaseName))

	// Add helpful hints based on error type
	switch e.ErrorType {
	case "network":
		msg.WriteString(". Please verify:")
		msg.WriteString(" - Database server is running")
		msg.WriteString(" - Host and port are correct")
		msg.WriteString(" - Network connectivity is available")
		msg.WriteString(" - Firewall rules allow connection")
	case "authentication":
		msg.WriteString(". Please verify:")
		msg.WriteString(" - Username and password are correct")
		msg.WriteString(" - User has permission to access the database")
	case "timeout":
		msg.WriteString(". Please verify:")
		msg.WriteString(" - Database server is responding")
		msg.WriteString(" - Network latency is acceptable")
		msg.WriteString(" - Connection timeout settings are appropriate")
	case "database_not_found":
		msg.WriteString(". Please verify:")
		msg.WriteString(" - Database name is correct")
		msg.WriteString(" - Database has been created")
	case "permission":
		msg.WriteString(". Please verify:")
		msg.WriteString(" - User has required permissions")
		msg.WriteString(" - Database exists and is accessible")
	}

	// Add retry information if applicable
	if e.RetryAttempts > 0 {
		msg.WriteString(fmt.Sprintf(" (Retried %d time(s))", e.RetryAttempts))
	}

	return msg.String()
}

// RetryConfig represents configuration for connection retry logic
type RetryConfig struct {
	MaxAttempts       int           // Maximum number of retry attempts (default: 3)
	InitialDelay      time.Duration // Initial delay before first retry (default: 100ms)
	MaxDelay          time.Duration // Maximum delay between retries (default: 5s)
	BackoffMultiplier float64       // Multiplier for exponential backoff (default: 2.0)
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          5 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// connectWithRetry attempts to connect with retry logic
// This function is available for future use when retry logic is needed.
// It implements exponential backoff and skips retries for authentication/database_not_found errors.
// For now, it's not used by default connection methods but can be called explicitly when needed.
//
//nolint:unused // Available for future use when retry logic is needed
func (cm *ConnectionManager) connectWithRetry(
	ctx context.Context,
	connectFunc func(context.Context) (*sqlx.DB, error),
	retryConfig RetryConfig,
	dbType string,
	dbName string,
	dbID string,
	instanceName string,
	host string,
	port int,
	username string,
	sslRequired bool,
	timeout time.Duration,
) (*sqlx.DB, error) {
	var lastErr error
	delay := retryConfig.InitialDelay

	for attempt := 0; attempt <= retryConfig.MaxAttempts; attempt++ {
		if attempt > 0 {
			cm.logger.Info("Retrying database connection",
				zap.String("database_type", dbType),
				zap.String("database_name", dbName),
				zap.String("host", host),
				zap.Int("port", port),
				zap.Int("attempt", attempt),
				zap.Int("max_attempts", retryConfig.MaxAttempts),
				zap.Duration("delay", delay),
			)

			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}

			// Exponential backoff
			delay = time.Duration(float64(delay) * retryConfig.BackoffMultiplier)
			if delay > retryConfig.MaxDelay {
				delay = retryConfig.MaxDelay
			}
		}

		db, err := connectFunc(ctx)
		if err == nil {
			if attempt > 0 {
				cm.logger.Info("Database connection succeeded after retry",
					zap.String("database_type", dbType),
					zap.String("database_name", dbName),
					zap.Int("attempts", attempt+1),
				)
			}
			return db, nil
		}

		lastErr = err

		// Don't retry on authentication errors or database not found errors
		connErr := NewConnectionError(dbType, dbName, dbID, instanceName, host, port, username, sslRequired, timeout, err)
		if connErr.ErrorType == "authentication" || connErr.ErrorType == "database_not_found" {
			connErr.RetryAttempts = attempt
			return nil, connErr
		}
	}

	// All retries exhausted
	connErr := NewConnectionError(dbType, dbName, dbID, instanceName, host, port, username, sslRequired, timeout, lastErr)
	connErr.RetryAttempts = retryConfig.MaxAttempts
	connErr.RetryDelay = delay
	return nil, connErr
}
