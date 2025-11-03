package services

import (
	"context"
	"database/sql"
	"fmt"
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

// MockExecutionQueueService is a mock implementation of ExecutionQueueService
type MockExecutionQueueService struct {
	mock.Mock
}

func (m *MockExecutionQueueService) Enqueue(ctx context.Context, changePlanID uuid.UUID, databaseID uuid.UUID, executorID uuid.UUID) (*models.ExecutionQueue, error) {
	args := m.Called(ctx, changePlanID, databaseID, executorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExecutionQueue), args.Error(1)
}

func (m *MockExecutionQueueService) GetQueueStatus(ctx context.Context, changePlanID uuid.UUID) (*models.ExecutionQueue, error) {
	args := m.Called(ctx, changePlanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExecutionQueue), args.Error(1)
}

func (m *MockExecutionQueueService) GetQueuePosition(ctx context.Context, changePlanID uuid.UUID) (int, error) {
	args := m.Called(ctx, changePlanID)
	return args.Int(0), args.Error(1)
}

func (m *MockExecutionQueueService) GetQueueItemsForDatabase(ctx context.Context, databaseID uuid.UUID) ([]*models.ExecutionQueue, error) {
	args := m.Called(ctx, databaseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ExecutionQueue), args.Error(1)
}

func (m *MockExecutionQueueService) GetPendingCount(ctx context.Context, databaseID uuid.UUID) (int, error) {
	args := m.Called(ctx, databaseID)
	return args.Int(0), args.Error(1)
}

// MockExecutionLogRepository is a mock implementation of ExecutionLogRepository
type MockExecutionLogRepository struct {
	mock.Mock
}

func (m *MockExecutionLogRepository) Create(ctx context.Context, log *models.ExecutionLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockExecutionLogRepository) GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.ExecutionLog, error) {
	args := m.Called(ctx, changePlanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ExecutionLog), args.Error(1)
}

func (m *MockExecutionLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ExecutionLog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExecutionLog), args.Error(1)
}

