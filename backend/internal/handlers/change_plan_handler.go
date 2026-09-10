package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/middleware"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/parser"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/review"
	"github.com/nsxbet/deeb/internal/services"
	"github.com/nsxbet/deeb/internal/workflow"
	"github.com/nsxbet/sql-reviewer/pkg/types"
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
	environmentRepo      repository.EnvironmentRepository
	statementResultRepo  repository.StatementExecutionResultRepository
	riskAssessmentSvc    *services.RiskAssessmentService
	reviewService        *review.ReviewService
	approvalService      *services.ApprovalService
	executionService     *services.ExecutionService
	connectionManager    *database.ConnectionManager
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
	environmentRepo repository.EnvironmentRepository,
	statementResultRepo repository.StatementExecutionResultRepository,
	riskAssessmentSvc *services.RiskAssessmentService,
	reviewService *review.ReviewService,
	approvalService *services.ApprovalService,
	executionService *services.ExecutionService,
	connectionManager *database.ConnectionManager,
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
		environmentRepo:      environmentRepo,
		statementResultRepo:  statementResultRepo,
		riskAssessmentSvc:    riskAssessmentSvc,
		reviewService:        reviewService,
		approvalService:      approvalService,
		executionService:     executionService,
		connectionManager:    connectionManager,
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
	if database.ProjectID == nil || *database.ProjectID != req.ProjectID {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Database does not belong to the specified project or is unassigned",
		})
	}

	// Get database instance to determine dialect
	databaseInstance, err := h.databaseInstanceRepo.GetByID(ctx, database.DatabaseInstanceID)
	if err != nil {
		h.logger.Error("Failed to get database instance", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get database instance",
		})
	}

	// Parse SQL to count statements and validate
	dialect := h.getParserDialect(databaseInstance.DatabaseEngine)
	parseResult, err := parser.SplitSQL(req.SQLContent, dialect)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": fmt.Sprintf("Invalid SQL: %s", err.Error()),
		})
	}

	if parseResult.TotalCount > 1000 {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Too many statements (max 1000)",
			"details": map[string]interface{}{
				"statement_count": parseResult.TotalCount,
				"max_statements":  1000,
			},
		})
	}

	// RBAC check is done in middleware (RequireProjectPermission)

	// Validate ghost executor selection (DDL + non-production environment).
	executionBackend, err := h.validateGhostSelection(ctx, req.ExecutionBackend, req.ChangeType, database)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Create change plan with Draft status
	statementCount := parseResult.TotalCount
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
		StatementCount:   &statementCount,
		ExecutionBackend: executionBackend,
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

	h.logger.Info(
		"Change plan created successfully",
		zap.String("change_plan_id", changePlan.ID.String()),
		zap.String("project_id", changePlan.ProjectID.String()),
	)

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

// validateGhostSelection enforces ghost executor constraints at the API
// boundary: only DDL plans on databases in non-production environments may
// pick execution_backend=ghost. Empty backend defaults to direct.
func (h *ChangePlanHandler) validateGhostSelection(
	ctx context.Context,
	backend models.ExecutionBackend,
	changeType models.ChangeType,
	db *models.Database,
) (models.ExecutionBackend, error) {
	if backend == "" || backend == models.ExecutionBackendDirect {
		return models.ExecutionBackendDirect, nil
	}
	if backend != models.ExecutionBackendGhost {
		return "", fmt.Errorf("invalid execution_backend: must be direct or ghost")
	}
	if changeType != models.ChangeTypeDDL {
		return "", fmt.Errorf("execution_backend=ghost is only supported for DDL change plans")
	}
	if db.EnvironmentID == nil {
		return "", fmt.Errorf("execution_backend=ghost requires the database to be assigned to an environment")
	}
	env, err := h.environmentRepo.GetByID(ctx, *db.EnvironmentID)
	if err != nil {
		return "", fmt.Errorf("failed to validate environment: %w", err)
	}
	if env.IsProduction {
		return "", fmt.Errorf("execution_backend=ghost is not permitted on production environments")
	}

	// Engine gate: pg-online-schema-change is not implemented yet, so ghost
	// is MySQL-only for now.
	dbInstance, err := h.databaseInstanceRepo.GetByID(ctx, db.DatabaseInstanceID)
	if err != nil {
		return "", fmt.Errorf("failed to validate database instance: %w", err)
	}
	if dbInstance.DatabaseEngine != models.DatabaseEngineMySQL {
		return "", fmt.Errorf("execution_backend=ghost is currently supported only for MySQL databases")
	}

	// Binlog gate: gh-ost requires ROW binlog format with FULL row image.
	// Verified live against the target so misconfigured RDS parameter groups
	// fail fast at plan creation instead of at execution time.
	if err := h.checkGhostBinlogRequirements(ctx, dbInstance, db); err != nil {
		return "", err
	}

	return models.ExecutionBackendGhost, nil
}

