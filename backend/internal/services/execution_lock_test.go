package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewExecutionLockManager(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()

	manager := NewExecutionLockManager(db, logger)

	assert.NotNil(t, manager)
	impl, ok := manager.(*executionLockManager)
	assert.True(t, ok)
	assert.Equal(t, db, impl.db)
	assert.Equal(t, logger, impl.logger)
}

func TestExecutionLockManager_AcquireLock_Success(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()
	changePlanID := uuid.New()
	executorID := uuid.New()

	// Mock transaction begin
	sqlMock.ExpectBegin()

	// Mock SELECT FOR UPDATE NOWAIT (returns no rows - database not locked)
	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1 FOR UPDATE NOWAIT").
		WithArgs(databaseID).
		WillReturnError(sql.ErrNoRows)

	// Mock INSERT lock record
	sqlMock.ExpectExec("INSERT INTO execution_locks").
		WithArgs(databaseID, changePlanID, executorID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock commit
	sqlMock.ExpectCommit()

	err = manager.AcquireLock(ctx, databaseID, changePlanID, executorID)
	require.NoError(t, err)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLockManager_AcquireLock_DatabaseLocked(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()
	changePlanID := uuid.New()
	executorID := uuid.New()

	// Mock transaction begin
	sqlMock.ExpectBegin()

	// Mock SELECT FOR UPDATE NOWAIT (returns existing lock that's not expired)
	existingLock := ExecutionLock{
		ID:              uuid.New(),
		TargetDatabaseID: databaseID,
		ChangePlanID:    uuid.New(),
		ExecutorID:      uuid.New(),
		LockedAt:        time.Now().Add(-30 * time.Minute),
		ExpiresAt:       time.Now().Add(30 * time.Minute), // Not expired
	}

	rows := sqlmock.NewRows([]string{"id", "target_database_id", "change_plan_id", "executor_id", "locked_at", "expires_at"}).
		AddRow(existingLock.ID, existingLock.TargetDatabaseID, existingLock.ChangePlanID, existingLock.ExecutorID, existingLock.LockedAt, existingLock.ExpiresAt)

	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1 FOR UPDATE NOWAIT").
		WithArgs(databaseID).
		WillReturnRows(rows)

	// Mock rollback (transaction will be rolled back)
	sqlMock.ExpectRollback()

	err = manager.AcquireLock(ctx, databaseID, changePlanID, executorID)
	assert.Error(t, err)
	assert.Equal(t, ErrDatabaseLocked, err)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLockManager_AcquireLock_ExpiredLock(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()
	changePlanID := uuid.New()
	executorID := uuid.New()

	// Mock transaction begin
	sqlMock.ExpectBegin()

	// Mock SELECT FOR UPDATE NOWAIT (returns expired lock)
	expiredLock := ExecutionLock{
		ID:              uuid.New(),
		TargetDatabaseID: databaseID,
		ChangePlanID:    uuid.New(),
		ExecutorID:      uuid.New(),
		LockedAt:        time.Now().Add(-2 * time.Hour),
		ExpiresAt:       time.Now().Add(-1 * time.Hour), // Expired
	}

	rows := sqlmock.NewRows([]string{"id", "target_database_id", "change_plan_id", "executor_id", "locked_at", "expires_at"}).
		AddRow(expiredLock.ID, expiredLock.TargetDatabaseID, expiredLock.ChangePlanID, expiredLock.ExecutorID, expiredLock.LockedAt, expiredLock.ExpiresAt)

	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1 FOR UPDATE NOWAIT").
		WithArgs(databaseID).
		WillReturnRows(rows)

	// Mock DELETE expired lock
	sqlMock.ExpectExec("DELETE FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock INSERT new lock record
	sqlMock.ExpectExec("INSERT INTO execution_locks").
		WithArgs(databaseID, changePlanID, executorID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock commit
	sqlMock.ExpectCommit()

	err = manager.AcquireLock(ctx, databaseID, changePlanID, executorID)
	require.NoError(t, err)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLockManager_AcquireLock_LockContention(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()
	changePlanID := uuid.New()
	executorID := uuid.New()

	// Mock transaction begin
	sqlMock.ExpectBegin()

	// Mock SELECT FOR UPDATE NOWAIT (returns lock contention error)
	pqErr := &pq.Error{
		Code: "55P03", // lock_not_available
	}
	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1 FOR UPDATE NOWAIT").
		WithArgs(databaseID).
		WillReturnError(pqErr)

	// Mock rollback
	sqlMock.ExpectRollback()

	err = manager.AcquireLock(ctx, databaseID, changePlanID, executorID)
	assert.Error(t, err)
	assert.Equal(t, ErrDatabaseLocked, err)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLockManager_AcquireLock_UniqueViolation(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()
	changePlanID := uuid.New()
	executorID := uuid.New()

	// Mock transaction begin
	sqlMock.ExpectBegin()

	// Mock SELECT FOR UPDATE NOWAIT (returns no rows)
	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1 FOR UPDATE NOWAIT").
		WithArgs(databaseID).
		WillReturnError(sql.ErrNoRows)

	// Mock INSERT lock record (returns unique violation error)
	pqErr := &pq.Error{
		Code: "23505", // unique_violation
	}
	sqlMock.ExpectExec("INSERT INTO execution_locks").
		WithArgs(databaseID, changePlanID, executorID, sqlmock.AnyArg()).
		WillReturnError(pqErr)

	// Mock rollback
	sqlMock.ExpectRollback()

	err = manager.AcquireLock(ctx, databaseID, changePlanID, executorID)
	assert.Error(t, err)
	assert.Equal(t, ErrDatabaseLocked, err)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLockManager_ReleaseLock_Success(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()

	// Mock DELETE lock
	sqlMock.ExpectExec("DELETE FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = manager.ReleaseLock(ctx, databaseID)
	require.NoError(t, err)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLockManager_ReleaseLock_NonExistent(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()

	// Mock DELETE lock (returns 0 rows affected)
	sqlMock.ExpectExec("DELETE FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = manager.ReleaseLock(ctx, databaseID)
	require.NoError(t, err) // Should not error, just log warning

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLockManager_IsLocked_NotLocked(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()

	// Mock SELECT (returns no rows)
	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnError(sql.ErrNoRows)

	isLocked, lock, err := manager.IsLocked(ctx, databaseID)
	require.NoError(t, err)
	assert.False(t, isLocked)
	assert.Nil(t, lock)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLockManager_IsLocked_Locked(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()

	existingLock := ExecutionLock{
		ID:              uuid.New(),
		TargetDatabaseID: databaseID,
		ChangePlanID:    uuid.New(),
		ExecutorID:      uuid.New(),
		LockedAt:        time.Now().Add(-30 * time.Minute),
		ExpiresAt:       time.Now().Add(30 * time.Minute), // Not expired
	}

	rows := sqlmock.NewRows([]string{"id", "target_database_id", "change_plan_id", "executor_id", "locked_at", "expires_at"}).
		AddRow(existingLock.ID, existingLock.TargetDatabaseID, existingLock.ChangePlanID, existingLock.ExecutorID, existingLock.LockedAt, existingLock.ExpiresAt)

	// Mock SELECT (returns lock)
	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnRows(rows)

	isLocked, lock, err := manager.IsLocked(ctx, databaseID)
	require.NoError(t, err)
	assert.True(t, isLocked)
	assert.NotNil(t, lock)
	assert.Equal(t, existingLock.ID, lock.ID)
	assert.Equal(t, databaseID, lock.TargetDatabaseID)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLockManager_IsLocked_Expired(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()

	expiredLock := ExecutionLock{
		ID:              uuid.New(),
		TargetDatabaseID: databaseID,
		ChangePlanID:    uuid.New(),
		ExecutorID:      uuid.New(),
		LockedAt:        time.Now().Add(-2 * time.Hour),
		ExpiresAt:       time.Now().Add(-1 * time.Hour), // Expired
	}

	rows := sqlmock.NewRows([]string{"id", "target_database_id", "change_plan_id", "executor_id", "locked_at", "expires_at"}).
		AddRow(expiredLock.ID, expiredLock.TargetDatabaseID, expiredLock.ChangePlanID, expiredLock.ExecutorID, expiredLock.LockedAt, expiredLock.ExpiresAt)

	// Mock SELECT (returns expired lock)
	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnRows(rows)

	// Mock DELETE expired lock
	sqlMock.ExpectExec("DELETE FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	isLocked, lock, err := manager.IsLocked(ctx, databaseID)
	require.NoError(t, err)
	assert.False(t, isLocked)
	assert.Nil(t, lock)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLockManager_CleanupExpiredLocks_Success(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()

	// Mock DELETE expired locks
	sqlMock.ExpectExec("DELETE FROM execution_locks WHERE expires_at < NOW\\(\\)").
		WillReturnResult(sqlmock.NewResult(0, 5)) // 5 locks deleted

	count, err := manager.CleanupExpiredLocks(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLockManager_CleanupExpiredLocks_NoLocks(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()

	// Mock DELETE expired locks (returns 0 rows affected)
	sqlMock.ExpectExec("DELETE FROM execution_locks WHERE expires_at < NOW\\(\\)").
		WillReturnResult(sqlmock.NewResult(0, 0))

	count, err := manager.CleanupExpiredLocks(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

