// Package utils provides utility functions for the Deeb application.
package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var (
	// ErrConnectionTimeout is returned when connection test times out
	ErrConnectionTimeout = errors.New("connection test timed out")
	// ErrInvalidCredentials is returned when credentials are invalid
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrConnectionFailed is returned when connection cannot be established
	ErrConnectionFailed = errors.New("connection failed")
)

// ConnectionTestParams represents parameters for testing a database connection
type ConnectionTestParams struct {
	DatabaseType             string // "mysql" or "postgresql"
	Host                     string
	Port                     int
	Username                 string
	Password                 string  // Plain password (will be decrypted if EncryptedPassword is provided)
	EncryptedPassword        *string // Encrypted password (if provided, Password will be ignored)
	EncryptionKey            string  // Encryption key for decrypting password
	Database                 string  // Database name (optional for connection test)
	UseSSL                   bool    // Whether to require SSL
	ConnectionTimeoutSeconds int     // Timeout in seconds (default: 10)
}

// connectionDetails represents the JSON structure for encrypted connection details
type connectionDetails struct {
	Password string `json:"password"`
}

// DecryptPassword decrypts password from encrypted JSON string
// This is a public wrapper around the private decryptPassword function
func DecryptPassword(encrypted *string, encryptionKey string) (string, error) {
	return decryptPassword(encrypted, encryptionKey)
}

// decryptPassword decrypts password from encrypted JSON string
func decryptPassword(encrypted *string, encryptionKey string) (string, error) {
	if encrypted == nil || *encrypted == "" {
		return "", nil
	}

	decrypted, err := Decrypt(*encrypted, encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	var details connectionDetails
	if err := json.Unmarshal([]byte(decrypted), &details); err != nil {
		return "", fmt.Errorf("failed to parse decrypted password: %w", err)
	}

	return details.Password, nil
}

// TestMySQLConnection tests a MySQL database connection with the provided parameters
func TestMySQLConnection(ctx context.Context, params ConnectionTestParams) error {
	// Determine password to use
	password := params.Password
	if params.EncryptedPassword != nil && params.EncryptionKey != "" {
		decrypted, err := decryptPassword(params.EncryptedPassword, params.EncryptionKey)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidCredentials, err)
		}
		password = decrypted
	}

	// Set default timeout if not specified
	timeout := params.ConnectionTimeoutSeconds
	if timeout <= 0 {
		timeout = 10
	}

	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Build MySQL DSN
	// Format: username:password@tcp(host:port)/database?params
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		params.Username,
		password,
		params.Host,
		params.Port,
		params.Database,
	)

	// Add SSL parameter if required
	if params.UseSSL {
		dsn += "?tls=true"
	} else {
		dsn += "?tls=false"
	}

	// Attempt connection
	db, err := sqlx.ConnectContext(testCtx, "mysql", dsn)
	if err != nil {
		// Check if error is due to timeout
		if testCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w: MySQL connection test timed out after %d seconds", ErrConnectionTimeout, timeout)
		}

		// Provide clear error messages
		if err.Error() == "Access denied for user" {
			return fmt.Errorf(
				"%w: MySQL authentication failed for user '%s' at %s:%d",
				ErrInvalidCredentials,
				params.Username,
				params.Host,
				params.Port,
			)
		}

		return fmt.Errorf("%w: MySQL connection failed to %s:%d - %v", ErrConnectionFailed, params.Host, params.Port, err)
	}
	defer db.Close()

	// Test connection with ping
	if err := db.PingContext(testCtx); err != nil {
		if testCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w: MySQL ping timed out after %d seconds", ErrConnectionTimeout, timeout)
		}
		return fmt.Errorf("%w: MySQL ping failed - %v", ErrConnectionFailed, err)
	}

	return nil
}

// TestPostgreSQLConnection tests a PostgreSQL database connection with the provided parameters
func TestPostgreSQLConnection(ctx context.Context, params ConnectionTestParams) error {
	// Determine password to use
	password := params.Password
	if params.EncryptedPassword != nil && params.EncryptionKey != "" {
		decrypted, err := decryptPassword(params.EncryptedPassword, params.EncryptionKey)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidCredentials, err)
		}
		password = decrypted
	}

	// Set default timeout if not specified
	timeout := params.ConnectionTimeoutSeconds
	if timeout <= 0 {
		timeout = 10
	}

	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Build PostgreSQL DSN
	// Format: postgres://username:password@host:port/database?sslmode=...
	sslMode := "disable"
	if params.UseSSL {
		sslMode = "require"
	}

	// Omit password parameter if empty to avoid PostgreSQL parsing issues
	var dsn string
	if password != "" {
		dsn = fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			params.Username,
			password,
			params.Host,
			params.Port,
			params.Database,
			sslMode,
		)
	} else {
		dsn = fmt.Sprintf(
			"postgres://%s@%s:%d/%s?sslmode=%s",
			params.Username,
			params.Host,
			params.Port,
			params.Database,
			sslMode,
		)
	}

	// Attempt connection
	db, err := sqlx.ConnectContext(testCtx, "postgres", dsn)
	if err != nil {
		// Check if error is due to timeout
		if testCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w: PostgreSQL connection test timed out after %d seconds", ErrConnectionTimeout, timeout)
		}

		// Provide clear error messages
		if err.Error() == "pq: password authentication failed for user" {
			return fmt.Errorf(
				"%w: PostgreSQL authentication failed for user '%s' at %s:%d",
				ErrInvalidCredentials,
				params.Username,
				params.Host,
				params.Port,
			)
		}

		return fmt.Errorf("%w: PostgreSQL connection failed to %s:%d - %v", ErrConnectionFailed, params.Host, params.Port, err)
	}
	defer db.Close()

	// Test connection with ping
	if err := db.PingContext(testCtx); err != nil {
		if testCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w: PostgreSQL ping timed out after %d seconds", ErrConnectionTimeout, timeout)
		}
		return fmt.Errorf("%w: PostgreSQL ping failed - %v", ErrConnectionFailed, err)
	}

	return nil
}
