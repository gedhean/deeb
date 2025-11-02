// Package services provides business logic services for the Deeb application.
package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// ConnectionManagerInterface defines the interface for database connection management
type ConnectionManagerInterface interface {
	Connect(ctx context.Context, params database.ConnectionParams) (*sqlx.DB, error)
}

// ExecutionService provides SQL execution functionality with transaction management
type ExecutionService struct {
	changePlanRepo  repository.ChangePlanRepository
	dbInstanceRepo  repository.DatabaseInstanceRepository
	databaseRepo    repository.DatabaseRepository
	snapshotService *SchemaSnapshotService
	lockManager     ExecutionLockManager
	connectionMgr   ConnectionManagerInterface
	encryptionKey   string
	logger          *zap.Logger
}

// ExecutionResult represents the result of a SQL execution
type ExecutionResult struct {
	Success      bool
	RowsAffected int
	Duration     time.Duration
	ErrorMessage *string
}

// NewExecutionService creates a new execution service
func NewExecutionService(
	changePlanRepo repository.ChangePlanRepository,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	snapshotService *SchemaSnapshotService,
	lockManager ExecutionLockManager,
	connectionMgr ConnectionManagerInterface,
	encryptionKey string,
	logger *zap.Logger,
) *ExecutionService {
	return &ExecutionService{
		changePlanRepo:  changePlanRepo,
		dbInstanceRepo:  dbInstanceRepo,
		databaseRepo:    databaseRepo,
		snapshotService: snapshotService,
		lockManager:     lockManager,
		connectionMgr:   connectionMgr,
		encryptionKey:   encryptionKey,
		logger:          logger,
	}
}

