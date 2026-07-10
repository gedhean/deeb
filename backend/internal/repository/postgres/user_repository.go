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

// userRepository implements the UserRepository interface for PostgreSQL
type userRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new PostgreSQL user repository
func NewUserRepository(db *sqlx.DB) repository.UserRepository {
	return &userRepository{db: db}
}

// Create creates a new user in the database
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	query := `
		INSERT INTO users (
			id, email, password_hash, display_name, organization_role,
			two_factor_enabled, two_factor_secret, is_active,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.OrganizationRole,
		user.TwoFactorEnabled,
		user.TwoFactorSecret,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by their ID
func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User

	query := `
		SELECT id, email, password_hash, display_name, organization_role,
			   is_active, oauth_provider, oauth_id,
			   two_factor_enabled, two_factor_secret,
			   last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by their email address
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User

	query := `
		SELECT id, email, password_hash, display_name, organization_role,
			   is_active, oauth_provider, oauth_id,
			   two_factor_enabled, two_factor_secret,
			   last_login_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// Update updates an existing user's information
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users
		SET email = $2,
			password_hash = $3,
			display_name = $4,
			organization_role = $5,
			two_factor_enabled = $6,
			two_factor_secret = $7,
			is_active = $8,
			updated_at = $9
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.OrganizationRole,
		user.TwoFactorEnabled,
		user.TwoFactorSecret,
		user.IsActive,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// Delete deletes a user by their ID
func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// userFilterConditions translates UserFilters into a reusable set of SQL
// conditions shared by List and Count.
func userFilterConditions(filters repository.UserFilters) *conditions {
	c := &conditions{}
	// Organization role is a multi-select: comma-separated -> IN (...).
	c.inCSV("organization_role", filters.OrganizationRole)
	if filters.IsActive != nil {
		c.equal("is_active", *filters.IsActive)
	}
	if filters.CreatedAfter != nil {
		c.gte("created_at", *filters.CreatedAfter)
	}
	if filters.CreatedBefore != nil {
		c.lte("created_at", *filters.CreatedBefore)
	}
	c.search(filters.Search, "email", "display_name")
	return c
}

// List retrieves users with pagination, applying the given filters.
func (r *userRepository) List(ctx context.Context, filters repository.UserFilters, limit, offset int) ([]*models.User, error) {
	users := []*models.User{}

	c := userFilterConditions(filters)
	where, args := c.clause()

	query := `
		SELECT id, email, password_hash, display_name, organization_role,
			   is_active, oauth_provider, oauth_id,
			   two_factor_enabled, two_factor_secret,
			   last_login_at, created_at, updated_at
		FROM users
	`
	if where != "" {
		query += " WHERE " + where
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	err := r.db.SelectContext(ctx, &users, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}

// Count returns the number of users matching the given filters.
func (r *userRepository) Count(ctx context.Context, filters repository.UserFilters) (int64, error) {
	c := userFilterConditions(filters)
	query := "SELECT COUNT(*) FROM users" + c.whereClause()
	_, args := c.clause()

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// UpdateLastLogin updates the user's last login timestamp
func (r *userRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	now := time.Now()

	query := `
		UPDATE users
		SET last_login_at = $2,
			updated_at = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, now)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
