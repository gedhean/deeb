//go:build integration
// +build integration

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/nsxbet/deeb/internal/handlers"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository/postgres"
	"github.com/nsxbet/deeb/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_AuditLogging_ResourceCRUD tests audit logging for resource CRUD operations
func TestIntegration_AuditLogging_ResourceCRUD(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)
	auditLogRepo := postgres.NewAuditLogRepository(db)
	ctx := context.Background()

	// Create admin user
	adminEmail := fmt.Sprintf("admin-%d@example.com", time.Now().Unix())
	adminID := createTestUser(t, db, adminEmail, models.OrganizationRoleAdmin)
	adminToken, _ := loginUser(t, db, adminEmail, "TestPassword123!")

	t.Run("Project Create Audit Logging", func(t *testing.T) {
		reqBody := models.CreateProjectRequest{
			Name:        "Test Project for Audit",
			Description: "Test Description",
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var project models.ProjectResponse
		err := json.NewDecoder(rec.Body).Decode(&project)
		require.NoError(t, err)

		// Verify audit log entry was created
		auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionCreate, 10, 0)
		require.NoError(t, err)

		var found bool
		for _, log := range auditLogs {
			if log.ResourceType == services.ResourceTypeProject && log.ResourceID == project.ID {
				found = true
				assert.Equal(t, adminID, log.UserID)
				assert.Equal(t, services.ActionCreate, log.Action)
				assert.Equal(t, services.ResourceTypeProject, log.ResourceType)
				assert.Equal(t, project.ID, log.ResourceID)
				assert.NotNil(t, log.IPAddress)
				assert.NotZero(t, log.Timestamp)
				break
			}
		}
		assert.True(t, found, "Create project audit log entry should be created")
	})

	t.Run("Project Update Audit Logging", func(t *testing.T) {
		// Create a project first
		reqBody := models.CreateProjectRequest{
			Name: "Update Test Project",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var project models.ProjectResponse
		err := json.NewDecoder(rec.Body).Decode(&project)
		require.NoError(t, err)

		// Update the project
		updateReq := models.UpdateProjectRequest{
			Name:        stringPtr("Updated Project Name"),
			Description: stringPtr("Updated Description"),
		}
		updateBody, _ := json.Marshal(updateReq)
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodPut,
			fmt.Sprintf("/api/v1/projects/%s", project.ID.String()),
			bytes.NewReader(updateBody),
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		// Verify audit log entry was created
		auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionUpdate, 10, 0)
		require.NoError(t, err)

		var found bool
		for _, log := range auditLogs {
			if log.ResourceType == services.ResourceTypeProject && log.ResourceID == project.ID {
				found = true
				assert.Equal(t, adminID, log.UserID)
				assert.Equal(t, services.ActionUpdate, log.Action)
				assert.Equal(t, services.ResourceTypeProject, log.ResourceType)
				assert.Equal(t, project.ID, log.ResourceID)
				assert.NotNil(t, log.IPAddress)
				assert.NotZero(t, log.Timestamp)
				break
			}
		}
		assert.True(t, found, "Update project audit log entry should be created")
	})

	t.Run("Project Delete Audit Logging", func(t *testing.T) {
		// Create a project first
		reqBody := models.CreateProjectRequest{
			Name: "Delete Test Project",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var project models.ProjectResponse
		err := json.NewDecoder(rec.Body).Decode(&project)
		require.NoError(t, err)

		// Delete the project
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodDelete,
			fmt.Sprintf("/api/v1/projects/%s", project.ID.String()),
			nil,
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		// Verify audit log entry was created
		auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionDelete, 10, 0)
		require.NoError(t, err)

		var found bool
		for _, log := range auditLogs {
			if log.ResourceType == services.ResourceTypeProject && log.ResourceID == project.ID {
				found = true
				assert.Equal(t, adminID, log.UserID)
				assert.Equal(t, services.ActionDelete, log.Action)
				assert.Equal(t, services.ResourceTypeProject, log.ResourceType)
				assert.Equal(t, project.ID, log.ResourceID)
				assert.NotNil(t, log.IPAddress)
				assert.NotZero(t, log.Timestamp)
				break
			}
		}
		assert.True(t, found, "Delete project audit log entry should be created")
	})

	t.Run("Environment Create Audit Logging", func(t *testing.T) {
		reqBody := models.CreateEnvironmentRequest{
			Name:        "Test Environment for Audit",
			Description: stringPtr("Test Description"),
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/environments", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var env models.EnvironmentResponse
		err := json.NewDecoder(rec.Body).Decode(&env)
		require.NoError(t, err)

		// Verify audit log entry was created
		auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionCreate, 10, 0)
		require.NoError(t, err)

		var found bool
		for _, log := range auditLogs {
			if log.ResourceType == services.ResourceTypeEnvironment && log.ResourceID == env.ID {
				found = true
				assert.Equal(t, adminID, log.UserID)
				assert.Equal(t, services.ActionCreate, log.Action)
				assert.Equal(t, services.ResourceTypeEnvironment, log.ResourceType)
				assert.Equal(t, env.ID, log.ResourceID)
				assert.NotNil(t, log.IPAddress)
				assert.NotZero(t, log.Timestamp)
				break
			}
		}
		assert.True(t, found, "Create environment audit log entry should be created")
	})

	t.Run("Environment Update Audit Logging", func(t *testing.T) {
		// Create an environment first
		reqBody := models.CreateEnvironmentRequest{
			Name: "Update Test Environment",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/environments", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var env models.EnvironmentResponse
		err := json.NewDecoder(rec.Body).Decode(&env)
		require.NoError(t, err)

		// Update the environment
		updateReq := models.UpdateEnvironmentRequest{
			Name:        stringPtr("Updated Environment Name"),
			Description: stringPtr("Updated Description"),
		}
		updateBody, _ := json.Marshal(updateReq)
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodPut,
			fmt.Sprintf("/api/v1/environments/%s", env.ID.String()),
			bytes.NewReader(updateBody),
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		// Verify audit log entry was created
		auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionUpdate, 10, 0)
		require.NoError(t, err)

		var found bool
		for _, log := range auditLogs {
			if log.ResourceType == services.ResourceTypeEnvironment && log.ResourceID == env.ID {
				found = true
				assert.Equal(t, adminID, log.UserID)
				assert.Equal(t, services.ActionUpdate, log.Action)
				assert.Equal(t, services.ResourceTypeEnvironment, log.ResourceType)
				assert.Equal(t, env.ID, log.ResourceID)
				assert.NotNil(t, log.IPAddress)
				assert.NotZero(t, log.Timestamp)
				break
			}
		}
		assert.True(t, found, "Update environment audit log entry should be created")
	})

	t.Run("Environment Delete Audit Logging", func(t *testing.T) {
		// Create an environment first
		reqBody := models.CreateEnvironmentRequest{
			Name: "Delete Test Environment",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/environments", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var env models.EnvironmentResponse
		err := json.NewDecoder(rec.Body).Decode(&env)
		require.NoError(t, err)

		// Delete the environment
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodDelete,
			fmt.Sprintf("/api/v1/environments/%s", env.ID.String()),
			nil,
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		// Verify audit log entry was created
		auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionDelete, 10, 0)
		require.NoError(t, err)

		var found bool
		for _, log := range auditLogs {
			if log.ResourceType == services.ResourceTypeEnvironment && log.ResourceID == env.ID {
				found = true
				assert.Equal(t, adminID, log.UserID)
				assert.Equal(t, services.ActionDelete, log.Action)
				assert.Equal(t, services.ResourceTypeEnvironment, log.ResourceType)
				assert.Equal(t, env.ID, log.ResourceID)
				assert.NotNil(t, log.IPAddress)
				assert.NotZero(t, log.Timestamp)
				break
			}
		}
		assert.True(t, found, "Delete environment audit log entry should be created")
	})

	t.Run("Database Instance Create Audit Logging", func(t *testing.T) {
		reqBody := models.CreateDatabaseInstanceRequest{
			Name:          "Test Database Instance for Audit",
			Description:   stringPtr("Test Description"),
			DatabaseType:  models.DatabaseTypePostgreSQL,
			Host:          "localhost",
			Port:          5432,
			Username:      "testuser",
			Password:      "testpassword",
			UseRDSIAMAuth: false,
			RequireSSL:    false,
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/database-instances", bytes.NewReader(body), adminToken)
		// Connection test may fail, but audit log should still be created if instance is created
		// Check if instance was created (StatusCreated) or connection test failed (StatusBadRequest)
		if rec.Code == http.StatusCreated {
			var instance models.DatabaseInstanceResponse
			err := json.NewDecoder(rec.Body).Decode(&instance)
			require.NoError(t, err)

			// Verify audit log entry was created
			auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionCreate, 10, 0)
			require.NoError(t, err)

			var found bool
			for _, log := range auditLogs {
				if log.ResourceType == services.ResourceTypeDatabaseInstance && log.ResourceID == instance.ID {
					found = true
					assert.Equal(t, adminID, log.UserID)
					assert.Equal(t, services.ActionCreate, log.Action)
					assert.Equal(t, services.ResourceTypeDatabaseInstance, log.ResourceType)
					assert.Equal(t, instance.ID, log.ResourceID)
					assert.NotNil(t, log.IPAddress)
					assert.NotZero(t, log.Timestamp)
					break
				}
			}
			assert.True(t, found, "Create database instance audit log entry should be created")
		}
	})

	t.Run("Database Create Audit Logging", func(t *testing.T) {
		// Create project and environment first
		projectReq := models.CreateProjectRequest{
			Name: "Test Project for Database Audit",
		}
		projectBody, _ := json.Marshal(projectReq)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(projectBody), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)
		var project models.ProjectResponse
		err := json.NewDecoder(rec.Body).Decode(&project)
		require.NoError(t, err)

		envReq := models.CreateEnvironmentRequest{
			Name: "Test Environment for Database Audit",
		}
		envBody, _ := json.Marshal(envReq)
		rec = makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/environments", bytes.NewReader(envBody), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)
		var env models.EnvironmentResponse
		err = json.NewDecoder(rec.Body).Decode(&env)
		require.NoError(t, err)

		// Create database instance (directly insert to bypass connection test)
		dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "12345678901234567890123456789012")
		encryptedDetails := "encrypted-details"
		instance := &models.DatabaseInstance{
			Name:                       "Test Instance for Audit",
			Description:                stringPtr("Test Description"),
			DatabaseType:               models.DatabaseTypePostgreSQL,
			Host:                       "localhost",
			Port:                       5432,
			Username:                   "testuser",
			ConnectionDetailsEncrypted: &encryptedDetails,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                adminID,
		}
		err = dbInstanceRepo.Create(ctx, instance)
		require.NoError(t, err)

		// Create database
		dbReq := models.CreateDatabaseRequest{
			ProjectID:          project.ID,
			EnvironmentID:      env.ID,
			DatabaseInstanceID: instance.ID,
			DatabaseName:       "testdb",
			Description:        stringPtr("Test Database"),
		}
		dbBody, _ := json.Marshal(dbReq)
		rec = makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/databases", bytes.NewReader(dbBody), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var database models.DatabaseResponse
		err = json.NewDecoder(rec.Body).Decode(&database)
		require.NoError(t, err)

		// Verify audit log entry was created
		auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionCreate, 10, 0)
		require.NoError(t, err)

		var found bool
		for _, log := range auditLogs {
			if log.ResourceType == services.ResourceTypeDatabase && log.ResourceID == database.ID {
				found = true
				assert.Equal(t, adminID, log.UserID)
				assert.Equal(t, services.ActionCreate, log.Action)
				assert.Equal(t, services.ResourceTypeDatabase, log.ResourceType)
				assert.Equal(t, database.ID, log.ResourceID)
				assert.NotNil(t, log.IPAddress)
				assert.NotZero(t, log.Timestamp)
				break
			}
		}
		assert.True(t, found, "Create database audit log entry should be created")
	})
}

