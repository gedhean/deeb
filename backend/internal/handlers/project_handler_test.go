package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// Mock ProjectRepository for testing
type mockProjectRepository struct {
	projects  []*models.Project
	getErr    error
	listErr   error
	createErr error
	updateErr error
	deleteErr error
}

func (m *mockProjectRepository) Create(ctx context.Context, project *models.Project) error {
	if m.createErr != nil {
		return m.createErr
	}
	project.ID = uuid.New()
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()
	m.projects = append(m.projects, project)
	return nil
}

func (m *mockProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, p := range m.projects {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockProjectRepository) List(ctx context.Context, limit, offset int) ([]*models.Project, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	start := offset
	end := offset + limit
	if start >= len(m.projects) {
		return []*models.Project{}, nil
	}
	if end > len(m.projects) {
		end = len(m.projects)
	}
	return m.projects[start:end], nil
}

func (m *mockProjectRepository) Update(ctx context.Context, project *models.Project) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	for i, p := range m.projects {
		if p.ID == project.ID {
			m.projects[i] = project
			return nil
		}
	}
	return repository.ErrNotFound
}

func (m *mockProjectRepository) Count(ctx context.Context) (int64, error) {
	return int64(len(m.projects)), nil
}

func (m *mockProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, p := range m.projects {
		if p.ID == id {
			m.projects = append(m.projects[:i], m.projects[i+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

// Mock DatabaseRepository for testing ProjectHandler
type mockDatabaseRepositoryForProject struct {
	databases []*models.Database
	listErr   error
}

func (m *mockDatabaseRepositoryForProject) Create(ctx context.Context, database *models.Database) error {
	return nil
}

func (m *mockDatabaseRepositoryForProject) GetByID(ctx context.Context, id uuid.UUID) (*models.Database, error) {
	return nil, nil
}

func (m *mockDatabaseRepositoryForProject) List(ctx context.Context, limit, offset int) ([]*models.Database, error) {
	return nil, nil
}

func (m *mockDatabaseRepositoryForProject) ListByProject(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*models.Database
	for _, db := range m.databases {
		if db.ProjectID == projectID {
			result = append(result, db)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockDatabaseRepositoryForProject) Count(ctx context.Context) (int, error) {
	return len(m.databases), nil
}

func (m *mockDatabaseRepositoryForProject) GetByProjectID(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*models.Database, error) {
	return m.ListByProject(ctx, projectID, limit, offset)
}

func (m *mockDatabaseRepositoryForProject) ListByInstanceID(ctx context.Context, instanceID uuid.UUID, limit, offset int) ([]*models.Database, error) {
	return nil, nil
}

func (m *mockDatabaseRepositoryForProject) ListByEnvironmentID(ctx context.Context, environmentID uuid.UUID, limit, offset int) ([]*models.Database, error) {
	return nil, nil
}

func (m *mockDatabaseRepositoryForProject) Update(ctx context.Context, database *models.Database) error {
	return nil
}

func (m *mockDatabaseRepositoryForProject) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

// Mock ProjectMemberRepository for testing
type mockProjectMemberRepositoryForProject struct{}
func (m *mockProjectMemberRepositoryForProject) AddMember(ctx context.Context, member *models.ProjectMember) error { return nil }
func (m *mockProjectMemberRepositoryForProject) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error { return nil }
func (m *mockProjectMemberRepositoryForProject) GetMembers(ctx context.Context, projectID uuid.UUID) ([]*models.ProjectMember, error) { return nil, nil }
func (m *mockProjectMemberRepositoryForProject) GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (*models.ProjectMember, error) { return nil, nil }
func (m *mockProjectMemberRepositoryForProject) UpdateRole(ctx context.Context, projectID, userID uuid.UUID, role models.ProjectRole) error { return nil }

// Mock UserRepository for testing
type mockUserRepositoryForProject struct{}
func (m *mockUserRepositoryForProject) Create(ctx context.Context, user *models.User) error { return nil }
func (m *mockUserRepositoryForProject) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) { return nil, nil }
func (m *mockUserRepositoryForProject) GetByEmail(ctx context.Context, email string) (*models.User, error) { return nil, nil }
func (m *mockUserRepositoryForProject) Update(ctx context.Context, user *models.User) error { return nil }
func (m *mockUserRepositoryForProject) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockUserRepositoryForProject) List(ctx context.Context, limit, offset int) ([]*models.User, error) { return nil, nil }
func (m *mockUserRepositoryForProject) UpdateLastLogin(ctx context.Context, id uuid.UUID) error { return nil }

// Helper to create test projects
func createTestProject(id string, name string, description string) *models.Project {
	uid, _ := uuid.Parse(id)
	createdByID := uuid.New()
	now := time.Now()
	descPtr := &description
	if description == "" {
		descPtr = nil
	}
	return &models.Project{
		ID:          uid,
		Name:        name,
		Description: descPtr,
		CreatedByID: createdByID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Helper to set authenticated user in Echo context
func setAuthContext(c echo.Context, userID uuid.UUID, email string, orgRole string) {
	claims := &auth.Claims{
		UserID:           userID,
		Email:            email,
		OrganizationRole: orgRole,
	}
	c.Set(string(auth.UserContextKey), claims)
}

func TestCreateProject(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		requestBody    string
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "create project successfully",
			requestBody:    `{"name":"Test Project","description":"Test Description"}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "create project with empty description",
			requestBody:    `{"name":"Test Project","description":""}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "validation error - empty name",
			requestBody:    `{"name":"","description":"Test Description"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error - name too long",
			requestBody:    `{"name":"` + strings.Repeat("a", 101) + `","description":"Test"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error - description too long",
			requestBody:    `{"name":"Test","description":"` + strings.Repeat("a", 501) + `"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			requestBody:    `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "authentication required",
			requestBody:    `{"name":"Test Project","description":"Test Description"}`,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "database create error",
			requestBody:    `{"name":"Test Project","description":"Test Description"}`,
			mockErr:        errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := &mockProjectRepository{
				projects:  []*models.Project{},
				createErr: tt.mockErr,
			}
			handler := NewProjectHandler(mockRepo, &mockDatabaseRepositoryForProject{databases: []*models.Database{}}, &mockProjectMemberRepositoryForProject{}, &mockUserRepositoryForProject{}, logger)

			// Create request
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(tt.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set authenticated user context for CreateProject (except for auth test)
			if !strings.Contains(tt.name, "authentication required") {
				userID := uuid.New()
				setAuthContext(c, userID, "admin@example.com", "OrganizationAdmin")
			}

			// Execute handler
			err := handler.CreateProject(c)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}

			// Verify status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}

			// Verify response for successful request
			if tt.expectedStatus == http.StatusCreated {
				var response models.ProjectResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.ID == uuid.Nil {
					t.Error("Expected non-nil project ID")
				}
			}
		})
	}
}

func TestListProjects(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		mockProjects   []*models.Project
		mockErr        error
		queryParams    map[string]string
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "list projects successfully",
			mockProjects: []*models.Project{
				createTestProject("550e8400-e29b-41d4-a716-446655440001", "Project 1", "Description 1"),
				createTestProject("550e8400-e29b-41d4-a716-446655440002", "Project 2", "Description 2"),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name: "list projects with pagination",
			mockProjects: []*models.Project{
				createTestProject("550e8400-e29b-41d4-a716-446655440001", "Project 1", "Description 1"),
			},
			queryParams:    map[string]string{"limit": "1", "offset": "0"},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "list projects with database error",
			mockErr:        errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
		{
			name:           "list empty projects",
			mockProjects:   []*models.Project{},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := &mockProjectRepository{
				projects: tt.mockProjects,
				listErr:  tt.mockErr,
			}
			handler := NewProjectHandler(mockRepo, &mockDatabaseRepositoryForProject{databases: []*models.Database{}}, &mockProjectMemberRepositoryForProject{}, &mockUserRepositoryForProject{}, logger)

			// Create request
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)

			// Add query parameters if any
			if len(tt.queryParams) > 0 {
				q := req.URL.Query()
				for k, v := range tt.queryParams {
					q.Add(k, v)
				}
				req.URL.RawQuery = q.Encode()
			}

			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute handler
			err := handler.ListProjects(c)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}

			// Verify status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Verify response body for successful requests
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				projects, ok := response["projects"].([]interface{})
				if !ok {
					t.Fatal("Response does not contain projects array")
				}

				if len(projects) != tt.expectedCount {
					t.Errorf("Expected %d projects, got %d", tt.expectedCount, len(projects))
				}
			}
		})
	}
}

func TestGetProject(t *testing.T) {
	logger := zap.NewNop()
	testProjectID := "550e8400-e29b-41d4-a716-446655440001"
	testProject := createTestProject(testProjectID, "Test Project", "Test Description")

	tests := []struct {
		name           string
		projectID      string
		mockProjects   []*models.Project
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "get project successfully",
			projectID:      testProjectID,
			mockProjects:   []*models.Project{testProject},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "project not found",
			projectID:      "550e8400-e29b-41d4-a716-446655440099",
			mockProjects:   []*models.Project{testProject},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid project ID format",
			projectID:      "invalid-uuid",
			mockProjects:   []*models.Project{testProject},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "database error",
			projectID:      testProjectID,
			mockErr:        errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := &mockProjectRepository{
				projects: tt.mockProjects,
				getErr:   tt.mockErr,
			}
			handler := NewProjectHandler(mockRepo, &mockDatabaseRepositoryForProject{databases: []*models.Database{}}, &mockProjectMemberRepositoryForProject{}, &mockUserRepositoryForProject{}, logger)

			// Create request
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+tt.projectID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.projectID)

			// Execute handler
			err := handler.GetProject(c)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}

			// Verify status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Verify response for successful request
			if tt.expectedStatus == http.StatusOK {
				var response models.ProjectResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Name != testProject.Name {
					t.Errorf("Expected name %s, got %s", testProject.Name, response.Name)
				}
			}
		})
	}
}

