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

	// ListByProject retrieves databases for a specific project
	ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*models.Database, error)

	// Update updates an existing database's information
	Update(ctx context.Context, database *models.Database) error

	// Delete deletes a database by its ID
	Delete(ctx context.Context, id uuid.UUID) error
}
