package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// PolicyFilters holds optional filters for listing policies. Zero-value fields
// (empty string / nil) are ignored, so the same struct drives both the
// unfiltered list and any combination of filters.
type PolicyFilters struct {
	Search        string            // ILIKE over name, description
	Type          models.PolicyType // "" = all; equals type
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// PolicyRepository defines the interface for policy data operations
type PolicyRepository interface {
	// Create creates a new policy in the database
	Create(ctx context.Context, policy *models.Policy) error

	// GetByID retrieves a policy by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Policy, error)

	// List retrieves policies with pagination, applying the given filters.
	List(ctx context.Context, filters PolicyFilters, limit, offset int) ([]*models.Policy, error)

	// Count returns the number of policies matching the given filters.
	Count(ctx context.Context, filters PolicyFilters) (int64, error)

	// GetByType retrieves policies of a specific type with pagination.
	GetByType(ctx context.Context, policyType models.PolicyType, limit, offset int) ([]*models.Policy, error)

	// CountByType returns the number of policies of a specific type.
	CountByType(ctx context.Context, policyType models.PolicyType) (int64, error)

	// Update updates an existing policy's information
	Update(ctx context.Context, policy *models.Policy) error

	// Delete deletes a policy by its ID
	Delete(ctx context.Context, id uuid.UUID) error
}
