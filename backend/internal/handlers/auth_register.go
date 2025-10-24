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

// Register handles POST /api/v1/auth/register
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