// TestIntegration_AuditLogging_ChangePlanCRUD tests audit logging for Change Plan CRUD operations
func TestIntegration_AuditLogging_ChangePlanCRUD(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)
	auditLogRepo := postgres.NewAuditLogRepository(db)
	ctx := context.Background()

	// Create admin user
	adminEmail := fmt.Sprintf("admin-%d@example.com", time.Now().Unix())
	adminID := createTestUser(t, db, adminEmail, models.OrganizationRoleAdmin)
	adminToken, _ := loginUser(t, db, adminEmail, "TestPassword123!")

	// Create project
	projectReq := models.CreateProjectRequest{
		Name: "Test Project for Change Plan Audit",
	}
	projectBody, _ := json.Marshal(projectReq)
	rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(projectBody), adminToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var project models.ProjectResponse
	err := json.NewDecoder(rec.Body).Decode(&project)
	require.NoError(t, err)

	// Create environment
	envReq := models.CreateEnvironmentRequest{
		Name: "Test Environment for Change Plan Audit",
	}
	envBody, _ := json.Marshal(envReq)
	rec = makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/environments", bytes.NewReader(envBody), adminToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var env models.EnvironmentResponse
	err = json.NewDecoder(rec.Body).Decode(&env)
	require.NoError(t, err)

	// Create database instance (directly insert to bypass connection test)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "12345678901234567890123456789012")
	encryptedDetails := "encrypted-details"
	instance := &models.DatabaseInstance{
		Name:                       "Test Instance for Change Plan Audit",
		Description:                stringPtr("Test Description"),
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       "localhost",
		Port:                       5432,
		Username:                   "testuser",
		ConnectionDetailsEncrypted: &encryptedDetails,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                adminID,
	}
	err = dbInstanceRepo.Create(ctx, instance)
	require.NoError(t, err)

	// Create database
	dbReq := models.CreateDatabaseRequest{
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: instance.ID,
		DatabaseName:       "testdb",
		Description:        stringPtr("Test Database"),
	}
	dbBody, _ := json.Marshal(dbReq)
	rec = makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/databases", bytes.NewReader(dbBody), adminToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var database models.DatabaseResponse
	err = json.NewDecoder(rec.Body).Decode(&database)
	require.NoError(t, err)

	t.Run("Change Plan Create Audit Logging", func(t *testing.T) {
		reqBody := models.CreateChangePlanRequest{
			Title:            "Test Change Plan for Audit",
			Description:      "Test Description",
			TargetDatabaseID: database.ID,
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:       "CREATE TABLE test (id UUID PRIMARY KEY);",
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var changePlan models.ChangePlanResponse
		err := json.NewDecoder(rec.Body).Decode(&changePlan)
		require.NoError(t, err)

		// Verify audit log entry was created
		auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionCreate, 10, 0)
		require.NoError(t, err)

		var found bool
		for _, log := range auditLogs {
			if log.ResourceType == services.ResourceTypeChangePlan && log.ResourceID == changePlan.ID {
				found = true
				assert.Equal(t, adminID, log.UserID)
				assert.Equal(t, services.ActionCreate, log.Action)
				assert.Equal(t, services.ResourceTypeChangePlan, log.ResourceType)
				assert.Equal(t, changePlan.ID, log.ResourceID)
				assert.NotNil(t, log.IPAddress)
				assert.NotZero(t, log.Timestamp)
				break
			}
		}
		assert.True(t, found, "Create change plan audit log entry should be created")
	})

	t.Run("Change Plan Update Audit Logging", func(t *testing.T) {
		// Create a change plan first
		reqBody := models.CreateChangePlanRequest{
			Title:            "Update Test Change Plan",
			Description:      "Test Description",
			TargetDatabaseID: database.ID,
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:       "CREATE TABLE test2 (id UUID PRIMARY KEY);",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var changePlan models.ChangePlanResponse
		err := json.NewDecoder(rec.Body).Decode(&changePlan)
		require.NoError(t, err)

		// Update the change plan
		updateReq := models.UpdateChangePlanRequest{
			Title:       stringPtr("Updated Change Plan Title"),
			Description: stringPtr("Updated Description"),
		}
		updateBody, _ := json.Marshal(updateReq)
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodPut,
			fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()),
			bytes.NewReader(updateBody),
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		// Verify audit log entry was created
		auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionUpdate, 10, 0)
		require.NoError(t, err)

		var found bool
		for _, log := range auditLogs {
			if log.ResourceType == services.ResourceTypeChangePlan && log.ResourceID == changePlan.ID {
				found = true
				assert.Equal(t, adminID, log.UserID)
				assert.Equal(t, services.ActionUpdate, log.Action)
				assert.Equal(t, services.ResourceTypeChangePlan, log.ResourceType)
				assert.Equal(t, changePlan.ID, log.ResourceID)
				assert.NotNil(t, log.IPAddress)
				assert.NotZero(t, log.Timestamp)
				break
			}
		}
		assert.True(t, found, "Update change plan audit log entry should be created")
	})

	t.Run("Change Plan Delete Audit Logging", func(t *testing.T) {
		// Create a change plan first
		reqBody := models.CreateChangePlanRequest{
			Title:            "Delete Test Change Plan",
			Description:      "Test Description",
			TargetDatabaseID: database.ID,
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:       "CREATE TABLE test3 (id UUID PRIMARY KEY);",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var changePlan models.ChangePlanResponse
		err := json.NewDecoder(rec.Body).Decode(&changePlan)
		require.NoError(t, err)

		// Delete the change plan
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodDelete,
			fmt.Sprintf("/api/v1/change-plans/%s", changePlan.ID.String()),
			nil,
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		// Verify audit log entry was created
		auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionDelete, 10, 0)
		require.NoError(t, err)

		var found bool
		for _, log := range auditLogs {
			if log.ResourceType == services.ResourceTypeChangePlan && log.ResourceID == changePlan.ID {
				found = true
				assert.Equal(t, adminID, log.UserID)
				assert.Equal(t, services.ActionDelete, log.Action)
				assert.Equal(t, services.ResourceTypeChangePlan, log.ResourceType)
				assert.Equal(t, changePlan.ID, log.ResourceID)
				assert.NotNil(t, log.IPAddress)
				assert.NotZero(t, log.Timestamp)
				break
			}
		}
		assert.True(t, found, "Delete change plan audit log entry should be created")
	})
}

