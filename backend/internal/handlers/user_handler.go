package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/middleware"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// UserHandler handles user management endpoints
type UserHandler struct {
	userRepo                 repository.UserRepository
	projectAssignmentService ProjectAssignmentService
	logger                   *zap.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	userRepo repository.UserRepository,
	projectAssignmentService ProjectAssignmentService,
	logger *zap.Logger,
) *UserHandler {
	return &UserHandler{
		userRepo:                 userRepo,
		projectAssignmentService: projectAssignmentService,
		logger:                   logger,
	}
}

// CreateUser handles POST /api/v1/users - create a new user (admin only)
func (h *UserHandler) CreateUser(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse create request
	var req models.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return middleware.HandleValidationError(c, err)
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return middleware.HandleValidationError(c, err)
	}

	// Normalize email
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Check for duplicate email
	existingUser, err := h.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		h.logger.Error("Failed to check for existing user", zap.Error(err), zap.String("email", req.Email))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to check user existence",
		})
	}
	if existingUser != nil {
		return c.JSON(http.StatusConflict, map[string]interface{}{
			"error": "Email already registered",
		})
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password, 0) // Use default cost (12)
	if err != nil {
		h.logger.Debug("Password validation failed", zap.Error(err))
		if errors.Is(err, auth.ErrPasswordTooShort) {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Password must be at least 8 characters",
			})
		}
		if errors.Is(err, auth.ErrPasswordTooLong) {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Password must not exceed 72 characters",
			})
		}
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid password",
		})
	}

	// Create user
	user := &models.User{
		Email:            req.Email,
		PasswordHash:     passwordHash,
		DisplayName:      req.DisplayName,
		OrganizationRole: req.OrganizationRole,
		TwoFactorEnabled: false,
		IsActive:         true,
	}

	if err := h.userRepo.Create(ctx, user); err != nil {
		h.logger.Error("Failed to create user", zap.Error(err), zap.String("email", req.Email))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create user",
		})
	}

	h.logger.Info("User created successfully",
		zap.String("email", user.Email),
		zap.String("user_id", user.ID.String()),
		zap.String("organization_role", string(user.OrganizationRole)))

	// Auto-assign user to existing projects (skip OrganizationAdmin)
	if user.OrganizationRole != rbac.OrganizationAdmin {
		if count, err := h.projectAssignmentService.AssignUserToExistingProjects(ctx, user.ID); err != nil {
			h.logger.Error("Failed to assign user to projects",
				zap.Error(err),
				zap.String("user_id", user.ID.String()))
			// Log error but don't fail user creation
		} else if count > 0 {
			h.logger.Info("Auto-assigned user to projects",
				zap.String("user_id", user.ID.String()),
				zap.Int("project_count", count))
		}
	}

	return c.JSON(http.StatusCreated, user.ToResponse())
}

// ListUsers handles GET /api/v1/users - list all users
func (h *UserHandler) ListUsers(c echo.Context) error {
	ctx := c.Request().Context()

	// Get pagination parameters with defaults
	limit := 100
	offset := 0

	// Parse query parameters if provided
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if parsedLimit, err := parseInt(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		if parsedOffset, err := parseInt(offsetParam); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Build filters from query parameters
	var filters repository.UserFilters
	filters.Search = c.QueryParam("search")
	filters.OrganizationRole = c.QueryParam("organization_role")
	filters.IsActive = parseBoolParam(c, "is_active")

	createdAfter, err := parseTimeParam(c, "created_after")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid created_after (must be RFC3339)",
		})
	}
	createdBefore, err := parseTimeParam(c, "created_before")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid created_before (must be RFC3339)",
		})
	}
	filters.CreatedAfter = createdAfter
	filters.CreatedBefore = createdBefore

	// Fetch users from repository
	users, err := h.userRepo.List(ctx, filters, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list users", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve users",
		})
	}

	// Get total count
	totalCount, err := h.userRepo.Count(ctx, filters)
	if err != nil {
		h.logger.Error("Failed to count users", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to count users",
		})
	}

	// Convert to response format
	responses := make([]*models.UserResponse, len(users))
	for i, user := range users {
		responses[i] = user.ToResponse()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users":  responses,
		"limit":  limit,
		"offset": offset,
		"total":  totalCount,
	})
}

