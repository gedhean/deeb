package handlers

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// UserHandler handles user management endpoints
type UserHandler struct {
	userRepo repository.UserRepository
	logger   *zap.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userRepo repository.UserRepository, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
		logger:   logger,
	}
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

	// Fetch users from repository
	users, err := h.userRepo.List(ctx, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list users", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve users",
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
		"count":  len(responses),
	})
}

// GetUser handles GET /api/v1/users/:id - get user by ID
func (h *UserHandler) GetUser(c echo.Context) error {
	ctx := c.Request().Context()

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
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "User not found",
			})
		}
		h.logger.Error("Failed to get user", zap.Error(err), zap.String("user_id", userID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve user",
		})
	}

	return c.JSON(http.StatusOK, user.ToResponse())
}

// UpdateUser handles PUT /api/v1/users/:id - update user
func (h *UserHandler) UpdateUser(c echo.Context) error {
	ctx := c.Request().Context()

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

	// Apply updates
	if req.OrganizationRole != nil {
		user.OrganizationRole = *req.OrganizationRole
	}
	if req.TwoFactorEnabled != nil {
		user.TwoFactorEnabled = *req.TwoFactorEnabled
	}
	if req.Status != nil {
		user.Status = *req.Status
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

	// Soft delete: set status to Inactive
	user.Status = "Inactive"

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
