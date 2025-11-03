package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockExecutionLockManager is a mock implementation of ExecutionLockManager
type MockExecutionLockManager struct {
	mock.Mock
}

func (m *MockExecutionLockManager) AcquireLock(ctx context.Context, databaseID uuid.UUID, changePlanID uuid.UUID, executorID uuid.UUID) error {
	args := m.Called(ctx, databaseID, changePlanID, executorID)
	return args.Error(0)
}

func (m *MockExecutionLockManager) ReleaseLock(ctx context.Context, databaseID uuid.UUID) error {
	args := m.Called(ctx, databaseID)
	return args.Error(0)
}

func (m *MockExecutionLockManager) IsLocked(ctx context.Context, databaseID uuid.UUID) (bool, *ExecutionLock, error) {
	args := m.Called(ctx, databaseID)
	if args.Get(0) == nil {
		return false, nil, args.Error(2)
	}
	if args.Get(0).(bool) {
		return true, args.Get(1).(*ExecutionLock), args.Error(2)
	}
	return false, nil, args.Error(2)
}

func (m *MockExecutionLockManager) CleanupExpiredLocks(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

// MockSchemaSnapshotService is a mock implementation of SchemaSnapshotService
type MockSchemaSnapshotService struct {
	mock.Mock
}

func (m *MockSchemaSnapshotService) CapturePostgreSQL(
	ctx context.Context,
	changePlanID uuid.UUID,
	databaseID uuid.UUID,
	snapshotType models.SnapshotType,
) (*models.SchemaSnapshot, error) {
	args := m.Called(ctx, changePlanID, databaseID, snapshotType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SchemaSnapshot), args.Error(1)
}

func (m *MockSchemaSnapshotService) CaptureMySQL(
	ctx context.Context,
	changePlanID uuid.UUID,
	databaseID uuid.UUID,
	snapshotType models.SnapshotType,
) (*models.SchemaSnapshot, error) {
	args := m.Called(ctx, changePlanID, databaseID, snapshotType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SchemaSnapshot), args.Error(1)
}

// MockConnectionManager is a mock implementation of ConnectionManager
type MockConnectionManager struct {
	mock.Mock
}

func (m *MockConnectionManager) Connect(ctx context.Context, params database.ConnectionParams) (*sqlx.DB, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqlx.DB), args.Error(1)
}

func TestNewExecutionService(t *testing.T) {
	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	snapshotRepo := new(MockSchemaSnapshotRepository)
	connectionMgr := database.NewConnectionManager("test-encryption-key-min-32-chars", zap.NewNop())
	snapshotService := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		"test-encryption-key-min-32-chars",
		zap.NewNop(),
	)
	logger := zap.NewNop()

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		snapshotService,
		&MockExecutionLockManager{},
		nil, // queueService - nil for this test
		nil, // executionLogRepo - nil for this test
		connectionMgr,
		"test-encryption-key-min-32-chars",
		logger,
	)

	assert.NotNil(t, service)
	assert.Equal(t, changePlanRepo, service.changePlanRepo)
	assert.Equal(t, dbInstanceRepo, service.dbInstanceRepo)
	assert.Equal(t, databaseRepo, service.databaseRepo)
	assert.Equal(t, snapshotService, service.snapshotService)
	assert.Equal(t, connectionMgr, service.connectionMgr)
	assert.Equal(t, "test-encryption-key-min-32-chars", service.encryptionKey)
}

func TestExecutionService_Execute_ChangePlanNotFound(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	logger := zap.NewNop()
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)
	ctx := context.Background()

	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	snapshotRepo := new(MockSchemaSnapshotRepository)
	snapshotService := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)

	changePlanID := uuid.New()
	executorID := uuid.New()
	changePlanRepo.On("GetByID", ctx, changePlanID).Return(nil, repository.ErrNotFound)

	lockManager := &MockExecutionLockManager{}

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		nil, // queueService - nil for this test
		nil, // executionLogRepo - nil for this test
		connectionMgr,
		encryptionKey,
		logger,
	)

	result, err := service.Execute(ctx, changePlanID, executorID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrChangePlanNotFound, err)

	changePlanRepo.AssertExpectations(t)
}

func TestExecutionService_executeInTransaction_Success(t *testing.T) {
	logger := zap.NewNop()

	// Create mock database
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	// Create mock ConnectionManager that returns our mocked database
	connectionMgr := &MockConnectionManager{}
	connectionMgr.On("Connect", mock.Anything, mock.Anything).Return(db, nil)

	service := &ExecutionService{
		connectionMgr: connectionMgr,
		logger:        logger,
	}

	ctx := context.Background()
	sqlContent := "CREATE TABLE test (id UUID PRIMARY KEY);"

	dbInstance := &models.DatabaseInstance{
		ID:           uuid.New(),
		DatabaseType: models.DatabaseTypePostgreSQL,
		Host:         "localhost",
		Port:         5432,
		Username:     "testuser",
	}
	dbModel := &models.Database{
		ID:                 uuid.New(),
		DatabaseName:       "testdb",
		DatabaseInstanceID: dbInstance.ID,
	}

	// Mock transaction begin with SERIALIZABLE isolation
	sqlMock.ExpectBegin()

	// Mock SQL execution
	sqlMock.ExpectExec("CREATE TABLE test").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock commit
	sqlMock.ExpectCommit()

	result, err := service.executeInTransaction(ctx, dbInstance, dbModel, sqlContent)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.RowsAffected)
	assert.Greater(t, result.Duration, time.Duration(0))

	require.NoError(t, sqlMock.ExpectationsWereMet())
	connectionMgr.AssertExpectations(t)
}

