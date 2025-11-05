// Package services provides business logic services for the Deeb application.
package services

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
)

// SQLExecutionError represents a structured error from SQL execution
type SQLExecutionError struct {
	// Database information
	DatabaseType string
	DatabaseName string
	DatabaseID   string
	InstanceName string

	// SQL content (may be masked)
	SQLContent       string
	SQLContentMasked bool

	// Database-specific error details
	DatabaseErrorCode string // PostgreSQL error code (e.g., "23505") or MySQL error number (e.g., "1062")
	DatabaseMessage   string // Database error message
	DatabaseDetail    string // Additional detail from database (e.g., PostgreSQL detail field)
	DatabaseHint      string // Database hint (e.g., PostgreSQL hint field)
	DatabasePosition  string // Character position in SQL where error occurred (PostgreSQL) - stored as string

	// Standard error information
	Message    string
	Underlying error

	// Stack trace
	StackTrace string
}

// Error implements the error interface
func (e *SQLExecutionError) Error() string {
	return e.Message
}

// Unwrap returns the underlying error
func (e *SQLExecutionError) Unwrap() error {
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

// NewSQLExecutionError creates a new SQLExecutionError from a database error
func NewSQLExecutionError(
	dbType string,
	dbName string,
	dbID string,
	instanceName string,
	sqlContent string,
	err error,
) *SQLExecutionError {
	sqlErr := &SQLExecutionError{
		DatabaseType: dbType,
		DatabaseName: dbName,
		DatabaseID:   dbID,
		InstanceName: instanceName,
		SQLContent:   sqlContent,
		Underlying:   err,
		StackTrace:   captureStackTrace(3), // Skip NewSQLExecutionError, caller, and their caller
	}

	// Extract database-specific error details
	sqlErr.extractDatabaseErrorDetails(err)

	// Generate user-friendly message
	sqlErr.Message = sqlErr.generateUserMessage()

	return sqlErr
}

// extractDatabaseErrorDetails extracts database-specific error details
func (e *SQLExecutionError) extractDatabaseErrorDetails(err error) {
	// Handle PostgreSQL errors
	if pqErr, ok := err.(*pq.Error); ok {
		e.DatabaseErrorCode = string(pqErr.Code)
		e.DatabaseMessage = pqErr.Message
		e.DatabaseDetail = pqErr.Detail
		e.DatabaseHint = pqErr.Hint
		e.DatabasePosition = pqErr.Position
		return
	}

	// Handle MySQL errors
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		e.DatabaseErrorCode = fmt.Sprintf("%d", mysqlErr.Number)
		e.DatabaseMessage = mysqlErr.Message
		return
	}

	// For generic errors, use the error message
	e.DatabaseMessage = err.Error()
}