// checkGhostBinlogRequirements verifies the target MySQL server has
// binlog_format=ROW and binlog_row_image=FULL, which gh-ost requires.
func (h *ChangePlanHandler) checkGhostBinlogRequirements(
	ctx context.Context,
	dbInstance *models.DatabaseInstance,
	db *models.Database,
) error {
	if h.connectionManager == nil {
		return fmt.Errorf("failed to connect to target database for ghost validation: connection manager not configured")
	}
	conn, err := h.connectionManager.Connect(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to target database for ghost validation: %w", err)
	}
	defer func() { _ = conn.Close() }()

	var name, value string
	if err := conn.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'binlog_format'").Scan(&name, &value); err != nil || !strings.EqualFold(value, "ROW") {
		return fmt.Errorf("ghost requires binlog_format=ROW (current: %s); check the RDS parameter group", value)
	}
	value = ""
	if err := conn.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'binlog_row_image'").Scan(&name, &value); err != nil || !strings.EqualFold(value, "FULL") {
		return fmt.Errorf("ghost requires binlog_row_image=FULL (current: %s); check the RDS parameter group", value)
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

	// RBAC check is done in middleware (RequireProjectPermission)

	h.logger.Info(
		"Change plan retrieved successfully",
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
	// Always initialize the slice to ensure consistent JSON serialization ([] instead of null)
	if len(executionLogs) > 0 {
		logResponses := make([]*models.ExecutionLogResponse, len(executionLogs))
		for i, log := range executionLogs {
			logResponses[i] = log.ToResponse()
		}
		response.ExecutionLogs = logResponses

		// Set last execution status from the most recent log (first in the list)
		lastStatus := executionLogs[0].Status
		response.LastExecutionStatus = &lastStatus
	} else {
		// Initialize empty slice to serialize as [] instead of null
		response.ExecutionLogs = []*models.ExecutionLogResponse{}
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

	// Optional case-insensitive search over title and description
	filters.Search = c.QueryParam("search")

	// Optional change type filter
	if changeTypeStr := c.QueryParam("change_type"); changeTypeStr != "" {
		filters.ChangeType = models.ChangeType(changeTypeStr)
	}

	// Optional created_at range filters (RFC3339)
	createdAfter, err := parseTimeParam(c, "created_after")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid created_after (must be RFC3339)",
		})
	}
	createdBefore, err := parseTimeParam(c, "created_before")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid created_before (must be RFC3339)",
		})
	}
	filters.CreatedAfter = createdAfter
	filters.CreatedBefore = createdBefore

	// Parse status filter (multi-select: comma-separated list of statuses).
	if statusStr := c.QueryParam("status"); statusStr != "" {
		for _, raw := range strings.Split(statusStr, ",") {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			if !models.ChangePlanStatus(raw).IsValid() {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Invalid status value",
				})
			}
		}
		filters.Status = models.ChangePlanStatus(statusStr)
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
		// Get all projects the user is a member of
		userProjectIDs, err := h.projectMemberRepo.GetUserProjectIDs(ctx, userID)
		if err != nil {
			h.logger.Error("Failed to get user project IDs", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to retrieve user projects",
			})
		}

		// If user provided a specific project_id filter, validate they have access to it
		if filters.ProjectID != uuid.Nil {
			hasAccess := false
			for _, projectID := range userProjectIDs {
				if projectID == filters.ProjectID {
					hasAccess = true
					break
				}
			}
			if !hasAccess {
				// User doesn't have access to the requested project, return empty results
				return c.JSON(http.StatusOK, map[string]interface{}{
					"change_plans": []*models.ChangePlanResponse{},
					"total":        0,
					"limit":        limit,
					"offset":       offset,
					"count":        0,
				})
			}
			// User has access, keep the ProjectID filter as-is
		} else {
			// No specific project requested, filter by all user's projects
			filters.ProjectIDs = userProjectIDs
		}

		// If user has no projects, return empty results
		if len(userProjectIDs) == 0 && filters.ProjectID == uuid.Nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"change_plans": []*models.ChangePlanResponse{},
				"total":        0,
				"limit":        limit,
				"offset":       offset,
				"count":        0,
			})
		}
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

	// RBAC check is done in middleware (RequireProjectPermission)
	// Additional business rule: Only author can update their change plans (except org admins)
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

	// Business rule: Only the author can update (organization admins can override)
	if organizationRole != rbac.OrganizationAdmin && changePlan.AuthorID != userID {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "Only the author can update this change plan",
		})
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

	// Execution backend switch: validated against DDL type and the target
	// database's environment (ghost only allowed on non-production).
	if req.ExecutionBackend != nil {
		database, err := h.databaseRepo.GetByID(ctx, changePlan.TargetDatabaseID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Failed to validate target database",
			})
		}
		backend, err := h.validateGhostSelection(ctx, *req.ExecutionBackend, changePlan.ChangeType, database)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
			})
		}
		changePlan.ExecutionBackend = backend
	}

	if req.Description != nil {
		changePlan.Description = *req.Description
	}

	if req.SQLContent != nil {
		changePlan.SQLContent = *req.SQLContent

		// Re-parse SQL to update statement count
		db, err := h.databaseRepo.GetByID(ctx, changePlan.TargetDatabaseID)
		if err != nil {
			h.logger.Warn("Failed to get database for statement count update", zap.Error(err))
			// Continue without updating statement count
		} else {
			databaseInstance, err := h.databaseInstanceRepo.GetByID(ctx, db.DatabaseInstanceID)
			if err != nil {
				h.logger.Warn("Failed to get database instance for statement count update", zap.Error(err))
				// Continue without updating statement count
			} else {
				dialect := h.getParserDialect(databaseInstance.DatabaseEngine)
				parseResult, err := parser.SplitSQL(*req.SQLContent, dialect)
				if err != nil {
					// Return error for invalid SQL
					return c.JSON(http.StatusBadRequest, map[string]interface{}{
						"error": fmt.Sprintf("Invalid SQL: %s", err.Error()),
					})
				}

				if parseResult.TotalCount > 1000 {
					return c.JSON(http.StatusBadRequest, map[string]interface{}{
						"error": "Too many statements (max 1000)",
						"details": map[string]interface{}{
							"statement_count": parseResult.TotalCount,
							"max_statements":  1000,
						},
					})
				}

				statementCount := parseResult.TotalCount
				changePlan.StatementCount = &statementCount
			}
		}

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
			h.logger.Info(
				"Status transitioned to Open due to SQL content update",
				zap.String("change_plan_id", changePlan.ID.String()),
				zap.String("previous_status", string(previousStatus)),
			)
		}

		// Delete existing approval records when SQL content is updated
		// This allows approvers to approve the updated version
		// Delete approvals whether status transitions to Open or is already Open
		if changePlan.Status == models.ChangePlanStatusOpen {
			if err := h.approvalRepo.DeleteByChangePlanID(ctx, changePlan.ID); err != nil {
				h.logger.Warn(
					"Failed to delete existing approvals after SQL content update",
					zap.String("change_plan_id", changePlan.ID.String()),
					zap.Error(err),
				)
				// Don't block the update - continue even if approval deletion fails
			} else {
				h.logger.Info(
					"Deleted existing approvals after SQL content update",
					zap.String("change_plan_id", changePlan.ID.String()),
				)
			}
		}
	}

	if req.RollbackSQL != nil {
		changePlan.RollbackSQL = req.RollbackSQL
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
							"message": reviewErr.Message,
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

	h.logger.Info(
		"Change plan updated successfully",
		zap.String("change_plan_id", changePlan.ID.String()),
	)

	return c.JSON(http.StatusOK, changePlan.ToResponse())
}

