package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = CustomErrorHandler

	// Add RequestID middleware
	e.Use(middleware.RequestID())
	e.Use(RequestLogger())

	// Test endpoint
	e.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"message": "success",
		})
	})

	t.Run("request ID generated for each request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		// Check that X-Request-ID header is set
		requestID := rec.Header().Get(echo.HeaderXRequestID)
		assert.NotEmpty(t, requestID, "X-Request-ID header should be set")
		// Echo's RequestID generates IDs that are at least 16 characters
		assert.GreaterOrEqual(t, len(requestID), 16, "Request ID should be at least 16 characters")

		// Check that X-Correlation-ID header is also set (for backwards compatibility)
		correlationID := rec.Header().Get(CorrelationIDHeader)
		assert.Equal(t, requestID, correlationID, "X-Correlation-ID should match X-Request-ID")
	})

	t.Run("request ID from header is used if provided", func(t *testing.T) {
		customRequestID := "custom-request-id-12345"
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(echo.HeaderXRequestID, customRequestID)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		// Request ID should be the one from header
		requestID := rec.Header().Get(echo.HeaderXRequestID)
		assert.Equal(t, customRequestID, requestID, "Should use request ID from header")
	})

	t.Run("request ID included in error responses", func(t *testing.T) {
		e := echo.New()
		e.HTTPErrorHandler = CustomErrorHandler
		e.Use(middleware.RequestID())
		e.Use(RequestLogger())

		// Test endpoint that returns error
		e.GET("/error", func(c echo.Context) error {
			return echo.NewHTTPError(http.StatusBadRequest, "Test error")
		})

		req := httptest.NewRequest(http.MethodGet, "/error", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		// Get request ID from header
		requestID := rec.Header().Get(echo.HeaderXRequestID)
		require.NotEmpty(t, requestID, "Request ID should be set")

		// Check that request ID is in error response
		var response ErrorResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		assert.Equal(t, requestID, response.RequestID, "Request ID should be in error response")
	})

	t.Run("request ID available in context", func(t *testing.T) {
		e := echo.New()
		e.HTTPErrorHandler = CustomErrorHandler
		e.Use(middleware.RequestID())
		e.Use(RequestLogger())

		var capturedRequestID string
		e.GET("/context", func(c echo.Context) error {
			// Try to get from Echo's context
			if id, ok := c.Get(echo.HeaderXRequestID).(string); ok {
				capturedRequestID = id
			}
			// Also check correlation ID key
			if id, ok := c.Get(CorrelationIDKey).(string); ok && capturedRequestID == "" {
				capturedRequestID = id
			}
			return c.JSON(http.StatusOK, map[string]string{"request_id": capturedRequestID})
		})

		req := httptest.NewRequest(http.MethodGet, "/context", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		// Get request ID from header
		requestID := rec.Header().Get(echo.HeaderXRequestID)
		require.NotEmpty(t, requestID, "Request ID should be set")

		// Check that request ID was captured from context
		assert.NotEmpty(t, capturedRequestID, "Request ID should be available in context")
		assert.Equal(t, requestID, capturedRequestID, "Request ID from context should match header")
	})

	t.Run("different requests get different request IDs", func(t *testing.T) {
		req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec1 := httptest.NewRecorder()

		req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec2 := httptest.NewRecorder()

		e.ServeHTTP(rec1, req1)
		e.ServeHTTP(rec2, req2)

		requestID1 := rec1.Header().Get(echo.HeaderXRequestID)
		requestID2 := rec2.Header().Get(echo.HeaderXRequestID)

		assert.NotEmpty(t, requestID1)
		assert.NotEmpty(t, requestID2)
		assert.NotEqual(t, requestID1, requestID2, "Different requests should have different IDs")
	})
}

func TestGetRequestID(t *testing.T) {
	e := echo.New()
	e.Use(middleware.RequestID())

	t.Run("get request ID from response header", func(t *testing.T) {
		e := echo.New()
		e.Use(middleware.RequestID())
		e.Use(RequestLogger())

		var capturedRequestID string
		e.GET("/test", func(c echo.Context) error {
			capturedRequestID = getRequestID(c)
			return c.String(http.StatusOK, capturedRequestID)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		// Request ID should be in response header
		requestID := rec.Header().Get(echo.HeaderXRequestID)
		assert.NotEmpty(t, requestID)
		assert.Equal(t, requestID, capturedRequestID)
	})

	t.Run("get request ID from context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Process request to set RequestID
		handler := func(c echo.Context) error {
			// Simulate RequestID middleware setting context
			requestID := "test-request-id-123"
			c.Set(echo.HeaderXRequestID, requestID)
			c.Response().Header().Set(echo.HeaderXRequestID, requestID)

			retrievedID := getRequestID(c)
			assert.Equal(t, requestID, retrievedID)
			return c.String(http.StatusOK, retrievedID)
		}

		_ = handler(c)
	})

	t.Run("fallback to correlation ID header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(CorrelationIDHeader, "correlation-id-456")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		requestID := getRequestID(c)
		assert.Equal(t, "correlation-id-456", requestID)
	})

	t.Run("generate UUID if no ID available", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		requestID := getRequestID(c)
		// Should generate a UUID (36 characters)
		assert.Len(t, requestID, 36)
	})
}

func TestRequestIDInLogs(t *testing.T) {
	// This test verifies that request IDs are included in logs
	// Since we're using structured logging, we can't easily capture logs in tests
	// But we can verify the RequestLogger middleware sets the correlation ID
	e := echo.New()
	e.Use(middleware.RequestID())
	e.Use(RequestLogger())

	var capturedCorrelationID string
	e.GET("/log-test", func(c echo.Context) error {
		// Check that correlation ID is set in context
		if id, ok := c.Get(CorrelationIDKey).(string); ok {
			capturedCorrelationID = id
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/log-test", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	requestID := rec.Header().Get(echo.HeaderXRequestID)
	require.NotEmpty(t, requestID)
	assert.Equal(t, requestID, capturedCorrelationID, "Correlation ID should be set in context for logging")
}

