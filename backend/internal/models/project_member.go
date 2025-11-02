package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ProjectRole represents the project-level role of a user
type ProjectRole string

const (
	ProjectRoleOwner     ProjectRole = "ProjectOwner"
	ProjectRoleDeveloper ProjectRole = "ProjectDeveloper"
	ProjectRoleApprover  ProjectRole = "ProjectApprover"
	ProjectRoleViewer    ProjectRole = "ProjectViewer"
)

// IsValid checks if the project role is valid
func (r ProjectRole) IsValid() bool {
	switch r {
	case ProjectRoleOwner, ProjectRoleDeveloper, ProjectRoleApprover, ProjectRoleViewer:
		return true
	default:
		return false
	}
}

// ProjectMember represents a user's membership in a project with a specific role
type ProjectMember struct {
	ID        uuid.UUID  `json:"id"         db:"id"`
	ProjectID uuid.UUID  `json:"project_id" db:"project_id"`
	UserID    uuid.UUID  `json:"user_id"    db:"user_id"`
	Role      ProjectRole `json:"role"       db:"project_role"`
	AddedByID uuid.UUID  `json:"added_by_id" db:"added_by_id"`
	AddedAt   time.Time  `json:"added_at"    db:"added_at"`
}

// Validate validates the project member model
func (pm *ProjectMember) Validate() error {
	if pm.ProjectID == uuid.Nil {
		return ErrProjectMemberProjectIDRequired
	}
	if pm.UserID == uuid.Nil {
		return ErrProjectMemberUserIDRequired
	}
	if !pm.Role.IsValid() {
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

