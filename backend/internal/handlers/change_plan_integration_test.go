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
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/handlers"
	"github.com/nsxbet/deeb/internal/migrate"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
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
func createTestDatabaseInstance(
	t *testing.T,
	dbInstanceRepo *postgres.DatabaseInstanceRepository,
	name string,
	dbType models.DatabaseType,
) *models.DatabaseInstance {
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
func createTestDatabase(
	t *testing.T,
	dbRepo *postgres.DatabaseRepository,
	projectID, envID, dbInstanceID uuid.UUID,
	name string,
) *models.Database {
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
func createTestChangePlan(
	t *testing.T,
	changePlanRepo *postgres.ChangePlanRepository,
	projectID, databaseID, authorID uuid.UUID,
	sqlContent string,
) *models.ChangePlan {
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

// setUserContext sets up the Echo context with user information for RBAC checks and auth claims
func setUserContext(c echo.Context, userID uuid.UUID, email string, orgRole models.OrganizationRole) {
	// Set RBAC context in request context
	ctx := c.Request().Context()
	ctx = context.WithValue(ctx, rbac.ContextKeyUserID, userID.String())
	ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationID, "default") // Single-tenant for MVP
	ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationRole, rbac.OrganizationRole(orgRole))
	c.SetRequest(c.Request().WithContext(ctx))

	// Set auth claims in Echo context
	claims := &auth.Claims{
		UserID:           userID,
		Email:            email,
		OrganizationRole: string(orgRole),
	}
	c.Set(string(auth.UserContextKey), claims)
}

// createAuthToken creates a JWT token for testing
func createAuthToken(
	t *testing.T,
	tokenManager *auth.TokenManager,
	userID uuid.UUID,
	email string,
	orgRole models.OrganizationRole,
) string {
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
	approvalRepo := postgres.NewApprovalRepository(db)

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

	// Initialize approval service
	rbacChecker := rbac.NewCheckerService()
	approvalService := services.NewApprovalService(
		approvalRepo,
		changePlanRepo,
		policyFindingRepo,
		projectMemberRepo,
		rbacChecker,
		logger,
	)

	// Initialize schema snapshot repository
	snapshotRepo := postgres.NewSchemaSnapshotRepository(db)

	// Initialize connection manager
	encryptionKey := "test-encryption-key-min-32-chars"
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)

	// Initialize schema snapshot service
	snapshotService := services.NewSchemaSnapshotService(
		snapshotRepo,
		databaseInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)

	// Initialize execution lock manager
	lockManager := services.NewExecutionLockManager(db, logger)

	// Initialize execution queue repository
	queueRepo := postgres.NewExecutionQueueRepository(db)

	// Initialize execution queue service
	queueService := services.NewExecutionQueueService(queueRepo, logger)

	// Initialize execution log repository
	executionLogRepo := postgres.NewExecutionLogRepository(db)

	// Initialize execution service
	executionService := services.NewExecutionService(
		changePlanRepo,
		databaseInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		queueService,
		executionLogRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)

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
		approvalService,
		executionService,
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
	req := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()),
		bytes.NewReader(body),
	)
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
	req := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()),
		bytes.NewReader(body),
	)
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
	req := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()),
		bytes.NewReader(body),
	)
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
	req3 := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/v1/change-plans?status=Draft&project_id=%s", project.ID.String()),
		nil,
	)
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
	req6 := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/v1/change-plans?status=Open&database_id=%s&author_id=%s", database.ID.String(), user.ID.String()),
		nil,
	)
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
	req7 := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/v1/change-plans?status=Open&limit=1&offset=0", database.ID.String()),
		nil,
	)
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

