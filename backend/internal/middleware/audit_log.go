package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/services"
	"go.uber.org/zap"
)

// AuditLogServiceInterface defines the interface for audit logging service
type AuditLogServiceInterface interface {
	LogWithHTTP(
		ctx context.Context,
		userID uuid.UUID,
		action, resourceType string,
		resourceID uuid.UUID,
		httpRequest *http.Request,
		details json.RawMessage,
	) error
}

// AuditLogMiddleware creates an HTTP middleware that logs all requests to the audit log.
// It captures request details (method, path, user) and response status.
type AuditLogMiddleware struct {
	auditService AuditLogServiceInterface
	logger       *zap.Logger
}

// NewAuditLogMiddleware creates a new audit logging middleware
func NewAuditLogMiddleware(auditService AuditLogServiceInterface, logger *zap.Logger) *AuditLogMiddleware {
	return &AuditLogMiddleware{
		auditService: auditService,
		logger:       logger,
	}
}

// AuditLog is the Echo middleware handler that logs requests to the audit log.
// It skips health check endpoints and static assets.
func (m *AuditLogMiddleware) AuditLog(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skip health check and static asset endpoints
		path := c.Request().URL.Path
		if shouldSkipAuditLog(path) {
			return next(c)
		}

		// Capture request details
		req := c.Request()
		method := req.Method
		userID := m.extractUserID(c)

		// Process request
		err := next(c)

		// Capture response status
		status := c.Response().Status
		// If status is 0 or 200 and there's an error, extract status from error
		if err != nil && (status == 0 || status == 200) {
			// Try to extract status from HTTPError
			if httpErr, ok := err.(*echo.HTTPError); ok {
				status = httpErr.Code
			} else {
				// Default to 500 for unknown errors
				status = http.StatusInternalServerError
			}
		}
		// If status is still 0, default to 200
		if status == 0 {
			status = http.StatusOK
		}

		// Determine action and resource type from request
		action, resourceType, resourceID := m.determineActionAndResource(method, path)

		// Build details JSON with request/response information
		details := m.buildDetails(method, path, status, err)

		// Log to audit log (only if user is authenticated)
		// For unauthenticated requests, we still log but with system user ID if needed
		if userID != uuid.Nil {
			ctx := c.Request().Context()
			if err := m.auditService.LogWithHTTP(ctx, userID, action, resourceType, resourceID, req, details); err != nil {
				// Log error but don't fail the request
				m.logger.Error("Failed to create audit log entry",
					zap.String("method", method),
					zap.String("path", path),
					zap.String("user_id", userID.String()),
					zap.Error(err),
				)
			}
		}

		return err
	}
}

// shouldSkipAuditLog returns true if the path should be skipped for audit logging
func shouldSkipAuditLog(path string) bool {
	// Skip health check endpoints
	if path == "/health" || path == "/healthz" || path == "/ready" {
		return true
	}

	// Skip static assets
	if strings.HasPrefix(path, "/static/") ||
		strings.HasPrefix(path, "/assets/") ||
		strings.HasSuffix(path, ".css") ||
		strings.HasSuffix(path, ".js") ||
		strings.HasSuffix(path, ".png") ||
		strings.HasSuffix(path, ".jpg") ||
		strings.HasSuffix(path, ".ico") {
		return true
	}

	// Skip Swagger UI assets (but log API docs access)
	if strings.HasPrefix(path, "/api/docs/swagger") {
		return true
	}

	return false
}

// extractUserID extracts the user ID from the Echo context
// Returns uuid.Nil if user is not authenticated
func (m *AuditLogMiddleware) extractUserID(c echo.Context) uuid.UUID {
	// Try to get user from context (set by AuthMiddleware)
	userClaims, ok := c.Get("user").(*auth.Claims)
	if !ok || userClaims == nil {
		return uuid.Nil
	}

	return userClaims.UserID
}

// determineActionAndResource determines the action type and resource type from the HTTP method and path
func (m *AuditLogMiddleware) determineActionAndResource(method, path string) (action, resourceType string, resourceID uuid.UUID) {
	// Map HTTP methods to actions
	switch method {
	case http.MethodPost:
		action = services.ActionCreate
	case http.MethodGet:
		action = services.ActionRead
	case http.MethodPut, http.MethodPatch:
		action = services.ActionUpdate
	case http.MethodDelete:
		action = services.ActionDelete
	default:
		action = strings.ToLower(method)
	}

	// Determine resource type from path
	// Example: /api/v1/change-plans/{id} -> resource_type = "change_plan"
	resourceType = m.extractResourceType(path)

	// Extract resource ID from path if present
	// Example: /api/v1/change-plans/{id} -> extract {id}
	resourceID = m.extractResourceID(path)

	// Special handling for specific endpoints
	if strings.Contains(path, "/approve") {
		action = services.ActionApproveChangePlan
	} else if strings.Contains(path, "/close") {
		action = services.ActionCloseChangePlan
	} else if strings.Contains(path, "/reject") {
		action = services.ActionRejectChangePlan
	} else if strings.Contains(path, "/execute") {
		action = services.ActionExecuteChangePlan
	} else if strings.Contains(path, "/login") {
		action = services.ActionLogin
		resourceType = services.ResourceTypeUser
	} else if strings.Contains(path, "/logout") {
		action = services.ActionLogout
		resourceType = services.ResourceTypeUser
	}

	return action, resourceType, resourceID
}

// extractResourceType extracts the resource type from the API path
func (m *AuditLogMiddleware) extractResourceType(path string) string {
	// Remove API prefix
	path = strings.TrimPrefix(path, "/api/v1/")

	// Extract first segment (e.g., "change-plans" -> "change_plan")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "unknown"
	}

	resourceType := parts[0]

	// Convert kebab-case to snake_case
	resourceType = strings.ReplaceAll(resourceType, "-", "_")

	// Handle plural forms (TrimSuffix is safe even if suffix doesn't exist)
	resourceType = strings.TrimSuffix(resourceType, "s")

	// Map common resource types
	switch resourceType {
	case "change_plan":
		return services.ResourceTypeChangePlan
	case "project":
		return services.ResourceTypeProject
	case "database":
		return services.ResourceTypeDatabase
	case "database_instance":
		return services.ResourceTypeDatabaseInstance
	case "user":
		return services.ResourceTypeUser
	case "environment":
		return services.ResourceTypeEnvironment
	case "audit_log":
		return services.ResourceTypeAuditLog
	default:
		return resourceType
	}
}

// extractResourceID extracts the resource ID from the path if it's a UUID
func (m *AuditLogMiddleware) extractResourceID(path string) uuid.UUID {
	// Remove query string if present
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}

	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Try to parse as UUID
		if id, err := uuid.Parse(part); err == nil {
			return id
		}
	}
	return uuid.Nil
}

// buildDetails builds the details JSON object for the audit log entry
func (m *AuditLogMiddleware) buildDetails(method, path string, status int, err error) json.RawMessage {
	result := "success"
	if status >= 400 {
		result = "failure"
	}

	detailsMap := map[string]interface{}{
		"method":      method,
		"path":        path,
		"status":      status,
		"result":      result,
		"status_code": status,
	}

	if err != nil {
		detailsMap["error_message"] = err.Error()
	}

	detailsJSON, jsonErr := json.Marshal(detailsMap)
	if jsonErr != nil {
		// Fallback to simple JSON if marshaling fails
		detailsJSON = []byte(`{"method":"` + method + `","path":"` + path + `","status":` + strconv.Itoa(status) + `}`)
	}

	return detailsJSON
}
