package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
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

func setupSchemaHistoryMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestSchemaHistoryRepository_Create(t *testing.T) {
	db, mock := setupSchemaHistoryMockDB(t)
	defer db.Close()

	repo := NewSchemaHistoryRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		databaseInstanceID := uuid.New()
		databaseID := uuid.New()
		metadata := map[string]interface{}{
			"database_name":   "test_db",
			"database_engine": "postgresql",
			"host":            "localhost",
			"port":            5432,
		}
		metadataJSON, _ := json.Marshal(metadata)
		rawDump := "CREATE TABLE users (id SERIAL PRIMARY KEY);"

		schemaHistory := &models.SchemaHistory{
			DatabaseInstanceID: databaseInstanceID,
			DatabaseID:         databaseID,
			Metadata:           metadataJSON,
			RawDump:            rawDump,
		}

		mock.ExpectExec("INSERT INTO schema_history").
			WithArgs(sqlmock.AnyArg(), schemaHistory.DatabaseInstanceID, schemaHistory.DatabaseID,
				schemaHistory.Metadata, schemaHistory.RawDump, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, schemaHistory)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, schemaHistory.ID)
		assert.False(t, schemaHistory.CreatedAt.IsZero())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with existing ID", func(t *testing.T) {
		existingID := uuid.New()
		databaseInstanceID := uuid.New()
		databaseID := uuid.New()
		metadata := json.RawMessage(`{"database_name":"test_db"}`)
		rawDump := "CREATE TABLE users (id SERIAL PRIMARY KEY);"

		schemaHistory := &models.SchemaHistory{
			ID:                 existingID,
			DatabaseInstanceID: databaseInstanceID,
			DatabaseID:         databaseID,
			Metadata:           metadata,
			RawDump:            rawDump,
		}

		mock.ExpectExec("INSERT INTO schema_history").
			WithArgs(existingID, schemaHistory.DatabaseInstanceID, schemaHistory.DatabaseID,
				schemaHistory.Metadata, schemaHistory.RawDump, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, schemaHistory)
		require.NoError(t, err)
		assert.Equal(t, existingID, schemaHistory.ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		schemaHistory := &models.SchemaHistory{
			DatabaseInstanceID: uuid.New(),
			DatabaseID:         uuid.New(),
			Metadata:           json.RawMessage(`{}`),
			RawDump:            "CREATE TABLE test (id INT);",
		}

		mock.ExpectExec("INSERT INTO schema_history").
			WillReturnError(errors.New("database error"))

		err := repo.Create(ctx, schemaHistory)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSchemaHistoryRepository_GetByID(t *testing.T) {
	db, mock := setupSchemaHistoryMockDB(t)
	defer db.Close()

	repo := NewSchemaHistoryRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		id := uuid.New()
		databaseInstanceID := uuid.New()
		databaseID := uuid.New()
		metadata := json.RawMessage(`{"database_name":"test_db","database_engine":"postgresql"}`)
		rawDump := "CREATE TABLE users (id SERIAL PRIMARY KEY);"
		createdAt := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "database_instance_id", "database_id", "metadata", "raw_dump", "created_at",
		}).AddRow(
			id, databaseInstanceID, databaseID, metadata, rawDump, createdAt,
		)

		mock.ExpectQuery("SELECT (.+) FROM schema_history WHERE id = \\$1").
			WithArgs(id).
			WillReturnRows(rows)

		schemaHistory, err := repo.GetByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, id, schemaHistory.ID)
		assert.Equal(t, databaseInstanceID, schemaHistory.DatabaseInstanceID)
		assert.Equal(t, databaseID, schemaHistory.DatabaseID)
		assert.Equal(t, metadata, schemaHistory.Metadata)
		assert.Equal(t, rawDump, schemaHistory.RawDump)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM schema_history WHERE id = \\$1").
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		schemaHistory, err := repo.GetByID(ctx, id)
		require.Error(t, err)
		assert.Equal(t, repository.ErrNotFound, err)
		assert.Nil(t, schemaHistory)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM schema_history WHERE id = \\$1").
			WithArgs(id).
			WillReturnError(errors.New("database error"))

		schemaHistory, err := repo.GetByID(ctx, id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, schemaHistory)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSchemaHistoryRepository_GetByDatabaseID(t *testing.T) {
	db, mock := setupSchemaHistoryMockDB(t)
	defer db.Close()

	repo := NewSchemaHistoryRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		databaseID := uuid.New()
		id1 := uuid.New()
		id2 := uuid.New()
		databaseInstanceID := uuid.New()
		metadata := json.RawMessage(`{"database_name":"test_db"}`)
		rawDump1 := "CREATE TABLE users (id SERIAL PRIMARY KEY);"
		rawDump2 := "CREATE TABLE posts (id SERIAL PRIMARY KEY);"
		createdAt1 := time.Now()
		createdAt2 := time.Now().Add(-1 * time.Hour)

		rows := sqlmock.NewRows([]string{
			"id", "database_instance_id", "database_id", "metadata", "raw_dump", "created_at",
		}).AddRow(
			id1, databaseInstanceID, databaseID, metadata, rawDump1, createdAt1,
		).AddRow(
			id2, databaseInstanceID, databaseID, metadata, rawDump2, createdAt2,
		)

		mock.ExpectQuery("SELECT (.+) FROM schema_history WHERE database_id = \\$1 ORDER BY created_at DESC").
			WithArgs(databaseID).
			WillReturnRows(rows)

		histories, err := repo.GetByDatabaseID(ctx, databaseID)
		require.NoError(t, err)
		assert.Len(t, histories, 2)
		assert.Equal(t, id1, histories[0].ID)
		assert.Equal(t, id2, histories[1].ID)
		assert.Equal(t, rawDump1, histories[0].RawDump)
		assert.Equal(t, rawDump2, histories[1].RawDump)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no results", func(t *testing.T) {
		databaseID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "database_instance_id", "database_id", "metadata", "raw_dump", "created_at",
		})

		mock.ExpectQuery("SELECT (.+) FROM schema_history WHERE database_id = \\$1 ORDER BY created_at DESC").
			WithArgs(databaseID).
			WillReturnRows(rows)

		histories, err := repo.GetByDatabaseID(ctx, databaseID)
		require.NoError(t, err)
		assert.Len(t, histories, 0)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSchemaHistoryRepository_GetLatestByDatabaseID(t *testing.T) {
	db, mock := setupSchemaHistoryMockDB(t)
	defer db.Close()

	repo := NewSchemaHistoryRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		databaseID := uuid.New()
		id := uuid.New()
		databaseInstanceID := uuid.New()
		metadata := json.RawMessage(`{"database_name":"test_db","database_engine":"postgresql"}`)
		rawDump := "CREATE TABLE users (id SERIAL PRIMARY KEY);"
		createdAt := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "database_instance_id", "database_id", "metadata", "raw_dump", "created_at",
		}).AddRow(
			id, databaseInstanceID, databaseID, metadata, rawDump, createdAt,
		)

		mock.ExpectQuery("SELECT (.+) FROM schema_history WHERE database_id = \\$1 ORDER BY created_at DESC LIMIT 1").
			WithArgs(databaseID).
			WillReturnRows(rows)

		schemaHistory, err := repo.GetLatestByDatabaseID(ctx, databaseID)
		require.NoError(t, err)
		assert.Equal(t, id, schemaHistory.ID)
		assert.Equal(t, databaseID, schemaHistory.DatabaseID)
		assert.Equal(t, rawDump, schemaHistory.RawDump)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		databaseID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM schema_history WHERE database_id = \\$1 ORDER BY created_at DESC LIMIT 1").
			WithArgs(databaseID).
			WillReturnError(sql.ErrNoRows)

		schemaHistory, err := repo.GetLatestByDatabaseID(ctx, databaseID)
		require.Error(t, err)
		assert.Equal(t, repository.ErrNotFound, err)
		assert.Nil(t, schemaHistory)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		databaseID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM schema_history WHERE database_id = \\$1 ORDER BY created_at DESC LIMIT 1").
			WithArgs(databaseID).
			WillReturnError(errors.New("database error"))

		schemaHistory, err := repo.GetLatestByDatabaseID(ctx, databaseID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		assert.Nil(t, schemaHistory)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
