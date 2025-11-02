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

func setupEnvironmentMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestEnvironmentRepository_Create(t *testing.T) {
	db, mock := setupEnvironmentMockDB(t)
	defer db.Close()

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		policyID := uuid.New()
		environment := &models.Environment{
			Name:        "Test Environment",
			Description: stringPtr("Test Description"),
			PolicyID:    &policyID,
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("INSERT INTO environments").
			WithArgs(sqlmock.AnyArg(), environment.Name, environment.Description, environment.PolicyID, environment.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, environment)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, environment.ID)
		assert.False(t, environment.CreatedAt.IsZero())
		assert.False(t, environment.UpdatedAt.IsZero())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with nil description and nil policy", func(t *testing.T) {
		environment := &models.Environment{
			Name:        "Test Environment",
			Description: nil,
			PolicyID:    nil,
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("INSERT INTO environments").
			WithArgs(sqlmock.AnyArg(), environment.Name, nil, nil, environment.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, environment)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, environment.ID)
		assert.Nil(t, environment.PolicyID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with existing ID", func(t *testing.T) {
		existingID := uuid.New()
		policyID := uuid.New()
		environment := &models.Environment{
			ID:          existingID,
			Name:        "Test Environment",
			Description: stringPtr("Test Description"),
			PolicyID:    &policyID,
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("INSERT INTO environments").
			WithArgs(existingID, environment.Name, environment.Description, environment.PolicyID, environment.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, environment)
		require.NoError(t, err)
		assert.Equal(t, existingID, environment.ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		policyID := uuid.New()
		environment := &models.Environment{
			Name:        "Test Environment",
			Description: stringPtr("Test Description"),
			PolicyID:    &policyID,
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("INSERT INTO environments").
			WithArgs(sqlmock.AnyArg(), environment.Name, environment.Description, environment.PolicyID, environment.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("database connection error"))

		err := repo.Create(ctx, environment)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEnvironmentRepository_GetByID(t *testing.T) {
	db, mock := setupEnvironmentMockDB(t)
	defer db.Close()

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		environmentID := uuid.New()
		policyID := uuid.New()
		createdByID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{"id", "name", "description", "policy_id", "created_by_id", "created_at", "updated_at"}).
			AddRow(environmentID, "Test Environment", "Test Description", policyID, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM environments WHERE id =").
			WithArgs(environmentID).
			WillReturnRows(rows)

		environment, err := repo.GetByID(ctx, environmentID)
		require.NoError(t, err)
		require.NotNil(t, environment)
		assert.Equal(t, environmentID, environment.ID)
		assert.Equal(t, "Test Environment", environment.Name)
		assert.NotNil(t, environment.Description)
		assert.Equal(t, "Test Description", *environment.Description)
		assert.NotNil(t, environment.PolicyID)
		assert.Equal(t, policyID, *environment.PolicyID)
		assert.Equal(t, createdByID, environment.CreatedByID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful retrieval with nil description and nil policy", func(t *testing.T) {
		environmentID := uuid.New()
		createdByID := uuid.New()
		now := time.Now()

		rows := sqlmock.NewRows([]string{"id", "name", "description", "policy_id", "created_by_id", "created_at", "updated_at"}).
			AddRow(environmentID, "Test Environment", nil, nil, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM environments WHERE id =").
			WithArgs(environmentID).
			WillReturnRows(rows)

		environment, err := repo.GetByID(ctx, environmentID)
		require.NoError(t, err)
		require.NotNil(t, environment)
		assert.Equal(t, environmentID, environment.ID)
		assert.Nil(t, environment.Description)
		assert.Nil(t, environment.PolicyID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("environment not found", func(t *testing.T) {
		environmentID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM environments WHERE id =").
			WithArgs(environmentID).
			WillReturnError(sql.ErrNoRows)

		environment, err := repo.GetByID(ctx, environmentID)
		assert.Equal(t, repository.ErrNotFound, err)
		assert.Nil(t, environment)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		environmentID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM environments WHERE id =").
			WithArgs(environmentID).
			WillReturnError(errors.New("database connection error"))

		environment, err := repo.GetByID(ctx, environmentID)
		require.Error(t, err)
		assert.Nil(t, environment)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEnvironmentRepository_List(t *testing.T) {
	db, mock := setupEnvironmentMockDB(t)
	defer db.Close()

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("successful list with results", func(t *testing.T) {
		now := time.Now()
		envID1 := uuid.New()
		envID2 := uuid.New()
		policyID := uuid.New()
		createdByID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "description", "policy_id", "created_by_id", "created_at", "updated_at"}).
			AddRow(envID1, "Environment 1", "Description 1", policyID, createdByID, now, now).
			AddRow(envID2, "Environment 2", "Description 2", nil, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM environments ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		environments, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		require.Len(t, environments, 2)
		assert.Equal(t, envID1, environments[0].ID)
		assert.Equal(t, "Environment 1", environments[0].Name)
		assert.NotNil(t, environments[0].PolicyID)
		assert.Equal(t, envID2, environments[1].ID)
		assert.Equal(t, "Environment 2", environments[1].Name)
		assert.Nil(t, environments[1].PolicyID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "description", "policy_id", "created_by_id", "created_at", "updated_at"})

		mock.ExpectQuery("SELECT (.+) FROM environments ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		environments, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		assert.NotNil(t, environments)
		assert.Len(t, environments, 0)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("list with pagination", func(t *testing.T) {
		now := time.Now()
		envID := uuid.New()
		policyID := uuid.New()
		createdByID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "description", "policy_id", "created_by_id", "created_at", "updated_at"}).
			AddRow(envID, "Environment", "Description", policyID, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM environments ORDER BY created_at DESC LIMIT").
			WithArgs(5, 10).
			WillReturnRows(rows)

		environments, err := repo.List(ctx, 5, 10)
		require.NoError(t, err)
		assert.Len(t, environments, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM environments ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnError(errors.New("database connection error"))

		environments, err := repo.List(ctx, 10, 0)
		require.Error(t, err)
		assert.Nil(t, environments)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEnvironmentRepository_Update(t *testing.T) {
	db, mock := setupEnvironmentMockDB(t)
	defer db.Close()

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		environmentID := uuid.New()
		policyID := uuid.New()
		beforeUpdate := time.Now()
		environment := &models.Environment{
			ID:          environmentID,
			Name:        "Updated Environment",
			Description: stringPtr("Updated Description"),
			PolicyID:    &policyID,
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("UPDATE environments").
			WithArgs(environmentID, environment.Name, environment.Description, environment.PolicyID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, environment)
		require.NoError(t, err)
		assert.False(t, environment.UpdatedAt.IsZero())
		assert.True(t, environment.UpdatedAt.After(beforeUpdate))

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful update with nil description and nil policy", func(t *testing.T) {
		environmentID := uuid.New()
		environment := &models.Environment{
			ID:          environmentID,
			Name:        "Updated Environment",
			Description: nil,
			PolicyID:    nil,
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("UPDATE environments").
			WithArgs(environmentID, environment.Name, nil, nil, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, environment)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update policy ID to nil", func(t *testing.T) {
		environmentID := uuid.New()
		environment := &models.Environment{
			ID:          environmentID,
			Name:        "Updated Environment",
			Description: stringPtr("Description"),
			PolicyID:    nil, // Updating policy ID from non-nil to nil
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("UPDATE environments").
			WithArgs(environmentID, environment.Name, environment.Description, nil, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, environment)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update policy ID from nil to value", func(t *testing.T) {
		environmentID := uuid.New()
		policyID := uuid.New()
		environment := &models.Environment{
			ID:          environmentID,
			Name:        "Updated Environment",
			Description: stringPtr("Description"),
			PolicyID:    &policyID, // Updating policy ID from nil to value
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("UPDATE environments").
			WithArgs(environmentID, environment.Name, environment.Description, &policyID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, environment)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("environment not found", func(t *testing.T) {
		environmentID := uuid.New()
		policyID := uuid.New()
		environment := &models.Environment{
			ID:          environmentID,
			Name:        "Updated Environment",
			Description: stringPtr("Updated Description"),
			PolicyID:    &policyID,
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("UPDATE environments").
			WithArgs(environmentID, environment.Name, environment.Description, environment.PolicyID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(ctx, environment)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		environmentID := uuid.New()
		policyID := uuid.New()
		environment := &models.Environment{
			ID:          environmentID,
			Name:        "Updated Environment",
			Description: stringPtr("Updated Description"),
			PolicyID:    &policyID,
			CreatedByID: uuid.New(),
		}

		mock.ExpectExec("UPDATE environments").
			WithArgs(environmentID, environment.Name, environment.Description, environment.PolicyID, sqlmock.AnyArg()).
			WillReturnError(errors.New("database connection error"))

		err := repo.Update(ctx, environment)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEnvironmentRepository_Delete(t *testing.T) {
	db, mock := setupEnvironmentMockDB(t)
	defer db.Close()

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		environmentID := uuid.New()

		mock.ExpectExec("DELETE FROM environments WHERE id").
			WithArgs(environmentID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(ctx, environmentID)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("environment not found", func(t *testing.T) {
		environmentID := uuid.New()

		mock.ExpectExec("DELETE FROM environments WHERE id").
			WithArgs(environmentID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(ctx, environmentID)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		environmentID := uuid.New()

		mock.ExpectExec("DELETE FROM environments WHERE id").
			WithArgs(environmentID).
			WillReturnError(errors.New("database connection error"))

		err := repo.Delete(ctx, environmentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEnvironmentRepository_Count(t *testing.T) {
	db, mock := setupEnvironmentMockDB(t)
	defer db.Close()

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("successful count", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(5)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM environments").
			WillReturnRows(rows)

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count zero", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(0)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM environments").
			WillReturnRows(rows)

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count large number", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(1000)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM environments").
			WillReturnRows(rows)

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1000), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM environments").
			WillReturnError(errors.New("database connection error"))

		count, err := repo.Count(ctx)
		require.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
