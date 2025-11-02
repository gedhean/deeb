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
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/handlers"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/migrate"
	"github.com/nsxbet/deeb/internal/repository/postgres"
	"github.com/nsxbet/deeb/internal/review"
	"github.com/nsxbet/deeb/internal/review/validators"
	"github.com/nsxbet/deeb/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

// setupTestDBForChangePlans sets up a test database connection using testcontainers
func setupTestDBForChangePlans(t *testing.T) (*sqlx.DB, func()) {
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

	// Run all migrations
	runAllMigrations(t, db, connStr)

	// Return cleanup function
	cleanup := func() {
		db.Close()
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	}

	return db, cleanup
}

// runAllMigrations runs all database migrations
func runAllMigrations(t *testing.T, db *sqlx.DB, connStr string) {
	t.Helper()

	migrator, err := migrate.New(migrate.Config{
		MigrationsPath: filepath.Join("..", "..", "migrations"),
		DatabaseURL:    connStr,
	})
	require.NoError(t, err, "failed to create migrator")

	defer func() {
		if closeErr := migrator.Close(); closeErr != nil {
			t.Logf("failed to close migrator: %v", closeErr)
		}
	}()

	err = migrator.Up()
	require.NoError(t, err, "failed to run migrations")
}

// cleanupChangePlanTestData cleans up test data
func cleanupChangePlanTestData(t *testing.T, db *sqlx.DB) {
	t.Helper()

	tables := []string{
		"policy_findings",
		"change_plans",
		"databases",
		"database_instances",
		"environments",
		"projects",
		"users",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("failed to clean up %s: %v", table, err)
		}
	}
}

// createTestUser creates a test user and returns it
func createTestUser(t *testing.T, userRepo *postgres.UserRepository, email string, role models.OrganizationRole) *models.User {
	t.Helper()

	ctx := context.Background()
	user := &models.User{
		Email:            email,
		PasswordHash:     "test_hash",
		OrganizationRole: role,
		IsActive:         true,
	}

	err := userRepo.Create(ctx, user)
	require.NoError(t, err, "failed to create test user")

	return user
}

// createTestProject creates a test project and returns it
func createTestProject(t *testing.T, projectRepo *postgres.ProjectRepository, name string) *models.Project {
	t.Helper()

	ctx := context.Background()
	project := &models.Project{
		Name:        name,
		Description: "Test project",
	}

	err := projectRepo.Create(ctx, project)
	require.NoError(t, err, "failed to create test project")

	return project
}

// createTestEnvironment creates a test environment
func createTestEnvironment(t *testing.T, envRepo *postgres.EnvironmentRepository, name string) *models.Environment {
	t.Helper()

	ctx := context.Background()
	env := &models.Environment{
		Name:        name,
		Description: "Test environment",
	}

	err := envRepo.Create(ctx, env)
	require.NoError(t, err, "failed to create test environment")

	return env
}

// createTestDatabaseInstance creates a test database instance
func createTestDatabaseInstance(t *testing.T, dbInstanceRepo *postgres.DatabaseInstanceRepository, name string, dbType models.DatabaseType) *models.DatabaseInstance {
	t.Helper()

	ctx := context.Background()
	dbInstance := &models.DatabaseInstance{
		Name:         name,
		DatabaseType: dbType,
		Host:         "localhost",
		Port:         5432,
		DatabaseName: "testdb",
		Username:     "testuser",
		Password:     "testpass",
	}

	err := dbInstanceRepo.Create(ctx, dbInstance)
	require.NoError(t, err, "failed to create test database instance")

	return dbInstance
}

// createTestDatabase creates a test database
func createTestDatabase(t *testing.T, dbRepo *postgres.DatabaseRepository, projectID, envID, dbInstanceID uuid.UUID, name string) *models.Database {
	t.Helper()

	ctx := context.Background()
	db := &models.Database{
		ProjectID:          projectID,
		EnvironmentID:      envID,
		DatabaseInstanceID: dbInstanceID,
		Name:               name,
		Description:        "Test database",
	}

	err := dbRepo.Create(ctx, db)
	require.NoError(t, err, "failed to create test database")

	return db
}

