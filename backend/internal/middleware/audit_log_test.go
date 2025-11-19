package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/config"
	"github.com/nsxbet/deeb/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockAuditLogService is a mock implementation of AuditLogServiceInterface
type MockAuditLogService struct {
	mock.Mock
}

func (m *MockAuditLogService) LogWithHTTP(
	ctx context.Context,
	userID *uuid.UUID,
	action, resourceType string,
	resourceID uuid.UUID,
	httpRequest *http.Request,
	details json.RawMessage,
) error {
	args := m.Called(ctx, userID, action, resourceType, resourceID, httpRequest, details)
	return args.Error(0)
}

// newTestAuditConfig creates a test audit logging configuration with allow-all policy
func newTestAuditConfig() *config.AuditLoggingConfig {
	return &config.AuditLoggingConfig{
		Enabled:          true,
		DefaultPolicy:    "allow", // Allow all for existing tests
		LogAllPaths:      false,
		SkipPaths:        []string{},
		Resources:        make(map[string]config.ResourceRule),
		RedactionEnabled: true,
	}
}

func TestAuditLogMiddleware_AuditLog(t *testing.T) {
	logger := zap.NewNop()
	e := echo.New()

	userID := uuid.New()
	claims := &auth.Claims{
		UserID:           userID,
		Email:            "test@example.com",
		OrganizationRole: "OrganizationDeveloper",
	}

	// Test authenticated request
	t.Run("AuthenticatedRequest", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, newTestAuditConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans", nil)
		req.RemoteAddr = "192.168.1.1:54321"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set user in context (as AuthMiddleware would)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, &userID, services.ActionRead, services.ResourceTypeChangePlan, uuid.Nil, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		mockAuditService.AssertExpectations(t)
	})

	// Test unauthenticated request (should not log)
	t.Run("UnauthenticatedRequest", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, newTestAuditConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Don't set user in context

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		}

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// Should not call audit service for unauthenticated requests
		mockAuditService.AssertNotCalled(t, "LogWithHTTP")
	})

	// Test POST request
	t.Run("POSTRequest", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, newTestAuditConfig())

		req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusCreated, map[string]string{"id": uuid.New().String()})
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, &userID, services.ActionCreate, services.ResourceTypeChangePlan, uuid.Nil, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		mockAuditService.AssertExpectations(t)
	})

	// Test request with resource ID
	t.Run("RequestWithResourceID", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, newTestAuditConfig())

		resourceID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans/"+resourceID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"id": resourceID.String()})
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, &userID, services.ActionRead, services.ResourceTypeChangePlan, resourceID, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		mockAuditService.AssertExpectations(t)
	})

	// Test error response
	t.Run("ErrorResponse", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, newTestAuditConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans/invalid-id", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, &userID, services.ActionRead, services.ResourceTypeChangePlan, uuid.Nil, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.Error(t, err)
		// Verify error is HTTPError with 404 status
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok, "Error should be HTTPError")
		assert.Equal(t, http.StatusNotFound, httpErr.Code)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		mockAuditService.AssertExpectations(t)
	})

	// Test skip health check
	t.Run("SkipHealthCheck", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, newTestAuditConfig())

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		}

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// Should not call audit service for health checks
		mockAuditService.AssertNotCalled(t, "LogWithHTTP")
	})

	// Test approve action
	t.Run("ApproveAction", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, newTestAuditConfig())

		resourceID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans/"+resourceID.String()+"/approve", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "approved"})
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, &userID, services.ActionApproveChangePlan, services.ResourceTypeChangePlan, resourceID, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		mockAuditService.AssertExpectations(t)
	})

	// Test execute action
	t.Run("ExecuteAction", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, newTestAuditConfig())

		resourceID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans/"+resourceID.String()+"/execute", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "executed"})
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, &userID, services.ActionExecuteChangePlan, services.ResourceTypeChangePlan, resourceID, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		mockAuditService.AssertExpectations(t)
	})

	// Test with enrichment
	t.Run("WithEnrichment", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, newTestAuditConfig())

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		// Handler sets enrichment
		handler := func(c echo.Context) error {
			c.Set("audit_enrichment", &AuditEnrichment{
				Email:  "user@example.com",
				Result: "success",
			})
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		}

		// Verify enrichment is included in details
		// Middleware detects login from path
		mockAuditService.On("LogWithHTTP", mock.Anything, &userID, services.ActionLogin, services.ResourceTypeUser, uuid.Nil, mock.AnythingOfType("*http.Request"), mock.MatchedBy(func(details json.RawMessage) bool {
			var d map[string]interface{}
			json.Unmarshal(details, &d)
			return d["email"] == "user@example.com" && d["result"] == "success"
		})).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		mockAuditService.AssertExpectations(t)
	})

	// Test with custom fields
	t.Run("WithCustomFields", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, newTestAuditConfig())

		req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			c.Set("audit_enrichment", &AuditEnrichment{
				Custom: map[string]interface{}{
					"title":       "Database Migration",
					"risk_level":  "high",
					"change_type": "ddl",
				},
			})
			return c.JSON(http.StatusCreated, map[string]string{"id": uuid.New().String()})
		}

		// Verify custom fields are merged
		mockAuditService.On("LogWithHTTP", mock.Anything, &userID, services.ActionCreate, services.ResourceTypeChangePlan, uuid.Nil, mock.AnythingOfType("*http.Request"), mock.MatchedBy(func(details json.RawMessage) bool {
			var d map[string]interface{}
			json.Unmarshal(details, &d)
			return d["title"] == "Database Migration" && d["risk_level"] == "high" && d["change_type"] == "ddl"
		})).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		mockAuditService.AssertExpectations(t)
	})

	// Test result override
	t.Run("ResultOverride", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, newTestAuditConfig())

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		// Handler sets failure result explicitly even though HTTP status is 401
		handler := func(c echo.Context) error {
			c.Set("audit_enrichment", &AuditEnrichment{
				Email:         "user@example.com",
				FailureReason: "invalid_credentials",
				Result:        "failure",
			})
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		}

		// Verify result is "failure" from enrichment
		// Middleware detects login from path
		mockAuditService.On("LogWithHTTP", mock.Anything, &userID, services.ActionLogin, services.ResourceTypeUser, uuid.Nil, mock.AnythingOfType("*http.Request"), mock.MatchedBy(func(details json.RawMessage) bool {
			var d map[string]interface{}
			json.Unmarshal(details, &d)
			return d["result"] == "failure" && d["failure_reason"] == "invalid_credentials"
		})).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		mockAuditService.AssertExpectations(t)
	})
}

