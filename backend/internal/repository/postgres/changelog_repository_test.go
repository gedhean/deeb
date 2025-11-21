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

func setupChangelogMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestChangelogRepository_Create(t *testing.T) {
	db, mock := setupChangelogMockDB(t)
	defer db.Close()

	repo := NewChangelogRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		databaseInstanceID := uuid.New()
		databaseID := uuid.New()
		changePlanID := uuid.New()
		prevSchemaHistoryID := uuid.New()

		changelog := &models.Changelog{
			DatabaseInstanceID:  databaseInstanceID,
			DatabaseID:          databaseID,
			ChangePlanID:        changePlanID,
			Status:              models.ChangelogStatusPending,
			PrevSchemaHistoryID: &prevSchemaHistoryID,
		}

		mock.ExpectExec("INSERT INTO changelog").
			WithArgs(sqlmock.AnyArg(), changelog.DatabaseInstanceID, changelog.DatabaseID,
				changelog.ChangePlanID, changelog.Status, changelog.PrevSchemaHistoryID,
				nil, nil, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, changelog)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, changelog.ID)
		assert.False(t, changelog.CreatedAt.IsZero())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with nil prev_schema_history_id", func(t *testing.T) {
		databaseInstanceID := uuid.New()
		databaseID := uuid.New()
		changePlanID := uuid.New()

		changelog := &models.Changelog{
			DatabaseInstanceID:  databaseInstanceID,
			DatabaseID:          databaseID,
			ChangePlanID:        changePlanID,
			Status:              models.ChangelogStatusPending,
			PrevSchemaHistoryID: nil,
		}

		mock.ExpectExec("INSERT INTO changelog").
			WithArgs(sqlmock.AnyArg(), changelog.DatabaseInstanceID, changelog.DatabaseID,
				changelog.ChangePlanID, changelog.Status, nil, nil, nil, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, changelog)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, changelog.ID)
		assert.Nil(t, changelog.PrevSchemaHistoryID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		changelog := &models.Changelog{
			DatabaseInstanceID: uuid.New(),
			DatabaseID:         uuid.New(),
			ChangePlanID:       uuid.New(),
			Status:             models.ChangelogStatusPending,
		}

		mock.ExpectExec("INSERT INTO changelog").
			WillReturnError(errors.New("database error"))

		err := repo.Create(ctx, changelog)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChangelogRepository_GetByID(t *testing.T) {
	db, mock := setupChangelogMockDB(t)
	defer db.Close()

	repo := NewChangelogRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		id := uuid.New()
		databaseInstanceID := uuid.New()
		databaseID := uuid.New()
		changePlanID := uuid.New()
		prevSchemaHistoryID := uuid.New()
		schemaHistoryID := uuid.New()
		createdAt := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "database_instance_id", "database_id", "change_plan_id",
			"status", "prev_schema_history_id", "schema_history_id", "created_at",
		}).AddRow(
			id, databaseInstanceID, databaseID, changePlanID,
			models.ChangelogStatusDone, prevSchemaHistoryID, schemaHistoryID, createdAt,
		)

		mock.ExpectQuery("SELECT (.+) FROM changelog WHERE id = \\$1").
			WithArgs(id).
			WillReturnRows(rows)

		changelog, err := repo.GetByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, id, changelog.ID)
		assert.Equal(t, databaseInstanceID, changelog.DatabaseInstanceID)
		assert.Equal(t, databaseID, changelog.DatabaseID)
		assert.Equal(t, changePlanID, changelog.ChangePlanID)
		assert.Equal(t, models.ChangelogStatusDone, changelog.Status)
		assert.Equal(t, prevSchemaHistoryID, *changelog.PrevSchemaHistoryID)
		assert.Equal(t, schemaHistoryID, *changelog.SchemaHistoryID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM changelog WHERE id = \\$1").
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		changelog, err := repo.GetByID(ctx, id)
		require.Error(t, err)
		assert.Equal(t, repository.ErrNotFound, err)
		assert.Nil(t, changelog)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM changelog WHERE id = \\$1").
			WithArgs(id).
			WillReturnError(errors.New("database error"))

		changelog, err := repo.GetByID(ctx, id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, changelog)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChangelogRepository_GetByChangePlanID(t *testing.T) {
	db, mock := setupChangelogMockDB(t)
	defer db.Close()

	repo := NewChangelogRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		changePlanID := uuid.New()
		id1 := uuid.New()
		id2 := uuid.New()
		databaseInstanceID := uuid.New()
		databaseID := uuid.New()
		createdAt1 := time.Now()
		createdAt2 := time.Now().Add(-1 * time.Hour)

		rows := sqlmock.NewRows([]string{
			"id", "database_instance_id", "database_id", "change_plan_id",
			"status", "prev_schema_history_id", "schema_history_id", "created_at",
		}).AddRow(
			id1, databaseInstanceID, databaseID, changePlanID,
			models.ChangelogStatusDone, nil, nil, createdAt1,
		).AddRow(
			id2, databaseInstanceID, databaseID, changePlanID,
			models.ChangelogStatusPending, nil, nil, createdAt2,
		)

		mock.ExpectQuery("SELECT (.+) FROM changelog WHERE change_plan_id = \\$1 ORDER BY created_at DESC").
			WithArgs(changePlanID).
			WillReturnRows(rows)

		changelogs, err := repo.GetByChangePlanID(ctx, changePlanID)
		require.NoError(t, err)
		assert.Len(t, changelogs, 2)
		assert.Equal(t, id1, changelogs[0].ID)
		assert.Equal(t, id2, changelogs[1].ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no results", func(t *testing.T) {
		changePlanID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "database_instance_id", "database_id", "change_plan_id",
			"status", "prev_schema_history_id", "schema_history_id", "created_at",
		})

		mock.ExpectQuery("SELECT (.+) FROM changelog WHERE change_plan_id = \\$1 ORDER BY created_at DESC").
			WithArgs(changePlanID).
			WillReturnRows(rows)

		changelogs, err := repo.GetByChangePlanID(ctx, changePlanID)
		require.NoError(t, err)
		assert.Len(t, changelogs, 0)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChangelogRepository_GetByDatabaseID(t *testing.T) {
	db, mock := setupChangelogMockDB(t)
	defer db.Close()

	repo := NewChangelogRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		databaseID := uuid.New()
		id1 := uuid.New()
		id2 := uuid.New()
		databaseInstanceID := uuid.New()
		changePlanID := uuid.New()
		createdAt1 := time.Now()
		createdAt2 := time.Now().Add(-1 * time.Hour)

		rows := sqlmock.NewRows([]string{
			"id", "database_instance_id", "database_id", "change_plan_id",
			"status", "prev_schema_history_id", "schema_history_id", "created_at",
		}).AddRow(
			id1, databaseInstanceID, databaseID, changePlanID,
			models.ChangelogStatusDone, nil, nil, createdAt1,
		).AddRow(
			id2, databaseInstanceID, databaseID, changePlanID,
			models.ChangelogStatusFailed, nil, nil, createdAt2,
		)

		mock.ExpectQuery("SELECT (.+) FROM changelog WHERE database_id = \\$1 ORDER BY created_at DESC").
			WithArgs(databaseID).
			WillReturnRows(rows)

		changelogs, err := repo.GetByDatabaseID(ctx, databaseID)
		require.NoError(t, err)
		assert.Len(t, changelogs, 2)
		assert.Equal(t, models.ChangelogStatusDone, changelogs[0].Status)
		assert.Equal(t, models.ChangelogStatusFailed, changelogs[1].Status)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestChangelogRepository_Update(t *testing.T) {
	db, mock := setupChangelogMockDB(t)
	defer db.Close()

	repo := NewChangelogRepository(db)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		id := uuid.New()
		schemaHistoryID := uuid.New()

		changelog := &models.Changelog{
			ID:              id,
			Status:          models.ChangelogStatusDone,
			SchemaHistoryID: &schemaHistoryID,
		}

		mock.ExpectExec("UPDATE changelog SET status = \\$2, prev_schema_history_id = \\$3, schema_history_id = \\$4, metadata = \\$5 WHERE id = \\$1").
			WithArgs(changelog.ID, changelog.Status, changelog.PrevSchemaHistoryID, changelog.SchemaHistoryID, nil).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, changelog)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		id := uuid.New()

		changelog := &models.Changelog{
			ID:     id,
			Status: models.ChangelogStatusDone,
		}

		mock.ExpectExec("UPDATE changelog SET status = \\$2, prev_schema_history_id = \\$3, schema_history_id = \\$4, metadata = \\$5 WHERE id = \\$1").
			WithArgs(changelog.ID, changelog.Status, changelog.PrevSchemaHistoryID, changelog.SchemaHistoryID, nil).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(ctx, changelog)
		require.Error(t, err)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		id := uuid.New()

		changelog := &models.Changelog{
			ID:     id,
			Status: models.ChangelogStatusDone,
		}

		mock.ExpectExec("UPDATE changelog SET status = \\$2, prev_schema_history_id = \\$3, schema_history_id = \\$4, metadata = \\$5 WHERE id = \\$1").
			WithArgs(changelog.ID, changelog.Status, changelog.PrevSchemaHistoryID, changelog.SchemaHistoryID, nil).
			WillReturnError(errors.New("database error"))

		err := repo.Update(ctx, changelog)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
