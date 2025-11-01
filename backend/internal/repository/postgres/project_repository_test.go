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

func setupProjectMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestCreate(t *testing.T) {
	db, mock := setupProjectMockDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		project := &models.Project{
			Name:        "Test Project",
			Description: stringPtr("Test Description"),
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("INSERT INTO projects").
			WithArgs(sqlmock.AnyArg(), project.Name, project.Description, project.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, project)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify ID and timestamps were set
		if project.ID == uuid.Nil {
			t.Error("Expected ID to be set")
		}
		if project.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
		if project.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		project := &models.Project{
			Name:        "",
			Description: stringPtr("Test Description"),
			CreatedByID: uuid.New(),
		}

		err := repo.Create(ctx, project)
		if err != models.ErrProjectNameRequired {
			t.Errorf("Expected ErrProjectNameRequired, got %v", err)
		}
	})

	t.Run("validation error - name too long", func(t *testing.T) {
		longName := string(make([]byte, 101))
		project := &models.Project{
			Name:        longName,
			Description: stringPtr("Test Description"),
			CreatedByID: uuid.New(),
		}

		err := repo.Create(ctx, project)
		if err != models.ErrProjectNameTooLong {
			t.Errorf("Expected ErrProjectNameTooLong, got %v", err)
		}
	})

	t.Run("validation error - description too long", func(t *testing.T) {
		longDesc := string(make([]byte, 501))
		project := &models.Project{
			Name:        "Test Project",
			Description: &longDesc,
			CreatedByID: uuid.New(),
		}

		err := repo.Create(ctx, project)
		if err != models.ErrDescriptionTooLong {
			t.Errorf("Expected ErrDescriptionTooLong, got %v", err)
		}
	})
}

func TestGetByID(t *testing.T) {
	db, mock := setupProjectMockDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		projectID := uuid.New()
		createdByID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{"id", "name", "description", "created_by_id", "created_at", "updated_at"}).
			AddRow(projectID, "Test Project", "Test Description", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM projects WHERE id =").
			WithArgs(projectID).
			WillReturnRows(rows)

		project, err := repo.GetByID(ctx, projectID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if project == nil {
			t.Fatal("Expected project to be returned")
		}

		if project.ID != projectID {
			t.Errorf("Expected ID %s, got %s", projectID, project.ID)
		}

		if project.Name != "Test Project" {
			t.Errorf("Expected name 'Test Project', got %s", project.Name)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("project not found", func(t *testing.T) {
		projectID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM projects WHERE id =").
			WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "created_by_id", "created_at", "updated_at"}))

		project, err := repo.GetByID(ctx, projectID)
		if err != repository.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}

		if project != nil {
			t.Error("Expected nil project")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestList(t *testing.T) {
	db, mock := setupProjectMockDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	t.Run("successful list with results", func(t *testing.T) {
		now := time.Now()
		projectID1 := uuid.New()
		projectID2 := uuid.New()
		createdByID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "description", "created_by_id", "created_at", "updated_at"}).
			AddRow(projectID1, "Project 1", "Description 1", createdByID, now, now).
			AddRow(projectID2, "Project 2", "Description 2", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM projects ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		projects, err := repo.List(ctx, 10, 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(projects) != 2 {
			t.Errorf("Expected 2 projects, got %d", len(projects))
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "description", "created_by_id", "created_at", "updated_at"})

		mock.ExpectQuery("SELECT (.+) FROM projects ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		projects, err := repo.List(ctx, 10, 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(projects) != 0 {
			t.Errorf("Expected 0 projects, got %d", len(projects))
		}

		if projects == nil {
			t.Error("Expected empty slice, got nil")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestUpdate(t *testing.T) {
	db, mock := setupProjectMockDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		projectID := uuid.New()
		project := &models.Project{
			ID:          projectID,
			Name:        "Updated Project",
			Description: stringPtr("Updated Description"),
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("UPDATE projects").
			WithArgs(projectID, project.Name, project.Description, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, project)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify UpdatedAt was set
		if project.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("project not found", func(t *testing.T) {
		projectID := uuid.New()
		project := &models.Project{
			ID:          projectID,
			Name:        "Updated Project",
			Description: stringPtr("Updated Description"),
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("UPDATE projects").
			WithArgs(projectID, project.Name, project.Description, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(ctx, project)
		if err != repository.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		project := &models.Project{
			ID:          uuid.New(),
			Name:        "",
			Description: stringPtr("Updated Description"),
			CreatedByID: uuid.New(),
		}

		err := repo.Update(ctx, project)
		if err != models.ErrProjectNameRequired {
			t.Errorf("Expected ErrProjectNameRequired, got %v", err)
		}
	})
}

func TestDelete(t *testing.T) {
	db, mock := setupProjectMockDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		projectID := uuid.New()

		mock.ExpectExec("DELETE FROM projects WHERE id").
			WithArgs(projectID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(ctx, projectID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("project not found", func(t *testing.T) {
		projectID := uuid.New()

		mock.ExpectExec("DELETE FROM projects WHERE id").
			WithArgs(projectID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(ctx, projectID)
		if err != repository.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestCount(t *testing.T) {
	db, mock := setupProjectMockDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	t.Run("successful count", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(5)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM projects").
			WillReturnRows(rows)

		count, err := repo.Count(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if count != 5 {
			t.Errorf("Expected count 5, got %d", count)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("count zero", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(0)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM projects").
			WillReturnRows(rows)

		count, err := repo.Count(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if count != 0 {
			t.Errorf("Expected count 0, got %d", count)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func stringPtr(s string) *string {
	return &s
}