// SQLReviewError represents an error during SQL Review
type SQLReviewError struct {
	HasErrors bool
	Message   string
}

func (e *SQLReviewError) Error() string {
	return e.Message
}

// triggerSQLReview triggers SQL Review for a change plan and stores findings
func (h *ChangePlanHandler) triggerSQLReview(c echo.Context, changePlan *models.ChangePlan) error {
	// Check if review service is configured
	if h.reviewService == nil {
		h.logger.Warn("ReviewService not configured, skipping SQL Review")
		return nil
	}

	ctx := c.Request().Context()

	// Get database type from database instance
	db, err := h.databaseRepo.GetByID(ctx, changePlan.TargetDatabaseID)
	if err != nil {
		return &SQLReviewError{
			HasErrors: false,
			Message:   "Failed to get database: " + err.Error(),
		}
	}

	databaseInstance, err := h.databaseInstanceRepo.GetByID(ctx, db.DatabaseInstanceID)
	if err != nil {
		return &SQLReviewError{
			HasErrors: false,
			Message:   "Failed to get database instance: " + err.Error(),
		}
	}

	h.logger.Info("Running SQL Review")

	// Attempt to establish database connection for advanced validation (DML dry-run, etc.)
	// Connection is optional - review will work without it but with limited capabilities
	var dbConn *sql.DB
	if h.connectionManager != nil {
		conn, connErr := h.connectionManager.Connect(ctx, database.ConnectionParams{
			DatabaseInstance: databaseInstance,
			Database:         db,
			MaxOpenConns:     2, // Limit connections for review
			MaxIdleConns:     1,
			ConnMaxLifetime:  time.Minute * 2,
		})
		if connErr != nil {
			h.logger.Warn("Failed to establish database connection for review - continuing without connection validation",
				zap.Error(connErr),
				zap.String("database_id", db.ID.String()))
		} else {
			dbConn = conn.DB
			defer func() {
				if closeErr := conn.Close(); closeErr != nil {
					h.logger.Warn("Failed to close database connection after review",
						zap.Error(closeErr))
				}
			}()
		}
	}

	// Determine change type based on SQL content
	changeType := types.PlanCheckRunConfig_SDL // Default to SDL (schema definition language) for comprehensive checking

	// Determine environment ID for review (use zero UUID if unassigned)
	var envID uuid.UUID
	if db.EnvironmentID != nil {
		envID = *db.EnvironmentID
	}

	reviewResult, err := h.reviewService.Review(ctx, review.ReviewOptions{
		ChangePlanID:             changePlan.ID,
		EnvironmentID:            envID,
		DatabaseID:               &db.ID,
		SQLContent:               changePlan.SQLContent,
		DatabaseEngine:           databaseInstance.DatabaseEngine,
		Driver:                   dbConn,
		ChangeType:               &changeType,
		EnableQueryLogging:       true,
		UsePostgresDatabaseOwner: databaseInstance.DatabaseEngine == models.DatabaseEnginePostgreSQL,
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
			HasErrors: reviewResult.ErrorCount > 0,
			Message:   "Failed to store review findings: " + err.Error(),
		}
	}

	// Check if there are errors that should prevent status transition
	if reviewResult.ErrorCount > 0 {
		return &SQLReviewError{
			HasErrors: true,
			Message:   "SQL Review found errors that prevent status transition",
		}
	}

	h.logger.Info(
		"SQL Review completed successfully",
		zap.String("change_plan_id", changePlan.ID.String()),
		zap.Int("errors", reviewResult.ErrorCount),
		zap.Int("warnings", reviewResult.WarningCount),
	)

	return nil
}

