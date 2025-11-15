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

// DatabaseRepository is the PostgreSQL implementation of repository.DatabaseRepository
type DatabaseRepository struct {
	db *sqlx.DB
}

// NewDatabaseRepository creates a new PostgreSQL database repository
func NewDatabaseRepository(db *sqlx.DB) *DatabaseRepository {
	return &DatabaseRepository{db: db}
}

// Create creates a new database in the database
func (r *DatabaseRepository) Create(ctx context.Context, database *models.Database) error {
	// Validate database
	if err := database.Validate(); err != nil {
		return err
	}

	// Set timestamps
	now := time.Now()
	if database.ID == uuid.Nil {
		database.ID = uuid.New()
	}
	database.CreatedAt = now
	database.UpdatedAt = now

	query := `
		INSERT INTO databases (
			id, project_id, database_instance_id, environment_id,
			database_name, description, status, created_by_id,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		database.ID,
		database.ProjectID,
		database.DatabaseInstanceID,
		database.EnvironmentID,
		database.DatabaseName,
		database.Description,
		database.Status,
		database.CreatedByID,
		database.CreatedAt,
		database.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a database by its ID
func (r *DatabaseRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Database, error) {
	query := `
		SELECT id, project_id, database_instance_id, environment_id,
		       database_name, description, status, created_by_id,
		       created_at, updated_at
		FROM databases
		WHERE id = $1
	`

	var database models.Database
	err := r.db.GetContext(ctx, &database, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &database, nil
}

// List retrieves all databases with pagination
func (r *DatabaseRepository) List(ctx context.Context, limit, offset int) ([]*models.Database, error) {
	query := `
		SELECT id, project_id, database_instance_id, environment_id,
		       database_name, description, status, created_by_id,
		       created_at, updated_at
		FROM databases
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var databases []*models.Database
	err := r.db.SelectContext(ctx, &databases, query, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no databases found
	if databases == nil {
		databases = []*models.Database{}
	}

	return databases, nil
}

// Count returns the total number of databases
func (r *DatabaseRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM databases`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetByProjectID retrieves databases for a specific project (with pagination)
func (r *DatabaseRepository) GetByProjectID(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	return r.ListByProject(ctx, projectID, limit, offset)
}

// ListByProject retrieves databases for a specific project
func (r *DatabaseRepository) ListByProject(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	query := `
		SELECT id, project_id, database_instance_id, environment_id,
		       database_name, description, status, created_by_id,
		       created_at, updated_at
		FROM databases
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var databases []*models.Database
	err := r.db.SelectContext(ctx, &databases, query, projectID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no databases found
	if databases == nil {
		databases = []*models.Database{}
	}

	return databases, nil
}

// ListByInstanceID retrieves databases for a specific database instance
func (r *DatabaseRepository) ListByInstanceID(
	ctx context.Context,
	instanceID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	query := `
		SELECT id, project_id, database_instance_id, environment_id,
		       database_name, description, status, created_by_id,
		       created_at, updated_at
		FROM databases
		WHERE database_instance_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var databases []*models.Database
	err := r.db.SelectContext(ctx, &databases, query, instanceID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no databases found
	if databases == nil {
		databases = []*models.Database{}
	}

	return databases, nil
}

// ListByEnvironmentID retrieves databases for a specific environment
func (r *DatabaseRepository) ListByEnvironmentID(
	ctx context.Context,
	environmentID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	query := `
		SELECT id, project_id, database_instance_id, environment_id,
		       database_name, description, status, created_by_id,
		       created_at, updated_at
		FROM databases
		WHERE environment_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var databases []*models.Database
	err := r.db.SelectContext(ctx, &databases, query, environmentID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no databases found
	if databases == nil {
		databases = []*models.Database{}
	}

	return databases, nil
}

// Update updates an existing database's information
func (r *DatabaseRepository) Update(ctx context.Context, database *models.Database) error {
	// Validate database
	if err := database.Validate(); err != nil {
		return err
	}

	// Update timestamp
	database.UpdatedAt = time.Now()

	query := `
		UPDATE databases
		SET database_name = $2, description = $3, status = $4, updated_at = $5
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		database.ID,
		database.DatabaseName,
		database.Description,
		database.Status,
		database.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// Delete deletes a database by its ID
func (r *DatabaseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM databases WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// GetByInstanceAndName retrieves a database by instance ID and database name
func (r *DatabaseRepository) GetByInstanceAndName(
	ctx context.Context,
	instanceID uuid.UUID,
	databaseName string,
) (*models.Database, error) {
	query := `
		SELECT id, project_id, database_instance_id, environment_id,
		       database_name, description, status, created_by_id,
		       created_at, updated_at
		FROM databases
		WHERE database_instance_id = $1 AND database_name = $2
	`

	var database models.Database
	err := r.db.GetContext(ctx, &database, query, instanceID, databaseName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &database, nil
}

// ListActiveForSync retrieves all active databases (excludes Inactive and Pending)
// Used by schema sync job to determine which databases should have metadata synced
func (r *DatabaseRepository) ListActiveForSync(ctx context.Context, limit, offset int) ([]*models.Database, error) {
	query := `
		SELECT id, project_id, database_instance_id, environment_id,
		       database_name, description, status, created_by_id,
		       created_at, updated_at
		FROM databases
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var databases []*models.Database
	err := r.db.SelectContext(ctx, &databases, query, models.DatabaseStatusActive, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no databases found
	if databases == nil {
		databases = []*models.Database{}
	}

	return databases, nil
}
