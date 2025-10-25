package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/nsxbet/deeb/internal/migrate"
	"github.com/spf13/cobra"
)

var (
	migrationsPath string
	databaseURL    string
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration management",
	Long: `Database migration management tool for applying, rolling back, and
managing database schema migrations.`,
}

// upCmd represents the migrate up command
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Apply all pending migrations",
	Long:  `Apply all pending migrations to the database.`,
	RunE:  runMigrateUp,
}

// downCmd represents the migrate down command
var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Rollback all migrations",
	Long:  `Rollback all migrations from the database.`,
	RunE:  runMigrateDown,
}

// versionCmd represents the migrate version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show current migration version",
	Long:  `Show the current migration version and dirty state.`,
	RunE:  runMigrateVersion,
}

// forceCmd represents the migrate force command
var forceCmd = &cobra.Command{
	Use:   "force <version>",
	Short: "Force migration to specific version",
	Long:  `Force the migration version to a specific value without running migrations.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runMigrateForce,
}

// stepsCmd represents the migrate steps command
var stepsCmd = &cobra.Command{
	Use:   "steps <n>",
	Short: "Run n migrations (positive=up, negative=down)",
	Long:  `Run a specific number of migrations. Positive values apply migrations, negative values rollback.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runMigrateSteps,
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	// Add subcommands
	migrateCmd.AddCommand(upCmd)
	migrateCmd.AddCommand(downCmd)
	migrateCmd.AddCommand(versionCmd)
	migrateCmd.AddCommand(forceCmd)
	migrateCmd.AddCommand(stepsCmd)

	// Persistent flags for all migrate commands
	migrateCmd.PersistentFlags().StringVarP(&migrationsPath, "path", "p", "migrations", "Path to migrations directory")
	migrateCmd.PersistentFlags().
		StringVarP(&databaseURL, "database", "d", os.Getenv("DATABASE_URL"), "Database connection string")
}

// createMigrator creates a new migrator instance
func createMigrator() (*migrate.Migrator, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable or --database flag must be set")
	}

	migrator, err := migrate.New(migrate.Config{
		MigrationsPath: migrationsPath,
		DatabaseURL:    databaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize migrator: %w", err)
	}

	return migrator, nil
}

func runMigrateUp(cmd *cobra.Command, args []string) error {
	migrator, err := createMigrator()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := migrator.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close migrator: %v\n", closeErr)
		}
	}()

	if err := migrator.Up(); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("Migrations applied successfully")
	return nil
}

func runMigrateDown(cmd *cobra.Command, args []string) error {
	migrator, err := createMigrator()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := migrator.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close migrator: %v\n", closeErr)
		}
	}()

	if err := migrator.Down(); err != nil {
		return fmt.Errorf("migration rollback failed: %w", err)
	}

	fmt.Println("Migrations rolled back successfully")
	return nil
}

func runMigrateVersion(cmd *cobra.Command, args []string) error {
	migrator, err := createMigrator()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := migrator.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close migrator: %v\n", closeErr)
		}
	}()

	version, dirty, err := migrator.Version()
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	fmt.Printf("Version: %d\n", version)
	fmt.Printf("Dirty: %v\n", dirty)
	return nil
}

func runMigrateForce(cmd *cobra.Command, args []string) error {
	version, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}

	migrator, err := createMigrator()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := migrator.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close migrator: %v\n", closeErr)
		}
	}()

	if err := migrator.Force(version); err != nil {
		return fmt.Errorf("force migration failed: %w", err)
	}

	fmt.Printf("Forced version to: %d\n", version)
	return nil
}

func runMigrateSteps(cmd *cobra.Command, args []string) error {
	n, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid steps number: %w", err)
	}

	migrator, err := createMigrator()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := migrator.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close migrator: %v\n", closeErr)
		}
	}()

	if err := migrator.Steps(n); err != nil {
		return fmt.Errorf("migration steps failed: %w", err)
	}

	fmt.Printf("Executed %d migration steps\n", n)
	return nil
}