// storeReviewFindings converts V2 review results to policy findings and stores them
func (h *ChangePlanHandler) storeReviewFindings(
	ctx context.Context,
	changePlanID uuid.UUID,
	reviewResult *review.ReviewResult,
) error {
	findings := make([]*models.PolicyFinding, 0)

	// Process violations (errors)
	for _, violation := range reviewResult.Violations {
		finding := &models.PolicyFinding{
			ChangePlanID:  changePlanID,
			ValidatorType: h.mapRuleTypeToValidatorType(violation.RuleType, changePlanID),
			Severity:      models.SeverityError,
			RuleName:      violation.Title,
			Message:       violation.Message,
		}

		if violation.Line > 0 {
			lineNum := int(violation.Line)
			finding.LineNumber = &lineNum
		}
		if violation.Column > 0 {
			colNum := int(violation.Column)
			finding.ColumnNumber = &colNum
		}

		findings = append(findings, finding)
	}

	// Process warnings
	for _, warning := range reviewResult.Warnings {
		finding := &models.PolicyFinding{
			ChangePlanID:  changePlanID,
			ValidatorType: h.mapRuleTypeToValidatorType(warning.RuleType, changePlanID),
			Severity:      models.SeverityWarning,
			RuleName:      warning.Title,
			Message:       warning.Message,
		}

		if warning.Line > 0 {
			lineNum := int(warning.Line)
			finding.LineNumber = &lineNum
		}
		if warning.Column > 0 {
			colNum := int(warning.Column)
			finding.ColumnNumber = &colNum
		}

		findings = append(findings, finding)
	}

	// Batch insert findings
	if len(findings) > 0 {
		if err := h.policyFindingRepo.CreateBatch(ctx, findings); err != nil {
			return err
		}
	}

	return nil
}

