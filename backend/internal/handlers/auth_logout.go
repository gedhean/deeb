package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
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

	h.logger.Info("User logged out successfully")

	return c.JSON(http.StatusOK, map[string]string{
		"message": "logged out successfully",
	})
}
