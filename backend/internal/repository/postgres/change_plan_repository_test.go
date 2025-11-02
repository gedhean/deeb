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
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupChangePlanMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestChangePlanRepository_Create(t *testing.T) {
	db, mock := setupChangePlanMockDB(t)
	defer db.Close()

	repo := NewChangePlanRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		changePlan := &models.ChangePlan{
			ProjectID:        uuid.New(),
			TargetDatabaseID: uuid.New(),
			Title:            "Test Change Plan",
			Description:      "Test Description",
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:        "CREATE TABLE test (id UUID);",
			AuthorID:          uuid.New(),
		}

		mock.ExpectExec("INSERT INTO change_plans").
			WithArgs(
				sqlmock.AnyArg(),
				changePlan.ProjectID,
				changePlan.TargetDatabaseID,
				changePlan.Title,
				changePlan.Description,
				changePlan.ChangeType,
				changePlan.SQLContent,
				nil,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				changePlan.AuthorID,
				nil,
				nil,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, changePlan)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, changePlan.ID)
		assert.Equal(t, models.ChangePlanStatusDraft, changePlan.Status)
		assert.Equal(t, models.RiskLevelMedium, changePlan.RiskLevel)
		assert.False(t, changePlan.CreatedAt.IsZero())
		assert.False(t, changePlan.UpdatedAt.IsZero())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with existing ID", func(t *testing.T) {
		existingID := uuid.New()
		changePlan := &models.ChangePlan{
			ID:               existingID,
			ProjectID:        uuid.New(),
			TargetDatabaseID: uuid.New(),
			Title:            "Test Change Plan",
			Description:      "Test Description",
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:        "CREATE TABLE test (id UUID);",
			AuthorID:          uuid.New(),
		}

		mock.ExpectExec("INSERT INTO change_plans").
			WithArgs(
				existingID,
				changePlan.ProjectID,
				changePlan.TargetDatabaseID,
				changePlan.Title,
				changePlan.Description,
				changePlan.ChangeType,
				changePlan.SQLContent,
				nil,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				changePlan.AuthorID,
				nil,
				nil,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, changePlan)
		require.NoError(t, err)
		assert.Equal(t, existingID, changePlan.ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with all fields", func(t *testing.T) {
		rollbackSQL := "DROP TABLE test;"
		scheduledFor := time.Now().Add(24 * time.Hour)
		reviewerID := uuid.New()
		changePlan := &models.ChangePlan{
			ProjectID:         uuid.New(),
			TargetDatabaseID:  uuid.New(),
			Title:             "Test Change Plan",
			Description:       "Test Description",
			ChangeType:        models.ChangeTypeDDL,
			SQLContent:         "CREATE TABLE test (id UUID);",
			RollbackSQL:       &rollbackSQL,
			Status:             models.ChangePlanStatusOpen,
			RiskLevel:          models.RiskLevelHigh,
			AuthorID:           uuid.New(),
			AssignedReviewerID: &reviewerID,
			ScheduledFor:       &scheduledFor,
		}

		mock.ExpectExec("INSERT INTO change_plans").
			WithArgs(
				sqlmock.AnyArg(),
				changePlan.ProjectID,
				changePlan.TargetDatabaseID,
				changePlan.Title,
				changePlan.Description,
				changePlan.ChangeType,
				changePlan.SQLContent,
				rollbackSQL,
				changePlan.Status,
				changePlan.RiskLevel,
				changePlan.AuthorID,
				reviewerID,
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
				sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, changePlan)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		changePlan := &models.ChangePlan{
			ProjectID:        uuid.New(),
			TargetDatabaseID: uuid.New(),
			Title:            "Test Change Plan",
			Description:      "Test Description",
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:        "CREATE TABLE test (id UUID);",
			AuthorID:          uuid.New(),
		}

		mock.ExpectExec("INSERT INTO change_plans").
			WillReturnError(errors.New("database connection error"))

		err := repo.Create(ctx, changePlan)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChangePlanRepository_GetByID(t *testing.T) {
	db, mock := setupChangePlanMockDB(t)
	defer db.Close()

	repo := NewChangePlanRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		changePlanID := uuid.New()
		projectID := uuid.New()
		databaseID := uuid.New()
		authorID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "target_database_id", "title", "description",
			"change_type", "sql_content", "rollback_sql", "status", "risk_level",
			"author_id", "assigned_reviewer_id", "scheduled_for", "executed_at",
			"execution_duration_ms", "affected_rows", "error_message", "created_at", "updated_at",
		}).AddRow(
			changePlanID, projectID, databaseID, "Test Change Plan", "Test Description",
			"DDL", "CREATE TABLE test (id UUID);", nil, "Draft", "Medium",
			authorID, nil, nil, nil, nil, nil, nil,
			time.Now(), time.Now(),
		)

		mock.ExpectQuery("SELECT.*FROM change_plans").
			WithArgs(changePlanID).
			WillReturnRows(rows)

		changePlan, err := repo.GetByID(ctx, changePlanID)
		require.NoError(t, err)
		assert.Equal(t, changePlanID, changePlan.ID)
		assert.Equal(t, projectID, changePlan.ProjectID)
		assert.Equal(t, databaseID, changePlan.TargetDatabaseID)
		assert.Equal(t, "Test Change Plan", changePlan.Title)
		assert.Equal(t, models.ChangePlanStatusDraft, changePlan.Status)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		changePlanID := uuid.New()

		mock.ExpectQuery("SELECT.*FROM change_plans").
			WithArgs(changePlanID).
			WillReturnError(sql.ErrNoRows)

		changePlan, err := repo.GetByID(ctx, changePlanID)
		assert.Nil(t, changePlan)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		changePlanID := uuid.New()

		mock.ExpectQuery("SELECT.*FROM change_plans").
			WithArgs(changePlanID).
			WillReturnError(errors.New("database error"))

		changePlan, err := repo.GetByID(ctx, changePlanID)
		assert.Nil(t, changePlan)
		assert.Error(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChangePlanRepository_Update(t *testing.T) {
	db, mock := setupChangePlanMockDB(t)
	defer db.Close()

	repo := NewChangePlanRepository(db)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		changePlan := &models.ChangePlan{
			ID:               uuid.New(),
			ProjectID:        uuid.New(),
			TargetDatabaseID: uuid.New(),
			Title:            "Updated Title",
			Description:      "Updated Description",
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:        "CREATE TABLE updated (id UUID);",
			Status:            models.ChangePlanStatusOpen,
			RiskLevel:         models.RiskLevelHigh,
			AuthorID:          uuid.New(),
		}

		mock.ExpectExec("UPDATE change_plans").
			WithArgs(
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
				sqlmock.AnyArg(), // updated_at
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Update(ctx, changePlan)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		changePlan := &models.ChangePlan{
			ID:               uuid.New(),
			ProjectID:        uuid.New(),
			TargetDatabaseID: uuid.New(),
			Title:            "Test",
			Description:      "Test",
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:        "CREATE TABLE test;",
			AuthorID:          uuid.New(),
		}

		mock.ExpectExec("UPDATE change_plans").
			WithArgs(
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
				sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(ctx, changePlan)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		changePlan := &models.ChangePlan{
			ID:               uuid.New(),
			ProjectID:        uuid.New(),
			TargetDatabaseID: uuid.New(),
			Title:            "Test",
			Description:      "Test",
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:        "CREATE TABLE test;",
			AuthorID:          uuid.New(),
		}

		mock.ExpectExec("UPDATE change_plans").
			WithArgs(
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
				sqlmock.AnyArg(),
			).
			WillReturnError(errors.New("database error"))

		err := repo.Update(ctx, changePlan)
		assert.Error(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChangePlanRepository_Delete(t *testing.T) {
	db, mock := setupChangePlanMockDB(t)
	defer db.Close()

	repo := NewChangePlanRepository(db)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		changePlanID := uuid.New()

		mock.ExpectExec("DELETE FROM change_plans").
			WithArgs(changePlanID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Delete(ctx, changePlanID)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		changePlanID := uuid.New()

		mock.ExpectExec("DELETE FROM change_plans").
			WithArgs(changePlanID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(ctx, changePlanID)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		changePlanID := uuid.New()

		mock.ExpectExec("DELETE FROM change_plans").
			WithArgs(changePlanID).
			WillReturnError(errors.New("database error"))

		err := repo.Delete(ctx, changePlanID)
		assert.Error(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChangePlanRepository_ListWithFilters(t *testing.T) {
	db, mock := setupChangePlanMockDB(t)
	defer db.Close()

	repo := NewChangePlanRepository(db)
	ctx := context.Background()

	t.Run("list all change plans", func(t *testing.T) {
		changePlanID1 := uuid.New()
		changePlanID2 := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "target_database_id", "title", "description",
			"change_type", "sql_content", "rollback_sql", "status", "risk_level",
			"author_id", "assigned_reviewer_id", "scheduled_for", "executed_at",
			"execution_duration_ms", "affected_rows", "error_message", "created_at", "updated_at",
		}).
			AddRow(
				changePlanID1, uuid.New(), uuid.New(), "Plan 1", "Desc 1",
				"DDL", "CREATE TABLE t1;", nil, "Draft", "Low",
				uuid.New(), nil, nil, nil, nil, nil, nil,
				time.Now(), time.Now(),
			).
			AddRow(
				changePlanID2, uuid.New(), uuid.New(), "Plan 2", "Desc 2",
				"DML", "INSERT INTO t2;", nil, "Open", "Medium",
				uuid.New(), nil, nil, nil, nil, nil, nil,
				time.Now(), time.Now(),
			)

		mock.ExpectQuery("SELECT.*FROM change_plans").
			WillReturnRows(rows)

		filters := repository.ChangePlanListFilters{
			Limit:  10,
			Offset: 0,
		}

		changePlans, err := repo.ListWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, 2, len(changePlans))

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("list with status filter", func(t *testing.T) {
		changePlanID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "target_database_id", "title", "description",
			"change_type", "sql_content", "rollback_sql", "status", "risk_level",
			"author_id", "assigned_reviewer_id", "scheduled_for", "executed_at",
			"execution_duration_ms", "affected_rows", "error_message", "created_at", "updated_at",
		}).
			AddRow(
				changePlanID, uuid.New(), uuid.New(), "Plan", "Desc",
				"DDL", "CREATE TABLE t;", nil, "Open", "Medium",
				uuid.New(), nil, nil, nil, nil, nil, nil,
				time.Now(), time.Now(),
			)

		mock.ExpectQuery("SELECT.*FROM change_plans.*WHERE.*status = \\$1.*LIMIT \\$2").
			WithArgs("Open", 10).
			WillReturnRows(rows)

		filters := repository.ChangePlanListFilters{
			Status: models.ChangePlanStatusOpen,
			Limit:  10,
			Offset: 0,
		}

		changePlans, err := repo.ListWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, 1, len(changePlans))
		assert.Equal(t, models.ChangePlanStatusOpen, changePlans[0].Status)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("list with project_id filter", func(t *testing.T) {
		projectID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "target_database_id", "title", "description",
			"change_type", "sql_content", "rollback_sql", "status", "risk_level",
			"author_id", "assigned_reviewer_id", "scheduled_for", "executed_at",
			"execution_duration_ms", "affected_rows", "error_message", "created_at", "updated_at",
		})

		mock.ExpectQuery("SELECT.*FROM change_plans.*WHERE.*project_id = \\$1.*LIMIT \\$2").
			WithArgs(projectID, 10).
			WillReturnRows(rows)

		filters := repository.ChangePlanListFilters{
			ProjectID: projectID,
			Limit:     10,
			Offset:    0,
		}

		changePlans, err := repo.ListWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, 0, len(changePlans))

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("list with combined filters", func(t *testing.T) {
		projectID := uuid.New()
		databaseID := uuid.New()
		authorID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "target_database_id", "title", "description",
			"change_type", "sql_content", "rollback_sql", "status", "risk_level",
			"author_id", "assigned_reviewer_id", "scheduled_for", "executed_at",
			"execution_duration_ms", "affected_rows", "error_message", "created_at", "updated_at",
		})

		mock.ExpectQuery("SELECT.*FROM change_plans.*WHERE").
			WithArgs("Open", projectID, databaseID, authorID, 10).
			WillReturnRows(rows)

		filters := repository.ChangePlanListFilters{
			Status:         models.ChangePlanStatusOpen,
			ProjectID:      projectID,
			DatabaseID:     databaseID,
			AuthorID:       authorID,
			Limit:          10,
			Offset:         0,
		}

		changePlans, err := repo.ListWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.NotNil(t, changePlans)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT.*FROM change_plans").
			WillReturnError(errors.New("database error"))

		filters := repository.ChangePlanListFilters{
			Limit:  10,
			Offset: 0,
		}

		changePlans, err := repo.ListWithFilters(ctx, filters)
		assert.Nil(t, changePlans)
		assert.Error(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChangePlanRepository_CountWithFilters(t *testing.T) {
	db, mock := setupChangePlanMockDB(t)
	defer db.Close()

	repo := NewChangePlanRepository(db)
	ctx := context.Background()

	t.Run("count all change plans", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).AddRow(5)

		mock.ExpectQuery("SELECT COUNT.*FROM change_plans").
			WillReturnRows(rows)

		filters := repository.ChangePlanListFilters{
			Limit:  10,
			Offset: 0,
		}

		count, err := repo.CountWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count with status filter", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).AddRow(3)

		mock.ExpectQuery("SELECT COUNT.*FROM change_plans.*WHERE.*status").
			WithArgs("Open").
			WillReturnRows(rows)

		filters := repository.ChangePlanListFilters{
			Status: models.ChangePlanStatusOpen,
			Limit:  10,
			Offset: 0,
		}

		count, err := repo.CountWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count with combined filters", func(t *testing.T) {
		projectID := uuid.New()
		rows := sqlmock.NewRows([]string{"count"}).AddRow(2)

		mock.ExpectQuery("SELECT COUNT.*FROM change_plans.*WHERE").
			WillReturnRows(rows)

		filters := repository.ChangePlanListFilters{
			Status:    models.ChangePlanStatusOpen,
			ProjectID: projectID,
			Limit:     10,
			Offset:    0,
		}

		count, err := repo.CountWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT.*FROM change_plans").
			WillReturnError(errors.New("database error"))

		filters := repository.ChangePlanListFilters{
			Limit:  10,
			Offset: 0,
		}

		count, err := repo.CountWithFilters(ctx, filters)
		assert.Equal(t, int64(0), count)
		assert.Error(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