// createTestChangePlan creates a test change plan
func createTestChangePlan(t *testing.T, changePlanRepo *postgres.ChangePlanRepository, projectID, databaseID, authorID uuid.UUID, sqlContent string) *models.ChangePlan {
	t.Helper()

	ctx := context.Background()
	changePlan := &models.ChangePlan{
		ProjectID:        projectID,
		TargetDatabaseID: databaseID,
		AuthorID:         authorID,
		Title:            "Test Change Plan",
		Description:      "Test description",
		ChangeType:       models.ChangeTypeDDL,
		SQLContent:       sqlContent,
		Status:           models.ChangePlanStatusDraft,
		RiskLevel:        models.RiskLevelMedium,
	}

	err := changePlanRepo.Create(ctx, changePlan)
	require.NoError(t, err, "failed to create test change plan")

	return changePlan
}

// createAuthToken creates a JWT token for testing
func createAuthToken(t *testing.T, tokenManager *auth.TokenManager, userID uuid.UUID, email string, orgRole models.OrganizationRole) string {
	t.Helper()

	token, err := tokenManager.GenerateToken(userID, email, string(orgRole))
	require.NoError(t, err, "failed to generate token")

	return token
}

// setupChangePlanHandler creates a fully configured ChangePlanHandler with all dependencies
func setupChangePlanHandler(t *testing.T, db *sqlx.DB, tokenManager *auth.TokenManager) *handlers.ChangePlanHandler {
	t.Helper()

	logger := zap.NewNop()

	// Initialize repositories
	changePlanRepo := postgres.NewChangePlanRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	databaseRepo := postgres.NewDatabaseRepository(db)
	environmentRepo := postgres.NewEnvironmentRepository(db)
	databaseInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)
	policyRepo := postgres.NewPolicyRepository(db)

	// Initialize risk assessment service
	riskAssessmentSvc := services.NewRiskAssessmentService(
		databaseRepo,
		environmentRepo,
		databaseInstanceRepo,
		logger,
	)

	// Initialize SQL Review engine and service
	reviewEngine := review.NewEngine()
	reviewService := review.NewReviewService(
		policyRepo,
		databaseRepo,
		reviewEngine,
		logger,
	)

	// Register all validators
	require.NoError(t, reviewService.RegisterValidator(validators.NewPostgreSQLSyntaxValidator()))
	require.NoError(t, reviewService.RegisterValidator(validators.NewMySQLSyntaxValidator()))
	require.NoError(t, reviewService.RegisterValidator(validators.NewSecurityValidator()))
	require.NoError(t, reviewService.RegisterValidator(validators.NewPerformanceValidator()))
	require.NoError(t, reviewService.RegisterValidator(validators.NewBackwardsCompatibilityValidator()))
	require.NoError(t, reviewService.RegisterValidator(validators.NewBestPracticesValidator()))

	// Create handler
	handler := handlers.NewChangePlanHandler(
		changePlanRepo,
		projectRepo,
		databaseRepo,
		databaseInstanceRepo,
		projectMemberRepo,
		policyFindingRepo,
		riskAssessmentSvc,
		reviewService,
		logger,
	)

	return handler
}

