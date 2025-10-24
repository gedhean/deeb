package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"

	"go.uber.org/zap"
)

// Email validation regex - basic RFC 5322 validation
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// RegisterHandler handles user registration requests
type RegisterHandler struct {
	userRepo repository.UserRepository
	logger   *zap.Logger
}

// NewRegisterHandler creates a new RegisterHandler
func NewRegisterHandler(userRepo repository.UserRepository, logger *zap.Logger) *RegisterHandler {
	return &RegisterHandler{
		userRepo: userRepo,
		logger:   logger,
	}
}

// ServeHTTP handles POST /api/v1/auth/register
func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Only accept POST requests
	if r.Method != http.MethodPost {
		h.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse request body
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Debug("Failed to decode registration request", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate email format
	if !h.isValidEmail(req.Email) {
		h.respondError(w, http.StatusBadRequest, "invalid email format")
		return
	}

	// Normalize email (lowercase)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Validate password (HashPassword will do length validation)
	if req.Password == "" {
		h.respondError(w, http.StatusBadRequest, "password is required")
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.logger.Debug("Password validation failed", zap.Error(err))
		if errors.Is(err, auth.ErrPasswordTooShort) {
			h.respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		if errors.Is(err, auth.ErrPasswordTooLong) {
			h.respondError(w, http.StatusBadRequest, "password must not exceed 72 characters")
			return
		}
		h.respondError(w, http.StatusBadRequest, "invalid password")
		return
	}

	// Check for duplicate email
	existingUser, err := h.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		h.logger.Error("Failed to check for existing user", zap.Error(err), zap.String("email", req.Email))
		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if existingUser != nil {
		h.logger.Debug("Registration attempt with existing email", zap.String("email", req.Email))
		h.respondError(w, http.StatusConflict, "email already registered")
		return
	}

	// Set default organization role if not provided
	orgRole := "OrganizationDeveloper"
	if req.OrganizationRole != nil {
		orgRole = *req.OrganizationRole
	}

	// Create user
	user := &models.User{
		Email:            req.Email,
		PasswordHash:     passwordHash,
		OrganizationRole: orgRole,
		TwoFactorEnabled: false,
		IsActive:         true,
	}

	err = h.userRepo.Create(ctx, user)
	if err != nil {
		h.logger.Error("Failed to create user", zap.Error(err), zap.String("email", req.Email))
		h.respondError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	h.logger.Info("User registered successfully",
		zap.String("email", user.Email),
		zap.String("user_id", user.ID.String()))

	// Return success response
	h.respondJSON(w, http.StatusCreated, RegisterResponse{
		User: user.ToResponse(),
	})
}

// isValidEmail validates email format using regex
func (h *RegisterHandler) isValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	if len(email) == 0 || len(email) > 254 {
		return false
	}
	return emailRegex.MatchString(email)
}

// respondJSON writes a JSON response
func (h *RegisterHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

// respondError writes an error response
func (h *RegisterHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, ErrorResponse{Error: message})
}
