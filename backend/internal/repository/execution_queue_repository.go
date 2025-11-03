package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// ExecutionQueueRepository defines the interface for execution queue data operations
type ExecutionQueueRepository interface {
	// Enqueue adds a Change Plan to the execution queue
	Enqueue(ctx context.Context, queueItem *models.ExecutionQueue) error

	// GetNextPending retrieves the next pending item for a database
	// Returns nil if no pending items exist
	GetNextPending(ctx context.Context, databaseID uuid.UUID) (*models.ExecutionQueue, error)

	// GetByID retrieves a queue item by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.ExecutionQueue, error)

	// GetByChangePlanID retrieves queue items for a specific change plan
	GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.ExecutionQueue, error)

	// GetByDatabaseID retrieves all queue items for a database
	// Ordered by priority DESC, position ASC
	GetByDatabaseID(ctx context.Context, databaseID uuid.UUID) ([]*models.ExecutionQueue, error)

	// GetPendingCount returns the number of pending items for a database
	GetPendingCount(ctx context.Context, databaseID uuid.UUID) (int, error)

	// GetPosition returns the position of a queue item (number of pending items before it)
	GetPosition(ctx context.Context, queueItemID uuid.UUID) (int, error)

	// UpdateStatus updates the status of a queue item
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.QueueStatus) error

	// Update updates a queue item
	Update(ctx context.Context, queueItem *models.ExecutionQueue) error

	// RecalculatePositions recalculates position values for all pending items for a database
	RecalculatePositions(ctx context.Context, databaseID uuid.UUID) error

	// MarkInProgress marks the next pending item as InProgress
	MarkInProgress(ctx context.Context, databaseID uuid.UUID) (*models.ExecutionQueue, error)

	// GetDatabasesWithPending returns all database IDs that have pending items
	GetDatabasesWithPending(ctx context.Context) ([]uuid.UUID, error)

	// MarkCompleted marks a queue item as completed
	MarkCompleted(ctx context.Context, id uuid.UUID) error

	// MarkFailed marks a queue item as failed
	MarkFailed(ctx context.Context, id uuid.UUID, errorMessage string) error
}
