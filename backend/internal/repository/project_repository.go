package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// ProjectRepository defines the interface for project data operations
type ProjectRepository interface {
	// Create creates a new project in the database
	Create(ctx context.Context, project *models.Project) error

	// GetByID retrieves a project by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error)

	// List retrieves all projects with pagination
	List(ctx context.Context, limit, offset int) ([]*models.Project, error)

	// Count returns the total number of projects
	Count(ctx context.Context) (int64, error)

	// Update updates an existing project's information
	Update(ctx context.Context, project *models.Project) error

	// Delete deletes a project by its ID
	Delete(ctx context.Context, id uuid.UUID) error
}