// TestIntegration_ChangePlan_AutomaticSQLReview_Success tests that SQL Review is triggered automatically
// when transitioning from Draft to Open, and the transition succeeds when there are no errors
func TestIntegration_ChangePlan_AutomaticSQLReview_Success(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", time.Hour)
	require.NoError(t, err)

	// Create test data
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	projectMemberRepo := postgres.NewProjectMemberRepository(db)

	ctx := context.Background()

	// Create user
	user := createTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleAdmin)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add user as project member
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    user.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: user.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan with valid SQL (no errors, only warnings/info)
	validSQL := `
		CREATE TABLE users (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		);
	`
	changePlan := createTestChangePlan(t, changePlanRepo, project.ID, database.ID, user.ID, validSQL)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create update request to transition to Open
	statusOpen := models.ChangePlanStatusOpen
	updateReq := models.UpdateChangePlanRequest{
		Status: &statusOpen,
	}
	body, _ := json.Marshal(updateReq)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()), bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	token := createAuthToken(t, tokenManager, user.ID, user.Email, user.OrganizationRole)
	req.Header.Set("Authorization", "Bearer "+token)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Set user in Echo context (as auth middleware does)
	claims := &auth.Claims{
		UserID:           user.ID,
		Email:            user.Email,
		OrganizationRole: string(user.OrganizationRole),
		RegisteredClaims: jwt.RegisteredClaims{},
	}
	c.Set(string(auth.UserContextKey), claims)

	// Execute request
	err = handler.UpdateChangePlan(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code, "should successfully transition to Open")

	// Verify change plan status is Open
	updatedChangePlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusOpen, updatedChangePlan.Status, "change plan should be in Open status")

	// Verify policy findings were created
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)
	findings, err := policyFindingRepo.GetByChangePlanID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Greater(t, len(findings), 0, "should have policy findings")

	// Verify no errors (only warnings/info allowed)
	severityCounts, err := policyFindingRepo.CountBySeverity(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, severityCounts["Error"], "should have no errors")
	assert.Greater(t, severityCounts["Warning"]+severityCounts["Info"], 0, "should have warnings or info")
}

// TestIntegration_ChangePlan_AutomaticSQLReview_ErrorsBlockTransition tests that SQL Review errors
// prevent the transition from Draft to Open
func TestIntegration_ChangePlan_AutomaticSQLReview_ErrorsBlockTransition(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", time.Hour)
	require.NoError(t, err)

	// Create test data
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	projectMemberRepo := postgres.NewProjectMemberRepository(db)

	ctx := context.Background()

	// Create user
	user := createTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleAdmin)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add user as project member
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    user.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: user.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan with SQL that triggers errors (DROP TABLE)
	invalidSQL := `
		DROP TABLE users;
	`
	changePlan := createTestChangePlan(t, changePlanRepo, project.ID, database.ID, user.ID, invalidSQL)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create update request to transition to Open
	statusOpen := models.ChangePlanStatusOpen
	updateReq := models.UpdateChangePlanRequest{
		Status: &statusOpen,
	}
	body, _ := json.Marshal(updateReq)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()), bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	token := createAuthToken(t, tokenManager, user.ID, user.Email, user.OrganizationRole)
	req.Header.Set("Authorization", "Bearer "+token)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Set user in Echo context (as auth middleware does)
	claims := &auth.Claims{
		UserID:           user.ID,
		Email:            user.Email,
		OrganizationRole: string(user.OrganizationRole),
		RegisteredClaims: jwt.RegisteredClaims{},
	}
	c.Set(string(auth.UserContextKey), claims)

	// Execute request
	err = handler.UpdateChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "should fail to transition to Open due to errors")

	// Verify change plan status is still Draft
	updatedChangePlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusDraft, updatedChangePlan.Status, "change plan should still be in Draft status")

	// Verify policy findings were created
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)
	findings, err := policyFindingRepo.GetByChangePlanID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Greater(t, len(findings), 0, "should have policy findings")

	// Verify errors exist
	severityCounts, err := policyFindingRepo.CountBySeverity(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Greater(t, severityCounts["Error"], 0, "should have errors")

	// Verify error response
	var errorResp map[string]interface{}
	err = json.NewDecoder(rec.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"].(string), "SQL Review failed", "error message should mention SQL Review")
}

// TestIntegration_ChangePlan_AutomaticSQLReview_OnlyWarnings tests that warnings don't block transition
func TestIntegration_ChangePlan_AutomaticSQLReview_OnlyWarnings(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", time.Hour)
	require.NoError(t, err)

	// Create test data
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	projectMemberRepo := postgres.NewProjectMemberRepository(db)

	ctx := context.Background()

	// Create user
	user := createTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleAdmin)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add user as project member
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    user.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: user.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan with SQL that triggers warnings but no errors
	// (e.g., missing primary key, missing indexes)
	warningSQL := `
		CREATE TABLE users (
			id UUID,
			name VARCHAR(255)
		);
	`
	changePlan := createTestChangePlan(t, changePlanRepo, project.ID, database.ID, user.ID, warningSQL)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create update request to transition to Open
	statusOpen := models.ChangePlanStatusOpen
	updateReq := models.UpdateChangePlanRequest{
		Status: &statusOpen,
	}
	body, _ := json.Marshal(updateReq)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()), bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	token := createAuthToken(t, tokenManager, user.ID, user.Email, user.OrganizationRole)
	req.Header.Set("Authorization", "Bearer "+token)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Set user in Echo context (as auth middleware does)
	claims := &auth.Claims{
		UserID:           user.ID,
		Email:            user.Email,
		OrganizationRole: string(user.OrganizationRole),
		RegisteredClaims: jwt.RegisteredClaims{},
	}
	c.Set(string(auth.UserContextKey), claims)

	// Execute request
	err = handler.UpdateChangePlan(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code, "should successfully transition to Open with only warnings")

	// Verify change plan status is Open
	updatedChangePlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusOpen, updatedChangePlan.Status, "change plan should be in Open status")

	// Verify policy findings were created
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)
	findings, err := policyFindingRepo.GetByChangePlanID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Greater(t, len(findings), 0, "should have policy findings")

	// Verify no errors (only warnings/info)
	severityCounts, err := policyFindingRepo.CountBySeverity(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, severityCounts["Error"], "should have no errors")
	assert.Greater(t, severityCounts["Warning"]+severityCounts["Info"], 0, "should have warnings or info")
}

