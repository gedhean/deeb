package migrate

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Config holds migration configuration
type Config struct {
	MigrationsPath string
	DatabaseURL    string
}

// Migrator handles database migrations
type Migrator struct {
	config Config
	m      *migrate.Migrate
}

// New creates a new Migrator instance
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

// Close closes the migrator
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

// Up runs all pending migrations
func (m *Migrator) Up() error {
	if err := m.m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// Down rolls back all migrations
func (m *Migrator) Down() error {
	if err := m.m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}
	return nil
}

// Version returns the current migration version and dirty state
func (m *Migrator) Version() (version uint, dirty bool, err error) {
	version, dirty, err = m.m.Version()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get version: %w", err)
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations
func (m *Migrator) Force(version int) error {
	if err := m.m.Force(version); err != nil {
		return fmt.Errorf("failed to force version: %w", err)
	}
	return nil
}

// Steps runs n migrations (positive for up, negative for down)
func (m *Migrator) Steps(n int) error {
	if err := m.m.Steps(n); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run %d steps: %w", n, err)
	}
	return nil
}
