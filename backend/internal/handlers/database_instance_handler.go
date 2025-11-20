package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/middleware"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/utils"
	"go.uber.org/zap"
)

// DatabaseInstanceHandler handles HTTP requests for database instances
type DatabaseInstanceHandler struct {
	dbInstanceRepo  repository.DatabaseInstanceRepository
	databaseRepo    repository.DatabaseRepository // For cascade delete checks
	projectRepo     repository.ProjectRepository
	environmentRepo repository.EnvironmentRepository
	changePlanRepo  repository.ChangePlanRepository
	connManager     *database.ConnectionManager
	encryptionKey   string
	logger          *zap.Logger
}

// NewDatabaseInstanceHandler creates a new database instance handler
func NewDatabaseInstanceHandler(
	dbInstanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	projectRepo repository.ProjectRepository,
	environmentRepo repository.EnvironmentRepository,
	changePlanRepo repository.ChangePlanRepository,
	connManager *database.ConnectionManager,
	encryptionKey string,
	logger *zap.Logger,
) *DatabaseInstanceHandler {
	return &DatabaseInstanceHandler{
		dbInstanceRepo:  dbInstanceRepo,
		databaseRepo:    databaseRepo,
		projectRepo:     projectRepo,
		environmentRepo: environmentRepo,
		changePlanRepo:  changePlanRepo,
		connManager:     connManager,
		encryptionKey:   encryptionKey,
		logger:          logger,
	}
}

// encryptConnectionDetails encrypts password and returns encrypted JSON string
func (h *DatabaseInstanceHandler) encryptConnectionDetails(password string) (string, error) {
	details := map[string]string{
		"password": password,
	}
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return "", err
	}
	return utils.Encrypt(string(detailsJSON), h.encryptionKey)
}

// CreateDatabaseInstance handles POST /api/v1/database-instances
func (h *DatabaseInstanceHandler) CreateDatabaseInstance(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse request
	var req models.CreateDatabaseInstanceRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		return middleware.HandleValidationError(c, err)
	}

	// Validate request
	if err := models.ValidateCreateDatabaseInstanceRequest(&req); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		return middleware.HandleValidationError(c, err)
	}

	// Test database connection before saving
	// Use a default database name for connection testing (e.g., "mysql" for MySQL, "postgres" for PostgreSQL)
	testDatabase := ""
	switch req.DatabaseEngine {
	case models.DatabaseEngineMySQL:
		testDatabase = "mysql"
	case models.DatabaseEnginePostgreSQL:
		testDatabase = "postgres"
	}

	connectionParams := utils.ConnectionTestParams{
		DatabaseEngine:           string(req.DatabaseEngine),
		Host:                     req.Host,
		Port:                     req.Port,
		Username:                 req.Username,
		Password:                 req.Password,
		Database:                 testDatabase,
		UseSSL:                   req.RequireSSL,
		ConnectionTimeoutSeconds: 10,
	}

	var connectionErr error
	switch req.DatabaseEngine {
	case models.DatabaseEngineMySQL:
		connectionErr = utils.TestMySQLConnection(ctx, connectionParams)
	case models.DatabaseEnginePostgreSQL:
		connectionErr = utils.TestPostgreSQLConnection(ctx, connectionParams)
	}

	if connectionErr != nil {
		h.logger.Warn(
			"Database connection test failed",
			zap.Error(connectionErr),
			zap.String("host", req.Host),
			zap.Int("port", req.Port),
		)
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Database connection test failed",
			"details": connectionErr.Error(),
		})
	}

	// Encrypt connection details (password)
	encryptedDetails, err := h.encryptConnectionDetails(req.Password)
	if err != nil {
		h.logger.Error("Failed to encrypt connection details", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to encrypt credentials",
		})
	}

	// Create database instance model
	instance := &models.DatabaseInstance{
		Name:                       req.Name,
		Description:                req.Description,
		DatabaseEngine:             req.DatabaseEngine,
		Host:                       req.Host,
		Port:                       req.Port,
		Username:                   req.Username,
		ConnectionDetailsEncrypted: &encryptedDetails,
		UseRDSIAMAuth:              req.UseRDSIAMAuth,
		RequireSSL:                 req.RequireSSL,
		Status:                     models.DatabaseInstanceStatusActive,
		EnvironmentID:              req.EnvironmentID,
		CreatedByID:                claims.UserID,
	}

	// Save to repository
	if err := h.dbInstanceRepo.Create(ctx, instance); err != nil {
		h.logger.Error("Failed to create database instance", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create database instance",
		})
	}

	h.logger.Info(
		"Database instance created successfully",
		zap.String("database_instance_id", instance.ID.String()),
		zap.String("name", instance.Name),
	)

	// Sync databases if requested
	var syncResult *models.DatabaseSyncResult
	if req.SyncAllDatabases {
		syncResult = h.syncDatabasesFromInstance(ctx, instance, claims.UserID)
		h.logger.Info(
			"Database sync completed",
			zap.String("database_instance_id", instance.ID.String()),
			zap.Int("total_found", syncResult.TotalFound),
			zap.Int("synced", syncResult.Synced),
			zap.Int("failed", syncResult.Failed),
			zap.Int("skipped", syncResult.Skipped),
		)
	}

	// Build response with sync results if available
	response := instance.ToResponse()
	response.SyncResults = syncResult

	return c.JSON(http.StatusCreated, response)
}

