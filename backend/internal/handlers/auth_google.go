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
}

// NewGoogleOAuthHandler creates a new Google OAuth handler
func NewGoogleOAuthHandler(
	userRepo repository.UserRepository,
	tokenManager *auth.TokenManager,
	clientID string,
	clientSecret string,
	callbackURL string,
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
		logger: logger,
		states: make(map[string]time.Time),
	}
}

// GoogleUserInfo represents the user information returned by Google
type GoogleUserInfo struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	EmailVerified bool   `json:"email_verified"`
}

// ServeHTTP handles the initial OAuth redirect to Google
func (h *GoogleOAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Generate CSRF state token
	state, err := h.generateStateToken()
	if err != nil {
		h.logger.Error("Failed to generate state token", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Store state with timestamp for validation
	h.states[state] = time.Now()

	// Redirect to Google OAuth
	url := h.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	h.logger.Debug("Redirecting to Google OAuth", zap.String("state", state))
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// HandleCallback handles the OAuth callback from Google
func (h *GoogleOAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()

	// Validate state parameter (CSRF protection)
	state := r.URL.Query().Get("state")
	if state == "" {
		h.logger.Warn("OAuth callback missing state parameter")
		h.respondError(w, http.StatusBadRequest, "missing state parameter")
		return
	}

	timestamp, exists := h.states[state]
	if !exists {
		h.logger.Warn("OAuth callback with invalid state", zap.String("state", state))
		h.respondError(w, http.StatusBadRequest, "invalid state parameter")
		return
	}

	// State should expire after 10 minutes
	if time.Since(timestamp) > 10*time.Minute {
		delete(h.states, state)
		h.logger.Warn("OAuth callback with expired state", zap.String("state", state))
		h.respondError(w, http.StatusBadRequest, "expired state parameter")
		return
	}

	// Remove used state
	delete(h.states, state)

	// Check for error from Google
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		h.logger.Warn("OAuth error from Google", zap.String("error", errMsg))
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("oauth error: %s", errMsg))
		return
	}

	// Exchange authorization code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		h.logger.Warn("OAuth callback missing code parameter")
		h.respondError(w, http.StatusBadRequest, "missing code parameter")
		return
	}

	token, err := h.oauthConfig.Exchange(ctx, code)
	if err != nil {
		h.logger.Error("Failed to exchange code for token", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to exchange authorization code")
		return
	}

	// Fetch user info from Google
	userInfo, err := h.fetchGoogleUserInfo(ctx, token)
	if err != nil {
		h.logger.Error("Failed to fetch user info from Google", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to fetch user information")
		return
	}

	// Validate email is verified
	if !userInfo.EmailVerified {
		h.logger.Warn("Google OAuth with unverified email", zap.String("email", userInfo.Email))
		h.respondError(w, http.StatusBadRequest, "email not verified with Google")
		return
	}

	// Normalize email
	email := normalizeEmail(userInfo.Email)

	// Check if user exists
	user, err := h.userRepo.GetByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		h.logger.Error("Database error during Google OAuth", zap.Error(err), zap.String("email", email))
		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Create new user if doesn't exist
	if user == nil {
		user = &models.User{
			Email:            email,
			PasswordHash:     "", // No password for OAuth users
			IsActive:         true,
			OrganizationRole: "OrganizationDeveloper",
		}

		if err := h.userRepo.Create(ctx, user); err != nil {
			h.logger.Error("Failed to create user from Google OAuth",
				zap.Error(err),
				zap.String("email", email))
			h.respondError(w, http.StatusInternalServerError, "failed to create user")
			return
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
		h.respondError(w, http.StatusUnauthorized, "account is inactive")
		return
	}

	// Generate JWT token
	jwtToken, err := h.tokenManager.GenerateToken(user.ID, user.Email, user.OrganizationRole)
	if err != nil {
		h.logger.Error("Failed to generate token",
			zap.Error(err),
			zap.String("user_id", user.ID.String()))
		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	response := LoginResponse{
		Token: jwtToken,
		User:  user.ToResponse(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// fetchGoogleUserInfo fetches user information from Google using the access token
func (h *GoogleOAuthHandler) fetchGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := h.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

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

func (h *GoogleOAuthHandler) respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// normalizeEmail normalizes an email address (lowercase and trim whitespace)
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
