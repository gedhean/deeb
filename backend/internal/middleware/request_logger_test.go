package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nsxbet/deeb/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestLoggerWithConfig_SkipPaths(t *testing.T) {
	// Initialize logger for testing
	err := logger.Initialize(logger.Config{
		Level:  "info",
		Format: "json",
	})
	require.NoError(t, err)

	e := echo.New()
	e.Use(middleware.RequestID())
	e.Use(RequestLoggerWithConfig(RequestLoggerConfig{
		SkipPaths: []string{"/api/v1/health", "/metrics"},
	}))

	// Test endpoint
	e.GET("/api/v1/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.GET("/api/v1/users", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "success"})
	})

	t.Run("handles health endpoint without logging", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		rec := httptest.NewRecorder()

		// Execute request
		e.ServeHTTP(rec, req)

		// Verify response is successful (the middleware still processes the request)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify correlation ID is still set even when skipping logging
		assert.NotEmpty(t, rec.Header().Get(CorrelationIDHeader))
		assert.NotEmpty(t, rec.Header().Get(echo.HeaderXRequestID))
	})

	t.Run("handles regular endpoints", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		rec := httptest.NewRecorder()

		// Execute request
		e.ServeHTTP(rec, req)

		// Verify response is successful
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify correlation ID is set
		assert.NotEmpty(t, rec.Header().Get(CorrelationIDHeader))
	})

	t.Run("handles metrics endpoint without logging", func(t *testing.T) {
		// Add metrics endpoint
		e.GET("/metrics", func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"metrics": "data"})
		})

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotEmpty(t, rec.Header().Get(CorrelationIDHeader))
	})
}

func TestRequestLogger_DefaultBehavior(t *testing.T) {
	// Initialize logger for testing
	err := logger.Initialize(logger.Config{
		Level:  "info",
		Format: "json",
	})
	require.NoError(t, err)

	e := echo.New()
	e.Use(middleware.RequestID())
	e.Use(RequestLogger()) // Default config with no skip paths

	e.GET("/api/v1/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	t.Run("handles all endpoints by default", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		rec := httptest.NewRecorder()

		// Execute request
		e.ServeHTTP(rec, req)

		// Verify response is successful
		assert.Equal(t, http.StatusOK, rec.Code)

		// Verify headers are set
		assert.NotEmpty(t, rec.Header().Get(CorrelationIDHeader))
		assert.NotEmpty(t, rec.Header().Get(echo.HeaderXRequestID))
	})
}

func TestRequestLoggerWithConfig_EmptySkipPaths(t *testing.T) {
	// Initialize logger for testing
	err := logger.Initialize(logger.Config{
		Level:  "info",
		Format: "json",
	})
	require.NoError(t, err)

	e := echo.New()
	e.Use(middleware.RequestID())
	e.Use(RequestLoggerWithConfig(RequestLoggerConfig{
		SkipPaths: []string{}, // Empty skip paths
	}))

	e.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "test"})
	})

	t.Run("handles all endpoints with empty skip paths", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		// Execute request
		e.ServeHTTP(rec, req)

		// Verify response and headers
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotEmpty(t, rec.Header().Get(CorrelationIDHeader))
	})
}

func TestRequestLoggerWithConfig_MultipleSkipPaths(t *testing.T) {
	// Initialize logger for testing
	err := logger.Initialize(logger.Config{
		Level:  "info",
		Format: "json",
	})
	require.NoError(t, err)

	e := echo.New()
	e.Use(middleware.RequestID())
	e.Use(RequestLoggerWithConfig(RequestLoggerConfig{
		SkipPaths: []string{
			"/api/v1/health",
			"/metrics",
			"/ready",
			"/healthz",
		},
	}))

	// Add all skip endpoints
	e.GET("/api/v1/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.GET("/metrics", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.GET("/ready", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.GET("/api/v1/data", func(c echo.Context) error {
		return c.String(http.StatusOK, "data")
	})

	testCases := []struct {
		name     string
		path     string
		expected int
	}{
		{"health endpoint skipped", "/api/v1/health", http.StatusOK},
		{"metrics endpoint skipped", "/metrics", http.StatusOK},
		{"ready endpoint skipped", "/ready", http.StatusOK},
		{"healthz endpoint skipped", "/healthz", http.StatusOK},
		{"data endpoint not skipped", "/api/v1/data", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.expected, rec.Code)
			assert.NotEmpty(t, rec.Header().Get(CorrelationIDHeader))
		})
	}
}
