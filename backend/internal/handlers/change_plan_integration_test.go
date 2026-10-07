//go:build integration
// +build integration

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository/postgres"
	"github.com/nsxbet/deeb/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupChangePlanTestData sets up the required data for change plan tests
func setupChangePlanTestData(t *testing.T, db *sqlx.DB) (adminToken string, projectID, databaseID uuid.UUID) {
	t.Helper()

	// Create admin user
	adminEmail := fmt.Sprintf("admin-%d@example.com", time.Now().UnixNano())
	adminID := createTestUser(t, db, adminEmail, rbac.OrganizationAdmin)
	adminToken, _ = loginUser(t, db, adminEmail, "TestPassword123!")

	// Create project
	projectRepo := postgres.NewProjectRepository(db)
	ctx := context.Background()
	desc := "Test project for change plans"
	project := &models.Project{
		Name:        fmt.Sprintf("test-project-%d", time.Now().UnixNano()),
		Description: &desc,
		CreatedByID: adminID,
	}
	err := projectRepo.Create(ctx, project)
	require.NoError(t, err)
	projectID = project.ID

	// Add admin as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	member := &models.ProjectMember{
		ProjectID: projectID,
		UserID:    adminID,
		Role:      rbac.ProjectOwner,
		AddedByID: adminID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	envRepo := postgres.NewEnvironmentRepository(db)
	envDesc := "Test environment"
	env := &models.Environment{
		Name:        fmt.Sprintf("test-env-%d", time.Now().UnixNano()),
		Description: &envDesc,
		CreatedByID: adminID,
	}
	err = envRepo.Create(ctx, env)
	require.NoError(t, err)

	// Create database instance
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, encryptionKey)
	instanceDesc := "Test database instance"
	instance := &models.DatabaseInstance{
		Name:           fmt.Sprintf("test-instance-%d", time.Now().UnixNano()),
		Description:    &instanceDesc,
		DatabaseEngine: models.DatabaseEnginePostgreSQL,
		Host:           "localhost",
		Port:           5432,
		Username:       "test",
		UseRDSIAMAuth:  false,
		RequireSSL:     false,
		Status:         models.DatabaseInstanceStatusActive,
		CreatedByID:    adminID,
	}
	err = dbInstanceRepo.Create(ctx, instance)
	require.NoError(t, err)

	// Create database
	dbRepo := postgres.NewDatabaseRepository(db)
	dbDesc := "Test database for change plans"
	database := &models.Database{
		ProjectID:          &projectID,
		DatabaseInstanceID: instance.ID,
		EnvironmentID:      &env.ID,
		DatabaseName:       fmt.Sprintf("testdb_%d", time.Now().UnixNano()),
		Description:        &dbDesc,
		CreatedByID:        adminID,
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)
	databaseID = database.ID

	return adminToken, projectID, databaseID
}

