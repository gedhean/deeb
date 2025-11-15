package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// DatabaseSchemaRepository defines the interface for database schema cache operations
type DatabaseSchemaRepository interface {
	// Upsert creates or updates the schema cache for a database
	Upsert(ctx context.Context, databaseID uuid.UUID, schemaData *models.DatabaseSchemaMetadata) error

	// GetByDatabaseID retrieves the cached schema for a database
	GetByDatabaseID(ctx context.Context, databaseID uuid.UUID) (*models.DatabaseSchemaMetadata, error)

	// Delete deletes the schema cache for a database
	Delete(ctx context.Context, databaseID uuid.UUID) error

	// GetStaleSchemas retrieves databases whose schema cache is older than the specified duration
	GetStaleSchemas(ctx context.Context, olderThan time.Duration) ([]uuid.UUID, error)

	// ListAll retrieves all cached schemas
	ListAll(ctx context.Context) ([]uuid.UUID, error)
}