func TestAuditLogMiddleware_Redaction(t *testing.T) {
	logger := zap.NewNop()
	mockAuditService := new(MockAuditLogService)
	middleware := NewAuditLogMiddleware(mockAuditService, logger, newTestAuditConfig())

	t.Run("RedactsPasswordInEnrichment", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set user in context
		userID := uuid.New()
		claims := &auth.Claims{UserID: userID}
		c.Set("user", claims)

		// Set enrichment with password
		enrichment := &AuditEnrichment{
			Email: "test@example.com",
			Custom: map[string]interface{}{
				"password":     "supersecret123",
				"new_password": "newsecret456",
				"username":     "testuser",
			},
		}
		c.Set("audit_enrichment", enrichment)

		// Handler that sets status
		handler := func(c echo.Context) error {
			return c.String(http.StatusCreated, "created")
		}

		// Mock expects redacted details
		mockAuditService.On("LogWithHTTP",
			mock.Anything,
			&userID,
			services.ActionCreate,
			"auth",
			uuid.Nil,
			mock.Anything,
			mock.MatchedBy(func(details json.RawMessage) bool {
				var detailsMap map[string]interface{}
				json.Unmarshal(details, &detailsMap)
				// Verify password is redacted
				password, hasPassword := detailsMap["password"]
				newPassword, hasNewPassword := detailsMap["new_password"]
				username, hasUsername := detailsMap["username"]

				return hasPassword && password == "[REDACTED]" &&
					hasNewPassword && newPassword == "[REDACTED]" &&
					hasUsername && username == "testuser" // non-sensitive field preserved
			}),
		).Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		mockAuditService.AssertExpectations(t)
	})

	t.Run("RedactsConnectionString", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/database-instances", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set user in context
		userID := uuid.New()
		claims := &auth.Claims{UserID: userID}
		c.Set("user", claims)

		// Set enrichment with connection string
		enrichment := &AuditEnrichment{
			Custom: map[string]interface{}{
				"connection_string": "postgresql://user:secret@localhost:5432/db",
				"host":              "localhost",
			},
		}
		c.Set("audit_enrichment", enrichment)

		// Handler that sets status
		handler := func(c echo.Context) error {
			return c.String(http.StatusCreated, "created")
		}

		// Mock expects redacted connection string
		mockAuditService.On("LogWithHTTP",
			mock.Anything,
			&userID,
			services.ActionCreate,
			"database_instance",
			uuid.Nil,
			mock.Anything,
			mock.MatchedBy(func(details json.RawMessage) bool {
				var detailsMap map[string]interface{}
				json.Unmarshal(details, &detailsMap)
				// Verify connection_string is redacted
				connStr, hasConnStr := detailsMap["connection_string"]
				host, hasHost := detailsMap["host"]

				return hasConnStr && connStr == "[REDACTED]" &&
					hasHost && host == "localhost"
			}),
		).Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		mockAuditService.AssertExpectations(t)
	})

	t.Run("RedactionCanBeDisabled", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Disable redaction
		middleware.SetRedactionEnabled(false)
		defer middleware.SetRedactionEnabled(true) // Re-enable for other tests

		// Set user in context
		userID := uuid.New()
		claims := &auth.Claims{UserID: userID}
		c.Set("user", claims)

		// Set enrichment with password
		enrichment := &AuditEnrichment{
			Custom: map[string]interface{}{
				"password": "supersecret123",
			},
		}
		c.Set("audit_enrichment", enrichment)

		// Handler that sets status
		handler := func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		}

		// Mock expects UN-redacted details (since redaction is disabled)
		mockAuditService.On("LogWithHTTP",
			mock.Anything,
			&userID,
			services.ActionCreate,
			"test",
			uuid.Nil,
			mock.Anything,
			mock.MatchedBy(func(details json.RawMessage) bool {
				var detailsMap map[string]interface{}
				json.Unmarshal(details, &detailsMap)
				// Verify password is NOT redacted
				password, hasPassword := detailsMap["password"]
				return hasPassword && password == "supersecret123"
			}),
		).Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		mockAuditService.AssertExpectations(t)
	})
}

