package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/services"
	"go.uber.org/zap"
)

// ProjectHandler handles project management endpoints
type ProjectHandler struct {
	projectRepo       repository.ProjectRepository
	databaseRepo      repository.DatabaseRepository
	projectMemberRepo repository.ProjectMemberRepository
	userRepo          repository.UserRepository
	auditService      *services.AuditLogService
	logger            *zap.Logger
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(
	projectRepo repository.ProjectRepository,
	databaseRepo repository.DatabaseRepository,
	projectMemberRepo repository.ProjectMemberRepository,
	userRepo repository.UserRepository,
	auditService *services.AuditLogService,
	logger *zap.Logger,
) *ProjectHandler {
	return &ProjectHandler{
		projectRepo:       projectRepo,
		databaseRepo:      databaseRepo,
		projectMemberRepo: projectMemberRepo,
		userRepo:          userRepo,
		auditService:      auditService,
		logger:            logger,
	}
}

// CreateProject handles POST /api/v1/projects - create project (Admin/DBA only)
func (h *ProjectHandler) CreateProject(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

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

	// Create project model with created_by_id set to authenticated user
	project := &models.Project{
		Name:        req.Name,
		Description: stringPtrOrNil(req.Description),
		CreatedByID: claims.UserID,
	}

	// Save to repository
	if err := h.projectRepo.Create(ctx, project); err != nil {
		h.logger.Error("Failed to create project", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create project",
		})
	}

	// Automatically add the creator as ProjectOwner member
	projectMember := &models.ProjectMember{
		ProjectID: project.ID,
		UserID:    claims.UserID,
		Role:      rbac.ProjectOwner,
		AddedByID: claims.UserID,
	}

	if err := h.projectMemberRepo.AddMember(ctx, projectMember); err != nil {
		h.logger.Error("Failed to add creator as project member", zap.Error(err))
		// Attempt to rollback project creation
		if deleteErr := h.projectRepo.Delete(ctx, project.ID); deleteErr != nil {
			h.logger.Error("Failed to rollback project creation", zap.Error(deleteErr))
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to add creator as project member",
		})
	}

	h.logger.Info("Project created successfully", zap.String("project_id", project.ID.String()))

	// Log audit entry
	if h.auditService != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"project_name": project.Name,
			"description":  project.Description,
		})
		if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionCreate, services.ResourceTypeProject, project.ID, c.Request(), detailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for project creation", zap.Error(err))
		}
	}

	return c.JSON(http.StatusCreated, project.ToResponse())
}

