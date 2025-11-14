package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
)

// AuthzMiddleware provides authorization middleware functions.
// It checks user permissions at both organization and project levels.
type AuthzMiddleware struct {
	checker           *rbac.CheckerService
	projectMemberRepo repository.ProjectMemberRepository
}

// NewAuthzMiddleware creates a new authorization middleware with an RBAC checker service.
func NewAuthzMiddleware() *AuthzMiddleware {
	return &AuthzMiddleware{
		checker: rbac.NewCheckerService(),
	}
}

// NewAuthzMiddlewareWithRepo creates a new authorization middleware with an RBAC checker service
// and project member repository for project-level permission checks.
func NewAuthzMiddlewareWithRepo(projectMemberRepo repository.ProjectMemberRepository) *AuthzMiddleware {
	return &AuthzMiddleware{
		checker:           rbac.NewCheckerService(),
		projectMemberRepo: projectMemberRepo,
	}
}

// RequireOrganizationPermission returns middleware that checks if the authenticated user
// has the specified organization-level permission.
// Returns 401 Unauthorized if user is not authenticated, 403 Forbidden if lacking permission.
func (m *AuthzMiddleware) RequireOrganizationPermission(permission rbac.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract user information from context (set by auth middleware)
			_, _, organizationRole, ok := rbac.GetUserFromContext(c.Request().Context())
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "Authentication required",
				})
			}

			// Check permission
			if !m.checker.HasOrganizationPermission(organizationRole, permission) {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "Insufficient permissions",
					"details": map[string]interface{}{
						"required_permission": permission,
						"user_role":           organizationRole,
					},
				})
			}

			return next(c)
		}
	}
}

// RequireProjectPermission returns middleware that checks if the authenticated user
// has the specified project-level permission for the project in the route.
// Organization admins bypass project-level checks.
// Project ID can be extracted from route parameters: "project_id" or "id".
func (m *AuthzMiddleware) RequireProjectPermission(permission rbac.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract user information from context
			userIDStr, _, organizationRole, ok := rbac.GetUserFromContext(c.Request().Context())
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "Authentication required",
				})
			}

			// Parse user ID
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "Invalid user ID",
				})
			}

			// Extract project ID - check in this order:
			// 1. c.Get() - set by middleware like ExtractProjectFromBody or ChangePlanProjectExtractor
			// 2. c.Param("project_id") - route parameter
			// 3. c.Param("id") - fallback for routes where :id refers to project ID
			var projectIDStr string
			if val := c.Get("project_id"); val != nil {
				if str, ok := val.(string); ok {
					projectIDStr = str
				}
			}
			if projectIDStr == "" {
				projectIDStr = c.Param("project_id")
			}
			if projectIDStr == "" {
				projectIDStr = c.Param("id")
			}

			if projectIDStr == "" {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Project ID required",
				})
			}

			// Parse project ID
			projectID, err := uuid.Parse(projectIDStr)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Invalid project ID format",
				})
			}

			// Organization admins bypass project-level checks
			if organizationRole == rbac.OrganizationAdmin {
				// Add project context for downstream handlers
				ctx := context.WithValue(c.Request().Context(), rbac.ContextKeyProjectID, projectIDStr)
				ctx = context.WithValue(ctx, rbac.ContextKeyProjectRole, rbac.ProjectOwner)
				c.SetRequest(c.Request().WithContext(ctx))
				return next(c)
			}

			// Check if project member repository is available
			if m.projectMemberRepo == nil {
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Project permission checking not configured",
				})
			}

			// Fetch user's project role from database
			ctx := c.Request().Context()
			member, err := m.projectMemberRepo.GetMemberRole(ctx, projectID, userID)
			if err != nil {
				if err == repository.ErrNotFound {
					return c.JSON(http.StatusForbidden, map[string]interface{}{
						"error": "Not a member of this project",
						"details": map[string]interface{}{
							"user_id":    userID.String(),
							"project_id": projectID.String(),
						},
					})
				}
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Failed to check project membership",
				})
			}

			// Cast rbac.ProjectRole to rbac.ProjectRole (both use PascalCase format)
			projectRole := rbac.ProjectRole(member.Role)
			if !rbac.IsValidProjectRole(projectRole) {
				return c.JSON(http.StatusInternalServerError, map[string]interface{}{
					"error": "Invalid project role",
				})
			}

			// Check if user has the required permission
			if !m.checker.HasProjectPermission(projectRole, permission) {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "Insufficient permissions",
					"details": map[string]interface{}{
						"required_permission": permission,
						"user_project_role":   projectRole,
					},
				})
			}

			// Add project context for downstream handlers
			reqCtx := context.WithValue(c.Request().Context(), rbac.ContextKeyProjectID, projectIDStr)
			reqCtx = context.WithValue(reqCtx, rbac.ContextKeyProjectRole, projectRole)
			c.SetRequest(c.Request().WithContext(reqCtx))

			return next(c)
		}
	}
}

