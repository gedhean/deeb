package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/middleware"
	"go.uber.org/zap"
)

// LogoutHandler handles user logout requests.
type LogoutHandler struct {
	logger *zap.Logger
}

// NewLogoutHandler creates a new LogoutHandler with the provided logger.
func NewLogoutHandler(logger *zap.Logger) *LogoutHandler {
	return &LogoutHandler{
		logger: logger,
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

	// Enrich audit log if user was authenticated
	if userID != uuid.Nil {
		c.Set("audit_enrichment", &middleware.AuditEnrichment{
			UserID: &userID, // Include user ID for audit log
			Result: "success",
		})
		h.logger.Info("User logged out successfully", zap.String("user_id", userID.String()))
	} else {
		h.logger.Info("Logout called without authentication")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}