func TestExecutionService_executeInTransaction_WithRowsAffected(t *testing.T) {
	logger := zap.NewNop()

	// Create mock database
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	// Create mock ConnectionManager
	connectionMgr := &MockConnectionManager{}
	connectionMgr.On("Connect", mock.Anything, mock.Anything).Return(db, nil)

	service := &ExecutionService{
		connectionMgr: connectionMgr,
		logger:        logger,
	}

	ctx := context.Background()
	sqlContent := "INSERT INTO users (id, name) VALUES (gen_random_uuid(), 'test');"

	dbInstance := &models.DatabaseInstance{
		ID:           uuid.New(),
		DatabaseType: models.DatabaseTypePostgreSQL,
		Host:         "localhost",
		Port:         5432,
		Username:     "testuser",
	}
	dbModel := &models.Database{
		ID:                 uuid.New(),
		DatabaseName:       "testdb",
		DatabaseInstanceID: dbInstance.ID,
	}

	// Mock transaction begin
	sqlMock.ExpectBegin()

	// Mock SQL execution with rows affected
	sqlMock.ExpectExec("INSERT INTO users").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Mock commit
	sqlMock.ExpectCommit()

	result, err := service.executeInTransaction(ctx, dbInstance, dbModel, sqlContent)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 1, result.RowsAffected)
	assert.Greater(t, result.Duration, time.Duration(0))

	require.NoError(t, sqlMock.ExpectationsWereMet())
	connectionMgr.AssertExpectations(t)
}

func TestExecutionService_executeInTransaction_ExecutionError(t *testing.T) {
	logger := zap.NewNop()

	// Create mock database
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	// Create mock ConnectionManager
	connectionMgr := &MockConnectionManager{}
	connectionMgr.On("Connect", mock.Anything, mock.Anything).Return(db, nil)

	service := &ExecutionService{
		connectionMgr: connectionMgr,
		logger:        logger,
	}

	ctx := context.Background()
	sqlContent := "CREATE TABLE test (id UUID PRIMARY KEY);"

	dbInstance := &models.DatabaseInstance{
		ID:           uuid.New(),
		DatabaseType: models.DatabaseTypePostgreSQL,
		Host:         "localhost",
		Port:         5432,
		Username:     "testuser",
	}
	dbModel := &models.Database{
		ID:                 uuid.New(),
		DatabaseName:       "testdb",
		DatabaseInstanceID: dbInstance.ID,
	}

	// Mock transaction begin
	sqlMock.ExpectBegin()

	// Mock SQL execution error
	sqlMock.ExpectExec("CREATE TABLE test").
		WillReturnError(sql.ErrConnDone)

	// Mock rollback
	sqlMock.ExpectRollback()

	result, err := service.executeInTransaction(ctx, dbInstance, dbModel, sqlContent)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "execution failed")

	require.NoError(t, sqlMock.ExpectationsWereMet())
	connectionMgr.AssertExpectations(t)
}

func TestExecutionService_executeInTransaction_CommitError(t *testing.T) {
	logger := zap.NewNop()

	// Create mock database
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	// Create mock ConnectionManager
	connectionMgr := &MockConnectionManager{}
	connectionMgr.On("Connect", mock.Anything, mock.Anything).Return(db, nil)

	service := &ExecutionService{
		connectionMgr: connectionMgr,
		logger:        logger,
	}

	ctx := context.Background()
	sqlContent := "CREATE TABLE test (id UUID PRIMARY KEY);"

	dbInstance := &models.DatabaseInstance{
		ID:           uuid.New(),
		DatabaseType: models.DatabaseTypePostgreSQL,
		Host:         "localhost",
		Port:         5432,
		Username:     "testuser",
	}
	dbModel := &models.Database{
		ID:                 uuid.New(),
		DatabaseName:       "testdb",
		DatabaseInstanceID: dbInstance.ID,
	}

	// Mock transaction begin
	sqlMock.ExpectBegin()

	// Mock SQL execution success
	sqlMock.ExpectExec("CREATE TABLE test").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock commit error
	sqlMock.ExpectCommit().
		WillReturnError(sql.ErrConnDone)

	result, err := service.executeInTransaction(ctx, dbInstance, dbModel, sqlContent)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to commit transaction")

	// Note: Rollback should be called by defer, but sqlmock may not match it
	// if the transaction is in a bad state after commit failure
	require.NoError(t, sqlMock.ExpectationsWereMet())
	connectionMgr.AssertExpectations(t)
}

func TestExecutionService_executeInTransaction_BeginError(t *testing.T) {
	logger := zap.NewNop()

	// Create mock database
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	// Create mock ConnectionManager
	connectionMgr := &MockConnectionManager{}
	connectionMgr.On("Connect", mock.Anything, mock.Anything).Return(db, nil)

	service := &ExecutionService{
		connectionMgr: connectionMgr,
		logger:        logger,
	}

	ctx := context.Background()
	sqlContent := "CREATE TABLE test (id UUID PRIMARY KEY);"

	dbInstance := &models.DatabaseInstance{
		ID:           uuid.New(),
		DatabaseType: models.DatabaseTypePostgreSQL,
		Host:         "localhost",
		Port:         5432,
		Username:     "testuser",
	}
	dbModel := &models.Database{
		ID:                 uuid.New(),
		DatabaseName:       "testdb",
		DatabaseInstanceID: dbInstance.ID,
	}

	// Mock transaction begin error
	sqlMock.ExpectBegin().
		WillReturnError(sql.ErrConnDone)

	result, err := service.executeInTransaction(ctx, dbInstance, dbModel, sqlContent)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to begin transaction")

	require.NoError(t, sqlMock.ExpectationsWereMet())
	connectionMgr.AssertExpectations(t)
}
