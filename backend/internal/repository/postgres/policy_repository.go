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

	_, err := r.db.ExecContext(ctx, query,
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

// List retrieves all policies with pagination
func (r *PolicyRepository) List(ctx context.Context, limit, offset int) ([]*models.Policy, error) {
	query := `
		SELECT id, name, description, type, policy_config, created_by_id, created_at, updated_at
		FROM policies
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var policies []*models.Policy
	err := r.db.SelectContext(ctx, &policies, query, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no policies found
	if policies == nil {
		policies = []*models.Policy{}
	}

	return policies, nil
}

// Count returns the total number of policies
func (r *PolicyRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM policies`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetByType retrieves all policies of a specific type with pagination
func (r *PolicyRepository) GetByType(ctx context.Context, policyType models.PolicyType, limit, offset int) ([]*models.Policy, error) {
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

// CountByType returns the total number of policies of a specific type
func (r *PolicyRepository) CountByType(ctx context.Context, policyType models.PolicyType) (int64, error) {
	query := `SELECT COUNT(*) FROM policies WHERE type = $1`

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

	result, err := r.db.ExecContext(ctx, query,
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

