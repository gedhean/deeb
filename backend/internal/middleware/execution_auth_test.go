package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChangePlanRepository is a mock implementation of ChangePlanRepository for testing
type mockChangePlanRepository struct {
	changePlans map[uuid.UUID]*models.ChangePlan
	getErr      error
}

func (m *mockChangePlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ChangePlan, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	cp, ok := m.changePlans[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return cp, nil
}

func (m *mockChangePlanRepository) Create(ctx context.Context, changePlan *models.ChangePlan) error {
	if m.changePlans == nil {
		m.changePlans = make(map[uuid.UUID]*models.ChangePlan)
	}
	m.changePlans[changePlan.ID] = changePlan
	return nil
}

func (m *mockChangePlanRepository) Update(ctx context.Context, changePlan *models.ChangePlan) error {
	if _, ok := m.changePlans[changePlan.ID]; !ok {
		return repository.ErrNotFound
	}
	m.changePlans[changePlan.ID] = changePlan
	return nil
}

func (m *mockChangePlanRepository) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.changePlans, id)
	return nil
}

func (m *mockChangePlanRepository) List(ctx context.Context, limit, offset int) ([]*models.ChangePlan, error) {
	// Not used in tests
	return nil, nil
}

func (m *mockChangePlanRepository) ListWithFilters(
	ctx context.Context,
	filters repository.ChangePlanListFilters,
) ([]*models.ChangePlan, error) {
	// Not used in tests
	return nil, nil
}

func (m *mockChangePlanRepository) Count(ctx context.Context) (int64, error) {
	// Not used in tests
	return 0, nil
}

func (m *mockChangePlanRepository) CountWithFilters(
	ctx context.Context,
	filters repository.ChangePlanListFilters,
) (int64, error) {
	// Not used in tests
	return 0, nil
}

func (m *mockChangePlanRepository) GetByStatus(
	ctx context.Context,
	status models.ChangePlanStatus,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	// Not used in tests
	return nil, nil
}

func (m *mockChangePlanRepository) GetByProjectID(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	// Not used in tests
	return nil, nil
}

func (m *mockChangePlanRepository) GetByDatabaseID(
	ctx context.Context,
	databaseID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	// Not used in tests
	return nil, nil
}

func (m *mockChangePlanRepository) GetByAuthorID(
	ctx context.Context,
	authorID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	// Not used in tests
	return nil, nil
}

