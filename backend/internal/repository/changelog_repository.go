package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// ChangelogRepository defines the interface for changelog data operations
type ChangelogRepository interface {
	// Create creates a new changelog entry in the database
	Create(ctx context.Context, changelog *models.Changelog) error

	// GetByID retrieves a changelog entry by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Changelog, error)

	// GetByChangePlanID retrieves all changelog entries for a specific change plan
	// Results are ordered by created_at DESC (most recent first)
	GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.Changelog, error)

	// GetByDatabaseID retrieves all changelog entries for a specific database
	// Results are ordered by created_at DESC (most recent first)
	GetByDatabaseID(ctx context.Context, databaseID uuid.UUID) ([]*models.Changelog, error)

	// Update updates an existing changelog entry
	// Typically used to update status and schema_history_id after execution
	Update(ctx context.Context, changelog *models.Changelog) error
}
