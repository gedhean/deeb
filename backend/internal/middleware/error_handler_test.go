package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Initialize logger for tests
	_ = logger.Initialize(logger.Config{
		Level:  "info",
		Format: "json",
	})
}

func TestCustomErrorHandler_EchoHTTPError(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedCode  int
		expectedError string
		checkMessage  bool
		expectedMsg   string
	}{
		{
			name:          "404 Not Found",
			err:           echo.NewHTTPError(http.StatusNotFound, "Resource not found"),
			expectedCode:  http.StatusNotFound,
			expectedError: "Not Found",
			checkMessage:  true,
			expectedMsg:   "Resource not found",
		},
		{
			name:          "400 Bad Request",
			err:           echo.NewHTTPError(http.StatusBadRequest, "Invalid input"),
			expectedCode:  http.StatusBadRequest,
			expectedError: "Bad Request",
			checkMessage:  true,
			expectedMsg:   "Invalid input",
		},
		{
			name:          "401 Unauthorized",
			err:           echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized access"),
			expectedCode:  http.StatusUnauthorized,
			expectedError: "Unauthorized",
			checkMessage:  true,
			expectedMsg:   "Unauthorized access",
		},
		{
			name:          "403 Forbidden",
			err:           echo.NewHTTPError(http.StatusForbidden, "Access denied"),
			expectedCode:  http.StatusForbidden,
			expectedError: "Forbidden",
			checkMessage:  true,
			expectedMsg:   "Access denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute
			CustomErrorHandler(tt.err, c)

			// Assert
			assert.Equal(t, tt.expectedCode, rec.Code)
			assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

			// Parse response
			var response ErrorResponse
			err := json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedError, response.Error)
			if tt.checkMessage {
				assert.Equal(t, tt.expectedMsg, response.Message)
			}
		})
	}
}

func TestCustomErrorHandler_ServerError(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create 500 error
	err := echo.NewHTTPError(http.StatusInternalServerError, "Database connection failed")

	// Execute
	CustomErrorHandler(err, c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response ErrorResponse
	jsonErr := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, jsonErr)

	// Server errors should not expose internal details
	assert.Equal(t, "Internal Server Error", response.Error)
	// Message should be sanitized for 500 errors
	assert.NotEmpty(t, response.Message)
}

func TestCustomErrorHandler_GenericError(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create generic error (not echo.HTTPError)
	err := errors.New("unexpected error")

	// Execute
	CustomErrorHandler(err, c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response ErrorResponse
	jsonErr := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, jsonErr)

	assert.Equal(t, "Internal Server Error", response.Error)
	// Generic errors treated as 500, message should be sanitized
	assert.NotEmpty(t, response.Message)
}

func TestCustomErrorHandler_WithCorrelationID(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set correlation ID
	correlationID := "test-correlation-123"
	c.Set(CorrelationIDKey, correlationID)

	// Create error
	err := echo.NewHTTPError(http.StatusBadRequest, "Invalid request")

	// Execute
	CustomErrorHandler(err, c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response ErrorResponse
	jsonErr := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, jsonErr)

	assert.Equal(t, "Bad Request", response.Error)
	assert.Equal(t, "Invalid request", response.Message)
	assert.Equal(t, correlationID, response.RequestID)
}

func TestCustomErrorHandler_CommittedResponse(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Commit response first
	c.Response().WriteHeader(http.StatusOK)
	c.Response().Committed = true

	// Try to handle error
	err := echo.NewHTTPError(http.StatusBadRequest, "This should not be sent")

	// Execute
	CustomErrorHandler(err, c)

	// Assert - response should not be modified
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestCustomErrorHandler_HTTPStatusMapping(t *testing.T) {
	statusCodes := []int{
		http.StatusBadRequest,          // 400
		http.StatusUnauthorized,        // 401
		http.StatusForbidden,           // 403
		http.StatusNotFound,            // 404
		http.StatusMethodNotAllowed,    // 405
		http.StatusConflict,            // 409
		http.StatusUnprocessableEntity, // 422
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusNotImplemented,      // 501
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
	}

	for _, code := range statusCodes {
		t.Run(http.StatusText(code), func(t *testing.T) {
			// Setup
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Create error with specific status code
			err := echo.NewHTTPError(code, "Test error")

			// Execute
			CustomErrorHandler(err, c)

			// Assert
			assert.Equal(t, code, rec.Code)

			var response ErrorResponse
			jsonErr := json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, jsonErr)

			assert.Equal(t, http.StatusText(code), response.Error)
		})
	}
}

func TestCustomErrorHandler_NoSensitiveInfoExposed(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Create error with sensitive information
	sensitiveErr := errors.New("database password is 'secret123', connection string: postgres://user:pass@host/db")
	err := echo.NewHTTPError(http.StatusInternalServerError).SetInternal(sensitiveErr)

	// Execute
	CustomErrorHandler(err, c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response ErrorResponse
	jsonErr := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, jsonErr)

	// Sensitive information should not be in the response
	assert.NotContains(t, response.Message, "secret123")
	assert.NotContains(t, response.Message, "password")
	assert.NotContains(t, response.Message, "user:pass")
}

func TestCustomErrorHandler_WithUserContext(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set correlation ID
	correlationID := "test-correlation-456"
	c.Set(CorrelationIDKey, correlationID)

	// Set user context
	userID := uuid.New()
	userEmail := "test@example.com"
	claims := &auth.Claims{
		UserID:           userID,
		Email:            userEmail,
		OrganizationRole: "OrganizationDeveloper",
	}
	c.Set("user", claims)

	// Create error
	err := echo.NewHTTPError(http.StatusBadRequest, "Invalid request")

	// Execute
	CustomErrorHandler(err, c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response ErrorResponse
	jsonErr := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, jsonErr)

	assert.Equal(t, "Bad Request", response.Error)
	assert.Equal(t, "Invalid request", response.Message)
	assert.Equal(t, correlationID, response.RequestID)
}

func TestCustomErrorHandler_RequestIDInResponse(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set correlation ID
	correlationID := "test-request-id-789"
	c.Set(CorrelationIDKey, correlationID)

	// Create error
	err := echo.NewHTTPError(http.StatusNotFound, "Resource not found")

	// Execute
	CustomErrorHandler(err, c)

	// Assert
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var response ErrorResponse
	jsonErr := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, jsonErr)

	assert.Equal(t, "Not Found", response.Error)
	assert.Equal(t, "Resource not found", response.Message)
	assert.Equal(t, correlationID, response.RequestID)
}

func TestCustomErrorHandler_NoRequestIDWhenMissing(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Don't set correlation ID

	// Create error
	err := echo.NewHTTPError(http.StatusBadRequest, "Invalid request")

	// Execute
	CustomErrorHandler(err, c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response ErrorResponse
	jsonErr := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, jsonErr)

	assert.Equal(t, "Bad Request", response.Error)
	assert.Equal(t, "Invalid request", response.Message)
	assert.Empty(t, response.RequestID)
}
