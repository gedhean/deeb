package postgres

import (
	"context"
	"database/sql"
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

func TestNewExecutionLogRepository(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	repo := NewExecutionLogRepository(db)
	assert.NotNil(t, repo)
}

func TestExecutionLogRepository_Create(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	repo := NewExecutionLogRepository(db)

	ctx := context.Background()
	log := &models.ExecutionLog{
		ID:                  uuid.New(),
		ChangePlanID:        uuid.New(),
		ExecutorID:          uuid.New(),
		Status:              models.ExecutionLogStatusCompleted,
		ExecutionDurationMs: 1500,
		RowsAffected:        intPtr(10),
		ErrorMessage:        nil,
		StartedAt:           time.Now(),
		CompletedAt:         time.Now(),
		CreatedAt:           time.Now(),
	}

	sqlMock.ExpectExec(`INSERT INTO execution_logs`).
		WithArgs(
			log.ID,
			log.ChangePlanID,
			log.ExecutorID,
			log.Status,
			log.ExecutionDurationMs,
			log.RowsAffected,
			log.ErrorMessage,
			log.StartedAt,
			log.CompletedAt,
			log.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.Create(ctx, log)
	require.NoError(t, err)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLogRepository_GetByChangePlanID(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	repo := NewExecutionLogRepository(db)

	ctx := context.Background()
	changePlanID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "change_plan_id", "executor_id", "status",
		"execution_duration_ms", "rows_affected", "error_message",
		"started_at", "completed_at", "created_at",
	}).
		AddRow(
			uuid.New(), changePlanID, uuid.New(), "Completed",
			1500, 10, nil,
			time.Now(), time.Now(), time.Now(),
		).
		AddRow(
			uuid.New(), changePlanID, uuid.New(), "Failed",
			500, nil, executionLogStringPtr("error message"),
			time.Now(), time.Now(), time.Now(),
		)

	sqlMock.ExpectQuery(`SELECT.*FROM execution_logs.*WHERE change_plan_id`).
		WithArgs(changePlanID).
		WillReturnRows(rows)

	logs, err := repo.GetByChangePlanID(ctx, changePlanID)
	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, models.ExecutionLogStatusCompleted, logs[0].Status)
	assert.Equal(t, models.ExecutionLogStatusFailed, logs[1].Status)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLogRepository_GetByID(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	repo := NewExecutionLogRepository(db)

	ctx := context.Background()
	logID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "change_plan_id", "executor_id", "status",
		"execution_duration_ms", "rows_affected", "error_message",
		"started_at", "completed_at", "created_at",
	}).
		AddRow(
			logID, uuid.New(), uuid.New(), "Completed",
			1500, 10, nil,
			time.Now(), time.Now(), time.Now(),
		)

	sqlMock.ExpectQuery(`SELECT.*FROM execution_logs.*WHERE id`).
		WithArgs(logID).
		WillReturnRows(rows)

	log, err := repo.GetByID(ctx, logID)
	require.NoError(t, err)
	assert.NotNil(t, log)
	assert.Equal(t, logID, log.ID)
	assert.Equal(t, models.ExecutionLogStatusCompleted, log.Status)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLogRepository_GetByID_NotFound(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	repo := NewExecutionLogRepository(db)

	ctx := context.Background()
	logID := uuid.New()

	sqlMock.ExpectQuery(`SELECT.*FROM execution_logs.*WHERE id`).
		WithArgs(logID).
		WillReturnError(sql.ErrNoRows)

	log, err := repo.GetByID(ctx, logID)
	assert.Error(t, err)
	assert.Nil(t, log)
	assert.Equal(t, repository.ErrNotFound, err)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLogRepository_GetLatestByChangePlanID(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	repo := NewExecutionLogRepository(db)

	ctx := context.Background()
	changePlanID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "change_plan_id", "executor_id", "status",
		"execution_duration_ms", "rows_affected", "error_message",
		"started_at", "completed_at", "created_at",
	}).
		AddRow(
			uuid.New(), changePlanID, uuid.New(), "Completed",
			1500, 10, nil,
			time.Now(), time.Now(), time.Now(),
		)

	sqlMock.ExpectQuery(`SELECT.*FROM execution_logs.*WHERE change_plan_id.*ORDER BY started_at DESC.*LIMIT 1`).
		WithArgs(changePlanID).
		WillReturnRows(rows)

	log, err := repo.GetLatestByChangePlanID(ctx, changePlanID)
	require.NoError(t, err)
	assert.NotNil(t, log)
	assert.Equal(t, changePlanID, log.ChangePlanID)
	assert.Equal(t, models.ExecutionLogStatusCompleted, log.Status)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestExecutionLogRepository_GetLatestByChangePlanID_NotFound(t *testing.T) {
	mockDB, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "sqlmock")
	defer db.Close()

	repo := NewExecutionLogRepository(db)

	ctx := context.Background()
	changePlanID := uuid.New()

	sqlMock.ExpectQuery(`SELECT.*FROM execution_logs.*WHERE change_plan_id.*ORDER BY started_at DESC.*LIMIT 1`).
		WithArgs(changePlanID).
		WillReturnError(sql.ErrNoRows)

	log, err := repo.GetLatestByChangePlanID(ctx, changePlanID)
	assert.Error(t, err)
	assert.Nil(t, log)
	assert.Equal(t, repository.ErrNotFound, err)
	assert.NoError(t, sqlMock.ExpectationsWereMet())
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func executionLogStringPtr(s string) *string {
	return &s
}