func TestRequireExecutionPermission(t *testing.T) {
	// Setup test data
	projectID := uuid.New()
	authorID := uuid.New()
	orgAdminID := uuid.New()
	orgDBAID := uuid.New()
	projectOwnerID := uuid.New()
	projectDeveloperID := uuid.New()
	projectApproverID := uuid.New()
	unauthorizedUserID := uuid.New()
	changePlanID := uuid.New()

	changePlan := &models.ChangePlan{
		ID:         changePlanID,
		ProjectID:  projectID,
		AuthorID:   authorID,
		Status:     models.ChangePlanStatusApproved,
		SQLContent: "CREATE TABLE test (id UUID PRIMARY KEY);",
	}

	changePlanRepo := &mockChangePlanRepository{
		changePlans: map[uuid.UUID]*models.ChangePlan{
			changePlanID: changePlan,
		},
	}

	projectMemberRepo := &mockProjectMemberRepository{
		members: map[string]*models.ProjectMember{
			projectID.String() + "-" + projectOwnerID.String(): {
				ProjectID: projectID,
				UserID:    projectOwnerID,
				Role:      models.ProjectRoleOwner,
			},
			projectID.String() + "-" + projectDeveloperID.String(): {
				ProjectID: projectID,
				UserID:    projectDeveloperID,
				Role:      models.ProjectRoleDeveloper,
			},
			projectID.String() + "-" + projectApproverID.String(): {
				ProjectID: projectID,
				UserID:    projectApproverID,
				Role:      models.ProjectRoleApprover,
			},
		},
	}

	middleware := NewExecutionAuthMiddleware(changePlanRepo, projectMemberRepo)

	handler := func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "authorized",
		})
	}

	tests := []struct {
		name           string
		userID         uuid.UUID
		userRole       rbac.OrganizationRole
		changePlanID   uuid.UUID
		setContext     bool
		expectedStatus int
		expectedError  string
		setupMocks     func()
	}{
		{
			name:           "organization admin can execute",
			userID:         orgAdminID,
			userRole:       rbac.OrganizationAdmin,
			changePlanID:   changePlanID,
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "organization DBA can execute",
			userID:         orgDBAID,
			userRole:       rbac.OrganizationDBA,
			changePlanID:   changePlanID,
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "change plan author can execute",
			userID:         authorID,
			userRole:       rbac.OrganizationDeveloper,
			changePlanID:   changePlanID,
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "project owner can execute",
			userID:         projectOwnerID,
			userRole:       rbac.OrganizationDeveloper,
			changePlanID:   changePlanID,
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "project approver can execute",
			userID:         projectApproverID,
			userRole:       rbac.OrganizationDeveloper,
			changePlanID:   changePlanID,
			setContext:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "project developer cannot execute without permission",
			userID:         projectDeveloperID,
			userRole:       rbac.OrganizationDeveloper,
			changePlanID:   changePlanID,
			setContext:     true,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Insufficient permissions",
		},
		{
			name:           "user not in project cannot execute",
			userID:         unauthorizedUserID,
			userRole:       rbac.OrganizationDeveloper,
			changePlanID:   changePlanID,
			setContext:     true,
			expectedStatus: http.StatusForbidden,
			expectedError:  "Not a member of this project",
		},
		{
			name:           "unauthenticated user cannot execute",
			userID:         uuid.New(),
			userRole:       rbac.OrganizationDeveloper,
			changePlanID:   changePlanID,
			setContext:     false,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Authentication required",
		},
		{
			name:           "invalid change plan ID",
			userID:         orgAdminID,
			userRole:       rbac.OrganizationAdmin,
			changePlanID:   uuid.Nil,
			setContext:     true,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Change plan ID required",
		},
		{
			name:           "change plan not found",
			userID:         orgAdminID,
			userRole:       rbac.OrganizationAdmin,
			changePlanID:   uuid.New(),
			setContext:     true,
			expectedStatus: http.StatusNotFound,
			expectedError:  "Change plan not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans/:id/execute", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/v1/change-plans/:id/execute")
			c.SetParamNames("id")
			if tt.changePlanID != uuid.Nil {
				c.SetParamValues(tt.changePlanID.String())
			} else {
				c.SetParamValues("")
			}

			// Set user context if needed
			if tt.setContext {
				ctx := context.WithValue(req.Context(), rbac.ContextKeyUserID, tt.userID.String())
				ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationID, uuid.New().String())
				ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationRole, tt.userRole)
				c.SetRequest(req.WithContext(ctx))
			}

			// Apply middleware
			mw := middleware.RequireExecutionPermission()
			err := mw(handler)(c)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rec.Code)

			// Check error message if expected
			if tt.expectedError != "" {
				if err != nil {
					he, ok := err.(*echo.HTTPError)
					if ok {
						assert.Contains(t, he.Message.(string), tt.expectedError)
					} else {
						assert.Contains(t, err.Error(), tt.expectedError)
					}
				} else {
					var response map[string]interface{}
					err := json.Unmarshal(rec.Body.Bytes(), &response)
					require.NoError(t, err)
					assert.Contains(t, response["error"].(string), tt.expectedError)
				}
			} else {
				// Should succeed
				require.NoError(t, err)
			}
		})
	}
}

func TestNewExecutionAuthMiddleware(t *testing.T) {
	changePlanRepo := &mockChangePlanRepository{}
	projectMemberRepo := &mockProjectMemberRepository{}

	middleware := NewExecutionAuthMiddleware(changePlanRepo, projectMemberRepo)

	assert.NotNil(t, middleware)
	assert.Equal(t, changePlanRepo, middleware.changePlanRepo)
	assert.Equal(t, projectMemberRepo, middleware.projectMemberRepo)
	assert.NotNil(t, middleware.checker)
}
