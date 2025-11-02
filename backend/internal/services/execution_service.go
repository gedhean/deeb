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
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// ExecutionService provides SQL execution functionality with transaction management
type ExecutionService struct {
	changePlanRepo   repository.ChangePlanRepository
	dbInstanceRepo   repository.DatabaseInstanceRepository
	databaseRepo     repository.DatabaseRepository
	connectionMgr    *database.ConnectionManager
	encryptionKey    string
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
	connectionMgr *database.ConnectionManager,
	encryptionKey string,
	logger *zap.Logger,
) *ExecutionService {
	return &ExecutionService{
		changePlanRepo: changePlanRepo,
		dbInstanceRepo: dbInstanceRepo,
		databaseRepo:   databaseRepo,
		connectionMgr:  connectionMgr,
		encryptionKey:  encryptionKey,
		logger:         logger,
	}
}

// Execute executes SQL statements from a change plan within a SERIALIZABLE transaction
func (s *ExecutionService) Execute(
	ctx context.Context,
	changePlanID uuid.UUID,
) (*ExecutionResult, error) {
	// Get change plan
	changePlan, err := s.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrChangePlanNotFound
		}
		return nil, fmt.Errorf("failed to get change plan: %w", err)
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

	// Execute SQL in transaction
	startTime := time.Now()
	result, err := s.executeInTransaction(ctx, dbConn, changePlan.SQLContent)
	duration := time.Since(startTime)

	// Store execution results in Change Plan
	now := time.Now()
	changePlan.ExecutedAt = &now
	durationMs := int(duration.Milliseconds())
	changePlan.ExecutionDurationMs = &durationMs
	changePlan.UpdatedAt = time.Now()

	if err != nil {
		errorMsg := err.Error()
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

	// Store success results
	changePlan.AffectedRows = &result.RowsAffected
	changePlan.ErrorMessage = nil // Clear any previous error message

	// Update Change Plan with execution results
	if err := s.changePlanRepo.Update(ctx, changePlan); err != nil {
		return nil, fmt.Errorf("failed to update change plan with execution results: %w", err)
	}

	s.logger.Info("Execution succeeded",
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
	db *sqlx.DB,
	sqlContent string,
) (*ExecutionResult, error) {
	// Begin transaction with SERIALIZABLE isolation
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{
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

