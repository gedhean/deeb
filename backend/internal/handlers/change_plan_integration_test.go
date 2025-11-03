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
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/repository/postgres"
	"github.com/nsxbet/deeb/internal/review"
	"github.com/nsxbet/deeb/internal/review/validators"
	"github.com/nsxbet/deeb/internal/services"
	"github.com/nsxbet/deeb/internal/utils"
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

// createChangePlanTestUser creates a test user and returns it
func createChangePlanTestUser(t *testing.T, userRepo repository.UserRepository, email string, role models.OrganizationRole) *models.User {
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
	desc := "Test project"
	project := &models.Project{
		Name:        name,
		Description: &desc,
		CreatedByID: uuid.New(), // Required field
	}

	err := projectRepo.Create(ctx, project)
	require.NoError(t, err, "failed to create test project")

	return project
}

// createTestEnvironment creates a test environment
func createTestEnvironment(t *testing.T, envRepo *postgres.EnvironmentRepository, name string) *models.Environment {
	t.Helper()

	ctx := context.Background()
	desc := "Test environment"
	env := &models.Environment{
		Name:        name,
		Description: &desc,
		CreatedByID: uuid.New(), // Required field
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
	desc := "Test database instance"
	encryptedConn := "encrypted_connection_details" // Mock encrypted connection details
	dbInstance := &models.DatabaseInstance{
		Name:                       name,
		Description:                &desc,
		DatabaseType:               dbType,
		Host:                       "localhost",
		Port:                       5432,
		Username:                   "testuser",
		ConnectionDetailsEncrypted: &encryptedConn,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                uuid.New(), // Required field
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
	desc := "Test database"
	db := &models.Database{
		ProjectID:          projectID,
		EnvironmentID:      envID,
		DatabaseInstanceID: dbInstanceID,
		DatabaseName:       name,
		Description:        &desc,
		Status:             models.DatabaseStatusActive,
		CreatedByID:        uuid.New(), // Required field
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

// setupChangePlanHandlerWithTimeout creates a ChangePlanHandler with custom execution timeout
func setupChangePlanHandlerWithTimeout(t *testing.T, db *sqlx.DB, tokenManager *auth.TokenManager, timeout time.Duration) *handlers.ChangePlanHandler {
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

	// Initialize execution service with custom timeout
	executionService := services.NewExecutionServiceWithTimeout(
		changePlanRepo,
		databaseInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		queueService,
		executionLogRepo,
		connectionMgr,
		encryptionKey,
		timeout,
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
	user := createChangePlanTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleAdmin)

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
	user := createChangePlanTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleAdmin)

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
	user := createChangePlanTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleAdmin)

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
	user := createChangePlanTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleAdmin)

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

	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", time.Hour)
	require.NoError(t, err)

	// Create test data
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create user (not a project member)
	user := createChangePlanTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleDeveloper)

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
	user := createChangePlanTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleAdmin)

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
		"/api/v1/change-plans?status=Open&limit=1&offset=0",
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
	user := createChangePlanTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleDeveloper)

	// Create project and add user as member
	project := createTestProject(t, projectRepo, "Test Project")
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
	err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "syntax_error",
		Severity:      "Error",
		RuleName:      "InvalidSyntax",
		Message:       "Syntax error at line 1",
		LineNumber:    intPtr(1),
		ColumnNumber:  intPtr(10),
	})
	require.NoError(t, err)

	err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "security",
		Severity:      "Warning",
		RuleName:      "MissingIndex",
		Message:       "Table missing index",
		LineNumber:    intPtr(1),
	})
	require.NoError(t, err)

	err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
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
	token, err := tokenManager.GenerateToken(user.ID, user.Email, string(models.OrganizationRoleDeveloper))
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
	user := createChangePlanTestUser(t, userRepo, fmt.Sprintf("test-%d@example.com", time.Now().Unix()), models.OrganizationRoleDeveloper)

	// Create project and add user as member
	project := createTestProject(t, projectRepo, "Test Project")
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
	token, err := tokenManager.GenerateToken(user.ID, user.Email, string(models.OrganizationRoleDeveloper))
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
	approver := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("approver-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add approver as member with Approver role
	project := createTestProject(t, projectRepo, "Test Project")
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    approver.ID,
		Role:      models.ProjectRoleApprover,
		AddedByID: approver.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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
	err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "syntax_error",
		Severity:      "Error",
		RuleName:      "InvalidSyntax",
		Message:       "Syntax error at line 1",
		LineNumber:    intPtr(1),
		ColumnNumber:  intPtr(10),
	})
	require.NoError(t, err)

	err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
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
	token, err := tokenManager.GenerateToken(approver.ID, approver.Email, string(models.OrganizationRoleDeveloper))
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
	approver := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("approver-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add approver as member with Approver role
	project := createTestProject(t, projectRepo, "Test Project")
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    approver.ID,
		Role:      models.ProjectRoleApprover,
		AddedByID: approver.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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
	err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
		ChangePlanID:  changePlan.ID,
		ValidatorType: "performance",
		Severity:      "Warning",
		RuleName:      "MissingIndex",
		Message:       "Table missing index",
		LineNumber:    intPtr(1),
	})
	require.NoError(t, err)

	err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
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
	token, err := tokenManager.GenerateToken(approver.ID, approver.Email, string(models.OrganizationRoleDeveloper))
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
	executor := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("executor-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add executor as member with Approver role (can execute)
	project := createTestProject(t, projectRepo, "Test Project")
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    executor.ID,
		Role:      models.ProjectRoleApprover,
		AddedByID: executor.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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
	err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
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
	token, err := tokenManager.GenerateToken(executor.ID, executor.Email, string(models.OrganizationRoleDeveloper))
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
	executor := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("executor-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add executor as member with Approver role (can execute)
	project := createTestProject(t, projectRepo, "Test Project")
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    executor.ID,
		Role:      models.ProjectRoleApprover,
		AddedByID: executor.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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
	err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
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
	token, err := tokenManager.GenerateToken(executor.ID, executor.Email, string(models.OrganizationRoleDeveloper))
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
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add author as member
	project := createTestProject(t, projectRepo, "Test Project")
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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
	err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create non-author user
	nonAuthor := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("nonauthor-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add both users as members
	project := createTestProject(t, projectRepo, "Test Project")
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
	require.NoError(t, err)

	member3 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    nonAuthor.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: nonAuthor.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member3)
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
	token, err := tokenManager.GenerateToken(nonAuthor.ID, nonAuthor.Email, string(models.OrganizationRoleDeveloper))
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
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add author as member
	project := createTestProject(t, projectRepo, "Test Project")
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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
	tokenManager, err := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing", time.Hour)
	require.NoError(t, err)

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)

	// Create test user
	author := createChangePlanTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member with Developer role
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	createReq := models.CreateChangePlanRequest{
		ProjectID:        project.ID,
		TargetDatabaseID: database.ID,
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
	var response models.ChangePlanResponse
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

	tokenManager, err := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing", time.Hour)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create request without token
	createReq := models.CreateChangePlanRequest{
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
	err = handler.CreateChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestIntegration_ChangePlan_Create_Forbidden tests POST /api/v1/change-plans with insufficient permissions
func TestIntegration_ChangePlan_Create_Forbidden(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	tokenManager, err := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing", time.Hour)
	require.NoError(t, err)

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)

	// Create test user (not a member of project)
	user := createChangePlanTestUser(t, userRepo, "user@example.com", models.OrganizationRoleDeveloper)

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
	token, err := tokenManager.GenerateToken(user.ID, user.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	createReq := models.CreateChangePlanRequest{
		ProjectID:        project.ID,
		TargetDatabaseID: database.ID,
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

// TestIntegration_ChangePlan_Create_SQLContentTooLarge tests POST /api/v1/change-plans with SQL content exceeding 10MB
func TestIntegration_ChangePlan_Create_SQLContentTooLarge(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	ctx := context.Background()
	tokenManager, err := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing", time.Hour)
	require.NoError(t, err)

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)

	// Create test user
	author := createChangePlanTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member with Developer role
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create SQL content larger than 10MB
	largeSQLContent := strings.Repeat("A", handlers.MaxSQLContentSize+1)

	// Create request with oversized SQL content
	createReq := models.CreateChangePlanRequest{
		ProjectID:        project.ID,
		TargetDatabaseID: database.ID,
		Title:            "Test Change Plan",
		Description:      "Test description",
		ChangeType:       "DDL",
		SQLContent:       largeSQLContent,
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
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "exceeds maximum size")
}

// TestIntegration_ChangePlan_GetByID tests GET /api/v1/change-plans/:id
func TestIntegration_ChangePlan_GetByID(t *testing.T) {
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	cleanupChangePlanTestData(t, db)
	defer cleanupChangePlanTestData(t, db)

	ctx := context.Background()
	tokenManager, err := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing", time.Hour)
	require.NoError(t, err)

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test user
	author := createChangePlanTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err = projectMemberRepo.AddMember(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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
	var response models.ChangePlanResponse
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

	tokenManager, err := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing", time.Hour)
	require.NoError(t, err)

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)

	// Create test user
	user := createChangePlanTestUser(t, userRepo, "user@example.com", models.OrganizationRoleDeveloper)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token
	token, err := tokenManager.GenerateToken(user.ID, user.Email, string(models.OrganizationRoleDeveloper))
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
	tokenManager, err := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing", time.Hour)
	require.NoError(t, err)

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test user
	author := createChangePlanTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err = projectMemberRepo.AddMember(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "development")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create multiple change plans
	_ = createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE users (id UUID PRIMARY KEY);",
	)
	_ = createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE orders (id UUID PRIMARY KEY);",
	)
	_ = createTestChangePlan(
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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
	tokenManager, err := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing", time.Hour)
	require.NoError(t, err)

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test user
	author := createChangePlanTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err = projectMemberRepo.AddMember(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create update request
	title := "Updated Change Plan Title"
	description := "Updated description"
	sqlContent := "CREATE TABLE users (id UUID PRIMARY KEY, email VARCHAR(255));"
	updateReq := models.UpdateChangePlanRequest{
		Title:       &title,
		Description: &description,
		SQLContent:  &sqlContent,
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
	var response models.ChangePlanResponse
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
	tokenManager, err := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing", time.Hour)
	require.NoError(t, err)

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test users
	author := createChangePlanTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)
	otherUser := createChangePlanTestUser(t, userRepo, "other@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add author as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err = projectMemberRepo.AddMember(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
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
	token, err := tokenManager.GenerateToken(otherUser.ID, otherUser.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create update request
	title := "Updated Title"
	updateReq := models.UpdateChangePlanRequest{
		Title: &title,
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
	tokenManager, err := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing", time.Hour)
	require.NoError(t, err)

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test user
	author := createChangePlanTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err = projectMemberRepo.AddMember(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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
	tokenManager, err := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing", time.Hour)
	require.NoError(t, err)

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test users
	author := createChangePlanTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)
	otherUser := createChangePlanTestUser(t, userRepo, "other@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add author as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err = projectMemberRepo.AddMember(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
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
	token, err := tokenManager.GenerateToken(otherUser.ID, otherUser.Email, string(models.OrganizationRoleDeveloper))
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
	tokenManager, err := auth.NewTokenManager("test-secret-key-min-32-chars-for-testing", time.Hour)
	require.NoError(t, err)

	// Setup repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "test-encryption-key-min-32-chars")
	dbRepo := postgres.NewDatabaseRepository(db)
	changePlanRepo := postgres.NewChangePlanRepository(db)

	// Create test user
	author := createChangePlanTestUser(t, userRepo, "author@example.com", models.OrganizationRoleDeveloper)

	// Create project
	project := createTestProject(t, projectRepo, "test-project")

	// Add user as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	err = projectMemberRepo.AddMember(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create reviewer user (Project Approver)
	reviewer := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("reviewer-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add author as project member with Developer role
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create another user (not the author)
	otherUser := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("other-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create reviewer user
	reviewer := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("reviewer-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add users as project members
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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
	token, err := tokenManager.GenerateToken(otherUser.ID, otherUser.Email, string(models.OrganizationRoleDeveloper))
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
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create developer user (not an approver)
	developer := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("developer-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add users as project members
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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
	author := createChangePlanTestUser(
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	unauthorizedUser := createChangePlanTestUser(
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
		unauthorizedUser.ID,
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
	author := createChangePlanTestUser(
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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

// setupExecutionTargetDB sets up a separate PostgreSQL database for execution target testing
func setupExecutionTargetDB(t *testing.T) (string, func()) {
	t.Helper()

	ctx := context.Background()

	// Start PostgreSQL container for execution target
	postgresContainer, err := postgrescontainer.Run(ctx,
		"postgres:16-alpine",
		postgrescontainer.WithDatabase("execution_target"),
		postgrescontainer.WithUsername("deeb"),
		postgrescontainer.WithPassword("deeb"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "failed to start execution target postgres container")

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get execution target connection string")

	// Return cleanup function
	cleanup := func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate execution target postgres container: %v", err)
		}
	}

	return connStr, cleanup
}

// TestIntegration_ChangePlan_ExecuteChangePlan_Success tests successful execution with real database
func TestIntegration_ChangePlan_ExecuteChangePlan_Success(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	// Setup execution target database
	targetConnStr, cleanupTarget := setupExecutionTargetDB(t)
	defer cleanupTarget()

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
	snapshotRepo := postgres.NewSchemaSnapshotRepository(db)

	// Create author user
	author := createChangePlanTestUser(
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

	// Parse target connection string to get host and port
	// Format: postgres://user:password@host:port/database?sslmode=disable
	targetURL, err := url.Parse(targetConnStr)
	require.NoError(t, err, "failed to parse target connection string")
	targetHost := targetURL.Hostname()
	targetPort, err := strconv.Atoi(targetURL.Port())
	if err != nil || targetPort == 0 {
		targetPort = 5432 // Default PostgreSQL port
	}
	targetDB := "execution_target"

	// Encrypt password
	detailsJSON, err := json.Marshal(map[string]string{"password": "deeb"})
	require.NoError(t, err)
	encryptedDetails, err := utils.Encrypt(string(detailsJSON), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	// Create database instance pointing to execution target
	dbInstance := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       targetHost,
		Port:                       targetPort,
		Username:                   "deeb",
		ConnectionDetailsEncrypted: &encryptedDetails,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance)
	require.NoError(t, err)

	// Create database pointing to execution target
	database := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       targetDB,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)

	// Create and approve change plan
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE execution_test (id UUID PRIMARY KEY, name VARCHAR(100));",
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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
	require.Equal(t, http.StatusOK, rec.Code, "execution should succeed")

	// Verify response structure
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "change_plan")
	assert.Contains(t, response, "execution")

	executionData := response["execution"].(map[string]interface{})
	assert.Contains(t, executionData, "success")
	assert.True(t, executionData["success"].(bool), "execution should be successful")
	assert.Contains(t, executionData, "rows_affected")
	assert.Contains(t, executionData, "duration_ms")

	// Verify change plan status is Completed
	updatedChangePlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusCompleted, updatedChangePlan.Status)
	assert.NotNil(t, updatedChangePlan.ExecutedAt)
	assert.NotNil(t, updatedChangePlan.ExecutionDurationMs)

	// Verify snapshots were captured (before and after)
	snapshots, err := snapshotRepo.GetByChangePlanID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Len(t, snapshots, 2, "should have before and after snapshots")

	var beforeSnapshot, afterSnapshot *models.SchemaSnapshot
	for _, snapshot := range snapshots {
		if snapshot.SnapshotType == models.SnapshotTypeBefore {
			beforeSnapshot = snapshot
		} else if snapshot.SnapshotType == models.SnapshotTypeAfter {
			afterSnapshot = snapshot
		}
	}
	require.NotNil(t, beforeSnapshot, "before snapshot should exist")
	require.NotNil(t, afterSnapshot, "after snapshot should exist")
	assert.NotEmpty(t, beforeSnapshot.SnapshotDataRaw, "before snapshot should have data")
	assert.NotEmpty(t, afterSnapshot.SnapshotDataRaw, "after snapshot should have data")

	// Verify SQL was actually executed by checking the target database
	targetDBConn, err := sqlx.Connect("postgres", targetConnStr)
	require.NoError(t, err)
	defer targetDBConn.Close()

	var tableExists bool
	err = targetDBConn.Get(&tableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'execution_test'
		)
	`)
	require.NoError(t, err)
	assert.True(t, tableExists, "execution_test table should exist after execution")
}

// TestIntegration_ChangePlan_ExecuteChangePlan_Failure tests execution failure and transaction rollback
func TestIntegration_ChangePlan_ExecuteChangePlan_Failure(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	// Setup execution target database
	targetConnStr, cleanupTarget := setupExecutionTargetDB(t)
	defer cleanupTarget()

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
	snapshotRepo := postgres.NewSchemaSnapshotRepository(db)

	// Create author user
	author := createChangePlanTestUser(
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

	// Parse target connection string to get host and port
	targetURL, err := url.Parse(targetConnStr)
	require.NoError(t, err, "failed to parse target connection string")
	targetHost := targetURL.Hostname()
	targetPort, err := strconv.Atoi(targetURL.Port())
	if err != nil || targetPort == 0 {
		targetPort = 5432 // Default PostgreSQL port
	}

	// Encrypt password
	detailsJSON, err := json.Marshal(map[string]string{"password": "deeb"})
	require.NoError(t, err)
	encryptedDetails, err := utils.Encrypt(string(detailsJSON), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	// Create database instance pointing to execution target
	dbInstance := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       targetHost,
		Port:                       targetPort,
		Username:                   "deeb",
		ConnectionDetailsEncrypted: &encryptedDetails,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance)
	require.NoError(t, err)

	// Create database pointing to execution target
	database := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       "execution_target",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)

	// Create and approve change plan with invalid SQL that will fail
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE rollback_test (id UUID PRIMARY KEY); INSERT INTO non_existent_table VALUES (1);",
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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

	// Execute - should fail
	err = handler.ExecuteChangePlan(c)
	require.NoError(t, err)

	// Status should be 500 (internal server error) or 200 with error in response
	// The execution service returns 200 even on failure, with error details in response
	assert.NotEqual(t, http.StatusBadRequest, rec.Code, "should not be validation error")
	assert.NotEqual(t, http.StatusForbidden, rec.Code, "should not be authorization error")
	assert.NotEqual(t, http.StatusNotFound, rec.Code, "should find change plan")

	// Verify response contains error information
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify change plan status is Failed
	updatedChangePlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusFailed, updatedChangePlan.Status)
	assert.NotNil(t, updatedChangePlan.ErrorMessage)
	assert.Contains(t, *updatedChangePlan.ErrorMessage, "non_existent_table", "error message should mention the failed table")

	// Verify transaction was rolled back - table should not exist
	targetDBConn, err := sqlx.Connect("postgres", targetConnStr)
	require.NoError(t, err)
	defer targetDBConn.Close()

	var tableExists bool
	err = targetDBConn.Get(&tableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'rollback_test'
		)
	`)
	require.NoError(t, err)
	assert.False(t, tableExists, "rollback_test table should not exist after failed execution (transaction rollback)")

	// Verify before snapshot was captured even on failure
	snapshots, err := snapshotRepo.GetByChangePlanID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(snapshots), 1, "should have at least before snapshot")

	var hasBeforeSnapshot bool
	for _, snapshot := range snapshots {
		if snapshot.SnapshotType == models.SnapshotTypeBefore {
			hasBeforeSnapshot = true
		}
	}
	assert.True(t, hasBeforeSnapshot, "before snapshot should exist even on failure")
}

// TestIntegration_ChangePlan_ExecuteChangePlan_SnapshotCapture tests snapshot capture during execution
func TestIntegration_ChangePlan_ExecuteChangePlan_SnapshotCapture(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	// Setup execution target database
	targetConnStr, cleanupTarget := setupExecutionTargetDB(t)
	defer cleanupTarget()

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
	snapshotRepo := postgres.NewSchemaSnapshotRepository(db)

	// Create author user
	author := createChangePlanTestUser(
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

	// Parse target connection string to get host and port
	targetURL, err := url.Parse(targetConnStr)
	require.NoError(t, err, "failed to parse target connection string")
	targetHost := targetURL.Hostname()
	targetPort, err := strconv.Atoi(targetURL.Port())
	if err != nil || targetPort == 0 {
		targetPort = 5432 // Default PostgreSQL port
	}

	// Encrypt password
	detailsJSON, err := json.Marshal(map[string]string{"password": "deeb"})
	require.NoError(t, err)
	encryptedDetails, err := utils.Encrypt(string(detailsJSON), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	// Create database instance pointing to execution target
	dbInstance := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       targetHost,
		Port:                       targetPort,
		Username:                   "deeb",
		ConnectionDetailsEncrypted: &encryptedDetails,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance)
	require.NoError(t, err)

	// Create database pointing to execution target
	database := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       "execution_target",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)

	// Create initial table in target database
	targetDBConn, err := sqlx.Connect("postgres", targetConnStr)
	require.NoError(t, err)
	defer targetDBConn.Close()

	_, err = targetDBConn.Exec("CREATE TABLE snapshot_test_initial (id UUID PRIMARY KEY);")
	require.NoError(t, err)

	// Create and approve change plan that adds a column
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"ALTER TABLE snapshot_test_initial ADD COLUMN name VARCHAR(100);",
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
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
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
	require.Equal(t, http.StatusOK, rec.Code, "execution should succeed")

	// Verify snapshots were captured with correct data
	snapshots, err := snapshotRepo.GetByChangePlanID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Len(t, snapshots, 2, "should have before and after snapshots")

	var beforeSnapshot, afterSnapshot *models.SchemaSnapshot
	for _, snapshot := range snapshots {
		if snapshot.SnapshotType == models.SnapshotTypeBefore {
			beforeSnapshot = snapshot
		} else if snapshot.SnapshotType == models.SnapshotTypeAfter {
			afterSnapshot = snapshot
		}
	}
	require.NotNil(t, beforeSnapshot, "before snapshot should exist")
	require.NotNil(t, afterSnapshot, "after snapshot should exist")

	// Verify snapshot data is not empty
	assert.NotEmpty(t, beforeSnapshot.SnapshotDataRaw, "before snapshot should have raw data")
	assert.NotEmpty(t, afterSnapshot.SnapshotDataRaw, "after snapshot should have raw data")

	// Verify snapshot data contains table schema
	assert.Contains(t, *beforeSnapshot.SnapshotDataRaw, "snapshot_test_initial", "before snapshot should contain table name")
	assert.Contains(t, *beforeSnapshot.SnapshotDataRaw, "id", "before snapshot should contain id column")
	assert.NotContains(t, *beforeSnapshot.SnapshotDataRaw, "name VARCHAR", "before snapshot should not contain new column")

	assert.Contains(t, *afterSnapshot.SnapshotDataRaw, "snapshot_test_initial", "after snapshot should contain table name")
	assert.Contains(t, *afterSnapshot.SnapshotDataRaw, "id", "after snapshot should contain id column")
	assert.Contains(t, *afterSnapshot.SnapshotDataRaw, "name VARCHAR", "after snapshot should contain new column")

	// Verify snapshots have correct metadata
	assert.Equal(t, changePlan.ID, beforeSnapshot.ChangePlanID)
	assert.Equal(t, changePlan.ID, afterSnapshot.ChangePlanID)
	assert.Equal(t, database.ID, beforeSnapshot.DatabaseID)
	assert.Equal(t, database.ID, afterSnapshot.DatabaseID)
	assert.False(t, beforeSnapshot.CreatedAt.IsZero())
	assert.False(t, afterSnapshot.CreatedAt.IsZero())
	assert.True(t, beforeSnapshot.CreatedAt.Before(afterSnapshot.CreatedAt) || beforeSnapshot.CreatedAt.Equal(afterSnapshot.CreatedAt), "before snapshot should be created before or at same time as after")
}

// TestIntegration_ChangePlan_ExecuteChangePlan_ConcurrentSameDatabase tests concurrent execution blocking on same database
func TestIntegration_ChangePlan_ExecuteChangePlan_ConcurrentSameDatabase(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	// Setup execution target database
	targetConnStr, cleanupTarget := setupExecutionTargetDB(t)
	defer cleanupTarget()

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
	queueRepo := postgres.NewExecutionQueueRepository(db)
	lockManager := services.NewExecutionLockManager(db, zap.NewNop())

	// Create two author users for concurrent execution
	author1 := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author1-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	author2 := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author2-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add authors as project members (Project Owner role for execution permission)
	member1 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author1.ID,
		Role:      models.ProjectRoleOwner,
		AddedByID: author1.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member1)
	require.NoError(t, err)

	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author2.ID,
		Role:      models.ProjectRoleOwner,
		AddedByID: author2.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Parse target connection string to get host and port
	targetURL, err := url.Parse(targetConnStr)
	require.NoError(t, err, "failed to parse target connection string")
	targetHost := targetURL.Hostname()
	targetPort, err := strconv.Atoi(targetURL.Port())
	if err != nil || targetPort == 0 {
		targetPort = 5432 // Default PostgreSQL port
	}

	// Encrypt password
	detailsJSON, err := json.Marshal(map[string]string{"password": "deeb"})
	require.NoError(t, err)
	encryptedDetails, err := utils.Encrypt(string(detailsJSON), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	// Create database instance pointing to execution target
	dbInstance := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       targetHost,
		Port:                       targetPort,
		Username:                   "deeb",
		ConnectionDetailsEncrypted: &encryptedDetails,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author1.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance)
	require.NoError(t, err)

	// Create database pointing to execution target
	database := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       "execution_target",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)

	// Create two change plans targeting the same database
	changePlan1 := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author1.ID,
		"CREATE TABLE concurrent_test1 (id UUID PRIMARY KEY);",
	)
	changePlan2 := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author2.ID,
		"CREATE TABLE concurrent_test2 (id UUID PRIMARY KEY);",
	)

	// Transition both to Approved
	changePlan1.Status = models.ChangePlanStatusApproved
	changePlan1.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan1)
	require.NoError(t, err)

	changePlan2.Status = models.ChangePlanStatusApproved
	changePlan2.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan2)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create tokens for both authors
	token1, err := tokenManager.GenerateToken(author1.ID, author1.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)
	token2, err := tokenManager.GenerateToken(author2.ID, author2.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Execute first change plan (should succeed and acquire lock)
	e := echo.New()
	req1 := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlan1.ID.String()),
		nil,
	)
	req1.Header.Set(echo.HeaderAuthorization, "Bearer "+token1)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	c1.SetPath("/api/v1/change-plans/:id/execute")
	c1.SetParamNames("id")
	c1.SetParamValues(changePlan1.ID.String())
	setUserContext(c1, author1.ID, author1.Email, author1.OrganizationRole)

	// Start first execution in goroutine (will take some time due to snapshot capture)
	exec1Done := make(chan error, 1)
	go func() {
		exec1Done <- handler.ExecuteChangePlan(c1)
	}()

	// Wait a bit to ensure first execution has acquired the lock
	time.Sleep(500 * time.Millisecond)

	// Verify lock is acquired
	isLocked, lock, err := lockManager.IsLocked(ctx, database.ID)
	require.NoError(t, err)
	require.True(t, isLocked, "database should be locked")
	require.NotNil(t, lock)
	require.Equal(t, changePlan1.ID, lock.ChangePlanID)

	// Try to execute second change plan concurrently (should be blocked/queued)
	req2 := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlan2.ID.String()),
		nil,
	)
	req2.Header.Set(echo.HeaderAuthorization, "Bearer "+token2)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.SetPath("/api/v1/change-plans/:id/execute")
	c2.SetParamNames("id")
	c2.SetParamValues(changePlan2.ID.String())
	setUserContext(c2, author2.ID, author2.Email, author2.OrganizationRole)

	err2 := handler.ExecuteChangePlan(c2)
	require.NoError(t, err2)

	// Second execution should return 200 but with error indicating database locked
	// OR should have queued the change plan
	// Check if change plan was queued
	queueItems, err := queueRepo.GetByChangePlanID(ctx, changePlan2.ID)
	require.NoError(t, err)
	require.Len(t, queueItems, 1, "change plan should be queued")
	require.Equal(t, models.QueueStatusPending, queueItems[0].Status)
	require.Equal(t, database.ID, queueItems[0].TargetDatabaseID)

	// Wait for first execution to complete
	err1 := <-exec1Done
	require.NoError(t, err1)
	require.Equal(t, http.StatusOK, rec1.Code, "first execution should succeed")

	// Verify first execution completed successfully
	updatedChangePlan1, err := changePlanRepo.GetByID(ctx, changePlan1.ID)
	require.NoError(t, err)
	require.Equal(t, models.ChangePlanStatusCompleted, updatedChangePlan1.Status)

	// Verify lock was released after first execution
	isLocked, _, err = lockManager.IsLocked(ctx, database.ID)
	require.NoError(t, err)
	require.False(t, isLocked, "lock should be released after execution")

	// Verify first change plan SQL was executed
	targetDBConn, err := sqlx.Connect("postgres", targetConnStr)
	require.NoError(t, err)
	defer targetDBConn.Close()

	var table1Exists bool
	err = targetDBConn.Get(&table1Exists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'concurrent_test1'
		)
	`)
	require.NoError(t, err)
	require.True(t, table1Exists, "concurrent_test1 table should exist")

	// Verify second change plan was NOT executed yet (still in queue)
	var table2Exists bool
	err = targetDBConn.Get(&table2Exists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'concurrent_test2'
		)
	`)
	require.NoError(t, err)
	require.False(t, table2Exists, "concurrent_test2 table should not exist yet (still queued)")
}

// TestIntegration_ChangePlan_ExecuteChangePlan_ConcurrentDifferentDatabases tests concurrent execution on different databases
func TestIntegration_ChangePlan_ExecuteChangePlan_ConcurrentDifferentDatabases(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	// Setup two execution target databases
	targetConnStr1, cleanupTarget1 := setupExecutionTargetDB(t)
	defer cleanupTarget1()
	targetConnStr2, cleanupTarget2 := setupExecutionTargetDB(t)
	defer cleanupTarget2()

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
	lockManager := services.NewExecutionLockManager(db, zap.NewNop())

	// Create two authors
	author1 := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author1-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	author2 := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author2-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add authors as project members
	member1 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author1.ID,
		Role:      models.ProjectRoleOwner,
		AddedByID: author1.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member1)
	require.NoError(t, err)

	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author2.ID,
		Role:      models.ProjectRoleOwner,
		AddedByID: author2.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Setup first database
	targetURL1, err := url.Parse(targetConnStr1)
	require.NoError(t, err)
	targetHost1 := targetURL1.Hostname()
	targetPort1, err := strconv.Atoi(targetURL1.Port())
	if err != nil || targetPort1 == 0 {
		targetPort1 = 5432
	}

	detailsJSON1, err := json.Marshal(map[string]string{"password": "deeb"})
	require.NoError(t, err)
	encryptedDetails1, err := utils.Encrypt(string(detailsJSON1), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	dbInstance1 := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance 1",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       targetHost1,
		Port:                       targetPort1,
		Username:                   "deeb",
		ConnectionDetailsEncrypted: &encryptedDetails1,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author1.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance1)
	require.NoError(t, err)

	database1 := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance1.ID,
		DatabaseName:       "execution_target",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database1)
	require.NoError(t, err)

	// Setup second database
	targetURL2, err := url.Parse(targetConnStr2)
	require.NoError(t, err)
	targetHost2 := targetURL2.Hostname()
	targetPort2, err := strconv.Atoi(targetURL2.Port())
	if err != nil || targetPort2 == 0 {
		targetPort2 = 5432
	}

	detailsJSON2, err := json.Marshal(map[string]string{"password": "deeb"})
	require.NoError(t, err)
	encryptedDetails2, err := utils.Encrypt(string(detailsJSON2), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	dbInstance2 := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance 2",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       targetHost2,
		Port:                       targetPort2,
		Username:                   "deeb",
		ConnectionDetailsEncrypted: &encryptedDetails2,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author2.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance2)
	require.NoError(t, err)

	database2 := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance2.ID,
		DatabaseName:       "execution_target",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database2)
	require.NoError(t, err)

	// Create change plans targeting different databases
	changePlan1 := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database1.ID,
		author1.ID,
		"CREATE TABLE different_db_test1 (id UUID PRIMARY KEY);",
	)
	changePlan2 := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database2.ID,
		author2.ID,
		"CREATE TABLE different_db_test2 (id UUID PRIMARY KEY);",
	)

	// Transition both to Approved
	changePlan1.Status = models.ChangePlanStatusApproved
	changePlan1.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan1)
	require.NoError(t, err)

	changePlan2.Status = models.ChangePlanStatusApproved
	changePlan2.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan2)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create tokens
	token1, err := tokenManager.GenerateToken(author1.ID, author1.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)
	token2, err := tokenManager.GenerateToken(author2.ID, author2.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Execute both change plans concurrently
	e := echo.New()
	req1 := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlan1.ID.String()),
		nil,
	)
	req1.Header.Set(echo.HeaderAuthorization, "Bearer "+token1)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	c1.SetPath("/api/v1/change-plans/:id/execute")
	c1.SetParamNames("id")
	c1.SetParamValues(changePlan1.ID.String())
	setUserContext(c1, author1.ID, author1.Email, author1.OrganizationRole)

	req2 := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlan2.ID.String()),
		nil,
	)
	req2.Header.Set(echo.HeaderAuthorization, "Bearer "+token2)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.SetPath("/api/v1/change-plans/:id/execute")
	c2.SetParamNames("id")
	c2.SetParamValues(changePlan2.ID.String())
	setUserContext(c2, author2.ID, author2.Email, author2.OrganizationRole)

	// Execute concurrently
	var wg sync.WaitGroup
	var err1, err2 error

	wg.Add(2)
	go func() {
		defer wg.Done()
		err1 = handler.ExecuteChangePlan(c1)
	}()
	go func() {
		defer wg.Done()
		err2 = handler.ExecuteChangePlan(c2)
	}()

	wg.Wait()

	// Both executions should succeed
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.Equal(t, http.StatusOK, rec1.Code, "first execution should succeed")
	require.Equal(t, http.StatusOK, rec2.Code, "second execution should succeed")

	// Verify both change plans completed
	updatedChangePlan1, err := changePlanRepo.GetByID(ctx, changePlan1.ID)
	require.NoError(t, err)
	require.Equal(t, models.ChangePlanStatusCompleted, updatedChangePlan1.Status)

	updatedChangePlan2, err := changePlanRepo.GetByID(ctx, changePlan2.ID)
	require.NoError(t, err)
	require.Equal(t, models.ChangePlanStatusCompleted, updatedChangePlan2.Status)

	// Verify both databases have their tables (concurrent execution succeeded)
	targetDBConn1, err := sqlx.Connect("postgres", targetConnStr1)
	require.NoError(t, err)
	defer targetDBConn1.Close()

	var table1Exists bool
	err = targetDBConn1.Get(&table1Exists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'different_db_test1'
		)
	`)
	require.NoError(t, err)
	require.True(t, table1Exists, "different_db_test1 table should exist in database 1")

	targetDBConn2, err := sqlx.Connect("postgres", targetConnStr2)
	require.NoError(t, err)
	defer targetDBConn2.Close()

	var table2Exists bool
	err = targetDBConn2.Get(&table2Exists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'different_db_test2'
		)
	`)
	require.NoError(t, err)
	require.True(t, table2Exists, "different_db_test2 table should exist in database 2")

	// Verify locks were released
	isLocked1, _, err := lockManager.IsLocked(ctx, database1.ID)
	require.NoError(t, err)
	require.False(t, isLocked1, "lock for database1 should be released")

	isLocked2, _, err := lockManager.IsLocked(ctx, database2.ID)
	require.NoError(t, err)
	require.False(t, isLocked2, "lock for database2 should be released")
}

