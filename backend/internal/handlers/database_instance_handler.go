package handlers

import (
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
	"github.com/nsxbet/deeb/internal/services"
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
	auditService    *services.AuditLogService
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
	auditService *services.AuditLogService,
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
		auditService:    auditService,
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

	// Log audit entry
	if h.auditService != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"name":            instance.Name,
			"description":     instance.Description,
			"database_engine": string(instance.DatabaseEngine),
			"host":            instance.Host,
			"port":            instance.Port,
			"username":        instance.Username,
		})
		if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionCreate, services.ResourceTypeDatabaseInstance, instance.ID, c.Request(), detailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for database instance creation", zap.Error(err))
		}
	}

	return c.JSON(http.StatusCreated, instance.ToResponse())
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

	// Log audit entry
	if h.auditService != nil {
		claims, ok := auth.GetUserFromEchoContext(c)
		if ok && claims != nil {
			detailsJSON, _ := json.Marshal(map[string]interface{}{
				"name":            instance.Name,
				"description":     instance.Description,
				"database_engine": string(instance.DatabaseEngine),
				"host":            instance.Host,
				"port":            instance.Port,
				"username":        instance.Username,
			})
			if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionUpdate, services.ResourceTypeDatabaseInstance, id, c.Request(), detailsJSON); err != nil {
				h.logger.Error("Failed to log audit entry for database instance update", zap.Error(err))
			}
		}
	}

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

	// Log audit entry
	if h.auditService != nil {
		claims, ok := auth.GetUserFromEchoContext(c)
		if ok && claims != nil {
			detailsJSON, _ := json.Marshal(map[string]interface{}{
				"database_instance_id": id.String(),
			})
			if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionDelete, services.ResourceTypeDatabaseInstance, id, c.Request(), detailsJSON); err != nil {
				h.logger.Error("Failed to log audit entry for database instance deletion", zap.Error(err))
			}
		}
	}

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
			ProjectID:          req.ProjectID,
			EnvironmentID:      req.EnvironmentID,
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

		// Log audit entry
		if h.auditService != nil {
			detailsJSON, _ := json.Marshal(map[string]interface{}{
				"database_name":        database.DatabaseName,
				"description":          database.Description,
				"project_id":           database.ProjectID.String(),
				"environment_id":       database.EnvironmentID.String(),
				"database_instance_id": database.DatabaseInstanceID.String(),
				"imported":             true,
			})
			if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionCreate, services.ResourceTypeDatabase, database.ID, c.Request(), detailsJSON); err != nil {
				h.logger.Error("Failed to log audit entry for database import", zap.Error(err))
			}
		}

		return c.JSON(http.StatusOK, database)
	}

	// Database doesn't exist, create Pending record and change plan
	database := &models.Database{
		DatabaseName:       req.DatabaseName,
		DatabaseInstanceID: instanceID,
		ProjectID:          req.ProjectID,
		EnvironmentID:      req.EnvironmentID,
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

	// Log audit entry
	if h.auditService != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"database_name":        database.DatabaseName,
			"description":          database.Description,
			"project_id":           database.ProjectID.String(),
			"environment_id":       database.EnvironmentID.String(),
			"database_instance_id": database.DatabaseInstanceID.String(),
			"change_plan_id":       changePlan.ID.String(),
			"pending":              true,
		})
		if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionCreate, services.ResourceTypeDatabase, database.ID, c.Request(), detailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for pending database creation", zap.Error(err))
		}
	}

	return c.JSON(http.StatusOK, changePlan)
}
