package database

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewConnectionManager(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	logger := zap.NewNop()

	cm := NewConnectionManager(encryptionKey, logger)

	assert.NotNil(t, cm)
	assert.Equal(t, encryptionKey, cm.encryptionKey)
	assert.NotNil(t, cm.logger)
}

func TestConnectionManager_ConnectMySQL_InvalidParams(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	logger := zap.NewNop()
	cm := NewConnectionManager(encryptionKey, logger)
	ctx := context.Background()

	tests := []struct {
		name    string
		params  ConnectionParams
		wantErr bool
		errMsg  string
	}{
		{
			name: "nil database instance",
			params: ConnectionParams{
				DatabaseInstance: nil,
				Database: &models.Database{
					ID:           uuid.New(),
					DatabaseName: "testdb",
				},
			},
			wantErr: true,
			errMsg:  "database instance is required",
		},
		{
			name: "nil database",
			params: ConnectionParams{
				DatabaseInstance: &models.DatabaseInstance{
					ID:             uuid.New(),
					DatabaseEngine: models.DatabaseEngineMySQL,
					Host:           "localhost",
					Port:           3306,
					Username:       "testuser",
				},
				Database: nil,
			},
			wantErr: true,
			errMsg:  "database is required",
		},
		{
			name: "invalid database type",
			params: ConnectionParams{
				DatabaseInstance: &models.DatabaseInstance{
					ID:             uuid.New(),
					DatabaseEngine: models.DatabaseEnginePostgreSQL,
					Host:           "localhost",
					Port:           3306,
					Username:       "testuser",
				},
				Database: &models.Database{
					ID:           uuid.New(),
					DatabaseName: "testdb",
				},
			},
			wantErr: true,
			errMsg:  "invalid database type",
		},
		{
			name: "failed to decrypt password",
			params: ConnectionParams{
				DatabaseInstance: &models.DatabaseInstance{
					ID:                         uuid.New(),
					DatabaseEngine:             models.DatabaseEngineMySQL,
					Host:                       "localhost",
					Port:                       3306,
					Username:                   "testuser",
					ConnectionDetailsEncrypted: stringPtr("invalid_encrypted_data"),
				},
				Database: &models.Database{
					ID:           uuid.New(),
					DatabaseName: "testdb",
				},
			},
			wantErr: true,
			errMsg:  "failed to decrypt password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := cm.ConnectMySQL(ctx, tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, db)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, db)
				if db != nil {
					db.Close()
				}
			}
		})
	}
}

func TestConnectionManager_ConnectPostgreSQL_InvalidParams(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	logger := zap.NewNop()
	cm := NewConnectionManager(encryptionKey, logger)
	ctx := context.Background()

	tests := []struct {
		name    string
		params  ConnectionParams
		wantErr bool
		errMsg  string
	}{
		{
			name: "nil database instance",
			params: ConnectionParams{
				DatabaseInstance: nil,
				Database: &models.Database{
					ID:           uuid.New(),
					DatabaseName: "testdb",
				},
			},
			wantErr: true,
			errMsg:  "database instance is required",
		},
		{
			name: "nil database",
			params: ConnectionParams{
				DatabaseInstance: &models.DatabaseInstance{
					ID:             uuid.New(),
					DatabaseEngine: models.DatabaseEnginePostgreSQL,
					Host:           "localhost",
					Port:           5432,
					Username:       "testuser",
				},
				Database: nil,
			},
			wantErr: true,
			errMsg:  "database is required",
		},
		{
			name: "invalid database type",
			params: ConnectionParams{
				DatabaseInstance: &models.DatabaseInstance{
					ID:             uuid.New(),
					DatabaseEngine: models.DatabaseEngineMySQL,
					Host:           "localhost",
					Port:           5432,
					Username:       "testuser",
				},
				Database: &models.Database{
					ID:           uuid.New(),
					DatabaseName: "testdb",
				},
			},
			wantErr: true,
			errMsg:  "invalid database type",
		},
		{
			name: "failed to decrypt password",
			params: ConnectionParams{
				DatabaseInstance: &models.DatabaseInstance{
					ID:                         uuid.New(),
					DatabaseEngine:             models.DatabaseEnginePostgreSQL,
					Host:                       "localhost",
					Port:                       5432,
					Username:                   "testuser",
					ConnectionDetailsEncrypted: stringPtr("invalid_encrypted_data"),
				},
				Database: &models.Database{
					ID:           uuid.New(),
					DatabaseName: "testdb",
				},
			},
			wantErr: true,
			errMsg:  "failed to decrypt password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := cm.ConnectPostgreSQL(ctx, tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, db)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, db)
				if db != nil {
					db.Close()
				}
			}
		})
	}
}

func TestConnectionManager_Connect_UnsupportedType(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	logger := zap.NewNop()
	cm := NewConnectionManager(encryptionKey, logger)
	ctx := context.Background()

	params := ConnectionParams{
		DatabaseInstance: &models.DatabaseInstance{
			ID:             uuid.New(),
			DatabaseEngine: "invalid_type",
			Host:           "localhost",
			Port:           5432,
			Username:       "testuser",
		},
		Database: &models.Database{
			ID:           uuid.New(),
			DatabaseName: "testdb",
		},
	}

	db, err := cm.Connect(ctx, params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database type")
	assert.Nil(t, db)
}

func TestConnectionManager_ConnectionPoolConfig(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	logger := zap.NewNop()
	_ = NewConnectionManager(encryptionKey, logger) // Not used in this test

	// Test default values
	params := ConnectionParams{
		MaxOpenConns:    0, // Should default to 10
		MaxIdleConns:    0, // Should default to 5
		ConnMaxLifetime: 0, // Should default to 5 minutes
	}

	assert.Equal(t, 0, params.MaxOpenConns)
	assert.Equal(t, 0, params.MaxIdleConns)
	assert.Equal(t, time.Duration(0), params.ConnMaxLifetime)

	// Test custom values
	params.MaxOpenConns = 20
	params.MaxIdleConns = 10
	params.ConnMaxLifetime = 10 * time.Minute

	assert.Equal(t, 20, params.MaxOpenConns)
	assert.Equal(t, 10, params.MaxIdleConns)
	assert.Equal(t, 10*time.Minute, params.ConnMaxLifetime)
}

func TestConnectionManager_PasswordDecryption(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	logger := zap.NewNop()
	_ = NewConnectionManager(encryptionKey, logger) // Not used in this test

	// Test with nil encrypted password
	password, err := utils.DecryptPassword(nil, encryptionKey)
	require.NoError(t, err)
	assert.Equal(t, "", password)

	// Test with empty encrypted password
	emptyStr := ""
	password, err = utils.DecryptPassword(&emptyStr, encryptionKey)
	require.NoError(t, err)
	assert.Equal(t, "", password)

	// Test with valid encrypted password
	details := map[string]string{
		"password": "testpass123",
	}
	detailsJSON, err := json.Marshal(details)
	require.NoError(t, err)

	encrypted, err := utils.Encrypt(string(detailsJSON), encryptionKey)
	require.NoError(t, err)

	password, err = utils.DecryptPassword(&encrypted, encryptionKey)
	require.NoError(t, err)
	assert.Equal(t, "testpass123", password)
}

// Helper functions

func stringPtr(s string) *string {
	return &s
}

func encryptPasswordForTest(t *testing.T, encryptionKey, password string) *string {
	details := map[string]string{
		"password": password,
	}
	detailsJSON, err := json.Marshal(details)
	require.NoError(t, err)

	encrypted, err := utils.Encrypt(string(detailsJSON), encryptionKey)
	require.NoError(t, err)

	return &encrypted
}
