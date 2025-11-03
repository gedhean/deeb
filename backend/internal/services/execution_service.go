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
	changePlanRepo   repository.ChangePlanRepository
	dbInstanceRepo   repository.DatabaseInstanceRepository
	databaseRepo     repository.DatabaseRepository
	snapshotService  *SchemaSnapshotService
	lockManager      ExecutionLockManager
	queueService     *ExecutionQueueService
	executionLogRepo repository.ExecutionLogRepository
	connectionMgr    ConnectionManagerInterface
	encryptionKey    string
	executionTimeout time.Duration // Timeout for SQL execution (default: 5 minutes)
	logger           *zap.Logger
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
	queueService *ExecutionQueueService,
	executionLogRepo repository.ExecutionLogRepository,
	connectionMgr ConnectionManagerInterface,
	encryptionKey string,
	logger *zap.Logger,
) *ExecutionService {
	executionTimeout := 5 * time.Minute // Default: 5 minutes

	return &ExecutionService{
		changePlanRepo:   changePlanRepo,
		dbInstanceRepo:   dbInstanceRepo,
		databaseRepo:     databaseRepo,
		snapshotService:  snapshotService,
		lockManager:      lockManager,
		queueService:     queueService,
		executionLogRepo: executionLogRepo,
		connectionMgr:    connectionMgr,
		encryptionKey:    encryptionKey,
		executionTimeout: executionTimeout,
		logger:           logger,
	}
}

// NewExecutionServiceWithTimeout creates a new execution service with custom timeout
func NewExecutionServiceWithTimeout(
	changePlanRepo repository.ChangePlanRepository,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	snapshotService *SchemaSnapshotService,
	lockManager ExecutionLockManager,
	queueService *ExecutionQueueService,
	executionLogRepo repository.ExecutionLogRepository,
	connectionMgr ConnectionManagerInterface,
	encryptionKey string,
	executionTimeout time.Duration,
	logger *zap.Logger,
) *ExecutionService {
	return &ExecutionService{
		changePlanRepo:   changePlanRepo,
		dbInstanceRepo:   dbInstanceRepo,
		databaseRepo:     databaseRepo,
		snapshotService:  snapshotService,
		lockManager:      lockManager,
		queueService:     queueService,
		executionLogRepo: executionLogRepo,
		connectionMgr:    connectionMgr,
		encryptionKey:    encryptionKey,
		executionTimeout: executionTimeout,
		logger:           logger,
	}
}

