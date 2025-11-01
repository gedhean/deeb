package utils

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecryptPassword(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012" // 32 bytes

	tests := []struct {
		name           string
		encrypted      *string
		encryptionKey  string
		wantErr        bool
		wantPassword   string
	}{
		{
			name:          "nil encrypted password",
			encrypted:     nil,
			encryptionKey: encryptionKey,
			wantErr:       false,
			wantPassword:  "",
		},
		{
			name:          "empty encrypted password",
			encrypted:     stringPtr(""),
			encryptionKey: encryptionKey,
			wantErr:       false,
			wantPassword:  "",
		},
		{
			name:          "valid encrypted password",
			encrypted:     encryptPasswordForTest(t, encryptionKey, "testpass123"),
			encryptionKey: encryptionKey,
			wantErr:       false,
			wantPassword:  "testpass123",
		},
		{
			name:          "invalid encrypted password",
			encrypted:     stringPtr("invalid_base64"),
			encryptionKey: encryptionKey,
			wantErr:       true,
		},
		{
			name:          "wrong encryption key",
			encrypted:     encryptPasswordForTest(t, encryptionKey, "testpass123"),
			encryptionKey: "wrong_key_12345678901234567890",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := decryptPassword(tt.encrypted, tt.encryptionKey)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPassword, password)
		})
	}
}

func TestTestMySQLConnection_InvalidParams(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		params ConnectionTestParams
		wantErr bool
	}{
		{
			name: "missing host",
			params: ConnectionTestParams{
				DatabaseType: "mysql",
				Port:         3306,
				Username:     "test",
				Password:     "pass",
				Database:     "testdb",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			params: ConnectionTestParams{
				DatabaseType: "mysql",
				Host:         "localhost",
				Port:         0,
				Username:     "test",
				Password:     "pass",
				Database:     "testdb",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			params: ConnectionTestParams{
				DatabaseType: "mysql",
				Host:         "localhost",
				Port:         3306,
				Password:     "pass",
				Database:     "testdb",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TestMySQLConnection(ctx, tt.params)
			if tt.wantErr {
				assert.Error(t, err)
			}
		})
	}
}

func TestTestPostgreSQLConnection_InvalidParams(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		params ConnectionTestParams
		wantErr bool
	}{
		{
			name: "missing host",
			params: ConnectionTestParams{
				DatabaseType: "postgresql",
				Port:         5432,
				Username:     "test",
				Password:     "pass",
				Database:     "testdb",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			params: ConnectionTestParams{
				DatabaseType: "postgresql",
				Host:         "localhost",
				Port:         0,
				Username:     "test",
				Password:     "pass",
				Database:     "testdb",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			params: ConnectionTestParams{
				DatabaseType: "postgresql",
				Host:         "localhost",
				Port:         5432,
				Password:     "pass",
				Database:     "testdb",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TestPostgreSQLConnection(ctx, tt.params)
			if tt.wantErr {
				assert.Error(t, err)
			}
		})
	}
}

func TestTestMySQLConnection_Timeout(t *testing.T) {
	ctx := context.Background()

	// Test with invalid host that will timeout
	params := ConnectionTestParams{
		DatabaseType:             "mysql",
		Host:                     "192.0.2.0", // Non-routable IP (RFC 3330)
		Port:                     3306,
		Username:                 "test",
		Password:                 "pass",
		Database:                 "testdb",
		ConnectionTimeoutSeconds: 1, // Very short timeout
	}

	err := TestMySQLConnection(ctx, params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestTestPostgreSQLConnection_Timeout(t *testing.T) {
	ctx := context.Background()

	// Test with invalid host that will timeout
	params := ConnectionTestParams{
		DatabaseType:             "postgresql",
		Host:                     "192.0.2.0", // Non-routable IP (RFC 3330)
		Port:                     5432,
		Username:                 "test",
		Password:                 "pass",
		Database:                 "testdb",
		ConnectionTimeoutSeconds: 1, // Very short timeout
	}

	err := TestPostgreSQLConnection(ctx, params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestTestMySQLConnection_EncryptedPassword(t *testing.T) {
	ctx := context.Background()
	encryptionKey := "12345678901234567890123456789012"
	encrypted := encryptPasswordForTest(t, encryptionKey, "testpass123")

	// Test with invalid host (will fail connection, but password decryption should work)
	params := ConnectionTestParams{
		DatabaseType:             "mysql",
		Host:                     "192.0.2.0",
		Port:                     3306,
		Username:                 "test",
		EncryptedPassword:       encrypted,
		EncryptionKey:            encryptionKey,
		Database:                 "testdb",
		ConnectionTimeoutSeconds: 1,
	}

	err := TestMySQLConnection(ctx, params)
	// Should get connection error, not decryption error
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "failed to decrypt")
	assert.NotContains(t, err.Error(), "failed to parse decrypted")
}

func TestTestPostgreSQLConnection_EncryptedPassword(t *testing.T) {
	ctx := context.Background()
	encryptionKey := "12345678901234567890123456789012"
	encrypted := encryptPasswordForTest(t, encryptionKey, "testpass123")

	// Test with invalid host (will fail connection, but password decryption should work)
	params := ConnectionTestParams{
		DatabaseType:             "postgresql",
		Host:                     "192.0.2.0",
		Port:                     5432,
		Username:                 "test",
		EncryptedPassword:       encrypted,
		EncryptionKey:            encryptionKey,
		Database:                 "testdb",
		ConnectionTimeoutSeconds: 1,
	}

	err := TestPostgreSQLConnection(ctx, params)
	// Should get connection error, not decryption error
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "failed to decrypt")
	assert.NotContains(t, err.Error(), "failed to parse decrypted")
}

// Helper functions for tests

func stringPtr(s string) *string {
	return &s
}

func encryptPasswordForTest(t *testing.T, encryptionKey, password string) *string {
	t.Helper()

	// Create connection details JSON
	details := connectionDetails{
		Password: password,
	}

	jsonData, err := json.Marshal(details)
	require.NoError(t, err)

	encrypted, err := Encrypt(string(jsonData), encryptionKey)
	require.NoError(t, err)

	return &encrypted
}