func (m *MockExecutionLogRepository) GetLatestByChangePlanID(ctx context.Context, changePlanID uuid.UUID) (*models.ExecutionLog, error) {
	args := m.Called(ctx, changePlanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExecutionLog), args.Error(1)
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

// NOTE: TestExecutionService_Execute_Success_PostgreSQL and TestExecutionService_Execute_Success_MySQL
// are skipped because they require external commands (pg_dump/mysqldump) to be available.
// These should be implemented as integration tests.
// For now, we focus on testing error paths and validation logic.

func TestExecutionService_Execute_Success_PostgreSQL(t *testing.T) {
	t.Skip("Skipping successful execution test - requires pg_dump to be available. Should be integration test.")
	logger := zap.NewNop()
	ctx := context.Background()
	encryptionKey := "12345678901234567890123456789012" // 32 bytes

	changePlanID := uuid.New()
	executorID := uuid.New()
	databaseID := uuid.New()
	databaseInstanceID := uuid.New()

	changePlan := &models.ChangePlan{
		ID:               changePlanID,
		Status:           models.ChangePlanStatusApproved,
		TargetDatabaseID: databaseID,
		SQLContent:       "CREATE TABLE test (id UUID PRIMARY KEY);",
	}

	db := &models.Database{
		ID:                 databaseID,
		DatabaseInstanceID: databaseInstanceID,
		DatabaseName:       "testdb",
	}

	dbInstance := &models.DatabaseInstance{
		ID:           databaseInstanceID,
		DatabaseType: models.DatabaseTypePostgreSQL,
		Host:         "localhost",
		Port:         5432,
		Username:     "testuser",
	}

	beforeSnapshot := &models.SchemaSnapshot{
		ID:           uuid.New(),
		ChangePlanID: changePlanID,
		DatabaseID:   databaseID,
		SnapshotType: models.SnapshotTypeBefore,
	}

	afterSnapshot := &models.SchemaSnapshot{
		ID:           uuid.New(),
		ChangePlanID: changePlanID,
		DatabaseID:   databaseID,
		SnapshotType: models.SnapshotTypeAfter,
	}

	// Create mock database for SQL execution
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dbConn := sqlx.NewDb(mockDB, "sqlmock")
	defer dbConn.Close()

	// Mock transaction
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("CREATE TABLE test").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectCommit()

	// Setup mocks
	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	snapshotRepo := new(MockSchemaSnapshotRepository)
	lockManager := new(MockExecutionLockManager)
	queueRepo := new(MockExecutionQueueRepository)
	executionLogRepo := new(MockExecutionLogRepository)
	
	// Use real ConnectionManager for SchemaSnapshotService (it will try to execute commands, but we'll handle that)
	realConnectionMgr := database.NewConnectionManager(encryptionKey, logger)
	
	// Create real services with mocked repositories
	snapshotService := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		realConnectionMgr,
		encryptionKey,
		logger,
	)
	queueService := NewExecutionQueueService(queueRepo, logger)

	// Setup expectations
	changePlanRepo.On("GetByID", ctx, changePlanID).Return(changePlan, nil)
	databaseRepo.On("GetByID", ctx, databaseID).Return(db, nil)
	dbInstanceRepo.On("GetByID", ctx, databaseInstanceID).Return(dbInstance, nil)
	lockManager.On("AcquireLock", ctx, databaseID, changePlanID, executorID).Return(nil)
	lockManager.On("ReleaseLock", ctx, databaseID).Return(nil)
	
	// Mock snapshot capture - the service will try to execute pg_dump, but we'll mock the repository save
	snapshotRepo.On("Create", ctx, mock.MatchedBy(func(ss *models.SchemaSnapshot) bool {
		return ss.ChangePlanID == changePlanID && ss.DatabaseID == databaseID && ss.SnapshotType == models.SnapshotTypeBefore
	})).Return(nil).Run(func(args mock.Arguments) {
		ss := args.Get(1).(*models.SchemaSnapshot)
		ss.ID = beforeSnapshot.ID
	})
	snapshotRepo.On("Create", ctx, mock.MatchedBy(func(ss *models.SchemaSnapshot) bool {
		return ss.ChangePlanID == changePlanID && ss.DatabaseID == databaseID && ss.SnapshotType == models.SnapshotTypeAfter
	})).Return(nil).Run(func(args mock.Arguments) {
		ss := args.Get(1).(*models.SchemaSnapshot)
		ss.ID = afterSnapshot.ID
	})
	
	// Mock connection for SQL execution
	mockDB, sqlMock, err = sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()
	dbConn = sqlx.NewDb(mockDB, "sqlmock")
	defer dbConn.Close()
	
	// Create a mock connection manager that returns our mocked DB for SQL execution
	mockConnectionMgr := new(MockConnectionManager)
	mockConnectionMgr.On("Connect", ctx, mock.Anything).Return(dbConn, nil)
	
	changePlanRepo.On("Update", ctx, mock.MatchedBy(func(cp *models.ChangePlan) bool {
		return cp.Status == models.ChangePlanStatusInProgress
	})).Return(nil)
	changePlanRepo.On("Update", ctx, mock.MatchedBy(func(cp *models.ChangePlan) bool {
		return cp.Status == models.ChangePlanStatusCompleted && cp.AffectedRows != nil && *cp.AffectedRows == 0
	})).Return(nil)
	executionLogRepo.On("Create", ctx, mock.MatchedBy(func(log *models.ExecutionLog) bool {
		return log.Status == models.ExecutionLogStatusCompleted && log.ChangePlanID == changePlanID
	})).Return(nil)

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		queueService,
		executionLogRepo,
		mockConnectionMgr, // Use mock for SQL execution
		encryptionKey,
		logger,
	)

	result, err := service.Execute(ctx, changePlanID, executorID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.RowsAffected)
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.Nil(t, result.ErrorMessage)

	require.NoError(t, sqlMock.ExpectationsWereMet())
	changePlanRepo.AssertExpectations(t)
	dbInstanceRepo.AssertExpectations(t)
	databaseRepo.AssertExpectations(t)
	snapshotRepo.AssertExpectations(t)
	lockManager.AssertExpectations(t)
	executionLogRepo.AssertExpectations(t)
	mockConnectionMgr.AssertExpectations(t)
}

