package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
)

// ChangePlanRepository is the PostgreSQL implementation of repository.ChangePlanRepository
type ChangePlanRepository struct {
	db *sqlx.DB
}

// NewChangePlanRepository creates a new PostgreSQL change plan repository
func NewChangePlanRepository(db *sqlx.DB) *ChangePlanRepository {
	return &ChangePlanRepository{db: db}
}

// Create creates a new change plan in the database
func (r *ChangePlanRepository) Create(ctx context.Context, changePlan *models.ChangePlan) error {
	// Set timestamps
	now := time.Now()
	if changePlan.ID == uuid.Nil {
		changePlan.ID = uuid.New()
	}
	changePlan.CreatedAt = now
	changePlan.UpdatedAt = now

	// Set default status if not set
	if changePlan.Status == "" {
		changePlan.Status = models.ChangePlanStatusDraft
	}

	// Set default risk level if not set
	if changePlan.RiskLevel == "" {
		changePlan.RiskLevel = models.RiskLevelMedium
	}

	query := `
		INSERT INTO change_plans (
			id, project_id, target_database_id, title, description, change_type,
			sql_content, rollback_sql, status, risk_level, author_id,
			assigned_reviewer_id, scheduled_for, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := r.db.ExecContext(ctx, query,
		changePlan.ID,
		changePlan.ProjectID,
		changePlan.TargetDatabaseID,
		changePlan.Title,
		changePlan.Description,
		changePlan.ChangeType,
		changePlan.SQLContent,
		changePlan.RollbackSQL,
		changePlan.Status,
		changePlan.RiskLevel,
		changePlan.AuthorID,
		changePlan.AssignedReviewerID,
		changePlan.ScheduledFor,
		changePlan.CreatedAt,
		changePlan.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a change plan by its ID
func (r *ChangePlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ChangePlan, error) {
	query := `
		SELECT id, project_id, target_database_id, title, description, change_type,
			sql_content, rollback_sql, status, risk_level, author_id,
			assigned_reviewer_id, scheduled_for, executed_at, execution_duration_ms,
			affected_rows, error_message, created_at, updated_at
		FROM change_plans
		WHERE id = $1
	`

	var changePlan models.ChangePlan
	err := r.db.GetContext(ctx, &changePlan, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &changePlan, nil
}

// Update updates an existing change plan's information
func (r *ChangePlanRepository) Update(ctx context.Context, changePlan *models.ChangePlan) error {
	// Update timestamp
	changePlan.UpdatedAt = time.Now()

	query := `
		UPDATE change_plans
		SET title = $2, description = $3, change_type = $4, sql_content = $5,
			rollback_sql = $6, status = $7, risk_level = $8,
			assigned_reviewer_id = $9, scheduled_for = $10, executed_at = $11,
			execution_duration_ms = $12, affected_rows = $13, error_message = $14,
			updated_at = $15
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		changePlan.ID,
		changePlan.Title,
		changePlan.Description,
		changePlan.ChangeType,
		changePlan.SQLContent,
		changePlan.RollbackSQL,
		changePlan.Status,
		changePlan.RiskLevel,
		changePlan.AssignedReviewerID,
		changePlan.ScheduledFor,
		changePlan.ExecutedAt,
		changePlan.ExecutionDurationMs,
		changePlan.AffectedRows,
		changePlan.ErrorMessage,
		changePlan.UpdatedAt,
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

// Delete deletes a change plan by its ID
func (r *ChangePlanRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM change_plans WHERE id = $1`

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

// List retrieves all change plans with pagination
func (r *ChangePlanRepository) List(ctx context.Context, limit, offset int) ([]*models.ChangePlan, error) {
	query := `
		SELECT id, project_id, target_database_id, title, description, change_type,
			sql_content, rollback_sql, status, risk_level, author_id,
			assigned_reviewer_id, scheduled_for, executed_at, execution_duration_ms,
			affected_rows, error_message, created_at, updated_at
		FROM change_plans
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var changePlans []*models.ChangePlan
	err := r.db.SelectContext(ctx, &changePlans, query, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no change plans found
	if changePlans == nil {
		changePlans = []*models.ChangePlan{}
	}

	return changePlans, nil
}

// Count returns the total number of change plans
func (r *ChangePlanRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM change_plans`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetByStatus retrieves change plans filtered by status
func (r *ChangePlanRepository) GetByStatus(
	ctx context.Context,
	status models.ChangePlanStatus,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	query := `
		SELECT id, project_id, target_database_id, title, description, change_type,
			sql_content, rollback_sql, status, risk_level, author_id,
			assigned_reviewer_id, scheduled_for, executed_at, execution_duration_ms,
			affected_rows, error_message, created_at, updated_at
		FROM change_plans
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var changePlans []*models.ChangePlan
	err := r.db.SelectContext(ctx, &changePlans, query, status, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no change plans found
	if changePlans == nil {
		changePlans = []*models.ChangePlan{}
	}

	return changePlans, nil
}

// GetByProjectID retrieves change plans filtered by project ID
func (r *ChangePlanRepository) GetByProjectID(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	query := `
		SELECT id, project_id, target_database_id, title, description, change_type,
			sql_content, rollback_sql, status, risk_level, author_id,
			assigned_reviewer_id, scheduled_for, executed_at, execution_duration_ms,
			affected_rows, error_message, created_at, updated_at
		FROM change_plans
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var changePlans []*models.ChangePlan
	err := r.db.SelectContext(ctx, &changePlans, query, projectID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no change plans found
	if changePlans == nil {
		changePlans = []*models.ChangePlan{}
	}

	return changePlans, nil
}

// GetByDatabaseID retrieves change plans filtered by database ID
func (r *ChangePlanRepository) GetByDatabaseID(
	ctx context.Context,
	databaseID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	query := `
		SELECT id, project_id, target_database_id, title, description, change_type,
			sql_content, rollback_sql, status, risk_level, author_id,
			assigned_reviewer_id, scheduled_for, executed_at, execution_duration_ms,
			affected_rows, error_message, created_at, updated_at
		FROM change_plans
		WHERE target_database_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var changePlans []*models.ChangePlan
	err := r.db.SelectContext(ctx, &changePlans, query, databaseID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no change plans found
	if changePlans == nil {
		changePlans = []*models.ChangePlan{}
	}

	return changePlans, nil
}

// GetByAuthorID retrieves change plans filtered by author ID
func (r *ChangePlanRepository) GetByAuthorID(
	ctx context.Context,
	authorID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	query := `
		SELECT id, project_id, target_database_id, title, description, change_type,
			sql_content, rollback_sql, status, risk_level, author_id,
			assigned_reviewer_id, scheduled_for, executed_at, execution_duration_ms,
			affected_rows, error_message, created_at, updated_at
		FROM change_plans
		WHERE author_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	var changePlans []*models.ChangePlan
	err := r.db.SelectContext(ctx, &changePlans, query, authorID, limit, offset)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no change plans found
	if changePlans == nil {
		changePlans = []*models.ChangePlan{}
	}

	return changePlans, nil
}

// ListWithFilters retrieves change plans with combined filters
func (r *ChangePlanRepository) ListWithFilters(
	ctx context.Context,
	filters repository.ChangePlanListFilters,
) ([]*models.ChangePlan, error) {
	// Build WHERE clause dynamically
	whereClauses := []string{}
	args := []interface{}{}
	argIndex := 1

	if filters.Status != "" {
		whereClauses = append(whereClauses, "status = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, filters.Status)
		argIndex++
	}

	if filters.ProjectID != uuid.Nil {
		whereClauses = append(whereClauses, "project_id = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, filters.ProjectID)
		argIndex++
	}

	if filters.DatabaseID != uuid.Nil {
		whereClauses = append(whereClauses, "target_database_id = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, filters.DatabaseID)
		argIndex++
	}

	if filters.AuthorID != uuid.Nil {
		whereClauses = append(whereClauses, "author_id = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, filters.AuthorID)
	}

	// Build query
	query := `
		SELECT id, project_id, target_database_id, title, description, change_type,
			sql_content, rollback_sql, status, risk_level, author_id,
			assigned_reviewer_id, scheduled_for, executed_at, execution_duration_ms,
			affected_rows, error_message, created_at, updated_at
		FROM change_plans
	`

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY created_at DESC"

	// Add pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
	}

	var changePlans []*models.ChangePlan
	err := r.db.SelectContext(ctx, &changePlans, query, args...)
	if err != nil {
		return nil, err
	}

	// Return empty slice instead of nil if no change plans found
	if changePlans == nil {
		changePlans = []*models.ChangePlan{}
	}

	return changePlans, nil
}

// CountWithFilters returns the total number of change plans matching filters
func (r *ChangePlanRepository) CountWithFilters(ctx context.Context, filters repository.ChangePlanListFilters) (int64, error) {
	// Build WHERE clause dynamically
	whereClauses := []string{}
	args := []interface{}{}
	argIndex := 1

	if filters.Status != "" {
		whereClauses = append(whereClauses, "status = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, filters.Status)
		argIndex++
	}

	if filters.ProjectID != uuid.Nil {
		whereClauses = append(whereClauses, "project_id = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, filters.ProjectID)
		argIndex++
	}

	if filters.DatabaseID != uuid.Nil {
		whereClauses = append(whereClauses, "target_database_id = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, filters.DatabaseID)
		argIndex++
	}

	if filters.AuthorID != uuid.Nil {
		whereClauses = append(whereClauses, "author_id = $"+fmt.Sprintf("%d", argIndex))
		args = append(args, filters.AuthorID)
	}

	// Build query
	query := `SELECT COUNT(*) FROM change_plans`

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}