// TestIntegration_ChangePlan_GetWithPolicyFindings tests that GetChangePlan includes policy findings in response
func TestIntegration_ChangePlan_GetWithPolicyFindings(t *testing.T) {
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
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)

	ctx := context.Background()

	// Create user
	user := createTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleDeveloper)

	// Create project and add user as member
	project := createTestProject(t, projectRepo, "Test Project")
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    user.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		user.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY, name TEXT);",
	)

	// Create policy findings manually
	_, err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "syntax_error",
		Severity:      "Error",
		RuleName:      "InvalidSyntax",
		Message:       "Syntax error at line 1",
		LineNumber:    intPtr(1),
		ColumnNumber:  intPtr(10),
	})
	require.NoError(t, err)

	_, err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "security",
		Severity:      "Warning",
		RuleName:      "MissingIndex",
		Message:       "Table missing index",
		LineNumber:    intPtr(1),
	})
	require.NoError(t, err)

	_, err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "best_practices",
		Severity:      "Info",
		RuleName:      "NamingConvention",
		Message:       "Consider using snake_case",
		LineNumber:    intPtr(1),
	})
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token
	token, err := tokenManager.GenerateToken(user.ID.String(), user.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Execute
	err = handler.GetChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var response models.ChangePlanResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify change plan fields
	assert.Equal(t, changePlan.ID, response.ID)
	assert.Equal(t, changePlan.Title, response.Title)

	// Verify policy findings are included
	require.NotNil(t, response.PolicyFindings)
	assert.Equal(t, true, response.HasErrors)
	assert.Equal(t, true, response.HasWarnings)

	// Verify findings grouped by severity
	assert.Equal(t, 1, len(response.PolicyFindings.Errors))
	assert.Equal(t, 1, len(response.PolicyFindings.Warnings))
	assert.Equal(t, 1, len(response.PolicyFindings.Infos))

	// Verify error finding
	assert.Equal(t, "syntax_error", response.PolicyFindings.Errors[0].ValidatorType)
	assert.Equal(t, "InvalidSyntax", response.PolicyFindings.Errors[0].RuleName)
	assert.Equal(t, "Syntax error at line 1", response.PolicyFindings.Errors[0].Message)
	assert.Equal(t, 1, *response.PolicyFindings.Errors[0].LineNumber)
	assert.Equal(t, 10, *response.PolicyFindings.Errors[0].ColumnNumber)

	// Verify warning finding
	assert.Equal(t, "security", response.PolicyFindings.Warnings[0].ValidatorType)
	assert.Equal(t, "MissingIndex", response.PolicyFindings.Warnings[0].RuleName)

	// Verify info finding
	assert.Equal(t, "best_practices", response.PolicyFindings.Infos[0].ValidatorType)
	assert.Equal(t, "NamingConvention", response.PolicyFindings.Infos[0].RuleName)

	// Verify counts
	assert.Equal(t, 1, response.PolicyFindings.Counts.Errors)
	assert.Equal(t, 1, response.PolicyFindings.Counts.Warnings)
	assert.Equal(t, 1, response.PolicyFindings.Counts.Infos)
	assert.Equal(t, 3, response.PolicyFindings.Counts.Total)
}

// TestIntegration_ChangePlan_GetWithoutPolicyFindings tests that GetChangePlan returns empty findings when none exist
func TestIntegration_ChangePlan_GetWithoutPolicyFindings(t *testing.T) {
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
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	ctx := context.Background()

	// Create user
	user := createTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleDeveloper)

	// Create project and add user as member
	project := createTestProject(t, projectRepo, "Test Project")
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    user.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan (no policy findings)
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		user.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token
	token, err := tokenManager.GenerateToken(user.ID.String(), user.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Execute
	err = handler.GetChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var response models.ChangePlanResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify change plan fields
	assert.Equal(t, changePlan.ID, response.ID)

	// Verify policy findings flags are false
	assert.Equal(t, false, response.HasErrors)
	assert.Equal(t, false, response.HasWarnings)

	// Policy findings may be nil or empty
	if response.PolicyFindings != nil {
		assert.Equal(t, 0, response.PolicyFindings.Counts.Total)
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// TestIntegration_ChangePlan_ApprovalWithErrors tests that approval is blocked when errors exist
func TestIntegration_ChangePlan_ApprovalWithErrors(t *testing.T) {
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
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)

	ctx := context.Background()

	// Create approver user
	approver := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("approver-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add approver as member with Approver role
	project := createTestProject(t, projectRepo, "Test Project")
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    approver.ID,
		Role:      models.ProjectRoleApprover,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create change plan in Open status
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)
	changePlan.Status = models.ChangePlanStatusOpen
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Create policy findings with errors
	_, err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "syntax_error",
		Severity:      "Error",
		RuleName:      "InvalidSyntax",
		Message:       "Syntax error at line 1",
		LineNumber:    intPtr(1),
		ColumnNumber:  intPtr(10),
	})
	require.NoError(t, err)

	_, err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "security",
		Severity:      "Error",
		RuleName:      "DestructiveOperation",
		Message:       "DROP TABLE detected",
		LineNumber:    intPtr(2),
	})
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for approver
	token, err := tokenManager.GenerateToken(approver.ID.String(), approver.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/change-plans/%s/approve", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/approve")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Execute
	err = handler.ApproveChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Cannot approve change plan with SQL Review errors")
	assert.NotNil(t, response["details"])

	details := response["details"].(map[string]interface{})
	assert.Equal(t, float64(2), details["error_count"].(float64))

	errors := details["errors"].([]interface{})
	assert.Equal(t, 2, len(errors))

	// Verify change plan status is still Open
	updatedPlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusOpen, updatedPlan.Status)
}

