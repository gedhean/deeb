package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
)

// IssueRepository implements repository.IssueRepository for PostgreSQL
type IssueRepository struct {
	db *sql.DB
}

// NewIssueRepository creates a new PostgreSQL issue repository
func NewIssueRepository(db *sql.DB) *IssueRepository {
	return &IssueRepository{db: db}
}

// Create creates a new issue in the database
func (r *IssueRepository) Create(ctx context.Context, issue *models.Issue) error {
	// Generate UUID if not provided
	if issue.ID == uuid.Nil {
		issue.ID = uuid.New()
	}

	// Set timestamps
	now := time.Now().UTC()
	issue.CreatedAt = now
	issue.UpdatedAt = now

	// Default status to Draft if not set
	if issue.Status == "" {
		issue.Status = models.IssueStatusDraft
	}

	// Default risk level to Medium if not set
	if issue.RiskLevel == "" {
		issue.RiskLevel = models.RiskLevelMedium
	}

	// Validate after setting defaults
	if err := issue.Validate(); err != nil {
		return err
	}

	query := `
		INSERT INTO issues (
			id, project_id, target_database_id, title, description, change_type,
			sql_content, rollback_sql, status, risk_level, author_id,
			assigned_reviewer_id, scheduled_for, executed_at, execution_duration_ms,
			affected_rows, error_message, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	_, err := r.db.ExecContext(ctx, query,
		issue.ID,
		issue.ProjectID,
		issue.TargetDatabaseID,
		issue.Title,
		issue.Description,
		issue.ChangeType,
		issue.SQLContent,
		issue.RollbackSQL,
		issue.Status,
		issue.RiskLevel,
		issue.AuthorID,
		issue.AssignedReviewerID,
		issue.ScheduledFor,
		issue.ExecutedAt,
		issue.ExecutionDurationMs,
		issue.AffectedRows,
		issue.ErrorMessage,
		issue.CreatedAt,
		issue.UpdatedAt,
	)

	return err
}

// GetByID retrieves an issue by its ID
func (r *IssueRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Issue, error) {
	query := `
		SELECT id, project_id, target_database_id, title, description, change_type,
		       sql_content, rollback_sql, status, risk_level, author_id,
		       assigned_reviewer_id, scheduled_for, executed_at, execution_duration_ms,
		       affected_rows, error_message, created_at, updated_at
		FROM issues
		WHERE id = $1
	`

	issue := &models.Issue{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&issue.ID,
		&issue.ProjectID,
		&issue.TargetDatabaseID,
		&issue.Title,
		&issue.Description,
		&issue.ChangeType,
		&issue.SQLContent,
		&issue.RollbackSQL,
		&issue.Status,
		&issue.RiskLevel,
		&issue.AuthorID,
		&issue.AssignedReviewerID,
		&issue.ScheduledFor,
		&issue.ExecutedAt,
		&issue.ExecutionDurationMs,
		&issue.AffectedRows,
		&issue.ErrorMessage,
		&issue.CreatedAt,
		&issue.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound{Resource: "issue", ID: id.String()}
	}
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// Update updates an existing issue
func (r *IssueRepository) Update(ctx context.Context, issue *models.Issue) error {
	if err := issue.Validate(); err != nil {
		return err
	}

	// Update the updated_at timestamp
	issue.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE issues
		SET title = $2,
		    description = $3,
		    change_type = $4,
		    sql_content = $5,
		    rollback_sql = $6,
		    status = $7,
		    risk_level = $8,
		    target_database_id = $9,
		    assigned_reviewer_id = $10,
		    scheduled_for = $11,
		    updated_at = $12
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		issue.ID,
		issue.Title,
		issue.Description,
		issue.ChangeType,
		issue.SQLContent,
		issue.RollbackSQL,
		issue.Status,
		issue.RiskLevel,
		issue.TargetDatabaseID,
		issue.AssignedReviewerID,
		issue.ScheduledFor,
		issue.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return models.ErrNotFound{Resource: "issue", ID: issue.ID.String()}
	}

	return nil
}

// Delete deletes an issue by its ID
func (r *IssueRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM issues WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return models.ErrNotFound{Resource: "issue", ID: id.String()}
	}

	return nil
}

// List retrieves a list of issues with optional filters
func (r *IssueRepository) List(ctx context.Context, filters repository.ListFilters) ([]*models.Issue, error) {
	query := `
		SELECT id, project_id, target_database_id, title, description, change_type,
		       sql_content, rollback_sql, status, risk_level, author_id,
		       assigned_reviewer_id, scheduled_for, executed_at, execution_duration_ms,
		       affected_rows, error_message, created_at, updated_at
		FROM issues
	`

	whereClause, args := r.buildWhereClause(filters)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	query += r.buildOrderClause(filters)
	query += r.buildPaginationClause(filters)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	return r.scanIssues(rows)
}

// Search searches for issues based on a query string
func (r *IssueRepository) Search(ctx context.Context, query string, filters repository.ListFilters) ([]*models.Issue, error) {
	sqlQuery := `
		SELECT id, project_id, target_database_id, title, description, change_type,
		       sql_content, rollback_sql, status, risk_level, author_id,
		       assigned_reviewer_id, scheduled_for, executed_at, execution_duration_ms,
		       affected_rows, error_message, created_at, updated_at
		FROM issues
		WHERE (
			title ILIKE $1 OR
			description ILIKE $1 OR
			sql_content ILIKE $1
		)
	`

	args := []interface{}{"%" + query + "%"}
	argIndex := 2

	// Add additional filters
	if filters.Status != "" {
		sqlQuery += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filters.Status)
		argIndex++
	}
	if filters.AuthorID != uuid.Nil {
		sqlQuery += fmt.Sprintf(" AND author_id = $%d", argIndex)
		args = append(args, filters.AuthorID)
		argIndex++
	}
	if filters.DatabaseID != uuid.Nil {
		sqlQuery += fmt.Sprintf(" AND target_database_id = $%d", argIndex)
		args = append(args, filters.DatabaseID)
		argIndex++
	}
	if filters.ProjectID != uuid.Nil {
		sqlQuery += fmt.Sprintf(" AND project_id = $%d", argIndex)
		args = append(args, filters.ProjectID)
	}

	sqlQuery += r.buildOrderClause(filters)
	sqlQuery += r.buildPaginationClause(filters)

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	return r.scanIssues(rows)
}

// Count returns the total number of issues matching the filters
func (r *IssueRepository) Count(ctx context.Context, filters repository.ListFilters) (int64, error) {
	query := "SELECT COUNT(*) FROM issues"

	whereClause, args := r.buildWhereClause(filters)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// buildWhereClause builds the WHERE clause and returns arguments
func (r *IssueRepository) buildWhereClause(filters repository.ListFilters) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, filters.Status)
		argIndex++
	}

	if filters.AuthorID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("author_id = $%d", argIndex))
		args = append(args, filters.AuthorID)
		argIndex++
	}

	if filters.DatabaseID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("target_database_id = $%d", argIndex))
		args = append(args, filters.DatabaseID)
		argIndex++
	}

	if filters.ProjectID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("project_id = $%d", argIndex))
		args = append(args, filters.ProjectID)
	}

	return strings.Join(conditions, " AND "), args
}

// buildOrderClause builds the ORDER BY clause
func (r *IssueRepository) buildOrderClause(filters repository.ListFilters) string {
	sortBy := filters.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}

	// Validate sortBy to prevent SQL injection
	validSortFields := map[string]bool{
		"id":         true,
		"title":      true,
		"status":     true,
		"risk_level": true,
		"created_at": true,
		"updated_at": true,
	}

	if !validSortFields[sortBy] {
		sortBy = "created_at"
	}

	sortOrder := strings.ToUpper(filters.SortOrder)
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}

	return fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)
}

// buildPaginationClause builds the LIMIT/OFFSET clause
func (r *IssueRepository) buildPaginationClause(filters repository.ListFilters) string {
	limit := filters.Limit
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if limit > 100 {
		limit = 100 // Max limit
	}

	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	return fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
}

// scanIssues scans multiple rows into Issue structs
func (r *IssueRepository) scanIssues(rows *sql.Rows) ([]*models.Issue, error) {
	var issues []*models.Issue

	for rows.Next() {
		issue := &models.Issue{}
		err := rows.Scan(
			&issue.ID,
			&issue.ProjectID,
			&issue.TargetDatabaseID,
			&issue.Title,
			&issue.Description,
			&issue.ChangeType,
			&issue.SQLContent,
			&issue.RollbackSQL,
			&issue.Status,
			&issue.RiskLevel,
			&issue.AuthorID,
			&issue.AssignedReviewerID,
			&issue.ScheduledFor,
			&issue.ExecutedAt,
			&issue.ExecutionDurationMs,
			&issue.AffectedRows,
			&issue.ErrorMessage,
			&issue.CreatedAt,
			&issue.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		issues = append(issues, issue)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return issues, nil
}
