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

	// GetHistory retrieves the approval history for a specific change plan
	// Returns approvals ordered by created_at DESC (most recent first)
	// This is an alias for GetByChangePlanID but semantically clearer for history queries
	GetHistory(ctx context.Context, changePlanID uuid.UUID) ([]*models.Approval, error)
}
