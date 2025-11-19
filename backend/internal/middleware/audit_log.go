package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/audit"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/config"
	"github.com/nsxbet/deeb/internal/services"
	"go.uber.org/zap"
)

// AuditLogServiceInterface defines the interface for audit logging service
type AuditLogServiceInterface interface {
	LogWithHTTP(
		ctx context.Context,
		userID *uuid.UUID,
		action, resourceType string,
		resourceID uuid.UUID,
		httpRequest *http.Request,
		details json.RawMessage,
	) error
}

// AuditEnrichment contains optional additional details that handlers can provide
// to enrich audit logs with business-specific context.
// Handlers can set this in Echo context: c.Set("audit_enrichment", &middleware.AuditEnrichment{...})
type AuditEnrichment struct {
	UserID        *uuid.UUID             `json:"-"` // User ID for unauthenticated operations (login, register)
	Email         string                 `json:"email,omitempty"`
	FailureReason string                 `json:"failure_reason,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Result        string                 `json:"result,omitempty"` // "success" or "failure"
	Custom        map[string]interface{} `json:"-"`                // For arbitrary fields (merged inline)
}

// responseWriter wraps http.ResponseWriter to capture response body
type responseWriter struct {
	http.ResponseWriter
	body io.Writer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

// AuditLogMiddleware creates an HTTP middleware that logs all requests to the audit log.
// It captures request details (method, path, user) and response status.
type AuditLogMiddleware struct {
	auditService     AuditLogServiceInterface
	logger           *zap.Logger
	redactor         *audit.Redactor
	redactionEnabled bool
	config           *config.AuditLoggingConfig
}

// NewAuditLogMiddleware creates a new audit logging middleware
func NewAuditLogMiddleware(
	auditService AuditLogServiceInterface,
	logger *zap.Logger,
	cfg *config.AuditLoggingConfig,
) *AuditLogMiddleware {
	// Use redaction setting from config
	redactionEnabled := true
	if cfg != nil {
		redactionEnabled = cfg.RedactionEnabled
	}

	return &AuditLogMiddleware{
		auditService:     auditService,
		logger:           logger,
		redactor:         audit.NewRedactor(),
		redactionEnabled: redactionEnabled,
		config:           cfg,
	}
}

// SetRedactionEnabled allows enabling/disabling redaction (default: enabled)
func (m *AuditLogMiddleware) SetRedactionEnabled(enabled bool) {
	m.redactionEnabled = enabled
}

// AuditLog is the Echo middleware handler that logs requests to the audit log.
// It skips health check endpoints and static assets.
// Logging is done asynchronously in a goroutine to avoid blocking the request.
func (m *AuditLogMiddleware) AuditLog(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Skip health check and static asset endpoints based on path
		path := c.Request().URL.Path
		if m.shouldSkipAuditLog(path) {
			return next(c)
		}

		// Capture request details
		req := c.Request()
		method := req.Method
		userID := m.extractUserID(c)

		// Capture request body for audit logging (read and restore)
		var requestBody []byte
		if req.Body != nil && (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch) {
			bodyBytes, readErr := io.ReadAll(req.Body)
			if readErr == nil {
				requestBody = bodyBytes
				// Restore the body for the handler to read
				req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Create a response writer wrapper to capture response body
		resBody := new(bytes.Buffer)
		mw := io.MultiWriter(c.Response().Writer, resBody)
		writer := &responseWriter{c.Response().Writer, mw}
		c.Response().Writer = writer

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
		m.logger.Debug(
			"Action and resource type",
			zap.String("action", action),
			zap.String("resource_type", resourceType),
			zap.String("resource_id", resourceID.String()),
		)

		// Check if this resource/action should be logged based on configuration
		shouldLogResource := m.shouldLogResource(resourceType, action)

		if !shouldLogResource {
			return err
		}

		// Check for enrichment from handler
		var enrichment *AuditEnrichment
		if enrichmentVal := c.Get("audit_enrichment"); enrichmentVal != nil {
			if e, ok := enrichmentVal.(*AuditEnrichment); ok {
				enrichment = e
			}
		}

		// Build details JSON with request/response information and enrichment
		details := m.buildDetails(method, path, status, err, enrichment, requestBody, resBody.Bytes())

		// Determine the user_id to use for audit logging
		// Priority:
		// 1. For authenticated requests, use the extracted userID from auth token
		// 2. For unauthenticated auth endpoints (login/register), use enrichment UserID if available
		// 3. Otherwise, use nil to represent unauthenticated events
		var auditUserID *uuid.UUID
		if userID != uuid.Nil {
			// Authenticated request - use the extracted user ID
			auditUserID = &userID
		} else if enrichment != nil && enrichment.UserID != nil {
			// Unauthenticated request with enrichment (e.g., successful login) - use enrichment user ID
			auditUserID = enrichment.UserID
		}
		// else: auditUserID remains nil for failed logins and other unauthenticated events

		// Determine if we should log this request
		// Log if:
		// 1. User is authenticated (userID != Nil), OR
		// 2. This is an auth endpoint (login, logout, register) - log attempts regardless of auth status
		isAuthEndpoint := m.isAuthEndpoint(action, path)
		shouldLog := userID != uuid.Nil || isAuthEndpoint

		// Log to audit log asynchronously
		// Run in a goroutine to avoid blocking the request
		if shouldLog {
			// Create a new context that is not tied to the request lifecycle
			// Use context.Background() instead of request context to avoid cancellation
			ctx := context.Background()

			// Capture variables for the goroutine
			capturedUserID := auditUserID
			capturedAction := action
			capturedResourceType := resourceType
			capturedResourceID := resourceID
			capturedReq := req
			capturedDetails := details
			capturedMethod := method
			capturedPath := path

			// Launch goroutine for async audit logging
			go func() {
				if logErr := m.auditService.LogWithHTTP(ctx, capturedUserID, capturedAction, capturedResourceType, capturedResourceID, capturedReq, capturedDetails); logErr != nil {
					// Log error but don't fail the request
					userIDStr := "nil"
					if capturedUserID != nil {
						userIDStr = capturedUserID.String()
					}
					m.logger.Error("Failed to create audit log entry",
						zap.String("method", capturedMethod),
						zap.String("path", capturedPath),
						zap.String("user_id", userIDStr),
						zap.Error(logErr),
					)
				}
			}()
		}

		return err
	}
}

// isAuthEndpoint returns true if the action is an authentication-related action
// that should be logged even for unauthenticated requests
func (m *AuditLogMiddleware) isAuthEndpoint(action, path string) bool {
	return action == services.ActionLogin ||
		action == services.ActionLogout ||
		(strings.Contains(path, "/auth/register") && action == services.ActionCreate)
}

// shouldLogResource determines if a resource and action should be logged based on configuration.
// Returns true if the resource/action should be logged, false if it should be skipped.
func (m *AuditLogMiddleware) shouldLogResource(resourceType, action string) bool {
	// If config is nil or audit logging is disabled, don't log
	if m.config == nil || !m.config.Enabled {
		return false
	}

	// Check if resource is configured
	rule, resourceConfigured := m.config.Resources[resourceType]

	// Apply default policy based on whether resource is configured
	if !resourceConfigured {
		// Resource not explicitly configured - use default policy
		return m.config.DefaultPolicy == "allow"
	}

	// Resource is configured - check if it's enabled
	if !rule.Enabled {
		return false
	}

	// Resource is enabled - check if action is in the allowed actions list
	// Empty actions list means log all actions for this resource
	if len(rule.Actions) == 0 {
		return true
	}

	// Check if action is in the allowed list
	for _, allowedAction := range rule.Actions {
		if allowedAction == action {
			return true
		}
	}

	// Action not in allowed list
	return false
}

// shouldSkipAuditLog returns true if the path should be skipped for audit logging
func (m *AuditLogMiddleware) shouldSkipAuditLog(path string) bool {
	// If log_all_paths is enabled, don't skip any paths based on hardcoded rules
	if m.config != nil && m.config.LogAllPaths {
		// Still check custom skip paths from config
		return m.matchesSkipPath(path)
	}

	// Apply hardcoded skip rules (health checks, static assets, etc.)
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

	// Check custom skip paths from config
	return m.matchesSkipPath(path)
}

// matchesSkipPath checks if the path matches any of the custom skip path patterns
func (m *AuditLogMiddleware) matchesSkipPath(path string) bool {
	if m.config == nil {
		return false
	}

	for _, skipPattern := range m.config.SkipPaths {
		// Simple wildcard matching (* at the end)
		if strings.HasSuffix(skipPattern, "*") {
			prefix := strings.TrimSuffix(skipPattern, "*")
			if strings.HasPrefix(path, prefix) {
				return true
			}
		} else if path == skipPattern {
			// Exact match
			return true
		}
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
	// Use HasSuffix for more precise matching to avoid false positives
	// (e.g., "/user/approve-something" should not match "/approve")
	if strings.HasSuffix(path, "/approve") {
		action = services.ActionApproveChangePlan
	} else if strings.HasSuffix(path, "/close") {
		action = services.ActionCloseChangePlan
	} else if strings.HasSuffix(path, "/reject") {
		action = services.ActionRejectChangePlan
	} else if strings.HasSuffix(path, "/execute") {
		action = services.ActionExecuteChangePlan
	} else if strings.Contains(path, "/login") {
		// Keep Contains for login since it can be /auth/login or /login
		action = services.ActionLogin
		resourceType = services.ResourceTypeUser
	} else if strings.Contains(path, "/logout") {
		// Keep Contains for logout since it can be /auth/logout or /logout
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

	// Map all known resource types explicitly
	// This ensures consistency with constants defined in services package
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
	case "approval":
		return services.ResourceTypeApproval
	case "execution_log":
		return services.ResourceTypeExecutionLog
	case "changelog":
		return services.ResourceTypeChangelog
	case "schema_history":
		return services.ResourceTypeSchemaHistory
	case "policy":
		return services.ResourceTypePolicy
	case "policy_finding":
		return services.ResourceTypePolicyFinding
	default:
		// Return the converted type for unknown resources
		// This allows the system to work with new resources not yet explicitly mapped
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

// buildDetails builds the details JSON object for the audit log entry.
// Merges enrichment data from handlers if provided.
func (m *AuditLogMiddleware) buildDetails(
	method, path string,
	status int,
	err error,
	enrichment *AuditEnrichment,
	requestBody []byte,
	responseBody []byte,
) json.RawMessage {
	detailsMap := map[string]interface{}{
		"method":      method,
		"path":        path,
		"status":      status,
		"status_code": status,
	}

	// Add request payload if present (limit size to avoid huge audit logs)
	if len(requestBody) > 0 {
		// Try to parse as JSON for better formatting
		var requestJSON interface{}
		if json.Unmarshal(requestBody, &requestJSON) == nil {
			detailsMap["request"] = requestJSON
		} else {
			// If not JSON, store as string (truncate if too large)
			if len(requestBody) > 1024 {
				detailsMap["request"] = string(requestBody[:1024]) + "... (truncated)"
			} else {
				detailsMap["request"] = string(requestBody)
			}
		}
	}

	// Add response data if present (limit size to avoid huge audit logs)
	if len(responseBody) > 0 {
		// Try to parse as JSON for better formatting
		var responseJSON interface{}
		if json.Unmarshal(responseBody, &responseJSON) == nil {
			// Truncate large responses
			if len(responseBody) > 2048 {
				detailsMap["response"] = string(responseBody[:2048]) + "... (truncated)"
			} else {
				detailsMap["response"] = responseJSON
			}
		} else {
			// If not JSON, store as string (truncate if too large)
			if len(responseBody) > 1024 {
				detailsMap["response"] = string(responseBody[:1024]) + "... (truncated)"
			} else {
				detailsMap["response"] = string(responseBody)
			}
		}
	}

	// Determine result from enrichment or status code
	if enrichment != nil && enrichment.Result != "" {
		detailsMap["result"] = enrichment.Result
	} else if status >= 400 {
		detailsMap["result"] = "failure"
	} else {
		detailsMap["result"] = "success"
	}

	// Merge enrichment data if provided
	if enrichment != nil {
		if enrichment.Email != "" {
			detailsMap["email"] = enrichment.Email
		}
		if enrichment.FailureReason != "" {
			detailsMap["failure_reason"] = enrichment.FailureReason
		}
		if enrichment.ErrorMessage != "" {
			detailsMap["error_message"] = enrichment.ErrorMessage
		}
		// Merge custom fields inline
		for k, v := range enrichment.Custom {
			detailsMap[k] = v
		}
	}

	// Add error message from handler error if not already set
	if err != nil && detailsMap["error_message"] == nil {
		detailsMap["error_message"] = err.Error()
	}

	// Apply redaction to sensitive fields if enabled
	if m.redactionEnabled {
		detailsMap = m.redactor.RedactMap(detailsMap)
	}

	detailsJSON, jsonErr := json.Marshal(detailsMap)
	if jsonErr != nil {
		m.logger.Error(
			"Failed to marshal details",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Error(jsonErr),
		)
		// Fallback to simple JSON if marshaling fails
		detailsJSON = []byte(`{"method":"` + method + `","path":"` + path + `","status":` + strconv.Itoa(status) + `}`)
	}

	return detailsJSON
}