func TestExecutionService_Execute_Success_MySQL(t *testing.T) {
	t.Skip("Skipping successful execution test - requires mysqldump to be available. Should be integration test.")
	logger := zap.NewNop()
	ctx := context.Background()
	encryptionKey := "12345678901234567890123456789012" // 32 bytes

	changePlanID := uuid.New()
	executorID := uuid.New()
	databaseID := uuid.New()
	databaseInstanceID := uuid.New()

	changePlan := &models.ChangePlan{
		ID:               changePlanID,
		Status:           models.ChangePlanStatusApproved,
		TargetDatabaseID: databaseID,
		SQLContent:       "CREATE TABLE test (id INT PRIMARY KEY);",
	}

	db := &models.Database{
		ID:                 databaseID,
		DatabaseInstanceID: databaseInstanceID,
		DatabaseName:       "testdb",
	}

	dbInstance := &models.DatabaseInstance{
		ID:           databaseInstanceID,
		DatabaseType: models.DatabaseTypeMySQL,
		Host:         "localhost",
		Port:         3306,
		Username:     "testuser",
	}

	beforeSnapshot := &models.SchemaSnapshot{
		ID:           uuid.New(),
		ChangePlanID: changePlanID,
		DatabaseID:   databaseID,
		SnapshotType: models.SnapshotTypeBefore,
	}

	afterSnapshot := &models.SchemaSnapshot{
		ID:           uuid.New(),
		ChangePlanID: changePlanID,
		DatabaseID:   databaseID,
		SnapshotType: models.SnapshotTypeAfter,
	}

	// Create mock database for SQL execution
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dbConn := sqlx.NewDb(mockDB, "sqlmock")
	defer dbConn.Close()

	// Mock transaction
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("CREATE TABLE test").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectCommit()

	// Setup mocks
	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	snapshotRepo := new(MockSchemaSnapshotRepository)
	lockManager := new(MockExecutionLockManager)
	queueRepo := new(MockExecutionQueueRepository)
	executionLogRepo := new(MockExecutionLogRepository)
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)

	// Create real services with mocked repositories
	snapshotService := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)
	queueService := NewExecutionQueueService(queueRepo, logger)

	// Setup expectations
	changePlanRepo.On("GetByID", ctx, changePlanID).Return(changePlan, nil)
	databaseRepo.On("GetByID", ctx, databaseID).Return(db, nil)
	dbInstanceRepo.On("GetByID", ctx, databaseInstanceID).Return(dbInstance, nil)
	lockManager.On("AcquireLock", ctx, databaseID, changePlanID, executorID).Return(nil)
	lockManager.On("ReleaseLock", ctx, databaseID).Return(nil)
	snapshotRepo.On("Create", ctx, mock.MatchedBy(func(ss *models.SchemaSnapshot) bool {
		return ss.ChangePlanID == changePlanID && ss.DatabaseID == databaseID && ss.SnapshotType == models.SnapshotTypeBefore
	})).Return(nil).Run(func(args mock.Arguments) {
		ss := args.Get(1).(*models.SchemaSnapshot)
		ss.ID = beforeSnapshot.ID
	})
	snapshotRepo.On("Create", ctx, mock.MatchedBy(func(ss *models.SchemaSnapshot) bool {
		return ss.ChangePlanID == changePlanID && ss.DatabaseID == databaseID && ss.SnapshotType == models.SnapshotTypeAfter
	})).Return(nil).Run(func(args mock.Arguments) {
		ss := args.Get(1).(*models.SchemaSnapshot)
		ss.ID = afterSnapshot.ID
	})
	
	// Mock connection for SQL execution  
	mockDB, sqlMock, err = sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()
	dbConn = sqlx.NewDb(mockDB, "sqlmock")
	defer dbConn.Close()
	
	// Create a mock connection manager that returns our mocked DB for SQL execution
	mockConnectionMgr := new(MockConnectionManager)
	mockConnectionMgr.On("Connect", ctx, mock.Anything).Return(dbConn, nil)
	
	changePlanRepo.On("Update", ctx, mock.MatchedBy(func(cp *models.ChangePlan) bool {
		return cp.Status == models.ChangePlanStatusInProgress
	})).Return(nil)
	changePlanRepo.On("Update", ctx, mock.MatchedBy(func(cp *models.ChangePlan) bool {
		return cp.Status == models.ChangePlanStatusCompleted
	})).Return(nil)
	executionLogRepo.On("Create", ctx, mock.MatchedBy(func(log *models.ExecutionLog) bool {
		return log.Status == models.ExecutionLogStatusCompleted
	})).Return(nil)

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		queueService,
		executionLogRepo,
		mockConnectionMgr, // Use mock for SQL execution
		encryptionKey,
		logger,
	)

	result, err := service.Execute(ctx, changePlanID, executorID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	require.NoError(t, sqlMock.ExpectationsWereMet())
	changePlanRepo.AssertExpectations(t)
	dbInstanceRepo.AssertExpectations(t)
	databaseRepo.AssertExpectations(t)
	snapshotRepo.AssertExpectations(t)
	lockManager.AssertExpectations(t)
	executionLogRepo.AssertExpectations(t)
	mockConnectionMgr.AssertExpectations(t)
}

