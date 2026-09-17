package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestParseAlterStatement(t *testing.T) {
	tests := []struct {
		name      string
		sql       string
		wantTable string
		wantAlter string
		wantErr   bool
	}{
		{
			name:      "simple add index",
			sql:       "ALTER TABLE ramp_markets ADD INDEX idx_x (nsx_event_id, nsx_market_id);",
			wantTable: "ramp_markets",
			wantAlter: "ADD INDEX idx_x (nsx_event_id, nsx_market_id)",
		},
		{
			name:      "no trailing semicolon",
			sql:       "ALTER TABLE users ADD COLUMN age INT",
			wantTable: "users",
			wantAlter: "ADD COLUMN age INT",
		},
		{
			name:      "backtick quoted table",
			sql:       "ALTER TABLE `orders` ADD COLUMN total DECIMAL(10,2);",
			wantTable: "orders",
			wantAlter: "ADD COLUMN total DECIMAL(10,2)",
		},
		{
			name:      "double quoted table (postgres style)",
			sql:       `ALTER TABLE "events" ADD COLUMN region TEXT;`,
			wantTable: "events",
			wantAlter: "ADD COLUMN region TEXT",
		},
		{
			name:      "schema-qualified table is normalized",
			sql:       "ALTER TABLE public.ramp_markets ADD INDEX idx_x (a);",
			wantTable: "ramp_markets",
			wantAlter: "ADD INDEX idx_x (a)",
		},
		{
			name:    "multi-statement rejected",
			sql:     "ALTER TABLE a ADD COLUMN x INT; ALTER TABLE b ADD COLUMN y INT;",
			wantErr: true,
		},
		{
			name:    "non-alter statement rejected",
			sql:     "CREATE TABLE foo (id INT);",
			wantErr: true,
		},
		{
			name:    "DML rejected",
			sql:     "UPDATE users SET age = 1;",
			wantErr: true,
		},
		{
			name:    "empty rejected",
			sql:     "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table, alter, err := ParseAlterStatement(tt.sql)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantTable, table)
			assert.Equal(t, tt.wantAlter, alter)
		})
	}
}

func TestGhostExecutor_BuildArgs_MySQL(t *testing.T) {
	g := NewGhostExecutor("/usr/local/bin/gh-ost", "", "/flags", nil, nil, nil, 0)

	bin, args, err := g.BuildArgs(
		models.DatabaseEngineMySQL,
		"db.internal", "deeb-executor", "secret", "mydb",
		"ramp_markets", "ADD INDEX idx_x (a)", 3306,
		"/flags/deeb-x.postpone", "/flags/deeb-x.panic",
	)
	require.NoError(t, err)
	assert.Equal(t, "/usr/local/bin/gh-ost", bin)

	joined := joinArgs(args)
	assert.Contains(t, joined, "--host=db.internal")
	assert.Contains(t, joined, "--port=3306")
	assert.Contains(t, joined, "--user=deeb-executor")
	assert.Contains(t, joined, "--password=secret")
	assert.Contains(t, joined, "--database=mydb")
	assert.Contains(t, joined, "--table=ramp_markets")
	assert.Contains(t, joined, "--alter=ADD INDEX idx_x (a)")
	assert.Contains(t, joined, "--allow-on-master")
	assert.Contains(t, joined, "--exact-rowcount")
	assert.Contains(t, joined, "--concurrent-rowcount")
	assert.Contains(t, joined, "--chunk-size=1000")
	assert.Contains(t, joined, "--max-load=Threads_running=25")
	assert.Contains(t, joined, "--critical-load=Threads_running=64")
	assert.Contains(t, joined, "--cut-over=default")
	assert.Contains(t, joined, "--postpone-cut-over-flag-file=/flags/deeb-x.postpone")
	assert.Contains(t, joined, "--panic-flag-file=/flags/deeb-x.panic")
	assert.Contains(t, joined, "--execute")
}

