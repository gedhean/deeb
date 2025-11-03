package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupApprovalMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestApprovalRepository_Create(t *testing.T) {
	db, mock := setupApprovalMockDB(t)
	defer db.Close()

	repo := NewApprovalRepository(db)
	ctx := context.Background()

	t.Run("successful creation with auto-generated ID", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		approval := &models.Approval{
			ChangePlanID: changePlanID,
			ApproverID:   approverID,
			Status:       models.ApprovalStatusApproved,
			Comments:     stringPtr("Looks good!"),
		}

		mock.ExpectExec("INSERT INTO approvals").
			WithArgs(
				sqlmock.AnyArg(), // id (UUID)
				changePlanID,
				approverID,
				models.ApprovalStatusApproved,
				approval.Comments,
				sqlmock.AnyArg(), // approved_at
				sqlmock.AnyArg(), // created_at
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, approval)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, approval.ID)
		assert.NotZero(t, approval.CreatedAt)
		assert.NotNil(t, approval.ApprovedAt) // Should be set when status is Approved

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with existing ID", func(t *testing.T) {
		approvalID := uuid.New()
		changePlanID := uuid.New()
		approverID := uuid.New()
		now := time.Now()
		approval := &models.Approval{
			ID:           approvalID,
			ChangePlanID: changePlanID,
			ApproverID:   approverID,
			Status:       models.ApprovalStatusRejected,
			Comments:     stringPtr("Needs more work"),
			CreatedAt:    now,
		}

		mock.ExpectExec("INSERT INTO approvals").
			WithArgs(
				approvalID,
				changePlanID,
				approverID,
				models.ApprovalStatusRejected,
				approval.Comments,
				sql.NullTime{}, // approved_at (nil for Rejected)
				now,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, approval)
		require.NoError(t, err)
		assert.Equal(t, approvalID, approval.ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with nil comments", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		approval := &models.Approval{
			ChangePlanID: changePlanID,
			ApproverID:   approverID,
			Status:       models.ApprovalStatusRejected, // Use Rejected instead of Pending
		}

		mock.ExpectExec("INSERT INTO approvals").
			WithArgs(
				sqlmock.AnyArg(), // id
				changePlanID,
				approverID,
				models.ApprovalStatusRejected,
				nil,              // comments
				sql.NullTime{},   // approved_at (nil for Rejected)
				sqlmock.AnyArg(), // created_at
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, approval)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		approval := &models.Approval{
			ChangePlanID: changePlanID,
			ApproverID:   approverID,
			Status:       models.ApprovalStatusApproved,
		}

		mock.ExpectExec("INSERT INTO approvals").
			WithArgs(
				sqlmock.AnyArg(),
				changePlanID,
				approverID,
				models.ApprovalStatusApproved,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).
			WillReturnError(errors.New("database error"))

		err := repo.Create(ctx, approval)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create approval")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestApprovalRepository_GetByChangePlanID(t *testing.T) {
	db, mock := setupApprovalMockDB(t)
	defer db.Close()

	repo := NewApprovalRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval with multiple approvals", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID1 := uuid.New()
		approverID2 := uuid.New()
		approvalID1 := uuid.New()
		approvalID2 := uuid.New()
		now := time.Now()
		approvedAt := now.Add(-1 * time.Hour)

		rows := sqlmock.NewRows([]string{
			"id", "change_plan_id", "approver_id", "status",
			"comments", "approved_at", "created_at",
		}).
			AddRow(
				approvalID1, changePlanID, approverID1, models.ApprovalStatusApproved,
				"Approved", approvedAt, now,
			).
			AddRow(
				approvalID2, changePlanID, approverID2, models.ApprovalStatusRejected,
				"Rejected", nil, now.Add(-2*time.Hour),
			)

		mock.ExpectQuery("SELECT id, change_plan_id, approver_id, status").
			WithArgs(changePlanID).
			WillReturnRows(rows)

		approvals, err := repo.GetByChangePlanID(ctx, changePlanID)
		require.NoError(t, err)
		require.Len(t, approvals, 2)

		assert.Equal(t, approvalID1, approvals[0].ID)
		assert.Equal(t, models.ApprovalStatusApproved, approvals[0].Status)
		assert.NotNil(t, approvals[0].ApprovedAt)

		assert.Equal(t, approvalID2, approvals[1].ID)
		assert.Equal(t, models.ApprovalStatusRejected, approvals[1].Status)
		assert.Nil(t, approvals[1].ApprovedAt)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful retrieval with no approvals", func(t *testing.T) {
		changePlanID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "change_plan_id", "approver_id", "status",
			"comments", "approved_at", "created_at",
		})

		mock.ExpectQuery("SELECT id, change_plan_id, approver_id, status").
			WithArgs(changePlanID).
			WillReturnRows(rows)

		approvals, err := repo.GetByChangePlanID(ctx, changePlanID)
		require.NoError(t, err)
		assert.Empty(t, approvals)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		changePlanID := uuid.New()

		mock.ExpectQuery("SELECT id, change_plan_id, approver_id, status").
			WithArgs(changePlanID).
			WillReturnError(errors.New("database error"))

		approvals, err := repo.GetByChangePlanID(ctx, changePlanID)
		require.Error(t, err)
		assert.Nil(t, approvals)
		assert.Contains(t, err.Error(), "failed to get approvals by change plan id")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestApprovalRepository_GetByApproverID(t *testing.T) {
	db, mock := setupApprovalMockDB(t)
	defer db.Close()

	repo := NewApprovalRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval with multiple approvals", func(t *testing.T) {
		approverID := uuid.New()
		changePlanID1 := uuid.New()
		changePlanID2 := uuid.New()
		approvalID1 := uuid.New()
		approvalID2 := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "change_plan_id", "approver_id", "status",
			"comments", "approved_at", "created_at",
		}).
			AddRow(
				approvalID1, changePlanID1, approverID, models.ApprovalStatusApproved,
				"Approved", now, now,
			).
			AddRow(
				approvalID2, changePlanID2, approverID, models.ApprovalStatusRejected,
				nil, nil, now.Add(-1*time.Hour),
			)

		mock.ExpectQuery("SELECT id, change_plan_id, approver_id, status").
			WithArgs(approverID).
			WillReturnRows(rows)

		approvals, err := repo.GetByApproverID(ctx, approverID)
		require.NoError(t, err)
		require.Len(t, approvals, 2)

		assert.Equal(t, approvalID1, approvals[0].ID)
		assert.Equal(t, changePlanID1, approvals[0].ChangePlanID)
		assert.Equal(t, models.ApprovalStatusApproved, approvals[0].Status)

		assert.Equal(t, approvalID2, approvals[1].ID)
		assert.Equal(t, changePlanID2, approvals[1].ChangePlanID)
		assert.Equal(t, models.ApprovalStatusRejected, approvals[1].Status)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful retrieval with no approvals", func(t *testing.T) {
		approverID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "change_plan_id", "approver_id", "status",
			"comments", "approved_at", "created_at",
		})

		mock.ExpectQuery("SELECT id, change_plan_id, approver_id, status").
			WithArgs(approverID).
			WillReturnRows(rows)

		approvals, err := repo.GetByApproverID(ctx, approverID)
		require.NoError(t, err)
		assert.Empty(t, approvals)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		approverID := uuid.New()

		mock.ExpectQuery("SELECT id, change_plan_id, approver_id, status").
			WithArgs(approverID).
			WillReturnError(errors.New("database error"))

		approvals, err := repo.GetByApproverID(ctx, approverID)
		require.Error(t, err)
		assert.Nil(t, approvals)
		assert.Contains(t, err.Error(), "failed to get approvals by approver id")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestApprovalRepository_GetHistory(t *testing.T) {
	db, mock := setupApprovalMockDB(t)
	defer db.Close()

	repo := NewApprovalRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		approvalID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "change_plan_id", "approver_id", "status",
			"comments", "approved_at", "created_at",
		}).
			AddRow(
				approvalID, changePlanID, approverID, models.ApprovalStatusApproved,
				"Approved", now, now,
			)

		mock.ExpectQuery("SELECT id, change_plan_id, approver_id, status").
			WithArgs(changePlanID).
			WillReturnRows(rows)

		approvals, err := repo.GetHistory(ctx, changePlanID)
		require.NoError(t, err)
		require.Len(t, approvals, 1)
		assert.Equal(t, approvalID, approvals[0].ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetHistory delegates to GetByChangePlanID", func(t *testing.T) {
		changePlanID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "change_plan_id", "approver_id", "status",
			"comments", "approved_at", "created_at",
		})

		mock.ExpectQuery("SELECT id, change_plan_id, approver_id, status").
			WithArgs(changePlanID).
			WillReturnRows(rows)

		approvals, err := repo.GetHistory(ctx, changePlanID)
		require.NoError(t, err)
		assert.Empty(t, approvals)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