// Helper function to read response body
func readResponseBody(t *testing.T, body io.ReadCloser) string {
	t.Helper()
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	return string(data)
}

// TestIntegration_ChangePlan_ManualSQLReview tests the manual SQL Review trigger endpoint
func TestIntegration_ChangePlan_ManualSQLReview(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", time.Hour)
	require.NoError(t, err)

	// Create test data
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	projectMemberRepo := postgres.NewProjectMemberRepository(db)

	ctx := context.Background()

	// Create user
	user := createTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleAdmin)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add user as project member
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    user.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: user.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan with SQL that triggers warnings
	warningSQL := `
		CREATE TABLE users (
			id UUID,
			name VARCHAR(255)
		);
	`
	changePlan := createTestChangePlan(t, changePlanRepo, project.ID, database.ID, user.ID, warningSQL)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/change-plans/%s/review", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	token := createAuthToken(t, tokenManager, user.ID, user.Email, user.OrganizationRole)
	req.Header.Set("Authorization", "Bearer "+token)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Set user in Echo context (as auth middleware does)
	claims := &auth.Claims{
		UserID:           user.ID,
		Email:            user.Email,
		OrganizationRole: string(user.OrganizationRole),
		RegisteredClaims: jwt.RegisteredClaims{},
	}
	c.Set(string(auth.UserContextKey), claims)

	// Execute request
	err = handler.TriggerSQLReview(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code, "should successfully trigger SQL Review")

	// Verify response contains review results
	var response map[string]interface{}
	err = json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotNil(t, response["review_result"], "should have review_result in response")

	reviewResult := response["review_result"].(map[string]interface{})
	assert.NotNil(t, reviewResult["errors"], "should have errors count")
	assert.NotNil(t, reviewResult["warnings"], "should have warnings count")
	assert.NotNil(t, reviewResult["results"], "should have results array")

	// Verify policy findings were created
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)
	findings, err := policyFindingRepo.GetByChangePlanID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Greater(t, len(findings), 0, "should have policy findings")

	// Verify severity counts
	severityCounts, err := policyFindingRepo.CountBySeverity(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Greater(t, severityCounts["Warning"]+severityCounts["Info"], 0, "should have warnings or info")
}

