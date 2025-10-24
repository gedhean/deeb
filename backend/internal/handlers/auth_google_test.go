package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/nsxbet/deeb/internal/auth"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestGoogleOAuthRedirect tests the initial redirect to Google
func TestGoogleOAuthRedirect(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-32-characters!!!")
	assert.NoError(t, err)

	handler := NewGoogleOAuthHandler(
		mockUserRepo,
		tokenManager,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		logger,
	)

	req := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)

	location := w.Header().Get("Location")
	assert.NotEmpty(t, location)
	assert.Contains(t, location, "accounts.google.com")
	assert.Contains(t, location, "client_id=test-client-id")
	assert.Contains(t, location, "redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fcallback")
	assert.Contains(t, location, "state=")

	// Verify state was stored
	parsedURL, parseErr := url.Parse(location)
	assert.NoError(t, parseErr)
	state := parsedURL.Query().Get("state")
	assert.NotEmpty(t, state)

	_, exists := handler.states[state]
	assert.True(t, exists)
}

// TestGoogleOAuthRedirect_WrongMethod tests non-GET request
func TestGoogleOAuthRedirect_WrongMethod(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-32-characters!!!")
	assert.NoError(t, err)

	handler := NewGoogleOAuthHandler(
		mockUserRepo,
		tokenManager,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		logger,
	)

	req := httptest.NewRequest(http.MethodPost, "/auth/google", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// TestGoogleOAuthCallback_MissingState tests callback without state
func TestGoogleOAuthCallback_MissingState(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-32-characters!!!")
	assert.NoError(t, err)

	handler := NewGoogleOAuthHandler(
		mockUserRepo,
		tokenManager,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		logger,
	)

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=test-code", nil)
	w := httptest.NewRecorder()

	handler.HandleCallback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "missing state parameter", resp.Error)
}

// TestGoogleOAuthCallback_InvalidState tests callback with invalid state
func TestGoogleOAuthCallback_InvalidState(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-32-characters!!!")
	assert.NoError(t, err)

	handler := NewGoogleOAuthHandler(
		mockUserRepo,
		tokenManager,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		logger,
	)

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state=invalid-state&code=test-code", nil)
	w := httptest.NewRecorder()

	handler.HandleCallback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "invalid state parameter", resp.Error)
}

// TestGoogleOAuthCallback_ExpiredState tests callback with expired state
func TestGoogleOAuthCallback_ExpiredState(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-32-characters!!!")
	assert.NoError(t, err)

	handler := NewGoogleOAuthHandler(
		mockUserRepo,
		tokenManager,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		logger,
	)

	// Add expired state (11 minutes ago)
	expiredState := "expired-state"
	handler.states[expiredState] = time.Now().Add(-11 * time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state="+expiredState+"&code=test-code", nil)
	w := httptest.NewRecorder()

	handler.HandleCallback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "expired state parameter", resp.Error)

	// Verify expired state was removed
	_, exists := handler.states[expiredState]
	assert.False(t, exists)
}

// TestGoogleOAuthCallback_ErrorFromGoogle tests callback with error from Google
func TestGoogleOAuthCallback_ErrorFromGoogle(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-32-characters!!!")
	assert.NoError(t, err)

	handler := NewGoogleOAuthHandler(
		mockUserRepo,
		tokenManager,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		logger,
	)

	// Add valid state
	validState := "valid-state"
	handler.states[validState] = time.Now()

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state="+validState+"&error=access_denied", nil)
	w := httptest.NewRecorder()

	handler.HandleCallback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Contains(t, resp.Error, "oauth error")
}

// TestGoogleOAuthCallback_MissingCode tests callback without authorization code
func TestGoogleOAuthCallback_MissingCode(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-32-characters!!!")
	assert.NoError(t, err)

	handler := NewGoogleOAuthHandler(
		mockUserRepo,
		tokenManager,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		logger,
	)

	// Add valid state
	validState := "valid-state"
	handler.states[validState] = time.Now()

	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state="+validState, nil)
	w := httptest.NewRecorder()

	handler.HandleCallback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "missing code parameter", resp.Error)
}

