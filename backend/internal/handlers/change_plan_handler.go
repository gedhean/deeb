package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/middleware"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/review"
	"github.com/nsxbet/deeb/internal/services"
	"github.com/nsxbet/deeb/internal/workflow"
	"go.uber.org/zap"
)

// ChangePlanHandler handles HTTP requests for change plan operations
type ChangePlanHandler struct {
	changePlanRepo       repository.ChangePlanRepository
	changeHistoryRepo    repository.ChangeHistoryRepository
	projectRepo          repository.ProjectRepository
	databaseRepo         repository.DatabaseRepository
	databaseInstanceRepo repository.DatabaseInstanceRepository
	projectMemberRepo    repository.ProjectMemberRepository
	policyFindingRepo    repository.PolicyFindingRepository
	approvalRepo         repository.ApprovalRepository
	riskAssessmentSvc    *services.RiskAssessmentService
	reviewService        *review.ReviewService
	approvalService      *services.ApprovalService
	executionService     *services.ExecutionService
	auditService         *services.AuditLogService
	logger               *zap.Logger
}

// NewChangePlanHandler creates a new change plan handler
func NewChangePlanHandler(
	changePlanRepo repository.ChangePlanRepository,
	changeHistoryRepo repository.ChangeHistoryRepository,
	projectRepo repository.ProjectRepository,
	databaseRepo repository.DatabaseRepository,
	databaseInstanceRepo repository.DatabaseInstanceRepository,
	projectMemberRepo repository.ProjectMemberRepository,
	policyFindingRepo repository.PolicyFindingRepository,
	approvalRepo repository.ApprovalRepository,
	riskAssessmentSvc *services.RiskAssessmentService,
	reviewService *review.ReviewService,
	approvalService *services.ApprovalService,
	executionService *services.ExecutionService,
	auditService *services.AuditLogService,
	logger *zap.Logger,
) *ChangePlanHandler {
	return &ChangePlanHandler{
		changePlanRepo:       changePlanRepo,
		changeHistoryRepo:    changeHistoryRepo,
		projectRepo:          projectRepo,
		databaseRepo:         databaseRepo,
		databaseInstanceRepo: databaseInstanceRepo,
		projectMemberRepo:    projectMemberRepo,
		policyFindingRepo:    policyFindingRepo,
		approvalRepo:         approvalRepo,
		riskAssessmentSvc:    riskAssessmentSvc,
		reviewService:        reviewService,
		approvalService:      approvalService,
		executionService:     executionService,
		auditService:         auditService,
		logger:               logger,
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
		return middleware.HandleValidationError(c, err)
	}

	// Validate request
	if err := validateCreateChangePlanRequest(&req); err != nil {
		h.logger.Error("Validation failed", zap.Error(err))
		return middleware.HandleValidationError(c, err)
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
		RiskLevel:        models.RiskLevelMedium, // Default, will be calculated
		AuthorID:         claims.UserID,
	}

	// Assess risk level based on SQL content, environment, and database type
	if h.riskAssessmentSvc != nil {
		riskLevel, err := h.riskAssessmentSvc.AssessRisk(ctx, req.SQLContent, req.ChangeType, req.TargetDatabaseID)
		if err != nil {
			h.logger.Warn("Failed to assess risk level, using default", zap.Error(err))
			// Continue with default Medium risk level
		} else {
			changePlan.RiskLevel = riskLevel
		}
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

	// Log audit entry
	if h.auditService != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"title":              changePlan.Title,
			"description":        changePlan.Description,
			"change_type":        string(changePlan.ChangeType),
			"project_id":         changePlan.ProjectID.String(),
			"target_database_id": changePlan.TargetDatabaseID.String(),
			"risk_level":         string(changePlan.RiskLevel),
			"status":             string(changePlan.Status),
		})
		if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionCreate, services.ResourceTypeChangePlan, changePlan.ID, c.Request(), detailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for change plan creation", zap.Error(err))
		}
	}

	return c.JSON(http.StatusCreated, changePlan.ToResponse())
}

// validateUpdateChangePlanRequest validates the update change plan request
func validateUpdateChangePlanRequest(req *models.UpdateChangePlanRequest) *middleware.ValidationError {
	ve := &middleware.ValidationError{}

	if req.Title != nil {
		if *req.Title == "" {
			ve.AddField("title", "title cannot be empty", "required")
		} else if len(*req.Title) > 255 {
			ve.AddField("title", "title must not exceed 255 characters", "max")
		}
	}

	if req.Description != nil {
		if *req.Description == "" {
			ve.AddField("description", "description cannot be empty", "required")
		}
	}

	if req.SQLContent != nil {
		if *req.SQLContent == "" {
			ve.AddField("sql_content", "sql_content cannot be empty", "required")
		} else if len(*req.SQLContent) > MaxSQLContentSize {
			ve.AddField("sql_content", "sql_content exceeds maximum size of 10MB", "max")
		}
	}

	if len(ve.Fields) > 0 {
		return ve
	}

	return nil
}

