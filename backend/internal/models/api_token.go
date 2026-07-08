package models

import (
	"time"

	"github.com/google/uuid"
)

// APIToken represents a long-lived, revocable API token (service account) used by
// unattended automation to authenticate against the REST API. The token binds to a
// user and inherits that user's organization role through the existing RBAC.
//
// Only the SHA-256 hash of the token is persisted; the plaintext is shown once at
// issue time and never stored. TokenHash is tagged json:"-" so it is never serialized.
type APIToken struct {
	ID          uuid.UUID  `json:"id"                     db:"id"`
	UserID      uuid.UUID  `json:"user_id"                db:"user_id"`
	Name        string     `json:"name"                   db:"name"`
	TokenPrefix string     `json:"token_prefix"           db:"token_prefix"`
	TokenHash   string     `json:"-"                      db:"token_hash"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"   db:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"   db:"revoked_at"`
	CreatedAt   time.Time  `json:"created_at"             db:"created_at"`
}

// IsActive reports whether the token can still be used: it must not be revoked and,
// if it has an expiry, that expiry must be in the future.
func (t *APIToken) IsActive() bool {
	if t.RevokedAt != nil {
		return false
	}
	if t.ExpiresAt != nil && !t.ExpiresAt.After(time.Now()) {
		return false
	}
	return true
}