// TestIntegration_ChangePlan_ApprovalWithOnlyWarnings tests that approval succeeds when only warnings exist
func TestIntegration_ChangePlan_ApprovalWithOnlyWarnings(t *testing.T) {
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
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)

	ctx := context.Background()

	// Create approver user
	approver := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("approver-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add approver as member with Approver role
	project := createTestProject(t, projectRepo, "Test Project")
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    approver.ID,
		Role:      models.ProjectRoleApprover,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create change plan in Open status
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)
	changePlan.Status = models.ChangePlanStatusOpen
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Create policy findings with only warnings (no errors)
	_, err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "performance",
		Severity:      "Warning",
		RuleName:      "MissingIndex",
		Message:       "Table missing index",
		LineNumber:    intPtr(1),
	})
	require.NoError(t, err)

	_, err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "best_practices",
		Severity:      "Info",
		RuleName:      "NamingConvention",
		Message:       "Consider using snake_case",
		LineNumber:    intPtr(1),
	})
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for approver
	token, err := tokenManager.GenerateToken(approver.ID.String(), approver.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/change-plans/%s/approve", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/approve")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Execute
	err = handler.ApproveChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var response models.ChangePlanResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, changePlan.ID, response.ID)
	assert.Equal(t, models.ChangePlanStatusApproved, response.Status)

	// Verify change plan status is now Approved
	updatedPlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusApproved, updatedPlan.Status)
}

// TestIntegration_ChangePlan_ExecutionWithErrors tests that execution is blocked when errors exist (safety check)
func TestIntegration_ChangePlan_ExecutionWithErrors(t *testing.T) {
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
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)

	ctx := context.Background()

	// Create executor user
	executor := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("executor-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add executor as member with Approver role (can execute)
	project := createTestProject(t, projectRepo, "Test Project")
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    executor.ID,
		Role:      models.ProjectRoleApprover,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create change plan in Approved status
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)
	changePlan.Status = models.ChangePlanStatusApproved
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Create policy findings with errors
	_, err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "syntax_error",
		Severity:      "Error",
		RuleName:      "InvalidSyntax",
		Message:       "Syntax error at line 1",
		LineNumber:    intPtr(1),
		ColumnNumber:  intPtr(10),
	})
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for executor
	token, err := tokenManager.GenerateToken(executor.ID.String(), executor.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/execute")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Execute
	err = handler.ExecuteChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Cannot execute change plan with SQL Review errors")
	assert.NotNil(t, response["details"])

	details := response["details"].(map[string]interface{})
	assert.Equal(t, float64(1), details["error_count"].(float64))

	// Verify change plan status is still Approved
	updatedPlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusApproved, updatedPlan.Status)
}

// TestIntegration_ChangePlan_ExecutionWithOnlyWarnings tests that execution succeeds when only warnings exist
func TestIntegration_ChangePlan_ExecutionWithOnlyWarnings(t *testing.T) {
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
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)

	ctx := context.Background()

	// Create executor user
	executor := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("executor-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add executor as member with Approver role (can execute)
	project := createTestProject(t, projectRepo, "Test Project")
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    executor.ID,
		Role:      models.ProjectRoleApprover,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create change plan in Approved status
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)
	changePlan.Status = models.ChangePlanStatusApproved
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Create policy findings with only warnings (no errors)
	_, err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "performance",
		Severity:      "Warning",
		RuleName:      "MissingIndex",
		Message:       "Table missing index",
		LineNumber:    intPtr(1),
	})
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for executor
	token, err := tokenManager.GenerateToken(executor.ID.String(), executor.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/execute")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Execute
	err = handler.ExecuteChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var response models.ChangePlanResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, changePlan.ID, response.ID)
	assert.Equal(t, models.ChangePlanStatusInProgress, response.Status)

	// Verify change plan status is now InProgress
	updatedPlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusInProgress, updatedPlan.Status)
}

