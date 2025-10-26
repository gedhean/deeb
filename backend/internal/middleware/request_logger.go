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

// RequestLogger creates a middleware that logs HTTP requests with Zap logger.
// Generates or extracts correlation IDs for request tracing and logs request/response details
// including method, URI, status, latency, and bytes transferred.
func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Generate correlation ID
			correlationID := generateCorrelationID(c)

			// Store in context for handler access
			c.Set(CorrelationIDKey, correlationID)

			// Add to response header
			c.Response().Header().Set(CorrelationIDHeader, correlationID)

			// Capture start time
			start := time.Now()

			// Get request details
			req := c.Request()
			method := req.Method
			uri := req.RequestURI
			if uri == "" {
				uri = "/"
			}

			// Process request
			err := next(c)

			// Calculate duration and get response details
			duration := time.Since(start)
			status := c.Response().Status
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

			return err
		}
	}
}

// generateCorrelationID generates or extracts a correlation ID for the request
func generateCorrelationID(c echo.Context) string {
	// Check if correlation ID already exists in request header
	correlationID := c.Request().Header.Get(CorrelationIDHeader)
	if correlationID != "" {
		return correlationID
	}

	// Check if Echo's RequestID middleware already set an ID
	if requestID := c.Response().Header().Get(echo.HeaderXRequestID); requestID != "" {
		return requestID
	}

	// Generate new UUID
	return uuid.New().String()
}

// formatDuration formats duration in a human-readable way (milliseconds)
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
}
