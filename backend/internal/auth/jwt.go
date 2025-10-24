package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// TokenExpiration is the default token expiration duration (24 hours)
	TokenExpiration = 24 * time.Hour
)

var (
	// ErrInvalidToken is returned when token validation fails
	ErrInvalidToken = errors.New("invalid token")

	// ErrExpiredToken is returned when token has expired
	ErrExpiredToken = errors.New("token has expired")

	// ErrInvalidTokenClaims is returned when token claims are invalid
	ErrInvalidTokenClaims = errors.New("invalid token claims")
)

// Claims represents the JWT claims for a user session
// Note: Only organization-level role is stored in JWT for authentication.
// Project-specific roles are fetched from the database when accessing project resources.
type Claims struct {
	UserID           uuid.UUID `json:"user_id"`
	Email            string    `json:"email"`
	OrganizationRole string    `json:"organization_role"` // OrganizationAdmin, OrganizationDBA, OrganizationDeveloper
	jwt.RegisteredClaims
}

// TokenManager handles JWT token generation and validation
type TokenManager struct {
	jwtSecret []byte
}

// NewTokenManager creates a new TokenManager with the provided JWT secret
func NewTokenManager(jwtSecret string) (*TokenManager, error) {
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT secret cannot be empty")
	}

	if len(jwtSecret) < 32 {
		return nil, fmt.Errorf("JWT secret must be at least 32 characters")
	}

	return &TokenManager{
		jwtSecret: []byte(jwtSecret),
	}, nil
}

// GenerateToken creates a new JWT token for the given user with their organization role.
// Project-specific roles are not included in the token and should be fetched from the
// database when the user accesses project resources.
func (tm *TokenManager) GenerateToken(userID uuid.UUID, email, organizationRole string) (string, error) {
	// Validate inputs
	if userID == uuid.Nil {
		return "", fmt.Errorf("user ID cannot be nil")
	}

	if email == "" {
		return "", fmt.Errorf("email cannot be empty")
	}

	if organizationRole == "" {
		return "", fmt.Errorf("organization role cannot be empty")
	}

	// Create claims with expiration
	now := time.Now()
	claims := Claims{
		UserID:           userID,
		Email:            email,
		OrganizationRole: organizationRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString(tm.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (tm *TokenManager) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token string cannot be empty")
	}

	// Parse token with claims
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.jwtSecret, nil
	})
	if err != nil {
		// Check if error is due to expiration
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Extract claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidTokenClaims
	}

	// Validate required fields
	if claims.UserID == uuid.Nil {
		return nil, fmt.Errorf("%w: missing user ID", ErrInvalidTokenClaims)
	}

	if claims.Email == "" {
		return nil, fmt.Errorf("%w: missing email", ErrInvalidTokenClaims)
	}

	if claims.OrganizationRole == "" {
		return nil, fmt.Errorf("%w: missing organization role", ErrInvalidTokenClaims)
	}

	return claims, nil
}
