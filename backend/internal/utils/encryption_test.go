package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncrypt_Decrypt_Success(t *testing.T) {
	key := "12345678901234567890123456789012" // Exactly 32 bytes
	plaintext := "sensitive-credential-data"

	encrypted, err := Encrypt(plaintext, key)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_InvalidKey(t *testing.T) {
	plaintext := "test-data"

	// Key too short
	_, err := Encrypt(plaintext, "short")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidEncryptionKey, err)

	// Key too long
	_, err = Encrypt(plaintext, "123456789012345678901234567890123")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidEncryptionKey, err)
}

func TestDecrypt_InvalidKey(t *testing.T) {
	key := "12345678901234567890123456789012"
	plaintext := "test-data"

	encrypted, err := Encrypt(plaintext, key)
	require.NoError(t, err)

	// Wrong key
	_, err = Decrypt(encrypted, "09876543210987654321098765432109")
	assert.Error(t, err)

	// Invalid key length
	_, err = Decrypt(encrypted, "short")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidEncryptionKey, err)
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	key := "12345678901234567890123456789012"

	// Invalid base64
	_, err := Decrypt("invalid-base64!!!", key)
	assert.Error(t, err)

	// Too short ciphertext
	_, err = Decrypt("dGVzdA==", key) // "test" in base64
	assert.Error(t, err)
}

func TestEncrypt_Decrypt_BinaryData(t *testing.T) {
	key := "12345678901234567890123456789012"
	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, 0xFC}

	encrypted, err := Encrypt(string(binaryData), key)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, string(binaryData), decrypted)
}

func TestEncrypt_Decrypt_LargeData(t *testing.T) {
	key := "12345678901234567890123456789012"
	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	encrypted, err := Encrypt(string(largeData), key)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, string(largeData), decrypted)
}

func TestEncrypt_KeyByteSlice(t *testing.T) {
	key := "12345678901234567890123456789012"
	plaintext := "test-data"

	encrypted, err := Encrypt(plaintext, key)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_DifferentResultsForSameInput(t *testing.T) {
	key := "12345678901234567890123456789012"
	plaintext := "same-input"

	encrypted1, err1 := Encrypt(plaintext, key)
	require.NoError(t, err1)

	encrypted2, err2 := Encrypt(plaintext, key)
	require.NoError(t, err2)

	// Encrypted values should be different due to random nonce
	assert.NotEqual(t, encrypted1, encrypted2)

	// But both should decrypt to the same plaintext
	decrypted1, err1 := Decrypt(encrypted1, key)
	require.NoError(t, err1)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err2 := Decrypt(encrypted2, key)
	require.NoError(t, err2)
	assert.Equal(t, plaintext, decrypted2)
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	key := "12345678901234567890123456789012"

	encrypted, err := Encrypt("", key)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := Decrypt(encrypted, key)
	require.NoError(t, err)
	assert.Equal(t, "", decrypted)
}