// generateUserMessage generates a user-friendly error message
func (e *SQLExecutionError) generateUserMessage() string {
	// Build base message
	var msg strings.Builder
	msg.WriteString("SQL execution failed")

	// Add database-specific context
	if e.DatabaseErrorCode != "" {
		switch e.DatabaseType {
		case "postgresql":
			msg.WriteString(fmt.Sprintf(" (PostgreSQL error code: %s)", e.DatabaseErrorCode))
		case "mysql":
			msg.WriteString(fmt.Sprintf(" (MySQL error code: %s)", e.DatabaseErrorCode))
		}
	}

	// Add helpful context based on error codes
	if e.DatabaseErrorCode != "" {
		switch e.DatabaseErrorCode {
		// PostgreSQL error codes
		case "23505": // unique_violation
			msg.WriteString(": Unique constraint violation")
			if e.DatabaseDetail != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseDetail))
			}
		case "23503": // foreign_key_violation
			msg.WriteString(": Foreign key constraint violation")
			if e.DatabaseDetail != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseDetail))
			}
		case "23514": // check_violation
			msg.WriteString(": Check constraint violation")
			if e.DatabaseDetail != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseDetail))
			}
		case "23502": // not_null_violation
			msg.WriteString(": Not null constraint violation")
			if e.DatabaseDetail != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseDetail))
			}
		case "42P01": // undefined_table
			msg.WriteString(": Table does not exist")
			if e.DatabaseMessage != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseMessage))
			}
		case "42703": // undefined_column
			msg.WriteString(": Column does not exist")
			if e.DatabaseMessage != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseMessage))
			}
		case "42601": // syntax_error
			msg.WriteString(": SQL syntax error")
			if e.DatabasePosition != "" {
				msg.WriteString(fmt.Sprintf(" at position %s", e.DatabasePosition))
			}
			if e.DatabaseMessage != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseMessage))
			}
		case "28P01": // invalid_password
			msg.WriteString(": Invalid database credentials")
		case "3D000": // invalid_catalog_name
			msg.WriteString(": Database does not exist")
		case "42P04": // duplicate_database
			msg.WriteString(": Database already exists")
		// MySQL error codes
		case "1062": // ER_DUP_ENTRY
			msg.WriteString(": Duplicate entry")
			if e.DatabaseMessage != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseMessage))
			}
		case "1452": // ER_NO_REFERENCED_ROW_2
			msg.WriteString(": Foreign key constraint violation")
			if e.DatabaseMessage != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseMessage))
			}
		case "1048": // ER_BAD_NULL_ERROR
			msg.WriteString(": Column cannot be null")
			if e.DatabaseMessage != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseMessage))
			}
		case "1146": // ER_NO_SUCH_TABLE
			msg.WriteString(": Table does not exist")
			if e.DatabaseMessage != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseMessage))
			}
		case "1054": // ER_BAD_FIELD_ERROR
			msg.WriteString(": Unknown column")
			if e.DatabaseMessage != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseMessage))
			}
		case "1064": // ER_PARSE_ERROR
			msg.WriteString(": SQL syntax error")
			if e.DatabaseMessage != "" {
				msg.WriteString(fmt.Sprintf(" - %s", e.DatabaseMessage))
			}
		case "1045": // ER_ACCESS_DENIED_ERROR
			msg.WriteString(": Access denied - invalid credentials")
		case "1049": // ER_BAD_DB_ERROR
			msg.WriteString(": Database does not exist")
		case "1007": // ER_DB_CREATE_EXISTS
			msg.WriteString(": Database already exists")
		default:
			// Generic error message
			if e.DatabaseMessage != "" {
				msg.WriteString(fmt.Sprintf(": %s", e.DatabaseMessage))
			}
		}
	} else {
		// Fallback to generic error message
		if e.DatabaseMessage != "" {
			msg.WriteString(fmt.Sprintf(": %s", e.DatabaseMessage))
		}
	}

	// Add hint if available (PostgreSQL)
	if e.DatabaseHint != "" {
		msg.WriteString(fmt.Sprintf(" (Hint: %s)", e.DatabaseHint))
	}

	return msg.String()
}

// MaskSQLContent masks sensitive information in SQL content
func MaskSQLContent(sqlContent string) string {
	if sqlContent == "" {
		return ""
	}

	// Common patterns to mask:
	// - Passwords in connection strings
	// - Credit card numbers
	// - SSN-like patterns
	// - Email addresses (optional, depending on requirements)
	// - API keys

	masked := sqlContent

	// Mask passwords in connection strings (PASSWORD='...', password='...')
	masked = maskPattern(masked, `(?i)password\s*=\s*['"][^'"]*['"]`, `password='***'`)

	// Mask MySQL password syntax (IDENTIFIED BY '...')
	masked = maskPattern(masked, `(?i)identified\s+by\s+['"][^'"]*['"]`, `IDENTIFIED BY '***'`)

	// Mask PostgreSQL password syntax (PASSWORD '...')
	masked = maskPattern(masked, `(?i)password\s+['"][^'"]*['"]`, `PASSWORD '***'`)

	// Mask credit card patterns (basic - 4 groups of 4 digits)
	masked = maskPattern(masked, `\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`, `****-****-****-****`)

	// Mask SSN patterns (XXX-XX-XXXX)
	masked = maskPattern(masked, `\b\d{3}-\d{2}-\d{4}\b`, `***-**-****`)

	// Mask API keys (common patterns: key='...', api_key='...', etc.)
	masked = maskPattern(masked, `(?i)(?:api[_-]?key|secret[_-]?key|auth[_-]?token)\s*=\s*['"][^'"]*['"]`, `api_key='***'`)

	return masked
}

// maskPattern masks a regex pattern in the given string
func maskPattern(text, pattern, replacement string) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		// If pattern compilation fails, return original text
		return text
	}
	return re.ReplaceAllString(text, replacement)
}