func TestUpdateProject(t *testing.T) {
	logger := zap.NewNop()
	testProjectID := "550e8400-e29b-41d4-a716-446655440001"
	testProject := createTestProject(testProjectID, "Test Project", "Test Description")

	tests := []struct {
		name           string
		projectID      string
		requestBody    string
		mockProjects   []*models.Project
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "update project successfully",
			projectID:      testProjectID,
			requestBody:    `{"name":"Updated Project","description":"Updated Description"}`,
			mockProjects:   []*models.Project{testProject},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "update project name only",
			projectID:      testProjectID,
			requestBody:    `{"name":"Updated Project"}`,
			mockProjects:   []*models.Project{testProject},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "project not found",
			projectID:      "550e8400-e29b-41d4-a716-446655440099",
			requestBody:    `{"name":"Updated Project"}`,
			mockProjects:   []*models.Project{testProject},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid project ID format",
			projectID:      "invalid-uuid",
			requestBody:    `{"name":"Updated Project"}`,
			mockProjects:   []*models.Project{testProject},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			projectID:      testProjectID,
			requestBody:    `{invalid json}`,
			mockProjects:   []*models.Project{testProject},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error - empty name",
			projectID:      testProjectID,
			requestBody:    `{"name":""}`,
			mockProjects:   []*models.Project{testProject},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "database update error",
			projectID:      testProjectID,
			requestBody:    `{"name":"Updated Project"}`,
			mockProjects:   []*models.Project{testProject},
			mockErr:        errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := &mockProjectRepository{
				projects:  tt.mockProjects,
				updateErr: tt.mockErr,
			}
			handler := NewProjectHandler(mockRepo, &mockDatabaseRepositoryForProject{databases: []*models.Database{}}, &mockProjectMemberRepositoryForProject{}, &mockUserRepositoryForProject{}, logger)

			// Create request
			e := echo.New()
			req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/"+tt.projectID, strings.NewReader(tt.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.projectID)

			// Execute handler
			err := handler.UpdateProject(c)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}

			// Verify status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestDeleteProject(t *testing.T) {
	logger := zap.NewNop()
	testProjectID := "550e8400-e29b-41d4-a716-446655440001"
	testProject := createTestProject(testProjectID, "Test Project", "Test Description")

	tests := []struct {
		name           string
		projectID      string
		mockProjects   []*models.Project
		mockDatabases  []*models.Database
		mockErr        error
		dbListErr      error
		expectedStatus int
	}{
		{
			name:           "delete project successfully",
			projectID:      testProjectID,
			mockProjects:   []*models.Project{testProject},
			mockDatabases:  []*models.Database{},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "project not found",
			projectID:      "550e8400-e29b-41d4-a716-446655440099",
			mockProjects:   []*models.Project{testProject},
			mockDatabases:  []*models.Database{},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid project ID format",
			projectID:      "invalid-uuid",
			mockProjects:   []*models.Project{testProject},
			mockDatabases:  []*models.Database{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "cannot delete project with databases",
			projectID:      testProjectID,
			mockProjects:   []*models.Project{testProject},
			mockDatabases:  []*models.Database{{ProjectID: uuid.MustParse(testProjectID)}},
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "database delete error",
			projectID:      testProjectID,
			mockProjects:   []*models.Project{testProject},
			mockDatabases:  []*models.Database{},
			mockErr:        errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "database list error",
			projectID:      testProjectID,
			mockProjects:   []*models.Project{testProject},
			mockDatabases:  []*models.Database{},
			dbListErr:      errors.New("database list error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := &mockProjectRepository{
				projects:  tt.mockProjects,
				deleteErr: tt.mockErr,
			}
			mockDBRepo := &mockDatabaseRepositoryForProject{
				databases: tt.mockDatabases,
				listErr:   tt.dbListErr,
			}
			handler := NewProjectHandler(mockRepo, mockDBRepo, &mockProjectMemberRepositoryForProject{}, &mockUserRepositoryForProject{}, logger)

			// Create request
			e := echo.New()
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+tt.projectID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.projectID)

			// Execute handler
			err := handler.DeleteProject(c)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}

			// Verify status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Verify response for successful request
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				message, ok := response["message"].(string)
				if !ok {
					t.Fatal("Response does not contain message")
				}

				if message != "Project deleted successfully" {
					t.Errorf("Expected success message, got %s", message)
				}
			}
		})
	}
}