func TestExecutionService_Execute_InvalidStatus(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()
	encryptionKey := "12345678901234567890123456789012"

	changePlanID := uuid.New()
	executorID := uuid.New()

	changePlan := &models.ChangePlan{
		ID:     changePlanID,
		Status: models.ChangePlanStatusOpen, // Not Approved
	}

	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	snapshotRepo := new(MockSchemaSnapshotRepository)
	lockManager := new(MockExecutionLockManager)
	queueRepo := new(MockExecutionQueueRepository)
	executionLogRepo := new(MockExecutionLogRepository)
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)

	// Create real services with mocked repositories
	snapshotService := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)
	queueService := NewExecutionQueueService(queueRepo, logger)

	changePlanRepo.On("GetByID", ctx, changePlanID).Return(changePlan, nil)

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		queueService,
		executionLogRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)

	result, err := service.Execute(ctx, changePlanID, executorID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "must be in Approved status")

	changePlanRepo.AssertExpectations(t)
}

func TestExecutionService_Execute_DatabaseNotFound(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()
	encryptionKey := "12345678901234567890123456789012"

	changePlanID := uuid.New()
	executorID := uuid.New()
	databaseID := uuid.New()

	changePlan := &models.ChangePlan{
		ID:               changePlanID,
		Status:           models.ChangePlanStatusApproved,
		TargetDatabaseID: databaseID,
	}

	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	snapshotRepo := new(MockSchemaSnapshotRepository)
	lockManager := new(MockExecutionLockManager)
	queueRepo := new(MockExecutionQueueRepository)
	executionLogRepo := new(MockExecutionLogRepository)
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)

	// Create real services with mocked repositories
	snapshotService := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)
	queueService := NewExecutionQueueService(queueRepo, logger)

	changePlanRepo.On("GetByID", ctx, changePlanID).Return(changePlan, nil)
	databaseRepo.On("GetByID", ctx, databaseID).Return(nil, repository.ErrNotFound)

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		queueService,
		executionLogRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)

	result, err := service.Execute(ctx, changePlanID, executorID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrDatabaseNotFound, err)

	changePlanRepo.AssertExpectations(t)
	databaseRepo.AssertExpectations(t)
}

func TestExecutionService_Execute_DatabaseInstanceNotFound(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()
	encryptionKey := "12345678901234567890123456789012"

	changePlanID := uuid.New()
	executorID := uuid.New()
	databaseID := uuid.New()
	databaseInstanceID := uuid.New()

	changePlan := &models.ChangePlan{
		ID:               changePlanID,
		Status:           models.ChangePlanStatusApproved,
		TargetDatabaseID: databaseID,
	}

	db := &models.Database{
		ID:                 databaseID,
		DatabaseInstanceID: databaseInstanceID,
	}

	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	snapshotRepo := new(MockSchemaSnapshotRepository)
	lockManager := new(MockExecutionLockManager)
	queueRepo := new(MockExecutionQueueRepository)
	executionLogRepo := new(MockExecutionLogRepository)
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)

	// Create real services with mocked repositories
	snapshotService := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)
	queueService := NewExecutionQueueService(queueRepo, logger)

	changePlanRepo.On("GetByID", ctx, changePlanID).Return(changePlan, nil)
	databaseRepo.On("GetByID", ctx, databaseID).Return(db, nil)
	dbInstanceRepo.On("GetByID", ctx, databaseInstanceID).Return(nil, repository.ErrNotFound)

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		queueService,
		executionLogRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)

	result, err := service.Execute(ctx, changePlanID, executorID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrDatabaseInstanceNotFound, err)

	changePlanRepo.AssertExpectations(t)
	databaseRepo.AssertExpectations(t)
	dbInstanceRepo.AssertExpectations(t)
}

