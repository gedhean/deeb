package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
)

// mockProjectMemberRepository is a mock implementation of ProjectMemberRepository for testing
type mockProjectMemberRepository struct {
	members map[string]*models.ProjectMember // key: projectID-userID
	getErr  error
}

func (m *mockProjectMemberRepository) GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (*models.ProjectMember, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	key := projectID.String() + "-" + userID.String()
	member, ok := m.members[key]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return member, nil
}

func (m *mockProjectMemberRepository) AddMember(ctx context.Context, member *models.ProjectMember) error {
	key := member.ProjectID.String() + "-" + member.UserID.String()
	if m.members == nil {
		m.members = make(map[string]*models.ProjectMember)
	}
	m.members[key] = member
	return nil
}

func (m *mockProjectMemberRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	key := projectID.String() + "-" + userID.String()
	delete(m.members, key)
	return nil
}

func (m *mockProjectMemberRepository) GetMembers(ctx context.Context, projectID uuid.UUID) ([]*models.ProjectMember, error) {
	var result []*models.ProjectMember
	for _, member := range m.members {
		if member.ProjectID == projectID {
			result = append(result, member)
		}
	}
	return result, nil
}

func (m *mockProjectMemberRepository) UpdateRole(ctx context.Context, projectID, userID uuid.UUID, role models.ProjectRole) error {
	key := projectID.String() + "-" + userID.String()
	if member, ok := m.members[key]; ok {
		member.Role = role
		return nil
	}
	return repository.ErrNotFound
}

