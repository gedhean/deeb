package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// PolicyFindingRepository defines the interface for policy finding data operations
type PolicyFindingRepository interface {
	// Create creates a new policy finding in the database
	Create(ctx context.Context, finding *models.PolicyFinding) error

	// CreateBatch creates multiple policy findings in a single transaction
	CreateBatch(ctx context.Context, findings []*models.PolicyFinding) error

	// GetByChangePlanID retrieves all policy findings for a specific change plan
	GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.PolicyFinding, error)

	// DeleteByChangePlanID deletes all policy findings for a specific change plan
	DeleteByChangePlanID(ctx context.Context, changePlanID uuid.UUID) error

	// CountBySeverity counts policy findings by severity for a specific change plan
	CountBySeverity(ctx context.Context, changePlanID uuid.UUID) (map[string]int, error)
}
