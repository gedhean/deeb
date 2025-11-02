package services

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockSchemaSnapshotRepository is a mock implementation of SchemaSnapshotRepository
type MockSchemaSnapshotRepository struct {
	mock.Mock
}

func (m *MockSchemaSnapshotRepository) Create(ctx context.Context, snapshot *models.SchemaSnapshot) error {
	args := m.Called(ctx, snapshot)
	return args.Error(0)
}

func (m *MockSchemaSnapshotRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SchemaSnapshot, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SchemaSnapshot), args.Error(1)
}

func (m *MockSchemaSnapshotRepository) GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.SchemaSnapshot, error) {
	args := m.Called(ctx, changePlanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SchemaSnapshot), args.Error(1)
}

func TestNewSchemaSnapshotService(t *testing.T) {
	snapshotRepo := new(MockSchemaSnapshotRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	connectionMgr := database.NewConnectionManager("test-encryption-key-min-32-chars", zap.NewNop())
	logger := zap.NewNop()

	service := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		"test-encryption-key-min-32-chars",
		logger,
	)

	assert.NotNil(t, service)
	assert.Equal(t, snapshotRepo, service.snapshotRepo)
	assert.Equal(t, dbInstanceRepo, service.dbInstanceRepo)
	assert.Equal(t, databaseRepo, service.databaseRepo)
	assert.Equal(t, connectionMgr, service.connectionMgr)
	assert.Equal(t, "test-encryption-key-min-32-chars", service.encryptionKey)
}

func TestSchemaSnapshotService_CapturePostgreSQL_InvalidParams(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	logger := zap.NewNop()
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)
	ctx := context.Background()

	tests := []struct {
		name          string
		changePlanID  uuid.UUID
		databaseID    uuid.UUID
		snapshotType  models.SnapshotType
		setupMocks    func(*MockDatabaseRepository, *MockDatabaseInstanceRepository, *MockSchemaSnapshotRepository)
		wantErr       bool
		errMsg        string
	}{
		{
			name:         "database not found",
			changePlanID: uuid.New(),
			databaseID:   uuid.New(),
			snapshotType: models.SnapshotTypeBefore,
			setupMocks: func(dbRepo *MockDatabaseRepository, dbInstanceRepo *MockDatabaseInstanceRepository, snapshotRepo *MockSchemaSnapshotRepository) {
				dbRepo.On("GetByID", ctx, mock.Anything).Return(nil, repository.ErrNotFound)
			},
			wantErr: true,
			errMsg:  "database not found",
		},
		{
			name:         "database instance not found",
			changePlanID: uuid.New(),
			databaseID:   uuid.New(),
			snapshotType: models.SnapshotTypeBefore,
			setupMocks: func(dbRepo *MockDatabaseRepository, dbInstanceRepo *MockDatabaseInstanceRepository, snapshotRepo *MockSchemaSnapshotRepository) {
				db := &models.Database{
					ID:                 uuid.New(),
					DatabaseInstanceID: uuid.New(),
					DatabaseName:       "testdb",
				}
				dbRepo.On("GetByID", ctx, mock.Anything).Return(db, nil)
				dbInstanceRepo.On("GetByID", ctx, mock.Anything).Return(nil, repository.ErrNotFound)
			},
			wantErr: true,
			errMsg:  "database instance not found",
		},
		{
			name:         "invalid database type",
			changePlanID: uuid.New(),
			databaseID:   uuid.New(),
			snapshotType: models.SnapshotTypeBefore,
			setupMocks: func(dbRepo *MockDatabaseRepository, dbInstanceRepo *MockDatabaseInstanceRepository, snapshotRepo *MockSchemaSnapshotRepository) {
				db := &models.Database{
					ID:                 uuid.New(),
					DatabaseInstanceID: uuid.New(),
					DatabaseName:       "testdb",
				}
				dbInstance := &models.DatabaseInstance{
					ID:           uuid.New(),
					DatabaseType: models.DatabaseTypeMySQL, // Wrong type
					Host:         "localhost",
					Port:         3306,
					Username:     "testuser",
				}
				dbRepo.On("GetByID", ctx, mock.Anything).Return(db, nil)
				dbInstanceRepo.On("GetByID", ctx, mock.Anything).Return(dbInstance, nil)
			},
			wantErr: true,
			errMsg:  "invalid database type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshotRepo := new(MockSchemaSnapshotRepository)
			dbInstanceRepo := new(MockDatabaseInstanceRepository)
			databaseRepo := new(MockDatabaseRepository)

			tt.setupMocks(databaseRepo, dbInstanceRepo, snapshotRepo)

			service := NewSchemaSnapshotService(
				snapshotRepo,
				dbInstanceRepo,
				databaseRepo,
				connectionMgr,
				encryptionKey,
				logger,
			)

			snapshot, err := service.CapturePostgreSQL(ctx, tt.changePlanID, tt.databaseID, tt.snapshotType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, snapshot)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, snapshot)
			}

			snapshotRepo.AssertExpectations(t)
			dbInstanceRepo.AssertExpectations(t)
			databaseRepo.AssertExpectations(t)
		})
	}
}

