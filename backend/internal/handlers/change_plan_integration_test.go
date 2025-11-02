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

