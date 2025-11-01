package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "test-secret-key-minimum-32-characters-long"

func TestNewTokenManager_Success(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)

	require.NoError(t, err)
	assert.NotNil(t, tm)
	assert.Equal(t, []byte(testJWTSecret), tm.jwtSecret)
}

func TestNewTokenManager_EmptySecret(t *testing.T) {
	tm, err := NewTokenManager("", 0)

	assert.Error(t, err)
	assert.Nil(t, tm)
	assert.Contains(t, err.Error(), "JWT secret cannot be empty")
}

func TestNewTokenManager_SecretTooShort(t *testing.T) {
	shortSecret := "short"

	tm, err := NewTokenManager(shortSecret, 0)

	assert.Error(t, err)
	assert.Nil(t, tm)
	assert.Contains(t, err.Error(), "JWT secret must be at least 32 characters")
}

func TestGenerateToken_Success(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	userID := uuid.New()
	email := "test@example.com"
	role := "OrganizationDeveloper"

	token, err := tm.GenerateToken(userID, email, role)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify token can be validated
	claims, err := tm.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.OrganizationRole)
}

func TestGenerateToken_NilUserID(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	token, err := tm.GenerateToken(uuid.Nil, "test@example.com", "OrganizationDeveloper")

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "user ID cannot be nil")
}

func TestGenerateToken_EmptyEmail(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	token, err := tm.GenerateToken(uuid.New(), "", "OrganizationDeveloper")

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "email cannot be empty")
}

func TestGenerateToken_EmptyRole(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	token, err := tm.GenerateToken(uuid.New(), "test@example.com", "")

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "organization role cannot be empty")
}

func TestGenerateToken_Expiration(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	userID := uuid.New()
	token, err := tm.GenerateToken(userID, "test@example.com", "OrganizationDeveloper")
	require.NoError(t, err)

	claims, err := tm.ValidateToken(token)
	require.NoError(t, err)

	// Check expiration is set to 24 hours from now
	expectedExpiry := time.Now().Add(TokenExpiration)
	actualExpiry := claims.ExpiresAt.Time

	// Allow 1 second tolerance
	timeDiff := actualExpiry.Sub(expectedExpiry).Abs()
	assert.Less(t, timeDiff, time.Second)
}

func TestValidateToken_Success(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	userID := uuid.New()
	email := "test@example.com"
	role := "OrganizationAdmin"

	token, err := tm.GenerateToken(userID, email, role)
	require.NoError(t, err)

	claims, err := tm.ValidateToken(token)

	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.OrganizationRole)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
}

func TestValidateToken_EmptyToken(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	claims, err := tm.ValidateToken("")

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "token string cannot be empty")
}

func TestValidateToken_InvalidToken(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	claims, err := tm.ValidateToken("invalid.token.string")

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	tm1, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	token, err := tm1.GenerateToken(uuid.New(), "test@example.com", "OrganizationDeveloper")
	require.NoError(t, err)

	// Try to validate with different secret
	tm2, err := NewTokenManager("different-secret-key-minimum-32-characters-long", 0)
	require.NoError(t, err)

	claims, err := tm2.ValidateToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	// Create a token that's already expired
	claims := Claims{
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
	tokenString, err := token.SignedString(tm.jwtSecret)
	require.NoError(t, err)

	// Validate expired token
	validatedClaims, err := tm.ValidateToken(tokenString)

	assert.Error(t, err)
	assert.Nil(t, validatedClaims)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestValidateToken_MissingUserID(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	// Create token with missing user ID
	claims := Claims{
		UserID:           uuid.Nil, // Missing
		Email:            "test@example.com",
		OrganizationRole: "OrganizationDeveloper",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(tm.jwtSecret)
	require.NoError(t, err)

	validatedClaims, err := tm.ValidateToken(tokenString)

	assert.Error(t, err)
	assert.Nil(t, validatedClaims)
	assert.ErrorIs(t, err, ErrInvalidTokenClaims)
	assert.Contains(t, err.Error(), "missing user ID")
}

func TestValidateToken_MissingEmail(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	// Create token with missing email
	claims := Claims{
		UserID:           uuid.New(),
		Email:            "", // Missing
		OrganizationRole: "OrganizationDeveloper",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(tm.jwtSecret)
	require.NoError(t, err)

	validatedClaims, err := tm.ValidateToken(tokenString)

	assert.Error(t, err)
	assert.Nil(t, validatedClaims)
	assert.ErrorIs(t, err, ErrInvalidTokenClaims)
	assert.Contains(t, err.Error(), "missing email")
}

func TestValidateToken_MissingRole(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	// Create token with missing role
	claims := Claims{
		UserID:           uuid.New(),
		Email:            "test@example.com",
		OrganizationRole: "", // Missing
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(tm.jwtSecret)
	require.NoError(t, err)

	validatedClaims, err := tm.ValidateToken(tokenString)

	assert.Error(t, err)
	assert.Nil(t, validatedClaims)
	assert.ErrorIs(t, err, ErrInvalidTokenClaims)
	assert.Contains(t, err.Error(), "missing organization role")
}

func TestGenerateAndValidateToken_DifferentRoles(t *testing.T) {
	tm, err := NewTokenManager(testJWTSecret, 0)
	require.NoError(t, err)

	roles := []string{
		"OrganizationAdmin",
		"OrganizationDBA",
		"OrganizationDeveloper",
	}

	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			userID := uuid.New()
			token, err := tm.GenerateToken(userID, "test@example.com", role)
			require.NoError(t, err)

			claims, err := tm.ValidateToken(token)
			require.NoError(t, err)
			assert.Equal(t, role, claims.OrganizationRole)
		})
	}
}

func BenchmarkGenerateToken(b *testing.B) {
	tm, _ := NewTokenManager(testJWTSecret, 0)
	userID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tm.GenerateToken(userID, "test@example.com", "OrganizationDeveloper")
	}
}

func BenchmarkValidateToken(b *testing.B) {
	tm, _ := NewTokenManager(testJWTSecret, 0)
	token, _ := tm.GenerateToken(uuid.New(), "test@example.com", "OrganizationDeveloper")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tm.ValidateToken(token)
	}
}