// validateCreateChangePlanRequest validates the create change plan request
func validateCreateChangePlanRequest(req *models.CreateChangePlanRequest) error {
	ve := &middleware.ValidationError{}

	if req.Title == "" {
		ve.AddField("title", "title is required", "required")
	} else if len(req.Title) > 255 {
		ve.AddField("title", "title must not exceed 255 characters", "max")
	}

	// Description is optional - no validation needed

	if req.SQLContent == "" {
		ve.AddField("sql_content", "sql_content is required", "required")
	} else if len(req.SQLContent) > MaxSQLContentSize {
		ve.AddField("sql_content", "sql_content exceeds maximum size of 10MB", "max")
	}

	if req.ChangeType != models.ChangeTypeDDL && req.ChangeType != models.ChangeTypeDML {
		ve.AddField("change_type", "change_type must be DDL or DML", "enum")
	}

	if req.ProjectID == uuid.Nil {
		ve.AddField("project_id", "project_id is required", "required")
	}

	if req.TargetDatabaseID == uuid.Nil {
		ve.AddField("target_database_id", "target_database_id is required", "required")
	}

	if len(ve.Fields) > 0 {
		return ve
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

	h.logger.Info("Change plan retrieved successfully",
		zap.String("change_plan_id", changePlan.ID.String()),
	)

	// Fetch policy findings for this change plan
	findings, err := h.policyFindingRepo.GetByChangePlanID(ctx, changePlan.ID)
	if err != nil {
		h.logger.Warn("Failed to fetch policy findings", zap.Error(err))
		// Continue - findings are optional
		findings = []*models.PolicyFinding{}
	}

	// Build response with policy findings
	response := changePlan.ToResponse()
	response = h.enrichWithPolicyFindings(response, findings)

	// Fetch execution logs and enrich response
	executionLogs, err := h.executionService.GetExecutionLogHistory(ctx, changePlan.ID)
	if err != nil {
		h.logger.Warn("Failed to fetch execution logs", zap.Error(err))
		// Continue - execution logs are optional
		executionLogs = []*models.ExecutionLog{}
	}

	// Convert execution logs to response format
	if len(executionLogs) > 0 {
		logResponses := make([]*models.ExecutionLogResponse, len(executionLogs))
		for i, log := range executionLogs {
			logResponses[i] = log.ToResponse()
		}
		response.ExecutionLogs = logResponses

		// Set last execution status from the most recent log (first in the list)
		lastStatus := executionLogs[0].Status
		response.LastExecutionStatus = &lastStatus
	}

	return c.JSON(http.StatusOK, response)
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

	// Parse filter parameters
	var filters repository.ChangePlanListFilters
	filters.Limit = limit
	filters.Offset = offset

	// Parse status filter
	if statusStr := c.QueryParam("status"); statusStr != "" {
		status := models.ChangePlanStatus(statusStr)
		if !status.IsValid() {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid status value",
			})
		}
		filters.Status = status
	}

	// Parse project_id filter
	if projectIDStr := c.QueryParam("project_id"); projectIDStr != "" {
		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid project_id format",
			})
		}
		filters.ProjectID = projectID
	}

	// Parse database_id filter
	if databaseIDStr := c.QueryParam("database_id"); databaseIDStr != "" {
		databaseID, err := uuid.Parse(databaseIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid database_id format",
			})
		}
		filters.DatabaseID = databaseID
	}

	// Parse author_id filter
	if authorIDStr := c.QueryParam("author_id"); authorIDStr != "" {
		authorID, err := uuid.Parse(authorIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid author_id format",
			})
		}
		filters.AuthorID = authorID
	}

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
	if organizationRole != rbac.OrganizationAdmin {
		// For non-admins, get all projects the user is a member of
		// and filter change plans to only those projects
		// For MVP, we'll filter by author_id (user's own change plans)
		// TODO: In future, fetch all projects user is member of and filter by those
		filters.AuthorID = userID // Override author_id filter for non-admins
	}

	// Retrieve change plans with filters
	changePlans, err := h.changePlanRepo.ListWithFilters(ctx, filters)
	if err != nil {
		h.logger.Error("Failed to list change plans", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve change plans",
		})
	}

	// Get total count with same filters
	countFilters := filters
	countFilters.Limit = 0
	countFilters.Offset = 0
	total, err := h.changePlanRepo.CountWithFilters(ctx, countFilters)
	if err != nil {
		h.logger.Error("Failed to count change plans", zap.Error(err))
		// Don't fail the request, just use the current page count
		total = int64(len(changePlans))
	}

	// Convert to response format
	responses := make([]*models.ChangePlanResponse, len(changePlans))
	for i, cp := range changePlans {
		responses[i] = cp.ToResponse()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_plans": responses,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
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

	// Parse request
	var req models.UpdateChangePlanRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		return middleware.HandleValidationError(c, err)
	}

	// Validate update request
	ve := validateUpdateChangePlanRequest(&req)
	if ve != nil {
		h.logger.Error("Validation failed", zap.Error(ve))
		return middleware.HandleValidationError(c, ve)
	}

	// Check if SQL content is being updated
	isUpdatingSQLContent := req.SQLContent != nil && *req.SQLContent != changePlan.SQLContent

	// If updating SQL content, allow editing if status allows SQL editing (not terminal)
	// Otherwise, only Draft status can be modified
	if isUpdatingSQLContent {
		if !workflow.CanEditSQLContent(changePlan.Status) {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "Cannot edit SQL content for change plans in terminal states",
				"details": map[string]interface{}{
					"current_status": changePlan.Status,
				},
			})
		}
	} else {
		// For other fields, only Draft status can be modified
		if !workflow.CanBeModified(changePlan.Status) {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "Only Draft change plans can be modified",
				"details": map[string]interface{}{
					"current_status": changePlan.Status,
				},
			})
		}
	}

	// Update fields if provided
	if req.Title != nil {
		changePlan.Title = *req.Title
	}

	if req.Description != nil {
		changePlan.Description = *req.Description
	}

	if req.SQLContent != nil {
		changePlan.SQLContent = *req.SQLContent

		// Re-assess risk level when SQL content changes
		if h.riskAssessmentSvc != nil {
			riskLevel, err := h.riskAssessmentSvc.AssessRisk(
				ctx,
				*req.SQLContent,
				changePlan.ChangeType,
				changePlan.TargetDatabaseID,
			)
			if err != nil {
				h.logger.Warn("Failed to reassess risk level after SQL content change", zap.Error(err))
				// Continue with existing risk level
			} else {
				changePlan.RiskLevel = riskLevel
			}
		}

		// If SQL content is updated and status is not "Open" and not terminal, transition to "Open"
		if changePlan.Status != models.ChangePlanStatusOpen && workflow.CanEditSQLContent(changePlan.Status) {
			previousStatus := changePlan.Status
			changePlan.Status = models.ChangePlanStatusOpen
			h.logger.Info("Status transitioned to Open due to SQL content update",
				zap.String("change_plan_id", changePlan.ID.String()),
				zap.String("previous_status", string(previousStatus)),
			)
		}

		// Delete existing approval records when SQL content is updated
		// This allows approvers to approve the updated version
		// Delete approvals whether status transitions to Open or is already Open
		if changePlan.Status == models.ChangePlanStatusOpen {
			if err := h.approvalRepo.DeleteByChangePlanID(ctx, changePlan.ID); err != nil {
				h.logger.Warn("Failed to delete existing approvals after SQL content update",
					zap.String("change_plan_id", changePlan.ID.String()),
					zap.Error(err),
				)
				// Don't block the update - continue even if approval deletion fails
			} else {
				h.logger.Info("Deleted existing approvals after SQL content update",
					zap.String("change_plan_id", changePlan.ID.String()),
				)
			}
		}
	}

	if req.RollbackSQL != nil {
		changePlan.RollbackSQL = req.RollbackSQL
	}

	// Capture old status before update if status is being changed
	var oldStatus models.ChangePlanStatus
	if req.Status != nil {
		oldStatus = changePlan.Status
	}

	if req.Status != nil {
		// Validate status transition
		if err := workflow.ValidateTransition(changePlan.Status, *req.Status); err != nil {
			// Check if it's a status transition error to return structured response
			if transitionErr, ok := err.(workflow.ErrInvalidStatusTransition); ok {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error":   transitionErr.UserMessage(),
					"details": transitionErr.Details(),
				})
			}

			// Fallback for other error types
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"details": map[string]interface{}{
					"current_status":   string(changePlan.Status),
					"attempted_status": string(*req.Status),
				},
			})
		}

		// If transitioning from Draft to Open, trigger SQL Review
		if changePlan.Status == models.ChangePlanStatusDraft && *req.Status == models.ChangePlanStatusOpen {
			if err := h.triggerSQLReview(c, changePlan); err != nil {
				h.logger.Error("SQL Review failed", zap.Error(err), zap.String("change_plan_id", changePlan.ID.String()))

				// Check if errors prevent transition
				if reviewErr, ok := err.(*SQLReviewError); ok && reviewErr.HasErrors {
					return c.JSON(http.StatusBadRequest, map[string]interface{}{
						"error": "SQL Review failed - cannot transition to Open status",
						"details": map[string]interface{}{
							"review_result":  reviewErr.ReviewResult,
							"errors_count":   reviewErr.ReviewResult.Errors,
							"warnings_count": reviewErr.ReviewResult.Warnings,
						},
					})
				}

				// Other errors (e.g., policy loading failure) - allow transition but log
				h.logger.Warn("SQL Review encountered issues but allowing transition", zap.Error(err))
			}
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

	// If SQL content was updated, trigger SQL Review
	if isUpdatingSQLContent {
		if err := h.triggerSQLReview(c, changePlan); err != nil {
			h.logger.Error(
				"SQL Review failed after SQL content update",
				zap.Error(err),
				zap.String("change_plan_id", changePlan.ID.String()),
			)
			// Don't block the update - SQL Review errors don't prevent saving the SQL content
			// The review results will be stored and displayed to the user
		}
	}

	// Track field changes before updating
	oldChangePlan := *changePlan // Make a copy of old values

	// Update in repository
	if err := h.changePlanRepo.Update(ctx, changePlan); err != nil {
		h.logger.Error("Failed to update change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update change plan",
		})
	}

	// Track field changes after successful update
	if h.changeHistoryRepo != nil {
		h.trackChangeHistory(ctx, &oldChangePlan, changePlan, userID)
	}

	h.logger.Info("Change plan updated successfully",
		zap.String("change_plan_id", changePlan.ID.String()),
	)

	// Log audit entry
	if h.auditService != nil {
		claims, ok := auth.GetUserFromEchoContext(c)
		if ok && claims != nil {
			detailsJSON, _ := json.Marshal(map[string]interface{}{
				"title":       changePlan.Title,
				"description": changePlan.Description,
				"change_type": string(changePlan.ChangeType),
				"risk_level":  string(changePlan.RiskLevel),
			})
			if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionUpdate, services.ResourceTypeChangePlan, changePlan.ID, c.Request(), detailsJSON); err != nil {
				h.logger.Error("Failed to log audit entry for change plan update", zap.Error(err))
			}

			// Log status transition if status was updated
			if req.Status != nil && oldStatus != *req.Status {
				transitionDetailsJSON, _ := json.Marshal(map[string]interface{}{
					"change_plan_id": changePlan.ID.String(),
					"title":          changePlan.Title,
				})
				if err := h.auditService.LogStateTransition(ctx, claims.UserID, services.ResourceTypeChangePlan, changePlan.ID, string(oldStatus), string(*req.Status), transitionDetailsJSON); err != nil {
					h.logger.Error("Failed to log audit entry for status transition", zap.Error(err))
				}
			}
		}
	}

	return c.JSON(http.StatusOK, changePlan.ToResponse())
}

