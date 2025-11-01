// Package database provides database connection management for PostgreSQL.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/nsxbet/deeb/internal/config"
	"go.uber.org/zap"
)

// Connect establishes a connection to PostgreSQL database with configuration.
// Configures connection pool settings and verifies connectivity with a ping.
func Connect(cfg *config.Config, logger *zap.Logger) (*sqlx.DB, error) {
	// Build DSN - omit password parameter if empty to avoid PostgreSQL parsing issues
	dsn := buildDSN(cfg.Database)

	logger.Info("Connecting to database",
		zap.String("host", cfg.Database.Host),
		zap.String("database", cfg.Database.Database),
		zap.String("user", cfg.Database.User),
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connected successfully",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("database", cfg.Database.Database),
		zap.Int("max_open_conns", cfg.Database.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.Database.MaxIdleConns),
		zap.Duration("conn_max_lifetime", cfg.Database.ConnMaxLifetime),
	)

	// Log pool stats
	stats := db.Stats()
	logger.Debug("Database connection pool stats",
		zap.Int("open_connections", stats.OpenConnections),
		zap.Int("in_use", stats.InUse),
		zap.Int("idle", stats.Idle),
		zap.Int64("wait_count", stats.WaitCount),
		zap.Duration("wait_duration", stats.WaitDuration),
		zap.Int64("max_idle_closed", stats.MaxIdleClosed),
		zap.Int64("max_idle_time_closed", stats.MaxIdleTimeClosed),
		zap.Int64("max_lifetime_closed", stats.MaxLifetimeClosed),
	)

	return db, nil
}

// buildDSN constructs a PostgreSQL connection string from database config
func buildDSN(cfg config.DatabaseConfig) string {
	// Omit password parameter if empty to avoid PostgreSQL parsing issues
	// When password is empty string with "password=" prefix, PostgreSQL
	// ignores subsequent parameters like dbname
	if cfg.Password != "" {
		return fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host,
			cfg.Port,
			cfg.User,
			cfg.Password,
			cfg.Database,
			cfg.SSLMode,
		)
	}

	return fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Database,
		cfg.SSLMode,
	)
}
