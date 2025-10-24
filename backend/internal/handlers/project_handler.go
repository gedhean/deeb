package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// ProjectHandler handles project management endpoints
type ProjectHandler struct {
	projectRepo repository.ProjectRepository
	logger      *zap.Logger
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(projectRepo repository.ProjectRepository, logger *zap.Logger) *ProjectHandler {
	return &ProjectHandler{
		projectRepo: projectRepo,
		logger:      logger,
	}
}

// CreateProject handles POST /api/v1/projects - create project (Admin/DBA only)
func (h *ProjectHandler) CreateProject(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse request
	var req models.CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := models.ValidateCreateRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}

	// Create project model
	project := &models.Project{
		Name:        req.Name,
		Description: req.Description,
	}

	// Save to repository
	if err := h.projectRepo.Create(ctx, project); err != nil {
		h.logger.Error("Failed to create project", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create project",
		})
	}

	h.logger.Info("Project created successfully", zap.String("project_id", project.ID.String()))
	return c.JSON(http.StatusCreated, project.ToResponse())
}

// ListProjects handles GET /api/v1/projects - list projects (all authenticated users)
func (h *ProjectHandler) ListProjects(c echo.Context) error {
	ctx := c.Request().Context()

	// Get pagination parameters with defaults
	limit := 100
	offset := 0

	// Parse query parameters if provided
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if parsedLimit, err := parseInt(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		if parsedOffset, err := parseInt(offsetParam); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Fetch projects from repository
	projects, err := h.projectRepo.List(ctx, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list projects", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve projects",
		})
	}

	// Convert to response format
	responses := make([]*models.ProjectResponse, len(projects))
	for i, project := range projects {
		responses[i] = project.ToResponse()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"projects": responses,
		"limit":    limit,
		"offset":   offset,
		"count":    len(responses),
	})
}

// GetProject handles GET /api/v1/projects/:id - get project by ID
func (h *ProjectHandler) GetProject(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse project ID from path parameter
	idParam := c.Param("id")
	projectID, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid project ID format",
		})
	}

	// Fetch project from repository
	project, err := h.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Project not found",
			})
		}
		h.logger.Error("Failed to get project", zap.Error(err), zap.String("project_id", projectID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve project",
		})
	}

	return c.JSON(http.StatusOK, project.ToResponse())
}

// UpdateProject handles PUT /api/v1/projects/:id - update project
func (h *ProjectHandler) UpdateProject(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse project ID from path parameter
	idParam := c.Param("id")
	projectID, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid project ID format",
		})
	}

	// Parse update request
	var req models.UpdateProjectRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := models.ValidateUpdateRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}

	// Fetch existing project
	project, err := h.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Project not found",
			})
		}
		h.logger.Error("Failed to get project for update", zap.Error(err), zap.String("project_id", projectID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve project",
		})
	}

	// Apply updates
	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Description != nil {
		project.Description = *req.Description
	}

	// Update project in repository
	if err := h.projectRepo.Update(ctx, project); err != nil {
		h.logger.Error("Failed to update project", zap.Error(err), zap.String("project_id", projectID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update project",
		})
	}

	h.logger.Info("Project updated successfully", zap.String("project_id", projectID.String()))
	return c.JSON(http.StatusOK, project.ToResponse())
}

// DeleteProject handles DELETE /api/v1/projects/:id - delete project
func (h *ProjectHandler) DeleteProject(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse project ID from path parameter
	idParam := c.Param("id")
	projectID, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid project ID format",
		})
	}

	// Check if project exists
	_, err = h.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Project not found",
			})
		}
		h.logger.Error("Failed to get project for deletion", zap.Error(err), zap.String("project_id", projectID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve project",
		})
	}

	// Delete project
	if err := h.projectRepo.Delete(ctx, projectID); err != nil {
		h.logger.Error("Failed to delete project", zap.Error(err), zap.String("project_id", projectID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to delete project",
		})
	}

	h.logger.Info("Project deleted successfully", zap.String("project_id", projectID.String()))
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Project deleted successfully",
	})
}
