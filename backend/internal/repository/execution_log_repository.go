package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// ExecutionLogRepository defines the interface for execution log data operations
type ExecutionLogRepository interface {
	// Create creates a new execution log entry
	Create(ctx context.Context, log *models.ExecutionLog) error

	// Update updates an existing execution log entry
	Update(ctx context.Context, log *models.ExecutionLog) error

	// GetByChangePlanID retrieves all execution logs for a specific change plan
	// Returns logs ordered by started_at DESC (most recent first)
	GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.ExecutionLog, error)

	// GetByID retrieves an execution log by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.ExecutionLog, error)

	// GetLatestByChangePlanID retrieves the most recent execution log for a change plan
	GetLatestByChangePlanID(ctx context.Context, changePlanID uuid.UUID) (*models.ExecutionLog, error)
}
