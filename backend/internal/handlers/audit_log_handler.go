package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// AuditLogHandler handles HTTP requests for audit log operations
type AuditLogHandler struct {
	auditLogRepo repository.AuditLogRepository
	logger       *zap.Logger
}

// NewAuditLogHandler creates a new audit log handler
func NewAuditLogHandler(
	auditLogRepo repository.AuditLogRepository,
	logger *zap.Logger,
) *AuditLogHandler {
	return &AuditLogHandler{
		auditLogRepo: auditLogRepo,
		logger:       logger,
	}
}

// ListAuditLogs handles GET /api/v1/audit-logs
// Lists audit logs with filtering and pagination (admin only)
func (h *AuditLogHandler) ListAuditLogs(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// RBAC check: admin only
	if claims.OrganizationRole != string(models.OrganizationRoleAdmin) {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "Admin access required",
		})
	}

	// Parse pagination parameters
	limit := 100 // default
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		} else {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid limit parameter",
			})
		}
	}

	offset := 0 // default
	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		} else {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid offset parameter",
			})
		}
	}

	// Build filters
	filters := repository.AuditLogListFilters{
		Limit:  limit,
		Offset: offset,
	}

	// Parse optional user_id filter
	if userIDStr := c.QueryParam("user_id"); userIDStr != "" {
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid user_id parameter",
			})
		}
		filters.UserID = userID
	}

	// Parse optional action filter
	if action := c.QueryParam("action"); action != "" {
		filters.Action = action
	}

	// Parse optional target_type filter (resource_type)
	if targetType := c.QueryParam("target_type"); targetType != "" {
		filters.ResourceType = targetType
	}

	// Parse optional target_id filter (resource_id)
	if targetIDStr := c.QueryParam("target_id"); targetIDStr != "" {
		targetID, err := uuid.Parse(targetIDStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid target_id parameter",
			})
		}
		filters.ResourceID = targetID
	}

	// Parse optional start_date filter
	if startDateStr := c.QueryParam("start_date"); startDateStr != "" {
		startDate, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid start_date parameter. Must be RFC3339 format (e.g., 2025-01-01T00:00:00Z)",
			})
		}
		filters.StartDate = &startDate
	}

	// Parse optional end_date filter
	if endDateStr := c.QueryParam("end_date"); endDateStr != "" {
		endDate, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid end_date parameter. Must be RFC3339 format (e.g., 2025-12-31T23:59:59Z)",
			})
		}
		filters.EndDate = &endDate
	}

	// Retrieve audit logs with filters
	auditLogs, err := h.auditLogRepo.Filter(ctx, filters)
	if err != nil {
		h.logger.Error("Failed to filter audit logs", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve audit logs",
		})
	}

	// Get total count with same filters (without pagination)
	countFilters := filters
	countFilters.Limit = 0
	countFilters.Offset = 0
	total, err := h.auditLogRepo.CountWithFilters(ctx, countFilters)
	if err != nil {
		h.logger.Error("Failed to count audit logs", zap.Error(err))
		// Don't fail the request, just use the current page count
		total = int64(len(auditLogs))
	}

	// Convert to response format
	responses := make([]*models.AuditLogResponse, len(auditLogs))
	for i, log := range auditLogs {
		responses[i] = log.ToResponse()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"audit_logs": responses,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
		"count":      len(responses),
	})
}
