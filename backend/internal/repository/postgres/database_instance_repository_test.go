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

func setupDatabaseInstanceMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestDatabaseInstanceRepository_Create(t *testing.T) {
	db, mock := setupDatabaseInstanceMockDB(t)
	defer db.Close()

	encryptionKey := "test-encryption-key-32-bytes-long!!"
	repo := NewDatabaseInstanceRepository(db, encryptionKey)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		createdByID := uuid.New()
		description := "Test instance"
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			Name:                       "Test Instance",
			Description:                &description,
			Host:                       "localhost",
			Port:                       5432,
			DatabaseEngine:             models.DatabaseEnginePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			UseRDSIAMAuth:              false,
			RequireSSL:                 true,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                createdByID,
		}

		mock.ExpectExec("INSERT INTO database_instances").
			WithArgs(sqlmock.AnyArg(), instance.Name, instance.Description, instance.DatabaseEngine,
				instance.Host, instance.Port, instance.Username, instance.ConnectionDetailsEncrypted,
				instance.UseRDSIAMAuth, instance.RequireSSL, instance.Status, instance.EnvironmentID,
				instance.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, instance)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, instance.ID)
		assert.False(t, instance.CreatedAt.IsZero())
		assert.False(t, instance.UpdatedAt.IsZero())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with nil description", func(t *testing.T) {
		createdByID := uuid.New()
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			Name:                       "Test Instance",
			Description:                nil,
			Host:                       "localhost",
			Port:                       5432,
			DatabaseEngine:             models.DatabaseEnginePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			UseRDSIAMAuth:              false,
			RequireSSL:                 true,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                createdByID,
		}

		mock.ExpectExec("INSERT INTO database_instances").
			WithArgs(sqlmock.AnyArg(), instance.Name, nil, instance.DatabaseEngine,
				instance.Host, instance.Port, instance.Username, instance.ConnectionDetailsEncrypted,
				instance.UseRDSIAMAuth, instance.RequireSSL, instance.Status, instance.EnvironmentID,
				instance.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, instance)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, instance.ID)
		assert.Nil(t, instance.Description)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful creation with existing ID", func(t *testing.T) {
		existingID := uuid.New()
		createdByID := uuid.New()
		description := "Test instance"
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			ID:                         existingID,
			Name:                       "Test Instance",
			Description:                &description,
			Host:                       "localhost",
			Port:                       5432,
			DatabaseEngine:             models.DatabaseEnginePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			UseRDSIAMAuth:              false,
			RequireSSL:                 true,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                createdByID,
		}

		mock.ExpectExec("INSERT INTO database_instances").
			WithArgs(existingID, instance.Name, instance.Description, instance.DatabaseEngine,
				instance.Host, instance.Port, instance.Username, instance.ConnectionDetailsEncrypted,
				instance.UseRDSIAMAuth, instance.RequireSSL, instance.Status, instance.EnvironmentID,
				instance.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, instance)
		require.NoError(t, err)
		assert.Equal(t, existingID, instance.ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			Name:                       "",
			Host:                       "localhost",
			Port:                       5432,
			DatabaseEngine:             models.DatabaseEnginePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                uuid.New(),
		}

		err := repo.Create(ctx, instance)
		assert.Equal(t, models.ErrDatabaseInstanceNameRequired, err)
	})

	t.Run("validation error - invalid port", func(t *testing.T) {
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			Name:                       "Test Instance",
			Host:                       "localhost",
			Port:                       0,
			DatabaseEngine:             models.DatabaseEnginePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                uuid.New(),
		}

		err := repo.Create(ctx, instance)
		assert.Equal(t, models.ErrPortInvalid, err)
	})

	t.Run("validation error - invalid database type", func(t *testing.T) {
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			Name:                       "Test Instance",
			Host:                       "localhost",
			Port:                       5432,
			DatabaseEngine:             "invalid",
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                uuid.New(),
		}

		err := repo.Create(ctx, instance)
		assert.Equal(t, models.ErrDatabaseEngineInvalid, err)
	})

	t.Run("database error", func(t *testing.T) {
		createdByID := uuid.New()
		description := "Test instance"
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			Name:                       "Test Instance",
			Description:                &description,
			Host:                       "localhost",
			Port:                       5432,
			DatabaseEngine:             models.DatabaseEnginePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			UseRDSIAMAuth:              false,
			RequireSSL:                 true,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                createdByID,
		}

		mock.ExpectExec("INSERT INTO database_instances").
			WithArgs(sqlmock.AnyArg(), instance.Name, instance.Description, instance.DatabaseEngine,
				instance.Host, instance.Port, instance.Username, instance.ConnectionDetailsEncrypted,
				instance.UseRDSIAMAuth, instance.RequireSSL, instance.Status, instance.EnvironmentID,
				instance.CreatedByID, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnError(errors.New("database error"))

		err := repo.Create(ctx, instance)
		require.Error(t, err)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseInstanceRepository_GetByID(t *testing.T) {
	db, mock := setupDatabaseInstanceMockDB(t)
	defer db.Close()

	encryptionKey := "test-encryption-key-32-bytes-long!!"
	repo := NewDatabaseInstanceRepository(db, encryptionKey)
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		instanceID := uuid.New()
		createdByID := uuid.New()
		now := time.Now()
		description := "Test instance"
		encryptedDetails := "encrypted-data"

		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "database_engine", "host", "port", "username",
			"connection_details_encrypted", "use_rds_iam_auth", "require_ssl", "status", "environment_id", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(instanceID, "Test Instance", description, "postgres", "localhost", 5432, "postgres",
				encryptedDetails, false, true, "Active", nil, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM database_instances WHERE id =").
			WithArgs(instanceID).
			WillReturnRows(rows)

		instance, err := repo.GetByID(ctx, instanceID)
		require.NoError(t, err)
		require.NotNil(t, instance)
		assert.Equal(t, instanceID, instance.ID)
		assert.Equal(t, "Test Instance", instance.Name)
		assert.Equal(t, &description, instance.Description)
		assert.Equal(t, models.DatabaseEnginePostgreSQL, instance.DatabaseEngine)
		assert.Equal(t, "localhost", instance.Host)
		assert.Equal(t, 5432, instance.Port)
		assert.Equal(t, "postgres", instance.Username)
		assert.Equal(t, &encryptedDetails, instance.ConnectionDetailsEncrypted)
		assert.False(t, instance.UseRDSIAMAuth)
		assert.True(t, instance.RequireSSL)
		assert.Equal(t, models.DatabaseInstanceStatusActive, instance.Status)
		assert.Equal(t, createdByID, instance.CreatedByID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful retrieval with nil description", func(t *testing.T) {
		instanceID := uuid.New()
		createdByID := uuid.New()
		now := time.Now()
		encryptedDetails := "encrypted-data"

		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "database_engine", "host", "port", "username",
			"connection_details_encrypted", "use_rds_iam_auth", "require_ssl", "status", "environment_id", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(instanceID, "Test Instance", nil, "mysql", "localhost", 3306, "root",
				encryptedDetails, true, false, "Active", nil, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM database_instances WHERE id =").
			WithArgs(instanceID).
			WillReturnRows(rows)

		instance, err := repo.GetByID(ctx, instanceID)
		require.NoError(t, err)
		require.NotNil(t, instance)
		assert.Nil(t, instance.Description)
		assert.Equal(t, models.DatabaseEngineMySQL, instance.DatabaseEngine)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("instance not found", func(t *testing.T) {
		instanceID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM database_instances WHERE id =").
			WithArgs(instanceID).
			WillReturnError(sql.ErrNoRows)

		instance, err := repo.GetByID(ctx, instanceID)
		assert.Equal(t, repository.ErrNotFound, err)
		assert.Nil(t, instance)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		instanceID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM database_instances WHERE id =").
			WithArgs(instanceID).
			WillReturnError(errors.New("database error"))

		instance, err := repo.GetByID(ctx, instanceID)
		require.Error(t, err)
		assert.Nil(t, instance)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseInstanceRepository_List(t *testing.T) {
	db, mock := setupDatabaseInstanceMockDB(t)
	defer db.Close()

	encryptionKey := "test-encryption-key-32-bytes-long!!"
	repo := NewDatabaseInstanceRepository(db, encryptionKey)
	ctx := context.Background()

	t.Run("successful list with results", func(t *testing.T) {
		now := time.Now()
		instanceID1 := uuid.New()
		instanceID2 := uuid.New()
		createdByID := uuid.New()
		encryptedDetails := "encrypted-data"

		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "database_engine", "host", "port", "username",
			"connection_details_encrypted", "use_rds_iam_auth", "require_ssl", "status", "environment_id", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(instanceID1, "Instance 1", nil, "postgres", "localhost", 5432, "postgres",
				encryptedDetails, false, true, "Active", nil, createdByID, now, now).
			AddRow(instanceID2, "Instance 2", nil, "mysql", "localhost", 3306, "root",
				encryptedDetails, false, false, "Active", nil, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM database_instances ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		instances, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		assert.Len(t, instances, 2)
		assert.Equal(t, instanceID1, instances[0].ID)
		assert.Equal(t, instanceID2, instances[1].ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "database_engine", "host", "port", "username",
			"connection_details_encrypted", "use_rds_iam_auth", "require_ssl", "status", "environment_id", "created_by_id", "created_at", "updated_at",
		})

		mock.ExpectQuery("SELECT (.+) FROM database_instances ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		instances, err := repo.List(ctx, 10, 0)
		require.NoError(t, err)
		assert.Len(t, instances, 0)
		assert.NotNil(t, instances) // Should return empty slice, not nil

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("list with pagination", func(t *testing.T) {
		now := time.Now()
		instanceID := uuid.New()
		createdByID := uuid.New()
		encryptedDetails := "encrypted-data"

		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "database_engine", "host", "port", "username",
			"connection_details_encrypted", "use_rds_iam_auth", "require_ssl", "status", "environment_id", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(instanceID, "Instance 1", nil, "postgres", "localhost", 5432, "postgres",
				encryptedDetails, false, true, "Active", nil, createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM database_instances ORDER BY created_at DESC LIMIT").
			WithArgs(5, 10).
			WillReturnRows(rows)

		instances, err := repo.List(ctx, 5, 10)
		require.NoError(t, err)
		assert.Len(t, instances, 1)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM database_instances ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnError(errors.New("database error"))

		instances, err := repo.List(ctx, 10, 0)
		require.Error(t, err)
		assert.Nil(t, instances)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseInstanceRepository_Count(t *testing.T) {
	db, mock := setupDatabaseInstanceMockDB(t)
	defer db.Close()

	encryptionKey := "test-encryption-key-32-bytes-long!!"
	repo := NewDatabaseInstanceRepository(db, encryptionKey)
	ctx := context.Background()

	t.Run("successful count", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM database_instances").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count zero", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM database_instances").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("count large number", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM database_instances").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1000))

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1000), count)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM database_instances").
			WillReturnError(errors.New("database error"))

		count, err := repo.Count(ctx)
		require.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseInstanceRepository_Update(t *testing.T) {
	db, mock := setupDatabaseInstanceMockDB(t)
	defer db.Close()

	encryptionKey := "test-encryption-key-32-bytes-long!!"
	repo := NewDatabaseInstanceRepository(db, encryptionKey)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		instanceID := uuid.New()
		createdByID := uuid.New()
		description := "Updated description"
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			ID:                         instanceID,
			Name:                       "Updated Instance",
			Description:                &description,
			Host:                       "newhost",
			Port:                       5433,
			DatabaseEngine:             models.DatabaseEnginePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			UseRDSIAMAuth:              false,
			RequireSSL:                 true,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                createdByID,
		}

		mock.ExpectExec("UPDATE database_instances").
			WithArgs(instanceID, instance.Name, instance.Description, instance.DatabaseEngine,
				instance.Host, instance.Port, instance.Username, instance.ConnectionDetailsEncrypted,
				instance.UseRDSIAMAuth, instance.RequireSSL, instance.Status, instance.EnvironmentID,
				sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, instance)
		require.NoError(t, err)
		assert.False(t, instance.UpdatedAt.IsZero())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful update with nil description", func(t *testing.T) {
		instanceID := uuid.New()
		createdByID := uuid.New()
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			ID:                         instanceID,
			Name:                       "Updated Instance",
			Description:                nil,
			Host:                       "newhost",
			Port:                       5433,
			DatabaseEngine:             models.DatabaseEnginePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			UseRDSIAMAuth:              false,
			RequireSSL:                 true,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                createdByID,
		}

		mock.ExpectExec("UPDATE database_instances").
			WithArgs(instanceID, instance.Name, nil, instance.DatabaseEngine,
				instance.Host, instance.Port, instance.Username, instance.ConnectionDetailsEncrypted,
				instance.UseRDSIAMAuth, instance.RequireSSL, instance.Status, instance.EnvironmentID,
				sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, instance)
		require.NoError(t, err)
		assert.Nil(t, instance.Description)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("instance not found", func(t *testing.T) {
		instanceID := uuid.New()
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			ID:                         instanceID,
			Name:                       "Updated Instance",
			Host:                       "newhost",
			Port:                       5433,
			DatabaseEngine:             models.DatabaseEnginePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                uuid.New(),
		}

		mock.ExpectExec("UPDATE database_instances").
			WithArgs(instanceID, instance.Name, instance.Description, instance.DatabaseEngine,
				instance.Host, instance.Port, instance.Username, instance.ConnectionDetailsEncrypted,
				instance.UseRDSIAMAuth, instance.RequireSSL, instance.Status, instance.EnvironmentID,
				sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(ctx, instance)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			ID:                         uuid.New(),
			Name:                       "",
			Host:                       "newhost",
			Port:                       5433,
			DatabaseEngine:             models.DatabaseEnginePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                uuid.New(),
		}

		err := repo.Update(ctx, instance)
		assert.Equal(t, models.ErrDatabaseInstanceNameRequired, err)
	})

	t.Run("validation error - invalid port", func(t *testing.T) {
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			ID:                         uuid.New(),
			Name:                       "Updated Instance",
			Host:                       "newhost",
			Port:                       0,
			DatabaseEngine:             models.DatabaseEnginePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                uuid.New(),
		}

		err := repo.Update(ctx, instance)
		assert.Equal(t, models.ErrPortInvalid, err)
	})

	t.Run("database error", func(t *testing.T) {
		instanceID := uuid.New()
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			ID:                         instanceID,
			Name:                       "Updated Instance",
			Host:                       "newhost",
			Port:                       5433,
			DatabaseEngine:             models.DatabaseEnginePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			UseRDSIAMAuth:              false,
			RequireSSL:                 true,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                uuid.New(),
		}

		mock.ExpectExec("UPDATE database_instances").
			WithArgs(instanceID, instance.Name, instance.Description, instance.DatabaseEngine,
				instance.Host, instance.Port, instance.Username, instance.ConnectionDetailsEncrypted,
				instance.UseRDSIAMAuth, instance.RequireSSL, instance.Status, instance.EnvironmentID,
				sqlmock.AnyArg()).
			WillReturnError(errors.New("database error"))

		err := repo.Update(ctx, instance)
		require.Error(t, err)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDatabaseInstanceRepository_Delete(t *testing.T) {
	db, mock := setupDatabaseInstanceMockDB(t)
	defer db.Close()

	encryptionKey := "test-encryption-key-32-bytes-long!!"
	repo := NewDatabaseInstanceRepository(db, encryptionKey)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		instanceID := uuid.New()

		mock.ExpectExec("DELETE FROM database_instances WHERE id").
			WithArgs(instanceID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(ctx, instanceID)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("instance not found", func(t *testing.T) {
		instanceID := uuid.New()

		mock.ExpectExec("DELETE FROM database_instances WHERE id").
			WithArgs(instanceID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(ctx, instanceID)
		assert.Equal(t, repository.ErrNotFound, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		instanceID := uuid.New()

		mock.ExpectExec("DELETE FROM database_instances WHERE id").
			WithArgs(instanceID).
			WillReturnError(errors.New("database error"))

		err := repo.Delete(ctx, instanceID)
		require.Error(t, err)
		assert.Equal(t, "database error", err.Error())

		require.NoError(t, mock.ExpectationsWereMet())
	})
}
