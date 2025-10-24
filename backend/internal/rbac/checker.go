package rbac

import (
	"context"
	"fmt"
)

// CheckerService provides permission checking functionality
type CheckerService struct{}

// NewCheckerService creates a new permission checker service
func NewCheckerService() *CheckerService {
	return &CheckerService{}
}

// HasOrganizationPermission checks if a user with the given organization role
// has the specified permission at the organization level
func (s *CheckerService) HasOrganizationPermission(role OrganizationRole, permission Permission) bool {
	if !IsValidOrganizationRole(role) {
		return false
	}

	permissions := GetOrganizationPermissions(role)
	return hasPermission(permissions, permission)
}

// HasProjectPermission checks if a user with the given project role
// has the specified permission at the project level
func (s *CheckerService) HasProjectPermission(role ProjectRole, permission Permission) bool {
	if !IsValidProjectRole(role) {
		return false
	}

	permissions := GetProjectPermissions(role)
	return hasPermission(permissions, permission)
}

// RequireOrganizationPermission returns an error if the user does not have
// the specified organization-level permission
func (s *CheckerService) RequireOrganizationPermission(role OrganizationRole, permission Permission) error {
	if !s.HasOrganizationPermission(role, permission) {
		return fmt.Errorf("insufficient permissions: role '%s' does not have permission '%s'", role, permission)
	}
	return nil
}

// RequireProjectPermission returns an error if the user does not have
// the specified project-level permission
func (s *CheckerService) RequireProjectPermission(role ProjectRole, permission Permission) error {
	if !s.HasProjectPermission(role, permission) {
		return fmt.Errorf("insufficient permissions: role '%s' does not have permission '%s'", role, permission)
	}
	return nil
}

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// ContextKeyUserID is the context key for user ID
	ContextKeyUserID ContextKey = "user_id"

	// ContextKeyOrganizationID is the context key for organization ID
	ContextKeyOrganizationID ContextKey = "organization_id"

	// ContextKeyOrganizationRole is the context key for organization role
	ContextKeyOrganizationRole ContextKey = "organization_role"

	// ContextKeyProjectID is the context key for project ID (when in project context)
	ContextKeyProjectID ContextKey = "project_id"

	// ContextKeyProjectRole is the context key for project role (when in project context)
	ContextKeyProjectRole ContextKey = "project_role"
)

// GetUserFromContext extracts user information from the request context
func GetUserFromContext(ctx context.Context) (userID, organizationID string, organizationRole OrganizationRole, ok bool) {
	userID, ok = ctx.Value(ContextKeyUserID).(string)
	if !ok || userID == "" {
		return "", "", "", false
	}

	organizationID, ok = ctx.Value(ContextKeyOrganizationID).(string)
	if !ok || organizationID == "" {
		return "", "", "", false
	}

	organizationRole, ok = ctx.Value(ContextKeyOrganizationRole).(OrganizationRole)
	if !ok {
		return "", "", "", false
	}

	return userID, organizationID, organizationRole, true
}

// GetProjectRoleFromContext extracts project role from the request context
func GetProjectRoleFromContext(ctx context.Context) (projectID string, projectRole ProjectRole, ok bool) {
	projectID, ok = ctx.Value(ContextKeyProjectID).(string)
	if !ok || projectID == "" {
		return "", "", false
	}

	projectRole, ok = ctx.Value(ContextKeyProjectRole).(ProjectRole)
	if !ok {
		return "", "", false
	}

	return projectID, projectRole, true
}