// TestIntegration_ChangePlan_ManualSQLReview_Unauthorized tests that unauthorized users cannot trigger review
func TestIntegration_ChangePlan_ManualSQLReview_Unauthorized(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", time.Hour)
	require.NoError(t, err)

	// Create test data
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	ctx := context.Background()

	// Create user (not a project member)
	user := createTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan
	changePlan := createTestChangePlan(t, changePlanRepo, project.ID, database.ID, user.ID, "CREATE TABLE test (id UUID);")

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/change-plans/%s/review", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	token := createAuthToken(t, tokenManager, user.ID, user.Email, user.OrganizationRole)
	req.Header.Set("Authorization", "Bearer "+token)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Set user in Echo context (as auth middleware does)
	claims := &auth.Claims{
		UserID:           user.ID,
		Email:            user.Email,
		OrganizationRole: string(user.OrganizationRole),
		RegisteredClaims: jwt.RegisteredClaims{},
	}
	c.Set(string(auth.UserContextKey), claims)

	// Execute request
	err = handler.TriggerSQLReview(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code, "should return 403 Forbidden")

	// Verify error message
	var errorResp map[string]interface{}
	err = json.NewDecoder(rec.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"].(string), "Not a member", "error message should mention project membership")
}

// TestIntegration_ChangePlan_ListWithFilters tests filtering functionality
func TestIntegration_ChangePlan_ListWithFilters(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", time.Hour)
	require.NoError(t, err)

	// Create test data
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	projectMemberRepo := postgres.NewProjectMemberRepository(db)

	ctx := context.Background()

	// Create user
	user := createTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleAdmin)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add user as project member
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    user.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: user.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create multiple change plans with different statuses
	cp1 := createTestChangePlan(t, changePlanRepo, project.ID, database.ID, user.ID, "CREATE TABLE test1 (id UUID);")
	cp1.Status = models.ChangePlanStatusDraft
	changePlanRepo.Update(ctx, cp1)

	cp2 := createTestChangePlan(t, changePlanRepo, project.ID, database.ID, user.ID, "CREATE TABLE test2 (id UUID);")
	cp2.Status = models.ChangePlanStatusOpen
	changePlanRepo.Update(ctx, cp2)

	cp3 := createTestChangePlan(t, changePlanRepo, project.ID, database.ID, user.ID, "CREATE TABLE test3 (id UUID);")
	cp3.Status = models.ChangePlanStatusOpen
	changePlanRepo.Update(ctx, cp3)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Test 1: Filter by status
	req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans?status=Open", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	token := createAuthToken(t, tokenManager, user.ID, user.Email, user.OrganizationRole)
	req.Header.Set("Authorization", "Bearer "+token)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	claims := &auth.Claims{
		UserID:           user.ID,
		Email:            user.Email,
		OrganizationRole: string(user.OrganizationRole),
		RegisteredClaims: jwt.RegisteredClaims{},
	}
	c.Set(string(auth.UserContextKey), claims)

	err = handler.ListChangePlans(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotNil(t, response["change_plans"])
	changePlans := response["change_plans"].([]interface{})
	assert.Equal(t, 2, len(changePlans), "should return 2 Open change plans")

	// Test 2: Filter by project_id
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/change-plans?project_id=%s", project.ID.String()), nil)
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req2.Header.Set("Authorization", "Bearer "+token)

	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.Set(string(auth.UserContextKey), claims)

	err = handler.ListChangePlans(c2)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec2.Code)

	var response2 map[string]interface{}
	err = json.NewDecoder(rec2.Body).Decode(&response2)
	require.NoError(t, err)
	assert.NotNil(t, response2["change_plans"])
	changePlans2 := response2["change_plans"].([]interface{})
	assert.Equal(t, 3, len(changePlans2), "should return 3 change plans for project")

	// Test 3: Combined filters (status + project_id)
	req3 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/change-plans?status=Draft&project_id=%s", project.ID.String()), nil)
	req3.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req3.Header.Set("Authorization", "Bearer "+token)

	rec3 := httptest.NewRecorder()
	c3 := e.NewContext(req3, rec3)
	c3.Set(string(auth.UserContextKey), claims)

	err = handler.ListChangePlans(c3)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec3.Code)

	var response3 map[string]interface{}
	err = json.NewDecoder(rec3.Body).Decode(&response3)
	require.NoError(t, err)
	assert.NotNil(t, response3["change_plans"])
	changePlans3 := response3["change_plans"].([]interface{})
	assert.Equal(t, 1, len(changePlans3), "should return 1 Draft change plan")

	// Test 4: Filter by database_id
	req4 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/change-plans?database_id=%s", database.ID.String()), nil)
	req4.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req4.Header.Set("Authorization", "Bearer "+token)

	rec4 := httptest.NewRecorder()
	c4 := e.NewContext(req4, rec4)
	c4.Set(string(auth.UserContextKey), claims)

	err = handler.ListChangePlans(c4)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec4.Code)

	var response4 map[string]interface{}
	err = json.NewDecoder(rec4.Body).Decode(&response4)
	require.NoError(t, err)
	assert.NotNil(t, response4["change_plans"])
	changePlans4 := response4["change_plans"].([]interface{})
	assert.Equal(t, 3, len(changePlans4), "should return 3 change plans for database")

	// Test 5: Filter by author_id
	req5 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/change-plans?author_id=%s", user.ID.String()), nil)
	req5.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req5.Header.Set("Authorization", "Bearer "+token)

	rec5 := httptest.NewRecorder()
	c5 := e.NewContext(req5, rec5)
	c5.Set(string(auth.UserContextKey), claims)

	err = handler.ListChangePlans(c5)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec5.Code)

	var response5 map[string]interface{}
	err = json.NewDecoder(rec5.Body).Decode(&response5)
	require.NoError(t, err)
	assert.NotNil(t, response5["change_plans"])
	changePlans5 := response5["change_plans"].([]interface{})
	assert.Equal(t, 3, len(changePlans5), "should return 3 change plans by author")

	// Test 6: Combined filters (status + database_id + author_id)
	req6 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/change-plans?status=Open&database_id=%s&author_id=%s", database.ID.String(), user.ID.String()), nil)
	req6.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req6.Header.Set("Authorization", "Bearer "+token)

	rec6 := httptest.NewRecorder()
	c6 := e.NewContext(req6, rec6)
	c6.Set(string(auth.UserContextKey), claims)

	err = handler.ListChangePlans(c6)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec6.Code)

	var response6 map[string]interface{}
	err = json.NewDecoder(rec6.Body).Decode(&response6)
	require.NoError(t, err)
	assert.NotNil(t, response6["change_plans"])
	changePlans6 := response6["change_plans"].([]interface{})
	assert.Equal(t, 2, len(changePlans6), "should return 2 Open change plans matching all filters")

	// Test 7: Pagination with filters
	req7 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/change-plans?status=Open&limit=1&offset=0", database.ID.String()), nil)
	req7.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req7.Header.Set("Authorization", "Bearer "+token)

	rec7 := httptest.NewRecorder()
	c7 := e.NewContext(req7, rec7)
	c7.Set(string(auth.UserContextKey), claims)

	err = handler.ListChangePlans(c7)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec7.Code)

	var response7 map[string]interface{}
	err = json.NewDecoder(rec7.Body).Decode(&response7)
	require.NoError(t, err)
	assert.NotNil(t, response7["change_plans"])
	changePlans7 := response7["change_plans"].([]interface{})
	assert.Equal(t, 1, len(changePlans7), "should return 1 change plan with limit=1")
	assert.Equal(t, float64(2), response7["total"], "total should be 2")
	assert.Equal(t, float64(1), response7["limit"], "limit should be 1")
	assert.Equal(t, float64(0), response7["offset"], "offset should be 0")
}

