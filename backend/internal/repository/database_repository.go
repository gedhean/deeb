package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// DatabaseFilters holds optional filters for listing databases. Zero-value
// fields (empty string / nil) are ignored, so the same struct drives both the
// unfiltered list and any combination of filters.
type DatabaseFilters struct {
	// Search is a case-insensitive substring matched over database_name and description.
	Search string
	// ProjectID, EnvironmentID and InstanceID restrict results to a given relation.
	ProjectID     *uuid.UUID
	EnvironmentID *uuid.UUID
	InstanceID    *uuid.UUID
	// Status restricts results to a specific database status (e.g. Active).
	Status string
	// CreatedAfter / CreatedBefore restrict results to a created_at range.
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// DatabaseRepository defines the interface for database data operations
type DatabaseRepository interface {
	// Create creates a new database in the database
	Create(ctx context.Context, database *models.Database) error

	// GetByID retrieves a database by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Database, error)

	// List retrieves databases with pagination, applying the given filters.
	List(ctx context.Context, filters DatabaseFilters, limit, offset int) ([]*models.Database, error)

	// Count returns the number of databases matching the given filters.
	Count(ctx context.Context, filters DatabaseFilters) (int, error)

	// CountByProject returns the total number of databases for a specific project
	CountByProject(ctx context.Context, projectID uuid.UUID) (int, error)

	// CountByInstanceID returns the total number of databases for a specific database instance
	CountByInstanceID(ctx context.Context, instanceID uuid.UUID) (int, error)

	// CountByEnvironmentID returns the total number of databases for a specific environment
	CountByEnvironmentID(ctx context.Context, environmentID uuid.UUID) (int, error)

	// GetByProjectID retrieves databases for a specific project (with pagination)
	GetByProjectID(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*models.Database, error)

	// ListByProject retrieves databases for a specific project (alias for GetByProjectID)
	ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*models.Database, error)

	// ListByInstanceID retrieves databases for a specific database instance
	ListByInstanceID(ctx context.Context, instanceID uuid.UUID, limit, offset int) ([]*models.Database, error)

	// ListByEnvironmentID retrieves databases for a specific environment (with pagination)
	ListByEnvironmentID(ctx context.Context, environmentID uuid.UUID, limit, offset int) ([]*models.Database, error)

	// Update updates an existing database's information
	Update(ctx context.Context, database *models.Database) error

	// Delete deletes a database by its ID
	Delete(ctx context.Context, id uuid.UUID) error

	// GetByInstanceAndName retrieves a database by instance ID and database name
	GetByInstanceAndName(ctx context.Context, instanceID uuid.UUID, databaseName string) (*models.Database, error)

	// ListActiveForSync retrieves all active databases (excludes Inactive and Pending)
	// Used by schema sync job to determine which databases should have metadata synced
	ListActiveForSync(ctx context.Context, limit, offset int) ([]*models.Database, error)
}