// ListDatabaseInstances handles GET /api/v1/database-instances
func (h *DatabaseInstanceHandler) ListDatabaseInstances(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse pagination parameters
	limit := 100
	offset := 0

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Check for environment filter
	var instances []*models.DatabaseInstance
	var err error

	environmentIDStr := c.QueryParam("environment_id")
	if environmentIDStr != "" {
		// Filter by environment
		environmentID, parseErr := uuid.Parse(environmentIDStr)
		if parseErr != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid environment ID format",
			})
		}
		instances, err = h.dbInstanceRepo.ListByEnvironment(ctx, environmentID, limit, offset)
	} else {
		// No filter, list all
		instances, err = h.dbInstanceRepo.List(ctx, limit, offset)
	}

	if err != nil {
		h.logger.Error("Failed to list database instances", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to list database instances",
		})
	}

	// Convert to response format
	responses := make([]*models.DatabaseInstanceResponse, len(instances))
	for i, instance := range instances {
		responses[i] = instance.ToResponse()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"database_instances": responses,
		"limit":              limit,
		"offset":             offset,
		"count":              len(responses),
	})
}

// GetDatabaseInstance handles GET /api/v1/database-instances/:id
func (h *DatabaseInstanceHandler) GetDatabaseInstance(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse ID parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid database instance ID",
		})
	}

	// Fetch database instance
	instance, err := h.dbInstanceRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Database instance not found",
			})
		}
		h.logger.Error("Failed to get database instance", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get database instance",
		})
	}

	return c.JSON(http.StatusOK, instance.ToResponse())
}

