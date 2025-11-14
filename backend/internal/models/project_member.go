package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/rbac"
)

// ProjectMember represents a user's membership in a project with a specific role
type ProjectMember struct {
	ID        uuid.UUID        `json:"id"          db:"id"`
	ProjectID uuid.UUID        `json:"project_id"  db:"project_id"`
	UserID    uuid.UUID        `json:"user_id"     db:"user_id"`
	Role      rbac.ProjectRole `json:"role"        db:"project_role"`
	AddedByID uuid.UUID        `json:"added_by_id" db:"added_by_id"`
	AddedAt   time.Time        `json:"added_at"    db:"added_at"`
}

// AddProjectMemberRequest represents the request to add a member to a project
type AddProjectMemberRequest struct {
	UserID uuid.UUID        `json:"user_id"      validate:"required"`
	Role   rbac.ProjectRole `json:"project_role" validate:"required,oneof=ProjectOwner ProjectDeveloper ProjectApprover ProjectViewer"`
}

// ValidateAddProjectMemberRequest validates an add project member request
func ValidateAddProjectMemberRequest(req *AddProjectMemberRequest) error {
	if req.UserID == uuid.Nil {
		return ErrProjectMemberUserIDRequired
	}
	if !rbac.IsValidProjectRole(req.Role) {
		return ErrProjectMemberInvalidRole
	}
	return nil
}

// ProjectMemberResponse represents a project member response
type ProjectMemberResponse struct {
	ID        uuid.UUID        `json:"id"`
	ProjectID uuid.UUID        `json:"project_id"`
	UserID    uuid.UUID        `json:"user_id"`
	Role      rbac.ProjectRole `json:"role"`
	AddedByID uuid.UUID        `json:"added_by_id"`
	AddedAt   time.Time        `json:"added_at"`
}

// ToResponse converts a ProjectMember to ProjectMemberResponse
func (pm *ProjectMember) ToResponse() *ProjectMemberResponse {
	return &ProjectMemberResponse{
		ID:        pm.ID,
		ProjectID: pm.ProjectID,
		UserID:    pm.UserID,
		Role:      pm.Role,
		AddedByID: pm.AddedByID,
		AddedAt:   pm.AddedAt,
	}
}

// Validate validates the project member model
func (pm *ProjectMember) Validate() error {
	if pm.ProjectID == uuid.Nil {
		return ErrProjectMemberProjectIDRequired
	}
	if pm.UserID == uuid.Nil {
		return ErrProjectMemberUserIDRequired
	}
	if !rbac.IsValidProjectRole(pm.Role) {
		return ErrProjectMemberInvalidRole
	}
	if pm.AddedByID == uuid.Nil {
		return ErrProjectMemberAddedByIDRequired
	}
	return nil
}

// Validation errors
var (
	ErrProjectMemberProjectIDRequired = errors.New("project ID is required")
	ErrProjectMemberUserIDRequired    = errors.New("user ID is required")
	ErrProjectMemberInvalidRole       = errors.New("invalid project role")
	ErrProjectMemberAddedByIDRequired = errors.New("added_by_id is required")
)