// TestIntegration_ChangePlan_ExecuteChangePlan_LockRelease tests lock release after execution
func TestIntegration_ChangePlan_ExecuteChangePlan_LockRelease(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	// Setup execution target database
	targetConnStr, cleanupTarget := setupExecutionTargetDB(t)
	defer cleanupTarget()

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
	lockManager := services.NewExecutionLockManager(db, zap.NewNop())

	// Create author user
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add author as project member
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

	// Parse target connection string
	targetURL, err := url.Parse(targetConnStr)
	require.NoError(t, err, "failed to parse target connection string")
	targetHost := targetURL.Hostname()
	targetPort, err := strconv.Atoi(targetURL.Port())
	if err != nil || targetPort == 0 {
		targetPort = 5432
	}

	// Encrypt password
	detailsJSON, err := json.Marshal(map[string]string{"password": "deeb"})
	require.NoError(t, err)
	encryptedDetails, err := utils.Encrypt(string(detailsJSON), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	// Create database instance
	dbInstance := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       targetHost,
		Port:                       targetPort,
		Username:                   "deeb",
		ConnectionDetailsEncrypted: &encryptedDetails,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance)
	require.NoError(t, err)

	// Create database
	database := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       "execution_target",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)

	// Create and approve change plan
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE lock_release_test (id UUID PRIMARY KEY);",
	)

	changePlan.Status = models.ChangePlanStatusApproved
	changePlan.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Verify no lock exists initially
	isLocked, _, err := lockManager.IsLocked(ctx, database.ID)
	require.NoError(t, err)
	require.False(t, isLocked, "database should not be locked initially")

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Execute change plan
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

	err = handler.ExecuteChangePlan(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code, "execution should succeed")

	// Verify execution completed
	updatedChangePlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	require.Equal(t, models.ChangePlanStatusCompleted, updatedChangePlan.Status)

	// Verify lock was released after execution
	isLocked, _, err = lockManager.IsLocked(ctx, database.ID)
	require.NoError(t, err)
	require.False(t, isLocked, "lock should be released after execution completes")

	// Verify we can acquire lock again (proving it was released)
	err = lockManager.AcquireLock(ctx, database.ID, uuid.New(), author.ID)
	require.NoError(t, err, "should be able to acquire lock after previous execution")

	// Cleanup the test lock
	err = lockManager.ReleaseLock(ctx, database.ID)
	require.NoError(t, err)
}

