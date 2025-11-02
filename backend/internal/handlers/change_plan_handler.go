package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
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
	projectRepo          repository.ProjectRepository
	databaseRepo         repository.DatabaseRepository
	databaseInstanceRepo repository.DatabaseInstanceRepository
	projectMemberRepo    repository.ProjectMemberRepository
	policyFindingRepo    repository.PolicyFindingRepository
	riskAssessmentSvc    *services.RiskAssessmentService
	reviewService        *review.ReviewService
	logger               *zap.Logger
}

// NewChangePlanHandler creates a new change plan handler
func NewChangePlanHandler(
	changePlanRepo repository.ChangePlanRepository,
	projectRepo repository.ProjectRepository,
	databaseRepo repository.DatabaseRepository,
	databaseInstanceRepo repository.DatabaseInstanceRepository,
	projectMemberRepo repository.ProjectMemberRepository,
	policyFindingRepo repository.PolicyFindingRepository,
	riskAssessmentSvc *services.RiskAssessmentService,
	reviewService *review.ReviewService,
	logger *zap.Logger,
) *ChangePlanHandler {
	return &ChangePlanHandler{
		changePlanRepo:       changePlanRepo,
		projectRepo:          projectRepo,
		databaseRepo:         databaseRepo,
		databaseInstanceRepo: databaseInstanceRepo,
		projectMemberRepo:    projectMemberRepo,
		policyFindingRepo:    policyFindingRepo,
		riskAssessmentSvc:    riskAssessmentSvc,
		reviewService:        reviewService,
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

		// Re-assess risk level when SQL content changes
		if h.riskAssessmentSvc != nil {
			riskLevel, err := h.riskAssessmentSvc.AssessRisk(ctx, *req.SQLContent, changePlan.ChangeType, changePlan.TargetDatabaseID)
			if err != nil {
				h.logger.Warn("Failed to reassess risk level after SQL content change", zap.Error(err))
				// Continue with existing risk level
			} else {
				changePlan.RiskLevel = riskLevel
			}
		}
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
func (h *ChangePlanHandler) storeReviewFindings(ctx context.Context, changePlanID uuid.UUID, reviewResult *review.ReviewResultWithSummary) error {
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

	return c.NoContent(http.StatusNoContent)
}

// enrichWithPolicyFindings enriches ChangePlanResponse with policy findings
func (h *ChangePlanHandler) enrichWithPolicyFindings(response *models.ChangePlanResponse, findings []*models.PolicyFinding) *models.ChangePlanResponse {
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

	// Check authorization: User must have approval permissions
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

	// Organization admins can approve any change plan
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

		// Check if user has PermissionChangesApprove
		projectRole := convertProjectRoleToRBAC(member.Role)
		checker := rbac.NewCheckerService()
		if !checker.HasProjectPermission(projectRole, rbac.PermissionChangesApprove) {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "Insufficient permissions to approve change plans",
			})
		}
	}

	// Validate status: Only Open status can be approved
	if changePlan.Status != models.ChangePlanStatusOpen {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Only Open change plans can be approved",
			"details": map[string]interface{}{
				"current_status": changePlan.Status,
			},
		})
	}

	// Check for SQL Review errors
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

		h.logger.Info("Change plan approval blocked due to SQL Review errors",
			zap.String("change_plan_id", changePlan.ID.String()),
			zap.Int("error_count", errorCount),
		)

		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Cannot approve change plan with SQL Review errors",
			"details": map[string]interface{}{
				"error_count": errorCount,
				"errors":      errorFindings,
			},
		})
	}

	// Validate transition
	if err := workflow.ValidateTransition(changePlan.Status, models.ChangePlanStatusApproved); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Update status to Approved
	changePlan.Status = models.ChangePlanStatusApproved
	if err := h.changePlanRepo.Update(ctx, changePlan); err != nil {
		h.logger.Error("Failed to approve change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to approve change plan",
		})
	}

	h.logger.Info("Change plan approved successfully",
		zap.String("change_plan_id", changePlan.ID.String()),
		zap.String("approved_by", userID.String()),
	)

	return c.JSON(http.StatusOK, changePlan.ToResponse())
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

	// Check authorization: User must have execution permissions
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

	// Organization admins can execute any change plan
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

		// Check if user has PermissionChangesExecute (or approve permission)
		projectRole := convertProjectRoleToRBAC(member.Role)
		checker := rbac.NewCheckerService()
		if !checker.HasProjectPermission(projectRole, rbac.PermissionChangesExecute) &&
			!checker.HasProjectPermission(projectRole, rbac.PermissionChangesApprove) {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "Insufficient permissions to execute change plans",
			})
		}
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

	// Validate transition
	if err := workflow.ValidateTransition(changePlan.Status, models.ChangePlanStatusInProgress); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Update status to InProgress
	changePlan.Status = models.ChangePlanStatusInProgress
	if err := h.changePlanRepo.Update(ctx, changePlan); err != nil {
		h.logger.Error("Failed to execute change plan", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to execute change plan",
		})
	}

	h.logger.Info("Change plan execution started",
		zap.String("change_plan_id", changePlan.ID.String()),
		zap.String("executed_by", userID.String()),
	)

	return c.JSON(http.StatusOK, changePlan.ToResponse())
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
