package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
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

func setupSchemaSnapshotMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestSchemaSnapshotRepository_Create(t *testing.T) {
	db, mock := setupSchemaSnapshotMockDB(t)
	defer db.Close()

	repo := NewSchemaSnapshotRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		snapshot := &models.SchemaSnapshot{
			DatabaseID:   uuid.New(),
			ChangePlanID: uuid.New(),
			SnapshotType: models.SnapshotTypeBefore,
			SnapshotData: json.RawMessage(`{"sql": "CREATE TABLE test (id UUID);", "format": "plain"}`),
		}

		mock.ExpectExec("INSERT INTO schema_snapshots").
			WithArgs(
				sqlmock.AnyArg(),
				snapshot.DatabaseID,
				snapshot.ChangePlanID,
				snapshot.SnapshotType,
				snapshot.SnapshotData,
				nil,
				sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, snapshot)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, snapshot.ID)
		assert.False(t, snapshot.CreatedAt.IsZero())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with existing ID", func(t *testing.T) {
		existingID := uuid.New()
		snapshot := &models.SchemaSnapshot{
			ID:           existingID,
			DatabaseID:   uuid.New(),
			ChangePlanID: uuid.New(),
			SnapshotType: models.SnapshotTypeAfter,
			SnapshotData: json.RawMessage(`{"sql": "ALTER TABLE test ADD COLUMN name VARCHAR(255);", "format": "plain"}`),
		}

		mock.ExpectExec("INSERT INTO schema_snapshots").
			WithArgs(
				existingID,
				snapshot.DatabaseID,
				snapshot.ChangePlanID,
				snapshot.SnapshotType,
				snapshot.SnapshotData,
				nil,
				sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, snapshot)
		require.NoError(t, err)
		assert.Equal(t, existingID, snapshot.ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with snapshot_data_raw", func(t *testing.T) {
		rawSQL := "CREATE TABLE test (id UUID PRIMARY KEY);"
		snapshot := &models.SchemaSnapshot{
			DatabaseID:      uuid.New(),
			ChangePlanID:    uuid.New(),
			SnapshotType:    models.SnapshotTypeBefore,
			SnapshotData:    json.RawMessage(`{"sql": "` + rawSQL + `", "format": "plain"}`),
			SnapshotDataRaw: &rawSQL,
		}

		mock.ExpectExec("INSERT INTO schema_snapshots").
			WithArgs(
				sqlmock.AnyArg(),
				snapshot.DatabaseID,
				snapshot.ChangePlanID,
				snapshot.SnapshotType,
				snapshot.SnapshotData,
				rawSQL,
				sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, snapshot)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSchemaSnapshotRepository_GetByID(t *testing.T) {
	db, mock := setupSchemaSnapshotMockDB(t)
	defer db.Close()

	repo := NewSchemaSnapshotRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		snapshotID := uuid.New()
		databaseID := uuid.New()
		changePlanID := uuid.New()
		snapshotData := json.RawMessage(`{"sql": "CREATE TABLE test (id UUID);", "format": "plain"}`)
		createdAt := time.Now()

		rows := sqlmock.NewRows([]string{"id", "database_id", "change_plan_id", "snapshot_type", "snapshot_data", "snapshot_data_raw", "created_at"}).
			AddRow(snapshotID, databaseID, changePlanID, models.SnapshotTypeBefore, snapshotData, nil, createdAt)

		mock.ExpectQuery("SELECT id, database_id, change_plan_id, snapshot_type").
			WithArgs(snapshotID).
			WillReturnRows(rows)

		snapshot, err := repo.GetByID(ctx, snapshotID)
		require.NoError(t, err)
		assert.NotNil(t, snapshot)
		assert.Equal(t, snapshotID, snapshot.ID)
		assert.Equal(t, databaseID, snapshot.DatabaseID)
		assert.Equal(t, changePlanID, snapshot.ChangePlanID)
		assert.Equal(t, models.SnapshotTypeBefore, snapshot.SnapshotType)
		assert.Equal(t, snapshotData, snapshot.SnapshotData)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		snapshotID := uuid.New()

		mock.ExpectQuery("SELECT id, database_id, change_plan_id, snapshot_type").
			WithArgs(snapshotID).
			WillReturnError(sql.ErrNoRows)

		snapshot, err := repo.GetByID(ctx, snapshotID)
		assert.Error(t, err)
		assert.Nil(t, snapshot)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSchemaSnapshotRepository_GetByChangePlanID(t *testing.T) {
	db, mock := setupSchemaSnapshotMockDB(t)
	defer db.Close()

	repo := NewSchemaSnapshotRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval with multiple snapshots", func(t *testing.T) {
		changePlanID := uuid.New()
		databaseID := uuid.New()
		snapshotID1 := uuid.New()
		snapshotID2 := uuid.New()

		snapshotData1 := json.RawMessage(`{"sql": "CREATE TABLE test (id UUID);", "format": "plain"}`)
		snapshotData2 := json.RawMessage(`{"sql": "ALTER TABLE test ADD COLUMN name VARCHAR(255);", "format": "plain"}`)
		createdAt1 := time.Now().Add(-2 * time.Hour)
		createdAt2 := time.Now()

		rows := sqlmock.NewRows([]string{"id", "database_id", "change_plan_id", "snapshot_type", "snapshot_data", "snapshot_data_raw", "created_at"}).
			AddRow(snapshotID2, databaseID, changePlanID, models.SnapshotTypeAfter, snapshotData2, nil, createdAt2).
			AddRow(snapshotID1, databaseID, changePlanID, models.SnapshotTypeBefore, snapshotData1, nil, createdAt1)

		mock.ExpectQuery("SELECT id, database_id, change_plan_id, snapshot_type").
			WithArgs(changePlanID).
			WillReturnRows(rows)

		snapshots, err := repo.GetByChangePlanID(ctx, changePlanID)
		require.NoError(t, err)
		assert.Len(t, snapshots, 2)
		assert.Equal(t, snapshotID2, snapshots[0].ID) // Most recent first
		assert.Equal(t, snapshotID1, snapshots[1].ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty result", func(t *testing.T) {
		changePlanID := uuid.New()

		rows := sqlmock.NewRows(
			[]string{"id", "database_id", "change_plan_id", "snapshot_type", "snapshot_data", "snapshot_data_raw", "created_at"},
		)

		mock.ExpectQuery("SELECT id, database_id, change_plan_id, snapshot_type").
			WithArgs(changePlanID).
			WillReturnRows(rows)

		snapshots, err := repo.GetByChangePlanID(ctx, changePlanID)
		require.NoError(t, err)
		assert.Len(t, snapshots, 0)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSchemaSnapshotRepository_GetByDatabaseID(t *testing.T) {
	db, mock := setupSchemaSnapshotMockDB(t)
	defer db.Close()

	repo := NewSchemaSnapshotRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval with multiple snapshots", func(t *testing.T) {
		databaseID := uuid.New()
		changePlanID1 := uuid.New()
		changePlanID2 := uuid.New()
		snapshotID1 := uuid.New()
		snapshotID2 := uuid.New()

		snapshotData1 := json.RawMessage(`{"sql": "CREATE TABLE test1 (id UUID);", "format": "plain"}`)
		snapshotData2 := json.RawMessage(`{"sql": "CREATE TABLE test2 (id UUID);", "format": "plain"}`)
		createdAt1 := time.Now().Add(-2 * time.Hour)
		createdAt2 := time.Now()

		rows := sqlmock.NewRows([]string{"id", "database_id", "change_plan_id", "snapshot_type", "snapshot_data", "snapshot_data_raw", "created_at"}).
			AddRow(snapshotID2, databaseID, changePlanID2, models.SnapshotTypeBefore, snapshotData2, nil, createdAt2).
			AddRow(snapshotID1, databaseID, changePlanID1, models.SnapshotTypeBefore, snapshotData1, nil, createdAt1)

		mock.ExpectQuery("SELECT id, database_id, change_plan_id, snapshot_type").
			WithArgs(databaseID).
			WillReturnRows(rows)

		snapshots, err := repo.GetByDatabaseID(ctx, databaseID)
		require.NoError(t, err)
		assert.Len(t, snapshots, 2)
		assert.Equal(t, snapshotID2, snapshots[0].ID) // Most recent first
		assert.Equal(t, snapshotID1, snapshots[1].ID)
		assert.Equal(t, databaseID, snapshots[0].DatabaseID)
		assert.Equal(t, databaseID, snapshots[1].DatabaseID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty result", func(t *testing.T) {
		databaseID := uuid.New()

		rows := sqlmock.NewRows(
			[]string{"id", "database_id", "change_plan_id", "snapshot_type", "snapshot_data", "snapshot_data_raw", "created_at"},
		)

		mock.ExpectQuery("SELECT id, database_id, change_plan_id, snapshot_type").
			WithArgs(databaseID).
			WillReturnRows(rows)

		snapshots, err := repo.GetByDatabaseID(ctx, databaseID)
		require.NoError(t, err)
		assert.Len(t, snapshots, 0)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