// mapRuleTypeToValidatorType maps V2 rule type to validator type string
// Rule types are in format "category.rule-name" (e.g., "column.required-column")
func (h *ChangePlanHandler) mapRuleTypeToValidatorType(ruleType string, changePlanID uuid.UUID) string {
	// Extract category from rule type (before the dot)
	var category string
	for i, ch := range ruleType {
		if ch == '.' {
			category = ruleType[:i]
			break
		}
	}

	// Default if no dot found
	if category == "" {
		category = ruleType
	}

	// Check for unknown rule type and log it
	if ruleType == "unknown" {
		h.logger.Warn("Unclassified rule type detected",
			zap.String("rule_type", ruleType),
			zap.String("change_plan_id", changePlanID.String()))
		return "best_practice_violation"
	}

	// Map category to validator type
	switch category {
	case "engine":
		return "syntax_error"
	case "naming":
		return "best_practice_violation"
	case "table", "column", "index":
		return "best_practice_violation"
	case "schema":
		return "backwards_incompatible"
	case "statement":
		return "security_risk"
	case "unknown":
		// Unclassified rule - log warning
		h.logger.Warn("Unclassified rule category detected",
			zap.String("rule_type", ruleType),
			zap.String("category", category),
			zap.String("change_plan_id", changePlanID.String()))
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

	// RBAC check is done in middleware (RequireProjectPermission)

	// Trigger SQL Review (without blocking on errors - manual review always returns results)
	if h.reviewService == nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "SQL Review service not configured",
		})
	}

	// Get database type from database instance
	db, err := h.databaseRepo.GetByID(ctx, changePlan.TargetDatabaseID)
	if err != nil {
		h.logger.Error("Failed to get database", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get database",
		})
	}

	databaseInstance, err := h.databaseInstanceRepo.GetByID(ctx, db.DatabaseInstanceID)
	if err != nil {
		h.logger.Error("Failed to get database instance", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get database instance",
		})
	}

	h.logger.Info("Running manual SQL Review")

	// Attempt to establish database connection for advanced validation (DML dry-run, etc.)
	// Connection is optional - review will work without it but with limited capabilities
	var dbConn *sql.DB
	if h.connectionManager != nil {
		conn, connErr := h.connectionManager.Connect(ctx, database.ConnectionParams{
			DatabaseInstance: databaseInstance,
			Database:         db,
			MaxOpenConns:     2, // Limit connections for review
			MaxIdleConns:     1,
			ConnMaxLifetime:  time.Minute * 2,
		})
		if connErr != nil {
			h.logger.Warn("Failed to establish database connection for review - continuing without connection validation",
				zap.Error(connErr),
				zap.String("database_id", db.ID.String()))
		} else {
			dbConn = conn.DB
			defer func() {
				if closeErr := conn.Close(); closeErr != nil {
					h.logger.Warn("Failed to close database connection after review",
						zap.Error(closeErr))
				}
			}()
		}
	}

	// Determine change type based on SQL content
	changeType := types.PlanCheckRunConfig_SDL // Default to SDL (schema definition language) for comprehensive checking

	// Determine environment ID for review (use zero UUID if unassigned)
	var envID uuid.UUID
	if db.EnvironmentID != nil {
		envID = *db.EnvironmentID
	}

	reviewResult, err := h.reviewService.Review(ctx, review.ReviewOptions{
		ChangePlanID:             changePlan.ID,
		EnvironmentID:            envID,
		DatabaseID:               &db.ID,
		SQLContent:               changePlan.SQLContent,
		DatabaseEngine:           databaseInstance.DatabaseEngine,
		Driver:                   dbConn,
		ChangeType:               &changeType,
		EnableQueryLogging:       true,
		UsePostgresDatabaseOwner: databaseInstance.DatabaseEngine == models.DatabaseEnginePostgreSQL,
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

	h.logger.Info(
		"SQL Review triggered manually",
		zap.String("change_plan_id", changePlan.ID.String()),
		zap.Int("errors", reviewResult.ErrorCount),
		zap.Int("warnings", reviewResult.WarningCount),
	)

	// Return review results in response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_plan_id": changePlan.ID,
		"review_result": map[string]interface{}{
			"valid":            reviewResult.Valid,
			"error_count":      reviewResult.ErrorCount,
			"warning_count":    reviewResult.WarningCount,
			"total_violations": reviewResult.TotalViolations,
			"violations":       reviewResult.Violations,
			"warnings":         reviewResult.Warnings,
			"policy_id":        reviewResult.PolicyID,
			"policy_name":      reviewResult.PolicyName,
			"enabled_rules":    reviewResult.EnabledRules,
			"skipped_rules":    reviewResult.SkippedRules,
			"executed_rules":   reviewResult.ExecutedRules,
		},
	})
}