// SQLReviewError represents an error during SQL Review
type SQLReviewError struct {
	HasErrors    bool
	ReviewResult *review.ReviewResultWithSummary
	Message      string
}

func (e *SQLReviewError) Error() string {
	return e.Message
}

// triggerSQLReview triggers SQL Review for a change plan and stores findings
func (h *ChangePlanHandler) triggerSQLReview(c echo.Context, changePlan *models.ChangePlan) error {
	if h.reviewService == nil {
		h.logger.Warn("ReviewService not configured, skipping SQL Review")
		return nil
	}

	ctx := c.Request().Context()

	// Get database type from database instance
	database, err := h.databaseRepo.GetByID(ctx, changePlan.TargetDatabaseID)
	if err != nil {
		return &SQLReviewError{
			HasErrors: false,
			Message:   "Failed to get database: " + err.Error(),
		}
	}

	databaseInstance, err := h.databaseInstanceRepo.GetByID(ctx, database.DatabaseInstanceID)
	if err != nil {
		return &SQLReviewError{
			HasErrors: false,
			Message:   "Failed to get database instance: " + err.Error(),
		}
	}

	// Run SQL Review
	reviewResult, err := h.reviewService.Review(ctx, review.ReviewOptions{
		ChangePlanID: changePlan.ID,
		SQLContent:   changePlan.SQLContent,
		DatabaseType: databaseInstance.DatabaseType,
	})
	if err != nil {
		return &SQLReviewError{
			HasErrors: false,
			Message:   "Failed to run SQL Review: " + err.Error(),
		}
	}

	// Delete existing findings for this change plan
	if err := h.policyFindingRepo.DeleteByChangePlanID(ctx, changePlan.ID); err != nil {
		h.logger.Warn("Failed to delete existing findings", zap.Error(err))
		// Continue - we'll try to create new ones
	}

	// Convert review results to policy findings and store them
	if err := h.storeReviewFindings(ctx, changePlan.ID, reviewResult); err != nil {
		h.logger.Error("Failed to store review findings", zap.Error(err))
		return &SQLReviewError{
			HasErrors:    reviewResult.Errors > 0,
			ReviewResult: reviewResult,
			Message:      "Failed to store review findings: " + err.Error(),
		}
	}

	// Check if there are errors that should prevent status transition
	if reviewResult.Errors > 0 {
		return &SQLReviewError{
			HasErrors:    true,
			ReviewResult: reviewResult,
			Message:      "SQL Review found errors that prevent status transition",
		}
	}

	h.logger.Info("SQL Review completed successfully",
		zap.String("change_plan_id", changePlan.ID.String()),
		zap.Int("errors", reviewResult.Errors),
		zap.Int("warnings", reviewResult.Warnings),
		zap.Int("infos", reviewResult.Infos),
	)

	return nil
}

