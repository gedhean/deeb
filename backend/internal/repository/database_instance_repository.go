package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// DatabaseInstanceRepository defines the interface for database instance data operations
type DatabaseInstanceRepository interface {
	// Create creates a new database instance in the database
	Create(ctx context.Context, instance *models.DatabaseInstance) error

	// GetByID retrieves a database instance by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.DatabaseInstance, error)

	// List retrieves all database instances with pagination
	List(ctx context.Context, limit, offset int) ([]*models.DatabaseInstance, error)

	// Update updates an existing database instance's information
	Update(ctx context.Context, instance *models.DatabaseInstance) error

	// Delete deletes a database instance by its ID
	Delete(ctx context.Context, id uuid.UUID) error
}
