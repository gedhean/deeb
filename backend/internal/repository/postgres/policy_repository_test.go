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

func setupPolicyMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestPolicyRepository_Create(t *testing.T) {
	db, mock := setupPolicyMockDB(t)
	defer db.Close()

	repo := NewPolicyRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		policyConfig := models.PolicyConfig{
			"max_query_time": 30,
			"require_index":  true,
		}

		policy := &models.Policy{
			Name:         "Test Policy",
			Description:  stringPtr("Test Description"),
			Type:         models.PolicyTypeSQLReview,
			PolicyConfig: policyConfig,
			CreatedByID:  uuid.New(),
		}

		mock.ExpectExec("INSERT INTO policies").
			WithArgs(sqlmock.AnyArg(), policy.Name, policy.Description, policy.Type, sqlmock.AnyArg(), policy.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, policy)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, policy.ID)
		assert.False(t, policy.CreatedAt.IsZero())
		assert.False(t, policy.UpdatedAt.IsZero())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with nil description", func(t *testing.T) {
		policyConfig := models.PolicyConfig{
			"max_query_time": 30,
		}

		policy := &models.Policy{
			Name:         "Test Policy",
			Description:  nil,
			Type:         models.PolicyTypeSQLReview,
			PolicyConfig: policyConfig,
			CreatedByID:  uuid.New(),
		}

		mock.ExpectExec("INSERT INTO policies").
			WithArgs(sqlmock.AnyArg(), policy.Name, nil, policy.Type, sqlmock.AnyArg(), policy.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, policy)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, policy.ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with existing ID", func(t *testing.T) {
		existingID := uuid.New()
		policyConfig := models.PolicyConfig{
			"max_query_time": 30,
		}

		policy := &models.Policy{
			ID:           existingID,
			Name:         "Test Policy",
			Description:  stringPtr("Test Description"),
			Type:         models.PolicyTypeSQLReview,
			PolicyConfig: policyConfig,
			CreatedByID:  uuid.New(),
		}

		mock.ExpectExec("INSERT INTO policies").
			WithArgs(existingID, policy.Name, policy.Description, policy.Type, sqlmock.AnyArg(), policy.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, policy)
		require.NoError(t, err)
		assert.Equal(t, existingID, policy.ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with empty policy config", func(t *testing.T) {
		policyConfig := models.PolicyConfig{}

		policy := &models.Policy{
			Name:         "Test Policy",
			Description:  stringPtr("Test Description"),
			Type:         models.PolicyTypeSQLReview,
			PolicyConfig: policyConfig,
			CreatedByID:  uuid.New(),
		}

		mock.ExpectExec("INSERT INTO policies").
			WithArgs(sqlmock.AnyArg(), policy.Name, policy.Description, policy.Type, sqlmock.AnyArg(), policy.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, policy)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		policyConfig := models.PolicyConfig{
			"max_query_time": 30,
		}

		policy := &models.Policy{
			Name:         "Test Policy",
			Description:  stringPtr("Test Description"),
			Type:         models.PolicyTypeSQLReview,
			PolicyConfig: policyConfig,
			CreatedByID:  uuid.New(),
		}

		mock.ExpectExec("INSERT INTO policies").
			WithArgs(sqlmock.AnyArg(), policy.Name, policy.Description, policy.Type, sqlmock.AnyArg(), policy.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("database connection error"))

		err := repo.Create(ctx, policy)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPolicyRepository_GetByID(t *testing.T) {
	db, mock := setupPolicyMockDB(t)
	defer db.Close()

	repo := NewPolicyRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		policyID := uuid.New()
		createdByID := uuid.New()
		now := time.Now()
		policyConfigJSON := `{"max_query_time": 30, "require_index": true}`

		rows := sqlmock.NewRows([]string{"id", "name", "description", "type", "policy_config", "created_by_id", "created_at", "updated_at"}).
			AddRow(policyID, "Test Policy", "Test Description", models.PolicyTypeSQLReview, policyConfigJSON, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM policies WHERE id =").
			WithArgs(policyID).
			WillReturnRows(rows)

		policy, err := repo.GetByID(ctx, policyID)
		require.NoError(t, err)
		require.NotNil(t, policy)
		assert.Equal(t, policyID, policy.ID)
		assert.Equal(t, "Test Policy", policy.Name)
		assert.NotNil(t, policy.Description)
		assert.Equal(t, "Test Description", *policy.Description)
		assert.Equal(t, models.PolicyTypeSQLReview, policy.Type)
		assert.NotNil(t, policy.PolicyConfig)
		assert.Equal(t, createdByID, policy.CreatedByID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful retrieval with nil description", func(t *testing.T) {
		policyID := uuid.New()
		createdByID := uuid.New()
		now := time.Now()
		policyConfigJSON := `{"max_query_time": 30}`

		rows := sqlmock.NewRows([]string{"id", "name", "description", "type", "policy_config", "created_by_id", "created_at", "updated_at"}).
			AddRow(policyID, "Test Policy", nil, models.PolicyTypeSQLReview, policyConfigJSON, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM policies WHERE id =").
			WithArgs(policyID).
			WillReturnRows(rows)

		policy, err := repo.GetByID(ctx, policyID)
		require.NoError(t, err)
		require.NotNil(t, policy)
		assert.Equal(t, policyID, policy.ID)
		assert.Nil(t, policy.Description)
		assert.NotNil(t, policy.PolicyConfig)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful retrieval with empty policy config", func(t *testing.T) {
		policyID := uuid.New()
		createdByID := uuid.New()
		now := time.Now()
		policyConfigJSON := `{}`

		rows := sqlmock.NewRows([]string{"id", "name", "description", "type", "policy_config", "created_by_id", "created_at", "updated_at"}).
			AddRow(policyID, "Test Policy", "Test Description", models.PolicyTypeSQLReview, policyConfigJSON, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM policies WHERE id =").
			WithArgs(policyID).
			WillReturnRows(rows)

		policy, err := repo.GetByID(ctx, policyID)
		require.NoError(t, err)
		require.NotNil(t, policy)
		assert.NotNil(t, policy.PolicyConfig)
		assert.Len(t, policy.PolicyConfig, 0)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("policy not found", func(t *testing.T) {
		policyID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM policies WHERE id =").
			WithArgs(policyID).
			WillReturnError(sql.ErrNoRows)

		policy, err := repo.GetByID(ctx, policyID)
		assert.Equal(t, repository.ErrNotFound, err)
		assert.Nil(t, policy)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		policyID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM policies WHERE id =").
			WithArgs(policyID).
			WillReturnError(errors.New("database connection error"))

		policy, err := repo.GetByID(ctx, policyID)
		require.Error(t, err)
		assert.Nil(t, policy)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPolicyRepository_List(t *testing.T) {
	db, mock := setupPolicyMockDB(t)
	defer db.Close()

	repo := NewPolicyRepository(db)
	ctx := context.Background()

	t.Run("successful list with results", func(t *testing.T) {
		now := time.Now()
		policyID1 := uuid.New()
		policyID2 := uuid.New()
		createdByID := uuid.New()
		policyConfig1JSON := `{"max_query_time": 30}`
		policyConfig2JSON := `{"require_index": true}`

		rows := sqlmock.NewRows([]string{"id", "name", "description", "type", "policy_config", "created_by_id", "created_at", "updated_at"}).
			AddRow(policyID1, "Policy 1", "Description 1", models.PolicyTypeSQLReview, policyConfig1JSON, createdByID, now, now).
			AddRow(policyID2, "Policy 2", "Description 2", models.PolicyTypeSQLReview, policyConfig2JSON, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM policies ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		policies, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		require.Len(t, policies, 2)
		assert.Equal(t, policyID1, policies[0].ID)
		assert.Equal(t, "Policy 1", policies[0].Name)
		assert.Equal(t, policyID2, policies[1].ID)
		assert.Equal(t, "Policy 2", policies[1].Name)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "description", "type", "policy_config", "created_by_id", "created_at", "updated_at"})

		mock.ExpectQuery("SELECT (.+) FROM policies ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		policies, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		assert.NotNil(t, policies)
		assert.Len(t, policies, 0)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("list with pagination", func(t *testing.T) {
		now := time.Now()
		policyID := uuid.New()
		createdByID := uuid.New()
		policyConfigJSON := `{"max_query_time": 30}`

		rows := sqlmock.NewRows([]string{"id", "name", "description", "type", "policy_config", "created_by_id", "created_at", "updated_at"}).
			AddRow(policyID, "Policy", "Description", models.PolicyTypeSQLReview, policyConfigJSON, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM policies ORDER BY created_at DESC LIMIT").
			WithArgs(5, 10).
			WillReturnRows(rows)

		policies, err := repo.List(ctx, 5, 10)
		require.NoError(t, err)
		assert.Len(t, policies, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM policies ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnError(errors.New("database connection error"))

		policies, err := repo.List(ctx, 10, 0)
		require.Error(t, err)
		assert.Nil(t, policies)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPolicyRepository_GetByType(t *testing.T) {
	db, mock := setupPolicyMockDB(t)
	defer db.Close()

	repo := NewPolicyRepository(db)
	ctx := context.Background()

	t.Run("successful retrieval by type", func(t *testing.T) {
		now := time.Now()
		policyID1 := uuid.New()
		policyID2 := uuid.New()
		createdByID := uuid.New()
		policyConfigJSON := `{"max_query_time": 30}`

		rows := sqlmock.NewRows([]string{"id", "name", "description", "type", "policy_config", "created_by_id", "created_at", "updated_at"}).
			AddRow(policyID1, "Policy 1", "Description 1", models.PolicyTypeSQLReview, policyConfigJSON, createdByID, now, now).
			AddRow(policyID2, "Policy 2", "Description 2", models.PolicyTypeSQLReview, policyConfigJSON, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM policies WHERE type =").
			WithArgs(models.PolicyTypeSQLReview, 10, 0).
			WillReturnRows(rows)

		policies, err := repo.GetByType(ctx, models.PolicyTypeSQLReview, 10, 0)
		require.NoError(t, err)
		require.Len(t, policies, 2)
		assert.Equal(t, policyID1, policies[0].ID)
		assert.Equal(t, policyID2, policies[1].ID)
		for _, policy := range policies {
			assert.Equal(t, models.PolicyTypeSQLReview, policy.Type)
		}

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list by type", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "description", "type", "policy_config", "created_by_id", "created_at", "updated_at"})

		mock.ExpectQuery("SELECT (.+) FROM policies WHERE type =").
			WithArgs(models.PolicyTypeSQLReview, 10, 0).
			WillReturnRows(rows)

		policies, err := repo.GetByType(ctx, models.PolicyTypeSQLReview, 10, 0)
		require.NoError(t, err)
		assert.NotNil(t, policies)
		assert.Len(t, policies, 0)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("get by type with pagination", func(t *testing.T) {
		now := time.Now()
		policyID := uuid.New()
		createdByID := uuid.New()
		policyConfigJSON := `{"max_query_time": 30}`

		rows := sqlmock.NewRows([]string{"id", "name", "description", "type", "policy_config", "created_by_id", "created_at", "updated_at"}).
			AddRow(policyID, "Policy", "Description", models.PolicyTypeSQLReview, policyConfigJSON, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM policies WHERE type =").
			WithArgs(models.PolicyTypeSQLReview, 5, 10).
			WillReturnRows(rows)

		policies, err := repo.GetByType(ctx, models.PolicyTypeSQLReview, 5, 10)
		require.NoError(t, err)
		assert.Len(t, policies, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM policies WHERE type =").
			WithArgs(models.PolicyTypeSQLReview, 10, 0).
			WillReturnError(errors.New("database connection error"))

		policies, err := repo.GetByType(ctx, models.PolicyTypeSQLReview, 10, 0)
		require.Error(t, err)
		assert.Nil(t, policies)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPolicyRepository_CountByType(t *testing.T) {
	db, mock := setupPolicyMockDB(t)
	defer db.Close()

	repo := NewPolicyRepository(db)
	ctx := context.Background()

	t.Run("successful count by type", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(5)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM policies WHERE type =").
			WithArgs(models.PolicyTypeSQLReview).
			WillReturnRows(rows)

		count, err := repo.CountByType(ctx, models.PolicyTypeSQLReview)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count zero by type", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(0)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM policies WHERE type =").
			WithArgs(models.PolicyTypeSQLReview).
			WillReturnRows(rows)

		count, err := repo.CountByType(ctx, models.PolicyTypeSQLReview)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count large number by type", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(1000)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM policies WHERE type =").
			WithArgs(models.PolicyTypeSQLReview).
			WillReturnRows(rows)

		count, err := repo.CountByType(ctx, models.PolicyTypeSQLReview)
		require.NoError(t, err)
		assert.Equal(t, int64(1000), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM policies WHERE type =").
			WithArgs(models.PolicyTypeSQLReview).
			WillReturnError(errors.New("database connection error"))

		count, err := repo.CountByType(ctx, models.PolicyTypeSQLReview)
		require.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPolicyRepository_Update(t *testing.T) {
	db, mock := setupPolicyMockDB(t)
	defer db.Close()

	repo := NewPolicyRepository(db)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		policyID := uuid.New()
		beforeUpdate := time.Now()
		policyConfig := models.PolicyConfig{
			"max_query_time": 60,
		}

		policy := &models.Policy{
			ID:           policyID,
			Name:         "Updated Policy",
			Description:  stringPtr("Updated Description"),
			Type:         models.PolicyTypeSQLReview,
			PolicyConfig: policyConfig,
			CreatedByID:  uuid.New(),
		}

		mock.ExpectExec("UPDATE policies").
			WithArgs(policyID, policy.Name, policy.Description, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, policy)
		require.NoError(t, err)
		assert.False(t, policy.UpdatedAt.IsZero())
		assert.True(t, policy.UpdatedAt.After(beforeUpdate))

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful update with nil description", func(t *testing.T) {
		policyID := uuid.New()
		policyConfig := models.PolicyConfig{
			"max_query_time": 60,
		}

		policy := &models.Policy{
			ID:           policyID,
			Name:         "Updated Policy",
			Description:  nil,
			Type:         models.PolicyTypeSQLReview,
			PolicyConfig: policyConfig,
			CreatedByID:  uuid.New(),
		}

		mock.ExpectExec("UPDATE policies").
			WithArgs(policyID, policy.Name, nil, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, policy)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful update with empty policy config", func(t *testing.T) {
		policyID := uuid.New()
		policyConfig := models.PolicyConfig{}

		policy := &models.Policy{
			ID:           policyID,
			Name:         "Updated Policy",
			Description:  stringPtr("Updated Description"),
			Type:         models.PolicyTypeSQLReview,
			PolicyConfig: policyConfig,
			CreatedByID:  uuid.New(),
		}

		mock.ExpectExec("UPDATE policies").
			WithArgs(policyID, policy.Name, policy.Description, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, policy)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("policy not found", func(t *testing.T) {
		policyID := uuid.New()
		policyConfig := models.PolicyConfig{
			"max_query_time": 60,
		}

		policy := &models.Policy{
			ID:           policyID,
			Name:         "Updated Policy",
			Description:  stringPtr("Updated Description"),
			Type:         models.PolicyTypeSQLReview,
			PolicyConfig: policyConfig,
			CreatedByID:  uuid.New(),
		}

		mock.ExpectExec("UPDATE policies").
			WithArgs(policyID, policy.Name, policy.Description, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(ctx, policy)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		policyID := uuid.New()
		policyConfig := models.PolicyConfig{
			"max_query_time": 60,
		}

		policy := &models.Policy{
			ID:           policyID,
			Name:         "Updated Policy",
			Description:  stringPtr("Updated Description"),
			Type:         models.PolicyTypeSQLReview,
			PolicyConfig: policyConfig,
			CreatedByID:  uuid.New(),
		}

		mock.ExpectExec("UPDATE policies").
			WithArgs(policyID, policy.Name, policy.Description, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("database connection error"))

		err := repo.Update(ctx, policy)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPolicyRepository_Delete(t *testing.T) {
	db, mock := setupPolicyMockDB(t)
	defer db.Close()

	repo := NewPolicyRepository(db)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		policyID := uuid.New()

		mock.ExpectExec("DELETE FROM policies WHERE id").
			WithArgs(policyID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(ctx, policyID)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("policy not found", func(t *testing.T) {
		policyID := uuid.New()

		mock.ExpectExec("DELETE FROM policies WHERE id").
			WithArgs(policyID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(ctx, policyID)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		policyID := uuid.New()

		mock.ExpectExec("DELETE FROM policies WHERE id").
			WithArgs(policyID).
			WillReturnError(errors.New("database connection error"))

		err := repo.Delete(ctx, policyID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPolicyRepository_Count(t *testing.T) {
	db, mock := setupPolicyMockDB(t)
	defer db.Close()

	repo := NewPolicyRepository(db)
	ctx := context.Background()

	t.Run("successful count", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(5)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM policies").
			WillReturnRows(rows)

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count zero", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(0)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM policies").
			WillReturnRows(rows)

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count large number", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(1000)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM policies").
			WillReturnRows(rows)

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1000), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM policies").
			WillReturnError(errors.New("database connection error"))

		count, err := repo.Count(ctx)
		require.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, err.Error(), "database connection error")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