// GetReviewFindings handles GET /api/v1/change-plans/:id/review/findings
// Retrieves stored policy findings for a change plan with optional filtering
func (h *ChangePlanHandler) GetReviewFindings(c echo.Context) error {
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

	// RBAC check is done in middleware (RequireProjectPermission)

	// Get findings from repository
	findings, err := h.policyFindingRepo.GetByChangePlanID(ctx, changePlanID)
	if err != nil {
		h.logger.Error("Failed to get findings", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve findings",
		})
	}

	// Apply filters from query parameters
	severity := c.QueryParam("severity")            // "error", "warning", "info"
	ruleName := c.QueryParam("rule_name")           // specific rule name
	validatorType := c.QueryParam("validator_type") // "syntax", "security", etc.

	var filteredFindings []*models.PolicyFinding
	for _, finding := range findings {
		// Apply severity filter
		if severity != "" && string(finding.Severity) != severity {
			continue
		}

		// Apply rule name filter
		if ruleName != "" && finding.RuleName != ruleName {
			continue
		}

		// Apply validator type filter
		if validatorType != "" && finding.ValidatorType != validatorType {
			continue
		}

		filteredFindings = append(filteredFindings, finding)
	}

	// Group findings by severity for summary
	summary := map[string]int{
		"total":   len(filteredFindings),
		"error":   0,
		"warning": 0,
		"info":    0,
	}
	for _, finding := range filteredFindings {
		summary[string(finding.Severity)]++
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_plan_id": changePlan.ID,
		"findings":       filteredFindings,
		"summary":        summary,
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
		projectRole := rbac.ProjectRole(member.Role)
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

	h.logger.Info(
		"Change plan deleted successfully",
		zap.String("change_plan_id", changePlanID.String()),
	)

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
		case models.SeverityError:
			errors = append(errors, item)
		case models.SeverityWarning:
			warnings = append(warnings, item)
		case models.SeverityInfo:
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

	// Ghost dry-run gate: ghost plans must pass a gh-ost dry run against the
	// live target before they can be approved (privileges, binlog access,
	// ALTER syntax validated without migrating).
	plan, err := h.changePlanRepo.GetByID(ctx, changePlanID)
	if err == nil && plan.ExecutionBackend == models.ExecutionBackendGhost {
		if dryRunErr := h.executionService.GhostDryRun(ctx, changePlanID); dryRunErr != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": fmt.Sprintf("Ghost dry-run failed; the plan cannot be approved: %s", dryRunErr.Error()),
			})
		}
	}

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

	h.logger.Info(
		"Change plan approved successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("approved_by", claims.UserID.String()),
	)

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

	h.logger.Info(
		"Change plan closed successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("closed_by", claims.UserID.String()),
	)

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
		projectRole := rbac.ProjectRole(member.Role)
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
	projectRole := rbac.ProjectRole(reviewerMember.Role)
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

	h.logger.Info(
		"Reviewer assigned successfully",
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

		h.logger.Warn(
			"Change plan execution blocked due to SQL Review errors (safety check)",
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

	h.logger.Info(
		"Change plan execution completed",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("executed_by", userID.String()),
		zap.Bool("success", result.Success),
		zap.Int("rows_affected", result.RowsAffected),
		zap.Duration("duration", result.Duration),
	)

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

	h.logger.Info(
		"Change plan resubmitted successfully",
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
			h.logger.Error(
				"Failed to create change history entry",
				zap.Error(err),
				zap.String("change_plan_id", newChangePlan.ID.String()),
				zap.String("field_name", change.fieldName),
			)
			// Continue tracking other fields even if one fails
		}
	}
}

// getParserDialect returns the parser dialect for a given database engine
func (h *ChangePlanHandler) getParserDialect(engine models.DatabaseEngine) parser.Dialect {
	switch engine {
	case models.DatabaseEngineMySQL:
		return parser.DialectMySQL
	case models.DatabaseEnginePostgreSQL:
		return parser.DialectPostgres
	default:
		// Default to PostgreSQL for unknown engines
		return parser.DialectPostgres
	}
}

// GetStatementResults handles GET /api/v1/change-plans/:id/execution-logs/:logId/statements
// Retrieves per-statement execution results for an execution log
func (h *ChangePlanHandler) GetStatementResults(c echo.Context) error {
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

	// Parse execution log ID
	executionLogIDStr := c.Param("logId")
	executionLogID, err := uuid.Parse(executionLogIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid execution log ID format",
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

	// RBAC check is done in middleware (RequireProjectPermission)

	// Get statement execution results
	results, err := h.statementResultRepo.GetByExecutionLogID(ctx, executionLogID)
	if err != nil {
		h.logger.Error("Failed to get statement execution results", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve statement execution results",
		})
	}

	// Convert to response format
	resultResponses := make([]*models.StatementExecutionResultResponse, len(results))
	for i, result := range results {
		resultResponses[i] = result.ToResponse()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_plan_id":    changePlanID.String(),
		"execution_log_id":  executionLogID.String(),
		"statement_results": resultResponses,
		"total":             len(resultResponses),
	})
}