// stringPtrOrNil returns a pointer to the string if non-empty, nil otherwise
func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ListProjects handles GET /api/v1/projects - list projects (all authenticated users)
func (h *ProjectHandler) ListProjects(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Get pagination parameters with defaults
	limit := 100
	offset := 0

	// Parse query parameters if provided
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Determine which projects to list based on organization role
	var projects []*models.Project
	var err error
	var totalCount int64

	orgRole := rbac.OrganizationRole(claims.OrganizationRole)
	if orgRole == rbac.OrganizationAdmin {
		// OrganizationAdmin sees all projects
		projects, err = h.projectRepo.List(ctx, limit, offset)
		if err != nil {
			h.logger.Error("Failed to list projects", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve projects",
			})
		}

		totalCount, err = h.projectRepo.Count(ctx)
		if err != nil {
			h.logger.Error("Failed to count projects", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to count projects",
			})
		}
	} else {
		// Non-admin users (OrganizationDBA, OrganizationDeveloper) only see projects they're members of
		projects, err = h.projectRepo.ListByUserID(ctx, claims.UserID, limit, offset)
		if err != nil {
			h.logger.Error("Failed to list projects for user", zap.Error(err), zap.String("user_id", claims.UserID.String()))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve projects",
			})
		}

		totalCount, err = h.projectRepo.CountByUserID(ctx, claims.UserID)
		if err != nil {
			h.logger.Error("Failed to count projects for user", zap.Error(err), zap.String("user_id", claims.UserID.String()))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to count projects",
			})
		}
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
		"total":    totalCount,
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
		project.Description = req.Description
	}

	// Update project in repository
	if err := h.projectRepo.Update(ctx, project); err != nil {
		h.logger.Error("Failed to update project", zap.Error(err), zap.String("project_id", projectID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update project",
		})
	}

	h.logger.Info("Project updated successfully", zap.String("project_id", projectID.String()))

	// Log audit entry
	if h.auditService != nil {
		claims, ok := auth.GetUserFromEchoContext(c)
		if ok && claims != nil {
			detailsJSON, _ := json.Marshal(map[string]interface{}{
				"project_name": project.Name,
				"description":  project.Description,
			})
			if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionUpdate, services.ResourceTypeProject, projectID, c.Request(), detailsJSON); err != nil {
				h.logger.Error("Failed to log audit entry for project update", zap.Error(err))
			}
		}
	}

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

	// Check if project has associated databases
	databases, err := h.databaseRepo.ListByProject(ctx, projectID, 1, 0)
	if err != nil {
		h.logger.Error("Failed to check databases for project", zap.Error(err), zap.String("project_id", projectID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to check project dependencies",
		})
	}

	if len(databases) > 0 {
		return c.JSON(http.StatusConflict, map[string]interface{}{
			"error": "Cannot delete project: project has associated databases. Please delete or reassign databases first.",
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

	// Log audit entry
	if h.auditService != nil {
		claims, ok := auth.GetUserFromEchoContext(c)
		if ok && claims != nil {
			detailsJSON, _ := json.Marshal(map[string]interface{}{
				"project_id": projectID.String(),
			})
			if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionDelete, services.ResourceTypeProject, projectID, c.Request(), detailsJSON); err != nil {
				h.logger.Error("Failed to log audit entry for project deletion", zap.Error(err))
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Project deleted successfully",
	})
}

// AddProjectMember handles POST /api/v1/projects/:id/members - add member to project
func (h *ProjectHandler) AddProjectMember(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse project ID from path parameter
	projectIDParam := c.Param("id")
	projectID, err := uuid.Parse(projectIDParam)
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
		h.logger.Error("Failed to get project", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve project",
		})
	}

	// RBAC check is done in middleware (RequireProjectPermission)

	// Parse request
	var req models.AddProjectMemberRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := models.ValidateAddProjectMemberRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}

	// Check if user exists
	_, err = h.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		h.logger.Error("Failed to get user", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "User not found",
		})
	}

	// Check for duplicate member
	existingMember, err := h.projectMemberRepo.GetMemberRole(ctx, projectID, req.UserID)
	if err != nil && err != repository.ErrNotFound {
		h.logger.Error("Failed to check for existing member", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to check for existing member",
		})
	}
	if existingMember != nil {
		return c.JSON(http.StatusConflict, map[string]interface{}{
			"error": "User is already a member of this project",
		})
	}

	// Create project member
	member := &models.ProjectMember{
		ProjectID: projectID,
		UserID:    req.UserID,
		Role:      req.Role,
		AddedByID: claims.UserID,
	}

	if err := h.projectMemberRepo.AddMember(ctx, member); err != nil {
		h.logger.Error("Failed to add project member", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to add project member",
		})
	}

	h.logger.Info("Project member added successfully",
		zap.String("project_id", projectID.String()),
		zap.String("user_id", req.UserID.String()))
	return c.JSON(http.StatusCreated, member.ToResponse())
}

