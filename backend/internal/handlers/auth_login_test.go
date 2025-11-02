package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestLoginHandler_SuccessfulLogin(t *testing.T) {
	// Create a test user with hashed password
	password := "testpassword123"
	hashedPassword, err := auth.HashPassword(password, 0)
	assert.NoError(t, err)

	testUser := &models.User{
		ID:               uuid.New(),
		Email:            "test@example.com",
		PasswordHash:     hashedPassword,
		OrganizationRole: "OrganizationDeveloper",
		IsActive:         true,
		CreatedAt:        time.Now(),
	}

	// Setup mocks
	mockRepo := new(MockUserRepository)
	mockRepo.On("GetByEmail", mock.Anything, testUser.Email).Return(testUser, nil)

	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long", 0)
	assert.NoError(t, err)

	rateLimiter := NewRateLimiter()
	defer rateLimiter.Stop()
	logger := zap.NewNop()

	handler := NewLoginHandler(mockRepo, tokenManager, rateLimiter, logger)

	// Create request
	loginReq := LoginRequest{
		Email:    testUser.Email,
		Password: password,
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	// Execute request
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	// Verify response
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response LoginResponse
	err = json.NewDecoder(rec.Body).Decode(&response)
	assert.NoError(t, err)

	// Verify cookie is set with token
	cookies := rec.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, "deeb_auth_token", cookies[0].Name)
	assert.NotEmpty(t, cookies[0].Value)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, 86400, cookies[0].MaxAge)

	// Verify user info in response
	assert.Equal(t, testUser.ID, response.User.ID)
	assert.Equal(t, testUser.Email, response.User.Email)

	mockRepo.AssertExpectations(t)
}

func TestLoginHandler_InvalidPassword(t *testing.T) {
	hashedPassword, _ := auth.HashPassword("correctpassword", 0)
	testUser := &models.User{
		ID:               uuid.New(),
		Email:            "test@example.com",
		PasswordHash:     hashedPassword,
		OrganizationRole: "OrganizationDeveloper",
		IsActive:         true,
	}

	mockRepo := new(MockUserRepository)
	mockRepo.On("GetByEmail", mock.Anything, testUser.Email).Return(testUser, nil)

	tokenManager, _ := auth.NewTokenManager("test-secret-key-minimum-32-characters-long", 0)
	rateLimiter := NewRateLimiter()
	defer rateLimiter.Stop()
	logger := zap.NewNop()

	handler := NewLoginHandler(mockRepo, tokenManager, rateLimiter, logger)

	loginReq := LoginRequest{
		Email:    testUser.Email,
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)
	assert.Equal(t, "invalid email or password", response.Error)

	mockRepo.AssertExpectations(t)
}

func TestLoginHandler_UserNotFound(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockRepo.On("GetByEmail", mock.Anything, "nonexistent@example.com").Return(nil, sql.ErrNoRows)

	tokenManager, _ := auth.NewTokenManager("test-secret-key-minimum-32-characters-long", 0)
	rateLimiter := NewRateLimiter()
	defer rateLimiter.Stop()
	logger := zap.NewNop()

	handler := NewLoginHandler(mockRepo, tokenManager, rateLimiter, logger)

	loginReq := LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "anypassword",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)
	assert.Equal(t, "invalid email or password", response.Error)

	mockRepo.AssertExpectations(t)
}

func TestLoginHandler_RateLimitExceeded(t *testing.T) {
	hashedPassword, _ := auth.HashPassword("testpassword", 0)
	testUser := &models.User{
		ID:               uuid.New(),
		Email:            "test@example.com",
		PasswordHash:     hashedPassword,
		OrganizationRole: "OrganizationDeveloper",
		IsActive:         true,
	}

	mockRepo := new(MockUserRepository)
	mockRepo.On("GetByEmail", mock.Anything, testUser.Email).Return(testUser, nil)

	tokenManager, _ := auth.NewTokenManager("test-secret-key-minimum-32-characters-long", 0)
	rateLimiter := NewRateLimiter()
	defer rateLimiter.Stop()
	logger := zap.NewNop()

	handler := NewLoginHandler(mockRepo, tokenManager, rateLimiter, logger)

	loginReq := LoginRequest{
		Email:    testUser.Email,
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(loginReq)

	// Make 10 failed attempts (which should succeed up to the limit)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		e := echo.New()
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = handler.Login(c)

		assert.Equal(t, http.StatusUnauthorized, rec.Code, "Attempt %d should return 401", i+1)
	}

	// 11th attempt should be rate limited
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	var response ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "too many login attempts, please try again later", response.Error)
}

func TestLoginHandler_MalformedJSON(t *testing.T) {
	mockRepo := new(MockUserRepository)
	tokenManager, _ := auth.NewTokenManager("test-secret-key-minimum-32-characters-long", 0)
	rateLimiter := NewRateLimiter()
	defer rateLimiter.Stop()
	logger := zap.NewNop()

	handler := NewLoginHandler(mockRepo, tokenManager, rateLimiter, logger)

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)
	assert.Equal(t, "invalid request body", response.Error)
}

