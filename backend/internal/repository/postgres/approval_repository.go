package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
)

// approvalRepository implements the ApprovalRepository interface for PostgreSQL
type approvalRepository struct {
	db *sqlx.DB
}

// NewApprovalRepository creates a new PostgreSQL approval repository
func NewApprovalRepository(db *sqlx.DB) repository.ApprovalRepository {
	return &approvalRepository{db: db}
}

// Create creates a new approval record in the database
func (r *approvalRepository) Create(ctx context.Context, approval *models.Approval) error {
	if approval.ID == uuid.Nil {
		approval.ID = uuid.New()
	}

	now := time.Now()
	if approval.CreatedAt.IsZero() {
		approval.CreatedAt = now
	}

	// Set approved_at if status is Approved
	if approval.Status == models.ApprovalStatusApproved && approval.ApprovedAt == nil {
		approval.ApprovedAt = &now
	}

	query := `
		INSERT INTO approvals (
			id, change_plan_id, approver_id, status,
			comments, approved_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		approval.ID,
		approval.ChangePlanID,
		approval.ApproverID,
		approval.Status,
		approval.Comments,
		approval.ApprovedAt,
		approval.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create approval: %w", err)
	}

	return nil
}

// GetByChangePlanID retrieves all approvals for a specific change plan
// Returns approvals ordered by created_at DESC (most recent first)
func (r *approvalRepository) GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.Approval, error) {
	approvals := []*models.Approval{}

	query := `
		SELECT id, change_plan_id, approver_id, status,
		       comments, approved_at, created_at
		FROM approvals
		WHERE change_plan_id = $1
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &approvals, query, changePlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get approvals by change plan id: %w", err)
	}

	return approvals, nil
}

// GetByApproverID retrieves all approvals made by a specific approver
// Returns approvals ordered by created_at DESC (most recent first)
func (r *approvalRepository) GetByApproverID(ctx context.Context, approverID uuid.UUID) ([]*models.Approval, error) {
	approvals := []*models.Approval{}

	query := `
		SELECT id, change_plan_id, approver_id, status,
		       comments, approved_at, created_at
		FROM approvals
		WHERE approver_id = $1
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &approvals, query, approverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get approvals by approver id: %w", err)
	}

	return approvals, nil
}

// GetByChangePlanIDAndApproverID retrieves an approval for a specific change plan and approver
// Returns the approval if found, or ErrNotFound if not found
func (r *approvalRepository) GetByChangePlanIDAndApproverID(
	ctx context.Context,
	changePlanID uuid.UUID,
	approverID uuid.UUID,
) (*models.Approval, error) {
	approval := &models.Approval{}

	query := `
		SELECT id, change_plan_id, approver_id, status,
		       comments, approved_at, created_at
		FROM approvals
		WHERE change_plan_id = $1 AND approver_id = $2
		LIMIT 1
	`

	err := r.db.GetContext(ctx, approval, query, changePlanID, approverID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get approval: %w", err)
	}

	return approval, nil
}

// GetHistory retrieves the approval history for a specific change plan
// Returns approvals ordered by created_at DESC (most recent first)
// This is an alias for GetByChangePlanID but semantically clearer for history queries
func (r *approvalRepository) GetHistory(ctx context.Context, changePlanID uuid.UUID) ([]*models.Approval, error) {
	return r.GetByChangePlanID(ctx, changePlanID)
}

// Update updates an existing approval record
func (r *approvalRepository) Update(ctx context.Context, approval *models.Approval) error {
	query := `
		UPDATE approvals
		SET status = $1,
		    comments = $2,
		    approved_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		approval.Status,
		approval.Comments,
		approval.ApprovedAt,
		approval.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update approval: %w", err)
	}

	return nil
}

// DeleteByChangePlanID deletes all approvals for a specific change plan
func (r *approvalRepository) DeleteByChangePlanID(ctx context.Context, changePlanID uuid.UUID) error {
	query := `DELETE FROM approvals WHERE change_plan_id = $1`

	result, err := r.db.ExecContext(ctx, query, changePlanID)
	if err != nil {
		return fmt.Errorf("failed to delete approvals: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// It's okay if no rows were affected (change plan might not have approvals)
	_ = rowsAffected

	return nil
}
