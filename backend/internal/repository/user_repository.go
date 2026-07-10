// Package repository defines repository interfaces for data access operations.
// Implementations are provided in the postgres subpackage.
package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// UserFilters holds optional filters for listing users. Zero-value fields
// (empty string / nil) are ignored, so the same struct drives both the
// unfiltered list and any combination of filters.
type UserFilters struct {
	Search           string // ILIKE over email, display_name
	OrganizationRole string // "" = all; equals organization_role
	IsActive         *bool  // nil = all; equals is_active
	CreatedAfter     *time.Time
	CreatedBefore    *time.Time
}

// UserRepository defines the interface for user data operations.
type UserRepository interface {
	// Create creates a new user in the database
	Create(ctx context.Context, user *models.User) error

	// GetByID retrieves a user by their ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)

	// GetByEmail retrieves a user by their email address
	GetByEmail(ctx context.Context, email string) (*models.User, error)

	// Update updates an existing user's information
	Update(ctx context.Context, user *models.User) error

	// Delete deletes a user by their ID
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves users with pagination, applying the given filters.
	List(ctx context.Context, filters UserFilters, limit, offset int) ([]*models.User, error)

	// Count returns the number of users matching the given filters.
	Count(ctx context.Context, filters UserFilters) (int64, error)

	// UpdateLastLogin updates the user's last login timestamp
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}
