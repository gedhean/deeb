package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "test-secret-key-minimum-32-characters-long"

func TestNewAuthMiddleware(t *testing.T) {
	tm, err := auth.NewTokenManager(testJWTSecret)
	require.NoError(t, err)

	middleware := NewAuthMiddleware(tm)

	assert.NotNil(t, middleware)
	assert.NotNil(t, middleware.tokenManager)
}

func TestAuthenticate_Success(t *testing.T) {
	// Setup token manager and middleware
	tm, err := auth.NewTokenManager(testJWTSecret)
	require.NoError(t, err)

	middleware := NewAuthMiddleware(tm)

	// Generate valid token
	userID := uuid.New()
	email := "test@example.com"
	role := "OrganizationDeveloper"
	token, err := tm.GenerateToken(userID, email, role)
	require.NoError(t, err)

	// Create test handler that verifies context
	var capturedClaims *auth.Claims
	testHandler := func(c echo.Context) error {
		claims, ok := c.Get("user").(*auth.Claims)
		require.True(t, ok)
		capturedClaims = claims
		return c.NoContent(http.StatusOK)
	}

	// Create Echo instance and request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute middleware
	handler := middleware.Authenticate(testHandler)
	err = handler(c)

	// Verify response
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify claims were stored correctly
	require.NotNil(t, capturedClaims)
	assert.Equal(t, userID, capturedClaims.UserID)
	assert.Equal(t, email, capturedClaims.Email)
	assert.Equal(t, role, capturedClaims.OrganizationRole)
}

func TestAuthenticate_MissingAuthorizationHeader(t *testing.T) {
	tm, err := auth.NewTokenManager(testJWTSecret)
	require.NoError(t, err)

	middleware := NewAuthMiddleware(tm)

	// Create test handler (should not be called)
	testHandler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}

	// Create Echo instance and request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute middleware
	handler := middleware.Authenticate(testHandler)
	err = handler(c)

	// Verify response
	require.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Contains(t, httpErr.Message, "missing authentication")
}

func TestAuthenticate_InvalidAuthorizationHeaderFormat(t *testing.T) {
	tm, err := auth.NewTokenManager(testJWTSecret)
	require.NoError(t, err)

	middleware := NewAuthMiddleware(tm)

	testCases := []struct {
		name   string
		header string
	}{
		{
			name:   "missing Bearer prefix",
			header: "sometoken",
		},
		{
			name:   "wrong scheme",
			header: "Basic sometoken",
		},
		{
			name:   "only Bearer",
			header: "Bearer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test handler (should not be called)
			testHandler := func(c echo.Context) error {
				t.Fatal("handler should not be called")
				return nil
			}

			// Create Echo instance and request
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tc.header)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Execute middleware
			handler := middleware.Authenticate(testHandler)
			err := handler(c)

			// Verify response
			require.Error(t, err)
			httpErr, ok := err.(*echo.HTTPError)
			require.True(t, ok)
			assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
			assert.Contains(t, httpErr.Message, "invalid authorization header format")
		})
	}
}

func TestAuthenticate_EmptyToken(t *testing.T) {
	tm, err := auth.NewTokenManager(testJWTSecret)
	require.NoError(t, err)

	middleware := NewAuthMiddleware(tm)

	// Create test handler (should not be called)
	testHandler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}

	// Create Echo instance and request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute middleware
	handler := middleware.Authenticate(testHandler)
	err = handler(c)

	// Verify response
	require.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Contains(t, httpErr.Message, "missing token")
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	tm, err := auth.NewTokenManager(testJWTSecret)
	require.NoError(t, err)

	middleware := NewAuthMiddleware(tm)

	// Create test handler (should not be called)
	testHandler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}

	// Create Echo instance and request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.string")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute middleware
	handler := middleware.Authenticate(testHandler)
	err = handler(c)

	// Verify response
	require.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Contains(t, httpErr.Message, "invalid token")
}

