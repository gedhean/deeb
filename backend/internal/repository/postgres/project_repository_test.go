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

func setupProjectMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestProjectRepository_Create(t *testing.T) {
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
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, project.ID)
		assert.False(t, project.CreatedAt.IsZero())
		assert.False(t, project.UpdatedAt.IsZero())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with nil description", func(t *testing.T) {
		project := &models.Project{
			Name:        "Test Project",
			Description: nil,
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("INSERT INTO projects").
			WithArgs(sqlmock.AnyArg(), project.Name, nil, project.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, project)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, project.ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with existing ID", func(t *testing.T) {
		existingID := uuid.New()
		project := &models.Project{
			ID:          existingID,
			Name:        "Test Project",
			Description: stringPtr("Test Description"),
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("INSERT INTO projects").
			WithArgs(existingID, project.Name, project.Description, project.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, project)
		require.NoError(t, err)
		assert.Equal(t, existingID, project.ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		project := &models.Project{
			Name:        "",
			Description: stringPtr("Test Description"),
			CreatedByID: uuid.New(),
		}

		err := repo.Create(ctx, project)
		assert.Equal(t, models.ErrProjectNameRequired, err)
	})

	t.Run("validation error - name too long", func(t *testing.T) {
		longName := string(make([]byte, 101))
		project := &models.Project{
			Name:        longName,
			Description: stringPtr("Test Description"),
			CreatedByID: uuid.New(),
		}

		err := repo.Create(ctx, project)
		assert.Equal(t, models.ErrProjectNameTooLong, err)
	})

	t.Run("validation error - description too long", func(t *testing.T) {
		longDesc := string(make([]byte, 501))
		project := &models.Project{
			Name:        "Test Project",
			Description: &longDesc,
			CreatedByID: uuid.New(),
		}

		err := repo.Create(ctx, project)
		assert.Equal(t, models.ErrDescriptionTooLong, err)
	})

	t.Run("database error", func(t *testing.T) {
		project := &models.Project{
			Name:        "Test Project",
			Description: stringPtr("Test Description"),
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("INSERT INTO projects").
			WithArgs(sqlmock.AnyArg(), project.Name, project.Description, project.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("database connection error"))

		err := repo.Create(ctx, project)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProjectRepository_GetByID(t *testing.T) {
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
		require.NoError(t, err)
		require.NotNil(t, project)
		assert.Equal(t, projectID, project.ID)
		assert.Equal(t, "Test Project", project.Name)
		assert.NotNil(t, project.Description)
		assert.Equal(t, "Test Description", *project.Description)
		assert.Equal(t, createdByID, project.CreatedByID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful retrieval with nil description", func(t *testing.T) {
		projectID := uuid.New()
		createdByID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{"id", "name", "description", "created_by_id", "created_at", "updated_at"}).
			AddRow(projectID, "Test Project", nil, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM projects WHERE id =").
			WithArgs(projectID).
			WillReturnRows(rows)

		project, err := repo.GetByID(ctx, projectID)
		require.NoError(t, err)
		require.NotNil(t, project)
		assert.Equal(t, projectID, project.ID)
		assert.Nil(t, project.Description)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("project not found", func(t *testing.T) {
		projectID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM projects WHERE id =").
			WithArgs(projectID).
			WillReturnError(sql.ErrNoRows)

		project, err := repo.GetByID(ctx, projectID)
		assert.Equal(t, repository.ErrNotFound, err)
		assert.Nil(t, project)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		projectID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM projects WHERE id =").
			WithArgs(projectID).
			WillReturnError(errors.New("database connection error"))

		project, err := repo.GetByID(ctx, projectID)
		require.Error(t, err)
		assert.Nil(t, project)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProjectRepository_List(t *testing.T) {
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
		require.NoError(t, err)
		require.Len(t, projects, 2)
		assert.Equal(t, projectID1, projects[0].ID)
		assert.Equal(t, "Project 1", projects[0].Name)
		assert.Equal(t, projectID2, projects[1].ID)
		assert.Equal(t, "Project 2", projects[1].Name)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "description", "created_by_id", "created_at", "updated_at"})

		mock.ExpectQuery("SELECT (.+) FROM projects ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		projects, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		assert.NotNil(t, projects)
		assert.Len(t, projects, 0)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("list with pagination", func(t *testing.T) {
		now := time.Now()
		projectID := uuid.New()
		createdByID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "description", "created_by_id", "created_at", "updated_at"}).
			AddRow(projectID, "Project", "Description", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM projects ORDER BY created_at DESC LIMIT").
			WithArgs(5, 10).
			WillReturnRows(rows)

		projects, err := repo.List(ctx, 5, 10)
		require.NoError(t, err)
		assert.Len(t, projects, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM projects ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnError(errors.New("database connection error"))

		projects, err := repo.List(ctx, 10, 0)
		require.Error(t, err)
		assert.Nil(t, projects)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProjectRepository_Update(t *testing.T) {
	db, mock := setupProjectMockDB(t)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		projectID := uuid.New()
		beforeUpdate := time.Now()
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
		require.NoError(t, err)
		assert.False(t, project.UpdatedAt.IsZero())
		assert.True(t, project.UpdatedAt.After(beforeUpdate))

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful update with nil description", func(t *testing.T) {
		projectID := uuid.New()
		project := &models.Project{
			ID:          projectID,
			Name:        "Updated Project",
			Description: nil,
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("UPDATE projects").
			WithArgs(projectID, project.Name, nil, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, project)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
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
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		project := &models.Project{
			ID:          uuid.New(),
			Name:        "",
			Description: stringPtr("Updated Description"),
			CreatedByID: uuid.New(),
		}

		err := repo.Update(ctx, project)
		assert.Equal(t, models.ErrProjectNameRequired, err)
	})

	t.Run("validation error - name too long", func(t *testing.T) {
		longName := string(make([]byte, 101))
		project := &models.Project{
			ID:          uuid.New(),
			Name:        longName,
			Description: stringPtr("Updated Description"),
			CreatedByID: uuid.New(),
		}

		err := repo.Update(ctx, project)
		assert.Equal(t, models.ErrProjectNameTooLong, err)
	})

	t.Run("validation error - description too long", func(t *testing.T) {
		longDesc := string(make([]byte, 501))
		project := &models.Project{
			ID:          uuid.New(),
			Name:        "Test Project",
			Description: &longDesc,
			CreatedByID: uuid.New(),
		}

		err := repo.Update(ctx, project)
		assert.Equal(t, models.ErrDescriptionTooLong, err)
	})

	t.Run("database error", func(t *testing.T) {
		projectID := uuid.New()
		project := &models.Project{
			ID:          projectID,
			Name:        "Updated Project",
			Description: stringPtr("Updated Description"),
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("UPDATE projects").
			WithArgs(projectID, project.Name, project.Description, sqlmock.AnyArg()).
			WillReturnError(errors.New("database connection error"))

		err := repo.Update(ctx, project)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProjectRepository_Delete(t *testing.T) {
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
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("project not found", func(t *testing.T) {
		projectID := uuid.New()

		mock.ExpectExec("DELETE FROM projects WHERE id").
			WithArgs(projectID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(ctx, projectID)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		projectID := uuid.New()

		mock.ExpectExec("DELETE FROM projects WHERE id").
			WithArgs(projectID).
			WillReturnError(errors.New("database connection error"))

		err := repo.Delete(ctx, projectID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProjectRepository_Count(t *testing.T) {
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
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count zero", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(0)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM projects").
			WillReturnRows(rows)

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count large number", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(1000)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM projects").
			WillReturnRows(rows)

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1000), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM projects").
			WillReturnError(errors.New("database connection error"))

		count, err := repo.Count(ctx)
		require.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func stringPtr(s string) *string {
	return &s
}