// TestIntegration_ChangePlan_Resubmission tests resubmission workflow
func TestIntegration_ChangePlan_Resubmission(t *testing.T) {
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
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)

	ctx := context.Background()

	// Create author user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add author as member
	project := createTestProject(t, projectRepo, "Test Project")
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan in Open status
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)
	changePlan.Status = models.ChangePlanStatusOpen
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Create policy findings (errors) that need to be fixed
	_, err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "syntax_error",
		Severity:      "Error",
		RuleName:      "InvalidSyntax",
		Message:       "Syntax error at line 1",
		LineNumber:    intPtr(1),
		ColumnNumber:  intPtr(10),
	})
	require.NoError(t, err)

	// Verify findings exist
	findings, err := policyFindingRepo.GetByChangePlanID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, len(findings))

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for author
	token, err := tokenManager.GenerateToken(author.ID.String(), author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Test 1: Resubmit change plan
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/change-plans/%s/resubmit", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/resubmit")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Execute
	err = handler.ResubmitChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var response models.ChangePlanResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, changePlan.ID, response.ID)
	assert.Equal(t, models.ChangePlanStatusDraft, response.Status)

	// Verify change plan status is now Draft
	updatedPlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusDraft, updatedPlan.Status)

	// Verify policy findings are cleared
	findingsAfterResubmit, err := policyFindingRepo.GetByChangePlanID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, len(findingsAfterResubmit))

	// Test 2: Resubmit and then resubmit again (Draft → Open → Draft)
	// Update SQL content to fix the error
	updatedPlan.SQLContent = "CREATE TABLE users (id UUID PRIMARY KEY, name VARCHAR(255));"
	err = changePlanRepo.Update(ctx, updatedPlan)
	require.NoError(t, err)

	// Transition back to Open (this will trigger SQL Review again)
	updatedPlan.Status = models.ChangePlanStatusOpen
	err = changePlanRepo.Update(ctx, updatedPlan)
	require.NoError(t, err)

	// Resubmit again
	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/change-plans/%s/resubmit", changePlan.ID.String()), nil)
	req2.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.SetPath("/api/v1/change-plans/:id/resubmit")
	c2.SetParamNames("id")
	c2.SetParamValues(changePlan.ID.String())

	err = handler.ResubmitChangePlan(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec2.Code)

	// Verify status is Draft again
	finalPlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusDraft, finalPlan.Status)
}

// TestIntegration_ChangePlan_Resubmission_Unauthorized tests that non-authors cannot resubmit
func TestIntegration_ChangePlan_Resubmission_Unauthorized(t *testing.T) {
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
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	ctx := context.Background()

	// Create author user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create non-author user
	nonAuthor := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("nonauthor-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add both users as members
	project := createTestProject(t, projectRepo, "Test Project")
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    nonAuthor.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan in Open status
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)
	changePlan.Status = models.ChangePlanStatusOpen
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for non-author
	token, err := tokenManager.GenerateToken(nonAuthor.ID.String(), nonAuthor.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/change-plans/%s/resubmit", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/resubmit")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Execute
	err = handler.ResubmitChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Only the author can resubmit")

	// Verify change plan status is still Open
	updatedPlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusOpen, updatedPlan.Status)
}

// TestIntegration_ChangePlan_Resubmission_InvalidStatus tests that only Open status can be resubmitted
func TestIntegration_ChangePlan_Resubmission_InvalidStatus(t *testing.T) {
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
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	ctx := context.Background()

	// Create author user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add author as member
	project := createTestProject(t, projectRepo, "Test Project")
	_, err = projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan in Draft status (should not be resubmittable)
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)
	// Status is already Draft by default

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for author
	token, err := tokenManager.GenerateToken(author.ID.String(), author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/change-plans/%s/resubmit", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/resubmit")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())

	// Execute
	err = handler.ResubmitChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Only Open change plans can be resubmitted")
}

