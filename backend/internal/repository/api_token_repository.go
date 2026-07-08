package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// APITokenRepository defines the interface for API token data operations.
type APITokenRepository interface {
	// Create persists a new API token.
	Create(ctx context.Context, token *models.APIToken) error

	// GetByTokenHash retrieves an active (non-revoked) token by its SHA-256 hash.
	// Returns ErrNotFound if no matching active token exists.
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.APIToken, error)

	// List retrieves all API tokens ordered by creation time (newest first).
	List(ctx context.Context) ([]*models.APIToken, error)

	// Revoke marks a token as revoked by setting revoked_at.
	Revoke(ctx context.Context, id uuid.UUID) error

	// UpdateLastUsed updates the token's last_used_at timestamp.
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}
