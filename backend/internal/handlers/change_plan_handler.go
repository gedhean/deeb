package handlers

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/workflow"
	"go.uber.org/zap"
)

// ChangePlanHandler handles HTTP requests for change plan operations
type ChangePlanHandler struct {
	changePlanRepo    repository.ChangePlanRepository
	projectRepo       repository.ProjectRepository
	databaseRepo      repository.DatabaseRepository
	projectMemberRepo repository.ProjectMemberRepository
	logger            *zap.Logger
}

// NewChangePlanHandler creates a new change plan handler
func NewChangePlanHandler(
	changePlanRepo repository.ChangePlanRepository,
	projectRepo repository.ProjectRepository,
	databaseRepo repository.DatabaseRepository,
	projectMemberRepo repository.ProjectMemberRepository,
	logger *zap.Logger,
) *ChangePlanHandler {
	return &ChangePlanHandler{
		changePlanRepo:    changePlanRepo,
		projectRepo:       projectRepo,
		databaseRepo:      databaseRepo,
		projectMemberRepo: projectMemberRepo,
		logger:            logger,
	}
}

// CreateChangePlan handles POST /api/v1/change-plans
// Creates a new change plan (Project Developer role required)
func (h *ChangePlanHandler) CreateChangePlan(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse request
	var req models.CreateChangePlanRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := validateCreateChangePlanRequest(&req); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Validate that project exists
	if _, err := h.projectRepo.GetByID(ctx, req.ProjectID); err != nil {
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

	// Validate that database exists and belongs to the project
	database, err := h.databaseRepo.GetByID(ctx, req.TargetDatabaseID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Database not found",
			})
		}
		h.logger.Error("Failed to validate database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to validate database",
		})
	}

	// Verify database belongs to the project
	if database.ProjectID != req.ProjectID {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Database does not belong to the specified project",
		})
	}

	// Check RBAC: User must have Project Developer role in the project
	// Organization admins bypass project-level checks
	userIDStr, _, organizationRole, ok := rbac.GetUserFromContext(ctx)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Invalid user ID",
		})
	}

	// Organization admins can create change plans
	if organizationRole != rbac.OrganizationAdmin {
		// Check project membership and role
		member, err := h.projectMemberRepo.GetMemberRole(ctx, req.ProjectID, userID)
		if err != nil {
			if err == repository.ErrNotFound {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "Not a member of this project",
				})
			}
			h.logger.Error("Failed to check project membership", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to check project membership",
			})
		}

		// Convert to RBAC role and check permission
		projectRole := convertProjectRoleToRBAC(member.Role)
		checker := rbac.NewCheckerService()
		if !checker.HasProjectPermission(projectRole, rbac.PermissionChangesCreate) {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "Project Developer role required to create change plans",
				"details": map[string]interface{}{
					"user_project_role": projectRole,
				},
			})
		}
	}

	// Create change plan with Draft status
	changePlan := &models.ChangePlan{
		ProjectID:        req.ProjectID,
		TargetDatabaseID: req.TargetDatabaseID,
		Title:            req.Title,
		Description:      req.Description,
		ChangeType:       req.ChangeType,
		SQLContent:       req.SQLContent,
		RollbackSQL:      req.RollbackSQL,
		Status:           models.ChangePlanStatusDraft,
		RiskLevel:        models.RiskLevelMedium, // Default, will be calculated later
		AuthorID:         claims.UserID,
	}

	// Validate the change plan model
	if err := changePlan.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Save to repository
	if err := h.changePlanRepo.Create(ctx, changePlan); err != nil {
		h.logger.Error("Failed to create change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create change plan",
		})
	}

	h.logger.Info("Change plan created successfully",
		zap.String("change_plan_id", changePlan.ID.String()),
		zap.String("project_id", changePlan.ProjectID.String()),
	)

	return c.JSON(http.StatusCreated, changePlan.ToResponse())
}

