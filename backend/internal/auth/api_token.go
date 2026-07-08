package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// APITokenPrefix identifies an API token on the Authorization: Bearer header,
	// distinguishing it from a JWT.
	APITokenPrefix = "deeb_pat_"

	// apiTokenRandomBytes is the number of random bytes in the token secret.
	apiTokenRandomBytes = 32

	// apiTokenDisplayLen is how many characters of the random part are kept in the
	// stored, human-readable prefix (for identifying a token without revealing it).
	apiTokenDisplayLen = 8
)

// GenerateAPIToken creates a new API token. It returns the plaintext (shown to the
// user exactly once), the SHA-256 hex hash to persist, and a short display prefix.
func GenerateAPIToken() (plaintext, hash, prefix string, err error) {
	b := make([]byte, apiTokenRandomBytes)
	if _, readErr := rand.Read(b); readErr != nil {
		return "", "", "", fmt.Errorf("failed to generate api token: %w", readErr)
	}

	randomPart := base64.RawURLEncoding.EncodeToString(b)
	plaintext = APITokenPrefix + randomPart
	hash = HashAPIToken(plaintext)
	prefix = APITokenPrefix + randomPart[:apiTokenDisplayLen]

	return plaintext, hash, prefix, nil
}

// HashAPIToken returns the SHA-256 hex digest of a plaintext API token. A fast hash
// is appropriate here because the token is a 256-bit random secret, and it allows an
// indexed equality lookup by hash.
func HashAPIToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// IsAPIToken reports whether a bearer value is an API token (by prefix) rather than a JWT.
func IsAPIToken(token string) bool {
	return strings.HasPrefix(token, APITokenPrefix)
}
