package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
)

// executionLogRepository implements the ExecutionLogRepository interface for PostgreSQL
type executionLogRepository struct {
	db *sqlx.DB
}

// NewExecutionLogRepository creates a new PostgreSQL execution log repository
func NewExecutionLogRepository(db *sqlx.DB) repository.ExecutionLogRepository {
	return &executionLogRepository{db: db}
}

// Create creates a new execution log entry
func (r *executionLogRepository) Create(ctx context.Context, log *models.ExecutionLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	now := time.Now()
	if log.CreatedAt.IsZero() {
		log.CreatedAt = now
	}

	query := `
		INSERT INTO execution_logs (
			id, change_plan_id, executor_id, status,
			execution_duration_ms, rows_affected, error_message,
			sql_content, database_id, database_name, database_instance_name, database_engine,
			started_at, completed_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		log.ID,
		log.ChangePlanID,
		log.ExecutorID,
		log.Status,
		log.ExecutionDurationMs,
		log.RowsAffected,
		log.ErrorMessage,
		log.SQLContent,
		log.DatabaseID,
		log.DatabaseName,
		log.DatabaseInstanceName,
		log.DatabaseEngine,
		log.StartedAt,
		log.CompletedAt,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create execution log: %w", err)
	}

	return nil
}

// GetByChangePlanID retrieves all execution logs for a specific change plan
func (r *executionLogRepository) GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.ExecutionLog, error) {
	logs := []*models.ExecutionLog{}

	query := `
		SELECT id, change_plan_id, executor_id, status,
		       execution_duration_ms, rows_affected, error_message,
		       sql_content, database_id, database_name, database_instance_name, database_engine,
		       started_at, completed_at, created_at
		FROM execution_logs
		WHERE change_plan_id = $1
		ORDER BY started_at DESC
	`

	err := r.db.SelectContext(ctx, &logs, query, changePlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution logs by change plan id: %w", err)
	}

	return logs, nil
}

// GetByID retrieves an execution log by ID
func (r *executionLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ExecutionLog, error) {
	var log models.ExecutionLog

	query := `
		SELECT id, change_plan_id, executor_id, status,
		       execution_duration_ms, rows_affected, error_message,
		       sql_content, database_id, database_name, database_instance_name, database_engine,
		       started_at, completed_at, created_at
		FROM execution_logs
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &log, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get execution log by id: %w", err)
	}

	return &log, nil
}

// Update updates an existing execution log entry
func (r *executionLogRepository) Update(ctx context.Context, log *models.ExecutionLog) error {
	query := `
		UPDATE execution_logs
		SET status = $2,
		    execution_duration_ms = $3,
		    rows_affected = $4,
		    error_message = $5,
		    completed_at = $6
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		log.ID,
		log.Status,
		log.ExecutionDurationMs,
		log.RowsAffected,
		log.ErrorMessage,
		log.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update execution log: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// GetLatestByChangePlanID retrieves the most recent execution log for a change plan
func (r *executionLogRepository) GetLatestByChangePlanID(
	ctx context.Context,
	changePlanID uuid.UUID,
) (*models.ExecutionLog, error) {
	var log models.ExecutionLog

	query := `
		SELECT id, change_plan_id, executor_id, status,
		       execution_duration_ms, rows_affected, error_message,
		       sql_content, database_id, database_name, database_instance_name, database_engine,
		       started_at, completed_at, created_at
		FROM execution_logs
		WHERE change_plan_id = $1
		ORDER BY started_at DESC
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &log, query, changePlanID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get latest execution log: %w", err)
	}

	return &log, nil
}
