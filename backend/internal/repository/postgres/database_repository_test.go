package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
)

func setupDatabaseMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestDatabaseCreate(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		database := &models.Database{
			Name:               "test_database",
			DatabaseInstanceID: dbInstanceID,
			ProjectID:          projectID,
			EnvironmentID:      envID,
		}

		mock.ExpectExec("INSERT INTO databases").
			WithArgs(sqlmock.AnyArg(), database.Name, database.DatabaseInstanceID, database.ProjectID, database.EnvironmentID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, database)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify ID and timestamps were set
		if database.ID == uuid.Nil {
			t.Error("Expected ID to be set")
		}
		if database.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
		if database.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		database := &models.Database{
			Name:               "",
			DatabaseInstanceID: uuid.New(),
			ProjectID:          uuid.New(),
			EnvironmentID:      uuid.New(),
		}

		err := repo.Create(ctx, database)
		if err != models.ErrDatabaseNameRequired {
			t.Errorf("Expected ErrDatabaseNameRequired, got %v", err)
		}
	})

	t.Run("validation error - nil database instance ID", func(t *testing.T) {
		database := &models.Database{
			Name:               "test_database",
			DatabaseInstanceID: uuid.Nil,
			ProjectID:          uuid.New(),
			EnvironmentID:      uuid.New(),
		}

		err := repo.Create(ctx, database)
		if err != models.ErrDatabaseInstanceIDRequired {
			t.Errorf("Expected ErrDatabaseInstanceIDRequired, got %v", err)
		}
	})

	t.Run("validation error - nil project ID", func(t *testing.T) {
		database := &models.Database{
			Name:               "test_database",
			DatabaseInstanceID: uuid.New(),
			ProjectID:          uuid.Nil,
			EnvironmentID:      uuid.New(),
		}

		err := repo.Create(ctx, database)
		if err != models.ErrDatabaseProjectIDRequired {
			t.Errorf("Expected ErrDatabaseProjectIDRequired, got %v", err)
		}
	})
}

func TestDatabaseGetByID(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		databaseID := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{"id", "name", "database_instance_id", "project_id", "environment_id", "created_at", "updated_at"}).
			AddRow(databaseID, "test_database", dbInstanceID, projectID, envID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE id =").
			WithArgs(databaseID).
			WillReturnRows(rows)

		database, err := repo.GetByID(ctx, databaseID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if database == nil {
			t.Fatal("Expected database to be returned")
		}

		if database.ID != databaseID {
			t.Errorf("Expected ID %s, got %s", databaseID, database.ID)
		}

		if database.Name != "test_database" {
			t.Errorf("Expected name 'test_database', got %s", database.Name)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("database not found", func(t *testing.T) {
		databaseID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE id =").
			WithArgs(databaseID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "database_instance_id", "project_id", "environment_id", "created_at", "updated_at"}))

		database, err := repo.GetByID(ctx, databaseID)
		if err != repository.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}

		if database != nil {
			t.Error("Expected nil database")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestDatabaseList(t *testing.T) {
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

		rows := sqlmock.NewRows([]string{"id", "name", "database_instance_id", "project_id", "environment_id", "created_at", "updated_at"}).
			AddRow(databaseID1, "database1", dbInstanceID, projectID, envID, now, now).
			AddRow(databaseID2, "database2", dbInstanceID, projectID, envID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM databases ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		databases, err := repo.List(ctx, 10, 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(databases) != 2 {
			t.Errorf("Expected 2 databases, got %d", len(databases))
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		rows := sqlmock.NewRows(
			[]string{"id", "name", "database_instance_id", "project_id", "environment_id", "created_at", "updated_at"},
		)

		mock.ExpectQuery("SELECT (.+) FROM databases ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		databases, err := repo.List(ctx, 10, 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(databases) != 0 {
			t.Errorf("Expected 0 databases, got %d", len(databases))
		}

		if databases == nil {
			t.Error("Expected empty slice, got nil")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestDatabaseListByProject(t *testing.T) {
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

		rows := sqlmock.NewRows([]string{"id", "name", "database_instance_id", "project_id", "environment_id", "created_at", "updated_at"}).
			AddRow(databaseID1, "database1", dbInstanceID, projectID, envID, now, now).
			AddRow(databaseID2, "database2", dbInstanceID, projectID, envID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE project_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(projectID, 10, 0).
			WillReturnRows(rows)

		databases, err := repo.ListByProject(ctx, projectID, 10, 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(databases) != 2 {
			t.Errorf("Expected 2 databases, got %d", len(databases))
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("empty list by project", func(t *testing.T) {
		projectID := uuid.New()
		rows := sqlmock.NewRows(
			[]string{"id", "name", "database_instance_id", "project_id", "environment_id", "created_at", "updated_at"},
		)

		mock.ExpectQuery("SELECT (.+) FROM databases WHERE project_id = (.+) ORDER BY created_at DESC LIMIT").
			WithArgs(projectID, 10, 0).
			WillReturnRows(rows)

		databases, err := repo.ListByProject(ctx, projectID, 10, 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(databases) != 0 {
			t.Errorf("Expected 0 databases, got %d", len(databases))
		}

		if databases == nil {
			t.Error("Expected empty slice, got nil")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestDatabaseUpdate(t *testing.T) {
	db, mock := setupDatabaseMockDB(t)
	defer db.Close()

	repo := NewDatabaseRepository(db)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		databaseID := uuid.New()
		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		database := &models.Database{
			ID:                 databaseID,
			Name:               "updated_database",
			DatabaseInstanceID: dbInstanceID,
			ProjectID:          projectID,
			EnvironmentID:      envID,
		}

		mock.ExpectExec("UPDATE databases").
			WithArgs(databaseID, database.Name, database.DatabaseInstanceID, database.ProjectID, database.EnvironmentID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, database)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify UpdatedAt was set
		if database.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("database not found", func(t *testing.T) {
		databaseID := uuid.New()
		database := &models.Database{
			ID:                 databaseID,
			Name:               "updated_database",
			DatabaseInstanceID: uuid.New(),
			ProjectID:          uuid.New(),
			EnvironmentID:      uuid.New(),
		}

		mock.ExpectExec("UPDATE databases").
			WithArgs(databaseID, database.Name, database.DatabaseInstanceID, database.ProjectID, database.EnvironmentID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(ctx, database)
		if err != repository.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		database := &models.Database{
			ID:                 uuid.New(),
			Name:               "",
			DatabaseInstanceID: uuid.New(),
			ProjectID:          uuid.New(),
			EnvironmentID:      uuid.New(),
		}

		err := repo.Update(ctx, database)
		if err != models.ErrDatabaseNameRequired {
			t.Errorf("Expected ErrDatabaseNameRequired, got %v", err)
		}
	})
}

func TestDatabaseDelete(t *testing.T) {
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
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("database not found", func(t *testing.T) {
		databaseID := uuid.New()

		mock.ExpectExec("DELETE FROM databases WHERE id").
			WithArgs(databaseID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(ctx, databaseID)
		if err != repository.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}
