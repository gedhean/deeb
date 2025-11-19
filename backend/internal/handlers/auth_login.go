package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/middleware"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// LoginHandler handles user login requests with rate limiting and JWT token generation.
type LoginHandler struct {
	userRepo     repository.UserRepository
	tokenManager *auth.TokenManager
	rateLimiter  *RateLimiter
	logger       *zap.Logger
}

// NewLoginHandler creates a new LoginHandler with the provided dependencies.
func NewLoginHandler(
	userRepo repository.UserRepository,
	tokenManager *auth.TokenManager,
	rateLimiter *RateLimiter,
	logger *zap.Logger,
) *LoginHandler {
	return &LoginHandler{
		userRepo:     userRepo,
		tokenManager: tokenManager,
		rateLimiter:  rateLimiter,
		logger:       logger,
	}
}

// setLoginAuditEnrichment is a helper to set audit enrichment in the context
// without duplication and key typos.
func setLoginAuditEnrichment(c echo.Context, enrichment middleware.AuditEnrichment) {
	c.Set("audit_enrichment", &enrichment)
}

// Login handles POST /api/v1/auth/login.
// Authenticates a user with email and password, checks rate limits,
// generates a JWT token on success, and sets it in an HttpOnly cookie.
func (h *LoginHandler) Login(c echo.Context) error {
	if c.Request().Method != http.MethodPost {
		return c.JSON(http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
	}

	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Debug("Invalid JSON in login request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email and password are required"})
	}

	// Check rate limit BEFORE database lookup to prevent enumeration
	if !h.rateLimiter.Allow(req.Email) {
		h.logger.Warn("Rate limit exceeded for login attempt",
			zap.String("email", req.Email))
		// Enrich audit log with failure details
		setLoginAuditEnrichment(c, middleware.AuditEnrichment{
			Email:         req.Email,
			FailureReason: "rate_limit_exceeded",
			ErrorMessage:  "too many login attempts",
			Result:        "failure",
		})
		return c.JSON(http.StatusTooManyRequests, ErrorResponse{Error: "too many login attempts, please try again later"})
	}

	ctx := c.Request().Context()

	// Look up user by email
	user, err := h.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Log the failure but return generic error to prevent enumeration
		if !errors.Is(err, sql.ErrNoRows) {
			h.logger.Error("Database error during login",
				zap.String("email", req.Email),
				zap.Error(err))
			// Enrich audit log with failure details - use generic message
			// to avoid leaking DB details into audit trail
			setLoginAuditEnrichment(c, middleware.AuditEnrichment{
				Email:         req.Email,
				FailureReason: "database_error",
				ErrorMessage:  "internal database error",
				Result:        "failure",
			})
		} else {
			h.logger.Info("Failed login attempt - user not found",
				zap.String("email", req.Email))
			// Enrich audit log with failure details
			setLoginAuditEnrichment(c, middleware.AuditEnrichment{
				Email:         req.Email,
				FailureReason: "user_not_found",
				ErrorMessage:  "invalid email or password",
				Result:        "failure",
			})
		}
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid email or password"})
	}

	// Verify password
	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		// Log the failure but return generic error
		h.logger.Info("Failed login attempt - invalid password",
			zap.String("email", req.Email),
			zap.String("user_id", user.ID.String()))
		// Enrich audit log with failure details
		setLoginAuditEnrichment(c, middleware.AuditEnrichment{
			Email:         req.Email,
			FailureReason: "invalid_password",
			ErrorMessage:  "invalid email or password",
			Result:        "failure",
		})
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid email or password"})
	}

	// Check if user is active
	if !user.IsActive {
		h.logger.Info("Login attempt by inactive user",
			zap.String("email", req.Email),
			zap.String("user_id", user.ID.String()))
		// Enrich audit log with failure details
		setLoginAuditEnrichment(c, middleware.AuditEnrichment{
			Email:         req.Email,
			FailureReason: "inactive_account",
			ErrorMessage:  "account is inactive",
			Result:        "failure",
		})
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "account is inactive"})
	}

	// Generate JWT token
	token, err := h.tokenManager.GenerateToken(user.ID, user.Email, string(user.OrganizationRole))
	if err != nil {
		h.logger.Error("Failed to generate token",
			zap.Error(err),
			zap.String("user_id", user.ID.String()))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}

	// Update last login timestamp
	now := time.Now()
	if err := h.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		h.logger.Warn("Failed to update last login timestamp",
			zap.Error(err),
			zap.String("user_id", user.ID.String()))
		// Don't fail the login if this update fails
	} else {
		// Update the in-memory user object with the new last login timestamp
		// so the response includes the updated value
		user.LastLogin = &now
		user.UpdatedAt = now
	}

	// Reset rate limit on successful login
	h.rateLimiter.Reset(req.Email)

	h.logger.Info("User logged in successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email))

	// Set JWT token in HttpOnly cookie
	cookie := &http.Cookie{
		Name:     "deeb_auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().URL.Scheme == "https", // HTTPS only in production
		SameSite: http.SameSiteStrictMode,           // Strict for production
		MaxAge:   86400,                             // 24 hours (matches token expiration)
	}
	c.SetCookie(cookie)

	// Enrich audit log with success details
	setLoginAuditEnrichment(c, middleware.AuditEnrichment{
		UserID: &user.ID, // Include user ID for audit log
		Email:  req.Email,
		Result: "success",
	})

	return c.JSON(http.StatusOK, LoginResponse{
		User: user.ToResponse(),
	})
}
