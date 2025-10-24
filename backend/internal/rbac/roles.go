package rbac

// OrganizationRole represents a user's role at the organization level
type OrganizationRole string

const (
	// OrganizationAdmin has full system access and can manage all resources
	OrganizationAdmin OrganizationRole = "organization_admin"

	// OrganizationDBA can manage databases and approve high-risk changes
	OrganizationDBA OrganizationRole = "organization_dba"

	// OrganizationDeveloper is the default member role with basic access
	OrganizationDeveloper OrganizationRole = "organization_developer"
)

// ProjectRole represents a user's role within a specific project
type ProjectRole string

const (
	// ProjectOwner has full control over the project and its resources
	ProjectOwner ProjectRole = "project_owner"

	// ProjectDeveloper can create and manage change requests
	ProjectDeveloper ProjectRole = "project_developer"

	// ProjectApprover can review and approve/reject change requests
	ProjectApprover ProjectRole = "project_approver"

	// ProjectViewer has read-only access to project resources
	ProjectViewer ProjectRole = "project_viewer"
)

// IsValidOrganizationRole checks if the given role is a valid organization role
func IsValidOrganizationRole(role OrganizationRole) bool {
	switch role {
	case OrganizationAdmin, OrganizationDBA, OrganizationDeveloper:
		return true
	default:
		return false
	}
}

// IsValidProjectRole checks if the given role is a valid project role
func IsValidProjectRole(role ProjectRole) bool {
	switch role {
	case ProjectOwner, ProjectDeveloper, ProjectApprover, ProjectViewer:
		return true
	default:
		return false
	}
}
