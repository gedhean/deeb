package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nsxbet/deeb/internal/config"
)

// Setup configures all Echo middleware
func Setup(e *echo.Echo, cfg *config.Config) {
	// Recovery middleware - recovers from panics
	e.Use(middleware.Recover())

	// Logger middleware - logs HTTP requests
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339}","method":"${method}","uri":"${uri}",` +
			`"status":${status},"latency":"${latency_human}","bytes_in":${bytes_in},` +
			`"bytes_out":${bytes_out},"error":"${error}"}` + "\n",
	}))

	// Request ID middleware - adds correlation ID to each request
	e.Use(middleware.RequestID())

	// CORS middleware - configure cross-origin requests
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.Server.CORSOrigins,
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.PATCH, echo.OPTIONS},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
		MaxAge:           300, // 5 minutes
	}))

	// Rate limiting middleware - prevent abuse
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	// Gzip compression middleware
	e.Use(middleware.Gzip())

	// Secure middleware - adds security headers
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		ContentSecurityPolicy: "default-src 'self'",
	}))
}