// TestGoogleOAuthCallback_WrongMethod tests callback with wrong HTTP method
func TestGoogleOAuthCallback_WrongMethod(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-32-characters!!!")
	assert.NoError(t, err)

	handler := NewGoogleOAuthHandler(
		mockUserRepo,
		tokenManager,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		logger,
	)

	req := httptest.NewRequest(http.MethodPost, "/auth/google/callback", nil)
	w := httptest.NewRecorder()

	handler.HandleCallback(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// TestGoogleOAuthCallback_DatabaseError tests callback with database error
func TestGoogleOAuthCallback_DatabaseError(t *testing.T) {
	// Note: This test would require mocking the OAuth exchange and user info fetch
	// which is complex due to external HTTP calls. In a real implementation,
	// you would use dependency injection for the HTTP client or use integration tests.
	t.Skip("Skipping test that requires mocking external OAuth flow")
}

// TestGoogleOAuthCallback_InactiveUser tests callback for inactive user
func TestGoogleOAuthCallback_InactiveUser(t *testing.T) {
	// Note: This test would require mocking the OAuth exchange and user info fetch
	t.Skip("Skipping test that requires mocking external OAuth flow")
}

// TestGenerateStateToken tests state token generation
func TestGenerateStateToken(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-32-characters!!!")
	assert.NoError(t, err)

	handler := NewGoogleOAuthHandler(
		mockUserRepo,
		tokenManager,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		logger,
	)

	// Generate multiple tokens and ensure they're unique
	token1, err := handler.generateStateToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token1)

	token2, err := handler.generateStateToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token2)

	assert.NotEqual(t, token1, token2)

	// Verify base64 URL encoding
	assert.False(t, strings.Contains(token1, "+"))
	assert.False(t, strings.Contains(token1, "/"))
}

// TestStateCleanup tests that used states are removed
func TestStateCleanup(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-32-characters!!!")
	assert.NoError(t, err)

	handler := NewGoogleOAuthHandler(
		mockUserRepo,
		tokenManager,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		logger,
	)

	// Add valid state
	validState := "valid-state"
	handler.states[validState] = time.Now()

	// Verify state exists
	_, exists := handler.states[validState]
	assert.True(t, exists)

	// Make callback request (will fail due to missing code, but should clean up state)
	req := httptest.NewRequest(http.MethodGet, "/auth/google/callback?state="+validState, nil)
	w := httptest.NewRecorder()

	handler.HandleCallback(w, req)

	// Verify state was removed
	_, exists = handler.states[validState]
	assert.False(t, exists)
}

// TestOAuthConfig tests that OAuth config is properly initialized
func TestOAuthConfig(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	logger := zap.NewNop()
	tokenManager, err := auth.NewTokenManager("test-secret-key-32-characters!!!")
	assert.NoError(t, err)

	handler := NewGoogleOAuthHandler(
		mockUserRepo,
		tokenManager,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		logger,
	)

	assert.Equal(t, "test-client-id", handler.oauthConfig.ClientID)
	assert.Equal(t, "test-client-secret", handler.oauthConfig.ClientSecret)
	assert.Equal(t, "http://localhost:8080/callback", handler.oauthConfig.RedirectURL)
	assert.Contains(t, handler.oauthConfig.Scopes, "https://www.googleapis.com/auth/userinfo.email")
	assert.Contains(t, handler.oauthConfig.Scopes, "https://www.googleapis.com/auth/userinfo.profile")
}

// Note: Integration tests with actual OAuth flow would require either:
// 1. Mocking the HTTP client used by oauth2.Config
// 2. Using a test OAuth provider
// 3. Running integration tests against Google's OAuth endpoints (not recommended for unit tests)
//
// For comprehensive testing, consider:
// - Integration tests with a mock OAuth server
// - End-to-end tests with test Google credentials
// - Manual testing in development environment
