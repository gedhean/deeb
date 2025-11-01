package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// EnvironmentRepository defines the interface for environment data operations
type EnvironmentRepository interface {
	// Create creates a new environment in the database
	Create(ctx context.Context, environment *models.Environment) error

	// GetByID retrieves an environment by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Environment, error)

	// List retrieves all environments with pagination
	List(ctx context.Context, limit, offset int) ([]*models.Environment, error)

	// Count returns the total number of environments
	Count(ctx context.Context) (int64, error)

	// Update updates an existing environment's information
	Update(ctx context.Context, environment *models.Environment) error

	// Delete deletes an environment by its ID
	Delete(ctx context.Context, id uuid.UUID) error
}

