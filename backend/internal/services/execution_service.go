// Package services provides business logic services for the Deeb application.
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/parser"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// ConnectionManagerInterface defines the interface for database connection management
type ConnectionManagerInterface interface {
	Connect(ctx context.Context, params database.ConnectionParams) (*sqlx.DB, error)
}

// ExecutionService provides SQL execution functionality with transaction management
type ExecutionService struct {
	changePlanRepo       repository.ChangePlanRepository
	dbInstanceRepo       repository.DatabaseInstanceRepository
	databaseRepo         repository.DatabaseRepository
	changelogRepo        repository.ChangelogRepository
	schemaHistoryRepo    repository.SchemaHistoryRepository
	schemaVersionService *SchemaVersionService
	lockManager          ExecutionLockManager
	queueService         *ExecutionQueueService
	executionLogRepo     repository.ExecutionLogRepository
	statementResultRepo  repository.StatementExecutionResultRepository
	connectionMgr        ConnectionManagerInterface
	encryptionKey        string
	executionTimeout     time.Duration // Timeout for SQL execution (default: 10 minutes)
	maxStatementCount    int           // Maximum number of statements allowed (default: 1000)
	logger               *zap.Logger
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
	changelogRepo repository.ChangelogRepository,
	schemaHistoryRepo repository.SchemaHistoryRepository,
	schemaVersionService *SchemaVersionService,
	lockManager ExecutionLockManager,
	queueService *ExecutionQueueService,
	executionLogRepo repository.ExecutionLogRepository,
	statementResultRepo repository.StatementExecutionResultRepository,
	connectionMgr ConnectionManagerInterface,
	encryptionKey string,
	logger *zap.Logger,
) *ExecutionService {
	executionTimeout := 10 * time.Minute // Default: 10 minutes
	maxStatementCount := 1000           // Default: 1000 statements max

	return &ExecutionService{
		changePlanRepo:       changePlanRepo,
		dbInstanceRepo:       dbInstanceRepo,
		databaseRepo:         databaseRepo,
		changelogRepo:        changelogRepo,
		schemaHistoryRepo:    schemaHistoryRepo,
		schemaVersionService: schemaVersionService,
		lockManager:          lockManager,
		queueService:         queueService,
		executionLogRepo:     executionLogRepo,
		statementResultRepo:  statementResultRepo,
		connectionMgr:        connectionMgr,
		encryptionKey:        encryptionKey,
		executionTimeout:     executionTimeout,
		maxStatementCount:    maxStatementCount,
		logger:               logger,
	}
}