// validateCreateChangePlanRequest validates the create change plan request
func validateCreateChangePlanRequest(req *models.CreateChangePlanRequest) error {
	if req.Title == "" {
		return models.ErrInvalidChangePlan("title is required")
	}
	if len(req.Title) > 255 {
		return models.ErrInvalidChangePlan("title must not exceed 255 characters")
	}
	if req.Description == "" {
		return models.ErrInvalidChangePlan("description is required")
	}
	if req.SQLContent == "" {
		return models.ErrInvalidChangePlan("sql_content is required")
	}
	if req.ChangeType != models.ChangeTypeDDL && req.ChangeType != models.ChangeTypeDML {
		return models.ErrInvalidChangePlan("change_type must be DDL or DML")
	}
	if req.ProjectID == uuid.Nil {
		return models.ErrInvalidChangePlan("project_id is required")
	}
	if req.TargetDatabaseID == uuid.Nil {
		return models.ErrInvalidChangePlan("target_database_id is required")
	}
	return nil
}

// GetChangePlan handles GET /api/v1/change-plans/:id
// Retrieves a single change plan by ID
func (h *ChangePlanHandler) GetChangePlan(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse change plan ID
	changePlanIDStr := c.Param("id")
	changePlanID, err := uuid.Parse(changePlanIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid change plan ID format",
		})
	}

	// Get change plan
	changePlan, err := h.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Change plan not found",
			})
		}
		h.logger.Error("Failed to get change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve change plan",
		})
	}

	// Check authorization: User must be able to view change plans in the project
	userIDStr, _, organizationRole, ok := rbac.GetUserFromContext(ctx)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Invalid user ID",
		})
	}

	// Organization admins can view any change plan
	if organizationRole != rbac.OrganizationAdmin {
		// Check if user is a member of the project
		member, err := h.projectMemberRepo.GetMemberRole(ctx, changePlan.ProjectID, userID)
		if err != nil {
			if err == repository.ErrNotFound {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "Not a member of this project",
				})
			}
			h.logger.Error("Failed to check project membership", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to check project membership",
			})
		}

		// Check if user has PermissionChangesRead
		projectRole := convertProjectRoleToRBAC(member.Role)
		checker := rbac.NewCheckerService()
		if !checker.HasProjectPermission(projectRole, rbac.PermissionChangesRead) {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "Insufficient permissions to view change plans",
			})
		}
	}

	return c.JSON(http.StatusOK, changePlan.ToResponse())
}

// ListChangePlans handles GET /api/v1/change-plans
// Lists change plans with optional filters
func (h *ChangePlanHandler) ListChangePlans(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
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
	statusStr := c.QueryParam("status")
	projectIDStr := c.QueryParam("project_id")
	databaseIDStr := c.QueryParam("database_id")
	authorIDStr := c.QueryParam("author_id")

	var changePlans []*models.ChangePlan
	var err error

	userIDStr, _, organizationRole, ok := rbac.GetUserFromContext(ctx)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Invalid user ID",
		})
	}

	// Organization admins can see all change plans
	// Other users can only see change plans in projects they're members of
	if organizationRole == rbac.OrganizationAdmin {
		// Admin: apply filters as requested
		if statusStr != "" {
			status := models.ChangePlanStatus(statusStr)
			if !status.IsValid() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Invalid status value",
				})
			}
			changePlans, err = h.changePlanRepo.GetByStatus(ctx, status, limit, offset)
			if err != nil {
				h.logger.Error("Failed to list change plans by status", zap.Error(err))
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Failed to retrieve change plans",
				})
			}
		} else if projectIDStr != "" {
			projectID, parseErr := uuid.Parse(projectIDStr)
			if parseErr != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Invalid project_id format",
				})
			}
			changePlans, err = h.changePlanRepo.GetByProjectID(ctx, projectID, limit, offset)
			if err != nil {
				h.logger.Error("Failed to list change plans by project", zap.Error(err))
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Failed to retrieve change plans",
				})
			}
		} else if databaseIDStr != "" {
			databaseID, parseErr := uuid.Parse(databaseIDStr)
			if parseErr != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Invalid database_id format",
				})
			}
			changePlans, err = h.changePlanRepo.GetByDatabaseID(ctx, databaseID, limit, offset)
			if err != nil {
				h.logger.Error("Failed to list change plans by database", zap.Error(err))
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Failed to retrieve change plans",
				})
			}
		} else if authorIDStr != "" {
			authorID, parseErr := uuid.Parse(authorIDStr)
			if parseErr != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Invalid author_id format",
				})
			}
			changePlans, err = h.changePlanRepo.GetByAuthorID(ctx, authorID, limit, offset)
			if err != nil {
				h.logger.Error("Failed to list change plans by author", zap.Error(err))
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Failed to retrieve change plans",
				})
			}
		} else {
			// List all change plans
			changePlans, err = h.changePlanRepo.List(ctx, limit, offset)
			if err != nil {
				h.logger.Error("Failed to list change plans", zap.Error(err))
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Failed to retrieve change plans",
				})
			}
		}
	} else {
		// Non-admin: filter by projects user is member of
		// For now, we'll filter by author_id (user's own change plans)
		// TODO: In future, fetch all projects user is member of and filter by those
		changePlans, err = h.changePlanRepo.GetByAuthorID(ctx, userID, limit, offset)
		if err != nil {
			h.logger.Error("Failed to list change plans", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve change plans",
			})
		}

		// Apply additional filters if provided
		if len(changePlans) > 0 {
			filtered := make([]*models.ChangePlan, 0)
			for _, cp := range changePlans {
				match := true

				if statusStr != "" && cp.Status != models.ChangePlanStatus(statusStr) {
					match = false
				}
				if projectIDStr != "" {
					projectID, _ := uuid.Parse(projectIDStr)
					if cp.ProjectID != projectID {
						match = false
					}
				}
				if databaseIDStr != "" {
					databaseID, _ := uuid.Parse(databaseIDStr)
					if cp.TargetDatabaseID != databaseID {
						match = false
					}
				}
				if authorIDStr != "" {
					authorID, _ := uuid.Parse(authorIDStr)
					if cp.AuthorID != authorID {
						match = false
					}
				}

				if match {
					filtered = append(filtered, cp)
				}
			}
			changePlans = filtered
		}
	}

	// Convert to response format
	responses := make([]*models.ChangePlanResponse, len(changePlans))
	for i, cp := range changePlans {
		responses[i] = cp.ToResponse()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_plans": responses,
		"count":        len(responses),
	})
}