// UpdateDatabaseInstance handles PUT /api/v1/database-instances/:id
func (h *DatabaseInstanceHandler) UpdateDatabaseInstance(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse ID parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid database instance ID",
		})
	}

	// Bind request
	var req models.UpdateDatabaseInstanceRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		return middleware.HandleValidationError(c, err)
	}

	// Validate request
	if err := models.ValidateUpdateDatabaseInstanceRequest(&req); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		return middleware.HandleValidationError(c, err)
	}

	// Fetch existing database instance
	instance, err := h.dbInstanceRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Database instance not found",
			})
		}
		h.logger.Error("Failed to get database instance", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get database instance",
		})
	}

	// Track if credentials are being updated
	credentialsUpdated := false
	passwordToTest := ""

	// Determine password for connection testing
	// Connection test is needed if: password, connection params (host, port, username), database type, or SSL settings change
	if req.Password != nil && *req.Password != "" {
		passwordToTest = *req.Password
		credentialsUpdated = true
	} else if req.Host != nil || req.Port != nil || req.Username != nil || req.DatabaseEngine != nil || req.RequireSSL != nil {
		// If connection params are being updated but password is not, we need to decrypt existing password for testing
		credentialsUpdated = true
		// Decrypt existing password for connection testing
		if instance.ConnectionDetailsEncrypted != nil {
			decrypted, err := utils.DecryptPassword(instance.ConnectionDetailsEncrypted, h.encryptionKey)
			if err != nil {
				h.logger.Error("Failed to decrypt existing password", zap.Error(err))
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Failed to decrypt existing credentials",
				})
			}
			passwordToTest = decrypted
		}
	}

	// Test connection if credentials or connection parameters are being updated
	if credentialsUpdated {
		// Use updated values or existing values
		testHost := instance.Host
		if req.Host != nil {
			testHost = *req.Host
		}

		testPort := instance.Port
		if req.Port != nil {
			testPort = *req.Port
		}

		testUsername := instance.Username
		if req.Username != nil {
			testUsername = *req.Username
		}

		testDatabaseEngine := instance.DatabaseEngine
		if req.DatabaseEngine != nil {
			testDatabaseEngine = *req.DatabaseEngine
		}

		testRequireSSL := instance.RequireSSL
		if req.RequireSSL != nil {
			testRequireSSL = *req.RequireSSL
		}

		// Use a default database name for connection testing
		testDatabase := ""
		switch testDatabaseEngine {
		case models.DatabaseEngineMySQL:
			testDatabase = "mysql"
		case models.DatabaseEnginePostgreSQL:
			testDatabase = "postgres"
		}

		connectionParams := utils.ConnectionTestParams{
			DatabaseEngine:           string(testDatabaseEngine),
			Host:                     testHost,
			Port:                     testPort,
			Username:                 testUsername,
			Password:                 passwordToTest,
			Database:                 testDatabase,
			UseSSL:                   testRequireSSL,
			ConnectionTimeoutSeconds: 10,
		}

		var connectionErr error
		switch testDatabaseEngine {
		case models.DatabaseEngineMySQL:
			connectionErr = utils.TestMySQLConnection(ctx, connectionParams)
		case models.DatabaseEnginePostgreSQL:
			connectionErr = utils.TestPostgreSQLConnection(ctx, connectionParams)
		}

		if connectionErr != nil {
			h.logger.Warn(
				"Database connection test failed",
				zap.Error(connectionErr),
				zap.String("host", testHost),
				zap.Int("port", testPort),
			)
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error":   "Database connection test failed",
				"details": connectionErr.Error(),
			})
		}

		// Encrypt password if provided
		if req.Password != nil && *req.Password != "" {
			encryptedDetails, err := h.encryptConnectionDetails(*req.Password)
			if err != nil {
				h.logger.Error("Failed to encrypt connection details", zap.Error(err))
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Failed to encrypt credentials",
				})
			}
			instance.ConnectionDetailsEncrypted = &encryptedDetails
		}
	}

	// Apply updates
	if req.Name != nil {
		instance.Name = *req.Name
	}
	if req.Description != nil {
		instance.Description = req.Description
	}
	if req.DatabaseEngine != nil {
		instance.DatabaseEngine = *req.DatabaseEngine
	}
	if req.Host != nil {
		instance.Host = *req.Host
	}
	if req.Port != nil {
		instance.Port = *req.Port
	}
	if req.Username != nil {
		instance.Username = *req.Username
	}
	if req.UseRDSIAMAuth != nil {
		instance.UseRDSIAMAuth = *req.UseRDSIAMAuth
	}
	if req.RequireSSL != nil {
		instance.RequireSSL = *req.RequireSSL
	}
	if req.Status != nil {
		instance.Status = *req.Status
	}
	if req.EnvironmentID != nil {
		instance.EnvironmentID = req.EnvironmentID
	}

	// Update database instance
	if err := h.dbInstanceRepo.Update(ctx, instance); err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Database instance not found",
			})
		}
		h.logger.Error("Failed to update database instance", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update database instance",
		})
	}

	h.logger.Info("Database instance updated successfully", zap.String("id", instance.ID.String()))

	return c.JSON(http.StatusOK, instance.ToResponse())
}

