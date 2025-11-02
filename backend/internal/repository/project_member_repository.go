package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// ProjectMemberRepository defines the interface for project member data operations
type ProjectMemberRepository interface {
	// AddMember adds a new member to a project
	AddMember(ctx context.Context, member *models.ProjectMember) error

	// RemoveMember removes a member from a project
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error

	// GetMembers retrieves all members of a project
	GetMembers(ctx context.Context, projectID uuid.UUID) ([]*models.ProjectMember, error)

	// GetMemberRole retrieves the role of a specific user in a project
	GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (*models.ProjectMember, error)

	// UpdateRole updates the role of a member in a project
	UpdateRole(ctx context.Context, projectID, userID uuid.UUID, role models.ProjectRole) error
}