// storeReviewFindings converts review results to policy findings and stores them
func (h *ChangePlanHandler) storeReviewFindings(
	ctx context.Context,
	changePlanID uuid.UUID,
	reviewResult *review.ReviewResultWithSummary,
) error {
	findings := make([]*models.PolicyFinding, 0)

	for _, result := range reviewResult.Results {
		for _, violation := range result.Violations {
			// Map validator name to validator type
			validatorType := mapValidatorNameToType(result.ValidatorName)

			finding := &models.PolicyFinding{
				ChangePlanID:  changePlanID,
				ValidatorType: validatorType,
				Severity:      string(violation.Severity),
				RuleName:      violation.Rule,
				Message:       violation.Message,
			}

			if violation.Line > 0 {
				lineNum := violation.Line
				finding.LineNumber = &lineNum
			}

			if violation.Column > 0 {
				colNum := violation.Column
				finding.ColumnNumber = &colNum
			}

			findings = append(findings, finding)
		}
	}

	// Batch insert findings
	if len(findings) > 0 {
		if err := h.policyFindingRepo.CreateBatch(ctx, findings); err != nil {
			return err
		}
	}

	return nil
}

// mapValidatorNameToType maps validator name to validator type string
func mapValidatorNameToType(validatorName string) string {
	switch validatorName {
	case "postgresql-syntax", "mysql-syntax":
		return "syntax_error"
	case "security":
		return "security_risk"
	case "performance":
		return "performance_issue"
	case "backwards-compatibility":
		return "backwards_incompatible"
	case "best-practices":
		return "best_practice_violation"
	default:
		return "best_practice_violation" // Default fallback
	}
}

