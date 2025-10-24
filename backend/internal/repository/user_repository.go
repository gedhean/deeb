package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// UserRepository defines the interface for user data operations
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

	// List retrieves all users with pagination
	List(ctx context.Context, limit, offset int) ([]*models.User, error)

	// UpdateLastLogin updates the user's last login timestamp
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}
