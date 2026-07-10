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

func setupDatabaseMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestDatabaseRepository_Create(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()
		description := "Test database description"
		database := &models.Database{
			DatabaseName:       "test_database",
			DatabaseInstanceID: dbInstanceID,
			ProjectID:          &projectID,
			EnvironmentID:      &envID,
			Description:        &description,
			Status:             models.DatabaseStatusActive,
			CreatedByID:        createdByID,
		}

		mock.ExpectExec("INSERT INTO databases").
			WithArgs(sqlmock.AnyArg(), database.ProjectID, database.DatabaseInstanceID, database.EnvironmentID,
				database.DatabaseName, database.Description, database.Status, database.CreatedByID,
				sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, database)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, database.ID)
		assert.False(t, database.CreatedAt.IsZero())
		assert.False(t, database.UpdatedAt.IsZero())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with nil description", func(t *testing.T) {
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()
		database := &models.Database{
			DatabaseName:       "test_database",
			DatabaseInstanceID: dbInstanceID,
			ProjectID:          &projectID,
			EnvironmentID:      &envID,
			Description:        nil,
			Status:             models.DatabaseStatusActive,
			CreatedByID:        createdByID,
		}

		mock.ExpectExec("INSERT INTO databases").
			WithArgs(sqlmock.AnyArg(), database.ProjectID, database.DatabaseInstanceID, database.EnvironmentID,
				database.DatabaseName, nil, database.Status, database.CreatedByID,
				sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, database)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, database.ID)
		assert.Nil(t, database.Description)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with existing ID", func(t *testing.T) {
		existingID := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()
		description := "Test database description"
		database := &models.Database{
			ID:                 existingID,
			DatabaseName:       "test_database",
			DatabaseInstanceID: dbInstanceID,
			ProjectID:          &projectID,
			EnvironmentID:      &envID,
			Description:        &description,
			Status:             models.DatabaseStatusActive,
			CreatedByID:        createdByID,
		}

		mock.ExpectExec("INSERT INTO databases").
			WithArgs(existingID, database.ProjectID, database.DatabaseInstanceID, database.EnvironmentID,
				database.DatabaseName, database.Description, database.Status, database.CreatedByID,
				sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, database)
		require.NoError(t, err)
		assert.Equal(t, existingID, database.ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		projectID := uuid.New()
		envID := uuid.New()
		database := &models.Database{
			DatabaseName:       "",
			DatabaseInstanceID: uuid.New(),
			ProjectID:          &projectID,
			EnvironmentID:      &envID,
			Status:             models.DatabaseStatusActive,
			CreatedByID:        uuid.New(),
		}

		err := repo.Create(ctx, database)
		assert.Equal(t, models.ErrDatabaseNameRequired, err)
	})

	t.Run("validation error - nil database instance ID", func(t *testing.T) {
		projectID := uuid.New()
		envID := uuid.New()
		database := &models.Database{
			DatabaseName:       "test_database",
			DatabaseInstanceID: uuid.Nil,
			ProjectID:          &projectID,
			EnvironmentID:      &envID,
			Status:             models.DatabaseStatusActive,
			CreatedByID:        uuid.New(),
		}

		err := repo.Create(ctx, database)
		assert.Equal(t, models.ErrDatabaseInstanceIDRequired, err)
	})

	t.Run("database error", func(t *testing.T) {
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()
		description := "Test database description"
		database := &models.Database{
			DatabaseName:       "test_database",
			DatabaseInstanceID: dbInstanceID,
			ProjectID:          &projectID,
			EnvironmentID:      &envID,
			Description:        &description,
			Status:             models.DatabaseStatusActive,
			CreatedByID:        createdByID,
		}

		mock.ExpectExec("INSERT INTO databases").
			WithArgs(sqlmock.AnyArg(), database.ProjectID, database.DatabaseInstanceID, database.EnvironmentID,
				database.DatabaseName, database.Description, database.Status, database.CreatedByID,
				sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("database error"))

		err := repo.Create(ctx, database)
		require.Error(t, err)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseRepository_GetByID(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		databaseID := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()
		description := "Test database description"
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(databaseID, projectID, dbInstanceID, envID, "test_database", description, "Active", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE id =").
			WithArgs(databaseID).
			WillReturnRows(rows)

		database, err := repo.GetByID(ctx, databaseID)
		require.NoError(t, err)
		require.NotNil(t, database)
		assert.Equal(t, databaseID, database.ID)
		assert.Equal(t, "test_database", database.DatabaseName)
		assert.Equal(t, &description, database.Description)
		assert.NotNil(t, database.ProjectID)
		assert.Equal(t, projectID, *database.ProjectID)
		assert.Equal(t, dbInstanceID, database.DatabaseInstanceID)
		assert.NotNil(t, database.EnvironmentID)
		assert.Equal(t, envID, *database.EnvironmentID)
		assert.Equal(t, models.DatabaseStatusActive, database.Status)
		assert.Equal(t, createdByID, database.CreatedByID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful retrieval with nil description", func(t *testing.T) {
		databaseID := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(databaseID, projectID, dbInstanceID, envID, "test_database", nil, "Active", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE id =").
			WithArgs(databaseID).
			WillReturnRows(rows)

		database, err := repo.GetByID(ctx, databaseID)
		require.NoError(t, err)
		require.NotNil(t, database)
		assert.Nil(t, database.Description)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database not found", func(t *testing.T) {
		databaseID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE id =").
			WithArgs(databaseID).
			WillReturnError(sql.ErrNoRows)

		database, err := repo.GetByID(ctx, databaseID)
		assert.Equal(t, repository.ErrNotFound, err)
		assert.Nil(t, database)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		databaseID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE id =").
			WithArgs(databaseID).
			WillReturnError(errors.New("database error"))

		database, err := repo.GetByID(ctx, databaseID)
		require.Error(t, err)
		assert.Nil(t, database)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseRepository_List(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful list with results", func(t *testing.T) {
		now := time.Now()
		databaseID1 := uuid.New()
		databaseID2 := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(databaseID1, projectID, dbInstanceID, envID, "database1", nil, "Active", createdByID, now, now).
			AddRow(databaseID2, projectID, dbInstanceID, envID, "database2", nil, "Active", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM databases ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		databases, err := repo.List(ctx, repository.DatabaseFilters{}, 10, 0)
		require.NoError(t, err)
		assert.Len(t, databases, 2)
		assert.Equal(t, databaseID1, databases[0].ID)
		assert.Equal(t, databaseID2, databases[1].ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		})

		mock.ExpectQuery("SELECT (.+) FROM databases ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		databases, err := repo.List(ctx, repository.DatabaseFilters{}, 10, 0)
		require.NoError(t, err)
		assert.Len(t, databases, 0)
		assert.NotNil(t, databases) // Should return empty slice, not nil

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("list with pagination", func(t *testing.T) {
		now := time.Now()
		databaseID := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(databaseID, projectID, dbInstanceID, envID, "database1", nil, "Active", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM databases ORDER BY created_at DESC LIMIT").
			WithArgs(5, 10).
			WillReturnRows(rows)

		databases, err := repo.List(ctx, repository.DatabaseFilters{}, 5, 10)
		require.NoError(t, err)
		assert.Len(t, databases, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM databases ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnError(errors.New("database error"))

		databases, err := repo.List(ctx, repository.DatabaseFilters{}, 10, 0)
		require.Error(t, err)
		assert.Nil(t, databases)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseRepository_Count(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful count", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM databases").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

		count, err := repo.Count(ctx, repository.DatabaseFilters{})
		require.NoError(t, err)
		assert.Equal(t, 42, count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count zero", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM databases").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		count, err := repo.Count(ctx, repository.DatabaseFilters{})
		require.NoError(t, err)
		assert.Equal(t, 0, count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count large number", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM databases").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1000))

		count, err := repo.Count(ctx, repository.DatabaseFilters{})
		require.NoError(t, err)
		assert.Equal(t, 1000, count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM databases").
			WillReturnError(errors.New("database error"))

		count, err := repo.Count(ctx, repository.DatabaseFilters{})
		require.Error(t, err)
		assert.Equal(t, 0, count)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseRepository_ListByProject(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful list by project", func(t *testing.T) {
		now := time.Now()
		databaseID1 := uuid.New()
		databaseID2 := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(databaseID1, projectID, dbInstanceID, envID, "database1", nil, "Active", createdByID, now, now).
			AddRow(databaseID2, projectID, dbInstanceID, envID, "database2", nil, "Active", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE project_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(projectID, 10, 0).
			WillReturnRows(rows)

		databases, err := repo.ListByProject(ctx, projectID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, databases, 2)
		assert.Equal(t, databaseID1, databases[0].ID)
		assert.Equal(t, databaseID2, databases[1].ID)
		assert.NotNil(t, databases[0].ProjectID)
		assert.Equal(t, projectID, *databases[0].ProjectID)
		assert.NotNil(t, databases[1].ProjectID)
		assert.Equal(t, projectID, *databases[1].ProjectID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list by project", func(t *testing.T) {
		projectID := uuid.New()
		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		})

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE project_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(projectID, 10, 0).
			WillReturnRows(rows)

		databases, err := repo.ListByProject(ctx, projectID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, databases, 0)
		assert.NotNil(t, databases)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("list by project with pagination", func(t *testing.T) {
		now := time.Now()
		databaseID := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(databaseID, projectID, dbInstanceID, envID, "database1", nil, "Active", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE project_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(projectID, 5, 10).
			WillReturnRows(rows)

		databases, err := repo.ListByProject(ctx, projectID, 5, 10)
		require.NoError(t, err)
		assert.Len(t, databases, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		projectID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE project_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(projectID, 10, 0).
			WillReturnError(errors.New("database error"))

		databases, err := repo.ListByProject(ctx, projectID, 10, 0)
		require.Error(t, err)
		assert.Nil(t, databases)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseRepository_GetByProjectID(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful get by project ID", func(t *testing.T) {
		now := time.Now()
		databaseID1 := uuid.New()
		databaseID2 := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(databaseID1, projectID, dbInstanceID, envID, "database1", nil, "Active", createdByID, now, now).
			AddRow(databaseID2, projectID, dbInstanceID, envID, "database2", nil, "Active", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE project_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(projectID, 10, 0).
			WillReturnRows(rows)

		databases, err := repo.GetByProjectID(ctx, projectID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, databases, 2)

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseRepository_ListByInstanceID(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful list by instance ID", func(t *testing.T) {
		now := time.Now()
		databaseID1 := uuid.New()
		databaseID2 := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(databaseID1, projectID, dbInstanceID, envID, "database1", nil, "Active", createdByID, now, now).
			AddRow(databaseID2, projectID, dbInstanceID, envID, "database2", nil, "Active", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE database_instance_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(dbInstanceID, 10, 0).
			WillReturnRows(rows)

		databases, err := repo.ListByInstanceID(ctx, dbInstanceID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, databases, 2)
		assert.Equal(t, databaseID1, databases[0].ID)
		assert.Equal(t, databaseID2, databases[1].ID)
		assert.Equal(t, dbInstanceID, databases[0].DatabaseInstanceID)
		assert.Equal(t, dbInstanceID, databases[1].DatabaseInstanceID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list by instance ID", func(t *testing.T) {
		dbInstanceID := uuid.New()
		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		})

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE database_instance_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(dbInstanceID, 10, 0).
			WillReturnRows(rows)

		databases, err := repo.ListByInstanceID(ctx, dbInstanceID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, databases, 0)
		assert.NotNil(t, databases)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		dbInstanceID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE database_instance_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(dbInstanceID, 10, 0).
			WillReturnError(errors.New("database error"))

		databases, err := repo.ListByInstanceID(ctx, dbInstanceID, 10, 0)
		require.Error(t, err)
		assert.Nil(t, databases)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseRepository_ListByEnvironmentID(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful list by environment ID", func(t *testing.T) {
		now := time.Now()
		databaseID1 := uuid.New()
		databaseID2 := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		createdByID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(databaseID1, projectID, dbInstanceID, envID, "database1", nil, "Active", createdByID, now, now).
			AddRow(databaseID2, projectID, dbInstanceID, envID, "database2", nil, "Active", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE environment_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(envID, 10, 0).
			WillReturnRows(rows)

		databases, err := repo.ListByEnvironmentID(ctx, envID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, databases, 2)
		assert.Equal(t, databaseID1, databases[0].ID)
		assert.Equal(t, databaseID2, databases[1].ID)
		assert.NotNil(t, databases[0].EnvironmentID)
		assert.Equal(t, envID, *databases[0].EnvironmentID)
		assert.NotNil(t, databases[1].EnvironmentID)
		assert.Equal(t, envID, *databases[1].EnvironmentID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list by environment ID", func(t *testing.T) {
		envID := uuid.New()
		rows := sqlmock.NewRows([]string{
			"id", "project_id", "database_instance_id", "environment_id",
			"database_name", "description", "status", "created_by_id", "created_at", "updated_at",
		})

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE environment_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(envID, 10, 0).
			WillReturnRows(rows)

		databases, err := repo.ListByEnvironmentID(ctx, envID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, databases, 0)
		assert.NotNil(t, databases)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		envID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE environment_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(envID, 10, 0).
			WillReturnError(errors.New("database error"))

		databases, err := repo.ListByEnvironmentID(ctx, envID, 10, 0)
		require.Error(t, err)
		assert.Nil(t, databases)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseRepository_Update(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		databaseID := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		description := "Updated description"
		database := &models.Database{
			ID:                 databaseID,
			DatabaseName:       "updated_database",
			DatabaseInstanceID: dbInstanceID,
			ProjectID:          &projectID,
			EnvironmentID:      &envID,
			Description:        &description,
			Status:             models.DatabaseStatusActive,
			CreatedByID:        uuid.New(),
		}

		mock.ExpectExec("UPDATE databases").
			WithArgs(databaseID, database.ProjectID, database.EnvironmentID, database.DatabaseName, database.Description, database.Status, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, database)
		require.NoError(t, err)
		assert.False(t, database.UpdatedAt.IsZero())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful update with nil description", func(t *testing.T) {
		databaseID := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		database := &models.Database{
			ID:                 databaseID,
			DatabaseName:       "updated_database",
			DatabaseInstanceID: dbInstanceID,
			ProjectID:          &projectID,
			EnvironmentID:      &envID,
			Description:        nil,
			Status:             models.DatabaseStatusActive,
			CreatedByID:        uuid.New(),
		}

		mock.ExpectExec("UPDATE databases").
			WithArgs(databaseID, database.ProjectID, database.EnvironmentID, database.DatabaseName, nil, database.Status, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, database)
		require.NoError(t, err)
		assert.Nil(t, database.Description)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database not found", func(t *testing.T) {
		databaseID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		database := &models.Database{
			ID:                 databaseID,
			DatabaseName:       "updated_database",
			DatabaseInstanceID: uuid.New(),
			ProjectID:          &projectID,
			EnvironmentID:      &envID,
			Status:             models.DatabaseStatusActive,
			CreatedByID:        uuid.New(),
		}

		mock.ExpectExec("UPDATE databases").
			WithArgs(databaseID, database.ProjectID, database.EnvironmentID, database.DatabaseName, database.Description, database.Status, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(ctx, database)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		projectID := uuid.New()
		envID := uuid.New()
		database := &models.Database{
			ID:                 uuid.New(),
			DatabaseName:       "",
			DatabaseInstanceID: uuid.New(),
			ProjectID:          &projectID,
			EnvironmentID:      &envID,
			Status:             models.DatabaseStatusActive,
			CreatedByID:        uuid.New(),
		}

		err := repo.Update(ctx, database)
		assert.Equal(t, models.ErrDatabaseNameRequired, err)
	})

	t.Run("database error", func(t *testing.T) {
		databaseID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		database := &models.Database{
			ID:                 databaseID,
			DatabaseName:       "updated_database",
			DatabaseInstanceID: uuid.New(),
			ProjectID:          &projectID,
			EnvironmentID:      &envID,
			Status:             models.DatabaseStatusActive,
			CreatedByID:        uuid.New(),
		}

		mock.ExpectExec("UPDATE databases").
			WithArgs(databaseID, database.ProjectID, database.EnvironmentID, database.DatabaseName, database.Description, database.Status, sqlmock.AnyArg()).
			WillReturnError(errors.New("database error"))

		err := repo.Update(ctx, database)
		require.Error(t, err)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseRepository_Delete(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		databaseID := uuid.New()

		mock.ExpectExec("DELETE FROM databases WHERE id").
			WithArgs(databaseID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(ctx, databaseID)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database not found", func(t *testing.T) {
		databaseID := uuid.New()

		mock.ExpectExec("DELETE FROM databases WHERE id").
			WithArgs(databaseID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(ctx, databaseID)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		databaseID := uuid.New()

		mock.ExpectExec("DELETE FROM databases WHERE id").
			WithArgs(databaseID).
			WillReturnError(errors.New("database error"))

		err := repo.Delete(ctx, databaseID)
		require.Error(t, err)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
