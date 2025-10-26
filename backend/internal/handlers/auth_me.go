package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// MeHandler handles requests for the current authenticated user's information.
type MeHandler struct {
	userRepo repository.UserRepository
	logger   *zap.Logger
}

// NewMeHandler creates a new MeHandler with the provided dependencies.
func NewMeHandler(userRepo repository.UserRepository, logger *zap.Logger) *MeHandler {
	return &MeHandler{
		userRepo: userRepo,
		logger:   logger,
	}
}

// GetCurrentUser handles GET /api/v1/auth/me.
// Returns the current authenticated user's information extracted from the JWT token claims.
func (h *MeHandler) GetCurrentUser(c echo.Context) error {
	// Extract user claims from context (set by auth middleware)
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		h.logger.Error("Failed to extract user claims from context")
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	}

	ctx := c.Request().Context()

	// Fetch full user details from database
	user, err := h.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		h.logger.Error("Failed to fetch user by ID",
			zap.String("user_id", claims.UserID.String()),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}

	if user == nil {
		h.logger.Warn("User not found in database",
			zap.String("user_id", claims.UserID.String()))
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
	}

	// Check if user is still active
	if !user.IsActive {
		h.logger.Info("Inactive user attempted to access /me endpoint",
			zap.String("user_id", user.ID.String()))
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "account is inactive"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user": user.ToResponse(),
	})
}
