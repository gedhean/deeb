package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nsxbet/deeb/internal/migrate"
)

func main() {
	// Parse command-line flags
	var migrationsPath string
	var databaseURL string
	var command string

	flag.StringVar(&migrationsPath, "path", "migrations", "Path to migrations directory")
	flag.StringVar(&databaseURL, "database", os.Getenv("DATABASE_URL"), "Database connection string")
	flag.StringVar(&command, "command", "up", "Migration command: up, down, version, force, steps")
	flag.Parse()

	// Validate database URL
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable or -database flag must be set")
	}

	// Create migrator
	migrator, err := migrate.New(migrate.Config{
		MigrationsPath: migrationsPath,
		DatabaseURL:    databaseURL,
	})
	if err != nil {
		log.Fatalf("Failed to initialize migrator: %v", err)
	}
	defer migrator.Close()

	// Execute command
	if err := executeCommand(migrator, command, flag.Args()); err != nil {
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
			return fmt.Errorf("force command requires version argument")
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
			return fmt.Errorf("steps command requires number argument")
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

	default:
		return fmt.Errorf("unknown command: %s (valid commands: up, down, version, force, steps)", command)
	}
}