// TriggerSQLReview handles POST /api/v1/change-plans/:id/review
// Manually triggers SQL Review for an existing Change Plan
func (h *ChangePlanHandler) TriggerSQLReview(c echo.Context) error {
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

	// Check authorization: User must have Project Developer role in the project
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

	// Organization admins can trigger review
	if organizationRole != rbac.OrganizationAdmin {
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

		// Convert to RBAC role and check permission
		projectRole := convertProjectRoleToRBAC(member.Role)
		checker := rbac.NewCheckerService()
		if !checker.HasProjectPermission(projectRole, rbac.PermissionChangesCreate) {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "Project Developer role required to trigger SQL Review",
				"details": map[string]interface{}{
					"user_project_role": projectRole,
				},
			})
		}
	}

	// Trigger SQL Review (without blocking on errors - manual review always returns results)
	if h.reviewService == nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "SQL Review service not configured",
		})
	}

	// Get database type from database instance
	database, err := h.databaseRepo.GetByID(ctx, changePlan.TargetDatabaseID)
	if err != nil {
		h.logger.Error("Failed to get database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get database",
		})
	}

	databaseInstance, err := h.databaseInstanceRepo.GetByID(ctx, database.DatabaseInstanceID)
	if err != nil {
		h.logger.Error("Failed to get database instance", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get database instance",
		})
	}

	// Run SQL Review
	reviewResult, err := h.reviewService.Review(ctx, review.ReviewOptions{
		ChangePlanID: changePlan.ID,
		SQLContent:   changePlan.SQLContent,
		DatabaseType: databaseInstance.DatabaseType,
	})
	if err != nil {
		h.logger.Error("Failed to run SQL Review", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to run SQL Review: " + err.Error(),
		})
	}

	// Delete existing findings for this change plan
	if err := h.policyFindingRepo.DeleteByChangePlanID(ctx, changePlan.ID); err != nil {
		h.logger.Warn("Failed to delete existing findings", zap.Error(err))
		// Continue - we'll try to create new ones
	}

	// Convert review results to policy findings and store them
	if err := h.storeReviewFindings(ctx, changePlan.ID, reviewResult); err != nil {
		h.logger.Error("Failed to store review findings", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to store review findings: " + err.Error(),
		})
	}

	h.logger.Info("SQL Review triggered manually",
		zap.String("change_plan_id", changePlan.ID.String()),
		zap.Int("errors", reviewResult.Errors),
		zap.Int("warnings", reviewResult.Warnings),
		zap.Int("infos", reviewResult.Infos),
	)

	// Return review results in response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_plan_id": changePlan.ID,
		"review_result": map[string]interface{}{
			"valid":              reviewResult.Valid,
			"errors":             reviewResult.Errors,
			"warnings":           reviewResult.Warnings,
			"infos":              reviewResult.Infos,
			"results":            reviewResult.Results,
			"policy_id":          reviewResult.PolicyID,
			"policy_name":        reviewResult.PolicyName,
			"enabled_validators": reviewResult.EnabledValidators,
			"skipped_validators": reviewResult.SkippedValidators,
		},
	})
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

	// Log audit entry
	if h.auditService != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"change_plan_id": changePlanID.String(),
			"title":          changePlan.Title,
			"status":         string(changePlan.Status),
		})
		if err := h.auditService.LogWithHTTP(ctx, userID, services.ActionDelete, services.ResourceTypeChangePlan, changePlanID, c.Request(), detailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for change plan deletion", zap.Error(err))
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// enrichWithPolicyFindings enriches ChangePlanResponse with policy findings
func (h *ChangePlanHandler) enrichWithPolicyFindings(
	response *models.ChangePlanResponse,
	findings []*models.PolicyFinding,
) *models.ChangePlanResponse {
	if len(findings) == 0 {
		response.HasErrors = false
		response.HasWarnings = false
		return response
	}

	// Group findings by severity
	errors := make([]models.PolicyFindingItem, 0)
	warnings := make([]models.PolicyFindingItem, 0)
	infos := make([]models.PolicyFindingItem, 0)

	for _, finding := range findings {
		item := models.PolicyFindingItem{
			ValidatorType: finding.ValidatorType,
			RuleName:      finding.RuleName,
			Message:       finding.Message,
			LineNumber:    finding.LineNumber,
			ColumnNumber:  finding.ColumnNumber,
		}

		switch finding.Severity {
		case "Error":
			errors = append(errors, item)
		case "Warning":
			warnings = append(warnings, item)
		case "Info":
			infos = append(infos, item)
		}
	}

	// Set flags
	response.HasErrors = len(errors) > 0
	response.HasWarnings = len(warnings) > 0

	// Create summary
	response.PolicyFindings = &models.PolicyFindingsSummary{
		Errors:   errors,
		Warnings: warnings,
		Infos:    infos,
		Counts: models.PolicyFindingCounts{
			Errors:   len(errors),
			Warnings: len(warnings),
			Infos:    len(infos),
			Total:    len(findings),
		},
	}

	return response
}

// ApproveChangePlanRequest represents the request body for approving a change plan
type ApproveChangePlanRequest struct {
	Comments *string `json:"comments,omitempty"`
}

// ApproveChangePlan handles POST /api/v1/change-plans/:id/approve
// Approves a change plan, blocking if SQL Review errors exist
func (h *ChangePlanHandler) ApproveChangePlan(c echo.Context) error {
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

	// Parse request body (optional comments)
	var req ApproveChangePlanRequest
	if err := c.Bind(&req); err != nil {
		// If binding fails, continue with empty comments (comments are optional)
		req.Comments = nil
	}

	// Get change plan to capture old status for audit logging
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
	oldStatus := changePlan.Status

	// Use approval service to approve the change plan
	// The service handles:
	// - Change plan existence check
	// - Status validation (must be Open)
	// - RBAC permission checks
	// - SQL Review errors check
	// - Approval record creation
	// - Change plan status update
	err = h.approvalService.Approve(ctx, changePlanID, claims.UserID, req.Comments)
	if err != nil {
		// Handle specific error types
		if err == services.ErrChangePlanNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Change plan not found",
			})
		}
		if err == services.ErrInvalidChangePlanStatus {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
			})
		}
		if err == services.ErrUnauthorizedApprover {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": err.Error(),
			})
		}
		if err == services.ErrCannotApproveWithErrors {
			// Extract error details from the error message
			// The service error message includes details about which errors exist
			errorMsg := err.Error()
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Cannot approve change plan with SQL Review errors",
				"details": map[string]interface{}{
					"message": errorMsg,
				},
			})
		}
		if err == services.ErrChangePlanAlreadyApproved {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
			})
		}
		if err == services.ErrChangePlanAlreadyClosed {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
			})
		}

		// Generic error
		h.logger.Error("Failed to approve change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to approve change plan",
		})
	}

	h.logger.Info("Change plan approved successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("approved_by", claims.UserID.String()),
	)

	// Log audit entry for status transition
	if h.auditService != nil {
		transitionDetailsJSON, _ := json.Marshal(map[string]interface{}{
			"change_plan_id": changePlanID.String(),
			"title":          changePlan.Title,
			"comments":       req.Comments,
		})
		if err := h.auditService.LogStateTransition(ctx, claims.UserID, services.ResourceTypeChangePlan, changePlanID, string(oldStatus), string(models.ChangePlanStatusApproved), transitionDetailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for approval", zap.Error(err))
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":        "Change plan approved successfully",
		"change_plan_id": changePlanID.String(),
	})
}

// CloseChangePlanRequest represents the request body for closing/rejecting a change plan
type CloseChangePlanRequest struct {
	Comments *string `json:"comments,omitempty"`
}