// ApproveGhostCutover handles POST /api/v1/change-plans/:id/cutover-approve
// Removes the ghost postpone flag so the running tool proceeds to cutover.
func (h *ChangePlanHandler) ApproveGhostCutover(c echo.Context) error {
	ctx := c.Request().Context()

	changePlanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid change plan ID format",
		})
	}

	if err := h.executionService.ApproveGhostCutover(ctx, changePlanID); err != nil {
		status := http.StatusBadRequest
		if err == repository.ErrNotFound {
			status = http.StatusNotFound
		}
		return c.JSON(status, map[string]interface{}{
			"error": err.Error(),
		})
	}

	h.logger.Info("Ghost cutover approved",
		zap.String("change_plan_id", changePlanID.String()),
	)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Cutover approved",
	})
}

// AbortGhostExecution handles POST /api/v1/change-plans/:id/abort
// Creates the ghost panic flag to abort cleanly before the cutover.
func (h *ChangePlanHandler) AbortGhostExecution(c echo.Context) error {
	ctx := c.Request().Context()

	changePlanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid change plan ID format",
		})
	}

	if err := h.executionService.AbortGhostExecution(ctx, changePlanID); err != nil {
		status := http.StatusBadRequest
		if err == repository.ErrNotFound {
			status = http.StatusNotFound
		}
		return c.JSON(status, map[string]interface{}{
			"error": err.Error(),
		})
	}

	h.logger.Info("Ghost execution aborted",
		zap.String("change_plan_id", changePlanID.String()),
	)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Abort requested",
	})
}