// Execute executes SQL statements from a change plan within a SERIALIZABLE transaction
// Implements complete workflow: lock → snapshot → execute → snapshot → update status → unlock
func (s *ExecutionService) Execute(
	ctx context.Context,
	changePlanID uuid.UUID,
	executorID uuid.UUID,
) (*ExecutionResult, error) {
	// Log execution start
	overallStartTime := time.Now()
	s.logger.Info("Execution started",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("executor_id", executorID.String()),
	)

	// Get change plan
	changePlan, err := s.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		if err == repository.ErrNotFound {
			s.logger.Error("Change plan not found",
				zap.String("change_plan_id", changePlanID.String()),
			)
			return nil, ErrChangePlanNotFound
		}
		s.logger.Error("Failed to get change plan",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get change plan: %w", err)
	}

	// Validate Change Plan is in Approved status
	if changePlan.Status != models.ChangePlanStatusApproved {
		s.logger.Error("Change plan not in Approved status",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("current_status", string(changePlan.Status)),
		)
		return nil, fmt.Errorf("change plan must be in Approved status to execute, current status: %s", changePlan.Status)
	}

	// Get database and database instance
	db, err := s.databaseRepo.GetByID(ctx, changePlan.TargetDatabaseID)
	if err != nil {
		if err == repository.ErrNotFound {
			s.logger.Error("Target database not found",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", changePlan.TargetDatabaseID.String()),
			)
			return nil, ErrDatabaseNotFound
		}
		s.logger.Error("Failed to get database",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", changePlan.TargetDatabaseID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	dbInstance, err := s.dbInstanceRepo.GetByID(ctx, db.DatabaseInstanceID)
	if err != nil {
		if err == repository.ErrNotFound {
			s.logger.Error("Database instance not found",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_instance_id", db.DatabaseInstanceID.String()),
			)
			return nil, ErrDatabaseInstanceNotFound
		}
		s.logger.Error("Failed to get database instance",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_instance_id", db.DatabaseInstanceID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Log database context for monitoring
	sqlContentLength := len(changePlan.SQLContent)
	s.logger.Info("Execution context loaded",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_name", db.DatabaseName),
		zap.String("database_instance_id", dbInstance.ID.String()),
		zap.String("database_instance_name", dbInstance.Name),
		zap.String("database_type", string(dbInstance.DatabaseType)),
		zap.Int("sql_content_length", sqlContentLength),
	)

	// Step 0: Acquire execution lock for database
	lockStartTime := time.Now()
	if err := s.lockManager.AcquireLock(ctx, db.ID, changePlanID, executorID); err != nil {
		if err == ErrDatabaseLocked {
			// Database is locked - enqueue the Change Plan
			if s.queueService != nil {
				queueItem, enqueueErr := s.queueService.Enqueue(ctx, changePlanID, db.ID, executorID)
				if enqueueErr != nil {
					s.logger.Error("Failed to enqueue change plan",
						zap.String("change_plan_id", changePlanID.String()),
						zap.String("database_id", db.ID.String()),
						zap.Error(enqueueErr),
					)
					return nil, fmt.Errorf("database locked and failed to enqueue: %w", enqueueErr)
				}
				s.logger.Info("Change plan enqueued due to database lock",
					zap.String("change_plan_id", changePlanID.String()),
					zap.String("database_id", db.ID.String()),
					zap.Int("position", queueItem.Position),
				)
				return nil, ErrDatabaseLocked // Return lock error so caller knows it was queued
			}
			// No queue service available - return lock error
			return nil, ErrDatabaseLocked
		}
		s.logger.Error("Failed to acquire execution lock",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", db.ID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to acquire execution lock: %w", err)
	}
	lockDuration := time.Since(lockStartTime)
	s.logger.Info("Execution lock acquired",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.Duration("lock_acquisition_duration", lockDuration),
	)

	// Guarantee lock release on exit (success or failure)
	defer func() {
		if releaseErr := s.lockManager.ReleaseLock(ctx, db.ID); releaseErr != nil {
			s.logger.Error("Failed to release execution lock",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
				zap.Error(releaseErr),
			)
		} else {
			s.logger.Debug("Execution lock released",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
			)
		}
	}()

	// Step 1: Update status to InProgress
	changePlan.Status = models.ChangePlanStatusInProgress
	changePlan.UpdatedAt = time.Now()
	if err := s.changePlanRepo.Update(ctx, changePlan); err != nil {
		s.logger.Error("Failed to update change plan status to InProgress",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to update change plan status to InProgress: %w", err)
	}
	s.logger.Info("Change plan status updated to InProgress",
		zap.String("change_plan_id", changePlanID.String()),
	)

	// Step 2: Capture before snapshot
	snapshotBeforeStartTime := time.Now()
	s.logger.Info("Capturing before snapshot",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_type", string(dbInstance.DatabaseType)),
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

	snapshotBeforeDuration := time.Since(snapshotBeforeStartTime)
	s.logger.Info("Before snapshot captured successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("snapshot_id", beforeSnapshot.ID.String()),
		zap.Duration("snapshot_capture_duration", snapshotBeforeDuration),
	)

	// Step 3: Execute SQL in transaction with timeout
	executionStartTime := time.Now()
	s.logger.Info("Starting SQL execution",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_type", string(dbInstance.DatabaseType)),
		zap.Int("sql_content_length", sqlContentLength),
		zap.Duration("execution_timeout", s.executionTimeout),
	)

	// Create timeout context for SQL execution
	execCtx, cancel := context.WithTimeout(ctx, s.executionTimeout)
	defer cancel()

	result, err := s.executeInTransaction(execCtx, dbInstance, db, changePlan.SQLContent, changePlanID)
	duration := time.Since(executionStartTime)

	// Check if timeout occurred
	if err != nil && execCtx.Err() == context.DeadlineExceeded {
		s.logger.Error("Execution timeout exceeded",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Duration("timeout", s.executionTimeout),
			zap.Duration("actual_duration", duration),
		)
		err = fmt.Errorf("execution timeout after %v: %w", s.executionTimeout, err)
	}

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

		// Enhanced error context logging
		sqlPreview := changePlan.SQLContent
		if len(sqlPreview) > 200 {
			sqlPreview = sqlPreview[:200] + "... (truncated)"
		}
		s.logger.Error("SQL execution failed",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", db.ID.String()),
			zap.String("database_name", db.DatabaseName),
			zap.String("database_instance_name", dbInstance.Name),
			zap.String("database_type", string(dbInstance.DatabaseType)),
			zap.Error(err),
			zap.Duration("execution_duration", duration),
			zap.Int("sql_content_length", sqlContentLength),
			zap.String("sql_preview", sqlPreview),
		)

		// Update Change Plan with error results
		if updateErr := s.changePlanRepo.Update(ctx, changePlan); updateErr != nil {
			s.logger.Error("Failed to update change plan with execution results", zap.Error(updateErr))
			// Don't fail the whole operation if update fails, but log it
		}

		// Create execution log entry for failed execution
		if s.executionLogRepo != nil {
			executionLog := &models.ExecutionLog{
				ChangePlanID:        changePlanID,
				ExecutorID:          executorID,
				Status:              models.ExecutionLogStatusFailed,
				ExecutionDurationMs: int(duration.Milliseconds()),
				RowsAffected:        nil,
				ErrorMessage:        &errorMsg,
				StartedAt:           executionStartTime,
				CompletedAt:         time.Now(),
			}
			if logErr := s.executionLogRepo.Create(ctx, executionLog); logErr != nil {
				s.logger.Warn("Failed to create execution log entry", zap.Error(logErr))
			}
		}

		return &ExecutionResult{
			Success:      false,
			RowsAffected: 0,
			Duration:     duration,
			ErrorMessage: &errorMsg,
		}, nil // Return result with error, don't propagate error
	}

	// Step 5: Execution succeeded - capture after snapshot
	s.logger.Info("SQL execution completed successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.Int("rows_affected", result.RowsAffected),
		zap.Duration("execution_duration", duration),
	)

	snapshotAfterStartTime := time.Now()
	s.logger.Info("Capturing after snapshot",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_type", string(dbInstance.DatabaseType)),
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

	snapshotAfterDuration := time.Since(snapshotAfterStartTime)
	s.logger.Info("After snapshot captured successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("snapshot_id", afterSnapshot.ID.String()),
		zap.Duration("snapshot_capture_duration", snapshotAfterDuration),
	)

	// Step 6: Update status to Completed
	changePlan.Status = models.ChangePlanStatusCompleted
	changePlan.AffectedRows = &result.RowsAffected
	changePlan.ErrorMessage = nil // Clear any previous error message
	changePlan.UpdatedAt = time.Now()

	// Update Change Plan with execution results
	if err := s.changePlanRepo.Update(ctx, changePlan); err != nil {
		s.logger.Error("Failed to update change plan with execution results",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to update change plan with execution results: %w", err)
	}

	overallDuration := time.Since(overallStartTime)
	s.logger.Info("Execution workflow completed successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_name", db.DatabaseName),
		zap.String("database_instance_name", dbInstance.Name),
		zap.String("database_type", string(dbInstance.DatabaseType)),
		zap.Int("rows_affected", result.RowsAffected),
		zap.Duration("sql_execution_duration", duration),
		zap.Duration("before_snapshot_duration", snapshotBeforeDuration),
		zap.Duration("after_snapshot_duration", snapshotAfterDuration),
		zap.Duration("lock_acquisition_duration", lockDuration),
		zap.Duration("total_execution_duration", overallDuration),
	)

	// Create execution log entry for successful execution
	if s.executionLogRepo != nil {
		executionLog := &models.ExecutionLog{
			ChangePlanID:        changePlanID,
			ExecutorID:          executorID,
			Status:              models.ExecutionLogStatusCompleted,
			ExecutionDurationMs: int(duration.Milliseconds()),
			RowsAffected:        &result.RowsAffected,
			ErrorMessage:        nil,
			StartedAt:           executionStartTime,
			CompletedAt:         time.Now(),
		}
		if logErr := s.executionLogRepo.Create(ctx, executionLog); logErr != nil {
			s.logger.Warn("Failed to create execution log entry", zap.Error(logErr))
		}
	}

	return result, nil
}

// GetQueueStatus returns the queue status for a Change Plan
func (s *ExecutionService) GetQueueStatus(ctx context.Context, changePlanID uuid.UUID) (*models.ExecutionQueue, error) {
	if s.queueService == nil {
		return nil, fmt.Errorf("queue service not available")
	}
	return s.queueService.GetQueueStatus(ctx, changePlanID)
}

// GetExecutionLogHistory returns the execution log history for a Change Plan
func (s *ExecutionService) GetExecutionLogHistory(ctx context.Context, changePlanID uuid.UUID) ([]*models.ExecutionLog, error) {
	if s.executionLogRepo == nil {
		return nil, fmt.Errorf("execution log repository not available")
	}
	return s.executionLogRepo.GetByChangePlanID(ctx, changePlanID)
}

// executeInTransaction executes SQL statements within a SERIALIZABLE transaction
func (s *ExecutionService) executeInTransaction(
	ctx context.Context,
	dbInstance *models.DatabaseInstance,
	db *models.Database,
	sqlContent string,
	changePlanID uuid.UUID,
) (*ExecutionResult, error) {
	// Get database connection
	connectStartTime := time.Now()
	dbConn, err := s.connectionMgr.Connect(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		s.logger.Error("Failed to connect to database",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", db.ID.String()),
			zap.String("database_name", db.DatabaseName),
			zap.String("database_instance_name", dbInstance.Name),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	connectDuration := time.Since(connectStartTime)
	s.logger.Debug("Database connection established",
		zap.String("change_plan_id", changePlanID.String()),
		zap.Duration("connection_duration", connectDuration),
	)

	defer func() {
		if err := dbConn.Close(); err != nil {
			s.logger.Warn("Failed to close database connection",
				zap.String("change_plan_id", changePlanID.String()),
				zap.Error(err),
			)
		}
	}()

	// Begin transaction with SERIALIZABLE isolation
	txStartTime := time.Now()
	tx, err := dbConn.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		s.logger.Error("Failed to begin transaction",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("isolation_level", "SERIALIZABLE"),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	txBeginDuration := time.Since(txStartTime)
	s.logger.Debug("Transaction begun",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("isolation_level", "SERIALIZABLE"),
		zap.Duration("transaction_begin_duration", txBeginDuration),
	)

	// Track if we should rollback
	shouldRollback := true
	defer func() {
		if shouldRollback {
			rollbackStartTime := time.Now()
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Error("Failed to rollback transaction",
					zap.String("change_plan_id", changePlanID.String()),
					zap.Duration("rollback_duration", time.Since(rollbackStartTime)),
					zap.Error(rollbackErr),
				)
			} else {
				s.logger.Info("Transaction rolled back",
					zap.String("change_plan_id", changePlanID.String()),
					zap.Duration("rollback_duration", time.Since(rollbackStartTime)),
				)
			}
		}
	}()

	// Execute SQL statements
	execStartTime := time.Now()
	s.logger.Debug("Executing SQL statements",
		zap.String("change_plan_id", changePlanID.String()),
		zap.Int("sql_content_length", len(sqlContent)),
	)
	result, err := tx.ExecContext(ctx, sqlContent)
	execDuration := time.Since(execStartTime)
	if err != nil {
		s.logger.Error("SQL execution failed",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Duration("execution_duration", execDuration),
			zap.Error(err),
		)
		return nil, fmt.Errorf("execution failed: %w", err)
	}

	// Get rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.logger.Error("Failed to get rows affected",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Commit transaction
	commitStartTime := time.Now()
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Int64("rows_affected", rowsAffected),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	commitDuration := time.Since(commitStartTime)
	s.logger.Debug("Transaction committed successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.Int64("rows_affected", rowsAffected),
		zap.Duration("commit_duration", commitDuration),
	)

	// Mark that we don't need to rollback (commit succeeded)
	shouldRollback = false

	duration := time.Since(execStartTime)

	return &ExecutionResult{
		Success:      true,
		RowsAffected: int(rowsAffected),
		Duration:     duration,
	}, nil
}
