package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/logger"
	"go.uber.org/zap"
)

// ErrorResponse represents a standardized JSON error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// CustomErrorHandler creates a custom error handler for Echo.
// It formats all errors as JSON and logs appropriately based on severity.
// Server errors (5xx) are logged with stack traces, client errors (4xx) at info level.
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

	// Get correlation ID if available
	correlationID := ""
	if id, ok := c.Get(CorrelationIDKey).(string); ok {
		correlationID = id
	}

	// Log based on error severity
	if code >= 500 {
		// Server errors: log with stack trace
		logger.Error("Server error",
			zap.String("correlation_id", correlationID),
			zap.String("method", c.Request().Method),
			zap.String("path", c.Request().URL.Path),
			zap.Int("status", code),
			zap.Error(err),
			zap.String("stack", string(debug.Stack())),
		)
		// Don't expose internal error details to client
		errorDetail = message
	} else if code >= 400 {
		// Client errors: log at info level
		logger.Info("Client error",
			zap.String("correlation_id", correlationID),
			zap.String("method", c.Request().Method),
			zap.String("path", c.Request().URL.Path),
			zap.Int("status", code),
			zap.String("error", errorDetail),
		)
	}

	// Send JSON error response
	if !c.Response().Committed {
		// Create error response
		response := ErrorResponse{
			Error:   http.StatusText(code),
			Message: errorDetail,
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
