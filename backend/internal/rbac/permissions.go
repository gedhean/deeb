package rbac

// Permission represents an action that can be performed on a resource.
// Permissions are granted through organization and project roles.
type Permission string

// Organization-level permissions
const (
	// User management
	PermissionUsersCreate Permission = "users.create"
	PermissionUsersRead   Permission = "users.read"
	PermissionUsersUpdate Permission = "users.update"
	PermissionUsersDelete Permission = "users.delete"

	// Organization management
	PermissionOrganizationUpdate Permission = "organization.update"
	PermissionOrganizationDelete Permission = "organization.delete"

	// Database management
	PermissionDatabasesCreate Permission = "databases.create"
	PermissionDatabasesRead   Permission = "databases.read"
	PermissionDatabasesUpdate Permission = "databases.update"
	PermissionDatabasesDelete Permission = "databases.delete"

	// Project management
	PermissionProjectsCreate Permission = "projects.create"
	PermissionProjectsRead   Permission = "projects.read"
	PermissionProjectsUpdate Permission = "projects.update"
	PermissionProjectsDelete Permission = "projects.delete"

	// Audit logs
	PermissionAuditLogsRead Permission = "audit_logs.read"

	// Wildcard permission (admin only)
	PermissionAll Permission = "*"
)

// Project-level permissions
const (
	// Project settings
	PermissionProjectSettingsUpdate Permission = "project.settings.update"

	// Project members
	PermissionProjectMembersCreate Permission = "project.members.create"
	PermissionProjectMembersRead   Permission = "project.members.read"
	PermissionProjectMembersUpdate Permission = "project.members.update"
	PermissionProjectMembersDelete Permission = "project.members.delete"

	// Change requests
	PermissionChangesCreate  Permission = "changes.create"
	PermissionChangesRead    Permission = "changes.read"
	PermissionChangesUpdate  Permission = "changes.update"
	PermissionChangesDelete  Permission = "changes.delete"
	PermissionChangesApprove Permission = "changes.approve"
	PermissionChangesReject  Permission = "changes.reject"
	PermissionChangesExecute Permission = "changes.execute"

	// Policy rules
	PermissionPolicyRulesCreate Permission = "policy_rules.create"
	PermissionPolicyRulesRead   Permission = "policy_rules.read"
	PermissionPolicyRulesUpdate Permission = "policy_rules.update"
	PermissionPolicyRulesDelete Permission = "policy_rules.delete"
)

// organizationRolePermissions maps organization roles to their permissions
var organizationRolePermissions = map[OrganizationRole][]Permission{
	OrganizationAdmin: {
		PermissionAll, // Admin has all permissions
	},
	OrganizationDBA: {
		// User management (read only)
		PermissionUsersRead,

		// Database management (full access)
		PermissionDatabasesCreate,
		PermissionDatabasesRead,
		PermissionDatabasesUpdate,
		PermissionDatabasesDelete,

		// Project management (read only)
		PermissionProjectsRead,

		// Audit logs
		PermissionAuditLogsRead,
	},
	OrganizationDeveloper: {
		// User management (read only)
		PermissionUsersRead,

		// Database management (read only)
		PermissionDatabasesRead,

		// Project management (create and read)
		PermissionProjectsCreate,
		PermissionProjectsRead,

		// Audit logs (read own)
		PermissionAuditLogsRead,
	},
}

// projectRolePermissions maps project roles to their permissions
var projectRolePermissions = map[ProjectRole][]Permission{
	ProjectOwner: {
		// Project settings
		PermissionProjectSettingsUpdate,

		// Project members (full access)
		PermissionProjectMembersCreate,
		PermissionProjectMembersRead,
		PermissionProjectMembersUpdate,
		PermissionProjectMembersDelete,

		// Change requests (full access)
		PermissionChangesCreate,
		PermissionChangesRead,
		PermissionChangesUpdate,
		PermissionChangesDelete,
		PermissionChangesApprove,
		PermissionChangesReject,
		PermissionChangesExecute,

		// Policy rules (full access)
		PermissionPolicyRulesCreate,
		PermissionPolicyRulesRead,
		PermissionPolicyRulesUpdate,
		PermissionPolicyRulesDelete,
	},
	ProjectDeveloper: {
		// Project members (read only)
		PermissionProjectMembersRead,

		// Change requests (create, read, update own)
		PermissionChangesCreate,
		PermissionChangesRead,
		PermissionChangesUpdate,
		PermissionChangesDelete, // Can delete own requests

		// Policy rules (read only)
		PermissionPolicyRulesRead,
	},
	ProjectApprover: {
		// Project members (read only)
		PermissionProjectMembersRead,

		// Change requests (read, approve, reject)
		PermissionChangesRead,
		PermissionChangesApprove,
		PermissionChangesReject,

		// Policy rules (read only)
		PermissionPolicyRulesRead,
	},
	ProjectViewer: {
		// Project members (read only)
		PermissionProjectMembersRead,

		// Change requests (read only)
		PermissionChangesRead,

		// Policy rules (read only)
		PermissionPolicyRulesRead,
	},
}

// GetOrganizationPermissions returns all permissions for a given organization role.
// Returns an empty slice if the role is not recognized.
func GetOrganizationPermissions(role OrganizationRole) []Permission {
	permissions, exists := organizationRolePermissions[role]
	if !exists {
		return []Permission{}
	}
	return permissions
}

// GetProjectPermissions returns all permissions for a given project role.
// Returns an empty slice if the role is not recognized.
func GetProjectPermissions(role ProjectRole) []Permission {
	permissions, exists := projectRolePermissions[role]
	if !exists {
		return []Permission{}
	}
	return permissions
}

// hasPermission checks if a permission is in the given list
func hasPermission(permissions []Permission, permission Permission) bool {
	// Check for wildcard permission (admin)
	for _, p := range permissions {
		if p == PermissionAll {
			return true
		}
		if p == permission {
			return true
		}
	}
	return false
}
