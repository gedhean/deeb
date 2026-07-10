package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// ProjectFilters holds optional filters for listing projects. Zero-value
// fields (empty string / nil) are ignored, so the same struct drives both the
// unfiltered list and any combination of filters.
type ProjectFilters struct {
	Search        string // ILIKE over name, description
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// ProjectRepository defines the interface for project data operations
type ProjectRepository interface {
	// Create creates a new project in the database
	Create(ctx context.Context, project *models.Project) error

	// GetByID retrieves a project by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error)

	// List retrieves projects with pagination, applying the given filters.
	List(ctx context.Context, filters ProjectFilters, limit, offset int) ([]*models.Project, error)

	// ListByUserID retrieves projects where the user is a member with pagination,
	// applying the given filters.
	ListByUserID(ctx context.Context, userID uuid.UUID, filters ProjectFilters, limit, offset int) ([]*models.Project, error)

	// Count returns the number of projects matching the given filters.
	Count(ctx context.Context, filters ProjectFilters) (int64, error)

	// CountByUserID returns the number of projects where the user is a member,
	// matching the given filters.
	CountByUserID(ctx context.Context, userID uuid.UUID, filters ProjectFilters) (int64, error)

	// Update updates an existing project's information
	Update(ctx context.Context, project *models.Project) error

	// Delete deletes a project by its ID
	Delete(ctx context.Context, id uuid.UUID) error
}