func TestGhostExecutor_BuildArgs_MySQL_MissingBinary(t *testing.T) {
	g := NewGhostExecutor("", "", "/flags", nil, nil, nil, 0)
	_, _, err := g.BuildArgs(
		models.DatabaseEngineMySQL, "h", "u", "p", "d", "t", "ADD COLUMN x INT", 3306, "", "",
	)
	assert.ErrorContains(t, err, "gh-ost binary not configured")
}

func TestGhostExecutor_BuildArgs_Postgres(t *testing.T) {
	g := NewGhostExecutor("", "/usr/local/bin/pg-online-schema-change", "/flags", nil, nil, nil, 0)

	bin, args, err := g.BuildArgs(
		models.DatabaseEnginePostgreSQL,
		"pg.internal", "deeb-executor", "secret", "mydb",
		"ramp_markets", "ADD INDEX idx_x (a)", 5432,
		"", "",
	)
	require.NoError(t, err)
	assert.Equal(t, "/usr/local/bin/pg-online-schema-change", bin)

	joined := joinArgs(args)
	assert.Contains(t, joined, "--host=pg.internal")
	assert.Contains(t, joined, "--port=5432")
	assert.Contains(t, joined, "--dbname=mydb")
	assert.Contains(t, joined, "--table=ramp_markets")
	assert.Contains(t, joined, "--alter=ADD INDEX idx_x (a)")
	assert.Contains(t, joined, "--yes-i-know-what-i-am-doing")
	assert.Contains(t, joined, "--drop-old-table=false")
}

func TestGhostExecutor_BuildArgs_Postgres_MissingBinary(t *testing.T) {
	g := NewGhostExecutor("", "", "/flags", nil, nil, nil, 0)
	_, _, err := g.BuildArgs(
		models.DatabaseEnginePostgreSQL, "h", "u", "p", "d", "t", "ADD COLUMN x INT", 5432, "", "",
	)
	assert.ErrorContains(t, err, "pg-online-schema-change binary not configured")
}

func TestGhostExecutor_ValidateGhostEligibility(t *testing.T) {
	g := NewGhostExecutor("", "", "", nil, nil, nil, 0)

	ddl := &models.ChangePlan{ChangeType: models.ChangeTypeDDL, TargetDatabaseID: uuid.New()}
	require.NoError(t, g.ValidateGhostEligibility(context.Background(), ddl))

	dml := &models.ChangePlan{ChangeType: models.ChangeTypeDML, TargetDatabaseID: uuid.New()}
	assert.ErrorContains(t, g.ValidateGhostEligibility(context.Background(), dml), "DDL")

	assert.Error(t, g.ValidateGhostEligibility(context.Background(), nil))
}

func TestGhostExecutor_ControlFlagPaths(t *testing.T) {
	g := NewGhostExecutor("", "", "/tmp/flags", nil, nil, nil, 0)
	id := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	postpone, panicPath := g.ControlFlagPaths(id)
	assert.Equal(t, "/tmp/flags/deeb-123e4567-e89b-12d3-a456-426614174000.postpone", postpone)
	assert.Equal(t, "/tmp/flags/deeb-123e4567-e89b-12d3-a456-426614174000.panic", panicPath)
}

func TestParseGhostStatus(t *testing.T) {
	tests := []struct {
		line       string
		wantStatus string
		wantFound  bool
	}{
		{"2026-09-04 00:00:00 INFO postponing cut-over", GhostStatusAwaiting, true},
		{"# Done", GhostStatusCompleted, true},
		{"copy complete: 100%", GhostStatusCopying, true},
		{"panic: something bad", GhostStatusFailed, true},
		{"ERROR could not connect", GhostStatusFailed, true},
		{"random log line", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			status, found := ParseGhostStatus(tt.line)
			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantFound, found)
		})
	}
}

func joinArgs(args []string) string {
	out := ""
	for _, a := range args {
		out += a + " "
	}
	return out
}

