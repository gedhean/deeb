// Package database provides database connection management for target databases (MySQL and PostgreSQL).
package database

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/utils"
	"go.uber.org/zap"
)

// ConnectionManager manages connections to target databases
type ConnectionManager struct {
	encryptionKey string
	logger        *zap.Logger
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(encryptionKey string, logger *zap.Logger) *ConnectionManager {
	return &ConnectionManager{
		encryptionKey: encryptionKey,
		logger:        logger,
	}
}

// ConnectionParams represents parameters for establishing a database connection
type ConnectionParams struct {
	DatabaseInstance *models.DatabaseInstance
	Database         *models.Database
	MaxOpenConns     int           // Maximum open connections (default: 10)
	MaxIdleConns     int           // Maximum idle connections (default: 5)
	ConnMaxLifetime  time.Duration // Connection max lifetime (default: 5 minutes)
}

// ConnectMySQL establishes a connection to a MySQL database
func (cm *ConnectionManager) ConnectMySQL(ctx context.Context, params ConnectionParams) (*sqlx.DB, error) {
	if params.DatabaseInstance == nil {
		return nil, fmt.Errorf("database instance is required")
	}
	if params.Database == nil {
		return nil, fmt.Errorf("database is required")
	}

	// Validate database type
	if params.DatabaseInstance.DatabaseType != models.DatabaseTypeMySQL {
		return nil, fmt.Errorf("invalid database type: expected mysql, got %s", params.DatabaseInstance.DatabaseType)
	}

	// Decrypt password
	password, err := utils.DecryptPassword(params.DatabaseInstance.ConnectionDetailsEncrypted, cm.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Build MySQL DSN
	// Format: username:password@tcp(host:port)/database?params
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		params.DatabaseInstance.Username,
		password,
		params.DatabaseInstance.Host,
		params.DatabaseInstance.Port,
		params.Database.DatabaseName,
	)

	// Add SSL parameter if required
	if params.DatabaseInstance.RequireSSL {
		dsn += "?tls=true"
	} else {
		dsn += "?tls=false"
	}

	// Add connection timeout
	dsn += "&timeout=10s&readTimeout=10s&writeTimeout=10s"

	cm.logger.Info("Connecting to MySQL database",
		zap.String("host", params.DatabaseInstance.Host),
		zap.Int("port", params.DatabaseInstance.Port),
		zap.String("database", params.Database.DatabaseName),
		zap.String("username", params.DatabaseInstance.Username),
	)

	// Connect to database
	db, err := sqlx.ConnectContext(ctx, "mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL database: %w", err)
	}

	// Configure connection pool with optimized settings for concurrent executions
	// Increased pool sizes to handle concurrent change plan executions
	maxOpenConns := params.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 25 // Increased from 10 for better concurrency
	}
	maxIdleConns := params.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10 // Increased from 5 to reduce connection churn
	}
	connMaxLifetime := params.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = 15 * time.Minute // Increased from 5min for better reuse
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	// Set connection max idle time to reduce stale connections
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Test connection
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(testCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping MySQL database: %w", err)
	}

	cm.logger.Info("MySQL database connected successfully",
		zap.String("host", params.DatabaseInstance.Host),
		zap.Int("port", params.DatabaseInstance.Port),
		zap.String("database", params.Database.DatabaseName),
		zap.Int("max_open_conns", maxOpenConns),
		zap.Int("max_idle_conns", maxIdleConns),
		zap.Duration("conn_max_lifetime", connMaxLifetime),
	)

	return db, nil
}

// ConnectPostgreSQL establishes a connection to a PostgreSQL database
func (cm *ConnectionManager) ConnectPostgreSQL(ctx context.Context, params ConnectionParams) (*sqlx.DB, error) {
	if params.DatabaseInstance == nil {
		return nil, fmt.Errorf("database instance is required")
	}
	if params.Database == nil {
		return nil, fmt.Errorf("database is required")
	}

	// Validate database type
	if params.DatabaseInstance.DatabaseType != models.DatabaseTypePostgreSQL {
		return nil, fmt.Errorf("invalid database type: expected postgresql, got %s", params.DatabaseInstance.DatabaseType)
	}

	// Decrypt password
	password, err := utils.DecryptPassword(params.DatabaseInstance.ConnectionDetailsEncrypted, cm.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Determine SSL mode
	sslMode := "disable"
	if params.DatabaseInstance.RequireSSL {
		sslMode = "require"
	}

	// Build PostgreSQL DSN
	// Format: postgres://username:password@host:port/database?sslmode=...
	// Omit password parameter if empty to avoid PostgreSQL parsing issues
	var dsn string
	if password != "" {
		dsn = fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			params.DatabaseInstance.Username,
			password,
			params.DatabaseInstance.Host,
			params.DatabaseInstance.Port,
			params.Database.DatabaseName,
			sslMode,
		)
	} else {
		dsn = fmt.Sprintf(
			"postgres://%s@%s:%d/%s?sslmode=%s",
			params.DatabaseInstance.Username,
			params.DatabaseInstance.Host,
			params.DatabaseInstance.Port,
			params.Database.DatabaseName,
			sslMode,
		)
	}

	cm.logger.Info("Connecting to PostgreSQL database",
		zap.String("host", params.DatabaseInstance.Host),
		zap.Int("port", params.DatabaseInstance.Port),
		zap.String("database", params.Database.DatabaseName),
		zap.String("username", params.DatabaseInstance.Username),
	)

	// Connect to database
	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL database: %w", err)
	}

	// Configure connection pool with optimized settings for concurrent executions
	// Increased pool sizes to handle concurrent change plan executions
	maxOpenConns := params.MaxOpenConns
	if maxOpenConns <= 0 {
		maxOpenConns = 25 // Increased from 10 for better concurrency
	}
	maxIdleConns := params.MaxIdleConns
	if maxIdleConns <= 0 {
		maxIdleConns = 10 // Increased from 5 to reduce connection churn
	}
	connMaxLifetime := params.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = 15 * time.Minute // Increased from 5min for better reuse
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	// Set connection max idle time to reduce stale connections
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Test connection
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(testCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	cm.logger.Info("PostgreSQL database connected successfully",
		zap.String("host", params.DatabaseInstance.Host),
		zap.Int("port", params.DatabaseInstance.Port),
		zap.String("database", params.Database.DatabaseName),
		zap.Int("max_open_conns", maxOpenConns),
		zap.Int("max_idle_conns", maxIdleConns),
		zap.Duration("conn_max_lifetime", connMaxLifetime),
	)

	return db, nil
}

// Connect establishes a connection to a database based on the database instance type
func (cm *ConnectionManager) Connect(ctx context.Context, params ConnectionParams) (*sqlx.DB, error) {
	if params.DatabaseInstance == nil {
		return nil, fmt.Errorf("database instance is required")
	}

	switch params.DatabaseInstance.DatabaseType {
	case models.DatabaseTypeMySQL:
		return cm.ConnectMySQL(ctx, params)
	case models.DatabaseTypePostgreSQL:
		return cm.ConnectPostgreSQL(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", params.DatabaseInstance.DatabaseType)
	}
}
