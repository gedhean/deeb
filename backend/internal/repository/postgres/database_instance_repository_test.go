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

func setupDatabaseInstanceMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestDatabaseInstanceCreate(t *testing.T) {
	db, mock := setupDatabaseInstanceMockDB(t)
	defer db.Close()

	repo := NewDatabaseInstanceRepository(db, "test-encryption-key-32-bytes-long!!")
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
			DatabaseType:               models.DatabaseTypePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			UseRDSIAMAuth:              false,
			RequireSSL:                 true,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                createdByID,
		}

		mock.ExpectExec("INSERT INTO database_instances").
			WithArgs(sqlmock.AnyArg(), instance.Name, instance.Description, instance.DatabaseType,
				instance.Host, instance.Port, instance.Username, instance.ConnectionDetailsEncrypted,
				instance.UseRDSIAMAuth, instance.RequireSSL, instance.Status, instance.CreatedByID,
				sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, instance)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify ID and timestamps were set
		if instance.ID == uuid.Nil {
			t.Error("Expected ID to be set")
		}
		if instance.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
		if instance.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			Name:                       "",
			Host:                       "localhost",
			Port:                       5432,
			DatabaseType:               models.DatabaseTypePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                uuid.New(),
		}

		err := repo.Create(ctx, instance)
		if err != models.ErrDatabaseInstanceNameRequired {
			t.Errorf("Expected ErrDatabaseInstanceNameRequired, got %v", err)
		}
	})

	t.Run("validation error - invalid port", func(t *testing.T) {
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			Name:                       "Test Instance",
			Host:                       "localhost",
			Port:                       0,
			DatabaseType:               models.DatabaseTypePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                uuid.New(),
		}

		err := repo.Create(ctx, instance)
		if err != models.ErrPortInvalid {
			t.Errorf("Expected ErrPortInvalid, got %v", err)
		}
	})

	t.Run("validation error - invalid database type", func(t *testing.T) {
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			Name:                       "Test Instance",
			Host:                       "localhost",
			Port:                       5432,
			DatabaseType:               "invalid",
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                uuid.New(),
		}

		err := repo.Create(ctx, instance)
		if err != models.ErrDatabaseTypeInvalid {
			t.Errorf("Expected ErrDatabaseTypeInvalid, got %v", err)
		}
	})
}

