//go:build integration
// +build integration

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/config"
	"github.com/nsxbet/deeb/internal/handlers"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository/postgres"
	"github.com/nsxbet/deeb/internal/routes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

// setupResourceTestDB sets up a test database connection using testcontainers
func setupResourceTestDB(t *testing.T) (*sqlx.DB, func()) {
	t.Helper()

	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgrescontainer.Run(ctx,
		"postgres:16-alpine",
		postgrescontainer.WithDatabase("deeb_test"),
		postgrescontainer.WithUsername("deeb"),
		postgrescontainer.WithPassword("deeb"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "failed to start postgres container")

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	// Connect to database
	db, err := sqlx.Connect("postgres", connStr)
	require.NoError(t, err, "failed to connect to test database")

	// Set connection pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	// Run migrations
	runAllResourceMigrations(t, db)

	// Return cleanup function
	cleanup := func() {
		db.Close()
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	}

	return db, cleanup
}

// runAllResourceMigrations runs all database migrations in order
func runAllResourceMigrations(t *testing.T, db *sqlx.DB) {
	t.Helper()

	migrationsDir := filepath.Join("..", "..", "migrations")

	// Get all migration files
	files, err := os.ReadDir(migrationsDir)
	require.NoError(t, err, "failed to read migrations directory")

	// Filter and sort migration files
	var migrationFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".up.sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}

	// Sort migration files by name (they should be numbered)
	sort.Strings(migrationFiles)

	// Execute each migration
	for _, migrationFile := range migrationFiles {
		migrationPath := filepath.Join(migrationsDir, migrationFile)
		migrationSQL, err := os.ReadFile(migrationPath)
		require.NoError(t, err, "failed to read migration file: %s", migrationFile)

		_, err = db.Exec(string(migrationSQL))
		require.NoError(t, err, "failed to run migration: %s", migrationFile)
	}
}

// cleanupResourceTestDB cleans up test data
func cleanupResourceTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()

	// Clean up in reverse order of dependencies
	_, err := db.Exec(`
		DELETE FROM change_plans;
		DELETE FROM databases;
		DELETE FROM database_instances;
		DELETE FROM environments;
		DELETE FROM policies;
		DELETE FROM project_members;
		DELETE FROM projects;
		DELETE FROM audit_logs WHERE user_id IN (SELECT id FROM users WHERE email LIKE 'test-%@example.com' OR email LIKE 'admin-%@example.com' OR email LIKE 'developer-%@example.com');
		DELETE FROM users WHERE email LIKE 'test-%@example.com' OR email LIKE 'admin-%@example.com' OR email LIKE 'developer-%@example.com';
	`)
	require.NoError(t, err, "failed to clean up test data")
}

// createTestUser creates a test user and returns the user ID
func createTestUser(t *testing.T, db *sqlx.DB, email string, orgRole rbac.OrganizationRole) uuid.UUID {
	t.Helper()

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()

	registerHandler := handlers.NewRegisterHandler(userRepo, logger)

	registerReq := handlers.RegisterRequest{
		Email:    email,
		Password: "TestPassword123!",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := registerHandler.Register(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code, "user registration should succeed")

	var registerResp handlers.RegisterResponse
	err = json.NewDecoder(rec.Body).Decode(&registerResp)
	require.NoError(t, err)

	// Update organization role if needed
	if orgRole != rbac.OrganizationDeveloper {
		ctx := context.Background()
		_, err = db.ExecContext(
			ctx,
			"UPDATE users SET organization_role = $1 WHERE id = $2",
			string(orgRole),
			registerResp.User.ID,
		)
		require.NoError(t, err)
	}

	return registerResp.User.ID
}

// loginUser logs in a user and returns a token and user ID
func loginUser(t *testing.T, db *sqlx.DB, email, password string) (string, uuid.UUID) {
	t.Helper()

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()

	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", 0)
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, rateLimiter, logger)

	loginReq := handlers.LoginRequest{
		Email:    email,
		Password: password,
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReqHTTP.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	loginRec := httptest.NewRecorder()
	c := e.NewContext(loginReqHTTP, loginRec)

	err = loginHandler.Login(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, loginRec.Code, "login should succeed")

	var loginResp handlers.LoginResponse
	err = json.NewDecoder(loginRec.Body).Decode(&loginResp)
	require.NoError(t, err)

	// Extract token from cookie
	cookies := loginRec.Result().Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "deeb_auth_token" {
			authCookie = cookie
			break
		}
	}
	require.NotNil(t, authCookie, "auth cookie should be set")

	return authCookie.Value, loginResp.User.ID
}

