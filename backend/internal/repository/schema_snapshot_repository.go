package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// SchemaSnapshotRepository defines the interface for schema snapshot data operations
type SchemaSnapshotRepository interface {
	// Create creates a new schema snapshot in the database
	Create(ctx context.Context, snapshot *models.SchemaSnapshot) error

	// GetByID retrieves a schema snapshot by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.SchemaSnapshot, error)

	// GetByChangePlanID retrieves all snapshots for a specific change plan
	// Returns snapshots ordered by created_at DESC (most recent first)
	GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.SchemaSnapshot, error)

	// GetByDatabaseID retrieves all snapshots for a specific database
	// Returns snapshots ordered by created_at DESC (most recent first)
	GetByDatabaseID(ctx context.Context, databaseID uuid.UUID) ([]*models.SchemaSnapshot, error)
}