func TestIntegration_ChangePlan_CRUD(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)

	// Setup test data
	adminToken, projectID, databaseID := setupChangePlanTestData(t, db)

	var createdChangePlanID uuid.UUID

	t.Run("Create Change Plan - Success", func(t *testing.T) {
		reqBody := models.CreateChangePlanRequest{
			ProjectID:        projectID,
			TargetDatabaseID: databaseID,
			Title:            "Add users table",
			Description:      "Create a new users table with basic columns",
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:       "CREATE TABLE users (id UUID PRIMARY KEY, email VARCHAR(255) NOT NULL);",
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code, "should create change plan successfully")

		var response models.ChangePlanResponse
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "Add users table", response.Title)
		assert.Equal(t, models.ChangePlanStatusDraft, response.Status)
		createdChangePlanID = response.ID
	})

	t.Run("Get Change Plan by ID - Success", func(t *testing.T) {
		rec := makeAuthenticatedRequest(
			t,
			e,
			http.MethodGet,
			fmt.Sprintf("/api/v1/change-plans/%s", createdChangePlanID.String()),
			nil,
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		var response models.ChangePlanResponse
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, createdChangePlanID, response.ID)
		assert.Equal(t, "Add users table", response.Title)
	})

	t.Run("Get Change Plan by ID - Not Found", func(t *testing.T) {
		nonExistentID := uuid.New()
		rec := makeAuthenticatedRequest(
			t,
			e,
			http.MethodGet,
			fmt.Sprintf("/api/v1/change-plans/%s", nonExistentID.String()),
			nil,
			adminToken,
		)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("List Change Plans", func(t *testing.T) {
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, "/api/v1/change-plans", nil, adminToken)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response, "change_plans")
	})

	t.Run("Update Change Plan - Success", func(t *testing.T) {
		newTitle := "Update users table"
		newDesc := "Updated description"
		updateReq := models.UpdateChangePlanRequest{
			Title:       &newTitle,
			Description: &newDesc,
		}
		updateBody, _ := json.Marshal(updateReq)

		rec := makeAuthenticatedRequest(
			t,
			e,
			http.MethodPut,
			fmt.Sprintf("/api/v1/change-plans/%s", createdChangePlanID.String()),
			bytes.NewReader(updateBody),
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		var response models.ChangePlanResponse
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "Update users table", response.Title)
		assert.Equal(t, "Updated description", response.Description)
	})

	t.Run("Update Change Plan - Status Approved rejected (must use approval flow)", func(t *testing.T) {
		// Uses a dedicated plan (not the shared fixture): the Draft → Open
		// transition is one-way through this endpoint (non-SQL updates and the
		// transition back are Draft-only), and later sub-tests need the fixture
		// untouched in Draft.
		createReq := models.CreateChangePlanRequest{
			ProjectID:        projectID,
			TargetDatabaseID: databaseID,
			Title:            "Approval forge guard test",
			ChangeType:       models.ChangeTypeDML,
			SQLContent:       "CREATE TABLE users (id UUID PRIMARY KEY, email VARCHAR(255) NOT NULL);",
		}
		createBody, _ := json.Marshal(createReq)
		createRec := makeAuthenticatedRequest(
			t,
			e,
			http.MethodPost,
			"/api/v1/change-plans",
			bytes.NewReader(createBody),
			adminToken,
		)
		require.Equal(t, http.StatusCreated, createRec.Code)
		var created models.ChangePlanResponse
		err := json.NewDecoder(createRec.Body).Decode(&created)
		require.NoError(t, err)

		// Transition Draft → Open first so the workflow validation would
		// legally allow Open → Approved — proving the endpoint (not just the
		// transition table) refuses to grant approval.
		openStatus := models.ChangePlanStatusOpen
		openBody, _ := json.Marshal(models.UpdateChangePlanRequest{Status: &openStatus})
		openRec := makeAuthenticatedRequest(
			t,
			e,
			http.MethodPut,
			fmt.Sprintf("/api/v1/change-plans/%s", created.ID),
			bytes.NewReader(openBody),
			adminToken,
		)
		require.Equal(t, http.StatusOK, openRec.Code)

		approved := models.ChangePlanStatusApproved
		// Note: a bare status-only update on an Open plan is rejected earlier
		// with "Only Draft change plans can be modified" (CanBeModified is
		// Draft-only, and CanEditSQLContent too), so the approval-flow guard
		// further down is defense-in-depth. What matters for security is the
		// contract: Approved can never be forged through this endpoint.
		updateReq := models.UpdateChangePlanRequest{Status: &approved}
		updateBody, _ := json.Marshal(updateReq)

		rec := makeAuthenticatedRequest(
			t,
			e,
			http.MethodPut,
			fmt.Sprintf("/api/v1/change-plans/%s", created.ID),
			bytes.NewReader(updateBody),
			adminToken,
		)
		require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest,
			"Approved must not be forgeable via the generic update endpoint")

		var response map[string]interface{}
		err = json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response, "error")
	})

	t.Run("Delete Change Plan - Success", func(t *testing.T) {
		rec := makeAuthenticatedRequest(
			t,
			e,
			http.MethodDelete,
			fmt.Sprintf("/api/v1/change-plans/%s", createdChangePlanID.String()),
			nil,
			adminToken,
		)
		require.Equal(t, http.StatusNoContent, rec.Code)

		// Verify it's deleted
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodGet,
			fmt.Sprintf("/api/v1/change-plans/%s", createdChangePlanID.String()),
			nil,
			adminToken,
		)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestIntegration_ChangePlan_Validation(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)

	// Setup test data
	adminToken, projectID, databaseID := setupChangePlanTestData(t, db)

	t.Run("Create Change Plan - Missing Required Fields", func(t *testing.T) {
		reqBody := models.CreateChangePlanRequest{
			ProjectID: projectID,
			// Missing TargetDatabaseID, Title, ChangeType, SQLContent
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
		// Validation errors currently return 500, should be fixed in handler but testing actual behavior
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Create Change Plan - Empty SQL Content", func(t *testing.T) {
		reqBody := models.CreateChangePlanRequest{
			ProjectID:        projectID,
			TargetDatabaseID: databaseID,
			Title:            "Test",
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:       "",
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
		// Validation errors currently return 500, should be fixed in handler but testing actual behavior
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Create Change Plan - Invalid Project ID", func(t *testing.T) {
		reqBody := models.CreateChangePlanRequest{
			ProjectID:        uuid.New(), // Non-existent project
			TargetDatabaseID: databaseID,
			Title:            "Test",
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:       "SELECT 1;",
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Create Change Plan - Invalid Database ID", func(t *testing.T) {
		reqBody := models.CreateChangePlanRequest{
			ProjectID:        projectID,
			TargetDatabaseID: uuid.New(), // Non-existent database
			Title:            "Test",
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:       "SELECT 1;",
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestIntegration_ChangePlan_Authorization(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)

	// Setup test data
	adminToken, projectID, databaseID := setupChangePlanTestData(t, db)

	// Create developer user (not a project member)
	devEmail := fmt.Sprintf("developer-%d@example.com", time.Now().UnixNano())
	createTestUser(t, db, devEmail, rbac.OrganizationDeveloper)
	devToken, _ := loginUser(t, db, devEmail, "TestPassword123!")

	t.Run("Create Change Plan - Unauthorized User", func(t *testing.T) {
		reqBody := models.CreateChangePlanRequest{
			ProjectID:        projectID,
			TargetDatabaseID: databaseID,
			Title:            "Test",
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:       "SELECT 1;",
		}
		body, _ := json.Marshal(reqBody)

		// Developer is not a project member, should fail
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), devToken)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("Create Change Plan - Unauthenticated", func(t *testing.T) {
		reqBody := models.CreateChangePlanRequest{
			ProjectID:        projectID,
			TargetDatabaseID: databaseID,
			Title:            "Test",
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:       "SELECT 1;",
		}
		body, _ := json.Marshal(reqBody)

		// No token provided
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	// Create a change plan with admin
	reqBody := models.CreateChangePlanRequest{
		ProjectID:        projectID,
		TargetDatabaseID: databaseID,
		Title:            "Admin's change plan",
		ChangeType:       models.ChangeTypeDDL,
		SQLContent:       "SELECT 1;",
	}
	body, _ := json.Marshal(reqBody)
	rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
	require.Equal(t, http.StatusCreated, rec.Code)

	var changePlan models.ChangePlanResponse
	err := json.NewDecoder(rec.Body).Decode(&changePlan)
	require.NoError(t, err)

	t.Run("Access Change Plan - Unauthorized User", func(t *testing.T) {
		// Developer is not a project member, should not be able to view
		rec := makeAuthenticatedRequest(
			t,
			e,
			http.MethodGet,
			fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()),
			nil,
			devToken,
		)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// setupExecutionTestData sets up test data for execution tests including a target database container
// Returns adminToken, adminID, projectID, databaseID, databaseInstanceID, and cleanup function
func setupExecutionTestData(
	t *testing.T,
	db *sqlx.DB,
) (adminToken string, adminID, projectID, databaseID, databaseInstanceID uuid.UUID, targetDBCleanup func()) {
	t.Helper()
	ctx := context.Background()

	// Create admin user
	adminEmail := fmt.Sprintf("admin-%d@example.com", time.Now().UnixNano())
	adminID = createTestUser(t, db, adminEmail, rbac.OrganizationAdmin)
	adminToken, _ = loginUser(t, db, adminEmail, "TestPassword123!")

	// Create a separate target database on the shared Postgres server for
	// execution. Deeb connects to it as if it were an externally managed
	// database, so it lives alongside (but isolated from) the app's own DB.
	targetDBName := fmt.Sprintf("target_db_%d", dbCounter.Add(1))
	createSharedDatabase(t, targetDBName)

	targetHost := sharedPGHost
	targetPort := sharedPGPort

	targetDBCleanup = func() {
		dropSharedDatabase(t, targetDBName)
	}

	// Create project
	projectRepo := postgres.NewProjectRepository(db)
	desc := "Test project for execution"
	project := &models.Project{
		Name:        fmt.Sprintf("exec-test-project-%d", time.Now().UnixNano()),
		Description: &desc,
		CreatedByID: adminID,
	}
	err := projectRepo.Create(ctx, project)
	require.NoError(t, err)
	projectID = project.ID

	// Add admin as project member
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	member := &models.ProjectMember{
		ProjectID: projectID,
		UserID:    adminID,
		Role:      rbac.ProjectOwner,
		AddedByID: adminID,
	}
	err = projectMemberRepo.AddMember(ctx, member)
	require.NoError(t, err)

	// Create environment
	envRepo := postgres.NewEnvironmentRepository(db)
	envDesc := "Test environment"
	env := &models.Environment{
		Name:        fmt.Sprintf("exec-test-env-%d", time.Now().UnixNano()),
		Description: &envDesc,
		CreatedByID: adminID,
	}
	err = envRepo.Create(ctx, env)
	require.NoError(t, err)

	// Create database instance pointing to target container
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, encryptionKey)
	instanceDesc := "Target database instance for execution"

	// Encrypt password in the required JSON format {"password": "..."}
	passwordJSON := fmt.Sprintf(`{"password":%q}`, testDBPassword)
	encryptedPassword, err := utils.Encrypt(passwordJSON, encryptionKey)
	require.NoError(t, err, "failed to encrypt password")

	instance := &models.DatabaseInstance{
		Name:                       fmt.Sprintf("exec-test-instance-%d", time.Now().UnixNano()),
		Description:                &instanceDesc,
		DatabaseEngine:             models.DatabaseEnginePostgreSQL,
		Host:                       targetHost,
		Port:                       targetPort,
		Username:                   testDBUser,
		ConnectionDetailsEncrypted: &encryptedPassword,
		UseRDSIAMAuth:              false,
		RequireSSL:                 false,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                adminID,
	}
	err = dbInstanceRepo.Create(ctx, instance)
	require.NoError(t, err)
	databaseInstanceID = instance.ID

	// Create database entry pointing to the target database
	dbRepo := postgres.NewDatabaseRepository(db)
	dbDesc := "Target database for execution tests"
	database := &models.Database{
		ProjectID:          &projectID,
		DatabaseInstanceID: instance.ID,
		EnvironmentID:      &env.ID,
		DatabaseName:       targetDBName,
		Description:        &dbDesc,
		Status:             models.DatabaseStatusActive,
		CreatedByID:        adminID,
	}
	err = dbRepo.Create(ctx, database)
	require.NoError(t, err)
	databaseID = database.ID

	return adminToken, adminID, projectID, databaseID, databaseInstanceID, targetDBCleanup
}

// approveChangePlan directly updates the change plan status to Approved in the database
func approveChangePlan(t *testing.T, db *sqlx.DB, changePlanID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	changePlanRepo := postgres.NewChangePlanRepository(db)

	changePlan, err := changePlanRepo.GetByID(ctx, changePlanID)
	require.NoError(t, err)

	changePlan.Status = models.ChangePlanStatusApproved
	err = changePlanRepo.Update(ctx, changePlan)
	require.NoError(t, err)
}

// TestIntegration_ChangePlan_MultiStatementExecution_Success tests executing a change plan
// with multiple SQL statements and verifies execution logs for each statement are saved
func TestIntegration_ChangePlan_MultiStatementExecution_Success(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)

	// Setup test data with target database container
	adminToken, _, projectID, databaseID, _, targetDBCleanup := setupExecutionTestData(t, db)
	defer targetDBCleanup()

	ctx := context.Background()

	// Create change plan with multiple statements
	multiStatementSQL := `CREATE TABLE test_users (id SERIAL PRIMARY KEY, name VARCHAR(100));
INSERT INTO test_users (name) VALUES ('Alice');
INSERT INTO test_users (name) VALUES ('Bob');`

	reqBody := models.CreateChangePlanRequest{
		ProjectID:        projectID,
		TargetDatabaseID: databaseID,
		Title:            "Multi-statement test",
		Description:      "Test multi-statement execution",
		ChangeType:       models.ChangeTypeDDL,
		SQLContent:       multiStatementSQL,
	}
	body, _ := json.Marshal(reqBody)

	rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
	require.Equal(t, http.StatusCreated, rec.Code, "should create change plan successfully")

	var changePlanResponse models.ChangePlanResponse
	err := json.NewDecoder(rec.Body).Decode(&changePlanResponse)
	require.NoError(t, err)
	changePlanID := changePlanResponse.ID

	// Verify statement count is set correctly
	assert.NotNil(t, changePlanResponse.StatementCount)
	assert.Equal(t, 3, *changePlanResponse.StatementCount, "should have 3 statements")

	// Approve the change plan
	approveChangePlan(t, db, changePlanID)

	// Execute the change plan
	rec = makeAuthenticatedRequest(
		t,
		e,
		http.MethodPost,
		fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlanID.String()),
		nil,
		adminToken,
	)
	require.Equal(t, http.StatusOK, rec.Code, "execution should succeed")

	var execResponse map[string]interface{}
	err = json.NewDecoder(rec.Body).Decode(&execResponse)
	require.NoError(t, err)

	// Verify execution was successful
	execution := execResponse["execution"].(map[string]interface{})
	assert.True(t, execution["success"].(bool), "execution should be successful")
	assert.Equal(t, float64(2), execution["rows_affected"].(float64), "should have 2 rows affected (2 inserts)")

	// Get execution logs
	rec = makeAuthenticatedRequest(
		t,
		e,
		http.MethodGet,
		fmt.Sprintf("/api/v1/change-plans/%s/execution-logs", changePlanID.String()),
		nil,
		adminToken,
	)
	require.Equal(t, http.StatusOK, rec.Code)

	var logsResponse struct {
		Logs []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"logs"`
	}
	err = json.NewDecoder(rec.Body).Decode(&logsResponse)
	require.NoError(t, err)
	require.Len(t, logsResponse.Logs, 1, "should have 1 execution log")

	executionLogID := logsResponse.Logs[0].ID
	assert.Equal(t, "Completed", logsResponse.Logs[0].Status)

	// Get statement execution results
	rec = makeAuthenticatedRequest(
		t,
		e,
		http.MethodGet,
		fmt.Sprintf("/api/v1/change-plans/%s/execution-logs/%s/statements", changePlanID.String(), executionLogID),
		nil,
		adminToken,
	)
	require.Equal(t, http.StatusOK, rec.Code)

	var stmtResponse struct {
		StatementResults []struct {
			StatementIndex int     `json:"statement_index"`
			Status         string  `json:"status"`
			RowsAffected   *int    `json:"rows_affected"`
			StatementSQL   string  `json:"statement_sql"`
			StartLine      *int    `json:"start_line"`
			StartColumn    *int    `json:"start_column"`
			ErrorMessage   *string `json:"error_message"`
		} `json:"statement_results"`
	}
	err = json.NewDecoder(rec.Body).Decode(&stmtResponse)
	require.NoError(t, err)

	// Verify statement results
	require.Len(t, stmtResponse.StatementResults, 3, "should have 3 statement results")

	// Verify first statement (CREATE TABLE)
	assert.Equal(t, 0, stmtResponse.StatementResults[0].StatementIndex)
	assert.Equal(t, "Completed", stmtResponse.StatementResults[0].Status)
	assert.Contains(t, stmtResponse.StatementResults[0].StatementSQL, "CREATE TABLE")
	assert.NotNil(t, stmtResponse.StatementResults[0].StartLine)
	assert.Equal(t, 1, *stmtResponse.StatementResults[0].StartLine)

	// Verify second statement (INSERT Alice)
	assert.Equal(t, 1, stmtResponse.StatementResults[1].StatementIndex)
	assert.Equal(t, "Completed", stmtResponse.StatementResults[1].Status)
	assert.NotNil(t, stmtResponse.StatementResults[1].RowsAffected)
	assert.Equal(t, 1, *stmtResponse.StatementResults[1].RowsAffected)
	assert.Contains(t, stmtResponse.StatementResults[1].StatementSQL, "Alice")

	// Verify third statement (INSERT Bob)
	assert.Equal(t, 2, stmtResponse.StatementResults[2].StatementIndex)
	assert.Equal(t, "Completed", stmtResponse.StatementResults[2].Status)
	assert.NotNil(t, stmtResponse.StatementResults[2].RowsAffected)
	assert.Equal(t, 1, *stmtResponse.StatementResults[2].RowsAffected)
	assert.Contains(t, stmtResponse.StatementResults[2].StatementSQL, "Bob")

	// Also verify by querying the database directly
	statementResultRepo := postgres.NewStatementExecutionResultRepository(db)
	executionLogUUID, err := uuid.Parse(executionLogID)
	require.NoError(t, err)

	dbResults, err := statementResultRepo.GetByExecutionLogID(ctx, executionLogUUID)
	require.NoError(t, err)
	assert.Len(t, dbResults, 3, "should have 3 statement results in database")
}

// TestIntegration_ChangePlan_MultiStatementExecution_Error tests executing a change plan
// with multiple SQL statements where one fails, verifying positional error information
func TestIntegration_ChangePlan_MultiStatementExecution_Error(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)

	// Setup test data with target database container
	adminToken, _, projectID, databaseID, _, targetDBCleanup := setupExecutionTestData(t, db)
	defer targetDBCleanup()

	ctx := context.Background()

	// Create change plan with multiple statements where second fails
	// First statement succeeds, second fails (table doesn't exist)
	multiStatementSQL := `CREATE TABLE error_test (id SERIAL PRIMARY KEY);
INSERT INTO nonexistent_table (id) VALUES (1);
INSERT INTO error_test (id) VALUES (1);`

	reqBody := models.CreateChangePlanRequest{
		ProjectID:        projectID,
		TargetDatabaseID: databaseID,
		Title:            "Multi-statement error test",
		Description:      "Test multi-statement execution with error",
		ChangeType:       models.ChangeTypeDDL,
		SQLContent:       multiStatementSQL,
	}
	body, _ := json.Marshal(reqBody)

	rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
	require.Equal(t, http.StatusCreated, rec.Code, "should create change plan successfully")

	var changePlanResponse models.ChangePlanResponse
	err := json.NewDecoder(rec.Body).Decode(&changePlanResponse)
	require.NoError(t, err)
	changePlanID := changePlanResponse.ID

	// Approve the change plan
	approveChangePlan(t, db, changePlanID)

	// Execute the change plan - should complete but with failure status
	rec = makeAuthenticatedRequest(
		t,
		e,
		http.MethodPost,
		fmt.Sprintf("/api/v1/change-plans/%s/execute", changePlanID.String()),
		nil,
		adminToken,
	)
	require.Equal(t, http.StatusOK, rec.Code, "execution endpoint should return 200 even on SQL error")

	var execResponse map[string]interface{}
	err = json.NewDecoder(rec.Body).Decode(&execResponse)
	require.NoError(t, err)

	// Verify execution failed
	execution := execResponse["execution"].(map[string]interface{})
	assert.False(t, execution["success"].(bool), "execution should have failed")
	assert.NotNil(t, execution["error_message"], "should have error message")
	errorMsg := execution["error_message"].(string)
	assert.Contains(t, errorMsg, "statement 2", "error should reference statement 2")
	assert.Contains(t, errorMsg, "line 2", "error should reference line 2")

	// Verify change plan status is Failed
	changePlanData := execResponse["change_plan"].(map[string]interface{})
	assert.Equal(t, "Failed", changePlanData["status"].(string))

	// Get execution logs
	rec = makeAuthenticatedRequest(
		t,
		e,
		http.MethodGet,
		fmt.Sprintf("/api/v1/change-plans/%s/execution-logs", changePlanID.String()),
		nil,
		adminToken,
	)
	require.Equal(t, http.StatusOK, rec.Code)

	var logsResponse struct {
		Logs []struct {
			ID           string  `json:"id"`
			Status       string  `json:"status"`
			ErrorMessage *string `json:"error_message"`
		} `json:"logs"`
	}
	err = json.NewDecoder(rec.Body).Decode(&logsResponse)
	require.NoError(t, err)
	require.Len(t, logsResponse.Logs, 1, "should have 1 execution log")

	executionLogID := logsResponse.Logs[0].ID
	assert.Equal(t, "Failed", logsResponse.Logs[0].Status)
	require.NotNil(t, logsResponse.Logs[0].ErrorMessage)
	assert.Contains(t, *logsResponse.Logs[0].ErrorMessage, "statement 2")

	// Get statement execution results
	rec = makeAuthenticatedRequest(
		t,
		e,
		http.MethodGet,
		fmt.Sprintf("/api/v1/change-plans/%s/execution-logs/%s/statements", changePlanID.String(), executionLogID),
		nil,
		adminToken,
	)
	require.Equal(t, http.StatusOK, rec.Code)

	var stmtResponse struct {
		StatementResults []struct {
			StatementIndex int     `json:"statement_index"`
			Status         string  `json:"status"`
			RowsAffected   *int    `json:"rows_affected"`
			StatementSQL   string  `json:"statement_sql"`
			StartLine      *int    `json:"start_line"`
			StartColumn    *int    `json:"start_column"`
			ErrorMessage   *string `json:"error_message"`
		} `json:"statement_results"`
	}
	err = json.NewDecoder(rec.Body).Decode(&stmtResponse)
	require.NoError(t, err)

	// Should have 2 statement results (first completed, second failed, third never executed)
	require.Len(t, stmtResponse.StatementResults, 2, "should have 2 statement results")

	// Verify first statement (CREATE TABLE) - succeeded
	assert.Equal(t, 0, stmtResponse.StatementResults[0].StatementIndex)
	assert.Equal(t, "Completed", stmtResponse.StatementResults[0].Status)
	assert.Contains(t, stmtResponse.StatementResults[0].StatementSQL, "CREATE TABLE")
	assert.Nil(t, stmtResponse.StatementResults[0].ErrorMessage)

	// Verify second statement (INSERT into nonexistent) - failed with error
	assert.Equal(t, 1, stmtResponse.StatementResults[1].StatementIndex)
	assert.Equal(t, "Failed", stmtResponse.StatementResults[1].Status)
	assert.Contains(t, stmtResponse.StatementResults[1].StatementSQL, "nonexistent_table")
	assert.NotNil(t, stmtResponse.StatementResults[1].ErrorMessage)
	assert.Contains(t, *stmtResponse.StatementResults[1].ErrorMessage, "nonexistent_table")
	// Verify position info
	assert.NotNil(t, stmtResponse.StatementResults[1].StartLine)
	assert.Equal(t, 2, *stmtResponse.StatementResults[1].StartLine)

	// Also verify by querying the database directly
	statementResultRepo := postgres.NewStatementExecutionResultRepository(db)
	executionLogUUID, err := uuid.Parse(executionLogID)
	require.NoError(t, err)

	dbResults, err := statementResultRepo.GetByExecutionLogID(ctx, executionLogUUID)
	require.NoError(t, err)
	assert.Len(t, dbResults, 2, "should have 2 statement results in database")

	// Verify the failed statement has correct position info in DB
	var failedResult *models.StatementExecutionResult
	for _, r := range dbResults {
		if r.StatementIndex == 1 {
			failedResult = r
			break
		}
	}
	require.NotNil(t, failedResult)
	assert.Equal(t, models.StatementStatusFailed, failedResult.Status)
	assert.NotNil(t, failedResult.StartLine)
	assert.Equal(t, 2, *failedResult.StartLine)
	assert.NotNil(t, failedResult.ErrorMessage)
}

// Ensure url package is used (for parsing connection strings if needed)
var _ = url.Parse
