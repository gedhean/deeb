package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// ApprovalRepository defines the interface for approval data operations
type ApprovalRepository interface {
	// Create creates a new approval record in the database
	Create(ctx context.Context, approval *models.Approval) error

	// GetByChangePlanID retrieves all approvals for a specific change plan
	// Returns approvals ordered by created_at DESC (most recent first)
	GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.Approval, error)

	// GetByApproverID retrieves all approvals made by a specific approver
	// Returns approvals ordered by created_at DESC (most recent first)
	GetByApproverID(ctx context.Context, approverID uuid.UUID) ([]*models.Approval, error)

	// GetByChangePlanIDAndApproverID retrieves an approval for a specific change plan and approver
	// Returns the approval if found, or ErrNotFound if not found
	GetByChangePlanIDAndApproverID(ctx context.Context, changePlanID uuid.UUID, approverID uuid.UUID) (*models.Approval, error)

	// GetHistory retrieves the approval history for a specific change plan
	// Returns approvals ordered by created_at DESC (most recent first)
	// This is an alias for GetByChangePlanID but semantically clearer for history queries
	GetHistory(ctx context.Context, changePlanID uuid.UUID) ([]*models.Approval, error)

	// Update updates an existing approval record
	Update(ctx context.Context, approval *models.Approval) error

	// DeleteByChangePlanID deletes all approvals for a specific change plan
	DeleteByChangePlanID(ctx context.Context, changePlanID uuid.UUID) error
}
