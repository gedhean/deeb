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

// PolicyRepository is the PostgreSQL implementation of repository.PolicyRepository
type PolicyRepository struct {
	db *sqlx.DB
}

// NewPolicyRepository creates a new PostgreSQL policy repository
func NewPolicyRepository(db *sqlx.DB) *PolicyRepository {
	return &PolicyRepository{db: db}
}

// Create creates a new policy in the database
func (r *PolicyRepository) Create(ctx context.Context, policy *models.Policy) error {
	// Set timestamps
	now := time.Now()
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	policy.CreatedAt = now
	policy.UpdatedAt = now

	query := `
		INSERT INTO policies (id, name, description, type, policy_config, created_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(
		ctx, query,
		policy.ID,
		policy.Name,
		policy.Description,
		policy.Type,
		policy.PolicyConfig,
		policy.CreatedByID,
		policy.CreatedAt,
		policy.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a policy by its ID
func (r *PolicyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Policy, error) {
	query := `
		SELECT id, name, description, type, policy_config, created_by_id, created_at, updated_at
		FROM policies
		WHERE id = $1
	`

	var policy models.Policy
	err := r.db.GetContext(ctx, &policy, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &policy, nil
}

// policyFilterConditions translates PolicyFilters into a reusable set of SQL
// conditions shared by List and Count.
func policyFilterConditions(filters repository.PolicyFilters) *conditions {
	c := &conditions{}
	if filters.Type != "" {
		c.equal("type", filters.Type)
	}
	if filters.CreatedAfter != nil {
		c.gte("created_at", *filters.CreatedAfter)
	}
	if filters.CreatedBefore != nil {
		c.lte("created_at", *filters.CreatedBefore)
	}
	c.search(filters.Search, "name", "description")
	return c
}

// List retrieves all policies with pagination, applying the given filters.
func (r *PolicyRepository) List(
	ctx context.Context,
	filters repository.PolicyFilters,
	limit, offset int,
) ([]*models.Policy, error) {
	c := policyFilterConditions(filters)
	where, args := c.clause()

	query := `SELECT id, name, description, type, policy_config, created_by_id, created_at, updated_at FROM policies`
	if where != "" {
		query += " WHERE " + where
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	var policies []*models.Policy
	err := r.db.SelectContext(ctx, &policies, query, args...)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no policies found
	if policies == nil {
		policies = []*models.Policy{}
	}

	return policies, nil
}

// Count returns the number of policies matching the given filters.
func (r *PolicyRepository) Count(ctx context.Context, filters repository.PolicyFilters) (int64, error) {
	c := policyFilterConditions(filters)
	query := "SELECT COUNT(*) FROM policies" + c.whereClause()
	_, args := c.clause()

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetByType retrieves policies of a specific type with pagination.
func (r *PolicyRepository) GetByType(
	ctx context.Context,
	policyType models.PolicyType,
	limit, offset int,
) ([]*models.Policy, error) {
	query := `
		SELECT id, name, description, type, policy_config, created_by_id, created_at, updated_at
		FROM policies
		WHERE type = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var policies []*models.Policy
	err := r.db.SelectContext(ctx, &policies, query, policyType, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no policies found
	if policies == nil {
		policies = []*models.Policy{}
	}

	return policies, nil
}

// CountByType returns the number of policies of a specific type.
func (r *PolicyRepository) CountByType(ctx context.Context, policyType models.PolicyType) (int64, error) {
	query := "SELECT COUNT(*) FROM policies WHERE type = $1"

	var count int64
	err := r.db.QueryRowContext(ctx, query, policyType).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Update updates an existing policy's information
func (r *PolicyRepository) Update(ctx context.Context, policy *models.Policy) error {
	// Update timestamp
	policy.UpdatedAt = time.Now()

	query := `
		UPDATE policies
		SET name = $2, description = $3, policy_config = $4, updated_at = $5
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx, query,
		policy.ID,
		policy.Name,
		policy.Description,
		policy.PolicyConfig,
		policy.UpdatedAt,
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

// Delete deletes a policy by its ID
func (r *PolicyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM policies WHERE id = $1`

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
