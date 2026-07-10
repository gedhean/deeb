package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockUserRepository is a mock implementation of repository.UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	// Simulate setting ID and timestamps like real implementation does
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(
	ctx context.Context,
	filters repository.UserFilters,
	limit, offset int,
) ([]*models.User, error) {
	args := m.Called(ctx, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) Count(ctx context.Context, filters repository.UserFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestRegisterHandler_ServeHTTP_Success(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Mock expectations
	mockRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, sql.ErrNoRows)
	mockRepo.On("List", mock.Anything, mock.Anything, 1, 0).Return([]*models.User{{ID: uuid.New()}}, nil) // Existing user
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
		return u.Email == "test@example.com" &&
			u.PasswordHash != "" &&
			u.OrganizationRole == "OrganizationDeveloper" &&
			u.IsActive == true
	})).Return(nil)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Create request
	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "StrongPass123!",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp RegisterResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.NotNil(t, resp.User)
	assert.NotEqual(t, uuid.Nil, resp.User.ID)
	assert.Equal(t, "test@example.com", resp.User.Email)
	assert.Equal(t, rbac.OrganizationDeveloper, resp.User.OrganizationRole)

	mockRepo.AssertExpectations(t)
}

func TestRegisterHandler_ServeHTTP_CustomOrganizationRole(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	customRole := "OrganizationAdmin"

	// Mock expectations
	mockRepo.On("GetByEmail", mock.Anything, "admin@example.com").Return(nil, sql.ErrNoRows)
	mockRepo.On("List", mock.Anything, mock.Anything, 1, 0).Return([]*models.User{{ID: uuid.New()}}, nil) // Existing user
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
		return u.Email == "admin@example.com" &&
			u.OrganizationRole == "OrganizationAdmin"
	})).Return(nil)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Create request with custom role
	reqBody := RegisterRequest{
		Email:            "admin@example.com",
		Password:         "AdminPass123!",
		OrganizationRole: &customRole,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp RegisterResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, rbac.OrganizationAdmin, resp.User.OrganizationRole)

	mockRepo.AssertExpectations(t)
}

func TestRegisterHandler_ServeHTTP_InvalidEmailFormat(t *testing.T) {
	testCases := []struct {
		name  string
		email string
	}{
		{"missing @", "invalidemail"},
		{"missing domain", "test@"},
		{"missing local part", "@example.com"},
		{"spaces", "test @example.com"},
		{"no TLD", "test@example"},
		{"empty", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockUserRepository)
			mockProjectAssignmentService := new(MockProjectAssignmentService)
			logger := zap.NewNop()
			handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
			mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

			// Create request
			reqBody := RegisterRequest{
				Email:    tc.email,
				Password: "ValidPass123!",
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute
			_ = handler.Register(c)

			// Assert
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var resp ErrorResponse
			err := json.NewDecoder(rec.Body).Decode(&resp)
			assert.NoError(t, err)
			assert.Equal(t, "invalid email format", resp.Error)
		})
	}
}

func TestRegisterHandler_ServeHTTP_PasswordTooShort(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Create request with short password
	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "short",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "password must be at least 8 characters", resp.Error)
}

func TestRegisterHandler_ServeHTTP_PasswordTooLong(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Create request with password > 72 characters
	longPassword := make([]byte, 73)
	for i := range longPassword {
		longPassword[i] = 'a'
	}

	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: string(longPassword),
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "password must not exceed 72 characters", resp.Error)
}

func TestRegisterHandler_ServeHTTP_EmptyPassword(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Create request with empty password
	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "password is required", resp.Error)
}

func TestRegisterHandler_ServeHTTP_DuplicateEmail(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Mock existing user
	existingUser := &models.User{
		ID:               uuid.New(),
		Email:            "existing@example.com",
		PasswordHash:     "$2a$10$...",
		OrganizationRole: "OrganizationDeveloper",
		IsActive:         true,
	}

	mockRepo.On("GetByEmail", mock.Anything, "existing@example.com").Return(existingUser, nil)

	// Create request
	reqBody := RegisterRequest{
		Email:    "existing@example.com",
		Password: "ValidPass123!",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusConflict, rec.Code)

	var resp ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "email already registered", resp.Error)

	mockRepo.AssertExpectations(t)
}

func TestRegisterHandler_ServeHTTP_MalformedJSON(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Create request with malformed JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "invalid request body", resp.Error)
}

