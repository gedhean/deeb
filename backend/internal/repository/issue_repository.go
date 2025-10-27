package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// ListFilters defines common filtering options for listing issues
type ListFilters struct {
	Limit         int                // Maximum number of results to return
	Offset        int                // Number of results to skip
	Status        models.IssueStatus // Filter by status (empty for all)
	AuthorID      uuid.UUID          // Filter by author (Nil for all)
	DatabaseID    uuid.UUID          // Filter by target database (Nil for all)
	ProjectID     uuid.UUID          // Filter by project (Nil for all)
	CreatedAfter  *time.Time         // Filter by issues created after this time
	CreatedBefore *time.Time         // Filter by issues created before this time
	SearchQuery   string             // Search query for title and description
	SortBy        string             // Field to sort by (default: created_at)
	SortOrder     string             // Sort order: "asc" or "desc" (default: desc)
}

// IssueRepository defines the interface for issue data operations
type IssueRepository interface {
	// Create creates a new issue in the database
	Create(ctx context.Context, issue *models.Issue) error

	// GetByID retrieves an issue by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Issue, error)

	// Update updates an existing issue
	Update(ctx context.Context, issue *models.Issue) error

	// Delete deletes an issue by its ID
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves a list of issues with optional filters
	List(ctx context.Context, filters ListFilters) ([]*models.Issue, error)

	// Search searches for issues based on a query string
	// The query searches in title, description, and sql_content fields
	Search(ctx context.Context, query string, filters ListFilters) ([]*models.Issue, error)

	// Count returns the total number of issues matching the filters
	Count(ctx context.Context, filters ListFilters) (int64, error)
}
