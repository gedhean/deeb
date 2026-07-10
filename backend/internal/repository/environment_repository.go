package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// EnvironmentFilters holds optional filters for listing environments.
// Zero-value fields (empty string / nil) are ignored, so the same struct
// drives both the unfiltered list and any combination of filters.
type EnvironmentFilters struct {
	Search            string     // ILIKE over name
	PolicyID          *uuid.UUID // equals policy_id
	AllowSelfApproval *bool      // equals allow_self_approval
	CreatedAfter      *time.Time
	CreatedBefore     *time.Time
}

// EnvironmentRepository defines the interface for environment data operations
type EnvironmentRepository interface {
	// Create creates a new environment in the database
	Create(ctx context.Context, environment *models.Environment) error

	// GetByID retrieves an environment by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Environment, error)

	// List retrieves environments with pagination, applying the given filters.
	List(ctx context.Context, filters EnvironmentFilters, limit, offset int) ([]*models.Environment, error)

	// Count returns the number of environments matching the given filters.
	Count(ctx context.Context, filters EnvironmentFilters) (int64, error)

	// Update updates an existing environment's information
	Update(ctx context.Context, environment *models.Environment) error

	// Delete deletes an environment by its ID
	Delete(ctx context.Context, id uuid.UUID) error
}
