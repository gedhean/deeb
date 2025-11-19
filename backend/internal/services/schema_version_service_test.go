package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockSchemaHistoryRepository, MockDatabaseInstanceRepository and MockDatabaseRepository are defined in execution_service_test.go

func TestSchemaVersionService_GetLatestSchemaVersion(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		mockSchemaHistoryRepo := new(MockSchemaHistoryRepository)
		mockDBInstanceRepo := new(MockDatabaseInstanceRepository)
		mockDatabaseRepo := new(MockDatabaseRepository)

		service := NewSchemaVersionService(
			mockSchemaHistoryRepo,
			mockDBInstanceRepo,
			mockDatabaseRepo,
			nil, // connectionMgr not needed for this test
			"test-encryption-key",
			logger,
		)

		databaseID := uuid.New()
		expectedHistory := &models.SchemaHistory{
			ID:                 uuid.New(),
			DatabaseInstanceID: uuid.New(),
			DatabaseID:         databaseID,
			Metadata:           json.RawMessage(`{"database_name":"test_db"}`),
			RawDump:            "CREATE TABLE users (id SERIAL PRIMARY KEY);",
		}

		mockSchemaHistoryRepo.On("GetLatestByDatabaseID", ctx, databaseID).
			Return(expectedHistory, nil)

		result, err := service.GetLatestSchemaVersion(ctx, databaseID)
		require.NoError(t, err)
		assert.Equal(t, expectedHistory.ID, result.ID)
		assert.Equal(t, expectedHistory.RawDump, result.RawDump)

		mockSchemaHistoryRepo.AssertExpectations(t)
	})

	t.Run("no schema history exists", func(t *testing.T) {
		mockSchemaHistoryRepo := new(MockSchemaHistoryRepository)
		mockDBInstanceRepo := new(MockDatabaseInstanceRepository)
		mockDatabaseRepo := new(MockDatabaseRepository)

		service := NewSchemaVersionService(
			mockSchemaHistoryRepo,
			mockDBInstanceRepo,
			mockDatabaseRepo,
			nil,
			"test-encryption-key",
			logger,
		)

		databaseID := uuid.New()

		mockSchemaHistoryRepo.On("GetLatestByDatabaseID", ctx, databaseID).
			Return(nil, repository.ErrNotFound)

		result, err := service.GetLatestSchemaVersion(ctx, databaseID)
		require.NoError(t, err)
		assert.Nil(t, result)

		mockSchemaHistoryRepo.AssertExpectations(t)
	})

	t.Run("repository error", func(t *testing.T) {
		mockSchemaHistoryRepo := new(MockSchemaHistoryRepository)
		mockDBInstanceRepo := new(MockDatabaseInstanceRepository)
		mockDatabaseRepo := new(MockDatabaseRepository)

		service := NewSchemaVersionService(
			mockSchemaHistoryRepo,
			mockDBInstanceRepo,
			mockDatabaseRepo,
			nil,
			"test-encryption-key",
			logger,
		)

		databaseID := uuid.New()

		mockSchemaHistoryRepo.On("GetLatestByDatabaseID", ctx, databaseID).
			Return(nil, assert.AnError)

		result, err := service.GetLatestSchemaVersion(ctx, databaseID)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get latest schema version")

		mockSchemaHistoryRepo.AssertExpectations(t)
	})
}

func TestSchemaVersionService_CaptureSchemaVersion_DatabaseNotFound(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	mockSchemaHistoryRepo := new(MockSchemaHistoryRepository)
	mockDBInstanceRepo := new(MockDatabaseInstanceRepository)
	mockDatabaseRepo := new(MockDatabaseRepository)

	service := NewSchemaVersionService(
		mockSchemaHistoryRepo,
		mockDBInstanceRepo,
		mockDatabaseRepo,
		nil,
		"test-encryption-key",
		logger,
	)

	databaseID := uuid.New()

	mockDatabaseRepo.On("GetByID", ctx, databaseID).
		Return(nil, repository.ErrNotFound)

	result, err := service.CaptureSchemaVersion(ctx, databaseID)
	require.Error(t, err)
	assert.Equal(t, ErrDatabaseNotFound, err)
	assert.Nil(t, result)

	mockDatabaseRepo.AssertExpectations(t)
}

func TestSchemaVersionService_CaptureSchemaVersion_DatabaseInstanceNotFound(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	mockSchemaHistoryRepo := new(MockSchemaHistoryRepository)
	mockDBInstanceRepo := new(MockDatabaseInstanceRepository)
	mockDatabaseRepo := new(MockDatabaseRepository)

	service := NewSchemaVersionService(
		mockSchemaHistoryRepo,
		mockDBInstanceRepo,
		mockDatabaseRepo,
		nil,
		"test-encryption-key",
		logger,
	)

	databaseID := uuid.New()
	dbInstanceID := uuid.New()

	database := &models.Database{
		ID:                 databaseID,
		DatabaseInstanceID: dbInstanceID,
		DatabaseName:       "test_db",
	}

	mockDatabaseRepo.On("GetByID", ctx, databaseID).
		Return(database, nil)

	mockDBInstanceRepo.On("GetByID", ctx, dbInstanceID).
		Return(nil, repository.ErrNotFound)

	result, err := service.CaptureSchemaVersion(ctx, databaseID)
	require.Error(t, err)
	assert.Equal(t, ErrDatabaseInstanceNotFound, err)
	assert.Nil(t, result)

	mockDatabaseRepo.AssertExpectations(t)
	mockDBInstanceRepo.AssertExpectations(t)
}

func TestSchemaVersionService_CaptureSchemaVersion_UnsupportedEngine(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	mockSchemaHistoryRepo := new(MockSchemaHistoryRepository)
	mockDBInstanceRepo := new(MockDatabaseInstanceRepository)
	mockDatabaseRepo := new(MockDatabaseRepository)

	service := NewSchemaVersionService(
		mockSchemaHistoryRepo,
		mockDBInstanceRepo,
		mockDatabaseRepo,
		nil,
		"test-encryption-key",
		logger,
	)

	databaseID := uuid.New()
	dbInstanceID := uuid.New()

	database := &models.Database{
		ID:                 databaseID,
		DatabaseInstanceID: dbInstanceID,
		DatabaseName:       "test_db",
	}

	dbInstance := &models.DatabaseInstance{
		ID:             dbInstanceID,
		DatabaseEngine: "unsupported_engine",
		Host:           "localhost",
		Port:           5432,
	}

	mockDatabaseRepo.On("GetByID", ctx, databaseID).
		Return(database, nil)

	mockDBInstanceRepo.On("GetByID", ctx, dbInstanceID).
		Return(dbInstance, nil)

	result, err := service.CaptureSchemaVersion(ctx, databaseID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database engine")
	assert.Nil(t, result)

	mockDatabaseRepo.AssertExpectations(t)
	mockDBInstanceRepo.AssertExpectations(t)
}
