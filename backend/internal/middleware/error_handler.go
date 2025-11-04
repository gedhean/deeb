package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/logger"
	"go.uber.org/zap"
)

// ErrorResponse represents a standardized JSON error response.
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message,omitempty"`
	Code      string    `json:"code,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
}

// CustomErrorHandler creates a custom error handler for Echo.
// It formats all errors as JSON and logs appropriately based on severity.
// Server errors (5xx) are logged with stack traces, client errors (4xx) at info level.
// Captures error context including request ID, user, path, and method.
func CustomErrorHandler(err error, c echo.Context) {
	// Don't process if response already started
	if c.Response().Committed {
		return
	}

	// Default to 500 Internal Server Error
	code := http.StatusInternalServerError
	message := "Internal server error"
	errorDetail := err.Error()

	// Check if it's an Echo HTTPError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		// Use the error message if it's a string
		if msg, ok := he.Message.(string); ok {
			message = msg
		} else {
			message = fmt.Sprintf("%v", he.Message)
		}
		// For client errors (4xx), include the error detail
		if code >= 400 && code < 500 {
			errorDetail = message
		}
	}

	// Get correlation ID (request ID) if available
	correlationID := ""
	if id, ok := c.Get(CorrelationIDKey).(string); ok {
		correlationID = id
	}

	// Get user context if available
	var userID uuid.UUID
	var userEmail string
	claims, ok := auth.GetUserFromEchoContext(c)
	if ok && claims != nil {
		userID = claims.UserID
		userEmail = claims.Email
	}

	// Log based on error severity
	if code >= 500 {
		// Server errors: log with stack trace
		logFields := []zap.Field{
			zap.String("correlation_id", correlationID),
			zap.String("method", c.Request().Method),
			zap.String("path", c.Request().URL.Path),
			zap.Int("status", code),
			zap.Error(err),
			zap.String("stack", string(debug.Stack())),
		}
		if userID != uuid.Nil {
			logFields = append(logFields, zap.String("user_id", userID.String()))
		}
		if userEmail != "" {
			logFields = append(logFields, zap.String("user_email", userEmail))
		}
		logger.Error("Server error", logFields...)
		// Don't expose internal error details to client
		errorDetail = message
	} else if code >= 400 {
		// Client errors: log at info level
		logFields := []zap.Field{
			zap.String("correlation_id", correlationID),
			zap.String("method", c.Request().Method),
			zap.String("path", c.Request().URL.Path),
			zap.Int("status", code),
			zap.String("error", errorDetail),
		}
		if userID != uuid.Nil {
			logFields = append(logFields, zap.String("user_id", userID.String()))
		}
		if userEmail != "" {
			logFields = append(logFields, zap.String("user_email", userEmail))
		}
		logger.Info("Client error", logFields...)
	}

	// Send JSON error response
	if !c.Response().Committed {
		// Create error response
		response := ErrorResponse{
			Error:     http.StatusText(code),
			Message:   errorDetail,
			RequestID: correlationID,
		}

		// Send response
		if err := c.JSON(code, response); err != nil {
			logger.Error("Failed to send error response",
				zap.String("correlation_id", correlationID),
				zap.Error(err),
			)
		}
	}
}
