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

// Connect establishes a connection to PostgreSQL database with configuration
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
