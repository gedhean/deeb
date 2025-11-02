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
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewExecutionService(t *testing.T) {
	changePlanRepo := new(MockChangePlanRepository)
	dbInstanceRepo := new(MockDatabaseInstanceRepository)
	databaseRepo := new(MockDatabaseRepository)
	connectionMgr := database.NewConnectionManager("test-encryption-key-min-32-chars", zap.NewNop())
	logger := zap.NewNop()

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		"test-encryption-key-min-32-chars",
		logger,
	)

	assert.NotNil(t, service)
	assert.Equal(t, changePlanRepo, service.changePlanRepo)
	assert.Equal(t, dbInstanceRepo, service.dbInstanceRepo)
	assert.Equal(t, databaseRepo, service.databaseRepo)
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

	changePlanID := uuid.New()
	changePlanRepo.On("GetByID", ctx, changePlanID).Return(nil, repository.ErrNotFound)

	service := NewExecutionService(
		changePlanRepo,
		dbInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger,
	)

	result, err := service.Execute(ctx, changePlanID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrChangePlanNotFound, err)

	changePlanRepo.AssertExpectations(t)
}

func TestExecutionService_executeInTransaction_Success(t *testing.T) {
	logger := zap.NewNop()

	// Create mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	service := &ExecutionService{
		logger: logger,
	}

	ctx := context.Background()
	sqlContent := "CREATE TABLE test (id UUID PRIMARY KEY);"

	// Mock transaction begin
	mock.ExpectBegin()

	// Mock SQL execution
	mock.ExpectExec("CREATE TABLE test").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock commit
	mock.ExpectCommit()

	result, err := service.executeInTransaction(ctx, db, sqlContent)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.RowsAffected)
	assert.Greater(t, result.Duration, time.Duration(0))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecutionService_executeInTransaction_WithRowsAffected(t *testing.T) {
	logger := zap.NewNop()

	// Create mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	service := &ExecutionService{
		logger: logger,
	}

	ctx := context.Background()
	sqlContent := "INSERT INTO users (id, name) VALUES (gen_random_uuid(), 'test');"

	// Mock transaction begin
	mock.ExpectBegin()

	// Mock SQL execution with rows affected
	mock.ExpectExec("INSERT INTO users").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Mock commit
	mock.ExpectCommit()

	result, err := service.executeInTransaction(ctx, db, sqlContent)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 1, result.RowsAffected)
	assert.Greater(t, result.Duration, time.Duration(0))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecutionService_executeInTransaction_ExecutionError(t *testing.T) {
	logger := zap.NewNop()

	// Create mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	service := &ExecutionService{
		logger: logger,
	}

	ctx := context.Background()
	sqlContent := "CREATE TABLE test (id UUID PRIMARY KEY);"

	// Mock transaction begin
	mock.ExpectBegin()

	// Mock SQL execution error
	mock.ExpectExec("CREATE TABLE test").
		WillReturnError(sql.ErrConnDone)

	// Mock rollback
	mock.ExpectRollback()

	result, err := service.executeInTransaction(ctx, db, sqlContent)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "execution failed")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecutionService_executeInTransaction_CommitError(t *testing.T) {
	logger := zap.NewNop()

	// Create mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	service := &ExecutionService{
		logger: logger,
	}

	ctx := context.Background()
	sqlContent := "CREATE TABLE test (id UUID PRIMARY KEY);"

	// Mock transaction begin
	mock.ExpectBegin()

	// Mock SQL execution success
	mock.ExpectExec("CREATE TABLE test").
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Mock commit error
	mock.ExpectCommit().
		WillReturnError(sql.ErrConnDone)

	result, err := service.executeInTransaction(ctx, db, sqlContent)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to commit transaction")

	// Note: Rollback should be called by defer, but sqlmock may not match it
	// if the transaction is in a bad state after commit failure
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExecutionService_executeInTransaction_BeginError(t *testing.T) {
	logger := zap.NewNop()

	// Create mock database
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	service := &ExecutionService{
		logger: logger,
	}

	ctx := context.Background()
	sqlContent := "CREATE TABLE test (id UUID PRIMARY KEY);"

	// Mock transaction begin error
	mock.ExpectBegin().
		WillReturnError(sql.ErrConnDone)

	result, err := service.executeInTransaction(ctx, db, sqlContent)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to begin transaction")

	require.NoError(t, mock.ExpectationsWereMet())
}