// TestIntegration_ChangePlan_ExecuteChangePlan_QueueHandling tests queue handling for multiple change plans
func TestIntegration_ChangePlan_ExecuteChangePlan_QueueHandling(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	// Setup execution target database
	targetConnStr, cleanupTarget := setupExecutionTargetDB(t)
	defer cleanupTarget()

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
	queueRepo := postgres.NewExecutionQueueRepository(db)
	lockManager := services.NewExecutionLockManager(db, zap.NewNop())

	// Create three authors for queuing
	author1 := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author1-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	author2 := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author2-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	author3 := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author3-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add authors as project members
	for _, author := range []*models.User{author1, author2, author3} {
		member := &models.ProjectMember{
			ProjectID: project.ID,
			UserID:    author.ID,
			Role:      models.ProjectRoleOwner,
			AddedByID: author.ID,
		}
		err = projectMemberRepo.AddMember(ctx, member)
		require.NoError(t, err)
	}

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Parse target connection string
	targetURL, err := url.Parse(targetConnStr)
	require.NoError(t, err)
	targetHost := targetURL.Hostname()
	targetPort, err := strconv.Atoi(targetURL.Port())
	if err != nil || targetPort == 0 {
		targetPort = 5432
	}

	// Encrypt password
	detailsJSON, err := json.Marshal(map[string]string{"password": "deeb"})
	require.NoError(t, err)
	encryptedDetails, err := utils.Encrypt(string(detailsJSON), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	// Create database instance
	dbInstance := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       targetHost,
		Port:                       targetPort,
		Username:                   "deeb",
		ConnectionDetailsEncrypted: &encryptedDetails,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author1.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance)
	require.NoError(t, err)

	// Create database
	database := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       "execution_target",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)

	// Create three change plans targeting the same database
	changePlan1 := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author1.ID,
		"CREATE TABLE queue_test1 (id UUID PRIMARY KEY);",
	)
	changePlan2 := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author2.ID,
		"CREATE TABLE queue_test2 (id UUID PRIMARY KEY);",
	)
	changePlan3 := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author3.ID,
		"CREATE TABLE queue_test3 (id UUID PRIMARY KEY);",
	)

	// Transition all to Approved
	for _, cp := range []*models.ChangePlan{changePlan1, changePlan2, changePlan3} {
		cp.Status = models.ChangePlanStatusApproved
		cp.UpdatedAt = time.Now()
		err = changePlanRepo.Update(ctx, cp)
		require.NoError(t, err)
	}

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Execute first change plan (acquires lock)
	token1, err := tokenManager.GenerateToken(author1.ID, author1.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	e := echo.New()
	req1 := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlan1.ID.String()),
		nil,
	)
	req1.Header.Set(echo.HeaderAuthorization, "Bearer "+token1)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	c1.SetPath("/api/v1/change-plans/:id/execute")
	c1.SetParamNames("id")
	c1.SetParamValues(changePlan1.ID.String())
	setUserContext(c1, author1.ID, author1.Email, author1.OrganizationRole)

	// Start first execution
	exec1Done := make(chan error, 1)
	go func() {
		exec1Done <- handler.ExecuteChangePlan(c1)
	}()

	// Wait for lock acquisition
	time.Sleep(500 * time.Millisecond)

	// Verify lock is acquired
	isLocked, _, err := lockManager.IsLocked(ctx, database.ID)
	require.NoError(t, err)
	require.True(t, isLocked, "database should be locked")

	// Try to execute second and third change plans (should be queued)
	token2, err := tokenManager.GenerateToken(author2.ID, author2.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)
	token3, err := tokenManager.GenerateToken(author3.ID, author3.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	req2 := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlan2.ID.String()),
		nil,
	)
	req2.Header.Set(echo.HeaderAuthorization, "Bearer "+token2)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	c2.SetPath("/api/v1/change-plans/:id/execute")
	c2.SetParamNames("id")
	c2.SetParamValues(changePlan2.ID.String())
	setUserContext(c2, author2.ID, author2.Email, author2.OrganizationRole)

	req3 := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlan3.ID.String()),
		nil,
	)
	req3.Header.Set(echo.HeaderAuthorization, "Bearer "+token3)
	rec3 := httptest.NewRecorder()
	c3 := e.NewContext(req3, rec3)
	c3.SetPath("/api/v1/change-plans/:id/execute")
	c3.SetParamNames("id")
	c3.SetParamValues(changePlan3.ID.String())
	setUserContext(c3, author3.ID, author3.Email, author3.OrganizationRole)

	err2 := handler.ExecuteChangePlan(c2)
	require.NoError(t, err2)
	err3 := handler.ExecuteChangePlan(c3)
	require.NoError(t, err3)

	// Verify both change plans were queued
	queueItems2, err := queueRepo.GetByChangePlanID(ctx, changePlan2.ID)
	require.NoError(t, err)
	require.Len(t, queueItems2, 1, "change plan 2 should be queued")
	require.Equal(t, models.QueueStatusPending, queueItems2[0].Status)

	queueItems3, err := queueRepo.GetByChangePlanID(ctx, changePlan3.ID)
	require.NoError(t, err)
	require.Len(t, queueItems3, 1, "change plan 3 should be queued")
	require.Equal(t, models.QueueStatusPending, queueItems3[0].Status)

	// Verify queue positions (second should be position 1, third should be position 2)
	position2 := queueItems2[0].Position
	position3 := queueItems3[0].Position
	require.Greater(t, position3, position2, "change plan 3 should be after change plan 2 in queue")

	// Verify pending count for database
	pendingCount, err := queueRepo.GetPendingCount(ctx, database.ID)
	require.NoError(t, err)
	require.Equal(t, 2, pendingCount, "should have 2 pending items in queue")

	// Wait for first execution to complete
	err1 := <-exec1Done
	require.NoError(t, err1)
	require.Equal(t, http.StatusOK, rec1.Code)

	// Verify lock was released
	isLocked, _, err = lockManager.IsLocked(ctx, database.ID)
	require.NoError(t, err)
	require.False(t, isLocked, "lock should be released")

	// Verify first change plan completed
	updatedChangePlan1, err := changePlanRepo.GetByID(ctx, changePlan1.ID)
	require.NoError(t, err)
	require.Equal(t, models.ChangePlanStatusCompleted, updatedChangePlan1.Status)

	// Verify queue items still exist (they would be processed by queue processor in real scenario)
	// For this test, we just verify they were queued correctly
	queueItems, err := queueRepo.GetByDatabaseID(ctx, database.ID)
	require.NoError(t, err)
	require.Len(t, queueItems, 2, "should have 2 queue items")
	for _, item := range queueItems {
		require.Equal(t, models.QueueStatusPending, item.Status)
		require.Equal(t, database.ID, item.TargetDatabaseID)
	}
}

