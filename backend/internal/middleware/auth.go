package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/nsxbet/deeb/internal/auth"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserContextKey is the key used to store user claims in request context
	UserContextKey ContextKey = "user"
)

// AuthMiddleware creates an HTTP middleware that validates JWT tokens
type AuthMiddleware struct {
	tokenManager *auth.TokenManager
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(tokenManager *auth.TokenManager) *AuthMiddleware {
	return &AuthMiddleware{
		tokenManager: tokenManager,
	}
}

// Authenticate is the middleware handler that validates JWT tokens
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Check Bearer scheme
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		if tokenString == "" {
			http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := m.tokenManager.ValidateToken(tokenString)
		if err != nil {
			// Return appropriate error based on validation failure
			if err == auth.ErrExpiredToken {
				http.Error(w, `{"error":"token expired"}`, http.StatusUnauthorized)
				return
			}
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Store claims in request context
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		r = r.WithContext(ctx)

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext extracts user claims from request context
func GetUserFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*auth.Claims)
	return claims, ok
}
