package handlers

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
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
		} else {
			h.logger.Info("Failed login attempt - user not found",
				zap.String("email", req.Email))
		}
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid email or password"})
	}

	// Verify password
	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		// Log the failure but return generic error
		h.logger.Info("Failed login attempt - invalid password",
			zap.String("email", req.Email),
			zap.String("user_id", user.ID.String()))
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid email or password"})
	}

	// Check if user is active
	if !user.IsActive {
		h.logger.Info("Login attempt by inactive user",
			zap.String("email", req.Email),
			zap.String("user_id", user.ID.String()))
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
	if err := h.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		h.logger.Warn("Failed to update last login timestamp",
			zap.Error(err),
			zap.String("user_id", user.ID.String()))
		// Don't fail the login if this update fails
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

	return c.JSON(http.StatusOK, LoginResponse{
		User: user.ToResponse(),
	})
}
