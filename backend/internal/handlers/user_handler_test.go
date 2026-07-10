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

// noOpValidator is a validator that always returns nil (no validation errors)
type noOpValidator struct{}

func (v *noOpValidator) Validate(i interface{}) error {
	return nil
}

// Mock UserRepository for testing
type mockUserRepository struct {
	users   []*models.User
	getErr  error
	listErr error
	updErr  error
}

func (m *mockUserRepository) Create(ctx context.Context, user *models.User) error {
	m.users = append(m.users, user)
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockUserRepository) Update(ctx context.Context, user *models.User) error {
	if m.updErr != nil {
		return m.updErr
	}
	for i, u := range m.users {
		if u.ID == user.ID {
			m.users[i] = user
			return nil
		}
	}
	return repository.ErrNotFound
}

func (m *mockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockUserRepository) List(ctx context.Context, _ repository.UserFilters, limit, offset int) ([]*models.User, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	start := offset
	end := offset + limit
	if start >= len(m.users) {
		return []*models.User{}, nil
	}
	if end > len(m.users) {
		end = len(m.users)
	}
	return m.users[start:end], nil
}

func (m *mockUserRepository) Count(ctx context.Context, _ repository.UserFilters) (int64, error) {
	return int64(len(m.users)), nil
}

func (m *mockUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	return nil
}

