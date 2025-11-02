package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/models"
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

			// Extract project ID from route parameter (try both "project_id" and "id")
			projectIDStr := c.Param("project_id")
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

			// Convert models.ProjectRole to rbac.ProjectRole
			projectRole := convertProjectRoleToRBAC(member.Role)
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

// convertProjectRoleToRBAC converts a models.ProjectRole to rbac.ProjectRole
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
		return rbac.ProjectViewer // Default to most restrictive
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
