//go:build integration
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
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/handlers"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository/postgres"
	"github.com/nsxbet/deeb/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

// setupTestDB sets up a test database connection using testcontainers
func setupTestDB(t *testing.T) (*sqlx.DB, func()) {
	t.Helper()

	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgrescontainer.Run(ctx,
		"postgres:16-alpine",
		postgrescontainer.WithDatabase("deeb_test"),
		postgrescontainer.WithUsername("deeb"),
		postgrescontainer.WithPassword("deeb"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err, "failed to start postgres container")

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	// Connect to database
	db, err := sqlx.Connect("postgres", connStr)
	require.NoError(t, err, "failed to connect to test database")

	// Set connection pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	// Run migrations
	runMigrations(t, db)

	// Return cleanup function
	cleanup := func() {
		db.Close()
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate postgres container: %v", err)
		}
	}

	return db, cleanup
}

// runMigrations runs the database migrations
func runMigrations(t *testing.T, db *sqlx.DB) {
	t.Helper()

	// Read and execute migration file
	migrationPath := filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err, "failed to read migration file")

	_, err = db.Exec(string(migrationSQL))
	require.NoError(t, err, "failed to run migrations")
}

// cleanupTestDB cleans up test data
func cleanupTestDB(t *testing.T, db *sqlx.DB) {
	t.Helper()

	// Clean up users table
	_, err := db.Exec("DELETE FROM users WHERE email LIKE 'test-%@example.com' OR email = 'test@example.com'")
	require.NoError(t, err, "failed to clean up test users")

	// Clean up audit logs table
	_, err = db.Exec("DELETE FROM audit_logs WHERE user_id IN (SELECT id FROM users WHERE email LIKE 'test-%@example.com' OR email = 'test@example.com')")
	require.NoError(t, err, "failed to clean up test audit logs")
}

func TestIntegration_RegisterAndLogin_Success(t *testing.T) {
	// Setup
	db, cleanup := setupTestDB(t)
	defer cleanup()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()
	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditService := services.NewAuditLogService(auditLogRepo, logger)

	// Create a dummy user first so the test user is not the first user
	// (first user gets OrganizationAdmin role)
	ctx := context.Background()
	dummyUser := &models.User{
		Email:            fmt.Sprintf("dummy-%d@example.com", time.Now().Unix()),
		PasswordHash:     "dummy_hash",
		OrganizationRole: models.OrganizationRoleDeveloper,
		IsActive:         true,
	}
	err := userRepo.Create(ctx, dummyUser)
	require.NoError(t, err, "failed to create dummy user")

	// Test Registration
	registerHandler := handlers.NewRegisterHandler(userRepo, auditService, logger)

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

	err = registerHandler.Register(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code, "registration should succeed")

	var registerResp handlers.RegisterResponse
	err = json.NewDecoder(rec.Body).Decode(&registerResp)
	require.NoError(t, err)
	assert.Equal(t, email, registerResp.User.Email)
	assert.Equal(t, models.OrganizationRoleDeveloper, registerResp.User.OrganizationRole)
	assert.True(t, registerResp.User.IsActive)

	// Test Login
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", 0)
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditService := services.NewAuditLogService(auditLogRepo, logger)

	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, rateLimiter, auditService, logger)

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
	require.Equal(t, http.StatusOK, loginRec.Code, "login should succeed")

	var loginResp handlers.LoginResponse
	err = json.NewDecoder(loginRec.Body).Decode(&loginResp)
	require.NoError(t, err)
	assert.Equal(t, email, loginResp.User.Email)
	assert.NotEmpty(t, loginRec.Result().Cookies(), "should have auth cookie")

	// Verify auth cookie
	cookies := loginRec.Result().Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "deeb_auth_token" {
			authCookie = cookie
			break
		}
	}
	require.NotNil(t, authCookie, "auth cookie should be set")
	assert.True(t, authCookie.HttpOnly, "auth cookie should be HttpOnly")
	assert.NotEmpty(t, authCookie.Value, "auth cookie should have a value")
}

