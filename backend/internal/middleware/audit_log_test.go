package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
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
	userID uuid.UUID,
	action, resourceType string,
	resourceID uuid.UUID,
	httpRequest *http.Request,
	details json.RawMessage,
) error {
	args := m.Called(ctx, userID, action, resourceType, resourceID, httpRequest, details)
	return args.Error(0)
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
		middleware := NewAuditLogMiddleware(mockAuditService, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans", nil)
		req.RemoteAddr = "192.168.1.1:54321"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set user in context (as AuthMiddleware would)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, userID, services.ActionRead, services.ResourceTypeChangePlan, uuid.Nil, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockAuditService.AssertExpectations(t)
	})

	// Test unauthenticated request (should not log)
	t.Run("UnauthenticatedRequest", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger)

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
		middleware := NewAuditLogMiddleware(mockAuditService, logger)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusCreated, map[string]string{"id": uuid.New().String()})
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, userID, services.ActionCreate, services.ResourceTypeChangePlan, uuid.Nil, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
		mockAuditService.AssertExpectations(t)
	})

	// Test request with resource ID
	t.Run("RequestWithResourceID", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger)

		resourceID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans/"+resourceID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"id": resourceID.String()})
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, userID, services.ActionRead, services.ResourceTypeChangePlan, resourceID, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockAuditService.AssertExpectations(t)
	})

	// Test error response
	t.Run("ErrorResponse", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans/invalid-id", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, userID, services.ActionRead, services.ResourceTypeChangePlan, uuid.Nil, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.Error(t, err)
		// Verify error is HTTPError with 404 status
		httpErr, ok := err.(*echo.HTTPError)
		require.True(t, ok, "Error should be HTTPError")
		assert.Equal(t, http.StatusNotFound, httpErr.Code)
		mockAuditService.AssertExpectations(t)
	})

	// Test skip health check
	t.Run("SkipHealthCheck", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger)

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
		middleware := NewAuditLogMiddleware(mockAuditService, logger)

		resourceID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans/"+resourceID.String()+"/approve", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "approved"})
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, userID, services.ActionApproveChangePlan, services.ResourceTypeChangePlan, resourceID, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockAuditService.AssertExpectations(t)
	})

	// Test execute action
	t.Run("ExecuteAction", func(t *testing.T) {
		mockAuditService := new(MockAuditLogService)
		middleware := NewAuditLogMiddleware(mockAuditService, logger)

		resourceID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/change-plans/"+resourceID.String()+"/execute", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", claims)

		handler := func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"status": "executed"})
		}

		mockAuditService.On("LogWithHTTP", mock.Anything, userID, services.ActionExecuteChangePlan, services.ResourceTypeChangePlan, resourceID, mock.AnythingOfType("*http.Request"), mock.Anything).
			Return(nil)

		err := middleware.AuditLog(handler)(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		mockAuditService.AssertExpectations(t)
	})
}
