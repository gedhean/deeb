package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
			id, name, description, database_engine, host, port, username,
			connection_details_encrypted, use_rds_iam_auth, require_ssl,
			status, environment_id, created_by_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := r.db.ExecContext(
		ctx, query,
		instance.ID,
		instance.Name,
		instance.Description,
		instance.DatabaseEngine,
		instance.Host,
		instance.Port,
		instance.Username,
		instance.ConnectionDetailsEncrypted,
		instance.UseRDSIAMAuth,
		instance.RequireSSL,
		instance.Status,
		instance.EnvironmentID,
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
		SELECT id, name, description, database_engine, host, port, username,
		       connection_details_encrypted, use_rds_iam_auth, require_ssl,
		       status, environment_id, created_by_id, created_at, updated_at
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

// databaseInstanceFilterConditions translates DatabaseInstanceFilters into a
// reusable set of SQL conditions shared by List and Count.
func databaseInstanceFilterConditions(filters repository.DatabaseInstanceFilters) *conditions {
	c := &conditions{}
	if len(filters.EnvironmentIDs) > 0 {
		ids := make([]interface{}, len(filters.EnvironmentIDs))
		for i, id := range filters.EnvironmentIDs {
			ids[i] = id
		}
		c.in("environment_id", ids)
	}
	// Engine and status are multi-select: comma-separated -> IN (...).
	c.inCSV("database_engine", filters.DatabaseEngine)
	c.inCSV("status", filters.Status)
	if filters.CreatedAfter != nil {
		c.gte("created_at", *filters.CreatedAfter)
	}
	if filters.CreatedBefore != nil {
		c.lte("created_at", *filters.CreatedBefore)
	}
	c.search(filters.Search, "name", "host")
	return c
}

// List retrieves all database instances with pagination, applying the given filters.
func (r *DatabaseInstanceRepository) List(
	ctx context.Context,
	filters repository.DatabaseInstanceFilters,
	limit, offset int,
) ([]*models.DatabaseInstance, error) {
	c := databaseInstanceFilterConditions(filters)
	where, args := c.clause()

	query := `
		SELECT id, name, description, database_engine, host, port, username,
		       connection_details_encrypted, use_rds_iam_auth, require_ssl,
		       status, environment_id, created_by_id, created_at, updated_at
		FROM database_instances`
	if where != "" {
		query += " WHERE " + where
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	var instances []*models.DatabaseInstance
	err := r.db.SelectContext(ctx, &instances, query, args...)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no instances found
	if instances == nil {
		instances = []*models.DatabaseInstance{}
	}

	return instances, nil
}

// ListByEnvironment retrieves database instances filtered by environment ID with pagination.
func (r *DatabaseInstanceRepository) ListByEnvironment(
	ctx context.Context,
	environmentID uuid.UUID,
	limit, offset int,
) ([]*models.DatabaseInstance, error) {
	query := `
		SELECT id, name, description, database_engine, host, port, username,
		       connection_details_encrypted, use_rds_iam_auth, require_ssl,
		       status, environment_id, created_by_id, created_at, updated_at
		FROM database_instances
		WHERE environment_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var instances []*models.DatabaseInstance
	err := r.db.SelectContext(ctx, &instances, query, environmentID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no instances found
	if instances == nil {
		instances = []*models.DatabaseInstance{}
	}

	return instances, nil
}

// Count returns the number of database instances matching the given filters.
func (r *DatabaseInstanceRepository) Count(ctx context.Context, filters repository.DatabaseInstanceFilters) (int64, error) {
	c := databaseInstanceFilterConditions(filters)
	query := "SELECT COUNT(*) FROM database_instances" + c.whereClause()
	_, args := c.clause()

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CountByEnvironment returns the number of database instances for a specific environment.
func (r *DatabaseInstanceRepository) CountByEnvironment(ctx context.Context, environmentID uuid.UUID) (int64, error) {
	query := "SELECT COUNT(*) FROM database_instances WHERE environment_id = $1"

	var count int64
	err := r.db.QueryRowContext(ctx, query, environmentID).Scan(&count)
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
		SET name = $2, description = $3, database_engine = $4, host = $5, port = $6,
		    username = $7, connection_details_encrypted = $8, use_rds_iam_auth = $9,
		    require_ssl = $10, status = $11, environment_id = $12, updated_at = $13
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx, query,
		instance.ID,
		instance.Name,
		instance.Description,
		instance.DatabaseEngine,
		instance.Host,
		instance.Port,
		instance.Username,
		instance.ConnectionDetailsEncrypted,
		instance.UseRDSIAMAuth,
		instance.RequireSSL,
		instance.Status,
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
