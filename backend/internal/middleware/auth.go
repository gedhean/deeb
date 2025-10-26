// Package middleware provides HTTP middleware for the Deeb API including
// authentication, authorization, request logging, error handling, and security headers.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
)

// AuthMiddleware creates an HTTP middleware that validates JWT tokens.
// It extracts tokens from HttpOnly cookies or Authorization headers.
type AuthMiddleware struct {
	tokenManager *auth.TokenManager
}

// NewAuthMiddleware creates a new authentication middleware with the provided token manager.
func NewAuthMiddleware(tokenManager *auth.TokenManager) *AuthMiddleware {
	return &AuthMiddleware{
		tokenManager: tokenManager,
	}
}

// Authenticate is the Echo middleware handler that validates JWT tokens.
// It first tries to extract the token from the HttpOnly cookie, then falls back to the Authorization header.
// On successful validation, it stores the user claims in the Echo context with key "user".
func (m *AuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Try to extract token from HttpOnly cookie first
		var tokenString string
		cookie, err := c.Cookie("deeb_auth_token")
		if err == nil && cookie.Value != "" {
			tokenString = cookie.Value
		} else {
			// Fallback to Authorization header for backwards compatibility
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authentication")
			}

			// Check Bearer scheme
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
			}

			tokenString = parts[1]
		}

		if tokenString == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
		}

		// Validate token
		claims, err := m.tokenManager.ValidateToken(tokenString)
		if err != nil {
			// Return appropriate error based on validation failure
			if err == auth.ErrExpiredToken {
				return echo.NewHTTPError(http.StatusUnauthorized, "token expired")
			}
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		// Store claims in Echo context
		c.Set("user", claims)

		// Call next handler
		return next(c)
	}
}

// GetUserFromContext extracts user claims from request context.
// Returns the claims and a boolean indicating if the extraction was successful.
func GetUserFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(auth.UserContextKey).(*auth.Claims)
	return claims, ok
}
