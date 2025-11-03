package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/services"
	"go.uber.org/zap"
)

// LogoutHandler handles user logout requests.
type LogoutHandler struct {
	auditService *services.AuditLogService
	logger       *zap.Logger
}

// NewLogoutHandler creates a new LogoutHandler with the provided logger.
func NewLogoutHandler(auditService *services.AuditLogService, logger *zap.Logger) *LogoutHandler {
	return &LogoutHandler{
		auditService: auditService,
		logger:       logger,
	}
}

// Logout handles POST /api/v1/auth/logout.
// Clears the authentication cookie to log the user out.
func (h *LogoutHandler) Logout(c echo.Context) error {
	// Extract user ID from context if authenticated (set by AuthMiddleware)
	var userID uuid.UUID
	if userClaims, ok := c.Get("user").(*auth.Claims); ok && userClaims != nil {
		userID = userClaims.UserID
	}

	// Clear the authentication cookie
	cookie := &http.Cookie{
		Name:     "deeb_auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().URL.Scheme == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1, // Delete cookie immediately
	}
	c.SetCookie(cookie)

	// Log logout if user was authenticated
	if userID != uuid.Nil && h.auditService != nil {
		ctx := c.Request().Context()
		detailsJSON, _ := json.Marshal(map[string]interface{}{})

		if err := h.auditService.LogWithHTTP(ctx, userID, services.ActionLogout, services.ResourceTypeUser, userID, c.Request(), detailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for logout",
				zap.String("user_id", userID.String()),
				zap.Error(err))
		}
	}

	h.logger.Info("User logged out successfully")

	return c.JSON(http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}
