package models

import (
	"time"

	"github.com/google/uuid"
)

// ApprovalStatus represents the status of an approval
type ApprovalStatus string

const (
	ApprovalStatusApproved ApprovalStatus = "Approved"
	ApprovalStatusRejected ApprovalStatus = "Rejected"
)

// Approval represents an approval/rejection record for a change plan
type Approval struct {
	ID           uuid.UUID      `json:"id"                    db:"id"`
	ChangePlanID uuid.UUID      `json:"change_plan_id"        db:"change_plan_id"`
	ApproverID   uuid.UUID      `json:"approver_id"           db:"approver_id"`
	Status       ApprovalStatus `json:"status"                db:"status"`
	Comments     *string        `json:"comments,omitempty"    db:"comments"`
	ApprovedAt   *time.Time     `json:"approved_at,omitempty" db:"approved_at"`
	CreatedAt    time.Time      `json:"created_at"            db:"created_at"`
}

// CreateApprovalRequest represents the request to create a new approval
type CreateApprovalRequest struct {
	ChangePlanID uuid.UUID      `json:"change_plan_id"     validate:"required"`
	Status       ApprovalStatus `json:"status"             validate:"required,oneof=Approved Rejected"`
	Comments     *string        `json:"comments,omitempty"`
}

// ApprovalResponse represents an approval response
type ApprovalResponse struct {
	ID           uuid.UUID      `json:"id"`
	ChangePlanID uuid.UUID      `json:"change_plan_id"`
	ApproverID   uuid.UUID      `json:"approver_id"`
	Status       ApprovalStatus `json:"status"`
	Comments     *string        `json:"comments,omitempty"`
	ApprovedAt   *time.Time     `json:"approved_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// ToResponse converts an Approval to ApprovalResponse
func (a *Approval) ToResponse() *ApprovalResponse {
	return &ApprovalResponse{
		ID:           a.ID,
		ChangePlanID: a.ChangePlanID,
		ApproverID:   a.ApproverID,
		Status:       a.Status,
		Comments:     a.Comments,
		ApprovedAt:   a.ApprovedAt,
		CreatedAt:    a.CreatedAt,
	}
}
