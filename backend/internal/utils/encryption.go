// Package utils provides utility functions for the Deeb application.
package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidEncryptionKey is returned when encryption key is invalid
	ErrInvalidEncryptionKey = errors.New("encryption key must be exactly 32 bytes")
	// ErrInvalidCiphertext is returned when ciphertext cannot be decrypted
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

// Encrypt encrypts plaintext using AES-256-GCM encryption with the provided key.
// The key must be exactly 32 bytes. Returns base64-encoded ciphertext.
func Encrypt(plaintext string, key string) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidEncryptionKey
	}

	// Convert string key to byte slice
	keyBytes := []byte(key)

	// Create AES cipher
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM mode: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM decryption with the provided key.
// The key must be exactly 32 bytes. Returns plaintext.
func Decrypt(ciphertext string, key string) (string, error) {
	if len(key) != 32 {
		return "", ErrInvalidEncryptionKey
	}

	// Decode from base64
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: invalid base64 encoding", ErrInvalidCiphertext)
	}

	// Convert string key to byte slice
	keyBytes := []byte(key)

	// Create AES cipher
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM mode: %w", err)
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertextBytes) < nonceSize {
		return "", fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}

	nonce, ciphertextBytes := ciphertextBytes[:nonceSize], ciphertextBytes[nonceSize:]

	// Decrypt ciphertext
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidCiphertext, err)
	}

	return string(plaintext), nil
}
