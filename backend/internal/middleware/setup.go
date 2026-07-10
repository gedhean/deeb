package middleware

import (
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nsxbet/deeb/internal/config"
	"golang.org/x/time/rate"
)

// Setup configures all Echo middleware in the correct order.
// Includes recovery, logging, CORS, rate limiting, compression, and security headers.
// Note: Authentication and audit logging are applied at the route group level, not globally.
func Setup(e *echo.Echo, cfg *config.Config) {
	// Set custom error handler for consistent JSON error responses
	e.HTTPErrorHandler = CustomErrorHandler

	// Recovery middleware - recovers from panics
	e.Use(middleware.Recover())

	// Request ID middleware - generates unique ID for each request
	// This must be before RequestLogger so the ID is available for logging
	e.Use(middleware.RequestID())

	// Custom request logger with Zap integration and correlation IDs
	// Skip logging for health check endpoints to reduce noise
	e.Use(RequestLoggerWithConfig(RequestLoggerConfig{
		SkipPaths: []string{"/api/v1/health"},
	}))

	// CORS middleware - configure cross-origin requests
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: cfg.Server.CORSOrigins,
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.PATCH, echo.OPTIONS},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			echo.HeaderXRequestID,
		},
		AllowCredentials: true,
		MaxAge:           300, // 5 minutes
	}))

	// Rate limiting middleware - prevent abuse (configurable; skipped when disabled).
	// Modeled as "RateLimitRequests per RateLimitWindow" per client IP: the window's
	// worth of requests is the burst, and tokens refill at requests/window per second.
	// This keeps a generous burst so a single SPA page load (which fans out many
	// parallel requests, plus SWR revalidation and React StrictMode double-fires in
	// dev) does not trip a 429, while still capping sustained abuse.
	if cfg.Security.RateLimitEnabled {
		window := cfg.Security.RateLimitWindow
		if window <= 0 {
			window = time.Minute
		}
		requests := cfg.Security.RateLimitRequests
		if requests <= 0 {
			requests = 1000
		}
		e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
			Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(float64(requests) / window.Seconds()),
				Burst:     requests,
				ExpiresIn: 3 * time.Minute,
			}),
		}))
	}

	// Gzip compression middleware
	e.Use(middleware.Gzip())

	// Secure middleware with conditional CSP for Swagger UI
	e.Use(ConditionalSecureMiddleware())
}

// ConditionalSecureMiddleware applies different CSP policies based on the route.
// Uses relaxed CSP for Swagger UI endpoints and strict CSP for all other endpoints.
func ConditionalSecureMiddleware() echo.MiddlewareFunc {
	// Relaxed CSP for Swagger UI endpoints
	swaggerCSP := "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data:; connect-src 'self' https://cdn.jsdelivr.net"

	// Strict CSP for all other endpoints
	strictCSP := "default-src 'self'"

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Apply security headers
			c.Response().Header().Set("X-XSS-Protection", "1; mode=block")
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("Strict-Transport-Security", "max-age=31536000")

			// Apply CSP based on path
			path := c.Request().URL.Path
			if strings.HasPrefix(path, "/api/docs") {
				c.Response().Header().Set("Content-Security-Policy", swaggerCSP)
			} else {
				c.Response().Header().Set("Content-Security-Policy", strictCSP)
			}

			return next(c)
		}
	}
}