func TestExecutionService_Execute_DatabaseLocked_WithQueue(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()
	encryptionKey := "12345678901234567890123456789012"

	changePlanID := uuid.New()
	executorID := uuid.New()
	databaseID := uuid.New()
	databaseInstanceID := uuid.New()

	changePlan := &models.ChangePlan{
		ID:               changePlanID,
		Status:           models.ChangePlanStatusApproved,
		TargetDatabaseID: databaseID,
	}

	db := &models.Database{
		ID:                 databaseID,
		DatabaseInstanceID: databaseInstanceID,
	}

	dbInstance := &models.DatabaseInstance{
		ID:           databaseInstanceID,
		DatabaseType: models.DatabaseTypePostgreSQL,
	}

	queueItem := &models.ExecutionQueue{
		ID:               uuid.New(),
		ChangePlanID:    changePlanID,
		TargetDatabaseID: databaseID,
		ExecutorID:      executorID,
		Status:          models.QueueStatusPending,
		Position:        1,
	}

	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	snapshotRepo := new(MockSchemaSnapshotRepository)
	lockManager := new(MockExecutionLockManager)
	queueRepo := new(MockExecutionQueueRepository)
	executionLogRepo := new(MockExecutionLogRepository)
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)

	// Create real services with mocked repositories
	snapshotService := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)
	queueService := NewExecutionQueueService(queueRepo, logger)

	changePlanRepo.On("GetByID", ctx, changePlanID).Return(changePlan, nil)
	databaseRepo.On("GetByID", ctx, databaseID).Return(db, nil)
	dbInstanceRepo.On("GetByID", ctx, databaseInstanceID).Return(dbInstance, nil)
	lockManager.On("AcquireLock", ctx, databaseID, changePlanID, executorID).Return(ErrDatabaseLocked)
	queueRepo.On("Enqueue", ctx, mock.MatchedBy(func(item *models.ExecutionQueue) bool {
		return item.ChangePlanID == changePlanID && item.TargetDatabaseID == databaseID && item.ExecutorID == executorID
	})).Return(nil).Run(func(args mock.Arguments) {
		item := args.Get(1).(*models.ExecutionQueue)
		item.ID = queueItem.ID
		item.Position = queueItem.Position
	})

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		queueService,
		executionLogRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)

	result, err := service.Execute(ctx, changePlanID, executorID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrDatabaseLocked, err)

	changePlanRepo.AssertExpectations(t)
	databaseRepo.AssertExpectations(t)
	dbInstanceRepo.AssertExpectations(t)
	lockManager.AssertExpectations(t)
	queueRepo.AssertExpectations(t)
}

func TestExecutionService_Execute_DatabaseLocked_WithoutQueue(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()
	encryptionKey := "12345678901234567890123456789012"

	changePlanID := uuid.New()
	executorID := uuid.New()
	databaseID := uuid.New()
	databaseInstanceID := uuid.New()

	changePlan := &models.ChangePlan{
		ID:               changePlanID,
		Status:           models.ChangePlanStatusApproved,
		TargetDatabaseID: databaseID,
	}

	db := &models.Database{
		ID:                 databaseID,
		DatabaseInstanceID: databaseInstanceID,
	}

	dbInstance := &models.DatabaseInstance{
		ID:           databaseInstanceID,
		DatabaseType: models.DatabaseTypePostgreSQL,
	}

	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	snapshotRepo := new(MockSchemaSnapshotRepository)
	lockManager := new(MockExecutionLockManager)
	executionLogRepo := new(MockExecutionLogRepository)
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)

	// Create real services with mocked repositories
	snapshotService := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)

	changePlanRepo.On("GetByID", ctx, changePlanID).Return(changePlan, nil)
	databaseRepo.On("GetByID", ctx, databaseID).Return(db, nil)
	dbInstanceRepo.On("GetByID", ctx, databaseInstanceID).Return(dbInstance, nil)
	lockManager.On("AcquireLock", ctx, databaseID, changePlanID, executorID).Return(ErrDatabaseLocked)

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		nil, // No queue service
		executionLogRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)

	result, err := service.Execute(ctx, changePlanID, executorID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrDatabaseLocked, err)

	changePlanRepo.AssertExpectations(t)
	databaseRepo.AssertExpectations(t)
	dbInstanceRepo.AssertExpectations(t)
	lockManager.AssertExpectations(t)
}