func TestLoginHandler_MissingCredentials(t *testing.T) {
	mockRepo := new(MockUserRepository)
	tokenManager, _ := auth.NewTokenManager("test-secret-key-minimum-32-characters-long", 0)
	rateLimiter := NewRateLimiter()
	defer rateLimiter.Stop()
	logger := zap.NewNop()

	handler := NewLoginHandler(mockRepo, tokenManager, rateLimiter, logger)

	testCases := []struct {
		name     string
		email    string
		password string
	}{
		{"missing email", "", "password123"},
		{"missing password", "test@example.com", ""},
		{"both missing", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			loginReq := LoginRequest{
				Email:    tc.email,
				Password: tc.password,
			}
			body, _ := json.Marshal(loginReq)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			_ = handler.Login(c)

			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var response ErrorResponse
			json.NewDecoder(rec.Body).Decode(&response)
			assert.Equal(t, "email and password are required", response.Error)
		})
	}
}

func TestLoginHandler_InactiveUser(t *testing.T) {
	hashedPassword, _ := auth.HashPassword("testpassword", 0)
	testUser := &models.User{
		ID:               uuid.New(),
		Email:            "test@example.com",
		PasswordHash:     hashedPassword,
		OrganizationRole: "OrganizationDeveloper",
		IsActive:         false, // User is inactive
	}

	mockRepo := new(MockUserRepository)
	mockRepo.On("GetByEmail", mock.Anything, testUser.Email).Return(testUser, nil)

	tokenManager, _ := auth.NewTokenManager("test-secret-key-minimum-32-characters-long", 0)
	rateLimiter := NewRateLimiter()
	defer rateLimiter.Stop()
	logger := zap.NewNop()

	handler := NewLoginHandler(mockRepo, tokenManager, rateLimiter, logger)

	loginReq := LoginRequest{
		Email:    testUser.Email,
		Password: "testpassword",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)
	assert.Equal(t, "account is inactive", response.Error)

	mockRepo.AssertExpectations(t)
}

func TestLoginHandler_DatabaseError(t *testing.T) {
	mockRepo := new(MockUserRepository)
	mockRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(nil, errors.New("database connection error"))

	tokenManager, _ := auth.NewTokenManager("test-secret-key-minimum-32-characters-long", 0)
	rateLimiter := NewRateLimiter()
	defer rateLimiter.Stop()
	logger := zap.NewNop()

	handler := NewLoginHandler(mockRepo, tokenManager, rateLimiter, logger)

	loginReq := LoginRequest{
		Email:    "test@example.com",
		Password: "testpassword",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Should return generic error to prevent enumeration
	var response ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid email or password", response.Error)

	mockRepo.AssertExpectations(t)
}

func TestLoginHandler_MethodNotAllowed(t *testing.T) {
	mockRepo := new(MockUserRepository)
	tokenManager, _ := auth.NewTokenManager("test-secret-key-minimum-32-characters-long", 0)
	rateLimiter := NewRateLimiter()
	defer rateLimiter.Stop()
	logger := zap.NewNop()

	handler := NewLoginHandler(mockRepo, tokenManager, rateLimiter, logger)

	// Test GET request (should be POST only)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)

	var response ErrorResponse
	json.NewDecoder(rec.Body).Decode(&response)
	assert.Equal(t, "method not allowed", response.Error)
}

func TestLoginHandler_SuccessfulLoginResetsRateLimit(t *testing.T) {
	password := "testpassword123"
	hashedPassword, _ := auth.HashPassword(password, 0)

	testUser := &models.User{
		ID:               uuid.New(),
		Email:            "test@example.com",
		PasswordHash:     hashedPassword,
		OrganizationRole: "OrganizationDeveloper",
		IsActive:         true,
	}

	mockRepo := new(MockUserRepository)
	mockRepo.On("GetByEmail", mock.Anything, testUser.Email).Return(testUser, nil)

	tokenManager, _ := auth.NewTokenManager("test-secret-key-minimum-32-characters-long", 0)
	rateLimiter := NewRateLimiter()
	defer rateLimiter.Stop()
	logger := zap.NewNop()

	handler := NewLoginHandler(mockRepo, tokenManager, rateLimiter, logger)

	// Make 5 failed attempts
	wrongLoginReq := LoginRequest{
		Email:    testUser.Email,
		Password: "wrongpassword",
	}
	wrongBody, _ := json.Marshal(wrongLoginReq)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(wrongBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		e := echo.New()
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		_ = handler.Login(c)
	}

	// Now make a successful login
	correctLoginReq := LoginRequest{
		Email:    testUser.Email,
		Password: password,
	}
	correctBody, _ := json.Marshal(correctLoginReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(correctBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = handler.Login(c)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Rate limit should be reset, so we should be able to make more attempts
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(wrongBody))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	e2 := echo.New()
	rec2 := httptest.NewRecorder()
	c2 := e2.NewContext(req2, rec2)

	_ = handler.Login(c2)

	// Should get Unauthorized, not TooManyRequests
	assert.Equal(t, http.StatusUnauthorized, rec2.Code)

	mockRepo.AssertExpectations(t)
}
