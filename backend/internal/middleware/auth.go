// Package middleware provides HTTP middleware for the Deeb API including
// authentication, authorization, request logging, error handling, and security headers.
package middleware

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/logger"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// lastUsedUpdateTimeout bounds the detached best-effort last_used_at write so a
// stalled DB update cannot accumulate goroutines indefinitely.
const lastUsedUpdateTimeout = 5 * time.Second

// AuthMiddleware creates an HTTP middleware that validates JWT tokens or API tokens.
// It extracts tokens from HttpOnly cookies (JWT only) or Authorization headers (JWT or API token).
type AuthMiddleware struct {
	tokenManager *auth.TokenManager
	apiTokenRepo repository.APITokenRepository
	userRepo     repository.UserRepository
}

// NewAuthMiddleware creates a new authentication middleware.
// The apiTokenRepo and userRepo enable long-lived API-token (service account) authentication
// on the Authorization header, alongside the existing JWT flow.
func NewAuthMiddleware(
	tokenManager *auth.TokenManager,
	apiTokenRepo repository.APITokenRepository,
	userRepo repository.UserRepository,
) *AuthMiddleware {
	return &AuthMiddleware{
		tokenManager: tokenManager,
		apiTokenRepo: apiTokenRepo,
		userRepo:     userRepo,
	}
}

// Authenticate is the Echo middleware handler that validates authentication.
// It first tries to extract a JWT from the HttpOnly cookie, then falls back to the
// Authorization header, which may carry either an API token (prefixed with
// auth.APITokenPrefix) or a JWT. On success it populates the Echo context with the
// user claims; on failure it returns 401 Unauthorized.
func (m *AuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Try to extract token from HttpOnly cookie first. The cookie only ever carries a JWT.
		var tokenString string
		fromCookie := false
		cookie, err := c.Cookie("deeb_auth_token")
		if err == nil && cookie.Value != "" {
			tokenString = cookie.Value
			fromCookie = true
		} else {
			// Fallback to Authorization header for API tokens and bearer JWTs.
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

		// API tokens arrive only on the Authorization header (never the cookie) and are
		// recognized by their prefix.
		if !fromCookie && auth.IsAPIToken(tokenString) {
			return m.authenticateAPIToken(c, tokenString, next)
		}

		// Validate JWT
		claims, err := m.tokenManager.ValidateToken(tokenString)
		if err != nil {
			// Return appropriate error based on validation failure
			if err == auth.ErrExpiredToken {
				return echo.NewHTTPError(http.StatusUnauthorized, "token expired")
			}
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		setAuthContext(c, claims)

		// Call next handler
		return next(c)
	}
}

// authenticateAPIToken validates a long-lived API token, loads its associated user,
// and populates the context identically to the JWT path so all downstream RBAC and
// handlers behave the same.
func (m *AuthMiddleware) authenticateAPIToken(c echo.Context, tokenString string, next echo.HandlerFunc) error {
	ctx := c.Request().Context()

	hash := auth.HashAPIToken(tokenString)
	token, err := m.apiTokenRepo.GetByTokenHash(ctx, hash)
	if err != nil {
		if isNotFound(err) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}
		logger.Error("failed to look up api token", zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "authentication error")
	}
	if token == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}

	if !token.IsActive() {
		return echo.NewHTTPError(http.StatusUnauthorized, "token expired or revoked")
	}

	user, err := m.userRepo.GetByID(ctx, token.UserID)
	if err != nil {
		if isNotFound(err) {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}
		logger.Error("failed to look up api token user", zap.Error(err),
			zap.String("user_id", token.UserID.String()))
		return echo.NewHTTPError(http.StatusInternalServerError, "authentication error")
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
	}

	if !user.IsActive {
		return echo.NewHTTPError(http.StatusUnauthorized, "user is inactive")
	}

	// Synthesize the same claims the JWT path would produce so setAuthContext works verbatim.
	claims := &auth.Claims{
		UserID:           user.ID,
		Email:            user.Email,
		OrganizationRole: string(user.OrganizationRole),
	}
	setAuthContext(c, claims)

	// Best-effort last-used update on a detached, bounded context (the request context
	// may cancel; the timeout prevents stalled DB updates from accumulating).
	tokenID := token.ID
	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), lastUsedUpdateTimeout)
		defer cancel()
		if updateErr := m.apiTokenRepo.UpdateLastUsed(updateCtx, tokenID); updateErr != nil {
			logger.Warn("failed to update api token last_used_at", zap.Error(updateErr))
		}
	}()

	return next(c)
}

// isNotFound reports whether a repository error means "no such record" (as opposed to
// an infrastructure failure), across the sentinels the repos use.
func isNotFound(err error) bool {
	return errors.Is(err, repository.ErrNotFound) || errors.Is(err, sql.ErrNoRows)
}

// setAuthContext stores the user claims on the Echo context and populates the RBAC
// context values used by the authorization middleware. Both the JWT and API-token
// paths call this so downstream behavior is identical.
func setAuthContext(c echo.Context, claims *auth.Claims) {
	// Store claims in Echo context
	c.Set("user", claims)

	// Map organization role from JWT format to RBAC format
	// JWT uses: "OrganizationAdmin", "OrganizationDBA", "OrganizationDeveloper"
	var rbacRole rbac.OrganizationRole
	switch claims.OrganizationRole {
	case "OrganizationAdmin":
		rbacRole = rbac.OrganizationAdmin
	case "OrganizationDBA":
		rbacRole = rbac.OrganizationDBA
	case "OrganizationDeveloper":
		rbacRole = rbac.OrganizationDeveloper
	default:
		// Fallback to developer if unknown role
		rbacRole = rbac.OrganizationDeveloper
	}

	// Also set RBAC context values for authorization middleware, and the user claims
	// on the request context so GetUserFromContext (request-context helper) works too.
	ctx := c.Request().Context()
	ctx = context.WithValue(ctx, auth.UserContextKey, claims)
	ctx = context.WithValue(ctx, rbac.ContextKeyUserID, claims.UserID.String())
	ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationID, "default") // Single-tenant for MVP
	ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationRole, rbacRole)
	c.SetRequest(c.Request().WithContext(ctx))
}

// GetUserFromContext extracts user claims from request context.
// Returns the claims and a boolean indicating if the extraction was successful.
func GetUserFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(auth.UserContextKey).(*auth.Claims)
	return claims, ok
}