// UpdateChangePlan handles PUT /api/v1/change-plans/:id
// Updates a change plan (only Draft status can be modified)
func (h *ChangePlanHandler) UpdateChangePlan(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse change plan ID
	changePlanIDStr := c.Param("id")
	changePlanID, err := uuid.Parse(changePlanIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid change plan ID format",
		})
	}

	// Get existing change plan
	changePlan, err := h.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Change plan not found",
			})
		}
		h.logger.Error("Failed to get change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve change plan",
		})
	}

	// Check authorization: Only author can update their Draft change plans
	userIDStr, _, organizationRole, ok := rbac.GetUserFromContext(ctx)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Invalid user ID",
		})
	}

	// Organization admins can update any change plan
	if organizationRole != rbac.OrganizationAdmin {
		// Check if user is the author
		if changePlan.AuthorID != userID {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "Only the author can update this change plan",
			})
		}

		// Check project membership and role
		member, err := h.projectMemberRepo.GetMemberRole(ctx, changePlan.ProjectID, userID)
		if err != nil {
			if err == repository.ErrNotFound {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "Not a member of this project",
				})
			}
			h.logger.Error("Failed to check project membership", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to check project membership",
			})
		}

		// Check permission
		projectRole := convertProjectRoleToRBAC(member.Role)
		checker := rbac.NewCheckerService()
		if !checker.HasProjectPermission(projectRole, rbac.PermissionChangesUpdate) {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "Project Developer role required to update change plans",
			})
		}
	}

	// Only Draft status can be modified
	if !workflow.CanBeModified(changePlan.Status) {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "Only Draft change plans can be modified",
			"details": map[string]interface{}{
				"current_status": changePlan.Status,
			},
		})
	}

	// Parse request
	var req models.UpdateChangePlanRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Update fields if provided
	if req.Title != nil {
		if *req.Title == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "title cannot be empty",
			})
		}
		if len(*req.Title) > 255 {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "title must not exceed 255 characters",
			})
		}
		changePlan.Title = *req.Title
	}

	if req.Description != nil {
		if *req.Description == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "description cannot be empty",
			})
		}
		changePlan.Description = *req.Description
	}

	if req.SQLContent != nil {
		if *req.SQLContent == "" {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "sql_content cannot be empty",
			})
		}
		// Validate SQL content size limit (10MB)
		if len(*req.SQLContent) > MaxSQLContentSize {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "sql_content exceeds maximum size of 10MB",
			})
		}
		changePlan.SQLContent = *req.SQLContent
	}

	if req.RollbackSQL != nil {
		changePlan.RollbackSQL = req.RollbackSQL
	}

	if req.Status != nil {
		// Validate status transition
		if err := workflow.ValidateTransition(changePlan.Status, *req.Status); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
			})
		}
		changePlan.Status = *req.Status
	}

	if req.RiskLevel != nil {
		if *req.RiskLevel != models.RiskLevelLow &&
			*req.RiskLevel != models.RiskLevelMedium &&
			*req.RiskLevel != models.RiskLevelHigh {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "risk_level must be Low, Medium, or High",
			})
		}
		changePlan.RiskLevel = *req.RiskLevel
	}

	if req.AssignedReviewerID != nil {
		changePlan.AssignedReviewerID = req.AssignedReviewerID
	}

	// Validate the updated change plan
	if err := changePlan.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Update in repository
	if err := h.changePlanRepo.Update(ctx, changePlan); err != nil {
		h.logger.Error("Failed to update change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update change plan",
		})
	}

	h.logger.Info("Change plan updated successfully",
		zap.String("change_plan_id", changePlan.ID.String()),
	)

	return c.JSON(http.StatusOK, changePlan.ToResponse())
}

