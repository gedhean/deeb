package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
)

// databaseSchemaRepository implements DatabaseSchemaRepository for PostgreSQL
type databaseSchemaRepository struct {
	db *sqlx.DB
}

// NewDatabaseSchemaRepository creates a new PostgreSQL database schema repository
func NewDatabaseSchemaRepository(db *sqlx.DB) repository.DatabaseSchemaRepository {
	return &databaseSchemaRepository{db: db}
}

// databaseSchemaRow represents a row in the database_schema table
type databaseSchemaRow struct {
	ID         uuid.UUID `db:"id"`
	DatabaseID uuid.UUID `db:"database_id"`
	SchemaData []byte    `db:"schema_data"`
	SyncedAt   time.Time `db:"synced_at"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

// Upsert creates or updates the schema cache for a database
func (r *databaseSchemaRepository) Upsert(
	ctx context.Context,
	databaseID uuid.UUID,
	schemaData *models.DatabaseSchemaMetadata,
) error {
	// Marshal schema data to JSON
	schemaJSON, err := json.Marshal(schemaData)
	if err != nil {
		return fmt.Errorf("failed to marshal schema data: %w", err)
	}

	query := `
		INSERT INTO database_schema (database_id, schema_data, synced_at, updated_at)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT (database_id)
		DO UPDATE SET
			schema_data = EXCLUDED.schema_data,
			synced_at = EXCLUDED.synced_at,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	_, err = r.db.ExecContext(ctx, query, databaseID, schemaJSON, now)
	if err != nil {
		return fmt.Errorf("failed to upsert schema cache: %w", err)
	}

	return nil
}

// GetByDatabaseID retrieves the cached schema for a database
func (r *databaseSchemaRepository) GetByDatabaseID(
	ctx context.Context,
	databaseID uuid.UUID,
) (*models.DatabaseSchemaMetadata, error) {
	query := `
		SELECT id, database_id, schema_data, synced_at, created_at, updated_at
		FROM database_schema
		WHERE database_id = $1
	`

	var row databaseSchemaRow
	err := r.db.GetContext(ctx, &row, query, databaseID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get schema cache: %w", err)
	}

	// Unmarshal schema data from JSON
	var schemaData models.DatabaseSchemaMetadata
	if err := json.Unmarshal(row.SchemaData, &schemaData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema data: %w", err)
	}

	// Set last synced timestamp
	schemaData.LastSyncedAt = &row.SyncedAt

	return &schemaData, nil
}

// Delete deletes the schema cache for a database
func (r *databaseSchemaRepository) Delete(ctx context.Context, databaseID uuid.UUID) error {
	query := `DELETE FROM database_schema WHERE database_id = $1`

	result, err := r.db.ExecContext(ctx, query, databaseID)
	if err != nil {
		return fmt.Errorf("failed to delete schema cache: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// GetStaleSchemas retrieves databases whose schema cache is older than the specified duration
func (r *databaseSchemaRepository) GetStaleSchemas(
	ctx context.Context,
	olderThan time.Duration,
) ([]uuid.UUID, error) {
	query := `
		SELECT database_id
		FROM database_schema
		WHERE synced_at < $1
		ORDER BY synced_at ASC
	`

	threshold := time.Now().Add(-olderThan)
	var databaseIDs []uuid.UUID
	err := r.db.SelectContext(ctx, &databaseIDs, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get stale schemas: %w", err)
	}

	return databaseIDs, nil
}

// ListAll retrieves all cached schemas
func (r *databaseSchemaRepository) ListAll(ctx context.Context) ([]uuid.UUID, error) {
	query := `
		SELECT database_id
		FROM database_schema
		ORDER BY updated_at DESC
	`

	var databaseIDs []uuid.UUID
	err := r.db.SelectContext(ctx, &databaseIDs, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all schemas: %w", err)
	}

	return databaseIDs, nil
}
