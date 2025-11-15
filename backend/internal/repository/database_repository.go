package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// DatabaseRepository defines the interface for database data operations
type DatabaseRepository interface {
	// Create creates a new database in the database
	Create(ctx context.Context, database *models.Database) error

	// GetByID retrieves a database by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Database, error)

	// List retrieves all databases with pagination
	List(ctx context.Context, limit, offset int) ([]*models.Database, error)

	// Count returns the total number of databases
	Count(ctx context.Context) (int, error)

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