// GetUser handles GET /api/v1/users/:id - get user by ID
func (h *UserHandler) GetUser(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "User not authenticated",
		})
	}

	// Parse user ID from path parameter
	idParam := c.Param("id")
	userID, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid user ID format",
		})
	}

	// Fetch user from repository
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, repository.ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "User not found",
			})
		}
		h.logger.Error("Failed to get user", zap.Error(err), zap.String("user_id", userID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve user",
		})
	}

	// Allow users to view their own profile, or admins to view any profile
	// Check if user is viewing their own profile
	isOwnProfile := claims.UserID == userID
	isAdmin := claims.OrganizationRole == "OrganizationAdmin"

	if !isOwnProfile && !isAdmin {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "You can only view your own profile",
		})
	}

	return c.JSON(http.StatusOK, user.ToResponse())
}

// UpdateUser handles PUT /api/v1/users/:id - update user
func (h *UserHandler) UpdateUser(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "User not authenticated",
		})
	}

	// Parse user ID from path parameter
	idParam := c.Param("id")
	userID, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid user ID format",
		})
	}

	// Parse update request
	var req models.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}

	// Fetch existing user
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "User not found",
			})
		}
		h.logger.Error("Failed to get user for update", zap.Error(err), zap.String("user_id", userID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve user",
		})
	}

	// Check authorization: users can update their own profile (limited fields), admins can update any user
	isOwnProfile := claims.UserID == userID
	isAdmin := claims.OrganizationRole == "OrganizationAdmin"

	if !isOwnProfile && !isAdmin {
		return c.JSON(http.StatusForbidden, map[string]interface{}{
			"error": "You can only update your own profile",
		})
	}

	// If updating own profile, restrict to display_name and password only
	if isOwnProfile && !isAdmin {
		// Only allow updating display_name and password for own profile
		if req.OrganizationRole != nil {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "You cannot change your own organization role",
			})
		}
		if req.IsActive != nil {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "You cannot change your own active status",
			})
		}
		if req.TwoFactorEnabled != nil {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"error": "You cannot change your own two-factor authentication settings",
			})
		}
	}

	// Apply updates
	if req.Password != nil && *req.Password != "" {
		// Hash new password
		passwordHash, err := auth.HashPassword(*req.Password, 0) // Use default cost (12)
		if err != nil {
			h.logger.Debug("Password validation failed", zap.Error(err))
			if errors.Is(err, auth.ErrPasswordTooShort) {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Password must be at least 8 characters",
				})
			}
			if errors.Is(err, auth.ErrPasswordTooLong) {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error": "Password must not exceed 72 characters",
				})
			}
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid password",
			})
		}
		user.PasswordHash = passwordHash
	}
	if req.DisplayName != nil {
		user.DisplayName = req.DisplayName
	}
	if req.OrganizationRole != nil {
		user.OrganizationRole = *req.OrganizationRole
	}
	if req.TwoFactorEnabled != nil {
		user.TwoFactorEnabled = *req.TwoFactorEnabled
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	// Update user in repository
	if err := h.userRepo.Update(ctx, user); err != nil {
		h.logger.Error("Failed to update user", zap.Error(err), zap.String("user_id", userID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update user",
		})
	}

	h.logger.Info("User updated successfully", zap.String("user_id", userID.String()))
	return c.JSON(http.StatusOK, user.ToResponse())
}

// DeleteUser handles DELETE /api/v1/users/:id - deactivate user (soft delete)
func (h *UserHandler) DeleteUser(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse user ID from path parameter
	idParam := c.Param("id")
	userID, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid user ID format",
		})
	}

	// Fetch existing user
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "User not found",
			})
		}
		h.logger.Error("Failed to get user for deletion", zap.Error(err), zap.String("user_id", userID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve user",
		})
	}

	// Soft delete: set is_active to false
	user.IsActive = false

	// Update user in repository
	if err := h.userRepo.Update(ctx, user); err != nil {
		h.logger.Error("Failed to deactivate user", zap.Error(err), zap.String("user_id", userID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to deactivate user",
		})
	}

	h.logger.Info("User deactivated successfully", zap.String("user_id", userID.String()))
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User deactivated successfully",
		"user":    user.ToResponse(),
	})
}

// Helper function to parse integer query parameters
func parseInt(s string) (int, error) {
	var i int
	if _, err := fmt.Sscanf(s, "%d", &i); err != nil {
		return 0, err
	}
	return i, nil
}