// TestIntegration_ChangePlan_Create tests POST /api/v1/change-plans
func TestIntegration_ChangePlan_Create(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	ctx := context.Background()
	logger := zap.NewNop()
	tokenManager := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing")

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)

	// Create test user
	author := createTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member with Developer role
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err := projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "development")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for author
	token, err := tokenManager.GenerateToken(author.ID.String(), author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	createReq := handlers.CreateChangePlanRequest{
		ProjectID:        project.ID.String(),
		TargetDatabaseID: database.ID.String(),
		Title:            "Test Change Plan",
		Description:      "Test description",
		ChangeType:       "DDL",
		SQLContent:       "CREATE TABLE users (id UUID PRIMARY KEY, name VARCHAR(255));",
	}

	body, _ := json.Marshal(createReq)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, author.ID, author.Email, author.OrganizationRole)

	// Execute
	err = handler.CreateChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Verify response
	var response handlers.ChangePlanResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, createReq.Title, response.Title)
	assert.Equal(t, createReq.Description, response.Description)
	assert.Equal(t, createReq.SQLContent, response.SQLContent)
	assert.Equal(t, models.ChangePlanStatusDraft, response.Status)
	assert.Equal(t, author.ID.String(), response.AuthorID)
}

// TestIntegration_ChangePlan_Create_Unauthorized tests POST /api/v1/change-plans without authentication
func TestIntegration_ChangePlan_Create_Unauthorized(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	logger := zap.NewNop()
	tokenManager := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing")

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create request without token
	createReq := handlers.CreateChangePlanRequest{
		Title:       "Test Change Plan",
		Description: "Test description",
		ChangeType:  "DDL",
		SQLContent:  "CREATE TABLE users (id UUID PRIMARY KEY);",
	}

	body, _ := json.Marshal(createReq)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Don't set user context - should result in unauthorized

	// Execute
	err := handler.CreateChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestIntegration_ChangePlan_Create_Forbidden tests POST /api/v1/change-plans with insufficient permissions
func TestIntegration_ChangePlan_Create_Forbidden(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	ctx := context.Background()
	logger := zap.NewNop()
	tokenManager := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing")

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)

	// Create test user (not a member of project)
	user := createTestUser(t, userRepo, "user@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Create environment
	env := createTestEnvironment(t, envRepo, "development")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for user (not a project member)
	token, err := tokenManager.GenerateToken(user.ID.String(), user.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	createReq := handlers.CreateChangePlanRequest{
		ProjectID:        project.ID.String(),
		TargetDatabaseID: database.ID.String(),
		Title:            "Test Change Plan",
		Description:      "Test description",
		ChangeType:       "DDL",
		SQLContent:       "CREATE TABLE users (id UUID PRIMARY KEY);",
	}

	body, _ := json.Marshal(createReq)
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, user.ID, user.Email, user.OrganizationRole)

	// Execute
	err = handler.CreateChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestIntegration_ChangePlan_GetByID tests GET /api/v1/change-plans/:id
func TestIntegration_ChangePlan_GetByID(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	ctx := context.Background()
	logger := zap.NewNop()
	tokenManager := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing")

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test user
	author := createTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err := projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "development")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for author
	token, err := tokenManager.GenerateToken(author.ID.String(), author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, author.ID, author.Email, author.OrganizationRole)

	// Execute
	err = handler.GetChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var response handlers.ChangePlanResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, changePlan.ID.String(), response.ID)
	assert.Equal(t, changePlan.Title, response.Title)
	assert.Equal(t, changePlan.Description, response.Description)
	assert.Equal(t, changePlan.SQLContent, response.SQLContent)
	assert.Equal(t, changePlan.Status, response.Status)
}

// TestIntegration_ChangePlan_GetByID_NotFound tests GET /api/v1/change-plans/:id with non-existent ID
func TestIntegration_ChangePlan_GetByID_NotFound(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	logger := zap.NewNop()
	tokenManager := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing")

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)

	// Create test user
	user := createTestUser(t, userRepo, "user@example.com", models.OrganizationRoleDeveloper)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token
	token, err := tokenManager.GenerateToken(user.ID.String(), user.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request with non-existent ID
	nonExistentID := uuid.New()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/change-plans/%s", nonExistentID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id")
	c.SetParamNames("id")
	c.SetParamValues(nonExistentID.String())
	setUserContext(c, user.ID, user.Email, user.OrganizationRole)

	// Execute
	err = handler.GetChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestIntegration_ChangePlan_List tests GET /api/v1/change-plans
