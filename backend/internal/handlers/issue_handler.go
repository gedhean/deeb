package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

const (
	// MaxSQLContentSize is the maximum allowed size for SQL content (10MB)
	MaxSQLContentSize = 10 * 1024 * 1024 // 10MB in bytes
)

// IssueHandler handles HTTP requests for issue operations
type IssueHandler struct {
	issueRepo repository.IssueRepository
	logger    *zap.Logger
}

// NewIssueHandler creates a new issue handler
func NewIssueHandler(issueRepo repository.IssueRepository, logger *zap.Logger) *IssueHandler {
	return &IssueHandler{
		issueRepo: issueRepo,
		logger:    logger,
	}
}

// CreateIssueRequest represents the request body for creating an issue
type CreateIssueRequest struct {
	ProjectID        string            `json:"project_id"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	ChangeType       models.ChangeType `json:"change_type"`
	SQLContent       string            `json:"sql_content"`
	RollbackSQL      *string           `json:"rollback_sql,omitempty"`
	TargetDatabaseID string            `json:"target_database_id"`
	RiskLevel        *models.RiskLevel `json:"risk_level,omitempty"`
}

// CreateIssue handles POST /api/v1/issues
func (h *IssueHandler) CreateIssue(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse request body
	var req CreateIssueRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("Failed to bind request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Title == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "title is required",
		})
	}
	if req.Description == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "description is required",
		})
	}
	if req.SQLContent == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "sql_content is required",
		})
	}
	if req.TargetDatabaseID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "target_database_id is required",
		})
	}
	if req.ProjectID == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "project_id is required",
		})
	}

	// Validate SQL content size limit (10MB max)
	if len(req.SQLContent) > MaxSQLContentSize {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "sql_content exceeds maximum size of 10MB",
		})
	}

	// Parse UUIDs
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid project_id format",
		})
	}

	targetDatabaseID, err := uuid.Parse(req.TargetDatabaseID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "invalid target_database_id format",
		})
	}

	// Get authenticated user ID from context
	userIDStr, ok := c.Get("user_id").(string)
	if !ok || userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "User not authenticated",
		})
	}

	authorID, err := uuid.Parse(userIDStr)
	if err != nil {
		h.logger.Error("Failed to parse user_id", zap.String("user_id", userIDStr), zap.Error(err))
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Invalid user ID",
		})
	}

	// Set default risk level if not provided
	riskLevel := models.RiskLevelMedium
	if req.RiskLevel != nil {
		riskLevel = *req.RiskLevel
	}

	// Default change type to DDL if not provided
	changeType := req.ChangeType
	if changeType == "" {
		changeType = models.ChangeTypeDDL
	}

	// Create issue model with Draft status
	issue := &models.Issue{
		ProjectID:        projectID,
		Title:            req.Title,
		Description:      req.Description,
		ChangeType:       changeType,
		SQLContent:       req.SQLContent,
		RollbackSQL:      req.RollbackSQL,
		TargetDatabaseID: targetDatabaseID,
		AuthorID:         authorID,
		Status:           models.IssueStatusDraft,
		RiskLevel:        riskLevel,
	}

	// Create issue in repository
	if err := h.issueRepo.Create(ctx, issue); err != nil {
		h.logger.Error("Failed to create issue", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create issue",
		})
	}

	h.logger.Info("Issue created successfully",
		zap.String("issue_id", issue.ID.String()),
		zap.String("author_id", authorID.String()))

	// Return 201 Created with issue data
	return c.JSON(http.StatusCreated, issue)
}