// TestIntegration_ChangePlan_ExecuteChangePlan_SQLSyntaxError tests SQL syntax error handling
func TestIntegration_ChangePlan_ExecuteChangePlan_SQLSyntaxError(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	// Setup execution target database
	targetConnStr, cleanupTarget := setupExecutionTargetDB(t)
	defer cleanupTarget()

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
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add author as project member
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

	// Parse target connection string
	targetURL, err := url.Parse(targetConnStr)
	require.NoError(t, err)
	targetHost := targetURL.Hostname()
	targetPort, err := strconv.Atoi(targetURL.Port())
	if err != nil || targetPort == 0 {
		targetPort = 5432
	}

	// Encrypt password
	detailsJSON, err := json.Marshal(map[string]string{"password": "deeb"})
	require.NoError(t, err)
	encryptedDetails, err := utils.Encrypt(string(detailsJSON), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	// Create database instance
	dbInstance := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       targetHost,
		Port:                       targetPort,
		Username:                   "deeb",
		ConnectionDetailsEncrypted: &encryptedDetails,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance)
	require.NoError(t, err)

	// Create database
	database := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       "execution_target",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)

	// Create change plan with SQL syntax error
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE syntax_error_test (id UUID PRIMARY KEY;", // Missing closing parenthesis
	)

	// Transition to Approved
	changePlan.Status = models.ChangePlanStatusApproved
	changePlan.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Execute change plan
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

	err = handler.ExecuteChangePlan(c)
	require.NoError(t, err)

	// Execution should fail
	// Verify change plan status is Failed
	updatedChangePlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	require.Equal(t, models.ChangePlanStatusFailed, updatedChangePlan.Status)
	require.NotNil(t, updatedChangePlan.ErrorMessage)
	require.Contains(t, *updatedChangePlan.ErrorMessage, "syntax", "error message should mention syntax error")

	// Verify transaction was rolled back - table should not exist
	targetDBConn, err := sqlx.Connect("postgres", targetConnStr)
	require.NoError(t, err)
	defer targetDBConn.Close()

	var tableExists bool
	err = targetDBConn.Get(&tableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'syntax_error_test'
		)
	`)
	require.NoError(t, err)
	require.False(t, tableExists, "syntax_error_test table should not exist after failed execution (transaction rollback)")
}

// TestIntegration_ChangePlan_ExecuteChangePlan_Timeout tests execution timeout handling
func TestIntegration_ChangePlan_ExecuteChangePlan_Timeout(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	// Setup execution target database
	targetConnStr, cleanupTarget := setupExecutionTargetDB(t)
	defer cleanupTarget()

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
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add author as project member
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

	// Parse target connection string
	targetURL, err := url.Parse(targetConnStr)
	require.NoError(t, err)
	targetHost := targetURL.Hostname()
	targetPort, err := strconv.Atoi(targetURL.Port())
	if err != nil || targetPort == 0 {
		targetPort = 5432
	}

	// Encrypt password
	detailsJSON, err := json.Marshal(map[string]string{"password": "deeb"})
	require.NoError(t, err)
	encryptedDetails, err := utils.Encrypt(string(detailsJSON), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	// Create database instance
	dbInstance := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       targetHost,
		Port:                       targetPort,
		Username:                   "deeb",
		ConnectionDetailsEncrypted: &encryptedDetails,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance)
	require.NoError(t, err)

	// Create database
	database := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       "execution_target",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)

	// Create change plan with SQL that will timeout (pg_sleep for 3 seconds, timeout set to 1 second)
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"SELECT pg_sleep(3);", // Sleep for 3 seconds
	)

	// Transition to Approved
	changePlan.Status = models.ChangePlanStatusApproved
	changePlan.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Setup handler with short timeout (1 second)
	handler := setupChangePlanHandlerWithTimeout(t, db, tokenManager, 1*time.Second)

	// Create token
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Execute change plan
	startTime := time.Now()
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

	err = handler.ExecuteChangePlan(c)
	require.NoError(t, err)
	duration := time.Since(startTime)

	// Verify execution timed out (should fail within ~1-2 seconds, not wait for full 3 seconds)
	require.Less(t, duration, 3*time.Second, "execution should timeout before completing")
	require.GreaterOrEqual(t, duration, 1*time.Second, "execution should take at least the timeout duration")

	// Verify change plan status is Failed
	updatedChangePlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	require.Equal(t, models.ChangePlanStatusFailed, updatedChangePlan.Status)
	require.NotNil(t, updatedChangePlan.ErrorMessage)
	require.Contains(t, *updatedChangePlan.ErrorMessage, "timeout", "error message should mention timeout")
}

// TestIntegration_ChangePlan_ExecuteChangePlan_ConnectionFailure tests connection failure handling
func TestIntegration_ChangePlan_ExecuteChangePlan_ConnectionFailure(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

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
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add author as project member
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

	// Create database instance with invalid credentials (will cause connection failure)
	detailsJSON, err := json.Marshal(map[string]string{"password": "wrong-password"})
	require.NoError(t, err)
	encryptedDetails, err := utils.Encrypt(string(detailsJSON), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	dbInstance := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance (Invalid)",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       "localhost",
		Port:                       5432,
		Username:                   "invalid_user",
		ConnectionDetailsEncrypted: &encryptedDetails,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance)
	require.NoError(t, err)

	// Create database pointing to invalid instance
	database := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       "execution_target",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)

	// Create change plan
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE connection_test (id UUID PRIMARY KEY);",
	)

	// Transition to Approved
	changePlan.Status = models.ChangePlanStatusApproved
	changePlan.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Execute change plan
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

	err = handler.ExecuteChangePlan(c)
	require.NoError(t, err)

	// Execution should fail due to connection error
	// Verify change plan status is Failed
	updatedChangePlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	require.Equal(t, models.ChangePlanStatusFailed, updatedChangePlan.Status)
	require.NotNil(t, updatedChangePlan.ErrorMessage)
	// Connection error should be captured (either in snapshot capture or execution phase)
	require.NotEmpty(t, *updatedChangePlan.ErrorMessage, "error message should be present")
}

// TestIntegration_ChangePlan_ExecuteChangePlan_RollbackVerification tests comprehensive rollback verification
func TestIntegration_ChangePlan_ExecuteChangePlan_RollbackVerification(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	// Setup execution target database
	targetConnStr, cleanupTarget := setupExecutionTargetDB(t)
	defer cleanupTarget()

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
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add author as project member
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

	// Parse target connection string
	targetURL, err := url.Parse(targetConnStr)
	require.NoError(t, err)
	targetHost := targetURL.Hostname()
	targetPort, err := strconv.Atoi(targetURL.Port())
	if err != nil || targetPort == 0 {
		targetPort = 5432
	}

	// Encrypt password
	detailsJSON, err := json.Marshal(map[string]string{"password": "deeb"})
	require.NoError(t, err)
	encryptedDetails, err := utils.Encrypt(string(detailsJSON), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	// Create database instance
	dbInstance := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       targetHost,
		Port:                       targetPort,
		Username:                   "deeb",
		ConnectionDetailsEncrypted: &encryptedDetails,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance)
	require.NoError(t, err)

	// Create database
	database := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       "execution_target",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)

	// Create initial table in target database to test rollback of changes
	targetDBConn, err := sqlx.Connect("postgres", targetConnStr)
	require.NoError(t, err)
	defer targetDBConn.Close()

	_, err = targetDBConn.Exec("CREATE TABLE rollback_verification_initial (id UUID PRIMARY KEY, value VARCHAR(100));")
	require.NoError(t, err)
	_, err = targetDBConn.Exec("INSERT INTO rollback_verification_initial (id, value) VALUES (gen_random_uuid(), 'initial_value');")
	require.NoError(t, err)

	// Verify initial state
	var initialCount int
	err = targetDBConn.Get(&initialCount, "SELECT COUNT(*) FROM rollback_verification_initial")
	require.NoError(t, err)
	require.Equal(t, 1, initialCount, "should have 1 initial row")

	// Create change plan that will fail mid-execution (create table succeeds, insert fails)
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE rollback_verification_new (id UUID PRIMARY KEY); INSERT INTO rollback_verification_new VALUES (gen_random_uuid()); INSERT INTO non_existent_table VALUES (1);",
	)

	// Transition to Approved
	changePlan.Status = models.ChangePlanStatusApproved
	changePlan.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Execute change plan
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

	err = handler.ExecuteChangePlan(c)
	require.NoError(t, err)

	// Verify change plan status is Failed
	updatedChangePlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	require.Equal(t, models.ChangePlanStatusFailed, updatedChangePlan.Status)

	// Verify transaction was rolled back completely:
	// 1. New table should not exist (even though CREATE TABLE succeeded)
	var newTableExists bool
	err = targetDBConn.Get(&newTableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'rollback_verification_new'
		)
	`)
	require.NoError(t, err)
	require.False(t, newTableExists, "rollback_verification_new table should not exist (transaction rollback)")

	// 2. Initial table should be unchanged
	var finalCount int
	err = targetDBConn.Get(&finalCount, "SELECT COUNT(*) FROM rollback_verification_initial")
	require.NoError(t, err)
	require.Equal(t, initialCount, finalCount, "initial table should be unchanged")
}

