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
	connectionMgr        ConnectionManagerInterface
	encryptionKey        string
	executionTimeout     time.Duration // Timeout for SQL execution (default: 5 minutes)
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
	connectionMgr ConnectionManagerInterface,
	encryptionKey string,
	logger *zap.Logger,
) *ExecutionService {
	executionTimeout := 5 * time.Minute // Default: 5 minutes

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
		connectionMgr:        connectionMgr,
		encryptionKey:        encryptionKey,
		executionTimeout:     executionTimeout,
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
	connectionMgr ConnectionManagerInterface,
	encryptionKey string,
	executionTimeout time.Duration,
	logger *zap.Logger,
) *ExecutionService {
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
		connectionMgr:        connectionMgr,
		encryptionKey:        encryptionKey,
		executionTimeout:     executionTimeout,
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

	// Step 3: Execute SQL in transaction with timeout
	executionStartTime := time.Now()
	s.logger.Info("Starting SQL execution",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_engine", string(dbInstance.DatabaseEngine)),
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
		var errorMsg string
		var userFriendlyMsg string

		// Check if it's a SQLExecutionError for enhanced handling
		if sqlErr, ok := err.(*SQLExecutionError); ok {
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

		// Create execution log entry for failed execution with full context
		if s.executionLogRepo != nil {
			dbType := string(dbInstance.DatabaseEngine)
			sqlContent := changePlan.SQLContent
			executionLog := &models.ExecutionLog{
				ChangePlanID:         changePlanID,
				ExecutorID:           executorID,
				Status:               models.ExecutionLogStatusFailed,
				ExecutionDurationMs:  int(duration.Milliseconds()),
				RowsAffected:         nil,
				ErrorMessage:         &errorMsg,
				SQLContent:           &sqlContent,
				DatabaseID:           &db.ID,
				DatabaseName:         &db.DatabaseName,
				DatabaseInstanceName: &dbInstance.Name,
				DatabaseEngine:       &dbType,
				StartedAt:            executionStartTime,
				CompletedAt:          time.Now(),
			}
			if logErr := s.executionLogRepo.Create(ctx, executionLog); logErr != nil {
				s.logger.Warn("Failed to create execution log entry", zap.Error(logErr))
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

	// Create execution log entry for successful execution with full context
	if s.executionLogRepo != nil {
		dbType := string(dbInstance.DatabaseEngine)
		sqlContent := changePlan.SQLContent
		rowsAffected := result.RowsAffected
		executionLog := &models.ExecutionLog{
			ChangePlanID:         changePlanID,
			ExecutorID:           executorID,
			Status:               models.ExecutionLogStatusCompleted,
			ExecutionDurationMs:  int(duration.Milliseconds()),
			RowsAffected:         &rowsAffected,
			ErrorMessage:         nil,
			SQLContent:           &sqlContent,
			DatabaseID:           &db.ID,
			DatabaseName:         &db.DatabaseName,
			DatabaseInstanceName: &dbInstance.Name,
			DatabaseEngine:       &dbType,
			StartedAt:            executionStartTime,
			CompletedAt:          time.Now(),
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
	// For CREATE DATABASE operations (Pending status), connect to system database
	connectStartTime := time.Now()
	var dbConn *sqlx.DB
	var err error

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
		default:
			return nil, fmt.Errorf("unsupported database engine: %s", dbInstance.DatabaseEngine)
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
		})
	} else {
		dbConn, err = s.connectionMgr.Connect(ctx, database.ConnectionParams{
			DatabaseInstance: dbInstance,
			Database:         db,
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

	// Execute SQL statements
	execStartTime := time.Now()
	s.logger.Debug("Executing SQL statements",
		zap.String("change_plan_id", changePlanID.String()),
		zap.Int("sql_content_length", len(sqlContent)),
	)
	result, err := tx.ExecContext(ctx, sqlContent)
	execDuration := time.Since(execStartTime)
	if err != nil {
		// Create structured SQL execution error
		sqlErr := NewSQLExecutionError(
			string(dbInstance.DatabaseEngine),
			db.DatabaseName,
			db.ID.String(),
			dbInstance.Name,
			sqlContent,
			err,
		)

		// Mask SQL content for logging
		maskedSQL := MaskSQLContent(sqlContent)
		sqlErr.SQLContentMasked = true

		// Log comprehensive error details
		logFields := []zap.Field{
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", db.ID.String()),
			zap.String("database_name", db.DatabaseName),
			zap.String("database_instance_name", dbInstance.Name),
			zap.String("database_engine", string(dbInstance.DatabaseEngine)),
			zap.Duration("execution_duration", execDuration),
			zap.Int("sql_content_length", len(sqlContent)),
			zap.String("sql_content_masked", maskedSQL),
			zap.Error(err),
		}

		// Add database-specific error details
		if sqlErr.DatabaseErrorCode != "" {
			logFields = append(logFields, zap.String("database_error_code", sqlErr.DatabaseErrorCode))
		}
		if sqlErr.DatabaseMessage != "" {
			logFields = append(logFields, zap.String("database_message", sqlErr.DatabaseMessage))
		}
		if sqlErr.DatabaseDetail != "" {
			logFields = append(logFields, zap.String("database_detail", sqlErr.DatabaseDetail))
		}
		if sqlErr.DatabaseHint != "" {
			logFields = append(logFields, zap.String("database_hint", sqlErr.DatabaseHint))
		}
		if sqlErr.DatabasePosition != "" {
			logFields = append(logFields, zap.String("database_position", sqlErr.DatabasePosition))
		}

		// Add stack trace (only in debug mode or for internal errors)
		if sqlErr.StackTrace != "" {
			logFields = append(logFields, zap.String("stack_trace", sqlErr.StackTrace))
		}

		s.logger.Error("SQL execution failed", logFields...)

		return nil, sqlErr
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