// RequireOrganizationAdmin is a convenience middleware that requires organization admin role.
// Returns 403 Forbidden if the user does not have the OrganizationAdmin role.
func (m *AuthzMiddleware) RequireOrganizationAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			_, _, organizationRole, ok := rbac.GetUserFromContext(c.Request().Context())
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "Authentication required",
				})
			}

			if organizationRole != rbac.OrganizationAdmin {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "Organization admin role required",
					"details": map[string]interface{}{
						"user_role": organizationRole,
					},
				})
			}

			return next(c)
		}
	}
}

// RequireOrganizationDBA is a convenience middleware that requires organization DBA role or higher.
// Allows both OrganizationAdmin and OrganizationDBA roles to proceed.
func (m *AuthzMiddleware) RequireOrganizationDBA() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			_, _, organizationRole, ok := rbac.GetUserFromContext(c.Request().Context())
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "Authentication required",
				})
			}

			// Admin or DBA roles allowed
			if organizationRole != rbac.OrganizationAdmin && organizationRole != rbac.OrganizationDBA {
				return c.JSON(http.StatusForbidden, map[string]interface{}{
					"error": "Organization DBA or admin role required",
					"details": map[string]interface{}{
						"user_role": organizationRole,
					},
				})
			}

			return next(c)
		}
	}
}

// RequirePermission returns middleware that checks if the authenticated user has the specified permission.
// This method automatically determines if the permission is organization-level or project-level:
//   - Organization-level permissions (e.g., "databases.read", "users.create") are checked against the user's organization role
//   - Project-level permissions (e.g., "changes.create", "project.members.read") require a project_id in the route
//     and check the user's project role. Organization admins bypass project-level checks.
//
// For project-level permissions:
//   - The project ID must be available in route parameters as "project_id" or "id"
//   - Use ExtractProjectFromBody() middleware before this for routes with project_id in request body
//   - Use ChangePlanProjectExtractor middleware for change plan routes
//
// Returns 401 Unauthorized if not authenticated, 403 Forbidden if lacking permission.
func (m *AuthzMiddleware) RequirePermission(permission rbac.Permission) echo.MiddlewareFunc {
	// Determine permission scope at middleware creation time
	isProjectLevel := rbac.IsProjectLevelPermission(permission)

	if isProjectLevel {
		// For project-level permissions, delegate to RequireProjectPermission
		return m.RequireProjectPermission(permission)
	}

	// For organization-level permissions, delegate to RequireOrganizationPermission
	return m.RequireOrganizationPermission(permission)
}

// ExtractProjectFromBody is middleware that extracts project_id from the request body
// and sets it in the Echo context for use by downstream middleware.
// This allows RequireProjectPermission middleware to work with routes that pass project_id in body.
// The request body is read and buffered so it can be read again by the handler.
func ExtractProjectFromBody() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Read the body
			bodyBytes, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Failed to read request body",
				})
			}

			// Parse JSON to extract project_id
			var body map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &body); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Invalid JSON in request body",
				})
			}

			// Extract project_id
			projectIDValue, ok := body["project_id"]
			if !ok {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "project_id is required in request body",
				})
			}

			projectIDStr, ok := projectIDValue.(string)
			if !ok {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "project_id must be a string",
				})
			}

			// Validate it's a valid UUID
			if _, err := uuid.Parse(projectIDStr); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "project_id must be a valid UUID",
				})
			}

			// Store project_id in Echo context so RequireProjectPermission can use it
			c.Set("project_id", projectIDStr)

			// Restore the body for the handler to read
			c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			return next(c)
		}
	}
}

// ChangePlanProjectExtractor provides middleware to extract project_id from a change plan.
type ChangePlanProjectExtractor struct {
	changePlanRepo repository.ChangePlanRepository
}

// NewChangePlanProjectExtractor creates a new change plan project extractor middleware.
func NewChangePlanProjectExtractor(changePlanRepo repository.ChangePlanRepository) *ChangePlanProjectExtractor {
	return &ChangePlanProjectExtractor{
		changePlanRepo: changePlanRepo,
	}
}

// Extract returns middleware that extracts the project_id from the change plan identified by :id parameter.
func (e *ChangePlanProjectExtractor) Extract() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract change plan ID from route parameter
			changePlanIDStr := c.Param("id")
			if changePlanIDStr == "" {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Change plan ID required",
				})
			}

			// Parse change plan ID
			changePlanID, err := uuid.Parse(changePlanIDStr)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Invalid change plan ID format",
				})
			}

			// Get change plan
			changePlan, err := e.changePlanRepo.GetByID(c.Request().Context(), changePlanID)
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

			// Set project_id so RequireProjectPermission can use it
			// Use c.Set() which is checked by RequireProjectPermission middleware
			c.Set("project_id", changePlan.ProjectID.String())

			// Also store the change plan in context for handlers that need it
			ctx := context.WithValue(c.Request().Context(), rbac.ContextKey("change_plan_id"), changePlanID)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}