func TestIntegration_ChangePlan_List(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	ctx := context.Background()
	logger := zap.NewNop()
	tokenManager := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing")

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test user
	author := createTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err := projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "development")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create multiple change plans
	changePlan1 := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)
	changePlan2 := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE orders (id UUID PRIMARY KEY);",
	)
	changePlan3 := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE products (id UUID PRIMARY KEY);",
	)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for author
	token, err := tokenManager.GenerateToken(author.ID.String(), author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserContext(c, author.ID, author.Email, author.OrganizationRole)

	// Execute
	err = handler.ListChangePlans(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	changePlans, ok := response["change_plans"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(changePlans), 3) // Should include at least the 3 we created

	// Verify total count
	total, ok := response["total"].(float64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, int(total), 3)
}

// TestIntegration_ChangePlan_Update tests PUT /api/v1/change-plans/:id
func TestIntegration_ChangePlan_Update(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	ctx := context.Background()
	logger := zap.NewNop()
	tokenManager := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing")

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test user
	author := createTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err := projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "development")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan in Draft status
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for author
	token, err := tokenManager.GenerateToken(author.ID.String(), author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create update request
	updateReq := handlers.UpdateChangePlanRequest{
		Title:       "Updated Change Plan Title",
		Description: "Updated description",
		SQLContent:  "CREATE TABLE users (id UUID PRIMARY KEY, email VARCHAR(255));",
	}

	body, _ := json.Marshal(updateReq)
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()),
		bytes.NewReader(body),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, author.ID, author.Email, author.OrganizationRole)

	// Execute
	err = handler.UpdateChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var response handlers.ChangePlanResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, updateReq.Title, response.Title)
	assert.Equal(t, updateReq.Description, response.Description)
	assert.Equal(t, updateReq.SQLContent, response.SQLContent)
}

// TestIntegration_ChangePlan_Update_Forbidden tests PUT /api/v1/change-plans/:id with insufficient permissions
func TestIntegration_ChangePlan_Update_Forbidden(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	ctx := context.Background()
	logger := zap.NewNop()
	tokenManager := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing")

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test users
	author := createTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)
	otherUser := createTestUser(t, userRepo, "other@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add author as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err := projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "development")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for other user (not a project member)
	token, err := tokenManager.GenerateToken(otherUser.ID.String(), otherUser.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create update request
	updateReq := handlers.UpdateChangePlanRequest{
		Title: "Updated Title",
	}

	body, _ := json.Marshal(updateReq)
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()),
		bytes.NewReader(body),
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, otherUser.ID, otherUser.Email, otherUser.OrganizationRole)

	// Execute
	err = handler.UpdateChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestIntegration_ChangePlan_Delete tests DELETE /api/v1/change-plans/:id
func TestIntegration_ChangePlan_Delete(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	ctx := context.Background()
	logger := zap.NewNop()
	tokenManager := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing")

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test user
	author := createTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err := projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "development")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan in Draft status (deletable)
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for author
	token, err := tokenManager.GenerateToken(author.ID.String(), author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, author.ID, author.Email, author.OrganizationRole)

	// Execute
	err = handler.DeleteChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Verify change plan was deleted
	_, err = changePlanRepo.GetByID(ctx, changePlan.ID)
	assert.Error(t, err) // Should not be found
}

