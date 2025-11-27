// Package database provides database connection management for target databases (MySQL and PostgreSQL).
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
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
	if params.DatabaseInstance.DatabaseEngine != models.DatabaseEngineMySQL {
		return nil, fmt.Errorf("invalid database type: expected mysql, got %s", params.DatabaseInstance.DatabaseEngine)
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
		zap.String("ssl_mode", strconv.FormatBool(params.DatabaseInstance.RequireSSL)),
	)

	// Connect to database
	db, err := sqlx.ConnectContext(ctx, "mysql", dsn)
	if err != nil {
		// Create structured connection error
		connErr := NewConnectionError(
			string(params.DatabaseInstance.DatabaseEngine),
			params.Database.DatabaseName,
			params.Database.ID.String(),
			params.DatabaseInstance.Name,
			params.DatabaseInstance.Host,
			params.DatabaseInstance.Port,
			params.DatabaseInstance.Username,
			params.DatabaseInstance.RequireSSL,
			10*time.Second, // timeout from DSN
			err,
		)

		// Log comprehensive error details
		logFields := []zap.Field{
			zap.String("database_id", params.Database.ID.String()),
			zap.String("database_name", params.Database.DatabaseName),
			zap.String("database_instance_name", params.DatabaseInstance.Name),
			zap.String("database_engine", string(params.DatabaseInstance.DatabaseEngine)),
			zap.String("host", params.DatabaseInstance.Host),
			zap.Int("port", params.DatabaseInstance.Port),
			zap.String("username", params.DatabaseInstance.Username),
			zap.Bool("ssl_required", params.DatabaseInstance.RequireSSL),
			zap.String("error_type", connErr.ErrorType),
			zap.Error(err),
		}

		if connErr.ErrorCode != "" {
			logFields = append(logFields, zap.String("error_code", connErr.ErrorCode))
		}

		if connErr.StackTrace != "" {
			logFields = append(logFields, zap.String("stack_trace", connErr.StackTrace))
		}

		cm.logger.Error("MySQL database connection failed", logFields...)

		return nil, connErr
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
		// Create structured connection error for ping failure
		connErr := NewConnectionError(
			string(params.DatabaseInstance.DatabaseEngine),
			params.Database.DatabaseName,
			params.Database.ID.String(),
			params.DatabaseInstance.Name,
			params.DatabaseInstance.Host,
			params.DatabaseInstance.Port,
			params.DatabaseInstance.Username,
			params.DatabaseInstance.RequireSSL,
			5*time.Second, // ping timeout
			err,
		)

		// Log comprehensive error details
		logFields := []zap.Field{
			zap.String("database_id", params.Database.ID.String()),
			zap.String("database_name", params.Database.DatabaseName),
			zap.String("database_instance_name", params.DatabaseInstance.Name),
			zap.String("database_engine", string(params.DatabaseInstance.DatabaseEngine)),
			zap.String("host", params.DatabaseInstance.Host),
			zap.Int("port", params.DatabaseInstance.Port),
			zap.String("error_type", connErr.ErrorType),
			zap.Error(err),
		}

		if connErr.ErrorCode != "" {
			logFields = append(logFields, zap.String("error_code", connErr.ErrorCode))
		}

		cm.logger.Error("MySQL database ping failed", logFields...)

		return nil, connErr
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
	if params.DatabaseInstance.DatabaseEngine != models.DatabaseEnginePostgreSQL {
		return nil, fmt.Errorf("invalid database type: expected postgres, got %s", params.DatabaseInstance.DatabaseEngine)
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
		zap.String("ssl_mode", sslMode),
	)

	// Connect to database
	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		// Create structured connection error
		connErr := NewConnectionError(
			string(params.DatabaseInstance.DatabaseEngine),
			params.Database.DatabaseName,
			params.Database.ID.String(),
			params.DatabaseInstance.Name,
			params.DatabaseInstance.Host,
			params.DatabaseInstance.Port,
			params.DatabaseInstance.Username,
			params.DatabaseInstance.RequireSSL,
			10*time.Second, // reasonable default timeout
			err,
		)

		// Log comprehensive error details
		logFields := []zap.Field{
			zap.String("database_id", params.Database.ID.String()),
			zap.String("database_name", params.Database.DatabaseName),
			zap.String("database_instance_name", params.DatabaseInstance.Name),
			zap.String("database_engine", string(params.DatabaseInstance.DatabaseEngine)),
			zap.String("host", params.DatabaseInstance.Host),
			zap.Int("port", params.DatabaseInstance.Port),
			zap.String("username", params.DatabaseInstance.Username),
			zap.Bool("ssl_required", params.DatabaseInstance.RequireSSL),
			zap.String("ssl_mode", sslMode),
			zap.String("error_type", connErr.ErrorType),
			zap.Error(err),
		}

		if connErr.ErrorCode != "" {
			logFields = append(logFields, zap.String("error_code", connErr.ErrorCode))
		}

		if connErr.StackTrace != "" {
			logFields = append(logFields, zap.String("stack_trace", connErr.StackTrace))
		}

		cm.logger.Error("PostgreSQL database connection failed", logFields...)

		return nil, connErr
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
		// Create structured connection error for ping failure
		connErr := NewConnectionError(
			string(params.DatabaseInstance.DatabaseEngine),
			params.Database.DatabaseName,
			params.Database.ID.String(),
			params.DatabaseInstance.Name,
			params.DatabaseInstance.Host,
			params.DatabaseInstance.Port,
			params.DatabaseInstance.Username,
			params.DatabaseInstance.RequireSSL,
			5*time.Second, // ping timeout
			err,
		)

		// Log comprehensive error details
		logFields := []zap.Field{
			zap.String("database_id", params.Database.ID.String()),
			zap.String("database_name", params.Database.DatabaseName),
			zap.String("database_instance_name", params.DatabaseInstance.Name),
			zap.String("database_engine", string(params.DatabaseInstance.DatabaseEngine)),
			zap.String("host", params.DatabaseInstance.Host),
			zap.Int("port", params.DatabaseInstance.Port),
			zap.String("error_type", connErr.ErrorType),
			zap.Error(err),
		}

		if connErr.ErrorCode != "" {
			logFields = append(logFields, zap.String("error_code", connErr.ErrorCode))
		}

		cm.logger.Error("PostgreSQL database ping failed", logFields...)

		return nil, connErr
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

	switch params.DatabaseInstance.DatabaseEngine {
	case models.DatabaseEngineMySQL:
		return cm.ConnectMySQL(ctx, params)
	case models.DatabaseEnginePostgreSQL:
		return cm.ConnectPostgreSQL(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", params.DatabaseInstance.DatabaseEngine)
	}
}

// DatabaseExistsOnInstance checks if a database physically exists on the database instance
func (cm *ConnectionManager) DatabaseExistsOnInstance(
	ctx context.Context,
	instance *models.DatabaseInstance,
	databaseName string,
) (bool, error) {
	if instance == nil {
		return false, fmt.Errorf("database instance is required")
	}

	switch instance.DatabaseEngine {
	case models.DatabaseEngineMySQL:
		return cm.checkMySQLDatabaseExists(ctx, instance, databaseName)
	case models.DatabaseEnginePostgreSQL:
		return cm.checkPostgreSQLDatabaseExists(ctx, instance, databaseName)
	default:
		return false, fmt.Errorf("unsupported database type: %s", instance.DatabaseEngine)
	}
}

// checkMySQLDatabaseExists checks if a database exists on a MySQL instance
func (cm *ConnectionManager) checkMySQLDatabaseExists(
	ctx context.Context,
	instance *models.DatabaseInstance,
	databaseName string,
) (bool, error) {
	// Decrypt password
	password, err := utils.DecryptPassword(instance.ConnectionDetailsEncrypted, cm.encryptionKey)
	if err != nil {
		return false, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Build MySQL DSN without database name to connect to the server
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
		instance.Username,
		password,
		instance.Host,
		instance.Port,
	)

	// Add SSL parameter if required
	if instance.RequireSSL {
		dsn += "?tls=true"
	} else {
		dsn += "?tls=false"
	}

	// Connect to MySQL server
	db, err := sqlx.ConnectContext(ctx, "mysql", dsn)
	if err != nil {
		return false, fmt.Errorf("failed to connect to MySQL instance: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			cm.logger.Error("Failed to close MySQL connection", zap.Error(closeErr))
		}
	}()

	// Query for database existence
	query := "SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?"
	var foundName string
	err = db.GetContext(ctx, &foundName, query, databaseName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check database existence: %w", err)
	}

	return true, nil
}

// checkPostgreSQLDatabaseExists checks if a database exists on a PostgreSQL instance
func (cm *ConnectionManager) checkPostgreSQLDatabaseExists(
	ctx context.Context,
	instance *models.DatabaseInstance,
	databaseName string,
) (bool, error) {
	// Decrypt password
	password, err := utils.DecryptPassword(instance.ConnectionDetailsEncrypted, cm.encryptionKey)
	if err != nil {
		return false, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Determine SSL mode
	sslMode := "disable"
	if instance.RequireSSL {
		sslMode = "require"
	}

	// Build PostgreSQL DSN connecting to postgres database (default)
	var dsn string
	if password != "" {
		dsn = fmt.Sprintf(
			"postgres://%s:%s@%s:%d/postgres?sslmode=%s",
			instance.Username,
			password,
			instance.Host,
			instance.Port,
			sslMode,
		)
	} else {
		dsn = fmt.Sprintf(
			"postgres://%s@%s:%d/postgres?sslmode=%s",
			instance.Username,
			instance.Host,
			instance.Port,
			sslMode,
		)
	}

	// Connect to PostgreSQL server
	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return false, fmt.Errorf("failed to connect to PostgreSQL instance: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			cm.logger.Error("Failed to close PostgreSQL connection", zap.Error(closeErr))
		}
	}()

	// Query for database existence
	query := "SELECT datname FROM pg_database WHERE datname = $1"
	var foundName string
	err = db.GetContext(ctx, &foundName, query, databaseName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check database existence: %w", err)
	}

	return true, nil
}

// ListDatabasesOnInstance lists all databases on the database instance, excluding system databases
func (cm *ConnectionManager) ListDatabasesOnInstance(
	ctx context.Context,
	instance *models.DatabaseInstance,
) ([]string, error) {
	if instance == nil {
		return nil, fmt.Errorf("database instance is required")
	}

	switch instance.DatabaseEngine {
	case models.DatabaseEngineMySQL:
		return cm.listMySQLDatabases(ctx, instance)
	case models.DatabaseEnginePostgreSQL:
		return cm.listPostgreSQLDatabases(ctx, instance)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", instance.DatabaseEngine)
	}
}

// listMySQLDatabases lists all databases on a MySQL instance, excluding system databases
func (cm *ConnectionManager) listMySQLDatabases(
	ctx context.Context,
	instance *models.DatabaseInstance,
) ([]string, error) {
	// Decrypt password
	password, err := utils.DecryptPassword(instance.ConnectionDetailsEncrypted, cm.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Build MySQL DSN without database name to connect to the server
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
		instance.Username,
		password,
		instance.Host,
		instance.Port,
	)

	// Add SSL parameter if required
	if instance.RequireSSL {
		dsn += "?tls=true"
	} else {
		dsn += "?tls=false"
	}

	// Connect to MySQL server
	db, err := sqlx.ConnectContext(ctx, "mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL instance: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			cm.logger.Error("Failed to close MySQL connection", zap.Error(closeErr))
		}
	}()

	// Query for all databases excluding system databases
	query := `
		SELECT SCHEMA_NAME
		FROM information_schema.SCHEMATA
		WHERE SCHEMA_NAME NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
		ORDER BY SCHEMA_NAME
	`

	var databases []string
	err = db.SelectContext(ctx, &databases, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	return databases, nil
}

// listPostgreSQLDatabases lists all databases on a PostgreSQL instance, excluding system databases
func (cm *ConnectionManager) listPostgreSQLDatabases(
	ctx context.Context,
	instance *models.DatabaseInstance,
) ([]string, error) {
	// Decrypt password
	password, err := utils.DecryptPassword(instance.ConnectionDetailsEncrypted, cm.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Determine SSL mode
	sslMode := "disable"
	if instance.RequireSSL {
		sslMode = "require"
	}

	// Build PostgreSQL DSN connecting to postgres database (default)
	var dsn string
	if password != "" {
		dsn = fmt.Sprintf(
			"postgres://%s:%s@%s:%d/postgres?sslmode=%s",
			instance.Username,
			password,
			instance.Host,
			instance.Port,
			sslMode,
		)
	} else {
		dsn = fmt.Sprintf(
			"postgres://%s@%s:%d/postgres?sslmode=%s",
			instance.Username,
			instance.Host,
			instance.Port,
			sslMode,
		)
	}

	// Connect to PostgreSQL server
	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL instance: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			cm.logger.Error("Failed to close PostgreSQL connection", zap.Error(closeErr))
		}
	}()

	// Query for all databases excluding system databases
	query := `
		SELECT datname
		FROM pg_database
		WHERE datname NOT IN ('postgres', 'template0', 'template1')
		AND datistemplate = false
		ORDER BY datname
	`

	var databases []string
	err = db.SelectContext(ctx, &databases, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	return databases, nil
}
