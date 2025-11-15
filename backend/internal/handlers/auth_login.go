package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/services"
	"go.uber.org/zap"
)

// LoginHandler handles user login requests with rate limiting and JWT token generation.
type LoginHandler struct {
	userRepo     repository.UserRepository
	tokenManager *auth.TokenManager
	rateLimiter  *RateLimiter
	auditService *services.AuditLogService
	logger       *zap.Logger
}

// NewLoginHandler creates a new LoginHandler with the provided dependencies.
func NewLoginHandler(
	userRepo repository.UserRepository,
	tokenManager *auth.TokenManager,
	rateLimiter *RateLimiter,
	auditService *services.AuditLogService,
	logger *zap.Logger,
) *LoginHandler {
	return &LoginHandler{
		userRepo:     userRepo,
		tokenManager: tokenManager,
		rateLimiter:  rateLimiter,
		auditService: auditService,
		logger:       logger,
	}
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
		// Log failed login attempt (rate limit) - no user ID available)
		h.logAuditLoginFailure(c, uuid.Nil, req.Email, "rate_limit_exceeded", "too many login attempts")
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
			// Log audit entry for database error
			h.logAuditLoginFailure(c, uuid.Nil, req.Email, "database_error", err.Error())
		} else {
			h.logger.Info("Failed login attempt - user not found",
				zap.String("email", req.Email))
			// Log audit entry for user not found
			h.logAuditLoginFailure(c, uuid.Nil, req.Email, "user_not_found", "invalid email or password")
		}
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid email or password"})
	}

	// Verify password
	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		// Log the failure but return generic error
		h.logger.Info("Failed login attempt - invalid password",
			zap.String("email", req.Email),
			zap.String("user_id", user.ID.String()))
		// Log audit entry for invalid password
		h.logAuditLoginFailure(c, user.ID, req.Email, "invalid_password", "invalid email or password")
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid email or password"})
	}

	// Check if user is active
	if !user.IsActive {
		h.logger.Info("Login attempt by inactive user",
			zap.String("email", req.Email),
			zap.String("user_id", user.ID.String()))
		// Log audit entry for inactive user
		h.logAuditLoginFailure(c, user.ID, req.Email, "inactive_account", "account is inactive")
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

	// Log successful login
	h.logAuditLoginSuccess(c, user.ID, req.Email)

	return c.JSON(http.StatusOK, LoginResponse{
		User: user.ToResponse(),
	})
}

// logAuditLoginSuccess logs a successful login attempt
func (h *LoginHandler) logAuditLoginSuccess(c echo.Context, userID uuid.UUID, email string) {
	if h.auditService == nil {
		return
	}

	ctx := c.Request().Context()
	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"email": email,
	})

	if err := h.auditService.LogWithHTTP(ctx, userID, services.ActionLogin, services.ResourceTypeUser, userID, c.Request(), detailsJSON); err != nil {
		h.logger.Error("Failed to log audit entry for successful login",
			zap.String("user_id", userID.String()),
			zap.Error(err))
	}
}

// logAuditLoginFailure logs a failed login attempt
func (h *LoginHandler) logAuditLoginFailure(c echo.Context, userID uuid.UUID, email, reason, errorMessage string) {
	if h.auditService == nil {
		return
	}

	ctx := c.Request().Context()
	detailsJSON, _ := json.Marshal(map[string]interface{}{
		"email":          email,
		"failure_reason": reason,
		"error_message":  errorMessage,
	})

	// Use system UUID if user not found (to track failed attempts)
	logUserID := userID
	if userID == uuid.Nil {
		logUserID = uuid.Nil // System user for failed attempts
	}

	if err := h.auditService.LogWithResult(ctx, logUserID, services.ActionLogin, services.ResourceTypeUser, uuid.Nil, "failure", &errorMessage, detailsJSON); err != nil {
		h.logger.Error("Failed to log audit entry for failed login",
			zap.String("email", email),
			zap.Error(err))
	}
}