// DeleteDatabaseInstance handles DELETE /api/v1/database-instances/:id
func (h *DatabaseInstanceHandler) DeleteDatabaseInstance(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse ID parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid database instance ID",
		})
	}

	// Check if database instance exists
	_, err = h.dbInstanceRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Database instance not found",
			})
		}
		h.logger.Error("Failed to get database instance for deletion", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve database instance",
		})
	}

	// Check for associated databases before allowing deletion
	databases, err := h.databaseRepo.ListByInstanceID(ctx, id, 1, 0)
	if err != nil {
		h.logger.Error("Failed to check databases for instance", zap.Error(err), zap.String("instance_id", id.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to check database dependencies",
		})
	}

	if len(databases) > 0 {
		return c.JSON(http.StatusConflict, map[string]interface{}{
			"error": "Cannot delete database instance: instance is referenced by databases. Please delete or reassign databases first.",
		})
	}

	// Delete database instance
	if err := h.dbInstanceRepo.Delete(ctx, id); err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Database instance not found",
			})
		}
		h.logger.Error("Failed to delete database instance", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to delete database instance",
		})
	}

	h.logger.Info("Database instance deleted successfully", zap.String("id", id.String()))

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Database instance deleted successfully",
	})
}

// CreateDatabaseForInstance handles POST /api/v1/database-instances/:id/databases
func (h *DatabaseInstanceHandler) CreateDatabaseForInstance(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Get instance ID from URL
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid instance ID",
		})
	}

	// Parse request
	var req models.CreateDatabaseRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		return middleware.HandleValidationError(c, err)
	}

	// Set the database instance ID from URL
	req.DatabaseInstanceID = instanceID

	// Validate request
	if err := models.ValidateCreateDatabaseRequest(&req); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		return middleware.HandleValidationError(c, err)
	}

	// Validate that project exists
	_, err = h.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Project not found",
			})
		}
		h.logger.Error("Failed to validate project", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to validate project",
		})
	}

	// Validate that database instance exists
	instance, err := h.dbInstanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Database instance not found",
			})
		}
		h.logger.Error("Failed to validate database instance", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to validate database instance",
		})
	}

	// Validate that environment exists
	_, err = h.environmentRepo.GetByID(ctx, req.EnvironmentID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Environment not found",
			})
		}
		h.logger.Error("Failed to validate environment", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to validate environment",
		})
	}

	// Check if database already exists in Deeb registry
	existingDB, err := h.databaseRepo.GetByInstanceAndName(ctx, instanceID, req.DatabaseName)
	if err != nil && err != repository.ErrNotFound {
		h.logger.Error("Failed to check existing database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to check existing database",
		})
	}

	// If database exists in Deeb, return it
	if existingDB != nil {
		h.logger.Info("Database already exists in Deeb", zap.String("id", existingDB.ID.String()))
		return c.JSON(http.StatusOK, existingDB)
	}

	// Check if database physically exists on the instance
	dbExists, err := h.connManager.DatabaseExistsOnInstance(ctx, instance, req.DatabaseName)
	if err != nil {
		h.logger.Error("Failed to check database existence on instance", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to check database existence: %v", err),
		})
	}

	if dbExists {
		// Database exists physically, create Active record
		database := &models.Database{
			DatabaseName:       req.DatabaseName,
			DatabaseInstanceID: instanceID,
			ProjectID:          &req.ProjectID,
			EnvironmentID:      &req.EnvironmentID,
			Description:        req.Description,
			Status:             models.DatabaseStatusActive,
			CreatedByID:        claims.UserID,
		}

		if err := h.databaseRepo.Create(ctx, database); err != nil {
			h.logger.Error("Failed to create database", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to create database",
			})
		}

		h.logger.Info("Database imported", zap.String("id", database.ID.String()), zap.String("name", database.DatabaseName))

		return c.JSON(http.StatusOK, database)
	}

	// Database doesn't exist, create Pending record and change plan
	database := &models.Database{
		DatabaseName:       req.DatabaseName,
		DatabaseInstanceID: instanceID,
		ProjectID:          &req.ProjectID,
		EnvironmentID:      &req.EnvironmentID,
		Description:        req.Description,
		Status:             models.DatabaseStatusPending,
		CreatedByID:        claims.UserID,
	}

	if err := h.databaseRepo.Create(ctx, database); err != nil {
		h.logger.Error("Failed to create pending database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create database",
		})
	}

	// Generate CREATE DATABASE SQL
	var sqlContent string
	switch instance.DatabaseEngine {
	case models.DatabaseEngineMySQL:
		sqlContent = fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;", req.DatabaseName)
	case models.DatabaseEnginePostgreSQL:
		// Set the database owner to the instance username
		sqlContent = fmt.Sprintf("CREATE DATABASE \"%s\" OWNER \"%s\";", req.DatabaseName, instance.Username)
	default:
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Unsupported database engine",
		})
	}

	// Create change plan
	changePlan := &models.ChangePlan{
		ProjectID:        req.ProjectID,
		TargetDatabaseID: database.ID,
		Title:            fmt.Sprintf("Create database: %s", req.DatabaseName),
		Description:      fmt.Sprintf("Create database %s on instance %s", req.DatabaseName, instance.Name),
		ChangeType:       models.ChangeTypeDDL,
		SQLContent:       sqlContent,
		Status:           models.ChangePlanStatusDraft,
		RiskLevel:        models.RiskLevelMedium,
		AuthorID:         claims.UserID,
	}

	if req.Description != nil {
		changePlan.Description = *req.Description
	}

	if err := h.changePlanRepo.Create(ctx, changePlan); err != nil {
		h.logger.Error("Failed to create change plan", zap.Error(err))
		// Cleanup: delete the orphaned database record
		if deleteErr := h.databaseRepo.Delete(ctx, database.ID); deleteErr != nil {
			h.logger.Error("Failed to cleanup orphaned database record",
				zap.String("database_id", database.ID.String()),
				zap.Error(deleteErr),
			)
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create change plan",
		})
	}

	h.logger.Info(
		"Change plan created for database",
		zap.String("change_plan_id", changePlan.ID.String()),
		zap.String("database_id", database.ID.String()),
	)

	return c.JSON(http.StatusOK, changePlan)
}

