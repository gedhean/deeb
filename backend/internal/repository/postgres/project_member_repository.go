package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
)

// ProjectMemberRepository is the PostgreSQL implementation of repository.ProjectMemberRepository
type ProjectMemberRepository struct {
	db *sqlx.DB
}

// NewProjectMemberRepository creates a new PostgreSQL project member repository
func NewProjectMemberRepository(db *sqlx.DB) *ProjectMemberRepository {
	return &ProjectMemberRepository{db: db}
}

// AddMember adds a new member to a project
func (r *ProjectMemberRepository) AddMember(ctx context.Context, member *models.ProjectMember) error {
	// Validate member
	if err := member.Validate(); err != nil {
		return err
	}

	// Set timestamps and ID
	now := time.Now()
	if member.ID == uuid.Nil {
		member.ID = uuid.New()
	}
	if member.AddedAt.IsZero() {
		member.AddedAt = now
	}

	query := `
		INSERT INTO project_members (id, project_id, user_id, project_role, added_by_id, added_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(ctx, query,
		member.ID,
		member.ProjectID,
		member.UserID,
		member.Role,
		member.AddedByID,
		member.AddedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

// RemoveMember removes a member from a project
func (r *ProjectMemberRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	query := `
		DELETE FROM project_members
		WHERE project_id = $1 AND user_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, projectID, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// GetMembers retrieves all members of a project
func (r *ProjectMemberRepository) GetMembers(ctx context.Context, projectID uuid.UUID) ([]*models.ProjectMember, error) {
	query := `
		SELECT id, project_id, user_id, project_role, added_by_id, added_at
		FROM project_members
		WHERE project_id = $1
		ORDER BY added_at ASC
	`

	var members []*models.ProjectMember
	err := r.db.SelectContext(ctx, &members, query, projectID)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no members found
	if members == nil {
		members = []*models.ProjectMember{}
	}

	return members, nil
}

// GetMemberRole retrieves the role of a specific user in a project
func (r *ProjectMemberRepository) GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (*models.ProjectMember, error) {
	query := `
		SELECT id, project_id, user_id, project_role, added_by_id, added_at
		FROM project_members
		WHERE project_id = $1 AND user_id = $2
	`

	var member models.ProjectMember
	err := r.db.GetContext(ctx, &member, query, projectID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &member, nil
}

// UpdateRole updates the role of a member in a project
func (r *ProjectMemberRepository) UpdateRole(ctx context.Context, projectID, userID uuid.UUID, role models.ProjectRole) error {
	// Validate role
	if !role.IsValid() {
		return models.ErrProjectMemberInvalidRole
	}

	query := `
		UPDATE project_members
		SET project_role = $3
		WHERE project_id = $1 AND user_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, projectID, userID, role)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}
