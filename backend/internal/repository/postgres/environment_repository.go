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

// EnvironmentRepository is the PostgreSQL implementation of repository.EnvironmentRepository
type EnvironmentRepository struct {
	db *sqlx.DB
}

// NewEnvironmentRepository creates a new PostgreSQL environment repository
func NewEnvironmentRepository(db *sqlx.DB) *EnvironmentRepository {
	return &EnvironmentRepository{db: db}
}

// Create creates a new environment in the database
func (r *EnvironmentRepository) Create(ctx context.Context, environment *models.Environment) error {
	// Set timestamps
	now := time.Now()
	if environment.ID == uuid.Nil {
		environment.ID = uuid.New()
	}
	environment.CreatedAt = now
	environment.UpdatedAt = now

	query := `
		INSERT INTO environments (id, name, description, policy_id, allow_self_approval, created_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(
		ctx, query,
		environment.ID,
		environment.Name,
		environment.Description,
		environment.PolicyID,
		environment.AllowSelfApproval,
		environment.CreatedByID,
		environment.CreatedAt,
		environment.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves an environment by its ID
func (r *EnvironmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Environment, error) {
	query := `
		SELECT id, name, description, policy_id, allow_self_approval, created_by_id, created_at, updated_at
		FROM environments
		WHERE id = $1
	`

	var environment models.Environment
	err := r.db.GetContext(ctx, &environment, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &environment, nil
}

// environmentFilterConditions translates EnvironmentFilters into a reusable
// set of SQL conditions shared by List and Count.
func environmentFilterConditions(filters repository.EnvironmentFilters) *conditions {
	c := &conditions{}
	if filters.PolicyID != nil {
		c.equal("policy_id", *filters.PolicyID)
	}
	if filters.AllowSelfApproval != nil {
		c.equal("allow_self_approval", *filters.AllowSelfApproval)
	}
	if filters.CreatedAfter != nil {
		c.gte("created_at", *filters.CreatedAfter)
	}
	if filters.CreatedBefore != nil {
		c.lte("created_at", *filters.CreatedBefore)
	}
	c.search(filters.Search, "name")
	return c
}

// List retrieves all environments with pagination, applying the given filters.
func (r *EnvironmentRepository) List(
	ctx context.Context,
	filters repository.EnvironmentFilters,
	limit, offset int,
) ([]*models.Environment, error) {
	c := environmentFilterConditions(filters)
	where, args := c.clause()

	query := `SELECT id, name, description, policy_id, allow_self_approval, created_by_id, created_at, updated_at FROM environments`
	if where != "" {
		query += " WHERE " + where
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	var environments []*models.Environment
	err := r.db.SelectContext(ctx, &environments, query, args...)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no environments found
	if environments == nil {
		environments = []*models.Environment{}
	}

	return environments, nil
}

// Count returns the number of environments matching the given filters.
func (r *EnvironmentRepository) Count(ctx context.Context, filters repository.EnvironmentFilters) (int64, error) {
	c := environmentFilterConditions(filters)
	query := "SELECT COUNT(*) FROM environments" + c.whereClause()
	_, args := c.clause()

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Update updates an existing environment's information
func (r *EnvironmentRepository) Update(ctx context.Context, environment *models.Environment) error {
	// Update timestamp
	environment.UpdatedAt = time.Now()

	query := `
		UPDATE environments
		SET name = $2, description = $3, policy_id = $4, allow_self_approval = $5, updated_at = $6
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx, query,
		environment.ID,
		environment.Name,
		environment.Description,
		environment.PolicyID,
		environment.AllowSelfApproval,
		environment.UpdatedAt,
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

// Delete deletes an environment by its ID
func (r *EnvironmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM environments WHERE id = $1`

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
