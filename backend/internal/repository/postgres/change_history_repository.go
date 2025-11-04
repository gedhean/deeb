package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
)

// ChangeHistoryRepository is the PostgreSQL implementation of repository.ChangeHistoryRepository
type ChangeHistoryRepository struct {
	db *sqlx.DB
}

// NewChangeHistoryRepository creates a new PostgreSQL change history repository
func NewChangeHistoryRepository(db *sqlx.DB) *ChangeHistoryRepository {
	return &ChangeHistoryRepository{db: db}
}

// Create creates a new change history entry
func (r *ChangeHistoryRepository) Create(ctx context.Context, changeHistory *models.ChangeHistory) error {
	// Set timestamps
	now := time.Now()
	if changeHistory.ID == uuid.Nil {
		changeHistory.ID = uuid.New()
	}
	if changeHistory.ChangedAt.IsZero() {
		changeHistory.ChangedAt = now
	}

	query := `
		INSERT INTO change_history (
			id, change_plan_id, field_name, old_value, new_value,
			changed_by_id, changed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		changeHistory.ID,
		changeHistory.ChangePlanID,
		changeHistory.FieldName,
		changeHistory.OldValue,
		changeHistory.NewValue,
		changeHistory.ChangedByID,
		changeHistory.ChangedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

// GetByChangePlanID retrieves all change history entries for a specific change plan
func (r *ChangeHistoryRepository) GetByChangePlanID(
	ctx context.Context,
	changePlanID uuid.UUID,
) ([]*models.ChangeHistory, error) {
	query := `
		SELECT id, change_plan_id, field_name, old_value, new_value,
			changed_by_id, changed_at
		FROM change_history
		WHERE change_plan_id = $1
		ORDER BY changed_at DESC
	`

	var changeHistories []*models.ChangeHistory
	err := r.db.SelectContext(ctx, &changeHistories, query, changePlanID)
	if err != nil {
		return nil, err
	}

	return changeHistories, nil
}

// GetByChangePlanIDWithPagination retrieves change history entries for a change plan with pagination
func (r *ChangeHistoryRepository) GetByChangePlanIDWithPagination(
	ctx context.Context,
	changePlanID uuid.UUID,
	limit, offset int,
) ([]*models.ChangeHistory, error) {
	query := `
		SELECT id, change_plan_id, field_name, old_value, new_value,
			changed_by_id, changed_at
		FROM change_history
		WHERE change_plan_id = $1
		ORDER BY changed_at DESC
		LIMIT $2 OFFSET $3
	`

	var changeHistories []*models.ChangeHistory
	err := r.db.SelectContext(ctx, &changeHistories, query, changePlanID, limit, offset)
	if err != nil {
		return nil, err
	}

	return changeHistories, nil
}

// CountByChangePlanID returns the total number of change history entries for a change plan
func (r *ChangeHistoryRepository) CountByChangePlanID(ctx context.Context, changePlanID uuid.UUID) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM change_history
		WHERE change_plan_id = $1
	`

	var count int64
	err := r.db.GetContext(ctx, &count, query, changePlanID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}

	return count, nil
}
