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

// apiTokenRepository implements the APITokenRepository interface for PostgreSQL.
type apiTokenRepository struct {
	db *sqlx.DB
}

// NewAPITokenRepository creates a new PostgreSQL API token repository.
func NewAPITokenRepository(db *sqlx.DB) repository.APITokenRepository {
	return &apiTokenRepository{db: db}
}

// Create persists a new API token.
func (r *apiTokenRepository) Create(ctx context.Context, token *models.APIToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO api_tokens (
			id, user_id, name, token_prefix, token_hash,
			last_used_at, expires_at, revoked_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		token.ID,
		token.UserID,
		token.Name,
		token.TokenPrefix,
		token.TokenHash,
		token.LastUsedAt,
		token.ExpiresAt,
		token.RevokedAt,
		token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create api token: %w", err)
	}

	return nil
}

// GetByTokenHash retrieves an active (non-revoked) token by its SHA-256 hash.
// Expiry is checked by the caller via models.APIToken.IsActive.
func (r *apiTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.APIToken, error) {
	var token models.APIToken

	query := `
		SELECT id, user_id, name, token_prefix, token_hash,
			   last_used_at, expires_at, revoked_at, created_at
		FROM api_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL
	`

	err := r.db.GetContext(ctx, &token, query, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get api token: %w", err)
	}

	return &token, nil
}

// List retrieves all API tokens ordered by creation time (newest first).
func (r *apiTokenRepository) List(ctx context.Context) ([]*models.APIToken, error) {
	tokens := []*models.APIToken{}

	query := `
		SELECT id, user_id, name, token_prefix, token_hash,
			   last_used_at, expires_at, revoked_at, created_at
		FROM api_tokens
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &tokens, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list api tokens: %w", err)
	}

	return tokens, nil
}

// Revoke marks a token as revoked by setting revoked_at to the current time.
func (r *apiTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_tokens SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to revoke api token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// UpdateLastUsed updates the token's last_used_at timestamp.
func (r *apiTokenRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_tokens SET last_used_at = $2 WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update api token last_used_at: %w", err)
	}

	return nil
}
