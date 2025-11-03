package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// ExecutionQueueService provides queue management functionality for Change Plan execution
type ExecutionQueueService struct {
	queueRepo repository.ExecutionQueueRepository
	logger    *zap.Logger
}

// NewExecutionQueueService creates a new execution queue service
func NewExecutionQueueService(
	queueRepo repository.ExecutionQueueRepository,
	logger *zap.Logger,
) *ExecutionQueueService {
	return &ExecutionQueueService{
		queueRepo: queueRepo,
		logger:    logger,
	}
}

// Enqueue adds a Change Plan to the execution queue
func (s *ExecutionQueueService) Enqueue(
	ctx context.Context,
	changePlanID uuid.UUID,
	databaseID uuid.UUID,
	executorID uuid.UUID,
) (*models.ExecutionQueue, error) {
	queueItem := &models.ExecutionQueue{
		ChangePlanID:    changePlanID,
		TargetDatabaseID: databaseID,
		ExecutorID:      executorID,
		Status:          models.QueueStatusPending,
		Priority:        0, // Default priority
	}

	if err := s.queueRepo.Enqueue(ctx, queueItem); err != nil {
		return nil, fmt.Errorf("failed to enqueue change plan: %w", err)
	}

	s.logger.Info("Change plan enqueued",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", databaseID.String()),
		zap.String("executor_id", executorID.String()),
		zap.Int("position", queueItem.Position),
	)

	return queueItem, nil
}

// GetQueuePosition returns the position of a Change Plan in the queue
func (s *ExecutionQueueService) GetQueuePosition(
	ctx context.Context,
	changePlanID uuid.UUID,
) (int, error) {
	queueItems, err := s.queueRepo.GetByChangePlanID(ctx, changePlanID)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue items: %w", err)
	}

	// Find the most recent pending item
	for _, item := range queueItems {
		if item.Status == models.QueueStatusPending {
			return item.Position, nil
		}
	}

	return 0, fmt.Errorf("no pending queue item found for change plan")
}

// GetQueueStatus returns the queue status for a Change Plan
func (s *ExecutionQueueService) GetQueueStatus(
	ctx context.Context,
	changePlanID uuid.UUID,
) (*models.ExecutionQueue, error) {
	queueItems, err := s.queueRepo.GetByChangePlanID(ctx, changePlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue items: %w", err)
	}

	// Return the most recent item
	if len(queueItems) > 0 {
		return queueItems[0], nil
	}

	return nil, repository.ErrNotFound
}

// GetQueueItemsForDatabase returns all queue items for a database
func (s *ExecutionQueueService) GetQueueItemsForDatabase(
	ctx context.Context,
	databaseID uuid.UUID,
) ([]*models.ExecutionQueue, error) {
	return s.queueRepo.GetByDatabaseID(ctx, databaseID)
}

// GetPendingCount returns the number of pending items for a database
func (s *ExecutionQueueService) GetPendingCount(
	ctx context.Context,
	databaseID uuid.UUID,
) (int, error) {
	return s.queueRepo.GetPendingCount(ctx, databaseID)
}

// Queue service errors
var (
	ErrQueueItemNotFound = fmt.Errorf("queue item not found")
)

