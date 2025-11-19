package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// SchemaHistoryRepository defines the interface for schema history data operations
type SchemaHistoryRepository interface {
	// Create creates a new schema history entry in the database
	Create(ctx context.Context, history *models.SchemaHistory) error

	// GetByID retrieves a schema history entry by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.SchemaHistory, error)

	// GetByDatabaseID retrieves all schema history entries for a specific database
	// Results are ordered by created_at DESC (most recent first)
	GetByDatabaseID(ctx context.Context, databaseID uuid.UUID) ([]*models.SchemaHistory, error)

	// GetLatestByDatabaseID retrieves the most recent schema history entry for a specific database
	// Returns ErrNotFound if no schema history exists for the database
	GetLatestByDatabaseID(ctx context.Context, databaseID uuid.UUID) (*models.SchemaHistory, error)
}