func TestRegisterHandler_ServeHTTP_DatabaseErrorOnGetByEmail(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Mock database error
	mockRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("database connection failed"))

	// Create request
	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "ValidPass123!",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "internal server error", resp.Error)

	mockRepo.AssertExpectations(t)
}

func TestRegisterHandler_ServeHTTP_DatabaseErrorOnCreate(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Mock expectations
	mockRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, sql.ErrNoRows)
	mockRepo.On("List", mock.Anything, mock.Anything, 1, 0).Return([]*models.User{{ID: uuid.New()}}, nil)
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("database insert failed"))

	// Create request
	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "ValidPass123!",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "failed to create user", resp.Error)

	mockRepo.AssertExpectations(t)
}

func TestRegisterHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Create GET request (should only accept POST)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/register", nil)
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)

	var resp ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "method not allowed", resp.Error)
}

func TestRegisterHandler_ServeHTTP_EmailNormalization(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Mock expectations - email should be lowercased and trimmed
	mockRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, sql.ErrNoRows)
	mockRepo.On("List", mock.Anything, mock.Anything, 1, 0).Return([]*models.User{{ID: uuid.New()}}, nil)
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
		return u.Email == "test@example.com" // Should be normalized
	})).Return(nil)

	// Create request with uppercase and whitespace
	reqBody := RegisterRequest{
		Email:    "  TeSt@ExAmPlE.CoM  ",
		Password: "ValidPass123!",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusCreated, rec.Code)
	mockRepo.AssertExpectations(t)
}

func TestRegisterHandler_ServeHTTP_FirstUserBecomesAdmin(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Mock expectations for first user (no existing users)
	mockRepo.On("GetByEmail", mock.Anything, "first@example.com").Return(nil, sql.ErrNoRows)
	mockRepo.On("List", mock.Anything, mock.Anything, 1, 0).Return([]*models.User{}, nil) // No users exist
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
		return u.Email == "first@example.com" &&
			u.PasswordHash != "" &&
			u.OrganizationRole == "OrganizationAdmin" && // Should be admin for first user
			u.IsActive == true
	})).Return(nil)

	// Create request for first user
	reqBody := RegisterRequest{
		Email:    "first@example.com",
		Password: "FirstUserPass123!",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp RegisterResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.NotNil(t, resp.User)
	assert.Equal(t, "first@example.com", resp.User.Email)
	assert.Equal(t, rbac.OrganizationAdmin, resp.User.OrganizationRole)

	mockRepo.AssertExpectations(t)
}

func TestRegisterHandler_ServeHTTP_SecondUserGetsDeveloperRole(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Mock expectations for second user (one user already exists)
	existingUser := &models.User{
		ID:               uuid.New(),
		Email:            "first@example.com",
		OrganizationRole: "OrganizationAdmin",
		IsActive:         true,
	}

	mockRepo.On("GetByEmail", mock.Anything, "second@example.com").Return(nil, sql.ErrNoRows)
	mockRepo.On("List", mock.Anything, mock.Anything, 1, 0).Return([]*models.User{existingUser}, nil) // One user exists
	mockRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
		return u.Email == "second@example.com" &&
			u.PasswordHash != "" &&
			u.OrganizationRole == "OrganizationDeveloper" && // Should be developer for subsequent users
			u.IsActive == true
	})).Return(nil)

	// Create request for second user
	reqBody := RegisterRequest{
		Email:    "second@example.com",
		Password: "SecondUserPass123!",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp RegisterResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.NotNil(t, resp.User)
	assert.Equal(t, "second@example.com", resp.User.Email)
	assert.Equal(t, rbac.OrganizationDeveloper, resp.User.OrganizationRole)

	mockRepo.AssertExpectations(t)
}

func TestRegisterHandler_ServeHTTP_FirstUserSetup_DatabaseError(t *testing.T) {
	// Setup
	mockRepo := new(MockUserRepository)
	mockProjectAssignmentService := new(MockProjectAssignmentService)
	logger := zap.NewNop()
	handler := NewRegisterHandler(mockRepo, mockProjectAssignmentService, logger)
	mockProjectAssignmentService.On("AssignUserToExistingProjects", mock.Anything, mock.Anything).Return(0, nil)

	// Mock expectations - List() returns an error
	mockRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, sql.ErrNoRows)
	mockRepo.On("List", mock.Anything, mock.Anything, 1, 0).Return(nil, errors.New("database connection failed"))

	// Create request
	reqBody := RegisterRequest{
		Email:    "test@example.com",
		Password: "ValidPass123!",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	_ = handler.Register(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var resp ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "internal server error", resp.Error)

	mockRepo.AssertExpectations(t)
}