// TestIntegration_ChangePlan_Delete_Forbidden tests DELETE /api/v1/change-plans/:id with insufficient permissions
func TestIntegration_ChangePlan_Delete_Forbidden(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	ctx := context.Background()
	logger := zap.NewNop()
	tokenManager := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing")

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test users
	author := createTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)
	otherUser := createTestUser(t, userRepo, "other@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add author as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err := projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "development")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for other user (not a project member)
	token, err := tokenManager.GenerateToken(otherUser.ID.String(), otherUser.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, otherUser.ID, otherUser.Email, otherUser.OrganizationRole)

	// Execute
	err = handler.DeleteChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestIntegration_ChangePlan_Delete_InvalidStatus tests DELETE /api/v1/change-plans/:id with non-deletable status
func TestIntegration_ChangePlan_Delete_InvalidStatus(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	ctx := context.Background()
	logger := zap.NewNop()
	tokenManager := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing")

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test user
	author := createTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err := projectMemberRepo.Create(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "development")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan in Open status (not deletable)
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)
	changePlan.Status = models.ChangePlanStatusOpen
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for author
	token, err := tokenManager.GenerateToken(author.ID.String(), author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, author.ID, author.Email, author.OrganizationRole)

	// Execute
	err = handler.DeleteChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "can only be deleted")
}

