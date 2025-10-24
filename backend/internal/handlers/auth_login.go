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

type LoginHandler struct {
	userRepo     repository.UserRepository
	tokenManager *auth.TokenManager
	rateLimiter  *RateLimiter
	logger       *zap.Logger
}

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

// Login handles POST /api/v1/auth/login
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
	token, err := h.tokenManager.GenerateToken(user.ID, user.Email, user.OrganizationRole)
	if err != nil {
		h.logger.Error("Failed to generate token",
			zap.Error(err),
			zap.String("user_id", user.ID.String()))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}

	// Reset rate limit on successful login
	h.rateLimiter.Reset(req.Email)

	h.logger.Info("User logged in successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email))

	return c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User:  user.ToResponse(),
	})
}
