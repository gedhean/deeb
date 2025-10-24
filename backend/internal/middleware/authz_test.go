package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/rbac"
)

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
	tests := []struct {
		name           string
		permission     rbac.Permission
		userRole       rbac.OrganizationRole
		projectID      string
		setContext     bool
		expectedStatus int
	}{
		{
			name:           "admin bypasses project check",
			permission:     rbac.PermissionChangesApprove,
			userRole:       rbac.OrganizationAdmin,
			projectID:      "test-project",
			setContext:     true,
			expectedStatus: http.StatusOK, // Admin bypasses project checks
		},
		{
			name:           "no context returns unauthorized",
			permission:     rbac.PermissionChangesRead,
			userRole:       rbac.OrganizationDeveloper,
			projectID:      "test-project",
			setContext:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing project ID returns bad request",
			permission:     rbac.PermissionChangesRead,
			userRole:       rbac.OrganizationDeveloper,
			projectID:      "",
			setContext:     true,
			expectedStatus: http.StatusBadRequest,
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
			if tt.projectID != "" {
				c.SetParamNames("project_id")
				c.SetParamValues(tt.projectID)
			}

			middleware := NewAuthzMiddleware()
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