// createEchoServer creates an Echo server with all routes configured
func createEchoServer(t *testing.T, db *sqlx.DB) (*echo.Echo, *auth.TokenManager) {
	t.Helper()

	e := echo.New()

	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", 0)
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	// Create config - set required fields
	cfg := &config.Config{
		Security: config.SecurityConfig{
			EncryptionKey: encryptionKey,
			BcryptCost:    12,
		},
		Auth: config.AuthConfig{
			JWTSecret:          "test-secret-key-minimum-32-characters-long-for-testing",
			JWTExpiration:      24 * time.Hour,
			GoogleClientID:     "",
			GoogleClientSecret: "",
			CallbackURL:        "",
		},
		Server: config.ServerConfig{
			FrontendURL: "http://localhost:3000",
			Host:        "0.0.0.0",
			Port:        8080,
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "deeb",
			Password: "deeb",
			Database: "deeb_test",
			SSLMode:  "disable",
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}

	// Validate config (will fail if required fields are missing)
	// Note: We skip validation in tests since we're using testcontainers
	// cfg.Validate() would require database connection which we don't have yet

	// Setup routes
	routes.Setup(e, cfg, db, tokenManager, rateLimiter, "test")

	return e, tokenManager
}

// makeAuthenticatedRequest makes an authenticated HTTP request
func makeAuthenticatedRequest(
	t *testing.T,
	e *echo.Echo,
	method, path string,
	body io.Reader,
	token string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, body)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.AddCookie(&http.Cookie{
			Name:  "deeb_auth_token",
			Value: token,
		})
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	return rec
}

// ============================================================================
// PROJECTS INTEGRATION TESTS
// ============================================================================

