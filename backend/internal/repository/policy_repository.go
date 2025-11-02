package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// PolicyRepository defines the interface for policy data operations
type PolicyRepository interface {
	// Create creates a new policy in the database
	Create(ctx context.Context, policy *models.Policy) error

	// GetByID retrieves a policy by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Policy, error)

	// List retrieves all policies with pagination
	List(ctx context.Context, limit, offset int) ([]*models.Policy, error)

	// Count returns the total number of policies
	Count(ctx context.Context) (int64, error)

	// GetByType retrieves all policies of a specific type with pagination
	GetByType(ctx context.Context, policyType models.PolicyType, limit, offset int) ([]*models.Policy, error)

	// CountByType returns the total number of policies of a specific type
	CountByType(ctx context.Context, policyType models.PolicyType) (int64, error)

	// Update updates an existing policy's information
	Update(ctx context.Context, policy *models.Policy) error

	// Delete deletes a policy by its ID
	Delete(ctx context.Context, id uuid.UUID) error
}
