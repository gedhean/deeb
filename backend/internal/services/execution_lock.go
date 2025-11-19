package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

// ExecutionLockManager defines the interface for execution lock management
type ExecutionLockManager interface {
	// AcquireLock acquires execution lock for a database
	// Returns error if lock cannot be acquired (database already locked)
	AcquireLock(ctx context.Context, databaseID uuid.UUID, changePlanID uuid.UUID, executorID uuid.UUID) error

	// ReleaseLock releases execution lock for a database
	ReleaseLock(ctx context.Context, databaseID uuid.UUID) error

	// IsLocked checks if database is currently locked
	IsLocked(ctx context.Context, databaseID uuid.UUID) (bool, *ExecutionLock, error)

	// CleanupExpiredLocks removes expired locks (background job)
	CleanupExpiredLocks(ctx context.Context) (int, error)
}

// ExecutionLock represents a lock record
type ExecutionLock struct {
	ID               uuid.UUID `db:"id"`
	TargetDatabaseID uuid.UUID `db:"target_database_id"`
	ChangePlanID     uuid.UUID `db:"change_plan_id"`
	ExecutorID       uuid.UUID `db:"executor_id"`
	LockedAt         time.Time `db:"locked_at"`
	ExpiresAt        time.Time `db:"expires_at"`
}

// executionLockManager implements ExecutionLockManager
type executionLockManager struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewExecutionLockManager creates a new execution lock manager
func NewExecutionLockManager(db *sqlx.DB, logger *zap.Logger) ExecutionLockManager {
	return &executionLockManager{
		db:     db,
		logger: logger,
	}
}

// AcquireLock acquires execution lock for a database
// Uses SELECT FOR UPDATE NOWAIT for non-blocking lock acquisition
func (m *executionLockManager) AcquireLock(
	ctx context.Context,
	databaseID uuid.UUID,
	changePlanID uuid.UUID,
	executorID uuid.UUID,
) error {
	// Use transaction with SELECT FOR UPDATE NOWAIT
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	shouldRollback := true
	defer func() {
		if shouldRollback {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				m.logger.Warn("Failed to rollback transaction", zap.Error(rollbackErr))
			}
		}
	}()

	// Check if lock exists (non-blocking)
	var existingLock ExecutionLock
	err = tx.GetContext(
		ctx,
		&existingLock,
		"SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = $1 FOR UPDATE NOWAIT",
		databaseID,
	)

	if err == nil {
		// Lock exists - check if expired
		if time.Now().Before(existingLock.ExpiresAt) {
			return ErrDatabaseLocked
		}
		// Lock expired - delete it
		_, err = tx.ExecContext(ctx,
			"DELETE FROM execution_locks WHERE target_database_id = $1",
			databaseID,
		)
		if err != nil {
			return fmt.Errorf("failed to cleanup expired lock: %w", err)
		}
	} else if err != sql.ErrNoRows {
		// Check if error is lock contention (PostgreSQL error code 55P03)
		if isLockContentionError(err) {
			return ErrDatabaseLocked
		}
		return fmt.Errorf("failed to check lock: %w", err)
	}

	// Insert new lock record
	expiresAt := time.Now().Add(1 * time.Hour)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO execution_locks (target_database_id, change_plan_id, executor_id, expires_at)
         VALUES ($1, $2, $3, $4)`,
		databaseID, changePlanID, executorID, expiresAt,
	)
	if err != nil {
		// Handle unique constraint violation (race condition)
		if isUniqueViolationError(err) {
			return ErrDatabaseLocked
		}
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit lock: %w", err)
	}

	// Mark that we don't need to rollback (commit succeeded)
	shouldRollback = false

	m.logger.Info("Execution lock acquired",
		zap.String("database_id", databaseID.String()),
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("executor_id", executorID.String()),
	)

	return nil
}

// ReleaseLock releases execution lock for a database
func (m *executionLockManager) ReleaseLock(
	ctx context.Context,
	databaseID uuid.UUID,
) error {
	result, err := m.db.ExecContext(ctx,
		"DELETE FROM execution_locks WHERE target_database_id = $1",
		databaseID,
	)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		m.logger.Warn("Attempted to release non-existent lock", zap.String("database_id", databaseID.String()))
	} else {
		m.logger.Info("Execution lock released", zap.String("database_id", databaseID.String()))
	}

	return nil
}

// IsLocked checks if database is currently locked
func (m *executionLockManager) IsLocked(ctx context.Context, databaseID uuid.UUID) (bool, *ExecutionLock, error) {
	var lock ExecutionLock
	err := m.db.GetContext(
		ctx,
		&lock,
		"SELECT id, target_database_id, change_plan_id, executor_id, locked_at, expires_at FROM execution_locks WHERE target_database_id = $1",
		databaseID,
	)

	if err == sql.ErrNoRows {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to check lock status: %w", err)
	}

	// Check if lock is expired
	if time.Now().After(lock.ExpiresAt) {
		// Lock expired - clean it up
		if releaseErr := m.ReleaseLock(ctx, databaseID); releaseErr != nil {
			m.logger.Warn("Failed to cleanup expired lock", zap.Error(releaseErr))
		}
		return false, nil, nil
	}

	return true, &lock, nil
}

// CleanupExpiredLocks removes expired locks (background job)
func (m *executionLockManager) CleanupExpiredLocks(ctx context.Context) (int, error) {
	result, err := m.db.ExecContext(ctx,
		"DELETE FROM execution_locks WHERE expires_at < NOW()",
	)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired locks: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		m.logger.Info("Cleaned up expired execution locks", zap.Int64("count", rowsAffected))
	}

	return int(rowsAffected), nil
}

// isLockContentionError checks if error is PostgreSQL lock contention error (55P03)
func isLockContentionError(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "55P03" // lock_not_available
	}
	return false
}

// isUniqueViolationError checks if error is PostgreSQL unique constraint violation (23505)
func isUniqueViolationError(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		return pqErr.Code == "23505" // unique_violation
	}
	return false
}

// Execution lock errors
var (
	ErrDatabaseLocked           = fmt.Errorf("database is currently locked by another execution")
	ErrDatabaseNotFound         = fmt.Errorf("database not found")
	ErrDatabaseInstanceNotFound = fmt.Errorf("database instance not found")
)
