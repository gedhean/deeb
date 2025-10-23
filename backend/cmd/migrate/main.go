package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var migrationsPath string
	var databaseURL string
	var command string

	flag.StringVar(&migrationsPath, "path", "migrations", "Path to migrations directory")
	flag.StringVar(&databaseURL, "database", os.Getenv("DATABASE_URL"), "Database connection string")
	flag.StringVar(&command, "command", "up", "Migration command: up, down, version, force")
	flag.Parse()

	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable or -database flag must be set")
	}

	// Create migration instance
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		databaseURL,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	// Execute command
	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Migrations applied successfully")

	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		log.Println("Migrations rolled back successfully")

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Version: %d\nDirty: %v\n", version, dirty)

	case "force":
		if len(flag.Args()) < 1 {
			log.Fatal("force command requires version argument")
		}
		var version int
		if _, err := fmt.Sscanf(flag.Arg(0), "%d", &version); err != nil {
			log.Fatalf("Invalid version: %v", err)
		}
		if err := m.Force(version); err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}
		fmt.Printf("Forced version to: %d\n", version)

	default:
		log.Fatalf("Unknown command: %s (valid commands: up, down, version, force)", command)
	}
}
