package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// DatabaseInstanceFilters holds optional filters for listing database
// instances. Zero-value fields (empty string / nil) are ignored, so the same
// struct drives both the unfiltered list and any combination of filters.
type DatabaseInstanceFilters struct {
	Search         string      // ILIKE over name, host
	EnvironmentIDs []uuid.UUID // environment_id IN (...) (empty = all)
	DatabaseEngine string      // comma-separated; database_engine IN (...) ("" = all)
	Status         string      // comma-separated; status IN (...) ("" = all)
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
}

// DatabaseInstanceRepository defines the interface for database instance data operations
type DatabaseInstanceRepository interface {
	// Create creates a new database instance in the database
	Create(ctx context.Context, instance *models.DatabaseInstance) error

	// GetByID retrieves a database instance by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.DatabaseInstance, error)

	// List retrieves database instances with pagination, applying the given filters.
	List(ctx context.Context, filters DatabaseInstanceFilters, limit, offset int) ([]*models.DatabaseInstance, error)

	// ListByEnvironment retrieves database instances filtered by environment ID with pagination.
	ListByEnvironment(ctx context.Context, environmentID uuid.UUID, limit, offset int) ([]*models.DatabaseInstance, error)

	// Count returns the number of database instances matching the given filters.
	Count(ctx context.Context, filters DatabaseInstanceFilters) (int64, error)

	// CountByEnvironment returns the number of database instances for a specific environment.
	CountByEnvironment(ctx context.Context, environmentID uuid.UUID) (int64, error)

	// Update updates an existing database instance's information
	Update(ctx context.Context, instance *models.DatabaseInstance) error

	// Delete deletes a database instance by its ID
	Delete(ctx context.Context, id uuid.UUID) error
}
