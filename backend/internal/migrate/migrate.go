// Package migrate provides database migration management using golang-migrate.
package migrate

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Config holds migration configuration including the path to migration files and database connection URL.
type Config struct {
	MigrationsPath string
	DatabaseURL    string
}

// Migrator handles database migrations using golang-migrate library.
type Migrator struct {
	config Config
	m      *migrate.Migrate
}

// New creates a new Migrator instance with the provided configuration.
func New(cfg Config) (*Migrator, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	// Create migration instance
	m, err := migrate.New(
		fmt.Sprintf("file://%s", cfg.MigrationsPath),
		cfg.DatabaseURL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Migrator{
		config: cfg,
		m:      m,
	}, nil
}

// Close closes the migrator and releases resources.
func (m *Migrator) Close() error {
	srcErr, dbErr := m.m.Close()
	if srcErr != nil {
		return srcErr
	}
	if dbErr != nil {
		return dbErr
	}
	return nil
}

// Up runs all pending migrations to bring the database to the latest version.
func (m *Migrator) Up() error {
	if err := m.m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// Down rolls back all migrations to the initial empty database state.
func (m *Migrator) Down() error {
	if err := m.m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}
	return nil
}

// Version returns the current migration version and dirty state.
// A dirty state indicates an incomplete migration.
func (m *Migrator) Version() (version uint, dirty bool, err error) {
	version, dirty, err = m.m.Version()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations.
// Use with caution - this is for recovering from failed migrations.
func (m *Migrator) Force(version int) error {
	if err := m.m.Force(version); err != nil {
		return fmt.Errorf("failed to force version: %w", err)
	}
	return nil
}

// Steps runs n migrations where positive values migrate up and negative values migrate down.
func (m *Migrator) Steps(n int) error {
	if err := m.m.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run %d steps: %w", n, err)
	}
	return nil
}
