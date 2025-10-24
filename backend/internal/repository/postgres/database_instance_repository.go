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

// DatabaseInstanceRepository is the PostgreSQL implementation of repository.DatabaseInstanceRepository
type DatabaseInstanceRepository struct {
	db *sqlx.DB
}

// NewDatabaseInstanceRepository creates a new PostgreSQL database instance repository
func NewDatabaseInstanceRepository(db *sqlx.DB) *DatabaseInstanceRepository {
	return &DatabaseInstanceRepository{db: db}
}

// Create creates a new database instance in the database
func (r *DatabaseInstanceRepository) Create(ctx context.Context, instance *models.DatabaseInstance) error {
	// Validate instance
	if err := instance.Validate(); err != nil {
		return err
	}

	// Set timestamps
	now := time.Now()
	if instance.ID == uuid.Nil {
		instance.ID = uuid.New()
	}
	instance.CreatedAt = now
	instance.UpdatedAt = now

	query := `
		INSERT INTO database_instances (id, name, host, port, engine, environment_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		instance.ID,
		instance.Name,
		instance.Host,
		instance.Port,
		instance.Engine,
		instance.EnvironmentID,
		instance.CreatedAt,
		instance.UpdatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a database instance by its ID
func (r *DatabaseInstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DatabaseInstance, error) {
	query := `
		SELECT id, name, host, port, engine, environment_id, created_at, updated_at
		FROM database_instances
		WHERE id = $1
	`

	var instance models.DatabaseInstance
	err := r.db.GetContext(ctx, &instance, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &instance, nil
}

// List retrieves all database instances with pagination
func (r *DatabaseInstanceRepository) List(ctx context.Context, limit, offset int) ([]*models.DatabaseInstance, error) {
	query := `
		SELECT id, name, host, port, engine, environment_id, created_at, updated_at
		FROM database_instances
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var instances []*models.DatabaseInstance
	err := r.db.SelectContext(ctx, &instances, query, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no instances found
	if instances == nil {
		instances = []*models.DatabaseInstance{}
	}

	return instances, nil
}

// Update updates an existing database instance's information
func (r *DatabaseInstanceRepository) Update(ctx context.Context, instance *models.DatabaseInstance) error {
	// Validate instance
	if err := instance.Validate(); err != nil {
		return err
	}

	// Update timestamp
	instance.UpdatedAt = time.Now()

	query := `
		UPDATE database_instances
		SET name = $2, host = $3, port = $4, engine = $5, environment_id = $6, updated_at = $7
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		instance.ID,
		instance.Name,
		instance.Host,
		instance.Port,
		instance.Engine,
		instance.EnvironmentID,
		instance.UpdatedAt,
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

// Delete deletes a database instance by its ID
func (r *DatabaseInstanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM database_instances WHERE id = $1`

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