// TestDatabaseInstanceConnection handles both:
// - POST /api/v1/database-instances/:id/test-connection (existing instance)
// - POST /api/v1/database-instances/test-connection (new instance with credentials)
func (h *DatabaseInstanceHandler) TestDatabaseInstanceConnection(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	var instance *models.DatabaseInstance
	var testDatabaseName string

	// Check if this is for an existing instance (has :id param) or new credentials
	instanceIDStr := c.Param("id")
	if instanceIDStr != "" {
		// Existing instance - fetch from database
		instanceID, err := uuid.Parse(instanceIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid instance ID format",
			})
		}

		instance, err = h.dbInstanceRepo.GetByID(ctx, instanceID)
		if err != nil {
			if err == repository.ErrNotFound {
				return c.JSON(http.StatusNotFound, map[string]interface{}{
					"error": "Database instance not found",
				})
			}
			h.logger.Error("Failed to get database instance", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve database instance",
			})
		}

		// Parse optional database name from request body
		var req struct {
			DatabaseName string `json:"database_name,omitempty"`
		}
		if err := c.Bind(&req); err != nil {
			h.logger.Debug("No request body provided, will use default database", zap.Error(err))
		}
		testDatabaseName = req.DatabaseName
	} else {
		// New instance - parse credentials from request body
		var req struct {
			Host           string `json:"host" validate:"required"`
			Port           int    `json:"port" validate:"required,min=1,max=65535"`
			Username       string `json:"username" validate:"required"`
			Password       string `json:"password"`
			DatabaseEngine string `json:"database_engine" validate:"required,oneof=mysql postgresql"`
			DatabaseName   string `json:"database_name"`
			UseRDSIAMAuth  bool   `json:"use_rds_iam_auth"`
			RequireSSL     bool   `json:"require_ssl"`
		}

		if err := c.Bind(&req); err != nil {
			h.logger.Error("Failed to bind request", zap.Error(err))
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid request format",
			})
		}

		// Validate required fields
		if req.Host == "" || req.Port == 0 || req.Username == "" || req.DatabaseEngine == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Missing required fields: host, port, username, database_engine",
			})
		}

		// Encrypt the password for connection testing
		// The password needs to be in JSON format: {"password":"..."}
		var encryptedPassword *string
		if req.Password != "" {
			passwordJSON, err := json.Marshal(map[string]string{"password": req.Password})
			if err != nil {
				h.logger.Error("Failed to marshal password for connection test", zap.Error(err))
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Failed to process credentials",
				})
			}
			encrypted, err := utils.Encrypt(string(passwordJSON), h.encryptionKey)
			if err != nil {
				h.logger.Error("Failed to encrypt password for connection test", zap.Error(err))
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Failed to process credentials",
				})
			}
			encryptedPassword = &encrypted
		}

		// Create a temporary database instance for connection testing
		instance = &models.DatabaseInstance{
			Host:                       req.Host,
			Port:                       req.Port,
			Username:                   req.Username,
			ConnectionDetailsEncrypted: encryptedPassword,
			DatabaseEngine:             models.DatabaseEngine(req.DatabaseEngine),
			UseRDSIAMAuth:              req.UseRDSIAMAuth,
			RequireSSL:                 req.RequireSSL,
		}
		testDatabaseName = req.DatabaseName
	}

	// Determine which database to test against
	if testDatabaseName == "" {
		// Use default system database if no specific database provided
		switch instance.DatabaseEngine {
		case models.DatabaseEngineMySQL:
			testDatabaseName = "mysql"
		case models.DatabaseEnginePostgreSQL:
			// For PostgreSQL, try to connect to a database with the same name as the username
			// This is the default behavior when no database is specified
			// If the user doesn't have their own database, they likely have access to it
			testDatabaseName = "postgres"
		}
	}

	// Create a temporary database model for connection testing
	testDB := &models.Database{
		DatabaseName: testDatabaseName,
	}

	// Attempt connection using connection manager
	conn, connErr := h.connManager.Connect(ctx, database.ConnectionParams{
		DatabaseInstance: instance,
		Database:         testDB,
		MaxOpenConns:     1,
		MaxIdleConns:     0,
		ConnMaxLifetime:  0, // Close immediately after test
	})

	if connErr != nil {
		h.logger.Warn("Database connection test failed",
			zap.Error(connErr),
			zap.String("host", instance.Host),
			zap.Int("port", instance.Port),
			zap.String("database", testDatabaseName))

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":  false,
			"message":  "Connection test failed",
			"error":    connErr.Error(),
			"host":     instance.Host,
			"port":     instance.Port,
			"database": testDatabaseName,
		})
	}

	// Close connection immediately
	if closeErr := conn.Close(); closeErr != nil {
		h.logger.Warn("Failed to close test connection", zap.Error(closeErr))
	}

	h.logger.Info("Database connection test successful",
		zap.String("host", instance.Host),
		zap.Int("port", instance.Port),
		zap.String("database", testDatabaseName))

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":  true,
		"message":  "Connection test successful",
		"host":     instance.Host,
		"port":     instance.Port,
		"database": testDatabaseName,
		"engine":   string(instance.DatabaseEngine),
		"username": instance.Username,
	})
}

