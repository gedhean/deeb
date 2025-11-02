package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
)

// PolicyFindingRepository is the PostgreSQL implementation of repository.PolicyFindingRepository
type PolicyFindingRepository struct {
	db *sqlx.DB
}

// NewPolicyFindingRepository creates a new PostgreSQL policy finding repository
func NewPolicyFindingRepository(db *sqlx.DB) *PolicyFindingRepository {
	return &PolicyFindingRepository{db: db}
}

// Create creates a new policy finding in the database
func (r *PolicyFindingRepository) Create(ctx context.Context, finding *models.PolicyFinding) error {
	// Validate finding
	if err := finding.Validate(); err != nil {
		return err
	}

	// Set timestamps
	now := time.Now()
	if finding.ID == uuid.Nil {
		finding.ID = uuid.New()
	}
	finding.CreatedAt = now

	query := `
		INSERT INTO policy_findings (
			id, change_plan_id, validator_type, severity, rule_name,
			message, line_number, column_number, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		finding.ID,
		finding.ChangePlanID,
		finding.ValidatorType,
		finding.Severity,
		finding.RuleName,
		finding.Message,
		finding.LineNumber,
		finding.ColumnNumber,
		finding.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create policy finding: %w", err)
	}

	return nil
}

// CreateBatch creates multiple policy findings in a single transaction
func (r *PolicyFindingRepository) CreateBatch(ctx context.Context, findings []*models.PolicyFinding) error {
	if len(findings) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	query := `
		INSERT INTO policy_findings (
			id, change_plan_id, validator_type, severity, rule_name,
			message, line_number, column_number, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	stmt, err := tx.PreparexContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, finding := range findings {
		// Validate finding
		if err := finding.Validate(); err != nil {
			return fmt.Errorf("invalid finding: %w", err)
		}

		// Set timestamps
		if finding.ID == uuid.Nil {
			finding.ID = uuid.New()
		}
		finding.CreatedAt = now

		_, err := stmt.ExecContext(ctx,
			finding.ID,
			finding.ChangePlanID,
			finding.ValidatorType,
			finding.Severity,
			finding.RuleName,
			finding.Message,
			finding.LineNumber,
			finding.ColumnNumber,
			finding.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert finding: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByChangePlanID retrieves all policy findings for a specific change plan
func (r *PolicyFindingRepository) GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.PolicyFinding, error) {
	query := `
		SELECT id, change_plan_id, validator_type, severity, rule_name,
		       message, line_number, column_number, created_at
		FROM policy_findings
		WHERE change_plan_id = $1
		ORDER BY 
			CASE severity
				WHEN 'Error' THEN 1
				WHEN 'Warning' THEN 2
				WHEN 'Info' THEN 3
				ELSE 4
			END,
			created_at ASC
	`

	var findings []*models.PolicyFinding
	err := r.db.SelectContext(ctx, &findings, query, changePlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy findings: %w", err)
	}

	// Return empty slice instead of nil if no findings found
	if findings == nil {
		findings = []*models.PolicyFinding{}
	}

	return findings, nil
}

// DeleteByChangePlanID deletes all policy findings for a specific change plan
func (r *PolicyFindingRepository) DeleteByChangePlanID(ctx context.Context, changePlanID uuid.UUID) error {
	query := `DELETE FROM policy_findings WHERE change_plan_id = $1`

	result, err := r.db.ExecContext(ctx, query, changePlanID)
	if err != nil {
		return fmt.Errorf("failed to delete policy findings: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// It's okay if no rows were affected (change plan might not have findings)
	_ = rowsAffected

	return nil
}

// CountBySeverity counts policy findings by severity for a specific change plan
func (r *PolicyFindingRepository) CountBySeverity(ctx context.Context, changePlanID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT severity, COUNT(*) as count
		FROM policy_findings
		WHERE change_plan_id = $1
		GROUP BY severity
	`

	rows, err := r.db.QueryContext(ctx, query, changePlanID)
	if err != nil {
		return nil, fmt.Errorf("failed to count policy findings: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	counts := make(map[string]int)
	for rows.Next() {
		var severity string
		var count int

		if err := rows.Scan(&severity, &count); err != nil {
			return nil, fmt.Errorf("failed to scan count result: %w", err)
		}

		counts[severity] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Ensure all severity levels are present (even if count is 0)
	if _, ok := counts["Error"]; !ok {
		counts["Error"] = 0
	}
	if _, ok := counts["Warning"]; !ok {
		counts["Warning"] = 0
	}
	if _, ok := counts["Info"]; !ok {
		counts["Info"] = 0
	}

	return counts, nil
}
