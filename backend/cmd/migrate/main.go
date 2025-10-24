package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nsxbet/deeb/internal/migrate"
)

func main() {
	// Define flags (but not the command itself)
	migrationsPath := flag.String("path", "migrations", "Path to migrations directory")
	databaseURL := flag.String("database", os.Getenv("DATABASE_URL"), "Database connection string")
	flag.Parse()

	// Get subcommand (first positional argument after flags)
	args := flag.Args()
	if len(args) == 0 {
		args = []string{"up"} // Default to "up" if no command specified
	}
	command := args[0]
	commandArgs := args[1:] // Remaining args for the command

	// Validate database URL
	if *databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable or -database flag must be set")
	}

	// Create migrator
	migrator, err := migrate.New(migrate.Config{
		MigrationsPath: *migrationsPath,
		DatabaseURL:    *databaseURL,
	})
	if err != nil {
		log.Fatalf("Failed to initialize migrator: %v", err)
	}
	defer migrator.Close()

	// Execute command
	if err := executeCommand(migrator, command, commandArgs); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
}

func executeCommand(m *migrate.Migrator, command string, args []string) error {
	switch command {
	case "up":
		if err := m.Up(); err != nil {
			return err
		}
		log.Println("Migrations applied successfully")
		return nil

	case "down":
		if err := m.Down(); err != nil {
			return err
		}
		log.Println("Migrations rolled back successfully")
		return nil

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			return err
		}
		fmt.Printf("Version: %d\nDirty: %v\n", version, dirty)
		return nil

	case "force":
		if len(args) < 1 {
			return fmt.Errorf("force command requires version argument: migrate force <version>")
		}
		var version int
		if _, err := fmt.Sscanf(args[0], "%d", &version); err != nil {
			return fmt.Errorf("invalid version: %w", err)
		}
		if err := m.Force(version); err != nil {
			return err
		}
		fmt.Printf("Forced version to: %d\n", version)
		return nil

	case "steps":
		if len(args) < 1 {
			return fmt.Errorf("steps command requires number argument: migrate steps <n>")
		}
		var n int
		if _, err := fmt.Sscanf(args[0], "%d", &n); err != nil {
			return fmt.Errorf("invalid steps number: %w", err)
		}
		if err := m.Steps(n); err != nil {
			return err
		}
		fmt.Printf("Executed %d migration steps\n", n)
		return nil

	case "help", "-h", "--help":
		printHelp()
		return nil

	default:
		return fmt.Errorf("unknown command: %s\n\nRun 'migrate help' for usage", command)
	}
}

func printHelp() {
	fmt.Println(`Database Migration Tool

Usage:
  migrate [command] [options]

Commands:
  up              Apply all pending migrations (default)
  down            Rollback all migrations
  version         Show current migration version
  force <version> Force migration to specific version
  steps <n>       Run n migrations (positive=up, negative=down)
  help            Show this help message

Options:
  -database string
        Database connection string (default: $DATABASE_URL)
  -path string
        Path to migrations directory (default: "migrations")

Examples:
  migrate                          # Apply all pending migrations
  migrate up                       # Apply all pending migrations
  migrate down                     # Rollback all migrations
  migrate version                  # Show current version
  migrate force 5                  # Force to version 5
  migrate steps 2                  # Apply next 2 migrations
  migrate steps -1                 # Rollback last migration
  migrate -database "postgres://..." up
`)
}