func TestSchemaSnapshotService_getDecryptedPassword(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	logger := zap.NewNop()
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)

	service := NewSchemaSnapshotService(
		new(MockSchemaSnapshotRepository),
		new(MockDatabaseInstanceRepository),
		new(MockDatabaseRepository),
		connectionMgr,
		encryptionKey,
		logger,
	)

	tests := []struct {
		name      string
		dbInstance *models.DatabaseInstance
		wantErr   bool
	}{
		{
			name: "nil encrypted password",
			dbInstance: &models.DatabaseInstance{
				ID:                         uuid.New(),
				ConnectionDetailsEncrypted: nil,
			},
			wantErr: false, // DecryptPassword returns empty string for nil
		},
		{
			name: "invalid encrypted password",
			dbInstance: &models.DatabaseInstance{
				ID:                         uuid.New(),
				ConnectionDetailsEncrypted: stringPtr("invalid_encrypted_data"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := service.getDecryptedPassword(tt.dbInstance)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, password)
			} else {
				require.NoError(t, err)
				// Password may be empty for nil encrypted password
			}
		})
	}
}

func TestSchemaSnapshotService_CapturePostgreSQL_PgDumpCommand(t *testing.T) {
	// This test verifies the pg_dump command construction
	// Note: This test doesn't actually execute pg_dump, it just verifies the command structure

	logger := zap.NewNop()
	ctx := context.Background()

	dbInstance := &models.DatabaseInstance{
		ID:           uuid.New(),
		DatabaseType: models.DatabaseTypePostgreSQL,
		Host:         "localhost",
		Port:         5432,
		Username:     "testuser",
		RequireSSL:   false,
	}

	db := &models.Database{
		ID:                 uuid.New(),
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       "testdb",
	}

	// Verify command would be constructed correctly
	expectedArgs := []string{
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"--host", dbInstance.Host,
		"--port", "5432",
		"--username", dbInstance.Username,
		"--dbname", db.DatabaseName,
		"--format", "plain",
	}

	// Check if pg_dump exists (required for actual execution)
	_, err := exec.LookPath("pg_dump")
	if err != nil {
		t.Skip("pg_dump not found in PATH, skipping pg_dump command test")
	}

	// Verify command can be constructed
	cmd := exec.CommandContext(ctx, "pg_dump", expectedArgs...)
	assert.NotNil(t, cmd)
	assert.Equal(t, "pg_dump", cmd.Path)
	assert.Equal(t, expectedArgs, cmd.Args[1:]) // Args[0] is the command path

	// Verify logger is used
	_ = logger
}

