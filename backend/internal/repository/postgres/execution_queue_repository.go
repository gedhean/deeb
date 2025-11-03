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

// executionQueueRepository implements the ExecutionQueueRepository interface for PostgreSQL
type executionQueueRepository struct {
	db *sqlx.DB
}

// NewExecutionQueueRepository creates a new PostgreSQL execution queue repository
func NewExecutionQueueRepository(db *sqlx.DB) repository.ExecutionQueueRepository {
	return &executionQueueRepository{db: db}
}

// Enqueue adds a Change Plan to the execution queue
func (r *executionQueueRepository) Enqueue(ctx context.Context, queueItem *models.ExecutionQueue) error {
	if queueItem.ID == uuid.Nil {
		queueItem.ID = uuid.New()
	}

	now := time.Now()
	if queueItem.QueuedAt.IsZero() {
		queueItem.QueuedAt = now
	}
	if queueItem.CreatedAt.IsZero() {
		queueItem.CreatedAt = now
	}
	if queueItem.UpdatedAt.IsZero() {
		queueItem.UpdatedAt = now
	}

	// Set default status if not set
	if queueItem.Status == "" {
		queueItem.Status = models.QueueStatusPending
	}

	// Calculate position: number of pending items for this database
	pendingCount, err := r.GetPendingCount(ctx, queueItem.TargetDatabaseID)
	if err != nil {
		return fmt.Errorf("failed to get pending count: %w", err)
	}
	queueItem.Position = pendingCount

	query := `
		INSERT INTO execution_queue (
			id, change_plan_id, target_database_id, executor_id,
			status, position, priority, error_message,
			queued_at, started_at, completed_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`

	_, err = r.db.ExecContext(
		ctx,
		query,
		queueItem.ID,
		queueItem.ChangePlanID,
		queueItem.TargetDatabaseID,
		queueItem.ExecutorID,
		queueItem.Status,
		queueItem.Position,
		queueItem.Priority,
		queueItem.ErrorMessage,
		queueItem.QueuedAt,
		queueItem.StartedAt,
		queueItem.CompletedAt,
		queueItem.CreatedAt,
		queueItem.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to enqueue: %w", err)
	}

	return nil
}

// GetNextPending retrieves the next pending item for a database
// Returns nil if no pending items exist
func (r *executionQueueRepository) GetNextPending(ctx context.Context, databaseID uuid.UUID) (*models.ExecutionQueue, error) {
	var queueItem models.ExecutionQueue

	query := `
		SELECT id, change_plan_id, target_database_id, executor_id,
		       status, position, priority, error_message,
		       queued_at, started_at, completed_at, created_at, updated_at
		FROM execution_queue
		WHERE target_database_id = $1 AND status = $2
		ORDER BY priority DESC, position ASC, created_at ASC
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &queueItem, query, databaseID, models.QueueStatusPending)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get next pending: %w", err)
	}

	return &queueItem, nil
}

// GetByID retrieves a queue item by ID
func (r *executionQueueRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ExecutionQueue, error) {
	var queueItem models.ExecutionQueue

	query := `
		SELECT id, change_plan_id, target_database_id, executor_id,
		       status, position, priority, error_message,
		       queued_at, started_at, completed_at, created_at, updated_at
		FROM execution_queue
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &queueItem, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get queue item by id: %w", err)
	}

	return &queueItem, nil
}

