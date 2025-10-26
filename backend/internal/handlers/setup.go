// Package handlers provides HTTP request handlers for the Deeb API.
package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/repository"
)

// SetupHandler handles setup and initialization endpoints.
type SetupHandler struct {
	userRepo repository.UserRepository
}

// NewSetupHandler creates a new setup handler.
func NewSetupHandler(userRepo repository.UserRepository) *SetupHandler {
	return &SetupHandler{
		userRepo: userRepo,
	}
}

// SetupStatusResponse represents the setup status of the system.
type SetupStatusResponse struct {
	RequiresSetup bool `json:"requires_setup"`
	UserCount     int  `json:"user_count"`
}

// GetSetupStatus returns whether the system requires initial setup.
// Returns requires_setup=true if no users exist in the system.
//
// @Summary Get setup status
// @Description Check if the system requires initial setup (first user creation)
// @Tags setup
// @Accept json
// @Produce json
// @Success 200 {object} SetupStatusResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/setup/status [get]
func (h *SetupHandler) GetSetupStatus(c echo.Context) error {
	ctx := c.Request().Context()

	users, err := h.userRepo.List(ctx, 1, 0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to check setup status",
		})
	}

	return c.JSON(http.StatusOK, SetupStatusResponse{
		RequiresSetup: len(users) == 0,
		UserCount:     len(users),
	})
}
