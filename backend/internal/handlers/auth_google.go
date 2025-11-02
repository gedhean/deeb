package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleOAuthHandler handles Google OAuth authentication
type GoogleOAuthHandler struct {
	userRepo     repository.UserRepository
	tokenManager *auth.TokenManager
	oauthConfig  *oauth2.Config
	logger       *zap.Logger
	states       map[string]time.Time // In-memory state tracking for CSRF protection
	redirectURL  string               // Frontend URL for OAuth redirects
}

// NewGoogleOAuthHandler creates a new Google OAuth handler
func NewGoogleOAuthHandler(
	userRepo repository.UserRepository,
	tokenManager *auth.TokenManager,
	clientID string,
	clientSecret string,
	callbackURL string,
	redirectURL string,
	logger *zap.Logger,
) *GoogleOAuthHandler {
	return &GoogleOAuthHandler{
		userRepo:     userRepo,
		tokenManager: tokenManager,
		oauthConfig: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  callbackURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
		logger:      logger,
		states:      make(map[string]time.Time),
		redirectURL: redirectURL,
	}
}

// GoogleUserInfo represents the user information returned by Google's OAuth API.
type GoogleUserInfo struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	VerifiedEmail bool   `json:"verified_email"`
}

// InitiateGoogleAuth handles GET /api/v1/auth/google.
// Initiates the Google OAuth flow by generating a state token for CSRF protection
// and redirecting the user to Google's authorization page.
func (h *GoogleOAuthHandler) InitiateGoogleAuth(c echo.Context) error {
	if c.Request().Method != http.MethodGet {
		return c.JSON(http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
	}

	// Generate CSRF state token
	state, err := h.generateStateToken()
	if err != nil {
		h.logger.Error("Failed to generate state token", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}

	// Store state with timestamp for validation
	h.states[state] = time.Now()

	// Redirect to Google OAuth
	url := h.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	h.logger.Debug("Redirecting to Google OAuth", zap.String("state", state))
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback handles GET /api/v1/auth/google/callback.
// Processes the OAuth callback from Google, validates the state token,
// exchanges the authorization code for user information, and creates or logs in the user.
func (h *GoogleOAuthHandler) GoogleCallback(c echo.Context) error {
	if c.Request().Method != http.MethodGet {
		return c.JSON(http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
	}

	ctx := c.Request().Context()

	// Validate state parameter (CSRF protection)
	state := c.QueryParam("state")
	if state == "" {
		h.logger.Warn("OAuth callback missing state parameter")
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "missing state parameter"})
	}

	timestamp, exists := h.states[state]
	if !exists {
		h.logger.Warn("OAuth callback with invalid state", zap.String("state", state))
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid state parameter"})
	}

	// State should expire after 10 minutes
	if time.Since(timestamp) > 10*time.Minute {
		delete(h.states, state)
		h.logger.Warn("OAuth callback with expired state", zap.String("state", state))
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "expired state parameter"})
	}

	// Remove used state
	delete(h.states, state)

	// Check for error from Google
	if errMsg := c.QueryParam("error"); errMsg != "" {
		h.logger.Warn("OAuth error from Google", zap.String("error", errMsg))
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("oauth error: %s", errMsg)})
	}

	// Exchange authorization code for token
	code := c.QueryParam("code")
	if code == "" {
		h.logger.Warn("OAuth callback missing code parameter")
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "missing code parameter"})
	}

	token, err := h.oauthConfig.Exchange(ctx, code)
	if err != nil {
		h.logger.Error("Failed to exchange code for token", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to exchange authorization code"})
	}

	// Fetch user info from Google
	userInfo, err := h.fetchGoogleUserInfo(ctx, token)
	if err != nil {
		h.logger.Error("Failed to fetch user info from Google", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch user information"})
	}

	// Validate email is verified
	if !userInfo.VerifiedEmail {
		h.logger.Warn("Google OAuth with unverified email", zap.String("email", userInfo.Email))
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email not verified with Google"})
	}

	// Normalize email
	email := normalizeEmail(userInfo.Email)

	// Check if user exists
	user, err := h.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		h.logger.Error("Database error during Google OAuth", zap.Error(err), zap.String("email", email))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}

	// Determine organization role for new users
	var orgRole string
	if user == nil {
		// Check if this is the first user (initial setup)
		users, err := h.userRepo.List(ctx, 1, 0)
		if err != nil {
			h.logger.Error("Failed to check user count for first user setup", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		}

		if len(users) == 0 {
			// First user - automatically grant OrganizationAdmin role
			orgRole = "OrganizationAdmin"
			h.logger.Info(
				"First user registration via Google OAuth - granting OrganizationAdmin role",
				zap.String("email", email),
			)
		} else {
			// Subsequent users - use default role
			orgRole = "OrganizationDeveloper"
		}
	}

	// Create new user if doesn't exist
	if user == nil {
		user = &models.User{
			Email:            email,
			PasswordHash:     "", // No password for OAuth users
			IsActive:         true,
			OrganizationRole: models.OrganizationRole(orgRole),
			OAuthProvider:    stringPtr("google"),
			OAuthID:          stringPtr(userInfo.Email), // Using email as OAuth ID for now
		}

		if err := h.userRepo.Create(ctx, user); err != nil {
			h.logger.Error("Failed to create user from Google OAuth",
				zap.Error(err),
				zap.String("email", email))
			return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create user"})
		}

		h.logger.Info("Created new user from Google OAuth",
			zap.String("user_id", user.ID.String()),
			zap.String("email", email))
	} else {
		h.logger.Info("Existing user logged in via Google OAuth",
			zap.String("user_id", user.ID.String()),
			zap.String("email", email))
	}

	// Check if user is active
	if !user.IsActive {
		h.logger.Info("Google OAuth attempt by inactive user",
			zap.String("email", email),
			zap.String("user_id", user.ID.String()))
		return c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "account is inactive"})
	}

	// Generate JWT token
	jwtToken, err := h.tokenManager.GenerateToken(user.ID, user.Email, string(user.OrganizationRole))
	if err != nil {
		h.logger.Error("Failed to generate token",
			zap.Error(err),
			zap.String("user_id", user.ID.String()))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}

	// Update last login timestamp
	if err := h.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		h.logger.Warn("Failed to update last login timestamp",
			zap.Error(err),
			zap.String("user_id", user.ID.String()))
		// Don't fail the OAuth flow if this update fails
	}

	// Set JWT token in HttpOnly cookie
	cookie := &http.Cookie{
		Name:     "deeb_auth_token",
		Value:    jwtToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   c.Request().URL.Scheme == "https", // HTTPS only in production
		SameSite: http.SameSiteStrictMode,           // Strict for production
		MaxAge:   86400,                             // 24 hours (matches token expiration)
	}
	c.SetCookie(cookie)

	// Redirect to frontend dashboard after successful OAuth
	return c.Redirect(http.StatusFound, h.redirectURL+"/dashboard")
}

// fetchGoogleUserInfo fetches user information from Google using the access token
func (h *GoogleOAuthHandler) fetchGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := h.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			h.logger.Warn("Failed to close response body", zap.Error(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from Google: %d", resp.StatusCode)
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// generateStateToken generates a secure random state token for CSRF protection
func (h *GoogleOAuthHandler) generateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// normalizeEmail normalizes an email address (lowercase and trim whitespace)
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}