// TestAuditLogMiddleware_Configuration tests the configuration filtering logic
func TestAuditLogMiddleware_Configuration(t *testing.T) {
	logger := zap.NewNop()
	e := echo.New()

	userID := uuid.New()
	claims := &auth.Claims{
		UserID:           userID,
		Email:            "test@example.com",
		OrganizationRole: "OrganizationDeveloper",
	}

	t.Run("DenyPolicyWithoutConfiguration", func(t *testing.T) {
		// Config with deny policy and no resources configured
		cfg := &config.AuditLoggingConfig{
			Enabled:          true,
			DefaultPolicy:    "deny",
			LogAllPaths:      false,
			SkipPaths:        []string{},
			Resources:        make(map[string]config.ResourceRule),
			RedactionEnabled: true,
		}

		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, cfg)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		}

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// Should NOT log with deny policy and no configuration
		mockAuditService.AssertNotCalled(t, "LogWithHTTP")
	})

	t.Run("AllowPolicyWithoutConfiguration", func(t *testing.T) {
		// Config with allow policy and no resources configured
		cfg := &config.AuditLoggingConfig{
			Enabled:          true,
			DefaultPolicy:    "allow",
			LogAllPaths:      false,
			SkipPaths:        []string{},
			Resources:        make(map[string]config.ResourceRule),
			RedactionEnabled: true,
		}

		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, cfg)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, &userID, services.ActionRead, services.ResourceTypeChangePlan, uuid.Nil, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		// SHOULD log with allow policy
		mockAuditService.AssertExpectations(t)
	})

	t.Run("ResourceEnabledWithSpecificActions", func(t *testing.T) {
		// Config with resource enabled but only specific actions
		cfg := &config.AuditLoggingConfig{
			Enabled:       true,
			DefaultPolicy: "deny",
			LogAllPaths:   false,
			SkipPaths:     []string{},
			Resources: map[string]config.ResourceRule{
				"change_plan": {
					Enabled: true,
					Actions: []string{"create", "update", "delete"}, // No "read"
				},
			},
			RedactionEnabled: true,
		}

		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, cfg)

		// Test READ action (not in the allowed list)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		}

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		// Should NOT log READ action since it's not in the actions list
		mockAuditService.AssertNotCalled(t, "LogWithHTTP")
	})

	t.Run("ResourceEnabledWithAllowedAction", func(t *testing.T) {
		// Config with resource enabled and action in the list
		cfg := &config.AuditLoggingConfig{
			Enabled:       true,
			DefaultPolicy: "deny",
			LogAllPaths:   false,
			SkipPaths:     []string{},
			Resources: map[string]config.ResourceRule{
				"change_plan": {
					Enabled: true,
					Actions: []string{"create", "update", "delete"},
				},
			},
			RedactionEnabled: true,
		}

		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, cfg)

		// Test CREATE action (in the allowed list)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusCreated, map[string]string{"id": uuid.New().String()})
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, &userID, services.ActionCreate, services.ResourceTypeChangePlan, uuid.Nil, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
		// Wait for async goroutine to complete
		time.Sleep(50 * time.Millisecond)
		// SHOULD log CREATE action since it's in the actions list
		mockAuditService.AssertExpectations(t)
	})

	t.Run("ResourceDisabled", func(t *testing.T) {
		// Config with resource explicitly disabled
		cfg := &config.AuditLoggingConfig{
			Enabled:       true,
			DefaultPolicy: "deny",
			LogAllPaths:   false,
			SkipPaths:     []string{},
			Resources: map[string]config.ResourceRule{
				"change_plan": {
					Enabled: false,
					Actions: []string{}, // Doesn't matter, resource is disabled
				},
			},
			RedactionEnabled: true,
		}

		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, cfg)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusCreated, map[string]string{"id": uuid.New().String()})
		}

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
		// Should NOT log when resource is disabled
		mockAuditService.AssertNotCalled(t, "LogWithHTTP")
	})

	t.Run("CustomSkipPaths", func(t *testing.T) {
		// Config with custom skip paths
		cfg := &config.AuditLoggingConfig{
			Enabled:       true,
			DefaultPolicy: "allow",
			LogAllPaths:   false,
			SkipPaths: []string{
				"/api/v1/metrics",
				"/api/v1/debug/*",
			},
			Resources:        make(map[string]config.ResourceRule),
			RedactionEnabled: true,
		}

		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, cfg)

		// Test exact match skip
		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
		rec1 := httptest.NewRecorder()
		c1 := e.NewContext(req1, rec1)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		}

		err := middleware.AuditLog(handler)(c1)
		require.NoError(t, err)
		mockAuditService.AssertNotCalled(t, "LogWithHTTP")

		// Test wildcard match skip
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/debug/stats", nil)
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req2, rec2)

		err = middleware.AuditLog(handler)(c2)
		require.NoError(t, err)
		mockAuditService.AssertNotCalled(t, "LogWithHTTP")
	})

	t.Run("AuditLoggingDisabled", func(t *testing.T) {
		// Config with audit logging globally disabled
		cfg := &config.AuditLoggingConfig{
			Enabled:          false, // Disabled
			DefaultPolicy:    "allow",
			LogAllPaths:      false,
			SkipPaths:        []string{},
			Resources:        make(map[string]config.ResourceRule),
			RedactionEnabled: true,
		}

		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger, cfg)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusCreated, map[string]string{"id": uuid.New().String()})
		}

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
		// Should NOT log when audit logging is globally disabled
		mockAuditService.AssertNotCalled(t, "LogWithHTTP")
	})
}
