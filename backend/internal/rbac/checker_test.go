package rbac

import (
	"testing"
)

func TestHasOrganizationPermission(t *testing.T) {
	checker := NewCheckerService()

	tests := []struct {
		name       string
		role       OrganizationRole
		permission Permission
		expected   bool
	}{
		// OrganizationAdmin tests
		{
			name:       "admin has wildcard permission",
			role:       OrganizationAdmin,
			permission: PermissionAll,
			expected:   true,
		},
		{
			name:       "admin has users.create",
			role:       OrganizationAdmin,
			permission: PermissionUsersCreate,
			expected:   true,
		},
		{
			name:       "admin has databases.delete",
			role:       OrganizationAdmin,
			permission: PermissionDatabasesDelete,
			expected:   true,
		},

		// OrganizationDBA tests
		{
			name:       "dba has databases.create",
			role:       OrganizationDBA,
			permission: PermissionDatabasesCreate,
			expected:   true,
		},
		{
			name:       "dba has databases.delete",
			role:       OrganizationDBA,
			permission: PermissionDatabasesDelete,
			expected:   true,
		},
		{
			name:       "dba does not have users.create",
			role:       OrganizationDBA,
			permission: PermissionUsersCreate,
			expected:   false,
		},
		{
			name:       "dba has users.read",
			role:       OrganizationDBA,
			permission: PermissionUsersRead,
			expected:   true,
		},

		// OrganizationDeveloper tests
		{
			name:       "developer has projects.create",
			role:       OrganizationDeveloper,
			permission: PermissionProjectsCreate,
			expected:   true,
		},
		{
			name:       "developer has databases.read",
			role:       OrganizationDeveloper,
			permission: PermissionDatabasesRead,
			expected:   true,
		},
		{
			name:       "developer does not have databases.create",
			role:       OrganizationDeveloper,
			permission: PermissionDatabasesCreate,
			expected:   false,
		},
		{
			name:       "developer does not have users.delete",
			role:       OrganizationDeveloper,
			permission: PermissionUsersDelete,
			expected:   false,
		},

		// Invalid role
		{
			name:       "invalid role has no permissions",
			role:       OrganizationRole("invalid"),
			permission: PermissionUsersRead,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.HasOrganizationPermission(tt.role, tt.permission)
			if result != tt.expected {
				t.Errorf("HasOrganizationPermission(%s, %s) = %v, expected %v",
					tt.role, tt.permission, result, tt.expected)
			}
		})
	}
}

func TestHasProjectPermission(t *testing.T) {
	checker := NewCheckerService()

	tests := []struct {
		name       string
		role       ProjectRole
		permission Permission
		expected   bool
	}{
		// ProjectOwner tests
		{
			name:       "owner has changes.approve",
			role:       ProjectOwner,
			permission: PermissionChangesApprove,
			expected:   true,
		},
		{
			name:       "owner has project.members.delete",
			role:       ProjectOwner,
			permission: PermissionProjectMembersDelete,
			expected:   true,
		},
		{
			name:       "owner has policy_rules.create",
			role:       ProjectOwner,
			permission: PermissionPolicyRulesCreate,
			expected:   true,
		},

		// ProjectDeveloper tests
		{
			name:       "developer has changes.create",
			role:       ProjectDeveloper,
			permission: PermissionChangesCreate,
			expected:   true,
		},
		{
			name:       "developer has changes.read",
			role:       ProjectDeveloper,
			permission: PermissionChangesRead,
			expected:   true,
		},
		{
			name:       "developer does not have changes.approve",
			role:       ProjectDeveloper,
			permission: PermissionChangesApprove,
			expected:   false,
		},
		{
			name:       "developer has policy_rules.read",
			role:       ProjectDeveloper,
			permission: PermissionPolicyRulesRead,
			expected:   true,
		},

		// ProjectApprover tests
		{
			name:       "approver has changes.approve",
			role:       ProjectApprover,
			permission: PermissionChangesApprove,
			expected:   true,
		},
		{
			name:       "approver has changes.reject",
			role:       ProjectApprover,
			permission: PermissionChangesReject,
			expected:   true,
		},
		{
			name:       "approver does not have changes.create",
			role:       ProjectApprover,
			permission: PermissionChangesCreate,
			expected:   false,
		},
		{
			name:       "approver does not have changes.delete",
			role:       ProjectApprover,
			permission: PermissionChangesDelete,
			expected:   false,
		},

		// ProjectViewer tests
		{
			name:       "viewer has changes.read",
			role:       ProjectViewer,
			permission: PermissionChangesRead,
			expected:   true,
		},
		{
			name:       "viewer has project.members.read",
			role:       ProjectViewer,
			permission: PermissionProjectMembersRead,
			expected:   true,
		},
		{
			name:       "viewer does not have changes.create",
			role:       ProjectViewer,
			permission: PermissionChangesCreate,
			expected:   false,
		},
		{
			name:       "viewer does not have changes.approve",
			role:       ProjectViewer,
			permission: PermissionChangesApprove,
			expected:   false,
		},

		// Invalid role
		{
			name:       "invalid role has no permissions",
			role:       ProjectRole("invalid"),
			permission: PermissionChangesRead,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.HasProjectPermission(tt.role, tt.permission)
			if result != tt.expected {
				t.Errorf("HasProjectPermission(%s, %s) = %v, expected %v",
					tt.role, tt.permission, result, tt.expected)
			}
		})
	}
}