// DeleteChangePlan handles DELETE /api/v1/change-plans/:id
// Deletes a change plan (only Draft or Closed can be deleted)
func (h *ChangePlanHandler) DeleteChangePlan(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse change plan ID
	changePlanIDStr := c.Param("id")
	changePlanID, err := uuid.Parse(changePlanIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid change plan ID format",
		})
	}

	// Get existing change plan
	changePlan, err := h.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Change plan not found",
			})
		}
		h.logger.Error("Failed to get change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve change plan",
		})
	}

	// Check authorization: Only author can delete their change plans
	userIDStr, _, organizationRole, ok := rbac.GetUserFromContext(ctx)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Invalid user ID",
		})
	}

	// Organization admins can delete any change plan
	if organizationRole != rbac.OrganizationAdmin {
		// Check if user is the author
		if changePlan.AuthorID != userID {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "Only the author can delete this change plan",
			})
		}

		// Check project membership and role
		member, err := h.projectMemberRepo.GetMemberRole(ctx, changePlan.ProjectID, userID)
		if err != nil {
			if err == repository.ErrNotFound {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "Not a member of this project",
				})
			}
			h.logger.Error("Failed to check project membership", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to check project membership",
			})
		}

		// Check permission
		projectRole := convertProjectRoleToRBAC(member.Role)
		checker := rbac.NewCheckerService()
		if !checker.HasProjectPermission(projectRole, rbac.PermissionChangesDelete) {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "Project Developer role required to delete change plans",
			})
		}
	}

	// Only Draft or Closed status can be deleted
	if !workflow.CanBeDeleted(changePlan.Status) {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "Only Draft or Closed change plans can be deleted",
			"details": map[string]interface{}{
				"current_status": changePlan.Status,
			},
		})
	}

	// Delete from repository (cascade deletion of policy findings is handled by DB)
	if err := h.changePlanRepo.Delete(ctx, changePlanID); err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Change plan not found",
			})
		}
		h.logger.Error("Failed to delete change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to delete change plan",
		})
	}

	h.logger.Info("Change plan deleted successfully",
		zap.String("change_plan_id", changePlanID.String()),
	)

	return c.NoContent(http.StatusNoContent)
}

// convertProjectRoleToRBAC converts models.ProjectRole to rbac.ProjectRole
func convertProjectRoleToRBAC(role models.ProjectRole) rbac.ProjectRole {
	switch role {
	case models.ProjectRoleOwner:
		return rbac.ProjectOwner
	case models.ProjectRoleDeveloper:
		return rbac.ProjectDeveloper
	case models.ProjectRoleApprover:
		return rbac.ProjectApprover
	case models.ProjectRoleViewer:
		return rbac.ProjectViewer
	default:
		return rbac.ProjectViewer
	}
}