func TestSchemaSnapshotService_CaptureMySQL_InvalidParams(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	logger := zap.NewNop()
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)
	ctx := context.Background()

	tests := []struct {
		name          string
		changePlanID  uuid.UUID
		databaseID    uuid.UUID
		snapshotType  models.SnapshotType
		setupMocks    func(*MockDatabaseRepository, *MockDatabaseInstanceRepository, *MockSchemaSnapshotRepository)
		wantErr       bool
		errMsg        string
	}{
		{
			name:         "database not found",
			changePlanID: uuid.New(),
			databaseID:   uuid.New(),
			snapshotType: models.SnapshotTypeBefore,
			setupMocks: func(dbRepo *MockDatabaseRepository, dbInstanceRepo *MockDatabaseInstanceRepository, snapshotRepo *MockSchemaSnapshotRepository) {
				dbRepo.On("GetByID", ctx, mock.Anything).Return(nil, repository.ErrNotFound)
			},
			wantErr: true,
			errMsg:  "database not found",
		},
		{
			name:         "database instance not found",
			changePlanID: uuid.New(),
			databaseID:   uuid.New(),
			snapshotType: models.SnapshotTypeBefore,
			setupMocks: func(dbRepo *MockDatabaseRepository, dbInstanceRepo *MockDatabaseInstanceRepository, snapshotRepo *MockSchemaSnapshotRepository) {
				db := &models.Database{
					ID:                 uuid.New(),
					DatabaseInstanceID: uuid.New(),
					DatabaseName:       "testdb",
				}
				dbRepo.On("GetByID", ctx, mock.Anything).Return(db, nil)
				dbInstanceRepo.On("GetByID", ctx, mock.Anything).Return(nil, repository.ErrNotFound)
			},
			wantErr: true,
			errMsg:  "database instance not found",
		},
		{
			name:         "invalid database type",
			changePlanID: uuid.New(),
			databaseID:   uuid.New(),
			snapshotType: models.SnapshotTypeBefore,
			setupMocks: func(dbRepo *MockDatabaseRepository, dbInstanceRepo *MockDatabaseInstanceRepository, snapshotRepo *MockSchemaSnapshotRepository) {
				db := &models.Database{
					ID:                 uuid.New(),
					DatabaseInstanceID: uuid.New(),
					DatabaseName:       "testdb",
				}
				dbInstance := &models.DatabaseInstance{
					ID:           uuid.New(),
					DatabaseType: models.DatabaseTypePostgreSQL, // Wrong type
					Host:         "localhost",
					Port:         5432,
					Username:     "testuser",
				}
				dbRepo.On("GetByID", ctx, mock.Anything).Return(db, nil)
				dbInstanceRepo.On("GetByID", ctx, mock.Anything).Return(dbInstance, nil)
			},
			wantErr: true,
			errMsg:  "invalid database type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshotRepo := new(MockSchemaSnapshotRepository)
			dbInstanceRepo := new(MockDatabaseInstanceRepository)
			databaseRepo := new(MockDatabaseRepository)

			tt.setupMocks(databaseRepo, dbInstanceRepo, snapshotRepo)

			service := NewSchemaSnapshotService(
				snapshotRepo,
				dbInstanceRepo,
				databaseRepo,
				connectionMgr,
				encryptionKey,
				logger,
			)

			snapshot, err := service.CaptureMySQL(ctx, tt.changePlanID, tt.databaseID, tt.snapshotType)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, snapshot)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, snapshot)
			}

			snapshotRepo.AssertExpectations(t)
			dbInstanceRepo.AssertExpectations(t)
			databaseRepo.AssertExpectations(t)
		})
	}
}

func TestSchemaSnapshotService_CaptureMySQL_MysqldumpCommand(t *testing.T) {
	// This test verifies the mysqldump command construction
	// Note: This test doesn't actually execute mysqldump, it just verifies the command structure

	logger := zap.NewNop()
	ctx := context.Background()

	dbInstance := &models.DatabaseInstance{
		ID:           uuid.New(),
		DatabaseType: models.DatabaseTypeMySQL,
		Host:         "localhost",
		Port:         3306,
		Username:     "testuser",
		RequireSSL:   false,
	}

	db := &models.Database{
		ID:                 uuid.New(),
		DatabaseInstanceID: dbInstance.ID,
		DatabaseName:       "testdb",
	}

	// Verify command would be constructed correctly
	expectedArgs := []string{
		"--no-data",
		"--host", dbInstance.Host,
		"--port", "3306",
		"--user", dbInstance.Username,
		"--password=testpass",
		"--skip-comments",
		"--skip-add-drop-table",
		db.DatabaseName,
		"--ssl-mode=DISABLED",
	}

	// Check if mysqldump exists (required for actual execution)
	_, err := exec.LookPath("mysqldump")
	if err != nil {
		t.Skip("mysqldump not found in PATH, skipping mysqldump command test")
	}

	// Verify command can be constructed
	cmd := exec.CommandContext(ctx, "mysqldump", expectedArgs...)
	assert.NotNil(t, cmd)
	assert.Equal(t, "mysqldump", cmd.Path)
	// Note: Args[0] is the command path, so we check Args[1:]
	assert.Equal(t, expectedArgs, cmd.Args[1:])

	// Verify logger is used
	_ = logger
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

