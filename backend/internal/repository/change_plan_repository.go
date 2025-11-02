package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// ChangePlanListFilters defines filtering options for listing change plans
type ChangePlanListFilters struct {
	Limit      int                     // Maximum number of results to return
	Offset     int                     // Number of results to skip
	Status     models.ChangePlanStatus // Filter by status (empty for all)
	ProjectID  uuid.UUID               // Filter by project ID (Nil for all)
	DatabaseID uuid.UUID               // Filter by database ID (Nil for all)
	AuthorID   uuid.UUID               // Filter by author ID (Nil for all)
}

// ChangePlanRepository defines the interface for change plan data operations
type ChangePlanRepository interface {
	// Create creates a new change plan in the database
	Create(ctx context.Context, changePlan *models.ChangePlan) error

	// GetByID retrieves a change plan by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.ChangePlan, error)

	// Update updates an existing change plan's information
	Update(ctx context.Context, changePlan *models.ChangePlan) error

	// Delete deletes a change plan by its ID
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves all change plans with pagination
	List(ctx context.Context, limit, offset int) ([]*models.ChangePlan, error)

	// ListWithFilters retrieves change plans with combined filters
	ListWithFilters(ctx context.Context, filters ChangePlanListFilters) ([]*models.ChangePlan, error)

	// Count returns the total number of change plans
	Count(ctx context.Context) (int64, error)

	// CountWithFilters returns the total number of change plans matching filters
	CountWithFilters(ctx context.Context, filters ChangePlanListFilters) (int64, error)

	// GetByStatus retrieves change plans filtered by status
	GetByStatus(ctx context.Context, status models.ChangePlanStatus, limit, offset int) ([]*models.ChangePlan, error)

	// GetByProjectID retrieves change plans filtered by project ID
	GetByProjectID(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*models.ChangePlan, error)

	// GetByDatabaseID retrieves change plans filtered by database ID
	GetByDatabaseID(ctx context.Context, databaseID uuid.UUID, limit, offset int) ([]*models.ChangePlan, error)

	// GetByAuthorID retrieves change plans filtered by author ID
	GetByAuthorID(ctx context.Context, authorID uuid.UUID, limit, offset int) ([]*models.ChangePlan, error)
}