// GhostPreflightCheck handles GET /api/v1/change-plans/:id/ghost-preflight
// Validates the target database meets ghost executor requirements:
// MySQL — binlog ROW + FULL row image, REPLICATION SLAVE/CLIENT grants,
// and ALL privileges on the database; PostgreSQL — table has PK and no
// triggers (pg-online-schema-change constraints). Returns per-check
// pass/fail with remediation guidance.
func (h *ChangePlanHandler) GhostPreflightCheck(c echo.Context) error {
	ctx := c.Request().Context()

	changePlanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid change plan ID format",
		})
	}

	changePlan, err := h.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Change plan not found",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve change plan",
		})
	}

	db, err := h.databaseRepo.GetByID(ctx, changePlan.TargetDatabaseID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve target database",
		})
	}
	dbInstance, err := h.databaseInstanceRepo.GetByID(ctx, db.DatabaseInstanceID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve database instance",
		})
	}

	conn, err := h.connectionManager.Connect(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": fmt.Sprintf("Failed to connect to target database: %s", err.Error()),
		})
	}
	defer func() {
		_ = conn.Close()
	}()

	type preflightCheck struct {
		Name   string `json:"name"`
		Passed bool   `json:"passed"`
		Detail string `json:"detail"`
	}
	var checks []preflightCheck
	addCheck := func(name string, passed bool, detail string) {
		checks = append(checks, preflightCheck{Name: name, Passed: passed, Detail: detail})
	}

	switch dbInstance.DatabaseEngine {
	case models.DatabaseEngineMySQL:
		// binlog_format must be ROW
		var name, value string
		err := conn.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'binlog_format'").Scan(&name, &value)
		addCheck("binlog_format=ROW", err == nil && strings.EqualFold(value, "ROW"),
			fmt.Sprintf("current: %s (gh-ost requires ROW; check RDS parameter group)", value))

		// binlog_row_image must be FULL
		value = ""
		err = conn.QueryRowContext(ctx, "SHOW VARIABLES LIKE 'binlog_row_image'").Scan(&name, &value)
		addCheck("binlog_row_image=FULL", err == nil && strings.EqualFold(value, "FULL"),
			fmt.Sprintf("current: %s (gh-ost requires FULL row image)", value))

		// Grants: REPLICATION SLAVE + REPLICATION CLIENT + ALL ON <db>.*
		rows, err := conn.QueryContext(ctx, "SHOW GRANTS")
		grantsPass, grantsDetail := false, "could not read grants"
		if err == nil {
			defer func() {
				_ = rows.Close()
			}()
			var all string
			for rows.Next() {
				var g string
				if scanErr := rows.Scan(&g); scanErr == nil {
					all += " " + strings.ToUpper(g)
				}
			}
			hasReplication := strings.Contains(all, "REPLICATION SLAVE") && strings.Contains(all, "REPLICATION CLIENT")
			dbNameEsc := strings.ToUpper(db.DatabaseName)
			hasDBPrivs := strings.Contains(all, "ALL PRIVILEGES") &&
				(strings.Contains(all, "ON *.*") || strings.Contains(all, "ON `"+dbNameEsc+"`.*") || strings.Contains(all, "ON "+dbNameEsc+".*"))
			grantsPass = hasReplication && hasDBPrivs
			grantsDetail = fmt.Sprintf(
				"required: ALL PRIVILEGES ON %s.* plus REPLICATION SLAVE, REPLICATION CLIENT ON *.*",
				db.DatabaseName,
			)
		}
		addCheck("privileges", grantsPass, grantsDetail)

		// Disk headroom estimate: gh-ost copies the full table (data + all
		// indexes) plus the new index and binlog growth during the copy. We
		// report the estimated requirement from information_schema; verifying
		// actual free disk (CloudWatch FreeStorageSpace on RDS) is the
		// operator's responsibility — this check always passes with a warning.
		if table, _, parseErr := services.ParseAlterStatement(changePlan.SQLContent); parseErr == nil {
			var dataLen, indexLen sql.NullInt64
			_ = conn.QueryRowContext(ctx,
				`SELECT DATA_LENGTH, INDEX_LENGTH FROM information_schema.TABLES
				 WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`, db.DatabaseName, table,
			).Scan(&dataLen, &indexLen)
			totalBytes := dataLen.Int64 + indexLen.Int64
			estGB := float64(totalBytes) / (1024 * 1024 * 1024)
			addCheck("disk_headroom_estimate", true,
				fmt.Sprintf("table %s is %.2f GB (data+indexes); gh-ost needs roughly that much free disk again plus binlog growth during the copy — verify free disk on the instance before executing", table, estGB))
		}

	case models.DatabaseEnginePostgreSQL:
		// pg-online-schema-change constraints: target table needs PK and no triggers.
		table, _, parseErr := services.ParseAlterStatement(changePlan.SQLContent)
		if parseErr != nil {
			addCheck("sql_parse", false, parseErr.Error())
			break
		}

		var pkCount int
		err := conn.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM information_schema.table_constraints
			 WHERE table_name = $1 AND constraint_type = 'PRIMARY KEY'`, table,
		).Scan(&pkCount)
		addCheck("primary_key", err == nil && pkCount > 0,
			"pg-online-schema-change requires a primary key on the target table")

		var triggerCount int
		err = conn.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM information_schema.triggers
			 WHERE event_object_table = $1`, table,
		).Scan(&triggerCount)
		addCheck("no_triggers", err == nil && triggerCount == 0,
			"pg-online-schema-change cannot run on tables with triggers")
	}

	allPassed := true
	for _, ch := range checks {
		if !ch.Passed {
			allPassed = false
			break
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_plan_id": changePlanID.String(),
		"engine":         dbInstance.DatabaseEngine,
		"all_passed":     allPassed,
		"checks":         checks,
	})
}

// DropGhostBackupTable handles POST /api/v1/change-plans/:id/ghost-drop-backup
// Drops the retained _<table>_del backup table after a completed ghost migration.
func (h *ChangePlanHandler) DropGhostBackupTable(c echo.Context) error {
	ctx := c.Request().Context()

	changePlanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid change plan ID format",
		})
	}

	if err := h.executionService.DropGhostBackupTable(ctx, changePlanID); err != nil {
		status := http.StatusBadRequest
		if err == repository.ErrNotFound {
			status = http.StatusNotFound
		}
		return c.JSON(status, map[string]interface{}{
			"error": err.Error(),
		})
	}

	h.logger.Info("Ghost backup table dropped",
		zap.String("change_plan_id", changePlanID.String()),
	)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Backup table dropped",
	})
}

// RollbackGhostPlan handles POST /api/v1/change-plans/:id/ghost-rollback
// Atomically swaps the backup table back into place (data-loss warning:
// post-cutover writes are sidelined into _<table>_new).
func (h *ChangePlanHandler) RollbackGhostPlan(c echo.Context) error {
	ctx := c.Request().Context()

	changePlanID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid change plan ID format",
		})
	}

	if err := h.executionService.RollbackGhostPlan(ctx, changePlanID); err != nil {
		status := http.StatusBadRequest
		if err == repository.ErrNotFound {
			status = http.StatusNotFound
		}
		return c.JSON(status, map[string]interface{}{
			"error": err.Error(),
		})
	}

	h.logger.Info("Ghost plan rolled back",
		zap.String("change_plan_id", changePlanID.String()),
	)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Plan rolled back; post-cutover writes preserved in the sidelined table",
	})
}
