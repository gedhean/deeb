package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
)

// SchemaHistoryRepository is the PostgreSQL implementation of repository.SchemaHistoryRepository
type SchemaHistoryRepository struct {
	db *sqlx.DB
}

// NewSchemaHistoryRepository creates a new PostgreSQL schema history repository
func NewSchemaHistoryRepository(db *sqlx.DB) *SchemaHistoryRepository {
	return &SchemaHistoryRepository{db: db}
}

// Create creates a new schema history entry in the database
func (r *SchemaHistoryRepository) Create(ctx context.Context, history *models.SchemaHistory) error {
	// Set timestamp and generate ID if needed
	now := time.Now()
	if history.ID == uuid.Nil {
		history.ID = uuid.New()
	}
	history.CreatedAt = now

	query := `
		INSERT INTO schema_history (
			id, database_instance_id, database_id, metadata, raw_dump, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		history.ID,
		history.DatabaseInstanceID,
		history.DatabaseID,
		history.Metadata,
		history.RawDump,
		history.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a schema history entry by its ID
func (r *SchemaHistoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SchemaHistory, error) {
	query := `
		SELECT id, database_instance_id, database_id, metadata, raw_dump, created_at
		FROM schema_history
		WHERE id = $1
	`

	var history models.SchemaHistory
	err := r.db.GetContext(ctx, &history, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &history, nil
}

// GetByDatabaseID retrieves all schema history entries for a specific database
func (r *SchemaHistoryRepository) GetByDatabaseID(ctx context.Context, databaseID uuid.UUID) ([]*models.SchemaHistory, error) {
	query := `
		SELECT id, database_instance_id, database_id, metadata, raw_dump, created_at
		FROM schema_history
		WHERE database_id = $1
		ORDER BY created_at DESC
	`

	var histories []*models.SchemaHistory
	err := r.db.SelectContext(ctx, &histories, query, databaseID)
	if err != nil {
		return nil, err
	}

	return histories, nil
}

// GetLatestByDatabaseID retrieves the most recent schema history entry for a specific database
func (r *SchemaHistoryRepository) GetLatestByDatabaseID(
	ctx context.Context,
	databaseID uuid.UUID,
) (*models.SchemaHistory, error) {
	query := `
		SELECT id, database_instance_id, database_id, metadata, raw_dump, created_at
		FROM schema_history
		WHERE database_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var history models.SchemaHistory
	err := r.db.GetContext(ctx, &history, query, databaseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &history, nil
}