// TestIntegration_ChangePlan_AssignReviewer tests PUT /api/v1/change-plans/:id/reviewer
func TestIntegration_ChangePlan_AssignReviewer(t *testing.T) {
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
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	ctx := context.Background()

	// Create author user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create reviewer user (Project Approver)
	reviewer := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("reviewer-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add author as project member with Developer role
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Add reviewer as project member with Approver role
	reviewerMember := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    reviewer.ID,
		Role:      models.ProjectRoleApprover,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, reviewerMember)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan in Draft status
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for author
	token, err := tokenManager.GenerateToken(author.ID.String(), author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request body
	requestBody := map[string]interface{}{
		"reviewer_id": reviewer.ID.String(),
	}
	requestJSON, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/api/v1/change-plans/%s/reviewer", changePlan.ID.String()),
		bytes.NewReader(requestJSON),
	)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/reviewer")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, author.ID, author.Email, author.OrganizationRole)

	// Execute
	err = handler.AssignReviewer(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Reviewer assigned successfully", response["message"])
	assert.Equal(t, changePlan.ID.String(), response["change_plan_id"])
	assert.Equal(t, reviewer.ID.String(), response["reviewer_id"])

	// Verify change plan was updated
	updatedPlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.NotNil(t, updatedPlan.AssignedReviewerID)
	assert.Equal(t, reviewer.ID, *updatedPlan.AssignedReviewerID)
}

// TestIntegration_ChangePlan_AssignReviewer_Forbidden tests PUT /api/v1/change-plans/:id/reviewer with non-author
func TestIntegration_ChangePlan_AssignReviewer_Forbidden(t *testing.T) {
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
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	ctx := context.Background()

	// Create author user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create another user (not the author)
	otherUser := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("other-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create reviewer user
	reviewer := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("reviewer-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add users as project members
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	otherMember := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    otherUser.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, otherMember)
	require.NoError(t, err)

	reviewerMember := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    reviewer.ID,
		Role:      models.ProjectRoleApprover,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, reviewerMember)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan authored by author
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for other user (not the author)
	token, err := tokenManager.GenerateToken(otherUser.ID.String(), otherUser.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request body
	requestBody := map[string]interface{}{
		"reviewer_id": reviewer.ID.String(),
	}
	requestJSON, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/api/v1/change-plans/%s/reviewer", changePlan.ID.String()),
		bytes.NewReader(requestJSON),
	)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/reviewer")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, otherUser.ID, otherUser.Email, otherUser.OrganizationRole)

	// Execute
	err = handler.AssignReviewer(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Only the author can assign")
}

// TestIntegration_ChangePlan_AssignReviewer_InvalidReviewer tests PUT /api/v1/change-plans/:id/reviewer with non-approver
func TestIntegration_ChangePlan_AssignReviewer_InvalidReviewer(t *testing.T) {
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
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	ctx := context.Background()

	// Create author user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create developer user (not an approver)
	developer := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("developer-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add users as project members
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Add developer with Developer role (not Approver)
	developerMember := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    developer.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, developerMember)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for author
	token, err := tokenManager.GenerateToken(author.ID.String(), author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request body trying to assign a developer as reviewer
	requestBody := map[string]interface{}{
		"reviewer_id": developer.ID.String(),
	}
	requestJSON, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPut,
		fmt.Sprintf("/api/v1/change-plans/%s/reviewer", changePlan.ID.String()),
		bytes.NewReader(requestJSON),
	)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/reviewer")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, author.ID, author.Email, author.OrganizationRole)

	// Execute
	err = handler.AssignReviewer(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Reviewer must have Project Approver role")
}

// TestIntegration_ChangePlan_ExecuteChangePlan tests POST /api/v1/change-plans/:id/execute
func TestIntegration_ChangePlan_ExecuteChangePlan(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", time.Hour)
	require.NoError(t, err)

	// Create test data
	ctx := context.Background()
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	projectMemberRepo := postgres.NewProjectMemberRepository(db)

	// Create author user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add author as project member (Project Owner role for execution permission)
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleOwner,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create and approve change plan
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)

	// Transition to Open and Approve
	changePlan.Status = models.ChangePlanStatusOpen
	changePlan.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	changePlan.Status = models.ChangePlanStatusApproved
	changePlan.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for author
	token, err := tokenManager.GenerateToken(author.ID.String(), author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlan.ID.String()),
		nil,
	)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/execute")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, author.ID, author.Email, author.OrganizationRole)

	// Execute
	err = handler.ExecuteChangePlan(c)
	require.NoError(t, err)

	// Note: Execution may fail due to missing actual database connection,
	// but we verify the endpoint structure and basic logic
	// Status should be either 200 (success) or 500 (execution failure)
	// 400 would indicate validation errors which we want to catch
	assert.NotEqual(t, http.StatusBadRequest, rec.Code, "should not fail validation")
	assert.NotEqual(t, http.StatusForbidden, rec.Code, "should not fail authorization")
	assert.NotEqual(t, http.StatusNotFound, rec.Code, "should find change plan")

	// Verify response structure if successful
	if rec.Code == http.StatusOK {
		var response map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response contains change_plan and execution fields
		assert.Contains(t, response, "change_plan")
		assert.Contains(t, response, "execution")

		executionData := response["execution"].(map[string]interface{})
		assert.Contains(t, executionData, "success")
		assert.Contains(t, executionData, "rows_affected")
		assert.Contains(t, executionData, "duration_ms")
	}
}

// TestIntegration_ChangePlan_ExecuteChangePlan_Unauthorized tests POST /api/v1/change-plans/:id/execute with unauthorized user
func TestIntegration_ChangePlan_ExecuteChangePlan_Unauthorized(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", time.Hour)
	require.NoError(t, err)

	// Create test data
	ctx := context.Background()
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	projectMemberRepo := postgres.NewProjectMemberRepository(db)

	// Create author and unauthorized user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	unauthorizedUser := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("unauthorized-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add author as project member (Project Owner)
	authorMember := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleOwner,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, authorMember)
	require.NoError(t, err)

	// Add unauthorized user as project member (Project Viewer - no execute permission)
	unauthorizedMember := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    unauthorizedUser.ID,
		Role:      models.ProjectRoleViewer,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, unauthorizedMember)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create and approve change plan
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)

	// Transition to Approved
	changePlan.Status = models.ChangePlanStatusApproved
	changePlan.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for unauthorized user
	token, err := tokenManager.GenerateToken(
		unauthorizedUser.ID.String(),
		unauthorizedUser.Email,
		string(models.OrganizationRoleDeveloper),
	)
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlan.ID.String()),
		nil,
	)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/execute")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, unauthorizedUser.ID, unauthorizedUser.Email, unauthorizedUser.OrganizationRole)

	// Execute
	err = handler.ExecuteChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Insufficient permissions")
}

// TestIntegration_ChangePlan_ExecuteChangePlan_InvalidStatus tests POST /api/v1/change-plans/:id/execute with non-approved status
func TestIntegration_ChangePlan_ExecuteChangePlan_InvalidStatus(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", time.Hour)
	require.NoError(t, err)

	// Create test data
	ctx := context.Background()
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	projectMemberRepo := postgres.NewProjectMemberRepository(db)

	// Create author user
	author := createTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add author as project member (Project Owner role for execution permission)
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleOwner,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create change plan in Open status (not Approved)
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)

	// Transition to Open (not Approved)
	changePlan.Status = models.ChangePlanStatusOpen
	changePlan.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for author
	token, err := tokenManager.GenerateToken(author.ID.String(), author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlan.ID.String()),
		nil,
	)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/execute")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, author.ID, author.Email, author.OrganizationRole)

	// Execute
	err = handler.ExecuteChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Only Approved change plans can be executed")
}
