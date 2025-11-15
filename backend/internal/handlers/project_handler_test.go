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
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// Mock ProjectRepository for testing
type mockProjectRepository struct {
	projects        []*models.Project
	projectMembers  map[uuid.UUID][]uuid.UUID // projectID -> []userID
	getErr          error
	listErr         error
	listByUserIDErr error
	createErr       error
	updateErr       error
	deleteErr       error
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

func (m *mockProjectRepository) ListByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit, offset int,
) ([]*models.Project, error) {
	if m.listByUserIDErr != nil {
		return nil, m.listByUserIDErr
	}

	// If projectMembers map is not initialized, return empty list
	if m.projectMembers == nil {
		return []*models.Project{}, nil
	}

	// Find projects where user is a member
	var userProjects []*models.Project
	for _, project := range m.projects {
		if members, ok := m.projectMembers[project.ID]; ok {
			for _, memberID := range members {
				if memberID == userID {
					userProjects = append(userProjects, project)
					break
				}
			}
		}
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start >= len(userProjects) {
		return []*models.Project{}, nil
	}
	if end > len(userProjects) {
		end = len(userProjects)
	}
	return userProjects[start:end], nil
}

func (m *mockProjectRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	if m.listByUserIDErr != nil {
		return 0, m.listByUserIDErr
	}

	// If projectMembers map is not initialized, return 0
	if m.projectMembers == nil {
		return 0, nil
	}

	// Count projects where user is a member
	count := int64(0)
	for _, project := range m.projects {
		if members, ok := m.projectMembers[project.ID]; ok {
			for _, memberID := range members {
				if memberID == userID {
					count++
					break
				}
			}
		}
	}
	return count, nil
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

func (m *mockDatabaseRepositoryForProject) GetByProjectID(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	return m.ListByProject(ctx, projectID, limit, offset)
}

func (m *mockDatabaseRepositoryForProject) ListByInstanceID(
	ctx context.Context,
	instanceID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	return nil, nil
}

func (m *mockDatabaseRepositoryForProject) ListByEnvironmentID(
	ctx context.Context,
	environmentID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	return nil, nil
}

func (m *mockDatabaseRepositoryForProject) Update(ctx context.Context, database *models.Database) error {
	return nil
}

func (m *mockDatabaseRepositoryForProject) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockDatabaseRepositoryForProject) GetByInstanceAndName(
	ctx context.Context,
	instanceID uuid.UUID,
	databaseName string,
) (*models.Database, error) {
	for _, db := range m.databases {
		if db.DatabaseInstanceID == instanceID && db.DatabaseName == databaseName {
			return db, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockDatabaseRepositoryForProject) ListActiveForSync(ctx context.Context, limit, offset int) ([]*models.Database, error) {
	return nil, nil
}

// Mock ProjectMemberRepository for testing
type mockProjectMemberRepositoryForProject struct {
	members      []*models.ProjectMember
	addMemberErr error
}

func (m *mockProjectMemberRepositoryForProject) AddMember(ctx context.Context, member *models.ProjectMember) error {
	if m.addMemberErr != nil {
		return m.addMemberErr
	}
	if m.members == nil {
		m.members = []*models.ProjectMember{}
	}
	m.members = append(m.members, member)
	return nil
}

func (m *mockProjectMemberRepositoryForProject) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	return nil
}

func (m *mockProjectMemberRepositoryForProject) GetMembers(
	ctx context.Context,
	projectID uuid.UUID,
) ([]*models.ProjectMember, error) {
	var result []*models.ProjectMember
	for _, member := range m.members {
		if member.ProjectID == projectID {
			result = append(result, member)
		}
	}
	return result, nil
}

func (m *mockProjectMemberRepositoryForProject) GetMemberRole(
	ctx context.Context,
	projectID, userID uuid.UUID,
) (*models.ProjectMember, error) {
	for _, member := range m.members {
		if member.ProjectID == projectID && member.UserID == userID {
			return member, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockProjectMemberRepositoryForProject) UpdateRole(
	ctx context.Context,
	projectID, userID uuid.UUID,
	role rbac.ProjectRole,
) error {
	return nil
}

func (m *mockProjectMemberRepositoryForProject) GetUserProjectIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var projectIDs []uuid.UUID
	for _, member := range m.members {
		if member.UserID == userID {
			projectIDs = append(projectIDs, member.ProjectID)
		}
	}
	return projectIDs, nil
}

// Mock UserRepository for testing
type mockUserRepositoryForProject struct{}

func (m *mockUserRepositoryForProject) Create(ctx context.Context, user *models.User) error {
	return nil
}

func (m *mockUserRepositoryForProject) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return nil, nil
}

func (m *mockUserRepositoryForProject) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, nil
}

func (m *mockUserRepositoryForProject) Update(ctx context.Context, user *models.User) error {
	return nil
}
func (m *mockUserRepositoryForProject) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockUserRepositoryForProject) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	return nil, nil
}

