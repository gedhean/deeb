package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/middleware"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/services"
	"go.uber.org/zap"
)

// DatabaseHandler handles HTTP requests for databases
type DatabaseHandler struct {
	databaseRepo         repository.DatabaseRepository
	projectRepo          repository.ProjectRepository
	environmentRepo      repository.EnvironmentRepository
	databaseInstanceRepo repository.DatabaseInstanceRepository
	changePlanRepo       repository.ChangePlanRepository
	auditService         *services.AuditLogService
	logger               *zap.Logger
}

// NewDatabaseHandler creates a new database handler
func NewDatabaseHandler(
	databaseRepo repository.DatabaseRepository,
	projectRepo repository.ProjectRepository,
	environmentRepo repository.EnvironmentRepository,
	databaseInstanceRepo repository.DatabaseInstanceRepository,
	changePlanRepo repository.ChangePlanRepository,
	auditService *services.AuditLogService,
	logger *zap.Logger,
) *DatabaseHandler {
	return &DatabaseHandler{
		databaseRepo:         databaseRepo,
		projectRepo:          projectRepo,
		environmentRepo:      environmentRepo,
		databaseInstanceRepo: databaseInstanceRepo,
		changePlanRepo:       changePlanRepo,
		auditService:         auditService,
		logger:               logger,
	}
}

// CreateDatabase handles POST /api/v1/databases
func (h *DatabaseHandler) CreateDatabase(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	var req models.CreateDatabaseRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		return middleware.HandleValidationError(c, err)
	}

	// Validate request
	if err := models.ValidateCreateDatabaseRequest(&req); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		return middleware.HandleValidationError(c, err)
	}

	// Validate that project exists
	_, err := h.projectRepo.GetByID(ctx, req.ProjectID)
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
	_, err = h.databaseInstanceRepo.GetByID(ctx, req.DatabaseInstanceID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
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

	// Create database
	database := &models.Database{
		DatabaseName:       req.DatabaseName,
		DatabaseInstanceID: req.DatabaseInstanceID,
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

	h.logger.Info("Database created", zap.String("id", database.ID.String()), zap.String("name", database.DatabaseName))

	// Log audit entry
	if h.auditService != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"database_name":        database.DatabaseName,
			"description":          database.Description,
			"project_id":           database.ProjectID.String(),
			"environment_id":       database.EnvironmentID.String(),
			"database_instance_id": database.DatabaseInstanceID.String(),
		})
		if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionCreate, services.ResourceTypeDatabase, database.ID, c.Request(), detailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for database creation", zap.Error(err))
		}
	}

	return c.JSON(http.StatusCreated, database.ToResponse())
}

// ToResponseWithDetails converts a Database to DatabaseResponse with related entities
func (h *DatabaseHandler) ToResponseWithDetails(ctx context.Context, database *models.Database) *models.DatabaseResponse {
	resp := database.ToResponse()

	// Fetch project
	if project, err := h.projectRepo.GetByID(ctx, database.ProjectID); err == nil {
		resp.Project = project.ToResponse()
	}

	// Fetch database instance
	if instance, err := h.databaseInstanceRepo.GetByID(ctx, database.DatabaseInstanceID); err == nil {
		resp.DatabaseInstance = instance.ToResponse()
	}

	// Fetch environment
	if env, err := h.environmentRepo.GetByID(ctx, database.EnvironmentID); err == nil {
		resp.Environment = env.ToResponse()
	}

	return resp
}

// ListDatabases handles GET /api/v1/databases
func (h *DatabaseHandler) ListDatabases(c echo.Context) error {
	ctx := c.Request().Context()

	// Authorization check - authentication required
	_, ok := auth.GetUserFromEchoContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

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

	// Parse filters
	projectIDStr := c.QueryParam("project_id")
	environmentIDStr := c.QueryParam("environment_id")

	var databases []*models.Database
	var err error

	// Filter by both project_id and environment_id
	if projectIDStr != "" && environmentIDStr != "" {
		projectID, parseErr := uuid.Parse(projectIDStr)
		if parseErr != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid project ID",
			})
		}
		environmentID, parseErr := uuid.Parse(environmentIDStr)
		if parseErr != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid environment ID",
			})
		}
		// Filter by project first, then filter by environment in memory
		// (Alternatively, we could add a ListByProjectAndEnvironment method)
		projectDatabases, err := h.databaseRepo.ListByProject(ctx, projectID, limit*2, offset)
		if err != nil {
			h.logger.Error("Failed to list databases by project", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to list databases",
			})
		}
		// Filter by environment
		databases = []*models.Database{}
		for _, db := range projectDatabases {
			if db.EnvironmentID == environmentID {
				databases = append(databases, db)
			}
		}
		// Apply limit
		if len(databases) > limit {
			databases = databases[:limit]
		}
	} else if projectIDStr != "" {
		projectID, parseErr := uuid.Parse(projectIDStr)
		if parseErr != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid project ID",
			})
		}
		databases, err = h.databaseRepo.ListByProject(ctx, projectID, limit, offset)
	} else if environmentIDStr != "" {
		environmentID, parseErr := uuid.Parse(environmentIDStr)
		if parseErr != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid environment ID",
			})
		}
		databases, err = h.databaseRepo.ListByEnvironmentID(ctx, environmentID, limit, offset)
	} else {
		databases, err = h.databaseRepo.List(ctx, limit, offset)
	}

	if err != nil {
		h.logger.Error("Failed to list databases", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to list databases",
		})
	}

	// Convert to response format with related entities
	responses := make([]*models.DatabaseResponse, len(databases))
	for i, database := range databases {
		responses[i] = h.ToResponseWithDetails(ctx, database)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"databases": responses,
		"limit":     limit,
		"offset":    offset,
		"count":     len(responses),
	})
}