func TestExecutionService_Execute_BeforeSnapshotFailure(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()
	encryptionKey := "12345678901234567890123456789012"

	changePlanID := uuid.New()
	executorID := uuid.New()
	databaseID := uuid.New()
	databaseInstanceID := uuid.New()

	changePlan := &models.ChangePlan{
		ID:               changePlanID,
		Status:           models.ChangePlanStatusApproved,
		TargetDatabaseID: databaseID,
	}

	db := &models.Database{
		ID:                 databaseID,
		DatabaseInstanceID: databaseInstanceID,
	}

	dbInstance := &models.DatabaseInstance{
		ID:           databaseInstanceID,
		DatabaseType: models.DatabaseTypePostgreSQL,
	}

	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	snapshotRepo := new(MockSchemaSnapshotRepository)
	lockManager := new(MockExecutionLockManager)
	queueRepo := new(MockExecutionQueueRepository)
	executionLogRepo := new(MockExecutionLogRepository)
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)

	// Create real services with mocked repositories
	snapshotService := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)
	queueService := NewExecutionQueueService(queueRepo, logger)

	changePlanRepo.On("GetByID", ctx, changePlanID).Return(changePlan, nil)
	databaseRepo.On("GetByID", ctx, databaseID).Return(db, nil)
	dbInstanceRepo.On("GetByID", ctx, databaseInstanceID).Return(dbInstance, nil)
	lockManager.On("AcquireLock", ctx, databaseID, changePlanID, executorID).Return(nil)
	lockManager.On("ReleaseLock", ctx, databaseID).Return(nil)
	// Mock snapshot repository to return error on create
	snapshotRepo.On("Create", ctx, mock.MatchedBy(func(ss *models.SchemaSnapshot) bool {
		return ss.ChangePlanID == changePlanID && ss.DatabaseID == databaseID && ss.SnapshotType == models.SnapshotTypeBefore
	})).Return(fmt.Errorf("snapshot failed"))
	changePlanRepo.On("Update", ctx, mock.MatchedBy(func(cp *models.ChangePlan) bool {
		return cp.Status == models.ChangePlanStatusInProgress
	})).Return(nil)
	changePlanRepo.On("Update", ctx, mock.MatchedBy(func(cp *models.ChangePlan) bool {
		return cp.Status == models.ChangePlanStatusFailed
	})).Return(nil)

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		queueService,
		executionLogRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)

	result, err := service.Execute(ctx, changePlanID, executorID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to capture before snapshot")

	changePlanRepo.AssertExpectations(t)
	dbInstanceRepo.AssertExpectations(t)
	databaseRepo.AssertExpectations(t)
	lockManager.AssertExpectations(t)
}

func TestExecutionService_Execute_AfterSnapshotFailure(t *testing.T) {
	t.Skip("Skipping after snapshot failure test - requires pg_dump to be available. Should be integration test.")
	logger := zap.NewNop()
	ctx := context.Background()
	encryptionKey := "12345678901234567890123456789012"

	changePlanID := uuid.New()
	executorID := uuid.New()
	databaseID := uuid.New()
	databaseInstanceID := uuid.New()

	changePlan := &models.ChangePlan{
		ID:               changePlanID,
		Status:           models.ChangePlanStatusApproved,
		TargetDatabaseID: databaseID,
		SQLContent:       "CREATE TABLE test (id UUID PRIMARY KEY);",
	}

	db := &models.Database{
		ID:                 databaseID,
		DatabaseInstanceID: databaseInstanceID,
		DatabaseName:       "testdb",
	}

	dbInstance := &models.DatabaseInstance{
		ID:           databaseInstanceID,
		DatabaseType: models.DatabaseTypePostgreSQL,
		Host:         "localhost",
		Port:         5432,
		Username:     "testuser",
	}

	beforeSnapshot := &models.SchemaSnapshot{
		ID:           uuid.New(),
		ChangePlanID: changePlanID,
		DatabaseID:   databaseID,
		SnapshotType: models.SnapshotTypeBefore,
	}

	// Create mock database for SQL execution
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dbConn := sqlx.NewDb(mockDB, "sqlmock")
	defer dbConn.Close()

	// Mock transaction
	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("CREATE TABLE test").
		WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectCommit()

	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	snapshotRepo := new(MockSchemaSnapshotRepository)
	lockManager := new(MockExecutionLockManager)
	queueRepo := new(MockExecutionQueueRepository)
	executionLogRepo := new(MockExecutionLogRepository)
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)

	// Create real services with mocked repositories  
	snapshotService := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)
	queueService := NewExecutionQueueService(queueRepo, logger)

	changePlanRepo.On("GetByID", ctx, changePlanID).Return(changePlan, nil)
	databaseRepo.On("GetByID", ctx, databaseID).Return(db, nil)
	dbInstanceRepo.On("GetByID", ctx, databaseInstanceID).Return(dbInstance, nil)
	lockManager.On("AcquireLock", ctx, databaseID, changePlanID, executorID).Return(nil)
	lockManager.On("ReleaseLock", ctx, databaseID).Return(nil)
	snapshotRepo.On("Create", ctx, mock.MatchedBy(func(ss *models.SchemaSnapshot) bool {
		return ss.ChangePlanID == changePlanID && ss.DatabaseID == databaseID && ss.SnapshotType == models.SnapshotTypeBefore
	})).Return(nil).Run(func(args mock.Arguments) {
		ss := args.Get(1).(*models.SchemaSnapshot)
		ss.ID = beforeSnapshot.ID
	})
	snapshotRepo.On("Create", ctx, mock.MatchedBy(func(ss *models.SchemaSnapshot) bool {
		return ss.ChangePlanID == changePlanID && ss.DatabaseID == databaseID && ss.SnapshotType == models.SnapshotTypeAfter
	})).Return(fmt.Errorf("after snapshot failed"))
	mockConnectionMgr := new(MockConnectionManager)
	mockConnectionMgr.On("Connect", ctx, mock.Anything).Return(dbConn, nil)
	changePlanRepo.On("Update", ctx, mock.MatchedBy(func(cp *models.ChangePlan) bool {
		return cp.Status == models.ChangePlanStatusInProgress
	})).Return(nil)
	changePlanRepo.On("Update", ctx, mock.MatchedBy(func(cp *models.ChangePlan) bool {
		return cp.Status == models.ChangePlanStatusFailed
	})).Return(nil)

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		queueService,
		executionLogRepo,
		mockConnectionMgr,
		encryptionKey,
		logger,
	)

	result, err := service.Execute(ctx, changePlanID, executorID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to capture after snapshot")

	require.NoError(t, sqlMock.ExpectationsWereMet())
	changePlanRepo.AssertExpectations(t)
	snapshotRepo.AssertExpectations(t)
	lockManager.AssertExpectations(t)
	mockConnectionMgr.AssertExpectations(t)
}

