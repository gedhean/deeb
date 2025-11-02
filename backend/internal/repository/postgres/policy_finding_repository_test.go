package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyFindingRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTestDB(t)
	repo := NewPolicyFindingRepository(db)
	ctx := context.Background()

	// Create a test change plan ID
	changePlanID := uuid.New()

	// Clean up after test
	defer func() {
		_, _ = db.ExecContext(ctx, "DELETE FROM policy_findings WHERE change_plan_id = $1", changePlanID)
	}()

	t.Run("Create policy finding", func(t *testing.T) {
		finding := &models.PolicyFinding{
			ChangePlanID:  changePlanID,
			ValidatorType: "security_risk",
			Severity:      "Error",
			RuleName:      "drop-table",
			Message:       "DROP TABLE operation detected",
		}

		err := repo.Create(ctx, finding)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, finding.ID)
		assert.False(t, finding.CreatedAt.IsZero())
	})

	t.Run("CreateBatch policy findings", func(t *testing.T) {
		findings := []*models.PolicyFinding{
			{
				ChangePlanID:  changePlanID,
				ValidatorType: "security_risk",
				Severity:      "Error",
				RuleName:      "drop-table",
				Message:       "DROP TABLE operation detected",
			},
			{
				ChangePlanID:  changePlanID,
				ValidatorType: "performance_issue",
				Severity:      "Warning",
				RuleName:      "missing-indexes",
				Message:       "Missing indexes detected",
			},
			{
				ChangePlanID:  changePlanID,
				ValidatorType: "best_practice_violation",
				Severity:      "Info",
				RuleName:      "naming-convention",
				Message:       "Table name should use snake_case",
			},
		}

		err := repo.CreateBatch(ctx, findings)
		require.NoError(t, err)

		// Verify all findings were created
		for _, finding := range findings {
			assert.NotEqual(t, uuid.Nil, finding.ID)
		}
	})

	t.Run("GetByChangePlanID retrieves findings", func(t *testing.T) {
		// Create a test finding
		finding := &models.PolicyFinding{
			ChangePlanID:  changePlanID,
			ValidatorType: "security_risk",
			Severity:      "Error",
			RuleName:      "drop-database",
			Message:       "DROP DATABASE operation detected",
		}
		err := repo.Create(ctx, finding)
		require.NoError(t, err)

		// Retrieve findings
		findings, err := repo.GetByChangePlanID(ctx, changePlanID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(findings), 1)

		// Verify finding is in results
		found := false
		for _, f := range findings {
			if f.ID == finding.ID {
				found = true
				assert.Equal(t, changePlanID, f.ChangePlanID)
				assert.Equal(t, "security_risk", f.ValidatorType)
				assert.Equal(t, "Error", f.Severity)
				break
			}
		}
		assert.True(t, found, "Finding should be in results")
	})

	t.Run("GetByChangePlanID returns empty slice for non-existent change plan", func(t *testing.T) {
		nonExistentID := uuid.New()
		findings, err := repo.GetByChangePlanID(ctx, nonExistentID)
		require.NoError(t, err)
		assert.Empty(t, findings)
	})

	t.Run("DeleteByChangePlanID deletes all findings", func(t *testing.T) {
		// Create a test change plan ID
		testChangePlanID := uuid.New()

		// Create findings
		findings := []*models.PolicyFinding{
			{
				ChangePlanID:  testChangePlanID,
				ValidatorType: "security_risk",
				Severity:      "Error",
				RuleName:      "drop-table",
				Message:       "DROP TABLE operation detected",
			},
			{
				ChangePlanID:  testChangePlanID,
				ValidatorType: "performance_issue",
				Severity:      "Warning",
				RuleName:      "missing-indexes",
				Message:       "Missing indexes detected",
			},
		}
		err := repo.CreateBatch(ctx, findings)
		require.NoError(t, err)

		// Verify findings exist
		existingFindings, err := repo.GetByChangePlanID(ctx, testChangePlanID)
		require.NoError(t, err)
		assert.Len(t, existingFindings, 2)

		// Delete findings
		err = repo.DeleteByChangePlanID(ctx, testChangePlanID)
		require.NoError(t, err)

		// Verify findings are deleted
		deletedFindings, err := repo.GetByChangePlanID(ctx, testChangePlanID)
		require.NoError(t, err)
		assert.Empty(t, deletedFindings)
	})

	t.Run("CountBySeverity counts findings correctly", func(t *testing.T) {
		// Create a test change plan ID
		testChangePlanID := uuid.New()

		// Create findings with different severities
		findings := []*models.PolicyFinding{
			{
				ChangePlanID:  testChangePlanID,
				ValidatorType: "security_risk",
				Severity:      "Error",
				RuleName:      "drop-table",
				Message:       "DROP TABLE operation detected",
			},
			{
				ChangePlanID:  testChangePlanID,
				ValidatorType: "security_risk",
				Severity:      "Error",
				RuleName:      "drop-database",
				Message:       "DROP DATABASE operation detected",
			},
			{
				ChangePlanID:  testChangePlanID,
				ValidatorType: "performance_issue",
				Severity:      "Warning",
				RuleName:      "missing-indexes",
				Message:       "Missing indexes detected",
			},
			{
				ChangePlanID:  testChangePlanID,
				ValidatorType: "best_practice_violation",
				Severity:      "Info",
				RuleName:      "naming-convention",
				Message:       "Table name should use snake_case",
			},
		}
		err := repo.CreateBatch(ctx, findings)
		require.NoError(t, err)

		// Clean up
		defer func() {
			_, _ = db.ExecContext(ctx, "DELETE FROM policy_findings WHERE change_plan_id = $1", testChangePlanID)
		}()

		// Count by severity
		counts, err := repo.CountBySeverity(ctx, testChangePlanID)
		require.NoError(t, err)

		assert.Equal(t, 2, counts["Error"])
		assert.Equal(t, 1, counts["Warning"])
		assert.Equal(t, 1, counts["Info"])
	})

	t.Run("CountBySeverity returns zero counts for non-existent change plan", func(t *testing.T) {
		nonExistentID := uuid.New()
		counts, err := repo.CountBySeverity(ctx, nonExistentID)
		require.NoError(t, err)

		assert.Equal(t, 0, counts["Error"])
		assert.Equal(t, 0, counts["Warning"])
		assert.Equal(t, 0, counts["Info"])
	})

	t.Run("Create validates required fields", func(t *testing.T) {
		invalidFinding := &models.PolicyFinding{
			// Missing required fields
		}

		err := repo.Create(ctx, invalidFinding)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("GetByChangePlanID orders by severity", func(t *testing.T) {
		// Create a test change plan ID
		testChangePlanID := uuid.New()

		// Create findings with different severities
		findings := []*models.PolicyFinding{
			{
				ChangePlanID:  testChangePlanID,
				ValidatorType: "best_practice_violation",
				Severity:      "Info",
				RuleName:      "naming-convention",
				Message:       "Table name should use snake_case",
			},
			{
				ChangePlanID:  testChangePlanID,
				ValidatorType: "security_risk",
				Severity:      "Error",
				RuleName:      "drop-table",
				Message:       "DROP TABLE operation detected",
			},
			{
				ChangePlanID:  testChangePlanID,
				ValidatorType: "performance_issue",
				Severity:      "Warning",
				RuleName:      "missing-indexes",
				Message:       "Missing indexes detected",
			},
		}
		err := repo.CreateBatch(ctx, findings)
		require.NoError(t, err)

		// Clean up
		defer func() {
			_, _ = db.ExecContext(ctx, "DELETE FROM policy_findings WHERE change_plan_id = $1", testChangePlanID)
		}()

		// Retrieve findings
		retrievedFindings, err := repo.GetByChangePlanID(ctx, testChangePlanID)
		require.NoError(t, err)
		require.Len(t, retrievedFindings, 3)

		// Verify ordering: Error first, then Warning, then Info
		assert.Equal(t, "Error", retrievedFindings[0].Severity)
		assert.Equal(t, "Warning", retrievedFindings[1].Severity)
		assert.Equal(t, "Info", retrievedFindings[2].Severity)
	})
}
