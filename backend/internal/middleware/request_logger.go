package middleware

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/logger"
)

const (
	// CorrelationIDHeader is the header name for correlation ID
	CorrelationIDHeader = "X-Correlation-ID"
	// CorrelationIDKey is the context key for storing correlation ID
	CorrelationIDKey = "correlation_id"
)

// RequestLoggerConfig holds configuration for request logger middleware
type RequestLoggerConfig struct {
	// SkipPaths is a list of paths to skip logging (e.g., health check endpoints)
	SkipPaths []string
}

// RequestLogger creates a middleware that logs HTTP requests with Zap logger.
// Uses Echo's RequestID middleware for request tracking and logs request/response details
// including method, URI, status, latency, and bytes transferred.
func RequestLogger() echo.MiddlewareFunc {
	return RequestLoggerWithConfig(RequestLoggerConfig{})
}

// RequestLoggerWithConfig creates a middleware that logs HTTP requests with Zap logger,
// with configurable skip paths.
func RequestLoggerWithConfig(config RequestLoggerConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get request ID from Echo's RequestID middleware
			requestID := getRequestID(c)

			// Store in context for handler access (for backwards compatibility)
			c.Set(CorrelationIDKey, requestID)

			// Also set X-Correlation-ID header for backwards compatibility
			c.Response().Header().Set(CorrelationIDHeader, requestID)

			// Capture start time
			start := time.Now()

			// Get request details
			req := c.Request()
			method := req.Method
			uri := req.RequestURI
			if uri == "" {
				uri = "/"
			}

			// Check if this path should be skipped
			shouldSkip := false
			for _, skipPath := range config.SkipPaths {
				if uri == skipPath {
					shouldSkip = true
					break
				}
			}

			// Process request
			err := next(c)

			// Calculate duration and get response details
			duration := time.Since(start)
			status := c.Response().Status

			// Skip logging if configured to do so
			if !shouldSkip {
				bytesIn := req.ContentLength
				if bytesIn < 0 {
					bytesIn = 0
				}
				bytesOut := c.Response().Size

				// Format error message
				errorMsg := ""
				if err != nil {
					errorMsg = err.Error()
				}

				// Build log entry matching the requested format
				// Using a struct to maintain field order in JSON output
				logEntry := struct {
					Time     string `json:"time"`
					Method   string `json:"method"`
					URI      string `json:"uri"`
					Status   int    `json:"status"`
					Latency  string `json:"latency"`
					BytesIn  int64  `json:"bytes_in"`
					BytesOut int64  `json:"bytes_out"`
					Error    string `json:"error"`
				}{
					Time:     start.Add(duration).Format(time.RFC3339),
					Method:   method,
					URI:      uri,
					Status:   status,
					Latency:  formatDuration(duration),
					BytesIn:  bytesIn,
					BytesOut: bytesOut,
					Error:    errorMsg,
				}

				// Convert to JSON
				logJSON, _ := json.Marshal(logEntry)

				// Determine log level based on status code
				logFunc := logger.Info
				if status >= 500 {
					logFunc = logger.Error
				} else if status >= 400 {
					logFunc = logger.Warn
				}

				// Log with the JSON string
				logFunc(string(logJSON))
			}

			return err
		}
	}
}

// getRequestID gets the request ID from Echo's RequestID middleware
// Falls back to correlation ID header or generates a new UUID if not available
func getRequestID(c echo.Context) string {
	// First, try to get from Echo's RequestID middleware
	if requestID := c.Response().Header().Get(echo.HeaderXRequestID); requestID != "" {
		return requestID
	}

	// Fallback: try to get from context (set by RequestID middleware)
	if requestID, ok := c.Get(echo.HeaderXRequestID).(string); ok && requestID != "" {
		return requestID
	}

	// Fallback: check if correlation ID already exists in request header
	if correlationID := c.Request().Header.Get(CorrelationIDHeader); correlationID != "" {
		return correlationID
	}

	// Last resort: generate new UUID (should not happen if RequestID middleware is properly configured)
	return uuid.New().String()
}

// formatDuration formats duration in a human-readable way (milliseconds)
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
}
