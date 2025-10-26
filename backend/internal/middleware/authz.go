package middleware

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/rbac"
)

// AuthzMiddleware provides authorization middleware functions.
// It checks user permissions at both organization and project levels.
type AuthzMiddleware struct {
	checker *rbac.CheckerService
}

// NewAuthzMiddleware creates a new authorization middleware with an RBAC checker service.
func NewAuthzMiddleware() *AuthzMiddleware {
	return &AuthzMiddleware{
		checker: rbac.NewCheckerService(),
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
// Organization admins bypass project-level checks. Currently returns 501 Not Implemented
// pending database integration for project role lookup.
func (m *AuthzMiddleware) RequireProjectPermission(permission rbac.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract user information from context
			userID, organizationID, organizationRole, ok := rbac.GetUserFromContext(c.Request().Context())
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]interface{}{
					"error": "Authentication required",
				})
			}

			// Extract project ID from route parameter
			projectID := c.Param("project_id")
			if projectID == "" {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Project ID required",
				})
			}

			// Organization admins bypass project-level checks
			if organizationRole == rbac.OrganizationAdmin {
				// Add project context for downstream handlers
				ctx := context.WithValue(c.Request().Context(), rbac.ContextKeyProjectID, projectID)
				ctx = context.WithValue(ctx, rbac.ContextKeyProjectRole, rbac.ProjectOwner)
				c.SetRequest(c.Request().WithContext(ctx))
				return next(c)
			}

			// TODO: Fetch user's project role from database
			// For now, this is a placeholder that would need to be replaced with
			// actual database lookup from project_members table
			//
			// Example implementation:
			// projectRole, err := m.getProjectRole(c.Request().Context(), userID, projectID)
			// if err != nil {
			//     return c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": "Failed to check permissions"})
			// }
			//
			// if projectRole == "" {
			//     return c.JSON(http.StatusForbidden, map[string]interface{}{"error": "Not a member of this project"})
			// }

			// PLACEHOLDER: For testing purposes, we'll return 501 Not Implemented
			// This will be replaced once we have database integration
			return c.JSON(http.StatusNotImplemented, map[string]interface{}{
				"error": "Project permission checking not yet implemented",
				"details": map[string]interface{}{
					"user_id":             userID,
					"organization_id":     organizationID,
					"project_id":          projectID,
					"required_permission": permission,
					"note":                "Database integration required for project role lookup",
				},
			})
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