// SyncDatabases handles POST /api/v1/database-instances/:id/sync
func (h *DatabaseInstanceHandler) SyncDatabases(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Get instance ID from path
	instanceIDStr := c.Param("id")
	instanceID, err := uuid.Parse(instanceIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid database instance ID",
		})
	}

	// Fetch database instance
	instance, err := h.dbInstanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Database instance not found",
			})
		}
		h.logger.Error("Failed to get database instance", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve database instance",
		})
	}

	// Perform sync
	syncResult := h.syncDatabasesFromInstance(ctx, instance, claims.UserID)

	h.logger.Info(
		"Database sync completed",
		zap.String("database_instance_id", instance.ID.String()),
		zap.Int("total_found", syncResult.TotalFound),
		zap.Int("synced", syncResult.Synced),
		zap.Int("failed", syncResult.Failed),
		zap.Int("skipped", syncResult.Skipped),
	)

	return c.JSON(http.StatusOK, syncResult)
}

// syncDatabasesFromInstance discovers and imports all databases from a database instance
// Returns sync results with statistics about the operation
func (h *DatabaseInstanceHandler) syncDatabasesFromInstance(
	ctx context.Context,
	instance *models.DatabaseInstance,
	createdByID uuid.UUID,
) *models.DatabaseSyncResult {
	result := &models.DatabaseSyncResult{
		FailedDatabases:  []string{},
		SkippedDatabases: []string{},
	}

	// Discover databases on the instance
	databaseNames, err := h.connManager.ListDatabasesOnInstance(ctx, instance)
	if err != nil {
		h.logger.Error("Failed to list databases on instance",
			zap.String("instance_id", instance.ID.String()),
			zap.Error(err),
		)
		// Return early with error - total found is 0, all failed
		result.TotalFound = 0
		result.Failed = 1
		result.FailedDatabases = append(result.FailedDatabases, fmt.Sprintf("discovery-error: %s", err.Error()))
		return result
	}

	result.TotalFound = len(databaseNames)

	// Process each discovered database
	for _, dbName := range databaseNames {
		// Check if database already exists in Deeb
		existingDB, err := h.databaseRepo.GetByInstanceAndName(ctx, instance.ID, dbName)
		if err != nil && err != repository.ErrNotFound {
			// Error checking for existing database
			result.Failed++
			result.FailedDatabases = append(result.FailedDatabases, fmt.Sprintf("%s (check failed)", dbName))
			h.logger.Error("Failed to check if database exists",
				zap.String("database_name", dbName),
				zap.String("instance_id", instance.ID.String()),
				zap.Error(err),
			)
			continue
		}
		if existingDB != nil {
			// Database already exists, skip it
			result.Skipped++
			result.SkippedDatabases = append(result.SkippedDatabases, dbName)
			h.logger.Debug("Database already exists, skipping",
				zap.String("database_name", dbName),
				zap.String("instance_id", instance.ID.String()),
				zap.String("existing_id", existingDB.ID.String()),
			)
			continue
		}

		// Create new database record with nil project_id and environment_id
		database := &models.Database{
			DatabaseName:       dbName,
			DatabaseInstanceID: instance.ID,
			ProjectID:          nil, // Unassigned
			EnvironmentID:      nil, // Unassigned
			Status:             models.DatabaseStatusActive,
			CreatedByID:        createdByID,
		}

		if err := h.databaseRepo.Create(ctx, database); err != nil {
			result.Failed++
			result.FailedDatabases = append(result.FailedDatabases, dbName)
			h.logger.Error("Failed to create database record during sync",
				zap.String("database_name", dbName),
				zap.String("instance_id", instance.ID.String()),
				zap.Error(err),
			)
			continue
		}

		result.Synced++
		h.logger.Debug("Database synced successfully",
			zap.String("database_id", database.ID.String()),
			zap.String("database_name", dbName),
			zap.String("instance_id", instance.ID.String()),
		)
	}

	return result
}