// Helper to create test users
func createTestUser(id string, email string, role string) *models.User {
	uid, _ := uuid.Parse(id)
	now := time.Now()
	return &models.User{
		ID:               uid,
		Email:            email,
		PasswordHash:     "hashed",
		OrganizationRole: rbac.OrganizationRole(role),
		TwoFactorEnabled: false,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func TestListUsers(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name           string
		mockUsers      []*models.User
		mockErr        error
		queryParams    map[string]string
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "list users successfully",
			mockUsers: []*models.User{
				createTestUser("550e8400-e29b-41d4-a716-446655440001", "user1@example.com", "organization_admin"),
				createTestUser("550e8400-e29b-41d4-a716-446655440002", "user2@example.com", "organization_developer"),
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name: "list users with pagination",
			mockUsers: []*models.User{
				createTestUser("550e8400-e29b-41d4-a716-446655440001", "user1@example.com", "organization_admin"),
			},
			queryParams:    map[string]string{"limit": "1", "offset": "0"},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "list users with database error",
			mockErr:        errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
		{
			name:           "list empty users",
			mockUsers:      []*models.User{},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := &mockUserRepository{
				users:   tt.mockUsers,
				listErr: tt.mockErr,
			}
			mockProjectAssignmentService := new(MockProjectAssignmentService)
			handler := NewUserHandler(mockRepo, mockProjectAssignmentService, logger)

			// Create request
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

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
			err := handler.ListUsers(c)
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

				users, ok := response["users"].([]interface{})
				if !ok {
					t.Fatal("Response does not contain users array")
				}

				if len(users) != tt.expectedCount {
					t.Errorf("Expected %d users, got %d", tt.expectedCount, len(users))
				}
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	logger := zap.NewNop()
	testUserID := "550e8400-e29b-41d4-a716-446655440001"
	testUser := createTestUser(testUserID, "test@example.com", "organization_admin")

	tests := []struct {
		name           string
		userID         string
		mockUsers      []*models.User
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "get user successfully",
			userID:         testUserID,
			mockUsers:      []*models.User{testUser},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user not found",
			userID:         "550e8400-e29b-41d4-a716-446655440099",
			mockUsers:      []*models.User{testUser},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid user ID format",
			userID:         "invalid-uuid",
			mockUsers:      []*models.User{testUser},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "database error",
			userID:         testUserID,
			mockErr:        errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := &mockUserRepository{
				users:  tt.mockUsers,
				getErr: tt.mockErr,
			}
			mockProjectAssignmentService := new(MockProjectAssignmentService)
			handler := NewUserHandler(mockRepo, mockProjectAssignmentService, logger)

			// Create request
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+tt.userID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.userID)

			// Set authenticated user context
			userID := uuid.New()
			c.Set("user", &auth.Claims{
				UserID:           userID,
				Email:            "admin@example.com",
				OrganizationRole: string(rbac.OrganizationAdmin),
			})

			// Execute handler
			err := handler.GetUser(c)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}

			// Verify status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Verify response for successful request
			if tt.expectedStatus == http.StatusOK {
				var response models.UserResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Email != testUser.Email {
					t.Errorf("Expected email %s, got %s", testUser.Email, response.Email)
				}
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	logger := zap.NewNop()
	testUserID := "550e8400-e29b-41d4-a716-446655440001"
	testUser := createTestUser(testUserID, "test@example.com", "organization_developer")

	tests := []struct {
		name           string
		userID         string
		requestBody    string
		mockUsers      []*models.User
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "update user successfully",
			userID:         testUserID,
			requestBody:    `{"organization_role":"organization_admin"}`,
			mockUsers:      []*models.User{testUser},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "update user is_active",
			userID:         testUserID,
			requestBody:    `{"is_active":false}`,
			mockUsers:      []*models.User{testUser},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user not found",
			userID:         "550e8400-e29b-41d4-a716-446655440099",
			requestBody:    `{"organization_role":"organization_admin"}`,
			mockUsers:      []*models.User{testUser},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid user ID format",
			userID:         "invalid-uuid",
			requestBody:    `{"organization_role":"organization_admin"}`,
			mockUsers:      []*models.User{testUser},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid request body",
			userID:         testUserID,
			requestBody:    `{invalid json}`,
			mockUsers:      []*models.User{testUser},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "database update error",
			userID:         testUserID,
			requestBody:    `{"organization_role":"organization_admin"}`,
			mockUsers:      []*models.User{testUser},
			mockErr:        errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := &mockUserRepository{
				users:  tt.mockUsers,
				updErr: tt.mockErr,
			}
			mockProjectAssignmentService := new(MockProjectAssignmentService)
			handler := NewUserHandler(mockRepo, mockProjectAssignmentService, logger)

			// Create request
			e := echo.New()
			// Set a no-op validator since we're not testing validation here
			e.Validator = &noOpValidator{}
			req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+tt.userID, strings.NewReader(tt.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.userID)

			// Set authenticated user context (admin user for permissions)
			adminUserID := uuid.New()
			c.Set("user", &auth.Claims{
				UserID:           adminUserID,
				Email:            "admin@example.com",
				OrganizationRole: string(rbac.OrganizationAdmin),
			})

			// Execute handler
			err := handler.UpdateUser(c)
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

func TestDeleteUser(t *testing.T) {
	logger := zap.NewNop()
	testUserID := "550e8400-e29b-41d4-a716-446655440001"
	testUser := createTestUser(testUserID, "test@example.com", "organization_developer")

	tests := []struct {
		name           string
		userID         string
		mockUsers      []*models.User
		mockErr        error
		expectedStatus int
	}{
		{
			name:           "deactivate user successfully",
			userID:         testUserID,
			mockUsers:      []*models.User{testUser},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "user not found",
			userID:         "550e8400-e29b-41d4-a716-446655440099",
			mockUsers:      []*models.User{testUser},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid user ID format",
			userID:         "invalid-uuid",
			mockUsers:      []*models.User{testUser},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "database update error",
			userID:         testUserID,
			mockUsers:      []*models.User{testUser},
			mockErr:        errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := &mockUserRepository{
				users:  tt.mockUsers,
				updErr: tt.mockErr,
			}
			mockProjectAssignmentService := new(MockProjectAssignmentService)
			handler := NewUserHandler(mockRepo, mockProjectAssignmentService, logger)

			// Create request
			e := echo.New()
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+tt.userID, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.userID)

			// Execute handler
			err := handler.DeleteUser(c)
			if err != nil {
				t.Fatalf("Handler returned error: %v", err)
			}

			// Verify status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Verify user is deactivated for successful request
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				user, ok := response["user"].(map[string]interface{})
				if !ok {
					t.Fatal("Response does not contain user object")
				}

				isActive, ok := user["is_active"].(bool)
				if !ok {
					t.Fatal("User object does not contain is_active field")
				}

				if isActive {
					t.Error("Expected user to be deactivated (is_active=false)")
				}
			}
		})
	}
}
