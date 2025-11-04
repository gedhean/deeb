package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
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

func TestNewAuditLogRepository(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	repo := NewAuditLogRepository(db)
	assert.NotNil(t, repo)
}

func TestAuditLogRepository_Create(t *testing.T) {
	t.Run("successful creation with all fields", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		userID := uuid.New()
		resourceID := uuid.New()
		details := json.RawMessage(`{"key": "value"}`)
		ipAddr := net.ParseIP("192.168.1.1")
		userAgent := "Mozilla/5.0"
		timestamp := time.Now()

		auditLog := &models.AuditLog{
			ID:           uuid.New(),
			UserID:       userID,
			Action:       "create_change_plan",
			ResourceType: "change_plan",
			ResourceID:   resourceID,
			Details:      details,
			IPAddress:    &ipAddr,
			UserAgent:    &userAgent,
			Timestamp:    timestamp,
		}

		sqlMock.ExpectExec(`INSERT INTO audit_logs`).
			WithArgs(
				auditLog.ID,
				auditLog.UserID,
				auditLog.Action,
				auditLog.ResourceType,
				auditLog.ResourceID,
				auditLog.Details,
				auditLog.IPAddress,
				auditLog.UserAgent,
				auditLog.Timestamp,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.Create(ctx, auditLog)
		require.NoError(t, err)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("successful creation with auto-generated ID", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		auditLog := &models.AuditLog{
			ID:           uuid.Nil, // Will be auto-generated
			UserID:       uuid.New(),
			Action:       "update_change_plan",
			ResourceType: "change_plan",
			ResourceID:   uuid.New(),
			Timestamp:    time.Now(),
		}

		sqlMock.ExpectExec(`INSERT INTO audit_logs`).
			WithArgs(
				sqlmock.AnyArg(), // ID will be generated
				auditLog.UserID,
				auditLog.Action,
				auditLog.ResourceType,
				auditLog.ResourceID,
				auditLog.Details,
				auditLog.IPAddress,
				auditLog.UserAgent,
				auditLog.Timestamp,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.Create(ctx, auditLog)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, auditLog.ID)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("successful creation with auto-generated timestamp", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		auditLog := &models.AuditLog{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			Action:       "delete_change_plan",
			ResourceType: "change_plan",
			ResourceID:   uuid.New(),
			Timestamp:    time.Time{}, // Zero time, will be auto-generated
		}

		sqlMock.ExpectExec(`INSERT INTO audit_logs`).
			WithArgs(
				auditLog.ID,
				auditLog.UserID,
				auditLog.Action,
				auditLog.ResourceType,
				auditLog.ResourceID,
				auditLog.Details,
				auditLog.IPAddress,
				auditLog.UserAgent,
				sqlmock.AnyArg(), // Timestamp will be generated
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err = repo.Create(ctx, auditLog)
		require.NoError(t, err)
		assert.False(t, auditLog.Timestamp.IsZero())
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		auditLog := &models.AuditLog{
			ID:           uuid.New(),
			UserID:       uuid.New(),
			Action:       "create_change_plan",
			ResourceType: "change_plan",
			ResourceID:   uuid.New(),
			Timestamp:    time.Now(),
		}

		sqlMock.ExpectExec(`INSERT INTO audit_logs`).
			WillReturnError(sql.ErrConnDone)

		err = repo.Create(ctx, auditLog)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create audit log")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestAuditLogRepository_List(t *testing.T) {
	t.Run("successful list with pagination", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		limit := 10
		offset := 5

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		}).
			AddRow(
				uuid.New(), uuid.New(), "create_change_plan", "change_plan", uuid.New(),
				json.RawMessage(`{"key": "value"}`), net.ParseIP("192.168.1.1"), "Mozilla/5.0", time.Now(),
			).
			AddRow(
				uuid.New(), uuid.New(), "update_change_plan", "change_plan", uuid.New(),
				[]byte{}, sql.NullString{}, sql.NullString{}, time.Now(),
			)

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*ORDER BY timestamp DESC.*LIMIT.*OFFSET`).
			WithArgs(limit, offset).
			WillReturnRows(rows)

		logs, err := repo.List(ctx, limit, offset)
		require.NoError(t, err)
		assert.Len(t, logs, 2)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("successful list with zero offset", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		limit := 20
		offset := 0

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		})

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*ORDER BY timestamp DESC.*LIMIT.*OFFSET`).
			WithArgs(limit, offset).
			WillReturnRows(rows)

		logs, err := repo.List(ctx, limit, offset)
		require.NoError(t, err)
		assert.Len(t, logs, 0)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs`).
			WillReturnError(sql.ErrConnDone)

		logs, err := repo.List(ctx, 10, 0)
		require.Error(t, err)
		assert.Nil(t, logs)
		assert.Contains(t, err.Error(), "failed to list audit logs")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestAuditLogRepository_GetByUserID(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		userID := uuid.New()
		limit := 10
		offset := 0

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		}).
			AddRow(
				uuid.New(), userID, "create_change_plan", "change_plan", uuid.New(),
				[]byte{}, sql.NullString{}, sql.NullString{}, time.Now(),
			)

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE user_id`).
			WithArgs(userID, limit, offset).
			WillReturnRows(rows)

		logs, err := repo.GetByUserID(ctx, userID, limit, offset)
		require.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Equal(t, userID, logs[0].UserID)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("empty results", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		userID := uuid.New()

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		})

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE user_id`).
			WithArgs(userID, 10, 0).
			WillReturnRows(rows)

		logs, err := repo.GetByUserID(ctx, userID, 10, 0)
		require.NoError(t, err)
		assert.Len(t, logs, 0)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE user_id`).
			WillReturnError(sql.ErrConnDone)

		logs, err := repo.GetByUserID(ctx, uuid.New(), 10, 0)
		require.Error(t, err)
		assert.Nil(t, logs)
		assert.Contains(t, err.Error(), "failed to get audit logs by user ID")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestAuditLogRepository_GetByAction(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		action := "create_change_plan"
		limit := 10
		offset := 0

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		}).
			AddRow(
				uuid.New(), uuid.New(), action, "change_plan", uuid.New(),
				[]byte{}, sql.NullString{}, sql.NullString{}, time.Now(),
			)

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE action`).
			WithArgs(action, limit, offset).
			WillReturnRows(rows)

		logs, err := repo.GetByAction(ctx, action, limit, offset)
		require.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Equal(t, action, logs[0].Action)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE action`).
			WillReturnError(sql.ErrConnDone)

		logs, err := repo.GetByAction(ctx, "create_change_plan", 10, 0)
		require.Error(t, err)
		assert.Nil(t, logs)
		assert.Contains(t, err.Error(), "failed to get audit logs by action")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestAuditLogRepository_GetByTargetType(t *testing.T) {
	t.Run("successful retrieval", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		resourceType := "change_plan"
		limit := 10
		offset := 0

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		}).
			AddRow(
				uuid.New(), uuid.New(), "create_change_plan", resourceType, uuid.New(),
				[]byte{}, sql.NullString{}, sql.NullString{}, time.Now(),
			)

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE resource_type`).
			WithArgs(resourceType, limit, offset).
			WillReturnRows(rows)

		logs, err := repo.GetByTargetType(ctx, resourceType, limit, offset)
		require.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Equal(t, resourceType, logs[0].ResourceType)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE resource_type`).
			WillReturnError(sql.ErrConnDone)

		logs, err := repo.GetByTargetType(ctx, "change_plan", 10, 0)
		require.Error(t, err)
		assert.Nil(t, logs)
		assert.Contains(t, err.Error(), "failed to get audit logs by resource type")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestAuditLogRepository_Filter(t *testing.T) {
	t.Run("filter by user ID only", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		userID := uuid.New()
		filters := repository.AuditLogListFilters{
			Limit:  10,
			Offset: 0,
			UserID: userID,
		}

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		}).
			AddRow(
				uuid.New(), userID, "create_change_plan", "change_plan", uuid.New(),
				[]byte{}, sql.NullString{}, sql.NullString{}, time.Now(),
			)

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE user_id`).
			WithArgs(userID, 10).
			WillReturnRows(rows)

		logs, err := repo.Filter(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("filter by action only", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		action := "create_change_plan"
		filters := repository.AuditLogListFilters{
			Limit:  10,
			Offset: 0,
			Action: action,
		}

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		})

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE action`).
			WithArgs(action, 10).
			WillReturnRows(rows)

		logs, err := repo.Filter(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, logs, 0)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("filter by resource type only", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		resourceType := "change_plan"
		filters := repository.AuditLogListFilters{
			Limit:        10,
			Offset:       0,
			ResourceType: resourceType,
		}

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		})

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE resource_type`).
			WithArgs(resourceType, 10).
			WillReturnRows(rows)

		logs, err := repo.Filter(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, logs, 0)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("filter by resource ID only", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		resourceID := uuid.New()
		filters := repository.AuditLogListFilters{
			Limit:      10,
			Offset:     0,
			ResourceID: resourceID,
		}

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		})

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE resource_id`).
			WithArgs(resourceID, 10).
			WillReturnRows(rows)

		logs, err := repo.Filter(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, logs, 0)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("filter by date range", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now()
		filters := repository.AuditLogListFilters{
			Limit:     10,
			Offset:    0,
			StartDate: &startDate,
			EndDate:   &endDate,
		}

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		})

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE timestamp`).
			WithArgs(startDate, endDate, 10).
			WillReturnRows(rows)

		logs, err := repo.Filter(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, logs, 0)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("filter by multiple criteria", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		userID := uuid.New()
		action := "create_change_plan"
		resourceType := "change_plan"
		resourceID := uuid.New()
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now()

		filters := repository.AuditLogListFilters{
			Limit:        10,
			Offset:       5,
			UserID:       userID,
			Action:       action,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			StartDate:    &startDate,
			EndDate:      &endDate,
		}

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		})

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE.*AND.*AND.*AND.*AND.*AND.*ORDER BY timestamp DESC.*LIMIT.*OFFSET`).
			WithArgs(userID, action, resourceType, resourceID, startDate, endDate, 10, 5).
			WillReturnRows(rows)

		logs, err := repo.Filter(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, logs, 0)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("filter without pagination", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		userID := uuid.New()
		filters := repository.AuditLogListFilters{
			Limit:  0, // No limit
			Offset:  0,
			UserID: userID,
		}

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		})

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*WHERE user_id.*ORDER BY timestamp DESC`).
			WithArgs(userID).
			WillReturnRows(rows)

		logs, err := repo.Filter(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, logs, 0)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("filter without any criteria", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		filters := repository.AuditLogListFilters{
			Limit:  10,
			Offset: 0,
		}

		rows := sqlmock.NewRows([]string{
			"id", "user_id", "action", "resource_type", "resource_id",
			"details", "ip_address", "user_agent", "timestamp",
		})

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs.*ORDER BY timestamp DESC.*LIMIT`).
			WithArgs(10).
			WillReturnRows(rows)

		logs, err := repo.Filter(ctx, filters)
		require.NoError(t, err)
		assert.Len(t, logs, 0)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		filters := repository.AuditLogListFilters{
			Limit:  10,
			Offset: 0,
			UserID: uuid.New(),
		}

		sqlMock.ExpectQuery(`SELECT.*FROM audit_logs`).
			WillReturnError(sql.ErrConnDone)

		logs, err := repo.Filter(ctx, filters)
		require.Error(t, err)
		assert.Nil(t, logs)
		assert.Contains(t, err.Error(), "failed to filter audit logs")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestAuditLogRepository_Count(t *testing.T) {
	t.Run("successful count", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		expectedCount := int64(42)

		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(expectedCount)

		sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs`).
			WillReturnRows(rows)

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, expectedCount, count)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("count zero", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(0)

		sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs`).
			WillReturnRows(rows)

		count, err := repo.Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs`).
			WillReturnError(sql.ErrConnDone)

		count, err := repo.Count(ctx)
		require.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, err.Error(), "failed to count audit logs")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

func TestAuditLogRepository_CountWithFilters(t *testing.T) {
	t.Run("count with user ID filter", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		userID := uuid.New()
		filters := repository.AuditLogListFilters{
			UserID: userID,
		}

		expectedCount := int64(5)

		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(expectedCount)

		sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs.*WHERE user_id`).
			WithArgs(userID).
			WillReturnRows(rows)

		count, err := repo.CountWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, expectedCount, count)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("count with multiple filters", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		userID := uuid.New()
		action := "create_change_plan"
		resourceType := "change_plan"
		startDate := time.Now().Add(-24 * time.Hour)
		endDate := time.Now()

		filters := repository.AuditLogListFilters{
			UserID:       userID,
			Action:       action,
			ResourceType: resourceType,
			StartDate:    &startDate,
			EndDate:      &endDate,
		}

		expectedCount := int64(3)

		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(expectedCount)

		sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs.*WHERE.*AND.*AND.*AND.*AND`).
			WithArgs(userID, action, resourceType, startDate, endDate).
			WillReturnRows(rows)

		count, err := repo.CountWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, expectedCount, count)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("count without filters", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		filters := repository.AuditLogListFilters{}

		expectedCount := int64(100)

		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(expectedCount)

		sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs`).
			WillReturnRows(rows)

		count, err := repo.CountWithFilters(ctx, filters)
		require.NoError(t, err)
		assert.Equal(t, expectedCount, count)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mockDB, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		db := sqlx.NewDb(mockDB, "sqlmock")
		defer db.Close()

		repo := NewAuditLogRepository(db)
		ctx := context.Background()

		filters := repository.AuditLogListFilters{
			UserID: uuid.New(),
		}

		sqlMock.ExpectQuery(`SELECT COUNT\(\*\) FROM audit_logs`).
			WillReturnError(sql.ErrConnDone)

		count, err := repo.CountWithFilters(ctx, filters)
		require.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Contains(t, err.Error(), "failed to count audit logs with filters")
		assert.NoError(t, sqlMock.ExpectationsWereMet())
	})
}