// ListProjectMembers handles GET /api/v1/projects/:id/members - list project members
func (h *ProjectHandler) ListProjectMembers(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse project ID from path parameter
	projectIDParam := c.Param("id")
	projectID, err := uuid.Parse(projectIDParam)
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
		h.logger.Error("Failed to get project", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve project",
		})
	}

	// RBAC check is done in middleware (RequireProjectPermission)

	// Get members
	members, err := h.projectMemberRepo.GetMembers(ctx, projectID)
	if err != nil {
		h.logger.Error("Failed to get project members", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve project members",
		})
	}

	// Convert to response format with user details
	responses := make([]map[string]interface{}, len(members))
	for i, member := range members {
		// Get user details
		user, err := h.userRepo.GetByID(ctx, member.UserID)
		if err != nil {
			// If user not found, still include member info but without user details
			responses[i] = map[string]interface{}{
				"member": member.ToResponse(),
			}
			continue
		}

		responses[i] = map[string]interface{}{
			"member": member.ToResponse(),
			"user": map[string]interface{}{
				"id":           user.ID,
				"email":        user.Email,
				"display_name": user.DisplayName,
			},
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"members": responses,
		"count":   len(responses),
	})
}

// RemoveProjectMember handles DELETE /api/v1/projects/:id/members/:user_id - remove member from project
func (h *ProjectHandler) RemoveProjectMember(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse project ID from path parameter
	projectIDParam := c.Param("id")
	projectID, err := uuid.Parse(projectIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid project ID format",
		})
	}

	// Parse user ID from path parameter
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid user ID format",
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
		h.logger.Error("Failed to get project", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve project",
		})
	}

	// RBAC check is done in middleware (RequireProjectPermission)

	// Check if member exists and get their role
	member, err := h.projectMemberRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "User is not a member of this project",
			})
		}
		h.logger.Error("Failed to get member", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve member",
		})
	}

	// Cannot remove Project Owner if they're the only owner
	if member.Role == rbac.ProjectOwner {
		allMembers, err := h.projectMemberRepo.GetMembers(ctx, projectID)
		if err != nil {
			h.logger.Error("Failed to get all members", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to check owners",
			})
		}

		ownerCount := 0
		for _, m := range allMembers {
			if m.Role == rbac.ProjectOwner {
				ownerCount++
			}
		}

		if ownerCount <= 1 {
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": "Cannot remove Project Owner: They are the only owner of this project",
			})
		}
	}

	// Remove member
	if err := h.projectMemberRepo.RemoveMember(ctx, projectID, userID); err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Member not found",
			})
		}
		h.logger.Error("Failed to remove project member", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to remove project member",
		})
	}

	h.logger.Info("Project member removed successfully",
		zap.String("project_id", projectID.String()),
		zap.String("user_id", userID.String()))
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Project member removed successfully",
	})
}

// UpdateProjectMemberRole handles PUT /api/v1/projects/:id/members/:user_id - update member role
func (h *ProjectHandler) UpdateProjectMemberRole(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse project ID from path parameter
	projectIDParam := c.Param("id")
	projectID, err := uuid.Parse(projectIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid project ID format",
		})
	}

	// Parse user ID from path parameter
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid user ID format",
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
		h.logger.Error("Failed to get project", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve project",
		})
	}

	// RBAC check is done in middleware (RequireProjectPermission)

	// Parse request
	var req struct {
		Role rbac.ProjectRole `json:"project_role" validate:"required,oneof=ProjectOwner ProjectDeveloper ProjectApprover ProjectViewer"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate role
	if !rbac.IsValidProjectRole(req.Role) {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid project role",
		})
	}

	// Check if member exists
	_, err = h.projectMemberRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "User is not a member of this project",
			})
		}
		h.logger.Error("Failed to get member", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve member",
		})
	}

	// Update role
	if err := h.projectMemberRepo.UpdateRole(ctx, projectID, userID, req.Role); err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Member not found",
			})
		}
		h.logger.Error("Failed to update project member role", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update project member role",
		})
	}

	// Get updated member
	updatedMember, err := h.projectMemberRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		h.logger.Error("Failed to get updated member", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve updated member",
		})
	}

	h.logger.Info("Project member role updated successfully",
		zap.String("project_id", projectID.String()),
		zap.String("user_id", userID.String()),
		zap.String("role", string(req.Role)))
	return c.JSON(http.StatusOK, updatedMember.ToResponse())
}