// NewExecutionServiceWithTimeout creates a new execution service with custom timeout
func NewExecutionServiceWithTimeout(
	changePlanRepo repository.ChangePlanRepository,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	changelogRepo repository.ChangelogRepository,
	schemaHistoryRepo repository.SchemaHistoryRepository,
	schemaVersionService *SchemaVersionService,
	lockManager ExecutionLockManager,
	queueService *ExecutionQueueService,
	executionLogRepo repository.ExecutionLogRepository,
	statementResultRepo repository.StatementExecutionResultRepository,
	connectionMgr ConnectionManagerInterface,
	encryptionKey string,
	executionTimeout time.Duration,
	logger *zap.Logger,
) *ExecutionService {
	maxStatementCount := 1000 // Default: 1000 statements max

	return &ExecutionService{
		changePlanRepo:       changePlanRepo,
		dbInstanceRepo:       dbInstanceRepo,
		databaseRepo:         databaseRepo,
		changelogRepo:        changelogRepo,
		schemaHistoryRepo:    schemaHistoryRepo,
		schemaVersionService: schemaVersionService,
		lockManager:          lockManager,
		queueService:         queueService,
		executionLogRepo:     executionLogRepo,
		statementResultRepo:  statementResultRepo,
		connectionMgr:        connectionMgr,
		encryptionKey:        encryptionKey,
		executionTimeout:     executionTimeout,
		maxStatementCount:    maxStatementCount,
		logger:               logger,
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
		zap.String("database_engine", string(dbInstance.DatabaseEngine)),
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

	// Step 2: Get latest schema version and create changelog entry
	var prevSchemaHistoryID *uuid.UUID
	if db.Status != models.DatabaseStatusPending {
		// Get the latest schema version for this database (if exists)
		latestSchema, err := s.schemaVersionService.GetLatestSchemaVersion(ctx, db.ID)
		if err != nil {
			s.logger.Error("Failed to get latest schema version",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
				zap.Error(err),
			)
			// Update status to Failed
			changePlan.Status = models.ChangePlanStatusFailed
			errorMsg := fmt.Sprintf("failed to get latest schema version: %v", err)
			changePlan.ErrorMessage = &errorMsg
			changePlan.UpdatedAt = time.Now()
			if updateErr := s.changePlanRepo.Update(ctx, changePlan); updateErr != nil {
				s.logger.Error("Failed to update change plan status to Failed", zap.Error(updateErr))
			}
			return nil, fmt.Errorf("failed to get latest schema version: %w", err)
		}
		if latestSchema != nil {
			prevSchemaHistoryID = &latestSchema.ID
			s.logger.Info("Found previous schema version",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("prev_schema_history_id", prevSchemaHistoryID.String()),
			)
		}
	} else {
		s.logger.Info("No previous schema - database does not exist yet",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", db.ID.String()),
			zap.String("database_status", string(db.Status)),
		)
	}

	// Create changelog entry with status "pending"
	changelog := &models.Changelog{
		ID:                  uuid.New(),
		DatabaseInstanceID:  dbInstance.ID,
		DatabaseID:          db.ID,
		ChangePlanID:        changePlanID,
		Status:              models.ChangelogStatusPending,
		PrevSchemaHistoryID: prevSchemaHistoryID,
		SchemaHistoryID:     nil, // Will be set after execution
		CreatedAt:           time.Now(),
	}
	if err := s.changelogRepo.Create(ctx, changelog); err != nil {
		s.logger.Error("Failed to create changelog entry",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		// Update status to Failed
		changePlan.Status = models.ChangePlanStatusFailed
		errorMsg := fmt.Sprintf("failed to create changelog entry: %v", err)
		changePlan.ErrorMessage = &errorMsg
		changePlan.UpdatedAt = time.Now()
		if updateErr := s.changePlanRepo.Update(ctx, changePlan); updateErr != nil {
			s.logger.Error("Failed to update change plan status to Failed", zap.Error(updateErr))
		}
		return nil, fmt.Errorf("failed to create changelog entry: %w", err)
	}
	s.logger.Info("Changelog entry created",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("changelog_id", changelog.ID.String()),
		zap.String("status", string(changelog.Status)),
	)

	// Step 3: Create execution log entry (pre-execution)
	// We create this before execution so statement results can reference it via foreign key
	executionLogID := uuid.New()
	dbType := string(dbInstance.DatabaseEngine)
	sqlContent := changePlan.SQLContent
	executionLog := &models.ExecutionLog{
		ID:                   executionLogID,
		ChangePlanID:         changePlanID,
		ExecutorID:           executorID,
		Status:               models.ExecutionLogStatusFailed, // Will be updated to Completed on success
		SQLContent:           &sqlContent,
		DatabaseID:           &db.ID,
		DatabaseName:         &db.DatabaseName,
		DatabaseInstanceName: &dbInstance.Name,
		DatabaseEngine:       &dbType,
		StartedAt:            time.Now(),
	}

	// Persist execution log to database so statement results can reference it via foreign key
	if err := s.executionLogRepo.Create(ctx, executionLog); err != nil {
		s.logger.Error("Failed to create execution log entry",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create execution log: %w", err)
	}

	// Step 4: Execute SQL in transaction with timeout
	executionStartTime := time.Now()
	s.logger.Info("Starting SQL execution",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("execution_log_id", executionLogID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_engine", string(dbInstance.DatabaseEngine)),
		zap.Int("sql_content_length", sqlContentLength),
		zap.Duration("execution_timeout", s.executionTimeout),
	)

	// Create timeout context for SQL execution
	execCtx, cancel := context.WithTimeout(ctx, s.executionTimeout)
	defer cancel()

	result, err := s.executeInTransaction(execCtx, dbInstance, db, changePlan.SQLContent, changePlanID, executionLogID)
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
		// Step 5: Execution failed - update status to Failed
		var errorMsg string
		var userFriendlyMsg string

		// Check if it's a PositionedError (from multi-statement execution)
		if posErr, ok := err.(*PositionedError); ok {
			userFriendlyMsg = posErr.Error()
			errorMsg = userFriendlyMsg

			s.logger.Error("SQL execution failed (positioned)",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
				zap.Int("statement_index", posErr.StatementIndex),
				zap.Int("line", posErr.Line),
				zap.Int("column", posErr.Column),
				zap.String("user_message", userFriendlyMsg),
				zap.Duration("execution_duration", duration),
			)
		} else if sqlErr, ok := err.(*SQLExecutionError); ok {
			// Check if it's a SQLExecutionError for enhanced handling
			// Use the user-friendly message from SQLExecutionError
			userFriendlyMsg = sqlErr.Message
			errorMsg = userFriendlyMsg

			// Enhanced error context logging (already logged in executeInTransaction, but log summary here)
			s.logger.Error("SQL execution failed (summary)",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
				zap.String("database_name", db.DatabaseName),
				zap.String("database_instance_name", dbInstance.Name),
				zap.String("database_engine", string(dbInstance.DatabaseEngine)),
				zap.String("database_error_code", sqlErr.DatabaseErrorCode),
				zap.String("user_message", userFriendlyMsg),
				zap.Duration("execution_duration", duration),
				zap.Int("sql_content_length", sqlContentLength),
			)
		} else {
			// Generic error handling
			errorMsg = err.Error()
			userFriendlyMsg = errorMsg

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
				zap.String("database_engine", string(dbInstance.DatabaseEngine)),
				zap.Error(err),
				zap.Duration("execution_duration", duration),
				zap.Int("sql_content_length", sqlContentLength),
				zap.String("sql_preview", sqlPreview),
			)
		}

		changePlan.Status = models.ChangePlanStatusFailed
		changePlan.ErrorMessage = &errorMsg
		changePlan.AffectedRows = nil // No rows affected on error

		// Update Change Plan with error results
		if updateErr := s.changePlanRepo.Update(ctx, changePlan); updateErr != nil {
			s.logger.Error("Failed to update change plan with execution results", zap.Error(updateErr))
			// Don't fail the whole operation if update fails, but log it
		}

		// Update changelog status to "failed"
		changelog.Status = models.ChangelogStatusFailed
		if updateErr := s.changelogRepo.Update(ctx, changelog); updateErr != nil {
			s.logger.Error("Failed to update changelog status to failed",
				zap.String("changelog_id", changelog.ID.String()),
				zap.Error(updateErr),
			)
		} else {
			s.logger.Info("Changelog status updated to failed",
				zap.String("changelog_id", changelog.ID.String()),
			)
		}

		// Update execution log entry for failed execution
		if s.executionLogRepo != nil {
			executionLog.Status = models.ExecutionLogStatusFailed
			executionLog.ExecutionDurationMs = int(duration.Milliseconds())
			executionLog.ErrorMessage = &errorMsg
			executionLog.CompletedAt = time.Now()

			if logErr := s.executionLogRepo.Update(ctx, executionLog); logErr != nil {
				s.logger.Warn("Failed to update execution log entry", zap.Error(logErr))
			}
		}

		return &ExecutionResult{
			Success:      false,
			RowsAffected: 0,
			Duration:     duration,
			ErrorMessage: &userFriendlyMsg, // Use user-friendly message
		}, nil // Return result with error, don't propagate error
	}

	// Step 5: Execution succeeded - capture new schema version
	s.logger.Info("SQL execution completed successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.Int("rows_affected", result.RowsAffected),
		zap.Duration("execution_duration", duration),
	)

	schemaVersionStartTime := time.Now()
	s.logger.Info("Capturing schema version",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_engine", string(dbInstance.DatabaseEngine)),
	)

	// Build changelog metadata
	changelogMeta := models.ChangelogMetadata{}
	var schemaVersionDuration time.Duration

	schemaHistory, err := s.schemaVersionService.CaptureSchemaVersion(ctx, db.ID)
	if err != nil {
		s.logger.Warn("Failed to capture schema snapshot - execution will continue",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		// Record snapshot failure in metadata
		failedStatus := models.SnapshotStatusFailed
		errorMsg := err.Error()
		changelogMeta.SnapshotStatus = &failedStatus
		changelogMeta.SnapshotError = &errorMsg
		schemaVersionDuration = time.Since(schemaVersionStartTime)
	} else {
		schemaVersionDuration = time.Since(schemaVersionStartTime)
		s.logger.Info("Schema snapshot captured successfully",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("schema_history_id", schemaHistory.ID.String()),
			zap.Duration("schema_capture_duration", schemaVersionDuration),
		)
		// Record snapshot success in metadata
		successStatus := models.SnapshotStatusSuccess
		changelogMeta.SnapshotStatus = &successStatus
		changelog.SchemaHistoryID = &schemaHistory.ID
	}

	// Serialize metadata to JSON
	metadataJSON, err := json.Marshal(changelogMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize changelog metadata: %w", err)
	}
	changelog.Metadata = metadataJSON

	// Update changelog status to "done" and metadata
	changelog.Status = models.ChangelogStatusDone
	if updateErr := s.changelogRepo.Update(ctx, changelog); updateErr != nil {
		s.logger.Error("Failed to update changelog status to done",
			zap.String("changelog_id", changelog.ID.String()),
			zap.Error(updateErr),
		)
		// Continue anyway - the schema was captured successfully
	} else {
		s.logger.Info("Changelog status updated to done",
			zap.String("changelog_id", changelog.ID.String()),
			zap.String("schema_history_id", schemaHistory.ID.String()),
		)
	}

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

	// Step 7: Update database status from Pending to Active if this was a CREATE DATABASE operation
	if db.Status == models.DatabaseStatusPending {
		db.Status = models.DatabaseStatusActive
		db.UpdatedAt = time.Now()
		if err := s.databaseRepo.Update(ctx, db); err != nil {
			s.logger.Error("Failed to update database status to Active",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to update database status to Active: %w", err)
		}
		s.logger.Info("Database status updated to Active",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", db.ID.String()),
		)
	}

	overallDuration := time.Since(overallStartTime)
	s.logger.Info("Execution workflow completed successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_name", db.DatabaseName),
		zap.String("database_instance_name", dbInstance.Name),
		zap.String("database_engine", string(dbInstance.DatabaseEngine)),
		zap.Int("rows_affected", result.RowsAffected),
		zap.Duration("sql_execution_duration", duration),
		zap.Duration("schema_capture_duration", schemaVersionDuration),
		zap.Duration("lock_acquisition_duration", lockDuration),
		zap.Duration("total_execution_duration", overallDuration),
	)

	// Update execution log entry for successful execution
	if s.executionLogRepo != nil {
		rowsAffected := result.RowsAffected
		executionLog.Status = models.ExecutionLogStatusCompleted
		executionLog.ExecutionDurationMs = int(duration.Milliseconds())
		executionLog.RowsAffected = &rowsAffected
		executionLog.CompletedAt = time.Now()

		if logErr := s.executionLogRepo.Update(ctx, executionLog); logErr != nil {
			s.logger.Warn("Failed to update execution log entry", zap.Error(logErr))
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

// PositionedError wraps an error with position information for multi-statement execution
type PositionedError struct {
	Message        string
	Line           int
	Column         int
	StatementIndex int
	OriginalError  error
}

func (e *PositionedError) Error() string {
	return fmt.Sprintf("statement %d (line %d, column %d): %s",
		e.StatementIndex+1, e.Line, e.Column, e.Message)
}

func (e *PositionedError) Unwrap() error {
	return e.OriginalError
}

// executeInTransaction executes SQL statements within a SERIALIZABLE transaction
// Now supports multi-statement execution with per-statement tracking
func (s *ExecutionService) executeInTransaction(
	ctx context.Context,
	dbInstance *models.DatabaseInstance,
	db *models.Database,
	sqlContent string,
	changePlanID uuid.UUID,
	executionLogID uuid.UUID,
) (*ExecutionResult, error) {
	// Determine dialect for SQL parsing
	var dialect parser.Dialect
	switch dbInstance.DatabaseEngine {
	case models.DatabaseEnginePostgreSQL:
		dialect = parser.DialectPostgres
	case models.DatabaseEngineMySQL:
		dialect = parser.DialectMySQL
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", dbInstance.DatabaseEngine)
	}

	// Parse SQL into individual statements
	parseResult, err := parser.SplitSQL(sqlContent, dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SQL: %w", err)
	}

	// Validate statement count
	if parseResult.TotalCount > s.maxStatementCount {
		return nil, fmt.Errorf("too many statements: %d (max %d)", parseResult.TotalCount, s.maxStatementCount)
	}

	s.logger.Info("Parsed SQL statements",
		zap.String("change_plan_id", changePlanID.String()),
		zap.Int("statement_count", parseResult.TotalCount),
		zap.String("dialect", string(dialect)),
	)

	// Get database connection
	// For CREATE DATABASE operations (Pending status), connect to system database
	connectStartTime := time.Now()
	var dbConn *sqlx.DB

	if db.Status == models.DatabaseStatusPending {
		// Create a temporary database object for system database connection
		systemDB := &models.Database{
			ID:                 db.ID,
			ProjectID:          db.ProjectID,
			DatabaseInstanceID: db.DatabaseInstanceID,
			EnvironmentID:      db.EnvironmentID,
			Status:             db.Status,
			CreatedByID:        db.CreatedByID,
			CreatedAt:          db.CreatedAt,
			UpdatedAt:          db.UpdatedAt,
		}

		// Use system database based on engine type
		switch dbInstance.DatabaseEngine {
		case models.DatabaseEnginePostgreSQL:
			systemDB.DatabaseName = "postgres" // Default PostgreSQL system database
		case models.DatabaseEngineMySQL:
			systemDB.DatabaseName = "" // MySQL can execute CREATE DATABASE without specifying a database
		}

		s.logger.Info("Connecting to system database for CREATE DATABASE operation",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("target_database_name", db.DatabaseName),
			zap.String("system_database_name", systemDB.DatabaseName),
			zap.String("database_engine", string(dbInstance.DatabaseEngine)),
		)

		dbConn, err = s.connectionMgr.Connect(ctx, database.ConnectionParams{
			DatabaseInstance: dbInstance,
			Database:         systemDB,
			IOTimeout:        s.executionTimeout,
		})
	} else {
		dbConn, err = s.connectionMgr.Connect(ctx, database.ConnectionParams{
			DatabaseInstance: dbInstance,
			Database:         db,
			IOTimeout:        s.executionTimeout,
		})
	}

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

	// For CREATE DATABASE operations, we cannot use transactions (PostgreSQL restriction)
	// Execute directly without transaction for Pending databases
	if db.Status == models.DatabaseStatusPending {
		s.logger.Info("Executing CREATE DATABASE without transaction",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_name", db.DatabaseName),
		)

		execStartTime := time.Now()
		result, err := dbConn.ExecContext(ctx, sqlContent)
		execDuration := time.Since(execStartTime)

		if err != nil {
			s.logger.Error("Failed to execute CREATE DATABASE",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_name", db.DatabaseName),
				zap.Duration("execution_duration", execDuration),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to execute SQL: %w", err)
		}

		rowsAffected, _ := result.RowsAffected()
		s.logger.Info("CREATE DATABASE executed successfully",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_name", db.DatabaseName),
			zap.Int64("rows_affected", rowsAffected),
			zap.Duration("execution_duration", execDuration),
		)

		return &ExecutionResult{
			Success:      true,
			RowsAffected: int(rowsAffected),
			Duration:     execDuration,
		}, nil
	}

	// Begin transaction with SERIALIZABLE isolation for normal operations
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

	// Execute SQL statements sequentially
	execStartTime := time.Now()
	var totalRowsAffected int64
	var statementResults []models.StatementExecutionResult

	s.logger.Debug("Executing SQL statements",
		zap.String("change_plan_id", changePlanID.String()),
		zap.Int("statement_count", parseResult.TotalCount),
	)

	for _, stmt := range parseResult.Statements {
		if stmt.Empty {
			continue // Skip empty/comment-only statements
		}

		stmtStart := time.Now()
		result, err := tx.ExecContext(ctx, stmt.Text)
		stmtDuration := time.Since(stmtStart)

		stmtResult := models.StatementExecutionResult{
			ID:                  uuid.New(),
			ExecutionLogID:      executionLogID,
			StatementIndex:      stmt.Index,
			StatementSQL:        stmt.Text,
			ExecutionDurationMs: int(stmtDuration.Milliseconds()),
			StartLine:           &stmt.StartLine,
			StartColumn:         &stmt.StartColumn,
			EndLine:             &stmt.EndLine,
			EndColumn:           &stmt.EndColumn,
		}

		if err != nil {
			// Execution failed - record error
			errMsg := err.Error()
			stmtResult.Status = models.StatementStatusFailed
			stmtResult.ErrorMessage = &errMsg
			statementResults = append(statementResults, stmtResult)

			// Save statement results before returning
			s.saveStatementResults(ctx, statementResults)

			s.logger.Error("Statement execution failed",
				zap.String("change_plan_id", changePlanID.String()),
				zap.Int("statement_index", stmt.Index),
				zap.Int("start_line", stmt.StartLine),
				zap.Int("start_column", stmt.StartColumn),
				zap.Duration("statement_duration", stmtDuration),
				zap.Error(err),
			)

			// Create structured SQL execution error with position info
			sqlErr := NewSQLExecutionError(
				string(dbInstance.DatabaseEngine),
				db.DatabaseName,
				db.ID.String(),
				dbInstance.Name,
				stmt.Text,
				err,
			)

			// Return positioned error
			return nil, &PositionedError{
				Message:        sqlErr.Message,
				Line:           stmt.StartLine,
				Column:         stmt.StartColumn,
				StatementIndex: stmt.Index,
				OriginalError:  sqlErr,
			}
		}

		// Success - record result
		rowsAffected, _ := result.RowsAffected()
		rows := int(rowsAffected)
		totalRowsAffected += rowsAffected
		stmtResult.Status = models.StatementStatusCompleted
		stmtResult.RowsAffected = &rows
		statementResults = append(statementResults, stmtResult)

		s.logger.Debug("Statement executed successfully",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Int("statement_index", stmt.Index),
			zap.Int64("rows_affected", rowsAffected),
			zap.Duration("statement_duration", stmtDuration),
		)
	}

	// Commit transaction
	commitStartTime := time.Now()
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Int64("total_rows_affected", totalRowsAffected),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	commitDuration := time.Since(commitStartTime)
	s.logger.Debug("Transaction committed successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.Int64("total_rows_affected", totalRowsAffected),
		zap.Duration("commit_duration", commitDuration),
	)

	// Mark that we don't need to rollback (commit succeeded)
	shouldRollback = false

	// Save statement results after successful commit
	s.saveStatementResults(ctx, statementResults)

	duration := time.Since(execStartTime)

	return &ExecutionResult{
		Success:      true,
		RowsAffected: int(totalRowsAffected),
		Duration:     duration,
	}, nil
}

// saveStatementResults saves statement execution results to the repository
func (s *ExecutionService) saveStatementResults(ctx context.Context, results []models.StatementExecutionResult) {
	if s.statementResultRepo == nil || len(results) == 0 {
		return
	}

	if err := s.statementResultRepo.CreateBatch(ctx, results); err != nil {
		s.logger.Warn("Failed to save statement execution results",
			zap.Int("result_count", len(results)),
			zap.Error(err),
		)
	}
}