// TestExecutionService_GhostDispatch covers the Execute dispatch for ghost
// backend selection (validation gates before any subprocess runs).
func TestExecutionService_GhostDispatch(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012"
	logger := zap.NewNop()
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)
	ctx := context.Background()

	newService := func(changePlanRepo *MockChangePlanRepository, databaseRepo *MockDatabaseRepository, dbInstanceRepo *MockDatabaseInstanceRepository) *ExecutionService {
		return NewExecutionService(
			changePlanRepo,
			dbInstanceRepo,
			databaseRepo,
			new(MockChangelogRepository),
			new(MockSchemaHistoryRepository),
			NewSchemaVersionService(
				new(MockSchemaHistoryRepository),
				new(MockDatabaseInstanceRepository),
				new(MockDatabaseRepository),
				database.NewConnectionManager(encryptionKey, zap.NewNop()),
				encryptionKey,
				zap.NewNop(),
			),
			&MockExecutionLockManager{},
			nil, nil, nil,
			connectionMgr,
			encryptionKey,
			logger,
		)
	}

	t.Run("ghost backend without configured executor is rejected", func(t *testing.T) {
		changePlanID := uuid.New()
		changePlanRepo := new(MockChangePlanRepository)
		changePlanRepo.On("GetByID", mock.Anything, changePlanID).Return(&models.ChangePlan{
			ID:               changePlanID,
			Status:           models.ChangePlanStatusApproved,
			ChangeType:       models.ChangeTypeDDL,
			ExecutionBackend: models.ExecutionBackendGhost,
			TargetDatabaseID: uuid.New(),
		}, nil)

		service := newService(changePlanRepo, new(MockDatabaseRepository), new(MockDatabaseInstanceRepository))
		result, err := service.Execute(ctx, changePlanID, uuid.New())
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrGhostUnsupported)
	})

	t.Run("ghost backend with DML is rejected", func(t *testing.T) {
		changePlanID := uuid.New()
		changePlanRepo := new(MockChangePlanRepository)
		changePlanRepo.On("GetByID", mock.Anything, changePlanID).Return(&models.ChangePlan{
			ID:               changePlanID,
			Status:           models.ChangePlanStatusApproved,
			ChangeType:       models.ChangeTypeDML,
			ExecutionBackend: models.ExecutionBackendGhost,
			TargetDatabaseID: uuid.New(),
		}, nil)

		service := newService(changePlanRepo, new(MockDatabaseRepository), new(MockDatabaseInstanceRepository))
		service.WithGhostSupport(NewGhostExecutor("/bin/true", "", "", nil, nil, nil, 0), nil)

		result, err := service.Execute(ctx, changePlanID, uuid.New())
		assert.Nil(t, result)
		assert.ErrorContains(t, err, "DDL")
	})

	t.Run("ghost backend rejected on production environment", func(t *testing.T) {
		changePlanID := uuid.New()
		dbID := uuid.New()
		envID := uuid.New()

		changePlanRepo := new(MockChangePlanRepository)
		changePlanRepo.On("GetByID", mock.Anything, changePlanID).Return(&models.ChangePlan{
			ID:               changePlanID,
			Status:           models.ChangePlanStatusApproved,
			ChangeType:       models.ChangeTypeDDL,
			ExecutionBackend: models.ExecutionBackendGhost,
			TargetDatabaseID: dbID,
		}, nil)

		databaseRepo := new(MockDatabaseRepository)
		databaseRepo.On("GetByID", mock.Anything, dbID).Return(&models.Database{
			ID: dbID,
		}, nil)

		dbInstanceRepo := new(MockDatabaseInstanceRepository)
		dbInstanceRepo.On("GetByID", mock.Anything, mock.Anything).Return(&models.DatabaseInstance{
			EnvironmentID: &envID,
		}, nil)

		envRepo := new(MockEnvironmentRepository)
		envRepo.On("GetByID", mock.Anything, envID).Return(&models.Environment{
			ID: envID, Name: "Production", IsProduction: true,
		}, nil)

		service := newService(changePlanRepo, databaseRepo, dbInstanceRepo)
		service.WithGhostSupport(NewGhostExecutor("/bin/true", "", "", nil, nil, nil, 0), envRepo)

		result, err := service.Execute(ctx, changePlanID, uuid.New())
		assert.Nil(t, result)
		assert.ErrorIs(t, err, ErrGhostRejectedEnv)
	})

	t.Run("ghost backend with non-ALTER SQL is rejected", func(t *testing.T) {
		changePlanID := uuid.New()
		dbID := uuid.New()
		envID := uuid.New()

		changePlanRepo := new(MockChangePlanRepository)
		changePlanRepo.On("GetByID", mock.Anything, changePlanID).Return(&models.ChangePlan{
			ID:               changePlanID,
			Status:           models.ChangePlanStatusApproved,
			ChangeType:       models.ChangeTypeDDL,
			SQLContent:       "CREATE TABLE foo (id INT);",
			ExecutionBackend: models.ExecutionBackendGhost,
			TargetDatabaseID: dbID,
		}, nil)

		databaseRepo := new(MockDatabaseRepository)
		databaseRepo.On("GetByID", mock.Anything, dbID).Return(&models.Database{
			ID: dbID,
		}, nil)

		dbInstanceRepo := new(MockDatabaseInstanceRepository)
		dbInstanceRepo.On("GetByID", mock.Anything, mock.Anything).Return(&models.DatabaseInstance{
			EnvironmentID: &envID,
		}, nil)

		envRepo := new(MockEnvironmentRepository)
		envRepo.On("GetByID", mock.Anything, envID).Return(&models.Environment{
			ID: envID, Name: "Staging", IsProduction: false,
		}, nil)

		service := newService(changePlanRepo, databaseRepo, dbInstanceRepo)
		service.WithGhostSupport(NewGhostExecutor("/bin/true", "", "", nil, nil, nil, 0), envRepo)

		result, err := service.Execute(ctx, changePlanID, uuid.New())
		assert.Nil(t, result)
		assert.ErrorContains(t, err, "ALTER TABLE")
	})
}