// CloseChangePlan handles POST /api/v1/change-plans/:id/close
// Closes/rejects a change plan
func (h *ChangePlanHandler) CloseChangePlan(c echo.Context) error {
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

	// Parse request body (optional comments)
	var req CloseChangePlanRequest
	if err := c.Bind(&req); err != nil {
		// If binding fails, continue with empty comments (comments are optional)
		req.Comments = nil
	}

	// Get change plan to capture old status for audit logging
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
	oldStatus := changePlan.Status

	// Use approval service to close the change plan
	// The service handles:
	// - Change plan existence check
	// - Status validation (must be Open)
	// - RBAC permission checks
	// - Approval record creation with Rejected status
	// - Change plan status update to Closed
	err = h.approvalService.Close(ctx, changePlanID, claims.UserID, req.Comments)
	if err != nil {
		// Handle specific error types
		if err == services.ErrChangePlanNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Change plan not found",
			})
		}
		if err == services.ErrInvalidChangePlanStatus {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
			})
		}
		if err == services.ErrUnauthorizedApprover {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": err.Error(),
			})
		}
		if err == services.ErrChangePlanAlreadyApproved {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
			})
		}
		if err == services.ErrChangePlanAlreadyClosed {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
			})
		}

		// Generic error
		h.logger.Error("Failed to close change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to close change plan",
		})
	}

	h.logger.Info("Change plan closed successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("closed_by", claims.UserID.String()),
	)

	// Log audit entry for status transition
	if h.auditService != nil {
		transitionDetailsJSON, _ := json.Marshal(map[string]interface{}{
			"change_plan_id": changePlanID.String(),
			"title":          changePlan.Title,
			"comments":       req.Comments,
		})
		if err := h.auditService.LogStateTransition(ctx, claims.UserID, services.ResourceTypeChangePlan, changePlanID, string(oldStatus), string(models.ChangePlanStatusClosed), transitionDetailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for close", zap.Error(err))
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":        "Change plan closed successfully",
		"change_plan_id": changePlanID.String(),
	})
}

// GetApprovalHistory handles GET /api/v1/change-plans/:id/approvals
// Retrieves approval history for a change plan
func (h *ChangePlanHandler) GetApprovalHistory(c echo.Context) error {
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

	// Get change plan to verify it exists and check authorization
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

	// Get approval history using the approval service
	approvals, err := h.approvalService.GetApprovalHistory(ctx, changePlanID)
	if err != nil {
		h.logger.Error("Failed to get approval history", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve approval history",
		})
	}

	// Convert to response format
	approvalResponses := make([]map[string]interface{}, len(approvals))
	for i, approval := range approvals {
		approvalResponse := map[string]interface{}{
			"id":             approval.ID.String(),
			"change_plan_id": approval.ChangePlanID.String(),
			"approver_id":    approval.ApproverID.String(),
			"status":         string(approval.Status),
			"comments":       approval.Comments,
			"created_at":     approval.CreatedAt.Format(time.RFC3339),
		}
		if approval.ApprovedAt != nil {
			approvalResponse["approved_at"] = approval.ApprovedAt.Format(time.RFC3339)
		}
		approvalResponses[i] = approvalResponse
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_plan_id": changePlanID.String(),
		"approvals":      approvalResponses,
		"count":          len(approvalResponses),
	})
}

// AssignReviewerRequest represents the request body for assigning a reviewer
type AssignReviewerRequest struct {
	ReviewerID uuid.UUID `json:"reviewer_id" validate:"required"`
}

// AssignReviewer handles PUT /api/v1/change-plans/:id/reviewer
// Assigns an optional reviewer (Project Approver) to a change plan
func (h *ChangePlanHandler) AssignReviewer(c echo.Context) error {
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

	// Check authorization: Only the author can assign reviewer
	userIDStr, _, _, ok := rbac.GetUserFromContext(ctx)
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

	if changePlan.AuthorID != userID {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "Only the author can assign a reviewer to their change plan",
		})
	}

	// Parse request body
	var req AssignReviewerRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		return middleware.HandleValidationError(c, err)
	}

	// Validate reviewer_id is provided
	ve := &middleware.ValidationError{}
	if req.ReviewerID == uuid.Nil {
		ve.AddField("reviewer_id", "reviewer_id is required", "required")
	}

	if len(ve.Fields) > 0 {
		return middleware.HandleValidationError(c, ve)
	}

	// Validate that reviewer is a Project Approver in the project
	reviewerMember, err := h.projectMemberRepo.GetMemberRole(ctx, changePlan.ProjectID, req.ReviewerID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Reviewer is not a member of this project",
			})
		}
		h.logger.Error("Failed to check reviewer membership", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to validate reviewer",
		})
	}

	// Check that reviewer has Project Approver role (or Project Owner, who also has approve permissions)
	projectRole := convertProjectRoleToRBAC(reviewerMember.Role)
	checker := rbac.NewCheckerService()
	if !checker.HasProjectPermission(projectRole, rbac.PermissionChangesApprove) {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Reviewer must have Project Approver role",
			"details": map[string]interface{}{
				"reviewer_role": reviewerMember.Role,
			},
		})
	}

	// Update assigned reviewer
	changePlan.AssignedReviewerID = &req.ReviewerID
	changePlan.UpdatedAt = time.Now()

	// Update in repository
	if err := h.changePlanRepo.Update(ctx, changePlan); err != nil {
		h.logger.Error("Failed to assign reviewer", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to assign reviewer",
		})
	}

	h.logger.Info("Reviewer assigned successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("reviewer_id", req.ReviewerID.String()),
		zap.String("author_id", userID.String()),
	)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":              "Reviewer assigned successfully",
		"change_plan_id":       changePlanID.String(),
		"reviewer_id":          req.ReviewerID.String(),
		"assigned_reviewer_id": req.ReviewerID.String(),
	})
}