func TestExecutionService_Execute_Timeout(t *testing.T) {
	t.Skip("Skipping timeout test - requires database connection. Should be integration test.")
	logger := zap.NewNop()
	ctx := context.Background()
	encryptionKey := "12345678901234567890123456789012"
	timeout := 100 * time.Millisecond

	changePlanID := uuid.New()
	executorID := uuid.New()
	databaseID := uuid.New()
	databaseInstanceID := uuid.New()

	changePlan := &models.ChangePlan{
		ID:               changePlanID,
		Status:           models.ChangePlanStatusApproved,
		TargetDatabaseID: databaseID,
		SQLContent:       "SELECT pg_sleep(1);", // Will timeout
	}

	db := &models.Database{
		ID:                 databaseID,
		DatabaseInstanceID: databaseInstanceID,
		DatabaseName:       "testdb",
	}

	dbInstance := &models.DatabaseInstance{
		ID:           databaseInstanceID,
		DatabaseType: models.DatabaseTypePostgreSQL,
		Host:         "localhost",
		Port:         5432,
		Username:     "testuser",
	}

	beforeSnapshot := &models.SchemaSnapshot{
		ID:           uuid.New(),
		ChangePlanID: changePlanID,
		DatabaseID:   databaseID,
		SnapshotType: models.SnapshotTypeBefore,
	}

	// Create mock database for SQL execution
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	dbConn := sqlx.NewDb(mockDB, "sqlmock")
	defer dbConn.Close()

	// Mock transaction begin - but execution will timeout
	sqlMock.ExpectBegin()
	// Note: We can't actually simulate a timeout with sqlmock, but we can test the timeout error handling

	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	snapshotRepo := new(MockSchemaSnapshotRepository)
	lockManager := new(MockExecutionLockManager)
	queueRepo := new(MockExecutionQueueRepository)
	executionLogRepo := new(MockExecutionLogRepository)
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)

	// Create real services with mocked repositories
	snapshotService := NewSchemaSnapshotService(
		snapshotRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)
	queueService := NewExecutionQueueService(queueRepo, logger)

	changePlanRepo.On("GetByID", ctx, changePlanID).Return(changePlan, nil)
	databaseRepo.On("GetByID", ctx, databaseID).Return(db, nil)
	dbInstanceRepo.On("GetByID", ctx, databaseInstanceID).Return(dbInstance, nil)
	lockManager.On("AcquireLock", ctx, databaseID, changePlanID, executorID).Return(nil)
	lockManager.On("ReleaseLock", ctx, databaseID).Return(nil)
	snapshotRepo.On("Create", ctx, mock.MatchedBy(func(ss *models.SchemaSnapshot) bool {
		return ss.ChangePlanID == changePlanID && ss.DatabaseID == databaseID && ss.SnapshotType == models.SnapshotTypeBefore
	})).Return(nil).Run(func(args mock.Arguments) {
		ss := args.Get(1).(*models.SchemaSnapshot)
		ss.ID = beforeSnapshot.ID
	})
	mockConnectionMgr := new(MockConnectionManager)
	mockConnectionMgr.On("Connect", ctx, mock.Anything).Return(dbConn, nil)
	changePlanRepo.On("Update", ctx, mock.MatchedBy(func(cp *models.ChangePlan) bool {
		return cp.Status == models.ChangePlanStatusInProgress
	})).Return(nil)
	changePlanRepo.On("Update", ctx, mock.MatchedBy(func(cp *models.ChangePlan) bool {
		return cp.Status == models.ChangePlanStatusFailed
	})).Return(nil)
	executionLogRepo.On("Create", ctx, mock.MatchedBy(func(log *models.ExecutionLog) bool {
		return log.Status == models.ExecutionLogStatusFailed
	})).Return(nil)

	service := NewExecutionServiceWithTimeout(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		queueService,
		executionLogRepo,
		mockConnectionMgr,
		encryptionKey,
		timeout,
		logger,
	)

	// Use a context with timeout to simulate timeout scenario
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := service.Execute(timeoutCtx, changePlanID, executorID)

	// The timeout should occur during execution
	// Note: sqlmock doesn't actually delay, so we test the error path
	assert.Error(t, err)
	// The result will be nil on error, but if timeout handling works, error message should indicate timeout
	if err != nil {
		// Timeout error should be handled
		assert.NotNil(t, result) // Result is returned even on error for failed executions
	}

	changePlanRepo.AssertExpectations(t)
	snapshotRepo.AssertExpectations(t)
	lockManager.AssertExpectations(t)
	mockConnectionMgr.AssertExpectations(t)
}

