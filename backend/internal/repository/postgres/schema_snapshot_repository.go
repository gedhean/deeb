package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
)

// schemaSnapshotRepository implements the SchemaSnapshotRepository interface for PostgreSQL
type schemaSnapshotRepository struct {
	db *sqlx.DB
}

// NewSchemaSnapshotRepository creates a new PostgreSQL schema snapshot repository
func NewSchemaSnapshotRepository(db *sqlx.DB) repository.SchemaSnapshotRepository {
	return &schemaSnapshotRepository{db: db}
}

// Create creates a new schema snapshot in the database
func (r *schemaSnapshotRepository) Create(ctx context.Context, snapshot *models.SchemaSnapshot) error {
	if snapshot.ID == uuid.Nil {
		snapshot.ID = uuid.New()
	}

	now := time.Now()
	if snapshot.CreatedAt.IsZero() {
		snapshot.CreatedAt = now
	}

	query := `
		INSERT INTO schema_snapshots (
			id, database_id, change_plan_id, snapshot_type,
			snapshot_data, snapshot_data_raw, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		snapshot.ID,
		snapshot.DatabaseID,
		snapshot.ChangePlanID,
		snapshot.SnapshotType,
		snapshot.SnapshotData, // JSONB column - PostgreSQL handles JSONB automatically
		snapshot.SnapshotDataRaw,
		snapshot.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create schema snapshot: %w", err)
	}

	return nil
}

// GetByID retrieves a schema snapshot by its ID
func (r *schemaSnapshotRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SchemaSnapshot, error) {
	snapshot := &models.SchemaSnapshot{}

	query := `
		SELECT id, database_id, change_plan_id, snapshot_type,
		       snapshot_data, snapshot_data_raw, created_at
		FROM schema_snapshots
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, snapshot, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get schema snapshot by id: %w", err)
	}

	return snapshot, nil
}

// GetByChangePlanID retrieves all snapshots for a specific change plan
// Returns snapshots ordered by created_at DESC (most recent first)
func (r *schemaSnapshotRepository) GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.SchemaSnapshot, error) {
	snapshots := []*models.SchemaSnapshot{}

	query := `
		SELECT id, database_id, change_plan_id, snapshot_type,
		       snapshot_data, snapshot_data_raw, created_at
		FROM schema_snapshots
		WHERE change_plan_id = $1
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &snapshots, query, changePlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema snapshots by change plan id: %w", err)
	}

	return snapshots, nil
}

// GetByDatabaseID retrieves all snapshots for a specific database
// Returns snapshots ordered by created_at DESC (most recent first)
func (r *schemaSnapshotRepository) GetByDatabaseID(ctx context.Context, databaseID uuid.UUID) ([]*models.SchemaSnapshot, error) {
	snapshots := []*models.SchemaSnapshot{}

	query := `
		SELECT id, database_id, change_plan_id, snapshot_type,
		       snapshot_data, snapshot_data_raw, created_at
		FROM schema_snapshots
		WHERE database_id = $1
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &snapshots, query, databaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema snapshots by database id: %w", err)
	}

	return snapshots, nil
}

