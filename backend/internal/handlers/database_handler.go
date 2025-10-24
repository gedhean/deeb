package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// DatabaseHandler handles HTTP requests for databases
type DatabaseHandler struct {
	databaseRepo repository.DatabaseRepository
	logger       *zap.Logger
}

// NewDatabaseHandler creates a new database handler
func NewDatabaseHandler(databaseRepo repository.DatabaseRepository, logger *zap.Logger) *DatabaseHandler {
	return &DatabaseHandler{
		databaseRepo: databaseRepo,
		logger:       logger,
	}
}

// CreateDatabase handles POST /api/v1/databases
func (h *DatabaseHandler) CreateDatabase(c echo.Context) error {
	ctx := context.Background()

	var req models.CreateDatabaseRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := models.ValidateCreateDatabaseRequest(&req); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Create database
	database := &models.Database{
		Name:               req.Name,
		DatabaseInstanceID: req.DatabaseInstanceID,
		ProjectID:          req.ProjectID,
		EnvironmentID:      req.EnvironmentID,
	}

	if err := h.databaseRepo.Create(ctx, database); err != nil {
		h.logger.Error("Failed to create database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create database",
		})
	}

	h.logger.Info("Database created", zap.String("id", database.ID.String()), zap.String("name", database.Name))
	return c.JSON(http.StatusCreated, database.ToResponse())
}

// ListDatabases handles GET /api/v1/databases
func (h *DatabaseHandler) ListDatabases(c echo.Context) error {
	ctx := context.Background()

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

	// Check for project filter
	projectIDStr := c.QueryParam("project_id")
	var databases []*models.Database
	var err error

	if projectIDStr != "" {
		projectID, parseErr := uuid.Parse(projectIDStr)
		if parseErr != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid project ID",
			})
		}
		databases, err = h.databaseRepo.ListByProject(ctx, projectID, limit, offset)
	} else {
		databases, err = h.databaseRepo.List(ctx, limit, offset)
	}

	if err != nil {
		h.logger.Error("Failed to list databases", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to list databases",
		})
	}

	// Convert to response format
	responses := make([]*models.DatabaseResponse, len(databases))
	for i, database := range databases {
		responses[i] = database.ToResponse()
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
	ctx := context.Background()

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

	return c.JSON(http.StatusOK, database.ToResponse())
}

// ListProjectDatabases handles GET /api/v1/projects/:project_id/databases
func (h *DatabaseHandler) ListProjectDatabases(c echo.Context) error {
	ctx := context.Background()

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

	// Convert to response format
	responses := make([]*models.DatabaseResponse, len(databases))
	for i, database := range databases {
		responses[i] = database.ToResponse()
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
	ctx := context.Background()

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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := models.ValidateUpdateDatabaseRequest(&req); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
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

	// Apply updates
	if req.Name != nil {
		database.Name = *req.Name
	}
	if req.DatabaseInstanceID != nil {
		database.DatabaseInstanceID = *req.DatabaseInstanceID
	}
	if req.ProjectID != nil {
		database.ProjectID = *req.ProjectID
	}
	if req.EnvironmentID != nil {
		database.EnvironmentID = *req.EnvironmentID
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
	return c.JSON(http.StatusOK, database.ToResponse())
}

// DeleteDatabase handles DELETE /api/v1/databases/:id
func (h *DatabaseHandler) DeleteDatabase(c echo.Context) error {
	ctx := context.Background()

	// Parse ID parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid database ID",
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

	h.logger.Info("Database deleted", zap.String("id", id.String()))
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Database deleted successfully",
	})
}
