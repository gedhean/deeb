// +build integration

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/handlers"
	"github.com/nsxbet/deeb/internal/repository/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestDB sets up a test database connection
func setupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	// Get database URL from environment, fallback to default test DB
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://deeb:deeb@localhost:5432/deeb_test?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	require.NoError(t, err, "failed to connect to test database")

	// Set connection pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	return db
}

// cleanupTestDB cleans up test data
func cleanupTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()

	// Clean up users table
	_, err := db.Exec("DELETE FROM users WHERE email LIKE 'test-%@example.com' OR email = 'test@example.com'")
	require.NoError(t, err, "failed to clean up test users")
}

// teardownTestDB closes the database connection
func teardownTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()
	err := db.Close()
	require.NoError(t, err, "failed to close database connection")
}

func TestIntegration_RegisterAndLogin_Success(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()

	// Test Registration
	registerHandler := handlers.NewRegisterHandler(userRepo, logger)

	email := fmt.Sprintf("test-%d@example.com", time.Now().Unix())
	password := "TestPassword123!"

	registerReq := handlers.RegisterRequest{
		Email:    email,
		Password: password,
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := registerHandler.Register(c)
	require.NoError(t, err)

	if rec.Code != http.StatusCreated {
		t.Logf("Registration failed with status %d, body: %s", rec.Code, rec.Body.String())
	}
	require.Equal(t, http.StatusCreated, rec.Code, "registration should succeed")

	var registerResp handlers.RegisterResponse
	err = json.NewDecoder(rec.Body).Decode(&registerResp)
	require.NoError(t, err)
	assert.Equal(t, email, registerResp.User.Email)
	assert.Equal(t, "OrganizationDeveloper", registerResp.User.OrganizationRole)
	assert.True(t, registerResp.User.IsActive)
	assert.Equal(t, "Active", registerResp.User.Status)

	// Test Login
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing")
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, rateLimiter, logger)

	loginReq := handlers.LoginRequest{
		Email:    email,
		Password: password,
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReqHTTP.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e2 := echo.New()
	loginRec := httptest.NewRecorder()
	c2 := e2.NewContext(loginReqHTTP, loginRec)

	err = loginHandler.Login(c2)
	require.NoError(t, err)

	if loginRec.Code != http.StatusOK {
		t.Logf("Login failed with status %d, body: %s", loginRec.Code, loginRec.Body.String())
	}
	require.Equal(t, http.StatusOK, loginRec.Code, "login should succeed")

	var loginResp handlers.LoginResponse
	err = json.NewDecoder(loginRec.Body).Decode(&loginResp)
	require.NoError(t, err)
	assert.Equal(t, email, loginResp.User.Email)

	// Verify JWT cookie is set
	cookies := loginRec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "deeb_auth_token", cookies[0].Name)
	assert.NotEmpty(t, cookies[0].Value)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, 86400, cookies[0].MaxAge)

	// Verify token is valid
	claims, err := tokenManager.ValidateToken(cookies[0].Value)
	require.NoError(t, err)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, "OrganizationDeveloper", claims.OrganizationRole)
}

