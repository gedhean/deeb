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

// ChangelogRepository is the PostgreSQL implementation of repository.ChangelogRepository
type ChangelogRepository struct {
	db *sqlx.DB
}

// NewChangelogRepository creates a new PostgreSQL changelog repository
func NewChangelogRepository(db *sqlx.DB) *ChangelogRepository {
	return &ChangelogRepository{db: db}
}

// Create creates a new changelog entry in the database
func (r *ChangelogRepository) Create(ctx context.Context, changelog *models.Changelog) error {
	// Set timestamp and generate ID if needed
	now := time.Now()
	if changelog.ID == uuid.Nil {
		changelog.ID = uuid.New()
	}
	changelog.CreatedAt = now

	query := `
		INSERT INTO changelog (
			id, database_instance_id, database_id, change_plan_id, status,
			prev_schema_history_id, schema_history_id, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		changelog.ID,
		changelog.DatabaseInstanceID,
		changelog.DatabaseID,
		changelog.ChangePlanID,
		changelog.Status,
		changelog.PrevSchemaHistoryID,
		changelog.SchemaHistoryID,
		changelog.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a changelog entry by its ID
func (r *ChangelogRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Changelog, error) {
	query := `
		SELECT id, database_instance_id, database_id, change_plan_id, status,
			prev_schema_history_id, schema_history_id, created_at
		FROM changelog
		WHERE id = $1
	`

	var changelog models.Changelog
	err := r.db.GetContext(ctx, &changelog, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &changelog, nil
}

// GetByChangePlanID retrieves all changelog entries for a specific change plan
func (r *ChangelogRepository) GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.Changelog, error) {
	query := `
		SELECT id, database_instance_id, database_id, change_plan_id, status,
			prev_schema_history_id, schema_history_id, created_at
		FROM changelog
		WHERE change_plan_id = $1
		ORDER BY created_at DESC
	`

	var changelogs []*models.Changelog
	err := r.db.SelectContext(ctx, &changelogs, query, changePlanID)
	if err != nil {
		return nil, err
	}

	return changelogs, nil
}

// GetByDatabaseID retrieves all changelog entries for a specific database
func (r *ChangelogRepository) GetByDatabaseID(ctx context.Context, databaseID uuid.UUID) ([]*models.Changelog, error) {
	query := `
		SELECT id, database_instance_id, database_id, change_plan_id, status,
			prev_schema_history_id, schema_history_id, created_at
		FROM changelog
		WHERE database_id = $1
		ORDER BY created_at DESC
	`

	var changelogs []*models.Changelog
	err := r.db.SelectContext(ctx, &changelogs, query, databaseID)
	if err != nil {
		return nil, err
	}

	return changelogs, nil
}

// Update updates an existing changelog entry
func (r *ChangelogRepository) Update(ctx context.Context, changelog *models.Changelog) error {
	query := `
		UPDATE changelog
		SET status = $2,
			prev_schema_history_id = $3,
			schema_history_id = $4
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		changelog.ID,
		changelog.Status,
		changelog.PrevSchemaHistoryID,
		changelog.SchemaHistoryID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}
