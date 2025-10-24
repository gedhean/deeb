package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

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

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("Invalid JSON in login request", zap.Error(err))
		h.sendError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		h.sendError(w, "email and password are required", http.StatusBadRequest)
		return
	}

	// Check rate limit BEFORE database lookup to prevent enumeration
	if !h.rateLimiter.Allow(req.Email) {
		h.logger.Warn("Rate limit exceeded for login attempt",
			zap.String("email", req.Email))
		h.sendError(w, "too many login attempts, please try again later", http.StatusTooManyRequests)
		return
	}

	ctx := r.Context()

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
		h.sendError(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	// Verify password
	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		// Log the failure but return generic error
		h.logger.Info("Failed login attempt - invalid password",
			zap.String("email", req.Email),
			zap.String("user_id", user.ID.String()))
		h.sendError(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	// Check if user is active
	if !user.IsActive {
		h.logger.Info("Login attempt by inactive user",
			zap.String("email", req.Email),
			zap.String("user_id", user.ID.String()))
		h.sendError(w, "account is inactive", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := h.tokenManager.GenerateToken(user.ID, user.Email, user.OrganizationRole)
	if err != nil {
		h.logger.Error("Failed to generate token",
			zap.Error(err),
			zap.String("user_id", user.ID.String()))
		h.sendError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Reset rate limit on successful login
	h.rateLimiter.Reset(req.Email)

	h.logger.Info("User logged in successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email))

	response := LoginResponse{
		Token: token,
		User:  user.ToResponse(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *LoginHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