// ExecuteChangePlan handles POST /api/v1/change-plans/:id/execute
// Executes a change plan, blocking if SQL Review errors exist (safety check)
func (h *ChangePlanHandler) ExecuteChangePlan(c echo.Context) error {
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

	// Get change plan (authorization is handled by middleware)
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

	// Get user ID for logging (already validated by middleware)
	userIDStr, _, _, ok := rbac.GetUserFromContext(ctx)
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

	// Validate status: Only Approved status can be executed
	if changePlan.Status != models.ChangePlanStatusApproved {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Only Approved change plans can be executed",
			"details": map[string]interface{}{
				"current_status": changePlan.Status,
			},
		})
	}

	// Safety check: Verify no SQL Review errors exist
	severityCounts, err := h.policyFindingRepo.CountBySeverity(ctx, changePlan.ID)
	if err != nil {
		h.logger.Warn("Failed to fetch policy findings", zap.Error(err))
		// Continue - treat as no errors if we can't fetch
		severityCounts = map[string]int{}
	}

	errorCount := severityCounts["Error"]
	if errorCount > 0 {
		// Fetch error findings to return in response
		findings, err := h.policyFindingRepo.GetByChangePlanID(ctx, changePlan.ID)
		if err != nil {
			h.logger.Warn("Failed to fetch policy findings", zap.Error(err))
			findings = []*models.PolicyFinding{}
		}

		// Filter only errors
		errorFindings := make([]models.PolicyFindingItem, 0)
		for _, finding := range findings {
			if finding.Severity == "Error" {
				errorFindings = append(errorFindings, models.PolicyFindingItem{
					ValidatorType: finding.ValidatorType,
					RuleName:      finding.RuleName,
					Message:       finding.Message,
					LineNumber:    finding.LineNumber,
					ColumnNumber:  finding.ColumnNumber,
				})
			}
		}

		h.logger.Warn("Change plan execution blocked due to SQL Review errors (safety check)",
			zap.String("change_plan_id", changePlan.ID.String()),
			zap.Int("error_count", errorCount),
		)

		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Cannot execute change plan with SQL Review errors",
			"details": map[string]interface{}{
				"error_count": errorCount,
				"errors":      errorFindings,
			},
		})
	}

	// Execute the change plan using ExecutionService
	// The service handles:
	// - Lock acquisition (database-level concurrency control)
	// - Status validation (must be Approved)
	// - Before snapshot capture
	// - SQL execution in transaction
	// - After snapshot capture on success
	// - Status updates (InProgress → Completed/Failed)
	// - Execution result capture
	// - Lock release (guaranteed via defer)
	result, err := h.executionService.Execute(ctx, changePlanID, userID)
	if err != nil {
		// Handle specific error types
		if err == services.ErrChangePlanNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Change plan not found",
			})
		}
		if err == services.ErrDatabaseLocked {
			// Try to get queue position if available
			queueStatus, queueErr := h.executionService.GetQueueStatus(ctx, changePlanID)
			if queueErr == nil && queueStatus != nil {
				return c.JSON(http.StatusConflict, map[string]interface{}{
					"error":          "Database is currently locked by another execution",
					"queued":         true,
					"queue_position": queueStatus.Position,
					"queue_status":   queueStatus.Status,
				})
			}
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": "Database is currently locked by another execution",
			})
		}
		h.logger.Error("Failed to execute change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to execute change plan: %v", err),
		})
	}

	// Get updated change plan to return in response
	updatedChangePlan, err := h.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		h.logger.Error("Failed to get updated change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve execution results",
		})
	}

	h.logger.Info("Change plan execution completed",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("executed_by", userID.String()),
		zap.Bool("success", result.Success),
		zap.Int("rows_affected", result.RowsAffected),
		zap.Duration("duration", result.Duration),
	)

	// Log audit entry for execution
	if h.auditService != nil {
		executionDetailsJSON, _ := json.Marshal(map[string]interface{}{
			"change_plan_id": changePlanID.String(),
			"title":          changePlan.Title,
			"success":        result.Success,
			"rows_affected":  result.RowsAffected,
			"duration_ms":    result.Duration.Milliseconds(),
		})
		if err := h.auditService.LogWithHTTP(ctx, userID, services.ActionExecuteChangePlan, services.ResourceTypeChangePlan, changePlanID, c.Request(), executionDetailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for execution", zap.Error(err))
		}

		// Log status transition from Approved to final status (Completed or Failed)
		finalStatus := models.ChangePlanStatusCompleted
		if !result.Success {
			finalStatus = models.ChangePlanStatusFailed
		}
		transitionDetailsJSON, _ := json.Marshal(map[string]interface{}{
			"change_plan_id": changePlanID.String(),
			"title":          changePlan.Title,
			"success":        result.Success,
		})
		if err := h.auditService.LogStateTransition(ctx, userID, services.ResourceTypeChangePlan, changePlanID, string(models.ChangePlanStatusApproved), string(finalStatus), transitionDetailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for execution status transition", zap.Error(err))
		}
	}

	// Build response with execution results
	executionData := map[string]interface{}{
		"success":       result.Success,
		"rows_affected": result.RowsAffected,
		"duration_ms":   result.Duration.Milliseconds(),
	}

	if result.ErrorMessage != nil {
		executionData["error_message"] = *result.ErrorMessage
	}

	response := map[string]interface{}{
		"change_plan": updatedChangePlan.ToResponse(),
		"execution":   executionData,
	}

	return c.JSON(http.StatusOK, response)
}

// GetQueueStatus handles GET /api/v1/change-plans/:id/queue-status
// Returns the queue status and position for a Change Plan
func (h *ChangePlanHandler) GetQueueStatus(c echo.Context) error {
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

	// Get change plan to verify it exists
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

	// Get queue status
	queueStatus, err := h.executionService.GetQueueStatus(ctx, changePlanID)
	if err != nil {
		// Queue item not found - Change Plan is not queued
		return c.JSON(http.StatusOK, map[string]interface{}{
			"change_plan_id": changePlanID.String(),
			"queued":         false,
			"message":        "Change plan is not in the execution queue",
		})
	}

	// Calculate position
	position := queueStatus.Position

	response := map[string]interface{}{
		"change_plan_id": changePlanID.String(),
		"database_id":    changePlan.TargetDatabaseID.String(),
		"queued":         true,
		"status":         queueStatus.Status,
		"position":       position,
		"priority":       queueStatus.Priority,
		"queued_at":      queueStatus.QueuedAt,
	}

	if queueStatus.StartedAt != nil {
		response["started_at"] = queueStatus.StartedAt
	}

	if queueStatus.CompletedAt != nil {
		response["completed_at"] = queueStatus.CompletedAt
	}

	if queueStatus.ErrorMessage != nil {
		response["error_message"] = *queueStatus.ErrorMessage
	}

	return c.JSON(http.StatusOK, response)
}