// TestExecutionService_Execute_DetachesRequestContext verifies that cleanup
// continues even when the caller's context (e.g. HTTP request) is canceled.
func TestExecutionService_Execute_DetachesRequestContext(t *testing.T) {
	encryptionKey := "12345678901234567890123456789012"
	logger := zap.NewNop()
	connectionMgr := database.NewConnectionManager(encryptionKey, logger)

	changePlanID := uuid.New()
	changePlanRepo := new(MockChangePlanRepository)
	// Mock matches regardless of context (Execute detaches via WithoutCancel)
	changePlanRepo.On("GetByID", mock.Anything, changePlanID).Return(nil, repository.ErrNotFound)

	service := NewExecutionService(
		changePlanRepo,
		new(MockDatabaseInstanceRepository),
		new(MockDatabaseRepository),
		new(MockChangelogRepository),
		new(MockSchemaHistoryRepository),
		NewSchemaVersionService(
			new(MockSchemaHistoryRepository),
			new(MockDatabaseInstanceRepository),
			new(MockDatabaseRepository),
			database.NewConnectionManager(encryptionKey, zap.NewNop()),
			encryptionKey,
			zap.NewNop(),
		),
		&MockExecutionLockManager{},
		nil, nil, nil,
		connectionMgr,
		encryptionKey,
		logger,
	)

	// Caller context already canceled (simulates HTTP timeout before Execute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Execute must still reach the repository despite the canceled parent ctx
	_, err := service.Execute(ctx, changePlanID, uuid.New())
	assert.ErrorIs(t, err, ErrChangePlanNotFound)
	changePlanRepo.AssertExpectations(t)
}
