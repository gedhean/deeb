package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength is the minimum required password length
	MinPasswordLength = 8

	// MaxPasswordLength is the maximum password length (bcrypt has a 72-byte limit)
	MaxPasswordLength = 72

	// DefaultBcryptCost is the default work factor for bcrypt (12 = ~250ms on modern hardware)
	DefaultBcryptCost = 12
)

var (
	// ErrPasswordTooShort is returned when password is below minimum length
	ErrPasswordTooShort = errors.New("password must be at least 8 characters long")

	// ErrPasswordTooLong is returned when password exceeds maximum length
	ErrPasswordTooLong = errors.New("password must not exceed 72 characters")

	// ErrEmptyPassword is returned when password is empty
	ErrEmptyPassword = errors.New("password cannot be empty")

	// ErrInvalidPassword is returned when password verification fails
	ErrInvalidPassword = errors.New("invalid password")
)

// HashPassword generates a bcrypt hash of the password with the specified cost factor.
// If cost is 0, uses DefaultBcryptCost (12).
// Returns ErrPasswordTooShort, ErrPasswordTooLong, or ErrEmptyPassword for validation failures.
func HashPassword(password string, cost int) (string, error) {
	// Validate password
	if password == "" {
		return "", ErrEmptyPassword
	}

	if len(password) < MinPasswordLength {
		return "", ErrPasswordTooShort
	}

	if len(password) > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}

	// Use default cost if not specified
	if cost == 0 {
		cost = DefaultBcryptCost
	}

	// Validate cost range
	if cost < 10 || cost > 31 {
		return "", fmt.Errorf("bcrypt cost must be between 10 and 31, got %d", cost)
	}

	// Generate bcrypt hash
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// VerifyPassword compares a bcrypt hashed password with its possible plaintext equivalent.
// Returns nil if the password matches, ErrInvalidPassword if it does not match,
// or ErrEmptyPassword if the password parameter is empty.
func VerifyPassword(hashedPassword, password string) error {
	// Validate inputs
	if hashedPassword == "" {
		return fmt.Errorf("hashed password cannot be empty")
	}

	if password == "" {
		return ErrEmptyPassword
	}

	// Compare password with hash
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidPassword
		}
		return fmt.Errorf("failed to verify password: %w", err)
	}

	return nil
}