func TestAuthenticate_ExpiredToken(t *testing.T) {
	tm, err := auth.NewTokenManager(testJWTSecret)
	require.NoError(t, err)

	middleware := NewAuthMiddleware(tm)

	// Create expired token
	claims := auth.Claims{
		UserID:           uuid.New(),
		Email:            "test@example.com",
		OrganizationRole: "OrganizationDeveloper",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-25 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-25 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Use reflection to get jwtSecret from tokenManager
	// For testing, we'll create a new token manager and extract the secret
	tokenString, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)

	// Create test handler (should not be called)
	testHandler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}

	// Create Echo instance and request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute middleware
	handler := middleware.Authenticate(testHandler)
	err = handler(c)

	// Verify response
	require.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Contains(t, httpErr.Message, "token expired")
}

func TestAuthenticate_WrongSecret(t *testing.T) {
	// Create token with one secret
	tm1, err := auth.NewTokenManager(testJWTSecret)
	require.NoError(t, err)

	token, err := tm1.GenerateToken(uuid.New(), "test@example.com", "OrganizationDeveloper")
	require.NoError(t, err)

	// Create middleware with different secret
	tm2, err := auth.NewTokenManager("different-secret-key-minimum-32-characters-long")
	require.NoError(t, err)

	middleware := NewAuthMiddleware(tm2)

	// Create test handler (should not be called)
	testHandler := func(c echo.Context) error {
		t.Fatal("handler should not be called")
		return nil
	}

	// Create Echo instance and request
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute middleware
	handler := middleware.Authenticate(testHandler)
	err = handler(c)

	// Verify response
	require.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Contains(t, httpErr.Message, "invalid token")
}

func TestGetUserFromContext_Success(t *testing.T) {
	expectedClaims := &auth.Claims{
		UserID:           uuid.New(),
		Email:            "test@example.com",
		OrganizationRole: "OrganizationAdmin",
	}

	ctx := context.WithValue(context.Background(), UserContextKey, expectedClaims)

	claims, ok := GetUserFromContext(ctx)

	assert.True(t, ok)
	assert.Equal(t, expectedClaims, claims)
}

func TestGetUserFromContext_NotFound(t *testing.T) {
	ctx := context.Background()

	claims, ok := GetUserFromContext(ctx)

	assert.False(t, ok)
	assert.Nil(t, claims)
}

func TestGetUserFromContext_WrongType(t *testing.T) {
	// Store wrong type in context
	ctx := context.WithValue(context.Background(), UserContextKey, "not-a-claims-object")

	claims, ok := GetUserFromContext(ctx)

	assert.False(t, ok)
	assert.Nil(t, claims)
}

func TestAuthenticate_MultipleRequests(t *testing.T) {
	tm, err := auth.NewTokenManager(testJWTSecret)
	require.NoError(t, err)

	middleware := NewAuthMiddleware(tm)

	// Generate tokens for different users
	userID1 := uuid.New()
	userID2 := uuid.New()
	token1, err := tm.GenerateToken(userID1, "user1@example.com", "OrganizationAdmin")
	require.NoError(t, err)
	token2, err := tm.GenerateToken(userID2, "user2@example.com", "OrganizationDeveloper")
	require.NoError(t, err)

	// Create test handler that captures user ID
	var capturedUserIDs []uuid.UUID
	testHandler := func(c echo.Context) error {
		claims, ok := c.Get("user").(*auth.Claims)
		require.True(t, ok)
		capturedUserIDs = append(capturedUserIDs, claims.UserID)
		return c.NoContent(http.StatusOK)
	}

	handler := middleware.Authenticate(testHandler)

	// Create Echo instance
	e := echo.New()

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req1.Header.Set("Authorization", "Bearer "+token1)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)
	err = handler(c1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// Second request
	req2 := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	err = handler(c2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec2.Code)

	// Verify both requests were handled correctly
	require.Len(t, capturedUserIDs, 2)
	assert.Equal(t, userID1, capturedUserIDs[0])
	assert.Equal(t, userID2, capturedUserIDs[1])
}

func BenchmarkAuthenticate(b *testing.B) {
	tm, _ := auth.NewTokenManager(testJWTSecret)
	middleware := NewAuthMiddleware(tm)

	token, _ := tm.GenerateToken(uuid.New(), "test@example.com", "OrganizationDeveloper")

	testHandler := func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}

	handler := middleware.Authenticate(testHandler)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		handler(c)
	}
}
