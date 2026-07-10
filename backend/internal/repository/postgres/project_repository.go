package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
		INSERT INTO projects (id, name, description, created_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(
		ctx, query,
		project.ID,
		project.Name,
		project.Description,
		project.CreatedByID,
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
		SELECT id, name, description, created_by_id, created_at, updated_at
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

// applyProjectFilters adds the ProjectFilters conditions to c. The prefix
// (e.g. "p.") is applied to columns so it can be reused for the joined member
// queries that alias the projects table.
func applyProjectFilters(c *conditions, filters repository.ProjectFilters, prefix string) {
	if filters.CreatedAfter != nil {
		c.gte(prefix+"created_at", *filters.CreatedAfter)
	}
	if filters.CreatedBefore != nil {
		c.lte(prefix+"created_at", *filters.CreatedBefore)
	}
	c.search(filters.Search, prefix+"name", prefix+"description")
}

// List retrieves all projects with pagination, applying the given filters.
func (r *ProjectRepository) List(
	ctx context.Context,
	filters repository.ProjectFilters,
	limit, offset int,
) ([]*models.Project, error) {
	c := &conditions{}
	applyProjectFilters(c, filters, "")
	where, args := c.clause()

	query := `SELECT id, name, description, created_by_id, created_at, updated_at FROM projects`
	if where != "" {
		query += " WHERE " + where
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	var projects []*models.Project
	err := r.db.SelectContext(ctx, &projects, query, args...)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no projects found
	if projects == nil {
		projects = []*models.Project{}
	}

	return projects, nil
}

// Count returns the number of projects matching the given filters.
func (r *ProjectRepository) Count(ctx context.Context, filters repository.ProjectFilters) (int64, error) {
	c := &conditions{}
	applyProjectFilters(c, filters, "")
	query := "SELECT COUNT(*) FROM projects" + c.whereClause()
	_, args := c.clause()

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
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

	result, err := r.db.ExecContext(
		ctx, query,
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

// ListByUserID retrieves projects where the user is a member with pagination,
// applying the given filters.
func (r *ProjectRepository) ListByUserID(
	ctx context.Context,
	userID uuid.UUID,
	filters repository.ProjectFilters,
	limit, offset int,
) ([]*models.Project, error) {
	c := &conditions{}
	c.equal("pm.user_id", userID)
	applyProjectFilters(c, filters, "p.")
	where, args := c.clause()

	query := fmt.Sprintf(`
		SELECT DISTINCT p.id, p.name, p.description, p.created_by_id, p.created_at, p.updated_at
		FROM projects p
		INNER JOIN project_members pm ON p.id = pm.project_id
		WHERE %s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	var projects []*models.Project
	err := r.db.SelectContext(ctx, &projects, query, args...)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no projects found
	if projects == nil {
		projects = []*models.Project{}
	}

	return projects, nil
}

// CountByUserID returns the total number of projects where the user is a
// member, matching the given filters.
func (r *ProjectRepository) CountByUserID(
	ctx context.Context,
	userID uuid.UUID,
	filters repository.ProjectFilters,
) (int64, error) {
	c := &conditions{}
	c.equal("pm.user_id", userID)
	applyProjectFilters(c, filters, "p.")
	where, args := c.clause()

	query := fmt.Sprintf(`
		SELECT COUNT(DISTINCT p.id)
		FROM projects p
		INNER JOIN project_members pm ON p.id = pm.project_id
		WHERE %s
	`, where)

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
