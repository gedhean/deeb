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

// statementExecutionResultRepository implements the StatementExecutionResultRepository interface
type statementExecutionResultRepository struct {
	db *sqlx.DB
}

// NewStatementExecutionResultRepository creates a new PostgreSQL statement execution result repository
func NewStatementExecutionResultRepository(db *sqlx.DB) repository.StatementExecutionResultRepository {
	return &statementExecutionResultRepository{db: db}
}

// Create creates a new statement execution result
func (r *statementExecutionResultRepository) Create(
	ctx context.Context,
	result *models.StatementExecutionResult,
) error {
	if result.ID == uuid.Nil {
		result.ID = uuid.New()
	}

	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO statement_execution_results (
			id, execution_log_id, statement_index, statement_sql, status,
			rows_affected, execution_duration_ms, error_message,
			start_line, start_column, end_line, end_column, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		result.ID,
		result.ExecutionLogID,
		result.StatementIndex,
		result.StatementSQL,
		result.Status,
		result.RowsAffected,
		result.ExecutionDurationMs,
		result.ErrorMessage,
		result.StartLine,
		result.StartColumn,
		result.EndLine,
		result.EndColumn,
		result.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create statement execution result: %w", err)
	}

	return nil
}

// CreateBatch creates multiple statement execution results in a single transaction
func (r *statementExecutionResultRepository) CreateBatch(
	ctx context.Context,
	results []models.StatementExecutionResult,
) error {
	if len(results) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	query := `
		INSERT INTO statement_execution_results (
			id, execution_log_id, statement_index, statement_sql, status,
			rows_affected, execution_duration_ms, error_message,
			start_line, start_column, end_line, end_column, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`

	now := time.Now()
	for i := range results {
		result := &results[i]
		if result.ID == uuid.Nil {
			result.ID = uuid.New()
		}
		if result.CreatedAt.IsZero() {
			result.CreatedAt = now
		}

		_, err := tx.ExecContext(
			ctx,
			query,
			result.ID,
			result.ExecutionLogID,
			result.StatementIndex,
			result.StatementSQL,
			result.Status,
			result.RowsAffected,
			result.ExecutionDurationMs,
			result.ErrorMessage,
			result.StartLine,
			result.StartColumn,
			result.EndLine,
			result.EndColumn,
			result.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to create statement execution result %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByExecutionLogID retrieves all statement results for an execution log
func (r *statementExecutionResultRepository) GetByExecutionLogID(
	ctx context.Context,
	executionLogID uuid.UUID,
) ([]*models.StatementExecutionResult, error) {
	results := []*models.StatementExecutionResult{}

	query := `
		SELECT id, execution_log_id, statement_index, statement_sql, status,
		       rows_affected, execution_duration_ms, error_message,
		       start_line, start_column, end_line, end_column, created_at
		FROM statement_execution_results
		WHERE execution_log_id = $1
		ORDER BY statement_index ASC
	`

	err := r.db.SelectContext(ctx, &results, query, executionLogID)
	if err != nil {
		return nil, fmt.Errorf("failed to get statement results by execution log id: %w", err)
	}

	return results, nil
}

// GetByID retrieves a statement execution result by ID
func (r *statementExecutionResultRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*models.StatementExecutionResult, error) {
	var result models.StatementExecutionResult

	query := `
		SELECT id, execution_log_id, statement_index, statement_sql, status,
		       rows_affected, execution_duration_ms, error_message,
		       start_line, start_column, end_line, end_column, created_at
		FROM statement_execution_results
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &result, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get statement execution result by id: %w", err)
	}

	return &result, nil
}
