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
		ID:               uuid.New(),
		TargetDatabaseID: databaseID,
		ChangePlanID:     uuid.New(),
		ExecutorID:       uuid.New(),
		LockedAt:         time.Now().Add(-30 * time.Minute),
		ExpiresAt:        time.Now().Add(30 * time.Minute), // Not expired
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
		ID:               uuid.New(),
		TargetDatabaseID: databaseID,
		ChangePlanID:     uuid.New(),
		ExecutorID:       uuid.New(),
		LockedAt:         time.Now().Add(-2 * time.Hour),
		ExpiresAt:        time.Now().Add(-1 * time.Hour), // Expired
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
		ID:               uuid.New(),
		TargetDatabaseID: databaseID,
		ChangePlanID:     uuid.New(),
		ExecutorID:       uuid.New(),
		LockedAt:         time.Now().Add(-30 * time.Minute),
		ExpiresAt:        time.Now().Add(30 * time.Minute), // Not expired
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
		ID:               uuid.New(),
		TargetDatabaseID: databaseID,
		ChangePlanID:     uuid.New(),
		ExecutorID:       uuid.New(),
		LockedAt:         time.Now().Add(-2 * time.Hour),
		ExpiresAt:        time.Now().Add(-1 * time.Hour), // Expired
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

// TestExecutionLockManager_AcquireLock_TransactionBeginFailure tests transaction begin failure
func TestExecutionLockManager_AcquireLock_TransactionBeginFailure(t *testing.T) {
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

	// Mock transaction begin failure
	sqlMock.ExpectBegin().WillReturnError(sql.ErrConnDone)

	err = manager.AcquireLock(ctx, databaseID, changePlanID, executorID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to begin transaction")

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

// TestExecutionLockManager_AcquireLock_ExpiredLockCleanupFailure tests expired lock cleanup failure
func TestExecutionLockManager_AcquireLock_ExpiredLockCleanupFailure(t *testing.T) {
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
		ID:               uuid.New(),
		TargetDatabaseID: databaseID,
		ChangePlanID:     uuid.New(),
		ExecutorID:       uuid.New(),
		LockedAt:         time.Now().Add(-2 * time.Hour),
		ExpiresAt:        time.Now().Add(-1 * time.Hour), // Expired
	}

	rows := sqlmock.NewRows([]string{"id", "target_database_id", "change_plan_id", "executor_id", "locked_at", "expires_at"}).
		AddRow(expiredLock.ID, expiredLock.TargetDatabaseID, expiredLock.ChangePlanID, expiredLock.ExecutorID, expiredLock.LockedAt, expiredLock.ExpiresAt)

	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1 FOR UPDATE NOWAIT").
		WithArgs(databaseID).
		WillReturnRows(rows)

	// Mock DELETE expired lock failure
	sqlMock.ExpectExec("DELETE FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnError(sql.ErrConnDone)

	// Mock rollback
	sqlMock.ExpectRollback()

	err = manager.AcquireLock(ctx, databaseID, changePlanID, executorID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to cleanup expired lock")

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

// TestExecutionLockManager_AcquireLock_SelectQueryError tests non-pq error in SELECT
func TestExecutionLockManager_AcquireLock_SelectQueryError(t *testing.T) {
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

	// Mock SELECT FOR UPDATE NOWAIT (returns non-pq error)
	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1 FOR UPDATE NOWAIT").
		WithArgs(databaseID).
		WillReturnError(sql.ErrConnDone)

	// Mock rollback
	sqlMock.ExpectRollback()

	err = manager.AcquireLock(ctx, databaseID, changePlanID, executorID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check lock")

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

// TestExecutionLockManager_AcquireLock_InsertError tests non-unique violation error in INSERT
func TestExecutionLockManager_AcquireLock_InsertError(t *testing.T) {
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

	// Mock INSERT lock record (returns non-unique violation error)
	sqlMock.ExpectExec("INSERT INTO execution_locks").
		WithArgs(databaseID, changePlanID, executorID, sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	// Mock rollback
	sqlMock.ExpectRollback()

	err = manager.AcquireLock(ctx, databaseID, changePlanID, executorID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to acquire lock")

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

// TestExecutionLockManager_AcquireLock_CommitFailure tests commit failure
func TestExecutionLockManager_AcquireLock_CommitFailure(t *testing.T) {
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

	// Mock INSERT lock record
	sqlMock.ExpectExec("INSERT INTO execution_locks").
		WithArgs(databaseID, changePlanID, executorID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Mock commit failure
	sqlMock.ExpectCommit().WillReturnError(sql.ErrConnDone)

	err = manager.AcquireLock(ctx, databaseID, changePlanID, executorID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to commit lock")

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

// TestExecutionLockManager_ReleaseLock_ExecError tests ExecContext failure
func TestExecutionLockManager_ReleaseLock_ExecError(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()

	// Mock DELETE lock failure
	sqlMock.ExpectExec("DELETE FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnError(sql.ErrConnDone)

	err = manager.ReleaseLock(ctx, databaseID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to release lock")

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

// TestExecutionLockManager_ReleaseLock_RowsAffectedError tests RowsAffected failure
func TestExecutionLockManager_ReleaseLock_RowsAffectedError(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()

	// Mock DELETE lock (with RowsAffected error)
	result := sqlmock.NewErrorResult(sql.ErrConnDone)
	sqlMock.ExpectExec("DELETE FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnResult(result)

	err = manager.ReleaseLock(ctx, databaseID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get rows affected")

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

// TestExecutionLockManager_IsLocked_QueryError tests GetContext failure
func TestExecutionLockManager_IsLocked_QueryError(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()

	// Mock SELECT failure (non-ErrNoRows)
	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnError(sql.ErrConnDone)

	isLocked, lock, err := manager.IsLocked(ctx, databaseID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check lock status")
	assert.False(t, isLocked)
	assert.Nil(t, lock)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

// TestExecutionLockManager_IsLocked_ExpiredLockReleaseFailure tests ReleaseLock failure during cleanup
func TestExecutionLockManager_IsLocked_ExpiredLockReleaseFailure(t *testing.T) {
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
		ID:               uuid.New(),
		TargetDatabaseID: databaseID,
		ChangePlanID:     uuid.New(),
		ExecutorID:       uuid.New(),
		LockedAt:         time.Now().Add(-2 * time.Hour),
		ExpiresAt:        time.Now().Add(-1 * time.Hour), // Expired
	}

	rows := sqlmock.NewRows([]string{"id", "target_database_id", "change_plan_id", "executor_id", "locked_at", "expires_at"}).
		AddRow(expiredLock.ID, expiredLock.TargetDatabaseID, expiredLock.ChangePlanID, expiredLock.ExecutorID, expiredLock.LockedAt, expiredLock.ExpiresAt)

	// Mock SELECT (returns expired lock)
	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnRows(rows)

	// Mock DELETE expired lock failure
	sqlMock.ExpectExec("DELETE FROM execution_locks WHERE target_database_id = \\$1").
		WithArgs(databaseID).
		WillReturnError(sql.ErrConnDone)

	// Should still return false, nil (cleanup failure logged but doesn't fail the check)
	isLocked, lock, err := manager.IsLocked(ctx, databaseID)
	require.NoError(t, err) // Error is logged but not returned
	assert.False(t, isLocked)
	assert.Nil(t, lock)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

// TestExecutionLockManager_CleanupExpiredLocks_ExecError tests ExecContext failure
func TestExecutionLockManager_CleanupExpiredLocks_ExecError(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()

	// Mock DELETE expired locks failure
	sqlMock.ExpectExec("DELETE FROM execution_locks WHERE expires_at < NOW\\(\\)").
		WillReturnError(sql.ErrConnDone)

	count, err := manager.CleanupExpiredLocks(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to cleanup expired locks")
	assert.Equal(t, 0, count)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

// TestExecutionLockManager_CleanupExpiredLocks_RowsAffectedError tests RowsAffected failure
func TestExecutionLockManager_CleanupExpiredLocks_RowsAffectedError(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()

	// Mock DELETE expired locks (with RowsAffected error)
	result := sqlmock.NewErrorResult(sql.ErrConnDone)
	sqlMock.ExpectExec("DELETE FROM execution_locks WHERE expires_at < NOW\\(\\)").
		WillReturnResult(result)

	count, err := manager.CleanupExpiredLocks(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get rows affected")
	assert.Equal(t, 0, count)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}

// TestIsLockContentionError tests isLockContentionError helper function
func TestIsLockContentionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "pq lock contention error",
			err:      &pq.Error{Code: "55P03"},
			expected: true,
		},
		{
			name:     "pq non-lock contention error",
			err:      &pq.Error{Code: "23505"},
			expected: false,
		},
		{
			name:     "non-pq error",
			err:      sql.ErrConnDone,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLockContentionError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsUniqueViolationError tests isUniqueViolationError helper function
func TestIsUniqueViolationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "pq unique violation error",
			err:      &pq.Error{Code: "23505"},
			expected: true,
		},
		{
			name:     "pq non-unique violation error",
			err:      &pq.Error{Code: "55P03"},
			expected: false,
		},
		{
			name:     "non-pq error",
			err:      sql.ErrConnDone,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isUniqueViolationError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExecutionLockManager_ConcurrentLockAcquisition tests concurrent lock acquisition attempts
func TestExecutionLockManager_ConcurrentLockAcquisition(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	logger := zap.NewNop()
	manager := NewExecutionLockManager(db, logger)

	ctx := context.Background()
	databaseID := uuid.New()
	changePlanID1 := uuid.New()
	changePlanID2 := uuid.New()
	executorID1 := uuid.New()
	executorID2 := uuid.New()

	// First acquisition will succeed
	sqlMock.ExpectBegin()
	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1 FOR UPDATE NOWAIT").
		WithArgs(databaseID).
		WillReturnError(sql.ErrNoRows)
	sqlMock.ExpectExec("INSERT INTO execution_locks").
		WithArgs(databaseID, changePlanID1, executorID1, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	// Second acquisition will fail (database locked)
	sqlMock.ExpectBegin()
	existingLock := ExecutionLock{
		ID:               uuid.New(),
		TargetDatabaseID: databaseID,
		ChangePlanID:     changePlanID1,
		ExecutorID:       executorID1,
		LockedAt:         time.Now().Add(-30 * time.Minute),
		ExpiresAt:        time.Now().Add(30 * time.Minute),
	}
	rows := sqlmock.NewRows([]string{"id", "target_database_id", "change_plan_id", "executor_id", "locked_at", "expires_at"}).
		AddRow(existingLock.ID, existingLock.TargetDatabaseID, existingLock.ChangePlanID, existingLock.ExecutorID, existingLock.LockedAt, existingLock.ExpiresAt)
	sqlMock.ExpectQuery("SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = \\$1 FOR UPDATE NOWAIT").
		WithArgs(databaseID).
		WillReturnRows(rows)
	sqlMock.ExpectRollback()

	// Execute first acquisition
	err1 := manager.AcquireLock(ctx, databaseID, changePlanID1, executorID1)
	require.NoError(t, err1)

	// Execute second acquisition (should fail)
	err2 := manager.AcquireLock(ctx, databaseID, changePlanID2, executorID2)
	assert.Error(t, err2)
	assert.Equal(t, ErrDatabaseLocked, err2)

	require.NoError(t, sqlMock.ExpectationsWereMet())
}