// GetExecutionLogHistory handles GET /api/v1/change-plans/:id/execution-logs
// Returns the execution log history for a Change Plan
func (h *ChangePlanHandler) GetExecutionLogHistory(c echo.Context) error {
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

	// Verify change plan exists
	_, err = h.changePlanRepo.GetByID(ctx, changePlanID)
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

	// Get execution logs
	logs, err := h.executionService.GetExecutionLogHistory(ctx, changePlanID)
	if err != nil {
		h.logger.Error("Failed to get execution logs", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve execution logs",
		})
	}

	// Convert to response format
	logResponses := make([]*models.ExecutionLogResponse, len(logs))
	for i, log := range logs {
		logResponses[i] = log.ToResponse()
	}

	response := map[string]interface{}{
		"change_plan_id": changePlanID.String(),
		"total":          len(logResponses),
		"logs":           logResponses,
	}

	return c.JSON(http.StatusOK, response)
}

// ResubmitChangePlan handles POST /api/v1/change-plans/:id/resubmit
// Allows authors to move Open Change Plans back to Draft for fixing errors
func (h *ChangePlanHandler) ResubmitChangePlan(c echo.Context) error {
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

	// Check authorization: Only the author can resubmit
	userIDStr, _, _, ok := rbac.GetUserFromContext(ctx)
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

	if changePlan.AuthorID != userID {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "Only the author can resubmit their change plan",
		})
	}

	// Validate status: Only Open status can be resubmitted
	if changePlan.Status != models.ChangePlanStatusOpen {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Only Open change plans can be resubmitted",
			"details": map[string]interface{}{
				"current_status": changePlan.Status,
			},
		})
	}

	// Validate transition
	if err := workflow.ValidateTransition(changePlan.Status, models.ChangePlanStatusDraft); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Clear old policy findings
	if err := h.policyFindingRepo.DeleteByChangePlanID(ctx, changePlan.ID); err != nil {
		h.logger.Warn("Failed to delete policy findings during resubmission", zap.Error(err))
		// Continue - non-critical error
	}

	// Update status to Draft
	changePlan.Status = models.ChangePlanStatusDraft
	if err := h.changePlanRepo.Update(ctx, changePlan); err != nil {
		h.logger.Error("Failed to resubmit change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to resubmit change plan",
		})
	}

	h.logger.Info("Change plan resubmitted successfully",
		zap.String("change_plan_id", changePlan.ID.String()),
		zap.String("author_id", userID.String()),
	)

	return c.JSON(http.StatusOK, changePlan.ToResponse())
}

// trackChangeHistory tracks field-level changes between old and new change plan
func (h *ChangePlanHandler) trackChangeHistory(
	ctx context.Context,
	oldChangePlan, newChangePlan *models.ChangePlan,
	changedByID uuid.UUID,
) {
	if h.changeHistoryRepo == nil {
		return
	}

	// Helper function to convert value to string
	valueToString := func(v interface{}) *string {
		if v == nil {
			return nil
		}
		switch val := v.(type) {
		case string:
			if val == "" {
				return nil
			}
			return &val
		case *string:
			return val
		case models.ChangePlanStatus:
			s := string(val)
			return &s
		case models.RiskLevel:
			s := string(val)
			return &s
		case models.ChangeType:
			s := string(val)
			return &s
		case *uuid.UUID:
			if val == nil {
				return nil
			}
			s := val.String()
			return &s
		case uuid.UUID:
			s := val.String()
			return &s
		case *time.Time:
			if val == nil {
				return nil
			}
			s := val.Format(time.RFC3339)
			return &s
		case time.Time:
			if val.IsZero() {
				return nil
			}
			s := val.Format(time.RFC3339)
			return &s
		case *int:
			if val == nil {
				return nil
			}
			s := fmt.Sprintf("%d", *val)
			return &s
		case int:
			s := fmt.Sprintf("%d", val)
			return &s
		default:
			s := fmt.Sprintf("%v", val)
			return &s
		}
	}

	// Track changes for each field
	changes := []struct {
		fieldName string
		oldValue  interface{}
		newValue  interface{}
	}{
		{"title", oldChangePlan.Title, newChangePlan.Title},
		{"description", oldChangePlan.Description, newChangePlan.Description},
		{"sql_content", oldChangePlan.SQLContent, newChangePlan.SQLContent},
		{"rollback_sql", oldChangePlan.RollbackSQL, newChangePlan.RollbackSQL},
		{"status", oldChangePlan.Status, newChangePlan.Status},
		{"risk_level", oldChangePlan.RiskLevel, newChangePlan.RiskLevel},
		{"assigned_reviewer_id", oldChangePlan.AssignedReviewerID, newChangePlan.AssignedReviewerID},
		{"scheduled_for", oldChangePlan.ScheduledFor, newChangePlan.ScheduledFor},
	}

	for _, change := range changes {
		oldStr := valueToString(change.oldValue)
		newStr := valueToString(change.newValue)

		// Skip if values are the same
		if oldStr == nil && newStr == nil {
			continue
		}
		if oldStr != nil && newStr != nil && *oldStr == *newStr {
			continue
		}

		// Create change history entry
		changeHistory := &models.ChangeHistory{
			ID:           uuid.New(),
			ChangePlanID: newChangePlan.ID,
			FieldName:    change.fieldName,
			OldValue:     oldStr,
			NewValue:     newStr,
			ChangedByID:  changedByID,
			ChangedAt:    time.Now(),
		}

		if err := h.changeHistoryRepo.Create(ctx, changeHistory); err != nil {
			h.logger.Error("Failed to create change history entry",
				zap.Error(err),
				zap.String("change_plan_id", newChangePlan.ID.String()),
				zap.String("field_name", change.fieldName),
			)
			// Continue tracking other fields even if one fails
		}
	}
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
