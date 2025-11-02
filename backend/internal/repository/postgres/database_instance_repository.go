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
	db            *sqlx.DB
	encryptionKey string
}

// NewDatabaseInstanceRepository creates a new PostgreSQL database instance repository
func NewDatabaseInstanceRepository(db *sqlx.DB, encryptionKey string) *DatabaseInstanceRepository {
	return &DatabaseInstanceRepository{
		db:            db,
		encryptionKey: encryptionKey,
	}
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

	// Note: Encryption should be handled by the handler layer before calling Create
	// The repository just stores the encrypted string as-is

	query := `
		INSERT INTO database_instances (
			id, name, description, database_type, host, port, username,
			connection_details_encrypted, use_rds_iam_auth, require_ssl,
			status, created_by_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.ExecContext(ctx, query,
		instance.ID,
		instance.Name,
		instance.Description,
		instance.DatabaseType,
		instance.Host,
		instance.Port,
		instance.Username,
		instance.ConnectionDetailsEncrypted,
		instance.UseRDSIAMAuth,
		instance.RequireSSL,
		instance.Status,
		instance.CreatedByID,
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
		SELECT id, name, description, database_type, host, port, username,
		       connection_details_encrypted, use_rds_iam_auth, require_ssl,
		       status, created_by_id, created_at, updated_at
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

	// Note: Decryption is handled by handler layer when needed
	// Repository just returns the encrypted string as-is

	return &instance, nil
}

// List retrieves all database instances with pagination
func (r *DatabaseInstanceRepository) List(ctx context.Context, limit, offset int) ([]*models.DatabaseInstance, error) {
	query := `
		SELECT id, name, description, database_type, host, port, username,
		       connection_details_encrypted, use_rds_iam_auth, require_ssl,
		       status, created_by_id, created_at, updated_at
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

// Count returns the total number of database instances
func (r *DatabaseInstanceRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM database_instances`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Update updates an existing database instance's information
func (r *DatabaseInstanceRepository) Update(ctx context.Context, instance *models.DatabaseInstance) error {
	// Validate instance
	if err := instance.Validate(); err != nil {
		return err
	}

	// Update timestamp
	instance.UpdatedAt = time.Now()

	// Note: Encryption should be handled by the handler layer before calling Update
	// The repository just stores the encrypted string as-is

	query := `
		UPDATE database_instances
		SET name = $2, description = $3, database_type = $4, host = $5, port = $6,
		    username = $7, connection_details_encrypted = $8, use_rds_iam_auth = $9,
		    require_ssl = $10, status = $11, updated_at = $12
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		instance.ID,
		instance.Name,
		instance.Description,
		instance.DatabaseType,
		instance.Host,
		instance.Port,
		instance.Username,
		instance.ConnectionDetailsEncrypted,
		instance.UseRDSIAMAuth,
		instance.RequireSSL,
		instance.Status,
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
