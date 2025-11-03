package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// QueueProcessor processes queued Change Plan executions in the background
type QueueProcessor struct {
	queueRepo        repository.ExecutionQueueRepository
	executionService *ExecutionService
	logger           *zap.Logger
	stopChan         chan struct{}
	wg               sync.WaitGroup
	pollInterval     time.Duration
}

// NewQueueProcessor creates a new queue processor
func NewQueueProcessor(
	queueRepo repository.ExecutionQueueRepository,
	executionService *ExecutionService,
	logger *zap.Logger,
) *QueueProcessor {
	return &QueueProcessor{
		queueRepo:        queueRepo,
		executionService: executionService,
		logger:           logger,
		stopChan:         make(chan struct{}),
		pollInterval:     5 * time.Second, // Poll every 5 seconds
	}
}

// Start starts the queue processor background worker
func (p *QueueProcessor) Start(ctx context.Context) {
	p.wg.Add(1)
	go p.processLoop(ctx)
	p.logger.Info("Queue processor started")
}

// Stop stops the queue processor
func (p *QueueProcessor) Stop() {
	close(p.stopChan)
	p.wg.Wait()
	p.logger.Info("Queue processor stopped")
}

// processLoop continuously processes queued items
func (p *QueueProcessor) processLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Queue processor context cancelled")
			return
		case <-p.stopChan:
			p.logger.Info("Queue processor stop signal received")
			return
		case <-ticker.C:
			// Process all databases that have pending items
			if err := p.processPendingItems(ctx); err != nil {
				p.logger.Error("Error processing pending items", zap.Error(err))
			}
		}
	}
}

// processPendingItems processes pending queue items for all databases
func (p *QueueProcessor) processPendingItems(ctx context.Context) error {
	// Get all databases with pending items
	databaseIDs, err := p.queueRepo.GetDatabasesWithPending(ctx)
	if err != nil {
		return fmt.Errorf("failed to get databases with pending items: %w", err)
	}

	// Process each database
	for _, databaseID := range databaseIDs {
		if err := p.processNextItem(ctx, databaseID); err != nil {
			// Log error but continue processing other databases
			p.logger.Error("Failed to process next item for database",
				zap.String("database_id", databaseID.String()),
				zap.Error(err),
			)
		}
	}

	return nil
}

// processNextItem processes the next pending item for a specific database
func (p *QueueProcessor) processNextItem(ctx context.Context, databaseID uuid.UUID) error {
	// Try to mark the next pending item as InProgress
	queueItem, err := p.queueRepo.MarkInProgress(ctx, databaseID)
	if err != nil {
		if err == repository.ErrNotFound {
			// No pending items - this is fine
			return nil
		}
		return fmt.Errorf("failed to mark next item as in progress: %w", err)
	}

	// Process the item
	go func() {
		processCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := p.processQueueItem(processCtx, queueItem); err != nil {
			p.logger.Error("Failed to process queue item",
				zap.String("queue_item_id", queueItem.ID.String()),
				zap.String("change_plan_id", queueItem.ChangePlanID.String()),
				zap.Error(err),
			)
		}
	}()

	return nil
}

// processQueueItem processes a single queue item
func (p *QueueProcessor) processQueueItem(ctx context.Context, queueItem *models.ExecutionQueue) error {
	p.logger.Info("Processing queue item",
		zap.String("queue_item_id", queueItem.ID.String()),
		zap.String("change_plan_id", queueItem.ChangePlanID.String()),
		zap.String("database_id", queueItem.TargetDatabaseID.String()),
	)

	// Execute the Change Plan
	result, err := p.executionService.Execute(ctx, queueItem.ChangePlanID, queueItem.ExecutorID)
	if err != nil {
		// Check if it's a lock error - if so, put it back in pending state
		if err == ErrDatabaseLocked {
			p.logger.Warn("Database locked, requeuing item",
				zap.String("queue_item_id", queueItem.ID.String()),
				zap.String("database_id", queueItem.TargetDatabaseID.String()),
			)
			// Update status back to Pending
			if updateErr := p.queueRepo.UpdateStatus(ctx, queueItem.ID, models.QueueStatusPending); updateErr != nil {
				p.logger.Error("Failed to requeue item", zap.Error(updateErr))
			}
			return nil // Don't treat lock error as failure - will retry later
		}

		// Other errors - mark as failed
		errorMsg := err.Error()
		if updateErr := p.queueRepo.MarkFailed(ctx, queueItem.ID, errorMsg); updateErr != nil {
			p.logger.Error("Failed to mark queue item as failed", zap.Error(updateErr))
		}
		return fmt.Errorf("execution failed: %w", err)
	}

	// Execution succeeded
	if result.Success {
		if err := p.queueRepo.MarkCompleted(ctx, queueItem.ID); err != nil {
			p.logger.Error("Failed to mark queue item as completed", zap.Error(err))
			return err
		}

		p.logger.Info("Queue item processed successfully",
			zap.String("queue_item_id", queueItem.ID.String()),
			zap.String("change_plan_id", queueItem.ChangePlanID.String()),
			zap.Int("rows_affected", result.RowsAffected),
		)
	} else {
		// Execution failed
		errorMsg := "Execution failed"
		if result.ErrorMessage != nil {
			errorMsg = *result.ErrorMessage
		}
		if err := p.queueRepo.MarkFailed(ctx, queueItem.ID, errorMsg); err != nil {
			p.logger.Error("Failed to mark queue item as failed", zap.Error(err))
			return err
		}
	}

	return nil
}

// ProcessDatabase processes pending items for a specific database
func (p *QueueProcessor) ProcessDatabase(ctx context.Context, databaseID uuid.UUID) error {
	return p.processNextItem(ctx, databaseID)
}

// ProcessAll processes pending items for all databases
func (p *QueueProcessor) ProcessAll(ctx context.Context) error {
	return p.processPendingItems(ctx)
}