func TestRequireOrganizationPermission(t *testing.T) {
	tests := []struct {
		name           string
		permission     rbac.Permission
		userRole       rbac.OrganizationRole
		setContext     bool
		expectedStatus int
	}{
		{
			name:           "admin has users.create permission",
			permission:     rbac.PermissionUsersCreate,
			userRole:       rbac.OrganizationAdmin,
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "dba has databases.create permission",
			permission:     rbac.PermissionDatabasesCreate,
			userRole:       rbac.OrganizationDBA,
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "developer does not have users.delete permission",
			permission:     rbac.PermissionUsersDelete,
			userRole:       rbac.OrganizationDeveloper,
			setContext:     true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no context returns unauthorized",
			permission:     rbac.PermissionUsersRead,
			userRole:       rbac.OrganizationDeveloper,
			setContext:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "developer has projects.create permission",
			permission:     rbac.PermissionProjectsCreate,
			userRole:       rbac.OrganizationDeveloper,
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			if tt.setContext {
				// Add user context
				ctx := context.WithValue(req.Context(), rbac.ContextKeyUserID, "test-user-id")
				ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationID, "test-org-id")
				ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationRole, tt.userRole)
				req = req.WithContext(ctx)
			}

			c := e.NewContext(req, rec)

			// Create middleware and handler
			middleware := NewAuthzMiddleware()
			handler := middleware.RequireOrganizationPermission(tt.permission)(func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			})

			// Execute handler
			_ = handler(c)

			// Verify status
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRequireOrganizationAdmin(t *testing.T) {
	tests := []struct {
		name           string
		userRole       rbac.OrganizationRole
		setContext     bool
		expectedStatus int
	}{
		{
			name:           "admin role allowed",
			userRole:       rbac.OrganizationAdmin,
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "dba role denied",
			userRole:       rbac.OrganizationDBA,
			setContext:     true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "developer role denied",
			userRole:       rbac.OrganizationDeveloper,
			setContext:     true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no context returns unauthorized",
			userRole:       rbac.OrganizationAdmin,
			setContext:     false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			if tt.setContext {
				ctx := context.WithValue(req.Context(), rbac.ContextKeyUserID, "test-user-id")
				ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationID, "test-org-id")
				ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationRole, tt.userRole)
				req = req.WithContext(ctx)
			}

			c := e.NewContext(req, rec)

			middleware := NewAuthzMiddleware()
			handler := middleware.RequireOrganizationAdmin()(func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			})

			_ = handler(c)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRequireOrganizationDBA(t *testing.T) {
	tests := []struct {
		name           string
		userRole       rbac.OrganizationRole
		setContext     bool
		expectedStatus int
	}{
		{
			name:           "admin role allowed",
			userRole:       rbac.OrganizationAdmin,
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "dba role allowed",
			userRole:       rbac.OrganizationDBA,
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "developer role denied",
			userRole:       rbac.OrganizationDeveloper,
			setContext:     true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "no context returns unauthorized",
			userRole:       rbac.OrganizationDBA,
			setContext:     false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			if tt.setContext {
				ctx := context.WithValue(req.Context(), rbac.ContextKeyUserID, "test-user-id")
				ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationID, "test-org-id")
				ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationRole, tt.userRole)
				req = req.WithContext(ctx)
			}

			c := e.NewContext(req, rec)

			middleware := NewAuthzMiddleware()
			handler := middleware.RequireOrganizationDBA()(func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			})

			_ = handler(c)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRequireProjectPermission(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()

	tests := []struct {
		name           string
		permission     rbac.Permission
		userRole       rbac.OrganizationRole
		projectID      string
		paramName      string // "project_id" or "id"
		setContext     bool
		mockMember     *models.ProjectMember
		mockErr        error
		hasRepo        bool
		expectedStatus int
	}{
		{
			name:           "admin bypasses project check",
			permission:     rbac.PermissionChangesApprove,
			userRole:       rbac.OrganizationAdmin,
			projectID:      projectID.String(),
			paramName:      "project_id",
			setContext:     true,
			hasRepo:        false,
			expectedStatus: http.StatusOK, // Admin bypasses project checks
		},
		{
			name:           "admin bypasses with id parameter",
			permission:     rbac.PermissionChangesRead,
			userRole:       rbac.OrganizationAdmin,
			projectID:      projectID.String(),
			paramName:      "id",
			setContext:     true,
			hasRepo:        false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no context returns unauthorized",
			permission:     rbac.PermissionChangesRead,
			userRole:       rbac.OrganizationDeveloper,
			projectID:      projectID.String(),
			paramName:      "project_id",
			setContext:     false,
			hasRepo:        true,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing project ID returns bad request",
			permission:     rbac.PermissionChangesRead,
			userRole:       rbac.OrganizationDeveloper,
			projectID:      "",
			paramName:      "project_id",
			setContext:     true,
			hasRepo:        true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid project ID format returns bad request",
			permission:     rbac.PermissionChangesRead,
			userRole:       rbac.OrganizationDeveloper,
			projectID:      "invalid-uuid",
			paramName:      "project_id",
			setContext:     true,
			hasRepo:        true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "no repository configured returns internal server error",
			permission: rbac.PermissionChangesRead,
			userRole:   rbac.OrganizationDeveloper,
			projectID:  projectID.String(),
			paramName:  "project_id",
			setContext: true,
			hasRepo:    false,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "project owner has permission",
			permission: rbac.PermissionChangesApprove,
			userRole:   rbac.OrganizationDeveloper,
			projectID:  projectID.String(),
			paramName:  "project_id",
			setContext: true,
			hasRepo:    true,
			mockMember: &models.ProjectMember{
				ProjectID: projectID,
				UserID:    userID,
				Role:      models.ProjectRoleOwner,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "project approver has approve permission",
			permission: rbac.PermissionChangesApprove,
			userRole:   rbac.OrganizationDeveloper,
			projectID:  projectID.String(),
			paramName:  "project_id",
			setContext: true,
			hasRepo:    true,
			mockMember: &models.ProjectMember{
				ProjectID: projectID,
				UserID:    userID,
				Role:      models.ProjectRoleApprover,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "project developer does not have approve permission",
			permission: rbac.PermissionChangesApprove,
			userRole:   rbac.OrganizationDeveloper,
			projectID:  projectID.String(),
			paramName:  "project_id",
			setContext: true,
			hasRepo:    true,
			mockMember: &models.ProjectMember{
				ProjectID: projectID,
				UserID:    userID,
				Role:      models.ProjectRoleDeveloper,
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "project viewer does not have approve permission",
			permission: rbac.PermissionChangesApprove,
			userRole:   rbac.OrganizationDeveloper,
			projectID:  projectID.String(),
			paramName:  "project_id",
			setContext: true,
			hasRepo:    true,
			mockMember: &models.ProjectMember{
				ProjectID: projectID,
				UserID:    userID,
				Role:      models.ProjectRoleViewer,
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "not a project member returns forbidden",
			permission: rbac.PermissionChangesRead,
			userRole:   rbac.OrganizationDeveloper,
			projectID:  projectID.String(),
			paramName:  "project_id",
			setContext: true,
			hasRepo:    true,
			mockErr:    repository.ErrNotFound,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "project developer can read changes",
			permission: rbac.PermissionChangesRead,
			userRole:   rbac.OrganizationDeveloper,
			projectID:  projectID.String(),
			paramName:  "project_id",
			setContext: true,
			hasRepo:    true,
			mockMember: &models.ProjectMember{
				ProjectID: projectID,
				UserID:    userID,
				Role:      models.ProjectRoleDeveloper,
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			if tt.setContext {
				ctx := context.WithValue(req.Context(), rbac.ContextKeyUserID, userID.String())
				ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationID, "test-org-id")
				ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationRole, tt.userRole)
				req = req.WithContext(ctx)
			}

			c := e.NewContext(req, rec)
			if tt.projectID != "" {
				c.SetParamNames(tt.paramName)
				c.SetParamValues(tt.projectID)
			}

			var mockRepo *mockProjectMemberRepository
			if tt.hasRepo {
				mockRepo = &mockProjectMemberRepository{
					members: make(map[string]*models.ProjectMember),
					getErr:  tt.mockErr,
				}
				if tt.mockMember != nil {
					key := tt.mockMember.ProjectID.String() + "-" + tt.mockMember.UserID.String()
					mockRepo.members[key] = tt.mockMember
				}
			}

			var middleware *AuthzMiddleware
			if tt.hasRepo {
				middleware = NewAuthzMiddlewareWithRepo(mockRepo)
			} else {
				middleware = NewAuthzMiddleware()
			}

			handler := middleware.RequireProjectPermission(tt.permission)(func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			})

			_ = handler(c)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
