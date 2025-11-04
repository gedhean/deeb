package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// ChangeHistoryRepository defines the interface for change history data operations
type ChangeHistoryRepository interface {
	// Create creates a new change history entry
	Create(ctx context.Context, changeHistory *models.ChangeHistory) error

	// GetByChangePlanID retrieves all change history entries for a specific change plan
	GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.ChangeHistory, error)

	// GetByChangePlanIDWithPagination retrieves change history entries for a change plan with pagination
	GetByChangePlanIDWithPagination(
		ctx context.Context,
		changePlanID uuid.UUID,
		limit, offset int,
	) ([]*models.ChangeHistory, error)

	// CountByChangePlanID returns the total number of change history entries for a change plan
	CountByChangePlanID(ctx context.Context, changePlanID uuid.UUID) (int64, error)
}