func TestDatabaseInstanceGetByID(t *testing.T) {
	db, mock := setupDatabaseInstanceMockDB(t)
	defer db.Close()

	repo := NewDatabaseInstanceRepository(db, "test-encryption-key-32-bytes-long!!")
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		instanceID := uuid.New()
		createdByID := uuid.New()
		now := time.Now()
		description := "Test instance"

		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "database_type", "host", "port", "username",
			"connection_details_encrypted", "use_rds_iam_auth", "require_ssl", "status", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(instanceID, "Test Instance", description, "postgresql", "localhost", 5432, "postgres",
				"encrypted-data", false, true, "Active", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM database_instances WHERE id =").
			WithArgs(instanceID).
			WillReturnRows(rows)

		instance, err := repo.GetByID(ctx, instanceID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if instance == nil {
			t.Fatal("Expected instance to be returned")
		}

		if instance.ID != instanceID {
			t.Errorf("Expected ID %s, got %s", instanceID, instance.ID)
		}

		if instance.Name != "Test Instance" {
			t.Errorf("Expected name 'Test Instance', got %s", instance.Name)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("instance not found", func(t *testing.T) {
		instanceID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM database_instances WHERE id =").
			WithArgs(instanceID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "description", "database_type", "host", "port", "username",
				"connection_details_encrypted", "use_rds_iam_auth", "require_ssl", "status", "created_by_id", "created_at", "updated_at",
			}))

		instance, err := repo.GetByID(ctx, instanceID)
		if err != repository.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}

		if instance != nil {
			t.Error("Expected nil instance")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestDatabaseInstanceList(t *testing.T) {
	db, mock := setupDatabaseInstanceMockDB(t)
	defer db.Close()

	repo := NewDatabaseInstanceRepository(db, "test-encryption-key-32-bytes-long!!")
	ctx := context.Background()

	t.Run("successful list with results", func(t *testing.T) {
		now := time.Now()
		instanceID1 := uuid.New()
		instanceID2 := uuid.New()
		createdByID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "database_type", "host", "port", "username",
			"connection_details_encrypted", "use_rds_iam_auth", "require_ssl", "status", "created_by_id", "created_at", "updated_at",
		}).
			AddRow(instanceID1, "Instance 1", nil, "postgresql", "localhost", 5432, "postgres",
				"encrypted-data", false, true, "Active", createdByID, now, now).
			AddRow(instanceID2, "Instance 2", nil, "mysql", "localhost", 3306, "root",
				"encrypted-data", false, false, "Active", createdByID, now, now)

		mock.ExpectQuery("SELECT (.+) FROM database_instances ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		instances, err := repo.List(ctx, 10, 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(instances) != 2 {
			t.Errorf("Expected 2 instances, got %d", len(instances))
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "database_type", "host", "port", "username",
			"connection_details_encrypted", "use_rds_iam_auth", "require_ssl", "status", "created_by_id", "created_at", "updated_at",
		})

		mock.ExpectQuery("SELECT (.+) FROM database_instances ORDER BY created_at DESC LIMIT").
			WithArgs(10, 0).
			WillReturnRows(rows)

		instances, err := repo.List(ctx, 10, 0)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(instances) != 0 {
			t.Errorf("Expected 0 instances, got %d", len(instances))
		}

		if instances == nil {
			t.Error("Expected empty slice, got nil")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

func TestDatabaseInstanceUpdate(t *testing.T) {
	db, mock := setupDatabaseInstanceMockDB(t)
	defer db.Close()

	repo := NewDatabaseInstanceRepository(db, "test-encryption-key-32-bytes-long!!")
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
			DatabaseType:               models.DatabaseTypePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			UseRDSIAMAuth:              false,
			RequireSSL:                 true,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                createdByID,
		}

		mock.ExpectExec("UPDATE database_instances").
			WithArgs(instanceID, instance.Name, instance.Description, instance.DatabaseType,
				instance.Host, instance.Port, instance.Username, instance.ConnectionDetailsEncrypted,
				instance.UseRDSIAMAuth, instance.RequireSSL, instance.Status, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, instance)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify UpdatedAt was set
		if instance.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("instance not found", func(t *testing.T) {
		instanceID := uuid.New()
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			ID:                         instanceID,
			Name:                       "Updated Instance",
			Host:                       "newhost",
			Port:                       5433,
			DatabaseType:               models.DatabaseTypePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                uuid.New(),
		}

		mock.ExpectExec("UPDATE database_instances").
			WithArgs(instanceID, instance.Name, instance.Description, instance.DatabaseType,
				instance.Host, instance.Port, instance.Username, instance.ConnectionDetailsEncrypted,
				instance.UseRDSIAMAuth, instance.RequireSSL, instance.Status, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(ctx, instance)
		if err != repository.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("validation error - empty name", func(t *testing.T) {
		encryptedDetails := "encrypted-password-data"
		instance := &models.DatabaseInstance{
			ID:                         uuid.New(),
			Name:                       "",
			Host:                       "newhost",
			Port:                       5433,
			DatabaseType:               models.DatabaseTypePostgreSQL,
			Username:                   "postgres",
			ConnectionDetailsEncrypted: &encryptedDetails,
			Status:                     models.DatabaseInstanceStatusActive,
			CreatedByID:                uuid.New(),
		}

		err := repo.Update(ctx, instance)
		if err != models.ErrDatabaseInstanceNameRequired {
			t.Errorf("Expected ErrDatabaseInstanceNameRequired, got %v", err)
		}
	})
}

func TestDatabaseInstanceDelete(t *testing.T) {
	db, mock := setupDatabaseInstanceMockDB(t)
	defer db.Close()

	repo := NewDatabaseInstanceRepository(db, "test-encryption-key-32-bytes-long!!")
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		instanceID := uuid.New()

		mock.ExpectExec("DELETE FROM database_instances WHERE id").
			WithArgs(instanceID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(ctx, instanceID)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("instance not found", func(t *testing.T) {
		instanceID := uuid.New()

		mock.ExpectExec("DELETE FROM database_instances WHERE id").
			WithArgs(instanceID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(ctx, instanceID)
		if err != repository.ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}