// TestIntegration_AuditLogging_ChangePlanStateTransitions tests audit logging for Change Plan state transitions
func TestIntegration_AuditLogging_ChangePlanStateTransitions(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)
	auditLogRepo := postgres.NewAuditLogRepository(db)
	ctx := context.Background()

	// Create admin user
	adminEmail := fmt.Sprintf("admin-%d@example.com", time.Now().Unix())
	adminID := createTestUser(t, db, adminEmail, models.OrganizationRoleAdmin)
	adminToken, _ := loginUser(t, db, adminEmail, "TestPassword123!")

	// Create project
	projectReq := models.CreateProjectRequest{
		Name: "Test Project for State Transitions",
	}
	projectBody, _ := json.Marshal(projectReq)
	rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(projectBody), adminToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var project models.ProjectResponse
	err := json.NewDecoder(rec.Body).Decode(&project)
	require.NoError(t, err)

	// Create environment
	envReq := models.CreateEnvironmentRequest{
		Name: "Test Environment for State Transitions",
	}
	envBody, _ := json.Marshal(envReq)
	rec = makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/environments", bytes.NewReader(envBody), adminToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var env models.EnvironmentResponse
	err = json.NewDecoder(rec.Body).Decode(&env)
	require.NoError(t, err)

	// Create database instance (directly insert to bypass connection test)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, "12345678901234567890123456789012")
	encryptedDetails := "encrypted-details"
	instance := &models.DatabaseInstance{
		Name:                       "Test Instance for State Transitions",
		Description:                stringPtr("Test Description"),
		DatabaseType:               models.DatabaseTypePostgreSQL,
		Host:                       "localhost",
		Port:                       5432,
		Username:                   "testuser",
		ConnectionDetailsEncrypted: &encryptedDetails,
		Status:                     models.DatabaseInstanceStatusActive,
		CreatedByID:                adminID,
	}
	err = dbInstanceRepo.Create(ctx, instance)
	require.NoError(t, err)

	// Create database
	dbReq := models.CreateDatabaseRequest{
		ProjectID:          project.ID,
		EnvironmentID:      env.ID,
		DatabaseInstanceID: instance.ID,
		DatabaseName:       "testdb",
		Description:        stringPtr("Test Database"),
	}
	dbBody, _ := json.Marshal(dbReq)
	rec = makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/databases", bytes.NewReader(dbBody), adminToken)
	require.Equal(t, http.StatusCreated, rec.Code)
	var database models.DatabaseResponse
	err = json.NewDecoder(rec.Body).Decode(&database)
	require.NoError(t, err)

	t.Run("Change Plan Approve Audit Logging", func(t *testing.T) {
		// Create a change plan first
		reqBody := models.CreateChangePlanRequest{
			Title:            "Approve Test Change Plan",
			Description:      "Test Description",
			TargetDatabaseID: database.ID,
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:       "CREATE TABLE test_approve (id UUID PRIMARY KEY);",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var changePlan models.ChangePlanResponse
		err := json.NewDecoder(rec.Body).Decode(&changePlan)
		require.NoError(t, err)

		// Approve the change plan
		approveReq := handlers.ApproveChangePlanRequest{
			Comments: stringPtr("Approved for testing"),
		}
		approveBody, _ := json.Marshal(approveReq)
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodPost,
			fmt.Sprintf("/api/v1/change-plans/%s/approve", changePlan.ID.String()),
			bytes.NewReader(approveBody),
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		// Verify audit log entry was created for approval
		auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionApproveChangePlan, 10, 0)
		require.NoError(t, err)

		var found bool
		for _, log := range auditLogs {
			if log.ResourceType == services.ResourceTypeChangePlan && log.ResourceID == changePlan.ID {
				found = true
				assert.Equal(t, adminID, log.UserID)
				assert.Equal(t, services.ActionApproveChangePlan, log.Action)
				assert.Equal(t, services.ResourceTypeChangePlan, log.ResourceType)
				assert.Equal(t, changePlan.ID, log.ResourceID)
				assert.NotNil(t, log.IPAddress)
				assert.NotZero(t, log.Timestamp)
				break
			}
		}
		assert.True(t, found, "Approve change plan audit log entry should be created")
	})

	t.Run("Change Plan Close Audit Logging", func(t *testing.T) {
		// Create a change plan first
		reqBody := models.CreateChangePlanRequest{
			Title:            "Close Test Change Plan",
			Description:      "Test Description",
			TargetDatabaseID: database.ID,
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:       "CREATE TABLE test_close (id UUID PRIMARY KEY);",
		}
		body, _ := json.Marshal(reqBody)
		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans", bytes.NewReader(body), adminToken)
		require.Equal(t, http.StatusCreated, rec.Code)

		var changePlan models.ChangePlanResponse
		err := json.NewDecoder(rec.Body).Decode(&changePlan)
		require.NoError(t, err)

		// Close the change plan
		closeReq := handlers.CloseChangePlanRequest{
			Comments: stringPtr("Closed for testing"),
		}
		closeBody, _ := json.Marshal(closeReq)
		rec = makeAuthenticatedRequest(
			t,
			e,
			http.MethodPost,
			fmt.Sprintf("/api/v1/change-plans/%s/close", changePlan.ID.String()),
			bytes.NewReader(closeBody),
			adminToken,
		)
		require.Equal(t, http.StatusOK, rec.Code)

		// Verify audit log entry was created for close
		auditLogs, err := auditLogRepo.GetByAction(ctx, services.ActionCloseChangePlan, 10, 0)
		require.NoError(t, err)

		var found bool
		for _, log := range auditLogs {
			if log.ResourceType == services.ResourceTypeChangePlan && log.ResourceID == changePlan.ID {
				found = true
				assert.Equal(t, adminID, log.UserID)
				assert.Equal(t, services.ActionCloseChangePlan, log.Action)
				assert.Equal(t, services.ResourceTypeChangePlan, log.ResourceType)
				assert.Equal(t, changePlan.ID, log.ResourceID)
				assert.NotNil(t, log.IPAddress)
				assert.NotZero(t, log.Timestamp)
				break
			}
		}
		assert.True(t, found, "Close change plan audit log entry should be created")
	})
}