func (m *mockUserRepositoryForProject) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	return nil
}

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
			// Setup mock repositories
			mockRepo := &mockProjectRepository{
				projects:  []*models.Project{},
				createErr: tt.mockErr,
			}
			mockMemberRepo := &mockProjectMemberRepositoryForProject{}
			handler := NewProjectHandler(
				mockRepo,
				&mockDatabaseRepositoryForProject{databases: []*models.Database{}},
				mockMemberRepo,
				&mockUserRepositoryForProject{},
				nil,
				logger,
			)

			// Create request
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(tt.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set authenticated user context for CreateProject (except for auth test)
			var testUserID uuid.UUID
			if !strings.Contains(tt.name, "authentication required") {
				testUserID = uuid.New()
				setAuthContext(c, testUserID, "admin@example.com", "OrganizationAdmin")
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

				// Verify that the creator was added as ProjectOwner member
				if len(mockMemberRepo.members) != 1 {
					t.Errorf("Expected 1 project member (creator), got %d", len(mockMemberRepo.members))
				} else {
					member := mockMemberRepo.members[0]
					if member.UserID != testUserID {
						t.Errorf("Expected member user ID %s, got %s", testUserID, member.UserID)
					}
					if member.Role != rbac.ProjectOwner {
						t.Errorf("Expected member role %s, got %s", rbac.ProjectOwner, member.Role)
					}
					if member.AddedByID != testUserID {
						t.Errorf("Expected added by %s, got %s", testUserID, member.AddedByID)
					}
				}
			}
		})
	}
}

func TestListProjects(t *testing.T) {
	logger := zap.NewNop()

	// Create test users
	adminUserID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440010")
	dbaUserID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440011")
	devUserID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440012")

	// Create test projects
	project1 := createTestProject("550e8400-e29b-41d4-a716-446655440001", "Project 1", "Description 1")
	project2 := createTestProject("550e8400-e29b-41d4-a716-446655440002", "Project 2", "Description 2")
	project3 := createTestProject("550e8400-e29b-41d4-a716-446655440003", "Project 3", "Description 3")

	tests := []struct {
		name           string
		userClaims     *auth.Claims
		mockProjects   []*models.Project
		projectMembers map[uuid.UUID][]uuid.UUID
		mockListErr    error
		queryParams    map[string]string
		expectedStatus int
		expectedCount  int
		expectedTotal  int64
		checkUnauth    bool
	}{
		{
			name: "admin user sees all projects",
			userClaims: &auth.Claims{
				UserID:           adminUserID,
				OrganizationRole: string(rbac.OrganizationAdmin),
			},
			mockProjects:   []*models.Project{project1, project2, project3},
			expectedStatus: http.StatusOK,
			expectedCount:  3,
			expectedTotal:  3,
		},
		{
			name: "dba user sees only member projects",
			userClaims: &auth.Claims{
				UserID:           dbaUserID,
				OrganizationRole: string(rbac.OrganizationDBA),
			},
			mockProjects: []*models.Project{project1, project2, project3},
			projectMembers: map[uuid.UUID][]uuid.UUID{
				project1.ID: {dbaUserID},
				project2.ID: {dbaUserID},
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectedTotal:  2,
		},
		{
			name: "developer sees only member projects",
			userClaims: &auth.Claims{
				UserID:           devUserID,
				OrganizationRole: string(rbac.OrganizationDeveloper),
			},
			mockProjects: []*models.Project{project1, project2, project3},
			projectMembers: map[uuid.UUID][]uuid.UUID{
				project1.ID: {devUserID},
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			expectedTotal:  1,
		},
		{
			name: "developer with no memberships sees empty list",
			userClaims: &auth.Claims{
				UserID:           devUserID,
				OrganizationRole: string(rbac.OrganizationDeveloper),
			},
			mockProjects:   []*models.Project{project1, project2},
			projectMembers: map[uuid.UUID][]uuid.UUID{},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
			expectedTotal:  0,
		},
		{
			name: "admin with pagination",
			userClaims: &auth.Claims{
				UserID:           adminUserID,
				OrganizationRole: string(rbac.OrganizationAdmin),
			},
			mockProjects:   []*models.Project{project1, project2, project3},
			queryParams:    map[string]string{"limit": "2", "offset": "0"},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			expectedTotal:  3,
		},
		{
			name:           "missing authentication",
			checkUnauth:    true,
			expectedStatus: http.StatusUnauthorized,
			expectedCount:  0,
		},
		{
			name: "database error for admin",
			userClaims: &auth.Claims{
				UserID:           adminUserID,
				OrganizationRole: string(rbac.OrganizationAdmin),
			},
			mockListErr:    errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := &mockProjectRepository{
				projects:       tt.mockProjects,
				projectMembers: tt.projectMembers,
				listErr:        tt.mockListErr,
			}
			handler := NewProjectHandler(
				mockRepo,
				&mockDatabaseRepositoryForProject{databases: []*models.Database{}},
				&mockProjectMemberRepositoryForProject{},
				&mockUserRepositoryForProject{},
				nil,
				logger,
			)

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

			// Add user claims to context if not testing unauthorized
			if !tt.checkUnauth && tt.userClaims != nil {
				c.Set("user", tt.userClaims)
			}

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

				// Verify total count
				if total, ok := response["total"].(float64); ok {
					if int64(total) != tt.expectedTotal {
						t.Errorf("Expected total %d, got %d", tt.expectedTotal, int64(total))
					}
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
			handler := NewProjectHandler(
				mockRepo,
				&mockDatabaseRepositoryForProject{databases: []*models.Database{}},
				&mockProjectMemberRepositoryForProject{},
				&mockUserRepositoryForProject{},
				nil,
				logger,
			)

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
			handler := NewProjectHandler(
				mockRepo,
				&mockDatabaseRepositoryForProject{databases: []*models.Database{}},
				&mockProjectMemberRepositoryForProject{},
				&mockUserRepositoryForProject{},
				nil,
				logger,
			)

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
			handler := NewProjectHandler(
				mockRepo,
				mockDBRepo,
				&mockProjectMemberRepositoryForProject{},
				&mockUserRepositoryForProject{},
				nil,
				logger,
			)

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