// Execute executes SQL statements from a change plan within a SERIALIZABLE transaction
// Implements complete workflow: lock → snapshot → execute → snapshot → update status → unlock
func (s *ExecutionService) Execute(
	ctx context.Context,
	changePlanID uuid.UUID,
	executorID uuid.UUID,
) (*ExecutionResult, error) {
	// Get change plan
	changePlan, err := s.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrChangePlanNotFound
		}
		return nil, fmt.Errorf("failed to get change plan: %w", err)
	}

	// Validate Change Plan is in Approved status
	if changePlan.Status != models.ChangePlanStatusApproved {
		return nil, fmt.Errorf("change plan must be in Approved status to execute, current status: %s", changePlan.Status)
	}

	// Get database and database instance
	db, err := s.databaseRepo.GetByID(ctx, changePlan.TargetDatabaseID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrDatabaseNotFound
		}
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	dbInstance, err := s.dbInstanceRepo.GetByID(ctx, db.DatabaseInstanceID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrDatabaseInstanceNotFound
		}
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Step 0: Acquire execution lock for database
	if err := s.lockManager.AcquireLock(ctx, db.ID, changePlanID, executorID); err != nil {
		if err == ErrDatabaseLocked {
			return nil, ErrDatabaseLocked
		}
		return nil, fmt.Errorf("failed to acquire execution lock: %w", err)
	}

	// Guarantee lock release on exit (success or failure)
	defer func() {
		if releaseErr := s.lockManager.ReleaseLock(ctx, db.ID); releaseErr != nil {
			s.logger.Error("Failed to release execution lock", zap.Error(releaseErr))
		}
	}()

	// Step 1: Update status to InProgress
	changePlan.Status = models.ChangePlanStatusInProgress
	changePlan.UpdatedAt = time.Now()
	if err := s.changePlanRepo.Update(ctx, changePlan); err != nil {
		return nil, fmt.Errorf("failed to update change plan status to InProgress: %w", err)
	}

	// Step 2: Capture before snapshot
	s.logger.Info("Capturing before snapshot",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
	)

	var beforeSnapshot *models.SchemaSnapshot
	switch dbInstance.DatabaseType {
	case models.DatabaseTypePostgreSQL:
		beforeSnapshot, err = s.snapshotService.CapturePostgreSQL(ctx, changePlanID, db.ID, models.SnapshotTypeBefore)
	case models.DatabaseTypeMySQL:
		beforeSnapshot, err = s.snapshotService.CaptureMySQL(ctx, changePlanID, db.ID, models.SnapshotTypeBefore)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbInstance.DatabaseType)
	}

	if err != nil {
		s.logger.Error("Failed to capture before snapshot",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		// Update status to Failed
		changePlan.Status = models.ChangePlanStatusFailed
		errorMsg := fmt.Sprintf("failed to capture before snapshot: %v", err)
		changePlan.ErrorMessage = &errorMsg
		changePlan.UpdatedAt = time.Now()
		if updateErr := s.changePlanRepo.Update(ctx, changePlan); updateErr != nil {
			s.logger.Error("Failed to update change plan status to Failed", zap.Error(updateErr))
		}
		return nil, fmt.Errorf("failed to capture before snapshot: %w", err)
	}

	s.logger.Info("Before snapshot captured successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("snapshot_id", beforeSnapshot.ID.String()),
	)

	// Step 3: Execute SQL in transaction
	startTime := time.Now()
	result, err := s.executeInTransaction(ctx, dbInstance, db, changePlan.SQLContent)
	duration := time.Since(startTime)

	// Store execution results in Change Plan
	now := time.Now()
	changePlan.ExecutedAt = &now
	durationMs := int(duration.Milliseconds())
	changePlan.ExecutionDurationMs = &durationMs
	changePlan.UpdatedAt = time.Now()

	if err != nil {
		// Step 4: Execution failed - update status to Failed
		errorMsg := err.Error()
		changePlan.Status = models.ChangePlanStatusFailed
		changePlan.ErrorMessage = &errorMsg
		changePlan.AffectedRows = nil // No rows affected on error

		s.logger.Error("Execution failed",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", db.ID.String()),
			zap.Error(err),
			zap.Duration("duration", duration),
		)

		// Update Change Plan with error results
		if updateErr := s.changePlanRepo.Update(ctx, changePlan); updateErr != nil {
			s.logger.Error("Failed to update change plan with execution results", zap.Error(updateErr))
			// Don't fail the whole operation if update fails, but log it
		}

		return &ExecutionResult{
			Success:      false,
			RowsAffected: 0,
			Duration:     duration,
			ErrorMessage: &errorMsg,
		}, nil // Return result with error, don't propagate error
	}

	// Step 5: Execution succeeded - capture after snapshot
	s.logger.Info("Capturing after snapshot",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
	)

	var afterSnapshot *models.SchemaSnapshot
	switch dbInstance.DatabaseType {
	case models.DatabaseTypePostgreSQL:
		afterSnapshot, err = s.snapshotService.CapturePostgreSQL(ctx, changePlanID, db.ID, models.SnapshotTypeAfter)
	case models.DatabaseTypeMySQL:
		afterSnapshot, err = s.snapshotService.CaptureMySQL(ctx, changePlanID, db.ID, models.SnapshotTypeAfter)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbInstance.DatabaseType)
	}

	if err != nil {
		s.logger.Error("Failed to capture after snapshot",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		// Update status to Failed even though SQL executed successfully
		// This is a safety measure - we want to know if snapshot capture failed
		changePlan.Status = models.ChangePlanStatusFailed
		errorMsg := fmt.Sprintf("SQL executed successfully but failed to capture after snapshot: %v", err)
		changePlan.ErrorMessage = &errorMsg
		changePlan.UpdatedAt = time.Now()
		if updateErr := s.changePlanRepo.Update(ctx, changePlan); updateErr != nil {
			s.logger.Error("Failed to update change plan status to Failed", zap.Error(updateErr))
		}
		return nil, fmt.Errorf("failed to capture after snapshot: %w", err)
	}

	s.logger.Info("After snapshot captured successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("snapshot_id", afterSnapshot.ID.String()),
	)

	// Step 6: Update status to Completed
	changePlan.Status = models.ChangePlanStatusCompleted
	changePlan.AffectedRows = &result.RowsAffected
	changePlan.ErrorMessage = nil // Clear any previous error message
	changePlan.UpdatedAt = time.Now()

	// Update Change Plan with execution results
	if err := s.changePlanRepo.Update(ctx, changePlan); err != nil {
		return nil, fmt.Errorf("failed to update change plan with execution results: %w", err)
	}

	s.logger.Info("Execution workflow completed successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.Int("rows_affected", result.RowsAffected),
		zap.Duration("duration", duration),
	)

	return result, nil
}

// executeInTransaction executes SQL statements within a SERIALIZABLE transaction
func (s *ExecutionService) executeInTransaction(
	ctx context.Context,
	dbInstance *models.DatabaseInstance,
	db *models.Database,
	sqlContent string,
) (*ExecutionResult, error) {
	// Get database connection
	dbConn, err := s.connectionMgr.Connect(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if err := dbConn.Close(); err != nil {
			s.logger.Warn("Failed to close database connection", zap.Error(err))
		}
	}()

	// Begin transaction with SERIALIZABLE isolation
	tx, err := dbConn.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Track if we should rollback
	shouldRollback := true
	defer func() {
		if shouldRollback {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			}
		}
	}()

	// Execute SQL statements
	startTime := time.Now()
	result, err := tx.ExecContext(ctx, sqlContent)
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Get rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Mark that we don't need to rollback (commit succeeded)
	shouldRollback = false

	duration := time.Since(startTime)

	return &ExecutionResult{
		Success:      true,
		RowsAffected: int(rowsAffected),
		Duration:     duration,
	}, nil
}
