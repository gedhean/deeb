package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAPIToken(t *testing.T) {
	plaintext, hash, prefix, err := GenerateAPIToken()
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(plaintext, APITokenPrefix), "plaintext must carry the prefix")
	assert.True(t, strings.HasPrefix(prefix, APITokenPrefix), "display prefix must carry the prefix")
	assert.Equal(t, HashAPIToken(plaintext), hash, "returned hash must match HashAPIToken of plaintext")
	assert.Len(t, hash, 64, "sha256 hex is 64 chars")
	assert.NotEqual(t, plaintext, hash, "hash must not equal plaintext")
}

func TestGenerateAPIToken_Unique(t *testing.T) {
	p1, h1, _, err := GenerateAPIToken()
	require.NoError(t, err)
	p2, h2, _, err := GenerateAPIToken()
	require.NoError(t, err)

	assert.NotEqual(t, p1, p2, "tokens must be unique")
	assert.NotEqual(t, h1, h2, "hashes must be unique")
}

func TestHashAPIToken_Deterministic(t *testing.T) {
	assert.Equal(t, HashAPIToken("deeb_pat_abc"), HashAPIToken("deeb_pat_abc"))
	assert.NotEqual(t, HashAPIToken("deeb_pat_abc"), HashAPIToken("deeb_pat_xyz"))
}

func TestIsAPIToken(t *testing.T) {
	assert.True(t, IsAPIToken("deeb_pat_something"))
	assert.False(t, IsAPIToken("eyJhbGciOi.jwt.token"))
	assert.False(t, IsAPIToken(""))
}