// GetByChangePlanID retrieves queue items for a specific change plan
func (r *executionQueueRepository) GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.ExecutionQueue, error) {
	queueItems := []*models.ExecutionQueue{}

	query := `
		SELECT id, change_plan_id, target_database_id, executor_id,
		       status, position, priority, error_message,
		       queued_at, started_at, completed_at, created_at, updated_at
		FROM execution_queue
		WHERE change_plan_id = $1
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &queueItems, query, changePlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue items by change plan id: %w", err)
	}

	return queueItems, nil
}

// GetByDatabaseID retrieves all queue items for a database
func (r *executionQueueRepository) GetByDatabaseID(ctx context.Context, databaseID uuid.UUID) ([]*models.ExecutionQueue, error) {
	queueItems := []*models.ExecutionQueue{}

	query := `
		SELECT id, change_plan_id, target_database_id, executor_id,
		       status, position, priority, error_message,
		       queued_at, started_at, completed_at, created_at, updated_at
		FROM execution_queue
		WHERE target_database_id = $1
		ORDER BY priority DESC, position ASC, created_at ASC
	`

	err := r.db.SelectContext(ctx, &queueItems, query, databaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue items by database id: %w", err)
	}

	return queueItems, nil
}

// GetPendingCount returns the number of pending items for a database
func (r *executionQueueRepository) GetPendingCount(ctx context.Context, databaseID uuid.UUID) (int, error) {
	var count int

	query := `
		SELECT COUNT(*) FROM execution_queue
		WHERE target_database_id = $1 AND status = $2
	`

	err := r.db.GetContext(ctx, &count, query, databaseID, models.QueueStatusPending)
	if err != nil {
		return 0, fmt.Errorf("failed to get pending count: %w", err)
	}

	return count, nil
}

// GetPosition returns the position of a queue item (number of pending items before it)
func (r *executionQueueRepository) GetPosition(ctx context.Context, queueItemID uuid.UUID) (int, error) {
	var position int

	query := `
		SELECT COUNT(*) FROM execution_queue eq1
		WHERE eq1.target_database_id = (
			SELECT target_database_id FROM execution_queue WHERE id = $1
		)
		AND eq1.status = $2
		AND (eq1.priority > (
			SELECT priority FROM execution_queue WHERE id = $1
		) OR (eq1.priority = (
			SELECT priority FROM execution_queue WHERE id = $1
		) AND eq1.created_at < (
			SELECT created_at FROM execution_queue WHERE id = $1
		)))
	`

	err := r.db.GetContext(ctx, &position, query, queueItemID, models.QueueStatusPending)
	if err != nil {
		return 0, fmt.Errorf("failed to get position: %w", err)
	}

	return position, nil
}

// UpdateStatus updates the status of a queue item
func (r *executionQueueRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.QueueStatus) error {
	now := time.Now()

	query := `
		UPDATE execution_queue
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := r.db.ExecContext(ctx, query, status, now, id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// Update updates a queue item
func (r *executionQueueRepository) Update(ctx context.Context, queueItem *models.ExecutionQueue) error {
	queueItem.UpdatedAt = time.Now()

	query := `
		UPDATE execution_queue
		SET change_plan_id = $1, target_database_id = $2, executor_id = $3,
		    status = $4, position = $5, priority = $6, error_message = $7,
		    queued_at = $8, started_at = $9, completed_at = $10, updated_at = $11
		WHERE id = $12
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		queueItem.ChangePlanID,
		queueItem.TargetDatabaseID,
		queueItem.ExecutorID,
		queueItem.Status,
		queueItem.Position,
		queueItem.Priority,
		queueItem.ErrorMessage,
		queueItem.QueuedAt,
		queueItem.StartedAt,
		queueItem.CompletedAt,
		queueItem.UpdatedAt,
		queueItem.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update queue item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// RecalculatePositions recalculates position values for all pending items for a database
func (r *executionQueueRepository) RecalculatePositions(ctx context.Context, databaseID uuid.UUID) error {
	// Use a CTE to update positions based on priority and created_at
	query := `
		WITH ranked_items AS (
			SELECT id, 
			       ROW_NUMBER() OVER (ORDER BY priority DESC, created_at ASC) - 1 AS new_position
			FROM execution_queue
			WHERE target_database_id = $1 AND status = $2
		)
		UPDATE execution_queue eq
		SET position = ranked_items.new_position, updated_at = NOW()
		FROM ranked_items
		WHERE eq.id = ranked_items.id
	`

	_, err := r.db.ExecContext(ctx, query, databaseID, models.QueueStatusPending)
	if err != nil {
		return fmt.Errorf("failed to recalculate positions: %w", err)
	}

	return nil
}

// GetDatabasesWithPending returns all database IDs that have pending items
func (r *executionQueueRepository) GetDatabasesWithPending(ctx context.Context) ([]uuid.UUID, error) {
	var databaseIDs []uuid.UUID

	query := `
		SELECT DISTINCT target_database_id
		FROM execution_queue
		WHERE status = $1
		ORDER BY target_database_id
	`

	err := r.db.SelectContext(ctx, &databaseIDs, query, models.QueueStatusPending)
	if err != nil {
		return nil, fmt.Errorf("failed to get databases with pending items: %w", err)
	}

	return databaseIDs, nil
}

// MarkInProgress marks the next pending item as InProgress
func (r *executionQueueRepository) MarkInProgress(ctx context.Context, databaseID uuid.UUID) (*models.ExecutionQueue, error) {
	// Get the next pending item
	queueItem, err := r.GetNextPending(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	queueItem.Status = models.QueueStatusInProgress
	queueItem.StartedAt = &now
	queueItem.UpdatedAt = now

	if err := r.Update(ctx, queueItem); err != nil {
		return nil, fmt.Errorf("failed to mark in progress: %w", err)
	}

	// Recalculate positions for remaining pending items
	// Ignore error if recalculating fails - positions will be recalculated on next query
	_ = r.RecalculatePositions(ctx, databaseID)

	return queueItem, nil
}

// MarkCompleted marks a queue item as completed
func (r *executionQueueRepository) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	queueItem, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	queueItem.Status = models.QueueStatusCompleted
	queueItem.CompletedAt = &now
	queueItem.ErrorMessage = nil // Clear any previous error
	queueItem.UpdatedAt = now

	if err := r.Update(ctx, queueItem); err != nil {
		return fmt.Errorf("failed to mark completed: %w", err)
	}

	// Recalculate positions for remaining pending items
	// Ignore error if recalculating fails - positions will be recalculated on next query
	_ = r.RecalculatePositions(ctx, queueItem.TargetDatabaseID)

	return nil
}

// MarkFailed marks a queue item as failed
func (r *executionQueueRepository) MarkFailed(ctx context.Context, id uuid.UUID, errorMessage string) error {
	queueItem, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	queueItem.Status = models.QueueStatusFailed
	queueItem.CompletedAt = &now
	queueItem.ErrorMessage = &errorMessage
	queueItem.UpdatedAt = now

	if err := r.Update(ctx, queueItem); err != nil {
		return fmt.Errorf("failed to mark failed: %w", err)
	}

	// Recalculate positions for remaining pending items
	// Ignore error if recalculating fails - positions will be recalculated on next query
	_ = r.RecalculatePositions(ctx, queueItem.TargetDatabaseID)

	return nil
}

