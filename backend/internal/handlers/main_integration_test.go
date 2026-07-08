//go:build integration
// +build integration

package handlers_test

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Shared Postgres server used by every integration test in this package. It is
// started once in TestMain and each test gets its own freshly-created database
// on this server, which is dramatically cheaper than booting a container per test.
const (
	testDBUser     = "deeb"
	testDBPassword = "deeb"
)

var (
	sharedPGHost string
	sharedPGPort int
	dbCounter    atomic.Int64
)

// TestMain boots a single Postgres container for the whole package and tears it
// down once all tests have run.
func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := postgrescontainer.Run(
		ctx,
		"postgres:16-alpine",
		postgrescontainer.WithDatabase("deeb_test"),
		postgrescontainer.WithUsername(testDBUser),
		postgrescontainer.WithPassword(testDBPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start shared postgres container: %v\n", err)
		os.Exit(1)
	}

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get shared postgres host: %v\n", err)
		terminateShared(ctx, container)
		os.Exit(1)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get shared postgres port: %v\n", err)
		terminateShared(ctx, container)
		os.Exit(1)
	}
	sharedPGHost = host
	sharedPGPort = port.Int()

	code := m.Run()

	terminateShared(ctx, container)

	os.Exit(code)
}

// terminateShared tears down the shared container, logging (but not failing on)
// any error. Called on every TestMain exit path so a failure after container
// creation never orphans the fixture.
func terminateShared(ctx context.Context, c testcontainers.Container) {
	if err := c.Terminate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to terminate shared postgres container: %v\n", err)
	}
}

// sharedDSN builds a connection string to the given database on the shared server.
func sharedDSN(dbName string) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		sharedPGHost, sharedPGPort, testDBUser, testDBPassword, dbName)
}

// newSharedTestDB creates a fresh, isolated database on the shared Postgres
// server and returns a connection to it plus a cleanup function that drops it.
func newSharedTestDB(t *testing.T) (*sqlx.DB, func()) {
	t.Helper()

	dbName := fmt.Sprintf("deeb_test_%d", dbCounter.Add(1))
	createSharedDatabase(t, dbName)

	db, err := sqlx.Connect("postgres", sharedDSN(dbName))
	require.NoError(t, err, "failed to connect to test database")

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	cleanup := func() {
		db.Close()
		dropSharedDatabase(t, dbName)
	}

	return db, cleanup
}

// createSharedDatabase creates a new database on the shared server.
func createSharedDatabase(t *testing.T, dbName string) {
	t.Helper()

	admin, err := sqlx.Connect("postgres", sharedDSN("postgres"))
	require.NoError(t, err, "failed to connect to shared postgres server")
	defer admin.Close()

	_, err = admin.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	require.NoError(t, err, "failed to create test database %s", dbName)
}

// dropSharedDatabase drops a database from the shared server, forcing any
// remaining connections closed. Failures are logged, not fatal, so cleanup of
// one test never blocks another.
func dropSharedDatabase(t *testing.T, dbName string) {
	t.Helper()

	admin, err := sqlx.Connect("postgres", sharedDSN("postgres"))
	if err != nil {
		t.Logf("failed to connect for cleanup of database %s: %v", dbName, err)
		return
	}
	defer admin.Close()

	if _, err := admin.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName)); err != nil {
		t.Logf("failed to drop test database %s: %v", dbName, err)
	}
}