// GetDatabase handles GET /api/v1/databases/:id
func (h *DatabaseHandler) GetDatabase(c echo.Context) error {
	ctx := c.Request().Context()

	// Authorization check - authentication required
	_, ok := auth.GetUserFromEchoContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse ID parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid database ID",
		})
	}

	// Fetch database
	database, err := h.databaseRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Database not found",
			})
		}
		h.logger.Error("Failed to get database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get database",
		})
	}

	// Return response with related entities
	return c.JSON(http.StatusOK, h.ToResponseWithDetails(ctx, database))
}

// ListProjectDatabases handles GET /api/v1/projects/:project_id/databases
func (h *DatabaseHandler) ListProjectDatabases(c echo.Context) error {
	ctx := c.Request().Context()

	// Authorization check - authentication required
	_, ok := auth.GetUserFromEchoContext(c)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse project ID parameter
	projectIDStr := c.Param("project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid project ID",
		})
	}

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

	// Fetch databases for project
	databases, err := h.databaseRepo.ListByProject(ctx, projectID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list project databases", zap.Error(err), zap.String("project_id", projectID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to list project databases",
		})
	}

	// Convert to response format with related entities
	responses := make([]*models.DatabaseResponse, len(databases))
	for i, database := range databases {
		responses[i] = h.ToResponseWithDetails(ctx, database)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"databases":  responses,
		"project_id": projectID,
		"limit":      limit,
		"offset":     offset,
		"count":      len(responses),
	})
}

// UpdateDatabase handles PUT /api/v1/databases/:id
func (h *DatabaseHandler) UpdateDatabase(c echo.Context) error {
	ctx := c.Request().Context()

	// Authorization check - admin only
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Check if user is admin (this should be enforced by middleware, but double-check)
	if claims.OrganizationRole != string(models.OrganizationRoleAdmin) {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "Admin access required",
		})
	}

	// Parse ID parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid database ID",
		})
	}

	// Bind request
	var req models.UpdateDatabaseRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		return middleware.HandleValidationError(c, err)
	}

	// Validate request
	if err := models.ValidateUpdateDatabaseRequest(&req); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		return middleware.HandleValidationError(c, err)
	}

	// Fetch existing database
	database, err := h.databaseRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Database not found",
			})
		}
		h.logger.Error("Failed to get database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get database",
		})
	}

	// Validate that project/environment constraints are maintained
	// (The database must belong to exactly one project and one environment)
	// This is enforced by the schema, but we verify the related entities exist
	if _, err := h.projectRepo.GetByID(ctx, database.ProjectID); err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Database project not found",
			})
		}
		h.logger.Error("Failed to validate project", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to validate project",
		})
	}

	if _, err := h.environmentRepo.GetByID(ctx, database.EnvironmentID); err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Database environment not found",
			})
		}
		h.logger.Error("Failed to validate environment", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to validate environment",
		})
	}

	// Apply updates
	if req.DatabaseName != nil {
		database.DatabaseName = *req.DatabaseName
	}
	if req.Description != nil {
		database.Description = req.Description
	}
	if req.Status != nil {
		database.Status = *req.Status
	}

	// Update database
	if err := h.databaseRepo.Update(ctx, database); err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Database not found",
			})
		}
		h.logger.Error("Failed to update database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update database",
		})
	}

	h.logger.Info("Database updated", zap.String("id", database.ID.String()))

	// Log audit entry
	if h.auditService != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"database_name": database.DatabaseName,
			"description":   database.Description,
			"status":        string(database.Status),
		})
		if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionUpdate, services.ResourceTypeDatabase, id, c.Request(), detailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for database update", zap.Error(err))
		}
	}

	return c.JSON(http.StatusOK, h.ToResponseWithDetails(ctx, database))
}

// DeleteDatabase handles DELETE /api/v1/databases/:id
func (h *DatabaseHandler) DeleteDatabase(c echo.Context) error {
	ctx := c.Request().Context()

	// Authorization check - admin only
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Check if user is admin (this should be enforced by middleware, but double-check)
	if claims.OrganizationRole != string(models.OrganizationRoleAdmin) {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "Admin access required",
		})
	}

	// Parse ID parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid database ID",
		})
	}

	// Check if database exists
	database, err := h.databaseRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Database not found",
			})
		}
		h.logger.Error("Failed to get database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get database",
		})
	}

	// Check if Change Plans exist for this database
	// Use a small limit to just check existence
	changePlans, err := h.changePlanRepo.GetByDatabaseID(ctx, id, 1, 0)
	if err != nil {
		h.logger.Error("Failed to check for change plans", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to check for change plans",
		})
	}

	if len(changePlans) > 0 {
		return c.JSON(http.StatusConflict, map[string]interface{}{
			"error": "Cannot delete database: Change plans exist for this database",
		})
	}

	// Delete database
	if err := h.databaseRepo.Delete(ctx, id); err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Database not found",
			})
		}
		h.logger.Error("Failed to delete database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to delete database",
		})
	}

	h.logger.Info("Database deleted", zap.String("id", id.String()), zap.String("name", database.DatabaseName))

	// Log audit entry
	if h.auditService != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"database_id":   id.String(),
			"database_name": database.DatabaseName,
		})
		if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionDelete, services.ResourceTypeDatabase, id, c.Request(), detailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for database deletion", zap.Error(err))
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Database deleted successfully",
	})
}