func TestExecutionService_GetQueueStatus(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	changePlanID := uuid.New()
	queueItem := &models.ExecutionQueue{
		ID:            uuid.New(),
		ChangePlanID: changePlanID,
		Status:        models.QueueStatusPending,
		Position:      1,
	}

	queueRepo := new(MockExecutionQueueRepository)
	queueRepo.On("GetByChangePlanID", ctx, changePlanID).Return([]*models.ExecutionQueue{queueItem}, nil)
	queueService := NewExecutionQueueService(queueRepo, logger)

	service := &ExecutionService{
		queueService: queueService,
		logger:       logger,
	}

	result, err := service.GetQueueStatus(ctx, changePlanID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, queueItem.ID, result.ID)
	assert.Equal(t, changePlanID, result.ChangePlanID)

	queueRepo.AssertExpectations(t)
}

func TestExecutionService_GetQueueStatus_NoQueueService(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	changePlanID := uuid.New()

	service := &ExecutionService{
		queueService: nil,
		logger:       logger,
	}

	result, err := service.GetQueueStatus(ctx, changePlanID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "queue service not available")
}

func TestExecutionService_GetExecutionLogHistory(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	changePlanID := uuid.New()
	executorID := uuid.New()
	rowsAffected := 5

	logs := []*models.ExecutionLog{
		{
			ID:                  uuid.New(),
			ChangePlanID:        changePlanID,
			ExecutorID:          executorID,
			Status:              models.ExecutionLogStatusCompleted,
			RowsAffected:        &rowsAffected,
			ExecutionDurationMs: 100,
		},
	}

	executionLogRepo := new(MockExecutionLogRepository)
	executionLogRepo.On("GetByChangePlanID", ctx, changePlanID).Return(logs, nil)

	service := &ExecutionService{
		executionLogRepo: executionLogRepo,
		logger:          logger,
	}

	result, err := service.GetExecutionLogHistory(ctx, changePlanID)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, logs[0].ID, result[0].ID)

	executionLogRepo.AssertExpectations(t)
}

func TestExecutionService_GetExecutionLogHistory_NoRepository(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	changePlanID := uuid.New()

	service := &ExecutionService{
		executionLogRepo: nil,
		logger:          logger,
	}

	result, err := service.GetExecutionLogHistory(ctx, changePlanID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "execution log repository not available")
}