// TestIntegration_ChangePlan_ExecuteChangePlan_ErrorLogging tests error logging during execution failures
func TestIntegration_ChangePlan_ExecuteChangePlan_ErrorLogging(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

	// Setup execution target database
	targetConnStr, cleanupTarget := setupExecutionTargetDB(t)
	defer cleanupTarget()

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
	executionLogRepo := postgres.NewExecutionLogRepository(db)

	// Create author user
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add author as project member
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

	// Parse target connection string
	targetURL, err := url.Parse(targetConnStr)
	require.NoError(t, err)
	targetHost := targetURL.Hostname()
	targetPort, err := strconv.Atoi(targetURL.Port())
	if err != nil || targetPort == 0 {
		targetPort = 5432
	}

	// Encrypt password
	detailsJSON, err := json.Marshal(map[string]string{"password": "deeb"})
	require.NoError(t, err)
	encryptedDetails, err := utils.Encrypt(string(detailsJSON), "test-encryption-key-min-32-chars")
	require.NoError(t, err)

	// Create database instance
	dbInstance := &models.DatabaseInstance{
		ID:                         uuid.New(),
		Name:                       "Test Execution Instance",
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       targetHost,
		Port:                       targetPort,
		Username:                   "deeb",
		ConnectionDetailsEncrypted: &encryptedDetails,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                author.ID,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err = dbInstanceRepo.Create(ctx, dbInstance)
	require.NoError(t, err)

	// Create database
	database := &models.Database{
		ID:                 uuid.New(),
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       "execution_target",
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)

	// Create change plan with SQL that will fail
	changePlan := createTestChangePlan(
		t,
		changePlanRepo,
		project.ID,
		database.ID,
		author.ID,
		"CREATE TABLE error_logging_test (id UUID PRIMARY KEY); INSERT INTO non_existent_table VALUES (1);",
	)

	// Transition to Approved
	changePlan.Status = models.ChangePlanStatusApproved
	changePlan.UpdatedAt = time.Now()
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token
	token, err := tokenManager.GenerateToken(author.ID, author.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Execute change plan
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

	err = handler.ExecuteChangePlan(c)
	require.NoError(t, err)

	// Verify change plan status is Failed
	updatedChangePlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	require.Equal(t, models.ChangePlanStatusFailed, updatedChangePlan.Status)
	require.NotNil(t, updatedChangePlan.ErrorMessage)
	require.NotEmpty(t, *updatedChangePlan.ErrorMessage, "error message should be logged")

	// Verify execution log entry was created
	executionLogs, err := executionLogRepo.GetByChangePlanID(ctx, changePlan.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(executionLogs), 1, "should have at least one execution log entry")

	// Find the failed execution log
	var failedLog *models.ExecutionLog
	for _, log := range executionLogs {
		if log.Status == models.ExecutionLogStatusFailed {
			failedLog = log
			break
		}
	}
	require.NotNil(t, failedLog, "should have a failed execution log entry")
	require.NotNil(t, failedLog.ErrorMessage, "execution log should have error message")
	require.Contains(t, *failedLog.ErrorMessage, "non_existent_table", "execution log should contain error details")
	require.False(t, failedLog.StartedAt.IsZero(), "execution log should have start time")
	require.False(t, failedLog.CompletedAt.IsZero(), "execution log should have completion time")
	require.NotNil(t, failedLog.ExecutionDurationMs, "execution log should have duration")
}

// TestIntegration_ChangePlan_CloseChangePlan tests POST /api/v1/change-plans/:id/close
func TestIntegration_ChangePlan_CloseChangePlan(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

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
	approvalRepo := postgres.NewApprovalRepository(db)

	// Create approver user
	approver := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("approver-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add approver as member with Approver role
	project := createTestProject(t, projectRepo, "Test Project")
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    approver.ID,
		Role:      models.ProjectRoleApprover,
		AddedByID: approver.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for approver
	token, err := tokenManager.GenerateToken(approver.ID, approver.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request with comments
	comments := "SQL syntax error detected, please fix"
	reqBody := map[string]interface{}{
		"comments": comments,
	}
	reqBodyJSON, err := json.Marshal(reqBody)
	require.NoError(t, err)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/change-plans/%s/close", changePlan.ID.String()), bytes.NewReader(reqBodyJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/close")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, approver.ID, approver.Email, approver.OrganizationRole)

	// Execute
	err = handler.CloseChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Change plan closed successfully", response["message"])
	assert.Equal(t, changePlan.ID.String(), response["change_plan_id"])

	// Verify change plan status is now Closed
	updatedPlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusClosed, updatedPlan.Status)

	// Verify approval record was created with Rejected status
	approvals, err := approvalRepo.GetHistory(ctx, changePlan.ID)
	require.NoError(t, err)
	require.Len(t, approvals, 1)
	assert.Equal(t, models.ApprovalStatusRejected, approvals[0].Status)
	assert.Equal(t, approver.ID, approvals[0].ApproverID)
	assert.NotNil(t, approvals[0].Comments)
	assert.Equal(t, comments, *approvals[0].Comments)
	assert.Nil(t, approvals[0].ApprovedAt) // Rejected approvals don't have approved_at
}

// TestIntegration_ChangePlan_CloseChangePlan_Unauthorized tests POST /api/v1/change-plans/:id/close with unauthorized user
func TestIntegration_ChangePlan_CloseChangePlan_Unauthorized(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

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

	// Create unauthorized user (not an approver)
	unauthorizedUser := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("unauthorized-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add unauthorized user as project member with Developer role (not Approver)
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    unauthorizedUser.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: unauthorizedUser.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create approver user
	approver := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("approver-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	err = projectMemberRepo.AddMember(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    approver.ID,
		Role:      models.ProjectRoleApprover,
		AddedByID: approver.ID,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for unauthorized user
	token, err := tokenManager.GenerateToken(unauthorizedUser.ID, unauthorizedUser.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/change-plans/%s/close", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/close")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, unauthorizedUser.ID, unauthorizedUser.Email, unauthorizedUser.OrganizationRole)

	// Execute
	err = handler.CloseChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "unauthorized")
}

// TestIntegration_ChangePlan_GetApprovalHistory tests GET /api/v1/change-plans/:id/approvals
func TestIntegration_ChangePlan_GetApprovalHistory(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

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
	approvalRepo := postgres.NewApprovalRepository(db)
	_ = postgres.NewPolicyFindingRepository(db) // Initialize for future use

	// Create approver user
	approver := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("approver-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project and add approver as member with Approver role
	project := createTestProject(t, projectRepo, "Test Project")
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    approver.ID,
		Role:      models.ProjectRoleApprover,
		AddedByID: approver.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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

	// Create first approval (rejected)
	rejectComments := "SQL syntax error"
	rejectedApproval := &models.Approval{
		ID:           uuid.New(),
		ChangePlanID: changePlan.ID,
		ApproverID:   approver.ID,
		Status:       models.ApprovalStatusRejected,
		Comments:     &rejectComments,
		CreatedAt:    time.Now().Add(-2 * time.Hour),
	}
	err = approvalRepo.Create(ctx, rejectedApproval)
	require.NoError(t, err)

	// Update change plan back to Open (after rejection)
	changePlan.Status = models.ChangePlanStatusOpen
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Create second approval (approved)
	approveComments := "Looks good"
	approvedApproval := &models.Approval{
		ID:           uuid.New(),
		ChangePlanID: changePlan.ID,
		ApproverID:   approver.ID,
		Status:       models.ApprovalStatusApproved,
		Comments:     &approveComments,
		CreatedAt:    time.Now().Add(-1 * time.Hour),
	}
	now := time.Now().Add(-1 * time.Hour)
	approvedApproval.ApprovedAt = &now
	err = approvalRepo.Create(ctx, approvedApproval)
	require.NoError(t, err)

	// Update change plan to Approved
	changePlan.Status = models.ChangePlanStatusApproved
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for approver (who can view approval history)
	token, err := tokenManager.GenerateToken(approver.ID, approver.Email, string(models.OrganizationRoleDeveloper))
	require.NoError(t, err)

	// Create request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/change-plans/%s/approvals", changePlan.ID.String()), nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v1/change-plans/:id/approvals")
	c.SetParamNames("id")
	c.SetParamValues(changePlan.ID.String())
	setUserContext(c, approver.ID, approver.Email, approver.OrganizationRole)

	// Execute
	err = handler.GetApprovalHistory(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "approvals")
	approvals := response["approvals"].([]interface{})
	require.Len(t, approvals, 2) // Should return both approvals, most recent first

	// Verify first approval (most recent - approved)
	firstApproval := approvals[0].(map[string]interface{})
	assert.Equal(t, approvedApproval.ID.String(), firstApproval["id"])
	assert.Equal(t, string(models.ApprovalStatusApproved), firstApproval["status"])
	assert.Equal(t, approveComments, firstApproval["comments"])
	assert.NotNil(t, firstApproval["approved_at"])

	// Verify second approval (rejected)
	secondApproval := approvals[1].(map[string]interface{})
	assert.Equal(t, rejectedApproval.ID.String(), secondApproval["id"])
	assert.Equal(t, string(models.ApprovalStatusRejected), secondApproval["status"])
	assert.Equal(t, rejectComments, secondApproval["comments"])
	assert.Nil(t, secondApproval["approved_at"]) // Rejected approvals don't have approved_at
}

// TestIntegration_ChangePlan_ApproveChangePlan_Unauthorized tests POST /api/v1/change-plans/:id/approve with unauthorized user
func TestIntegration_ChangePlan_ApproveChangePlan_Unauthorized(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

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

	// Create unauthorized user (not an approver)
	unauthorizedUser := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("unauthorized-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Add unauthorized user as project member with Developer role (not Approver)
	member := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    unauthorizedUser.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: unauthorizedUser.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create approver user
	approver := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("approver-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	err = projectMemberRepo.AddMember(ctx, &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    approver.ID,
		Role:      models.ProjectRoleApprover,
		AddedByID: approver.ID,
	})
	require.NoError(t, err)

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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

	// Setup handler
	handler := setupChangePlanHandler(t, db, tokenManager)

	// Create token for unauthorized user
	token, err := tokenManager.GenerateToken(unauthorizedUser.ID, unauthorizedUser.Email, string(models.OrganizationRoleDeveloper))
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
	setUserContext(c, unauthorizedUser.ID, unauthorizedUser.Email, unauthorizedUser.OrganizationRole)

	// Execute
	err = handler.ApproveChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "unauthorized")
}

// TestIntegration_ChangePlan_ApproveChangePlan_OrganizationAdmin tests approval with Organization Admin role
func TestIntegration_ChangePlan_ApproveChangePlan_OrganizationAdmin(t *testing.T) {
	// Setup
	db, cleanup := setupTestDBForChangePlans(t)
	defer cleanup()
	defer cleanupChangePlanTestData(t, db)

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
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)

	// Create organization admin user (no project membership needed)
	orgAdmin := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("admin-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleAdmin,
	)

	// Create project
	project := createTestProject(t, projectRepo, "Test Project")

	// Create environment
	env := createTestEnvironment(t, envRepo, "production")

	// Create database instance
	dbInstance := createTestDatabaseInstance(t, dbInstanceRepo, "test-instance", models.DatabaseTypePostgreSQL)

	// Create database
	database := createTestDatabase(t, dbRepo, project.ID, env.ID, dbInstance.ID, "testdb")

	// Create author user
	author := createChangePlanTestUser(
		t,
		userRepo,
		fmt.Sprintf("author-%d@example.com", time.Now().Unix()),
		models.OrganizationRoleDeveloper,
	)
	member2 := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    author.ID,
		Role:      models.ProjectRoleDeveloper,
		AddedByID: author.ID,
	}
	err = projectMemberRepo.AddMember(ctx, member2)
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
	err = policyFindingRepo.Create(ctx, &models.PolicyFinding{
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

	// Create token for organization admin
	token, err := tokenManager.GenerateToken(orgAdmin.ID, orgAdmin.Email, string(models.OrganizationRoleAdmin))
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
	setUserContext(c, orgAdmin.ID, orgAdmin.Email, orgAdmin.OrganizationRole)

	// Execute
	err = handler.ApproveChangePlan(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Change plan approved successfully", response["message"])
	assert.Equal(t, changePlan.ID.String(), response["change_plan_id"])

	// Verify change plan status is now Approved
	updatedPlan, err := changePlanRepo.GetByID(ctx, changePlan.ID)
	require.NoError(t, err)
	assert.Equal(t, models.ChangePlanStatusApproved, updatedPlan.Status)
}