func TestIntegration_Register_DuplicateEmail(t *testing.T) {
	// Setup
	db, cleanup := setupTestDB(t)
	defer cleanup()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()
	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditService := services.NewAuditLogService(auditLogRepo, logger)

	registerHandler := handlers.NewRegisterHandler(userRepo, auditService, logger)

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
	require.Equal(t, http.StatusCreated, rec.Code, "first registration should succeed")

	// Second registration with same email - should fail
	body2, _ := json.Marshal(registerReq)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body2))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)

	err = registerHandler.Register(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, rec2.Code, "duplicate registration should fail")

	var errorResp handlers.ErrorResponse
	err = json.NewDecoder(rec2.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp.Error, "already registered")
}

func TestIntegration_Login_InvalidPassword(t *testing.T) {
	// Setup
	db, cleanup := setupTestDB(t)
	defer cleanup()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()

	// Create user first
	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditService := services.NewAuditLogService(auditLogRepo, logger)

	registerHandler := handlers.NewRegisterHandler(userRepo, auditService, logger)

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

	// Try to login with wrong password
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", 0)
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditService := services.NewAuditLogService(auditLogRepo, logger)

	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, rateLimiter, auditService, logger)

	loginReq := handlers.LoginRequest{
		Email:    email,
		Password: "WrongPassword123!",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReqHTTP.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	loginRec := httptest.NewRecorder()
	c2 := e.NewContext(loginReqHTTP, loginRec)

	err = loginHandler.Login(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, loginRec.Code, "login with wrong password should fail")

	var errorResp handlers.ErrorResponse
	err = json.NewDecoder(loginRec.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp.Error, "invalid email or password")
}

func TestIntegration_Login_UserNotFound(t *testing.T) {
	// Setup
	db, cleanup := setupTestDB(t)
	defer cleanup()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()

	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", 0)
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditService := services.NewAuditLogService(auditLogRepo, logger)

	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, rateLimiter, auditService, logger)

	// Try to login with non-existent user
	email := fmt.Sprintf("test-%d@example.com", time.Now().Unix())
	loginReq := handlers.LoginRequest{
		Email:    email,
		Password: "TestPassword123!",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReqHTTP.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	e := echo.New()
	loginRec := httptest.NewRecorder()
	c := e.NewContext(loginReqHTTP, loginRec)

	err = loginHandler.Login(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, loginRec.Code, "login with non-existent user should fail")

	var errorResp handlers.ErrorResponse
	err = json.NewDecoder(loginRec.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp.Error, "invalid email or password")
}

func TestIntegration_Login_InactiveUser(t *testing.T) {
	// Setup
	db, cleanup := setupTestDB(t)
	defer cleanup()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()

	// Create user first
	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditService := services.NewAuditLogService(auditLogRepo, logger)

	registerHandler := handlers.NewRegisterHandler(userRepo, auditService, logger)

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
	_, err = db.ExecContext(ctx, "UPDATE users SET is_active = false WHERE email = $1", email)
	require.NoError(t, err)

	// Try to login with inactive account
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", 0)
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditService := services.NewAuditLogService(auditLogRepo, logger)

	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, rateLimiter, auditService, logger)

	loginReq := handlers.LoginRequest{
		Email:    email,
		Password: password,
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReqHTTP.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	loginRec := httptest.NewRecorder()
	c2 := e.NewContext(loginReqHTTP, loginRec)

	err = loginHandler.Login(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, loginRec.Code, "login with inactive account should fail")

	var errorResp handlers.ErrorResponse
	err = json.NewDecoder(loginRec.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp.Error, "inactive")
}

