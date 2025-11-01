package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword_Success(t *testing.T) {
	password := "securePassword123"

	hash, err := HashPassword(password, 0)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Verify the hash starts with bcrypt identifier
	assert.True(t, strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$"))

	// Verify the hash can be used to verify the password
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	assert.NoError(t, err)
}

func TestHashPassword_EmptyPassword(t *testing.T) {
	hash, err := HashPassword("", 0)

	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPassword, err)
	assert.Empty(t, hash)
}

func TestHashPassword_TooShort(t *testing.T) {
	password := "short"

	hash, err := HashPassword(password, 0)

	assert.Error(t, err)
	assert.Equal(t, ErrPasswordTooShort, err)
	assert.Empty(t, hash)
}

func TestHashPassword_MinimumLength(t *testing.T) {
	password := "12345678" // Exactly 8 characters

	hash, err := HashPassword(password, 0)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

func TestHashPassword_DifferentHashesForSamePassword(t *testing.T) {
	password := "samePassword123"

	hash1, err1 := HashPassword(password, 0)
	hash2, err2 := HashPassword(password, 0)

	require.NoError(t, err1)
	require.NoError(t, err2)

	// Hashes should be different due to random salt
	assert.NotEqual(t, hash1, hash2)

	// But both should verify against the same password
	err := bcrypt.CompareHashAndPassword([]byte(hash1), []byte(password))
	assert.NoError(t, err)

	err = bcrypt.CompareHashAndPassword([]byte(hash2), []byte(password))
	assert.NoError(t, err)
}

func TestHashPassword_CorrectCost(t *testing.T) {
	password := "testPassword123"

	hash, err := HashPassword(password, 0)
	require.NoError(t, err)

	// Extract cost from hash
	cost, err := bcrypt.Cost([]byte(hash))
	require.NoError(t, err)

	assert.Equal(t, DefaultBcryptCost, cost)
	assert.GreaterOrEqual(t, cost, 10, "Bcrypt cost should be at least 10")
}

func TestVerifyPassword_Success(t *testing.T) {
	password := "correctPassword123"

	hash, err := HashPassword(password, 0)
	require.NoError(t, err)

	err = VerifyPassword(hash, password)
	assert.NoError(t, err)
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	password := "correctPassword123"
	wrongPassword := "wrongPassword123"

	hash, err := HashPassword(password, 0)
	require.NoError(t, err)

	err = VerifyPassword(hash, wrongPassword)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidPassword, err)
}

func TestVerifyPassword_EmptyPassword(t *testing.T) {
	hash, err := HashPassword("validPassword123", 0)
	require.NoError(t, err)

	err = VerifyPassword(hash, "")
	assert.Error(t, err)
	assert.Equal(t, ErrEmptyPassword, err)
}

func TestVerifyPassword_EmptyHash(t *testing.T) {
	err := VerifyPassword("", "somePassword123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hashed password cannot be empty")
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	err := VerifyPassword("invalid-hash-format", "somePassword123")
	assert.Error(t, err)
	assert.NotEqual(t, ErrInvalidPassword, err)
}

func TestVerifyPassword_CaseSensitive(t *testing.T) {
	password := "CaseSensitive123"

	hash, err := HashPassword(password, 0)
	require.NoError(t, err)

	// Correct case should work
	err = VerifyPassword(hash, password)
	assert.NoError(t, err)

	// Wrong case should fail
	err = VerifyPassword(hash, "casesensitive123")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidPassword, err)

	err = VerifyPassword(hash, "CASESENSITIVE123")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidPassword, err)
}

func TestVerifyPassword_SpecialCharacters(t *testing.T) {
	passwords := []string{
		"P@ssw0rd!",
		"Test#123$%^",
		"Spëciàl€Chars",
		"密码测试12345678",
	}

	for _, password := range passwords {
		t.Run(password, func(t *testing.T) {
			hash, err := HashPassword(password, 0)
			require.NoError(t, err)

			err = VerifyPassword(hash, password)
			assert.NoError(t, err)
		})
	}
}

func TestHashPassword_LongPassword(t *testing.T) {
	// Test with a password at bcrypt's 72-byte limit
	longPassword := strings.Repeat("a", 72)

	hash, err := HashPassword(longPassword, 0)
	require.NoError(t, err)

	// Should verify successfully
	err = VerifyPassword(hash, longPassword)
	assert.NoError(t, err)
}

func TestHashPassword_ExceedsBcryptLimit(t *testing.T) {
	// Test with a password exceeding bcrypt's 72-byte limit
	tooLongPassword := strings.Repeat("a", 100)

	hash, err := HashPassword(tooLongPassword, 0)

	// Should return ErrPasswordTooLong for passwords > 72 bytes
	assert.Error(t, err)
	assert.Equal(t, ErrPasswordTooLong, err)
	assert.Empty(t, hash)
}

func TestHashPassword_MaximumLength(t *testing.T) {
	// Test with a password at exactly 72 bytes (maximum allowed)
	maxPassword := strings.Repeat("a", MaxPasswordLength)

	hash, err := HashPassword(maxPassword, 0)

	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Should verify successfully
	err = VerifyPassword(hash, maxPassword)
	assert.NoError(t, err)
}

func TestHashPassword_ExceedsMaximumByOne(t *testing.T) {
	// Test with a password one character over the limit
	tooLongPassword := strings.Repeat("a", MaxPasswordLength+1)

	hash, err := HashPassword(tooLongPassword, 0)

	assert.Error(t, err)
	assert.Equal(t, ErrPasswordTooLong, err)
	assert.Empty(t, hash)
}

func BenchmarkHashPassword(b *testing.B) {
	password := "benchmarkPassword123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword(password, 0)
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := "benchmarkPassword123"
	hash, _ := HashPassword(password, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = VerifyPassword(hash, password)
	}
}