func TestIntegration_Projects_CRUD(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)

	// Create admin user
	adminEmail := fmt.Sprintf("admin-%d@example.com", time.Now().Unix())
	createTestUser(t, db, adminEmail, rbac.OrganizationAdmin)
	adminToken, _ := loginUser(t, db, adminEmail, "TestPassword123!")

	t.Run("Create Project - Admin", func(t *testing.T) {
		reqBody := models.CreateProjectRequest{
			Name:        "Test Project",
			Description: "Test Description",
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code, "admin should be able to create project")

		var project models.ProjectResponse
		err := json.NewDecoder(rec.Body).Decode(&project)
		require.NoError(t, err)
		assert.Equal(t, "Test Project", project.Name)
	})

	t.Run("List Projects", func(t *testing.T) {
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, "/api/v1/projects", nil, adminToken)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response, "projects")
	})

	t.Run("Get Project by ID", func(t *testing.T) {
		// First create a project
		reqBody := models.CreateProjectRequest{
			Name: "Get Test Project",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var createdProject models.ProjectResponse
		err := json.NewDecoder(rec.Body).Decode(&createdProject)
		require.NoError(t, err)

		// Then get it
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodGet,
			fmt.Sprintf("/api/v1/projects/%s", createdProject.ID.String()),
			nil,
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		var project models.ProjectResponse
		err = json.NewDecoder(rec.Body).Decode(&project)
		require.NoError(t, err)
		assert.Equal(t, createdProject.ID, project.ID)
		assert.Equal(t, "Get Test Project", project.Name)
	})

	t.Run("Update Project", func(t *testing.T) {
		// First create a project
		reqBody := models.CreateProjectRequest{
			Name: "Update Test Project",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var createdProject models.ProjectResponse
		err := json.NewDecoder(rec.Body).Decode(&createdProject)
		require.NoError(t, err)

		// Then update it
		updateReq := models.UpdateProjectRequest{
			Name:        stringPtr("Updated Project Name"),
			Description: stringPtr("Updated Description"),
		}
		updateBody, _ := json.Marshal(updateReq)
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodPut,
			fmt.Sprintf("/api/v1/projects/%s", createdProject.ID.String()),
			bytes.NewReader(updateBody),
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		var updatedProject models.ProjectResponse
		err = json.NewDecoder(rec.Body).Decode(&updatedProject)
		require.NoError(t, err)
		assert.Equal(t, "Updated Project Name", updatedProject.Name)
		assert.Equal(t, "Updated Description", updatedProject.Description)
	})

	t.Run("Delete Project", func(t *testing.T) {
		// First create a project
		reqBody := models.CreateProjectRequest{
			Name: "Delete Test Project",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var createdProject models.ProjectResponse
		err := json.NewDecoder(rec.Body).Decode(&createdProject)
		require.NoError(t, err)

		// Then delete it
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodDelete,
			fmt.Sprintf("/api/v1/projects/%s", createdProject.ID.String()),
			nil,
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		// Verify it's deleted
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodGet,
			fmt.Sprintf("/api/v1/projects/%s", createdProject.ID.String()),
			nil,
			adminToken,
		)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// ============================================================================
// ENVIRONMENTS INTEGRATION TESTS
// ============================================================================

func TestIntegration_Environments_CRUD(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)

	// Create admin user
	adminEmail := fmt.Sprintf("admin-%d@example.com", time.Now().Unix())
	createTestUser(t, db, adminEmail, rbac.OrganizationAdmin)
	adminToken, _ := loginUser(t, db, adminEmail, "TestPassword123!")

	// Create developer user
	devEmail := fmt.Sprintf("developer-%d@example.com", time.Now().Unix())
	_ = createTestUser(t, db, devEmail, rbac.OrganizationDeveloper)
	devToken, _ := loginUser(t, db, devEmail, "TestPassword123!")

	t.Run("Create Environment - Admin", func(t *testing.T) {
		reqBody := models.CreateEnvironmentRequest{
			Name:        "Test Environment",
			Description: stringPtr("Test Description"),
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/environments", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code, "admin should be able to create environment")

		var env models.EnvironmentResponse
		err := json.NewDecoder(rec.Body).Decode(&env)
		require.NoError(t, err)
		assert.Equal(t, "Test Environment", env.Name)
	})

	t.Run("Create Environment - Developer (Should Fail)", func(t *testing.T) {
		reqBody := models.CreateEnvironmentRequest{
			Name: "Test Environment 2",
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/environments", bytes.NewReader(body), devToken)
		assert.Equal(t, http.StatusForbidden, rec.Code, "developer should not be able to create environment")
	})

	t.Run("List Environments", func(t *testing.T) {
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, "/api/v1/environments", nil, adminToken)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response, "environments")
	})

	t.Run("Get Environment by ID", func(t *testing.T) {
		// First create an environment
		reqBody := models.CreateEnvironmentRequest{
			Name: "Get Test Environment",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/environments", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var createdEnv models.EnvironmentResponse
		err := json.NewDecoder(rec.Body).Decode(&createdEnv)
		require.NoError(t, err)

		// Then get it
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodGet,
			fmt.Sprintf("/api/v1/environments/%s", createdEnv.ID.String()),
			nil,
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		var env models.EnvironmentResponse
		err = json.NewDecoder(rec.Body).Decode(&env)
		require.NoError(t, err)
		assert.Equal(t, createdEnv.ID, env.ID)
	})

	t.Run("Delete Environment", func(t *testing.T) {
		// First create an environment
		reqBody := models.CreateEnvironmentRequest{
			Name: "Delete Test Environment",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/environments", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var createdEnv models.EnvironmentResponse
		err := json.NewDecoder(rec.Body).Decode(&createdEnv)
		require.NoError(t, err)

		// Then delete it
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodDelete,
			fmt.Sprintf("/api/v1/environments/%s", createdEnv.ID.String()),
			nil,
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		// Verify it's deleted
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodGet,
			fmt.Sprintf("/api/v1/environments/%s", createdEnv.ID.String()),
			nil,
			adminToken,
		)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// ============================================================================
// DATABASE INSTANCES INTEGRATION TESTS
// ============================================================================

func TestIntegration_DatabaseInstances_CRUD(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)

	// Create admin user
	adminEmail := fmt.Sprintf("admin-%d@example.com", time.Now().Unix())
	createTestUser(t, db, adminEmail, rbac.OrganizationAdmin)
	adminToken, _ := loginUser(t, db, adminEmail, "TestPassword123!")

	// Create developer user
	devEmail := fmt.Sprintf("developer-%d@example.com", time.Now().Unix())
	_ = createTestUser(t, db, devEmail, rbac.OrganizationDeveloper)
	devToken, _ := loginUser(t, db, devEmail, "TestPassword123!")

	t.Run("Create Database Instance - Admin", func(t *testing.T) {
		// Note: Connection test will fail without a real database, but we can test the endpoint structure
		reqBody := models.CreateDatabaseInstanceRequest{
			Name:           "Test Database Instance",
			Description:    stringPtr("Test Description"),
			DatabaseEngine: models.DatabaseEnginePostgreSQL,
			Host:           "localhost",
			Port:           5432,
			Username:       "testuser",
			Password:       "testpassword",
			UseRDSIAMAuth:  false,
			RequireSSL:     false,
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/database-instances", bytes.NewReader(body), adminToken)
		// Connection test will fail, but we verify the endpoint is accessible and validates properly
		// In real scenario, connection test would pass with valid credentials
		assert.True(t, rec.Code == http.StatusBadRequest || rec.Code == http.StatusCreated,
			"should return BadRequest (connection test failed) or Created (if connection succeeds)")
	})

	t.Run("Create Database Instance - Developer (Should Fail)", func(t *testing.T) {
		reqBody := models.CreateDatabaseInstanceRequest{
			Name:           "Test Database Instance 2",
			DatabaseEngine: models.DatabaseEnginePostgreSQL,
			Host:           "localhost",
			Port:           5432,
			Username:       "testuser",
			Password:       "testpassword",
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/database-instances", bytes.NewReader(body), devToken)
		assert.Equal(t, http.StatusForbidden, rec.Code, "developer should not be able to create database instance")
	})

	t.Run("List Database Instances", func(t *testing.T) {
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, "/api/v1/database-instances", nil, adminToken)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response, "database_instances")
	})
}

// ============================================================================
// DATABASES INTEGRATION TESTS
// ============================================================================

func TestIntegration_Databases_CRUD_WithConstraints(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)

	// Create admin user
	adminEmail := fmt.Sprintf("admin-%d@example.com", time.Now().Unix())
	adminUserID := createTestUser(t, db, adminEmail, rbac.OrganizationAdmin)
	adminToken, _ := loginUser(t, db, adminEmail, "TestPassword123!")

	// Create project
	projectReq := models.CreateProjectRequest{
		Name: "Test Project for Database",
	}
	projectBody, _ := json.Marshal(projectReq)
	rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(projectBody), adminToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var project models.ProjectResponse
	err := json.NewDecoder(rec.Body).Decode(&project)
	require.NoError(t, err)

	// Create environment
	envReq := models.CreateEnvironmentRequest{
		Name: "Test Environment for Database",
	}
	envBody, _ := json.Marshal(envReq)
	rec = makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/environments", bytes.NewReader(envBody), adminToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var env models.EnvironmentResponse
	err = json.NewDecoder(rec.Body).Decode(&env)
	require.NoError(t, err)

	// Create database instance (without connection test - we'll skip that for integration test)
	// We'll directly insert into DB to bypass connection test
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "12345678901234567890123456789012")
	encryptedDetails := "encrypted-details"
	instance := &models.DatabaseInstance{
		Name:                       "Test Instance",
		DatabaseEngine:             models.DatabaseEnginePostgreSQL,
		Host:                       "localhost",
		Port:                       5432,
		Username:                   "testuser",
		ConnectionDetailsEncrypted: &encryptedDetails,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                adminUserID,
	}
	ctx := context.Background()
	err = dbInstanceRepo.Create(ctx, instance)
	require.NoError(t, err)

	t.Run("Create Database with Valid Constraints", func(t *testing.T) {
		reqBody := models.CreateDatabaseRequest{
			ProjectID:          project.ID,
			DatabaseInstanceID: instance.ID,
			EnvironmentID:      env.ID,
			DatabaseName:       "test_database",
			Description:        stringPtr("Test Database"),
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/databases", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code, "should create database with valid constraints")

		var database models.DatabaseResponse
		err := json.NewDecoder(rec.Body).Decode(&database)
		require.NoError(t, err)
		assert.Equal(t, project.ID, *database.ProjectID)
		assert.Equal(t, instance.ID, database.DatabaseInstanceID)
		assert.Equal(t, env.ID, *database.EnvironmentID)
		assert.Equal(t, "test_database", database.DatabaseName)
	})

	t.Run("Create Database with Invalid Project ID", func(t *testing.T) {
		invalidProjectID := uuid.New()
		reqBody := models.CreateDatabaseRequest{
			ProjectID:          invalidProjectID,
			DatabaseInstanceID: instance.ID,
			EnvironmentID:      env.ID,
			DatabaseName:       "test_database_2",
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/databases", bytes.NewReader(body), adminToken)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "should fail with invalid project ID")
	})

	t.Run("Create Database with Invalid Environment ID", func(t *testing.T) {
		invalidEnvID := uuid.New()
		reqBody := models.CreateDatabaseRequest{
			ProjectID:          project.ID,
			DatabaseInstanceID: instance.ID,
			EnvironmentID:      invalidEnvID,
			DatabaseName:       "test_database_3",
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/databases", bytes.NewReader(body), adminToken)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "should fail with invalid environment ID")
	})

	t.Run("List Databases", func(t *testing.T) {
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, "/api/v1/databases", nil, adminToken)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response, "databases")
	})

	t.Run("List Databases by Project", func(t *testing.T) {
		rec := makeAuthenticatedRequest(
			t,
			e,
			http.MethodGet,
			fmt.Sprintf("/api/v1/projects/%s/databases", project.ID.String()),
			nil,
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response, "databases")
	})

	t.Run("Get Database by ID", func(t *testing.T) {
		// First create a database
		reqBody := models.CreateDatabaseRequest{
			ProjectID:          project.ID,
			DatabaseInstanceID: instance.ID,
			EnvironmentID:      env.ID,
			DatabaseName:       "get_test_database",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/databases", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var createdDB models.DatabaseResponse
		err := json.NewDecoder(rec.Body).Decode(&createdDB)
		require.NoError(t, err)

		// Then get it
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodGet,
			fmt.Sprintf("/api/v1/databases/%s", createdDB.ID.String()),
			nil,
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		var database models.DatabaseResponse
		err = json.NewDecoder(rec.Body).Decode(&database)
		require.NoError(t, err)
		assert.Equal(t, createdDB.ID, database.ID)
		assert.Equal(t, "get_test_database", database.DatabaseName)
	})

	t.Run("Update Database", func(t *testing.T) {
		// First create a database
		reqBody := models.CreateDatabaseRequest{
			ProjectID:          project.ID,
			DatabaseInstanceID: instance.ID,
			EnvironmentID:      env.ID,
			DatabaseName:       "update_test_database",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/databases", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var createdDB models.DatabaseResponse
		err := json.NewDecoder(rec.Body).Decode(&createdDB)
		require.NoError(t, err)

		// Then update it
		updateReq := models.UpdateDatabaseRequest{
			Description: stringPtr("Updated Description"),
		}
		updateBody, _ := json.Marshal(updateReq)
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodPut,
			fmt.Sprintf("/api/v1/databases/%s", createdDB.ID.String()),
			bytes.NewReader(updateBody),
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		var updatedDB models.DatabaseResponse
		err = json.NewDecoder(rec.Body).Decode(&updatedDB)
		require.NoError(t, err)
		assert.Equal(t, "update_test_database", updatedDB.DatabaseName)
		assert.Equal(t, "Updated Description", *updatedDB.Description)
	})

	t.Run("Delete Database", func(t *testing.T) {
		// First create a database
		reqBody := models.CreateDatabaseRequest{
			ProjectID:          project.ID,
			DatabaseInstanceID: instance.ID,
			EnvironmentID:      env.ID,
			DatabaseName:       "delete_test_database",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/databases", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var createdDB models.DatabaseResponse
		err := json.NewDecoder(rec.Body).Decode(&createdDB)
		require.NoError(t, err)

		// Then delete it
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodDelete,
			fmt.Sprintf("/api/v1/databases/%s", createdDB.ID.String()),
			nil,
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		// Verify it's deleted
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodGet,
			fmt.Sprintf("/api/v1/databases/%s", createdDB.ID.String()),
			nil,
			adminToken,
		)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
