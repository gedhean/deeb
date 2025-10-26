package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
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

// Register handles POST /api/v1/auth/register.
// Creates a new user account with email and password.
// Validates email format, checks for duplicates, and hashes the password before storing.
//
// First User Setup:
// The first user to register is automatically granted the OrganizationAdmin role
// regardless of the role specified in the request. This allows for initial system setup
// where no users exist yet. Subsequent users receive the OrganizationDeveloper role by default
// or a custom role if specified.
//
// @Summary Register a new user
// @Description Create a new user account with email and password. The first user becomes OrganizationAdmin.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} ErrorResponse "Invalid request (bad email, password too short/long, etc.)"
// @Failure 409 {object} ErrorResponse "Email already registered"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /api/v1/auth/register [post]
func (h *RegisterHandler) Register(c echo.Context) error {
	if c.Request().Method != http.MethodPost {
		return c.JSON(http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
	}

	ctx := c.Request().Context()

	// Parse request body
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Debug("Failed to decode registration request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
	}

	// Validate email format
	if !h.isValidEmail(req.Email) {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid email format"})
	}

	// Normalize email (lowercase)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Validate password (HashPassword will do length validation)
	if req.Password == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "password is required"})
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.logger.Debug("Password validation failed", zap.Error(err))
		if errors.Is(err, auth.ErrPasswordTooShort) {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "password must be at least 8 characters"})
		}
		if errors.Is(err, auth.ErrPasswordTooLong) {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "password must not exceed 72 characters"})
		}
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid password"})
	}

	// Check for duplicate email
	existingUser, err := h.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		h.logger.Error("Failed to check for existing user", zap.Error(err), zap.String("email", req.Email))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
	if existingUser != nil {
		h.logger.Debug("Registration attempt with existing email", zap.String("email", req.Email))
		return c.JSON(http.StatusConflict, ErrorResponse{Error: "email already registered"})
	}

	// Check if this is the first user (initial setup)
	// First user automatically becomes OrganizationAdmin
	users, err := h.userRepo.List(ctx, 1, 0)
	if err != nil {
		h.logger.Error("Failed to check user count for first user setup", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}

	// Set organization role based on whether this is the first user
	var orgRole string
	if len(users) == 0 {
		// First user - automatically grant OrganizationAdmin role
		orgRole = "OrganizationAdmin"
		h.logger.Info("First user registration - granting OrganizationAdmin role", zap.String("email", req.Email))
	} else {
		// Subsequent users - use default or provided role
		orgRole = "OrganizationDeveloper"
		if req.OrganizationRole != nil {
			orgRole = *req.OrganizationRole
		}
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
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create user"})
	}

	h.logger.Info("User registered successfully",
		zap.String("email", user.Email),
		zap.String("user_id", user.ID.String()))

	// Return success response
	return c.JSON(http.StatusCreated, RegisterResponse{
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