func TestRequireOrganizationPermission(t *testing.T) {
	checker := NewCheckerService()

	tests := []struct {
		name        string
		role        OrganizationRole
		permission  Permission
		expectError bool
	}{
		{
			name:        "admin passes permission check",
			role:        OrganizationAdmin,
			permission:  PermissionUsersCreate,
			expectError: false,
		},
		{
			name:        "dba passes database permission check",
			role:        OrganizationDBA,
			permission:  PermissionDatabasesCreate,
			expectError: false,
		},
		{
			name:        "developer fails user create check",
			role:        OrganizationDeveloper,
			permission:  PermissionUsersCreate,
			expectError: true,
		},
		{
			name:        "invalid role fails check",
			role:        OrganizationRole("invalid"),
			permission:  PermissionUsersRead,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.RequireOrganizationPermission(tt.role, tt.permission)
			if tt.expectError && err == nil {
				t.Errorf("RequireOrganizationPermission(%s, %s) expected error, got nil",
					tt.role, tt.permission)
			}
			if !tt.expectError && err != nil {
				t.Errorf("RequireOrganizationPermission(%s, %s) expected no error, got %v",
					tt.role, tt.permission, err)
			}
		})
	}
}

func TestRequireProjectPermission(t *testing.T) {
	checker := NewCheckerService()

	tests := []struct {
		name        string
		role        ProjectRole
		permission  Permission
		expectError bool
	}{
		{
			name:        "owner passes permission check",
			role:        ProjectOwner,
			permission:  PermissionChangesApprove,
			expectError: false,
		},
		{
			name:        "developer passes create check",
			role:        ProjectDeveloper,
			permission:  PermissionChangesCreate,
			expectError: false,
		},
		{
			name:        "viewer fails create check",
			role:        ProjectViewer,
			permission:  PermissionChangesCreate,
			expectError: true,
		},
		{
			name:        "invalid role fails check",
			role:        ProjectRole("invalid"),
			permission:  PermissionChangesRead,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.RequireProjectPermission(tt.role, tt.permission)
			if tt.expectError && err == nil {
				t.Errorf("RequireProjectPermission(%s, %s) expected error, got nil",
					tt.role, tt.permission)
			}
			if !tt.expectError && err != nil {
				t.Errorf("RequireProjectPermission(%s, %s) expected no error, got %v",
					tt.role, tt.permission, err)
			}
		})
	}
}

func TestIsValidOrganizationRole(t *testing.T) {
	tests := []struct {
		role     OrganizationRole
		expected bool
	}{
		{OrganizationAdmin, true},
		{OrganizationDBA, true},
		{OrganizationDeveloper, true},
		{OrganizationRole("invalid"), false},
		{OrganizationRole(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			result := IsValidOrganizationRole(tt.role)
			if result != tt.expected {
				t.Errorf("IsValidOrganizationRole(%s) = %v, expected %v",
					tt.role, result, tt.expected)
			}
		})
	}
}

func TestIsValidProjectRole(t *testing.T) {
	tests := []struct {
		role     ProjectRole
		expected bool
	}{
		{ProjectOwner, true},
		{ProjectDeveloper, true},
		{ProjectApprover, true},
		{ProjectViewer, true},
		{ProjectRole("invalid"), false},
		{ProjectRole(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			result := IsValidProjectRole(tt.role)
			if result != tt.expected {
				t.Errorf("IsValidProjectRole(%s) = %v, expected %v",
					tt.role, result, tt.expected)
			}
		})
	}
}