func TestIntegration_Register_DuplicateEmail(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()
	registerHandler := handlers.NewRegisterHandler(userRepo, logger)

	email := fmt.Sprintf("test-%d@example.com", time.Now().Unix())
	password := "TestPassword123!"

	// First registration - should succeed
	registerReq := handlers.RegisterRequest{
		Email:    email,
		Password: password,
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := registerHandler.Register(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Second registration with same email - should fail
	body2, _ := json.Marshal(registerReq)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body2))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e2 := echo.New()
	rec2 := httptest.NewRecorder()
	c2 := e2.NewContext(req2, rec2)

	err = registerHandler.Register(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, rec2.Code)

	var errorResp handlers.ErrorResponse
	err = json.NewDecoder(rec2.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Equal(t, "email already registered", errorResp.Error)
}

func TestIntegration_Login_InvalidCredentials(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()

	// First, register a user
	registerHandler := handlers.NewRegisterHandler(userRepo, logger)
	email := fmt.Sprintf("test-%d@example.com", time.Now().Unix())
	correctPassword := "TestPassword123!"

	registerReq := handlers.RegisterRequest{
		Email:    email,
		Password: correctPassword,
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := registerHandler.Register(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Now try to login with wrong password
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing")
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, rateLimiter, logger)

	loginReq := handlers.LoginRequest{
		Email:    email,
		Password: "WrongPassword456!",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReqHTTP.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e2 := echo.New()
	loginRec := httptest.NewRecorder()
	c2 := e2.NewContext(loginReqHTTP, loginRec)

	err = loginHandler.Login(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, loginRec.Code)

	var errorResp handlers.ErrorResponse
	err = json.NewDecoder(loginRec.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Equal(t, "invalid email or password", errorResp.Error)

	// Try with non-existent user
	loginReq2 := handlers.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "AnyPassword123!",
	}
	loginBody2, _ := json.Marshal(loginReq2)
	loginReqHTTP2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody2))
	loginReqHTTP2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e3 := echo.New()
	loginRec2 := httptest.NewRecorder()
	c3 := e3.NewContext(loginReqHTTP2, loginRec2)

	err = loginHandler.Login(c3)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, loginRec2.Code)

	var errorResp2 handlers.ErrorResponse
	err = json.NewDecoder(loginRec2.Body).Decode(&errorResp2)
	require.NoError(t, err)
	assert.Equal(t, "invalid email or password", errorResp2.Error)
}

func TestIntegration_Login_RateLimiting(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()

	// First, register a user
	registerHandler := handlers.NewRegisterHandler(userRepo, logger)
	email := fmt.Sprintf("test-%d@example.com", time.Now().Unix())
	password := "TestPassword123!"

	registerReq := handlers.RegisterRequest{
		Email:    email,
		Password: password,
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := registerHandler.Register(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Setup login handler
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing")
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, rateLimiter, logger)

	// Make 10 failed login attempts (the rate limit)
	wrongLoginReq := handlers.LoginRequest{
		Email:    email,
		Password: "WrongPassword123!",
	}
	wrongBody, _ := json.Marshal(wrongLoginReq)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(wrongBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		e := echo.New()
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := loginHandler.Login(c)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "Attempt %d should return 401", i+1)
	}

	// 11th attempt should be rate limited
	req11 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(wrongBody))
	req11.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	e11 := echo.New()
	rec11 := httptest.NewRecorder()
	c11 := e11.NewContext(req11, rec11)

	err = loginHandler.Login(c11)
	require.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec11.Code)

	var errorResp handlers.ErrorResponse
	err = json.NewDecoder(rec11.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Equal(t, "too many login attempts, please try again later", errorResp.Error)

	// Verify that even the correct password is now rate limited
	correctLoginReq := handlers.LoginRequest{
		Email:    email,
		Password: password,
	}
	correctBody, _ := json.Marshal(correctLoginReq)
	req12 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(correctBody))
	req12.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	e12 := echo.New()
	rec12 := httptest.NewRecorder()
	c12 := e12.NewContext(req12, rec12)

	err = loginHandler.Login(c12)
	require.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, rec12.Code, "Rate limit should apply even for correct password")
}

func TestIntegration_Login_InactiveUser(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()

	// Register a user
	registerHandler := handlers.NewRegisterHandler(userRepo, logger)
	email := fmt.Sprintf("test-%d@example.com", time.Now().Unix())
	password := "TestPassword123!"

	registerReq := handlers.RegisterRequest{
		Email:    email,
		Password: password,
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := registerHandler.Register(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)

	// Deactivate the user
	ctx := context.Background()
	_, err = db.ExecContext(ctx, "UPDATE users SET status = 'Inactive' WHERE email = $1", email)
	require.NoError(t, err)

	// Try to login with inactive account
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing")
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, rateLimiter, logger)

	loginReq := handlers.LoginRequest{
		Email:    email,
		Password: password,
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReqHTTP.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e2 := echo.New()
	loginRec := httptest.NewRecorder()
	c2 := e2.NewContext(loginReqHTTP, loginRec)

	err = loginHandler.Login(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, loginRec.Code)

	var errorResp handlers.ErrorResponse
	err = json.NewDecoder(loginRec.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Equal(t, "account is inactive", errorResp.Error)
}
