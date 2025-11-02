package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
)

// ExecutionAuthMiddleware provides authorization middleware for change plan execution.
// It checks if the authenticated user has permission to execute a specific change plan.
type ExecutionAuthMiddleware struct {
	checker           *rbac.CheckerService
	changePlanRepo    repository.ChangePlanRepository
	projectMemberRepo repository.ProjectMemberRepository
}

// NewExecutionAuthMiddleware creates a new execution authorization middleware.
func NewExecutionAuthMiddleware(
	changePlanRepo repository.ChangePlanRepository,
	projectMemberRepo repository.ProjectMemberRepository,
) *ExecutionAuthMiddleware {
	return &ExecutionAuthMiddleware{
		checker:           rbac.NewCheckerService(),
		changePlanRepo:    changePlanRepo,
		projectMemberRepo: projectMemberRepo,
	}
}

// RequireExecutionPermission returns middleware that checks if the authenticated user
// has permission to execute the change plan identified by the "id" route parameter.
// Permission is granted if the user is:
// - Organization Admin
// - Organization DBA
// - Project Owner of the change plan's project
// - Change Plan Author
// - Project member with PermissionChangesExecute or PermissionChangesApprove
// Returns 401 Unauthorized if user is not authenticated, 403 Forbidden if lacking permission.
func (m *ExecutionAuthMiddleware) RequireExecutionPermission() echo.MiddlewareFunc {
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
			changePlan, err := m.changePlanRepo.GetByID(c.Request().Context(), changePlanID)
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

			// Check organization-level permissions (Org Admin, Org DBA can execute)
			hasOrgPermission := m.checker.HasOrganizationPermission(organizationRole, rbac.PermissionChangesExecute) ||
				m.checker.HasOrganizationPermission(organizationRole, rbac.PermissionChangesApprove) ||
				organizationRole == rbac.OrganizationDBA // Explicitly allow Organization DBA

			if !hasOrgPermission {
				// Check if user is the author
				if changePlan.AuthorID != userID {
					// Check project-level permissions
					member, err := m.projectMemberRepo.GetMemberRole(c.Request().Context(), changePlan.ProjectID, userID)
					if err != nil {
						if err == repository.ErrNotFound {
							return c.JSON(http.StatusForbidden, map[string]interface{}{
								"error": "Not a member of this project",
							})
						}
						return c.JSON(http.StatusInternalServerError, map[string]interface{}{
							"error": "Failed to check project membership",
						})
					}

					// Check if user has Project Owner role or execution permission
					projectRole := convertProjectRoleToRBAC(member.Role)
					if projectRole != rbac.ProjectOwner &&
						!m.checker.HasProjectPermission(projectRole, rbac.PermissionChangesExecute) &&
						!m.checker.HasProjectPermission(projectRole, rbac.PermissionChangesApprove) {
						return c.JSON(http.StatusForbidden, map[string]interface{}{
							"error": "Insufficient permissions to execute change plans. Must be Organization Admin, Organization DBA, Project Owner, or Change Plan Author",
						})
					}
				}
			}

			return next(c)
		}
	}
}