func TestIntegration_Auth_AuditLogging_LoginSuccess(t *testing.T) {
	// Setup
	db, cleanup := setupTestDB(t)
	defer cleanup()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()
	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditService := services.NewAuditLogService(auditLogRepo, logger)

	// Create user first
	registerHandler := handlers.NewRegisterHandler(userRepo, auditService, logger)

	email := fmt.Sprintf("test-%d@example.com", time.Now().Unix())
	password := "TestPassword123!"

	registerReq := handlers.RegisterRequest{
		Email:    email,
		Password: password,
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.RemoteAddr = "192.168.1.1:54321"

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := registerHandler.Register(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)

	var registerResp handlers.RegisterResponse
	err = json.NewDecoder(rec.Body).Decode(&registerResp)
	require.NoError(t, err)
	userID := registerResp.User.ID

	// Test successful login and verify audit log
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", 0)
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, rateLimiter, auditService, logger)

	loginReq := handlers.LoginRequest{
		Email:    email,
		Password: password,
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReqHTTP.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	loginReqHTTP.RemoteAddr = "192.168.1.1:54321"

	loginRec := httptest.NewRecorder()
	c2 := e.NewContext(loginReqHTTP, loginRec)

	err = loginHandler.Login(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, loginRec.Code)

	// Verify audit log entry was created
	ctx := context.Background()
	auditLogs, err := auditLogRepo.GetByUserID(ctx, userID, 10, 0)
	require.NoError(t, err)

	// Find login audit log entry
	var loginLogFound bool
	for _, log := range auditLogs {
		if log.Action == services.ActionLogin && log.UserID == userID {
			loginLogFound = true
			assert.Equal(t, services.ResourceTypeUser, log.ResourceType)
			assert.Equal(t, userID, log.ResourceID)
			assert.NotEmpty(t, log.IPAddress)
			assert.NotZero(t, log.Timestamp)
			break
		}
	}
	assert.True(t, loginLogFound, "Login audit log entry should be created")
}

func TestIntegration_Auth_AuditLogging_LoginFailure(t *testing.T) {
	// Setup
	db, cleanup := setupTestDB(t)
	defer cleanup()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()
	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditService := services.NewAuditLogService(auditLogRepo, logger)

	// Create user first
	registerHandler := handlers.NewRegisterHandler(userRepo, auditService, logger)

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

	var registerResp handlers.RegisterResponse
	err = json.NewDecoder(rec.Body).Decode(&registerResp)
	require.NoError(t, err)
	userID := registerResp.User.ID

	// Test failed login and verify audit log
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", 0)
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, rateLimiter, auditService, logger)

	loginReq := handlers.LoginRequest{
		Email:    email,
		Password: "WrongPassword123!",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReqHTTP.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	loginReqHTTP.RemoteAddr = "192.168.1.1:54321"

	loginRec := httptest.NewRecorder()
	c2 := e.NewContext(loginReqHTTP, loginRec)

	err = loginHandler.Login(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, loginRec.Code)

	// Verify audit log entry was created for failed login
	ctx := context.Background()
	auditLogs, err := auditLogRepo.GetByUserID(ctx, userID, 10, 0)
	require.NoError(t, err)

	// Find failed login audit log entry
	var failureLogFound bool
	for _, log := range auditLogs {
		if log.Action == services.ActionLogin && log.UserID == userID {
			failureLogFound = true
			assert.Equal(t, services.ResourceTypeUser, log.ResourceType)
			assert.NotEmpty(t, log.IPAddress)
			assert.NotZero(t, log.Timestamp)
			// Check details contain error message
			var details map[string]interface{}
			if err := json.Unmarshal(log.Details, &details); err == nil {
				assert.Contains(t, details, "failure_reason")
				assert.Contains(t, details, "error_message")
			}
			break
		}
	}
	assert.True(t, failureLogFound, "Failed login audit log entry should be created")
}

func TestIntegration_Auth_AuditLogging_Registration(t *testing.T) {
	// Setup
	db, cleanup := setupTestDB(t)
	defer cleanup()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()
	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditService := services.NewAuditLogService(auditLogRepo, logger)

	registerHandler := handlers.NewRegisterHandler(userRepo, auditService, logger)

	email := fmt.Sprintf("test-%d@example.com", time.Now().Unix())
	password := "TestPassword123!"

	registerReq := handlers.RegisterRequest{
		Email:    email,
		Password: password,
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.RemoteAddr = "192.168.1.1:54321"

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := registerHandler.Register(c)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, rec.Code)

	var registerResp handlers.RegisterResponse
	err = json.NewDecoder(rec.Body).Decode(&registerResp)
	require.NoError(t, err)
	userID := registerResp.User.ID

	// Verify audit log entry was created for registration
	ctx := context.Background()
	auditLogs, err := auditLogRepo.GetByUserID(ctx, userID, 10, 0)
	require.NoError(t, err)

	// Find registration audit log entry
	var registrationLogFound bool
	for _, log := range auditLogs {
		if log.Action == services.ActionRegister && log.UserID == userID {
			registrationLogFound = true
			assert.Equal(t, services.ResourceTypeUser, log.ResourceType)
			assert.Equal(t, userID, log.ResourceID)
			assert.NotEmpty(t, log.IPAddress)
			assert.NotZero(t, log.Timestamp)
			break
		}
	}
	assert.True(t, registrationLogFound, "Registration audit log entry should be created")
}

func TestIntegration_Auth_AuditLogging_Logout(t *testing.T) {
	// Setup
	db, cleanup := setupTestDB(t)
	defer cleanup()
	cleanupTestDB(t, db)
	defer cleanupTestDB(t, db)

	userRepo := postgres.NewUserRepository(db)
	logger := zap.NewNop()
	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditService := services.NewAuditLogService(auditLogRepo, logger)

	// Create user and login first
	registerHandler := handlers.NewRegisterHandler(userRepo, auditService, logger)

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

	var registerResp handlers.RegisterResponse
	err = json.NewDecoder(rec.Body).Decode(&registerResp)
	require.NoError(t, err)
	userID := registerResp.User.ID

	// Login to get token
	tokenManager, err := auth.NewTokenManager("test-secret-key-minimum-32-characters-long-for-testing", 0)
	require.NoError(t, err)

	rateLimiter := handlers.NewRateLimiter()
	defer rateLimiter.Stop()

	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, rateLimiter, auditService, logger)

	loginReq := handlers.LoginRequest{
		Email:    email,
		Password: password,
	}
	loginBody, _ := json.Marshal(loginReq)
	loginReqHTTP := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginReqHTTP.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	loginRec := httptest.NewRecorder()
	c2 := e.NewContext(loginReqHTTP, loginRec)

	err = loginHandler.Login(c2)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, loginRec.Code)

	// Get token from cookie
	cookies := loginRec.Result().Cookies()
	var authCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "deeb_auth_token" {
			authCookie = cookie
			break
		}
	}
	require.NotNil(t, authCookie)

	// Test logout
	logoutHandler := handlers.NewLogoutHandler(auditService, logger)

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.AddCookie(authCookie)
	logoutReq.RemoteAddr = "192.168.1.1:54321"

	// Set user in context (as AuthMiddleware would)
	claims, err := tokenManager.ValidateToken(authCookie.Value)
	require.NoError(t, err)
	logoutRec := httptest.NewRecorder()
	c3 := e.NewContext(logoutReq, logoutRec)
	c3.Set("user", claims)

	err = logoutHandler.Logout(c3)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, logoutRec.Code)

	// Verify audit log entry was created for logout
	ctx := context.Background()
	auditLogs, err := auditLogRepo.GetByUserID(ctx, userID, 10, 0)
	require.NoError(t, err)

	// Find logout audit log entry
	var logoutLogFound bool
	for _, log := range auditLogs {
		if log.Action == services.ActionLogout && log.UserID == userID {
			logoutLogFound = true
			assert.Equal(t, services.ResourceTypeUser, log.ResourceType)
			assert.Equal(t, userID, log.ResourceID)
			assert.NotEmpty(t, log.IPAddress)
			assert.NotZero(t, log.Timestamp)
			break
		}
	}
	assert.True(t, logoutLogFound, "Logout audit log entry should be created")
}
