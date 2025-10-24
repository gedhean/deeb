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

// ProjectRepository is the PostgreSQL implementation of repository.ProjectRepository
type ProjectRepository struct {
	db *sqlx.DB
}

// NewProjectRepository creates a new PostgreSQL project repository
func NewProjectRepository(db *sqlx.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create creates a new project in the database
func (r *ProjectRepository) Create(ctx context.Context, project *models.Project) error {
	// Validate project
	if err := project.Validate(); err != nil {
		return err
	}

	// Set timestamps
	now := time.Now()
	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}
	project.CreatedAt = now
	project.UpdatedAt = now

	query := `
		INSERT INTO projects (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		project.ID,
		project.Name,
		project.Description,
		project.CreatedAt,
		project.UpdatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a project by its ID
func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM projects
		WHERE id = $1
	`

	var project models.Project
	err := r.db.GetContext(ctx, &project, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &project, nil
}

// List retrieves all projects with pagination
func (r *ProjectRepository) List(ctx context.Context, limit, offset int) ([]*models.Project, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM projects
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var projects []*models.Project
	err := r.db.SelectContext(ctx, &projects, query, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no projects found
	if projects == nil {
		projects = []*models.Project{}
	}

	return projects, nil
}

// Update updates an existing project's information
func (r *ProjectRepository) Update(ctx context.Context, project *models.Project) error {
	// Validate project
	if err := project.Validate(); err != nil {
		return err
	}

	// Update timestamp
	project.UpdatedAt = time.Now()

	query := `
		UPDATE projects
		SET name = $2, description = $3, updated_at = $4
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		project.ID,
		project.Name,
		project.Description,
		project.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// Delete deletes a project by its ID
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM projects WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return repository.ErrNotFound
	}

	return nil
}
