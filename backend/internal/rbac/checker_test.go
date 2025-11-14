package rbac

import (
	"context"
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
			name:       "developer has databases.create",
			role:       OrganizationDeveloper,
			permission: PermissionDatabasesCreate,
			expected:   true,
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
			name:       "owner does not have policy_rules.create (org-level only)",
			role:       ProjectOwner,
			permission: PermissionPolicyRulesCreate,
			expected:   false,
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

func TestGetOrganizationPermissions(t *testing.T) {
	tests := []struct {
		name     string
		role     OrganizationRole
		expected int // number of permissions
	}{
		{
			name:     "admin has wildcard permission",
			role:     OrganizationAdmin,
			expected: 1, // Only PermissionAll
		},
		{
			name:     "dba has multiple permissions",
			role:     OrganizationDBA,
			expected: 15, // 15 specific permissions (users read, database instances CRUD, databases CRUD, projects CRUD, environments read, audit logs read)
		},
		{
			name:     "developer has basic permissions",
			role:     OrganizationDeveloper,
			expected: 8, // 8 basic permissions (users read, database instances read, databases read/create/update, projects create/read, environments read)
		},
		{
			name:     "invalid role has no permissions",
			role:     OrganizationRole("invalid"),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := GetOrganizationPermissions(tt.role)
			if len(perms) != tt.expected {
				t.Errorf("GetOrganizationPermissions(%s) returned %d permissions, expected %d",
					tt.role, len(perms), tt.expected)
			}
		})
	}
}

func TestGetProjectPermissions(t *testing.T) {
	tests := []struct {
		name     string
		role     ProjectRole
		expected int // number of permissions
	}{
		{
			name:     "owner has full permissions",
			role:     ProjectOwner,
			expected: 12, // Full access to project settings, members, and changes (policy rules are org-level)
		},
		{
			name:     "developer has create permissions",
			role:     ProjectDeveloper,
			expected: 6, // Can create and manage own changes
		},
		{
			name:     "approver has review permissions",
			role:     ProjectApprover,
			expected: 5, // Can review and approve/reject
		},
		{
			name:     "viewer has read-only permissions",
			role:     ProjectViewer,
			expected: 3, // Read-only access
		},
		{
			name:     "invalid role has no permissions",
			role:     ProjectRole("invalid"),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := GetProjectPermissions(tt.role)
			if len(perms) != tt.expected {
				t.Errorf("GetProjectPermissions(%s) returned %d permissions, expected %d",
					tt.role, len(perms), tt.expected)
			}
		})
	}
}

func TestGetUserFromContext(t *testing.T) {
	tests := []struct {
		name            string
		setupContext    func() context.Context
		expectOK        bool
		expectedUserID  string
		expectedOrgID   string
		expectedOrgRole OrganizationRole
	}{
		{
			name: "valid context with all fields",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyUserID, "user-123")
				ctx = context.WithValue(ctx, ContextKeyOrganizationID, "org-456")
				ctx = context.WithValue(ctx, ContextKeyOrganizationRole, OrganizationDeveloper)
				return ctx
			},
			expectOK:        true,
			expectedUserID:  "user-123",
			expectedOrgID:   "org-456",
			expectedOrgRole: OrganizationDeveloper,
		},
		{
			name: "missing user ID",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyOrganizationID, "org-456")
				ctx = context.WithValue(ctx, ContextKeyOrganizationRole, OrganizationDeveloper)
				return ctx
			},
			expectOK: false,
		},
		{
			name: "empty user ID",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyUserID, "")
				ctx = context.WithValue(ctx, ContextKeyOrganizationID, "org-456")
				ctx = context.WithValue(ctx, ContextKeyOrganizationRole, OrganizationDeveloper)
				return ctx
			},
			expectOK: false,
		},
		{
			name: "missing organization ID",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyUserID, "user-123")
				ctx = context.WithValue(ctx, ContextKeyOrganizationRole, OrganizationDeveloper)
				return ctx
			},
			expectOK: false,
		},
		{
			name: "empty organization ID",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyUserID, "user-123")
				ctx = context.WithValue(ctx, ContextKeyOrganizationID, "")
				ctx = context.WithValue(ctx, ContextKeyOrganizationRole, OrganizationDeveloper)
				return ctx
			},
			expectOK: false,
		},
		{
			name: "missing organization role",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyUserID, "user-123")
				ctx = context.WithValue(ctx, ContextKeyOrganizationID, "org-456")
				return ctx
			},
			expectOK: false,
		},
		{
			name: "empty context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectOK: false,
		},
		{
			name: "wrong type for user ID",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyUserID, 123) // int instead of string
				ctx = context.WithValue(ctx, ContextKeyOrganizationID, "org-456")
				ctx = context.WithValue(ctx, ContextKeyOrganizationRole, OrganizationDeveloper)
				return ctx
			},
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			userID, orgID, orgRole, ok := GetUserFromContext(ctx)

			if ok != tt.expectOK {
				t.Errorf("GetUserFromContext() ok = %v, expected %v", ok, tt.expectOK)
			}

			if tt.expectOK {
				if userID != tt.expectedUserID {
					t.Errorf("GetUserFromContext() userID = %s, expected %s", userID, tt.expectedUserID)
				}
				if orgID != tt.expectedOrgID {
					t.Errorf("GetUserFromContext() orgID = %s, expected %s", orgID, tt.expectedOrgID)
				}
				if orgRole != tt.expectedOrgRole {
					t.Errorf("GetUserFromContext() orgRole = %s, expected %s", orgRole, tt.expectedOrgRole)
				}
			}
		})
	}
}

func TestGetProjectRoleFromContext(t *testing.T) {
	tests := []struct {
		name                string
		setupContext        func() context.Context
		expectOK            bool
		expectedProjectID   string
		expectedProjectRole ProjectRole
	}{
		{
			name: "valid context with project role",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyProjectID, "project-789")
				ctx = context.WithValue(ctx, ContextKeyProjectRole, ProjectDeveloper)
				return ctx
			},
			expectOK:            true,
			expectedProjectID:   "project-789",
			expectedProjectRole: ProjectDeveloper,
		},
		{
			name: "missing project ID",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyProjectRole, ProjectDeveloper)
				return ctx
			},
			expectOK: false,
		},
		{
			name: "empty project ID",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyProjectID, "")
				ctx = context.WithValue(ctx, ContextKeyProjectRole, ProjectDeveloper)
				return ctx
			},
			expectOK: false,
		},
		{
			name: "missing project role",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyProjectID, "project-789")
				return ctx
			},
			expectOK: false,
		},
		{
			name: "empty context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectOK: false,
		},
		{
			name: "wrong type for project ID",
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = context.WithValue(ctx, ContextKeyProjectID, 789) // int instead of string
				ctx = context.WithValue(ctx, ContextKeyProjectRole, ProjectDeveloper)
				return ctx
			},
			expectOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()
			projectID, projectRole, ok := GetProjectRoleFromContext(ctx)

			if ok != tt.expectOK {
				t.Errorf("GetProjectRoleFromContext() ok = %v, expected %v", ok, tt.expectOK)
			}

			if tt.expectOK {
				if projectID != tt.expectedProjectID {
					t.Errorf("GetProjectRoleFromContext() projectID = %s, expected %s", projectID, tt.expectedProjectID)
				}
				if projectRole != tt.expectedProjectRole {
					t.Errorf("GetProjectRoleFromContext() projectRole = %s, expected %s", projectRole, tt.expectedProjectRole)
				}
			}
		})
	}
}
