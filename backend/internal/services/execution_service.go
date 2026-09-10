// Package services provides business logic services for the Deeb application.
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/parser"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/utils"
	"go.uber.org/zap"
)

// ConnectionManagerInterface defines the interface for database connection management
type ConnectionManagerInterface interface {
	Connect(ctx context.Context, params database.ConnectionParams) (*sqlx.DB, error)
}

// ExecutionService provides SQL execution functionality with transaction management
type ExecutionService struct {
	changePlanRepo       repository.ChangePlanRepository
	dbInstanceRepo       repository.DatabaseInstanceRepository
	databaseRepo         repository.DatabaseRepository
	changelogRepo        repository.ChangelogRepository
	schemaHistoryRepo    repository.SchemaHistoryRepository
	schemaVersionService *SchemaVersionService
	lockManager          ExecutionLockManager
	queueService         *ExecutionQueueService
	executionLogRepo     repository.ExecutionLogRepository
	statementResultRepo  repository.StatementExecutionResultRepository
	connectionMgr        ConnectionManagerInterface
	encryptionKey        string
	executionTimeout     time.Duration // Timeout for SQL execution (default: 10 minutes)
	maxStatementCount    int           // Maximum number of statements allowed (default: 1000)
	logger               *zap.Logger

	// Optional ghost executor support (nil when unsupported)
	ghostExecutor *GhostExecutor
	envRepo       repository.EnvironmentRepository
}

// ExecutionResult represents the result of a SQL execution
type ExecutionResult struct {
	Success      bool
	RowsAffected int
	Duration     time.Duration
	ErrorMessage *string
}

// NewExecutionService creates a new execution service
func NewExecutionService(
	changePlanRepo repository.ChangePlanRepository,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	changelogRepo repository.ChangelogRepository,
	schemaHistoryRepo repository.SchemaHistoryRepository,
	schemaVersionService *SchemaVersionService,
	lockManager ExecutionLockManager,
	queueService *ExecutionQueueService,
	executionLogRepo repository.ExecutionLogRepository,
	statementResultRepo repository.StatementExecutionResultRepository,
	connectionMgr ConnectionManagerInterface,
	encryptionKey string,
	logger *zap.Logger,
) *ExecutionService {
	executionTimeout := 10 * time.Minute // Default: 10 minutes
	maxStatementCount := 1000            // Default: 1000 statements max

	return &ExecutionService{
		changePlanRepo:       changePlanRepo,
		dbInstanceRepo:       dbInstanceRepo,
		databaseRepo:         databaseRepo,
		changelogRepo:        changelogRepo,
		schemaHistoryRepo:    schemaHistoryRepo,
		schemaVersionService: schemaVersionService,
		lockManager:          lockManager,
		queueService:         queueService,
		executionLogRepo:     executionLogRepo,
		statementResultRepo:  statementResultRepo,
		connectionMgr:        connectionMgr,
		encryptionKey:        encryptionKey,
		executionTimeout:     executionTimeout,
		maxStatementCount:    maxStatementCount,
		logger:               logger,
	}
}

// WithGhostSupport attaches the ghost executor + environment repository for
// executor dispatch (optional; only used when set).
func (s *ExecutionService) WithGhostSupport(
	ghostExecutor *GhostExecutor,
	envRepo repository.EnvironmentRepository,
) *ExecutionService {
	s.ghostExecutor = ghostExecutor
	s.envRepo = envRepo
	return s
}

// GhostMaxDuration exposes the configured ghost executor max duration (0 when
// ghost support isn't configured). Used by the queue processor to size the
// per-item timeout window.
func (s *ExecutionService) GhostMaxDuration() time.Duration {
	if s.ghostExecutor == nil {
		return 0
	}
	return s.ghostExecutor.maxDuration
}

// ApproveGhostCutover removes the ghost postpone flag for a plan that is
// AwaitingCutover, letting the running tool proceed to the atomic rename.
func (s *ExecutionService) ApproveGhostCutover(ctx context.Context, changePlanID uuid.UUID) error {
	if s.ghostExecutor == nil {
		return ErrGhostUnsupported
	}
	plan, err := s.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		return err
	}
	if plan.ExecutionBackend != models.ExecutionBackendGhost {
		return fmt.Errorf("change plan is not running under the ghost executor")
	}
	if plan.Status != models.ChangePlanStatusAwaitingCutover {
		return fmt.Errorf("cutover can only be approved while the plan is AwaitingCutover (current: %s)", plan.Status)
	}
	return s.ghostExecutor.ApproveCutover(changePlanID)
}

// AbortGhostExecution creates the panic flag asking the ghost tool to abort
// cleanly before the cutover; the original table stays untouched.
func (s *ExecutionService) AbortGhostExecution(ctx context.Context, changePlanID uuid.UUID) error {
	if s.ghostExecutor == nil {
		return ErrGhostUnsupported
	}
	plan, err := s.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		return err
	}
	if plan.ExecutionBackend != models.ExecutionBackendGhost {
		return fmt.Errorf("change plan is not running under the ghost executor")
	}
	if plan.Status != models.ChangePlanStatusInProgress && plan.Status != models.ChangePlanStatusAwaitingCutover {
		return fmt.Errorf("abort is only available while the plan is executing (current: %s)", plan.Status)
	}
	return s.ghostExecutor.Abort(changePlanID)
}

// GhostCheckResult is one item in the ghost readiness report.
type GhostCheckResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// ghostCheckName enumerates the individual checks for targeted re-runs.
const (
	GhostCheckConnection  = "connection"
	GhostCheckEngine      = "engine_mysql"
	GhostCheckBinlogFmt   = "binlog_format"
	GhostCheckBinlogImage = "binlog_row_image"
	GhostCheckPrivileges  = "privileges"
	GhostCheckDisk        = "disk_headroom_estimate"
	GhostCheckDryRun      = "dry_run"
)

// ghostCheckRunner is a single check executable independently.
type ghostCheckRunner func(ctx context.Context, plan *models.ChangePlan) GhostCheckResult

// ghostCheckFuncs maps check names to their runner for targeted re-runs.
// RunGhostReadinessChecks uses the same functions in a fixed order.
func (s *ExecutionService) ghostCheckFuncs() map[string]ghostCheckRunner {
	return map[string]ghostCheckRunner{
		GhostCheckConnection:  s.checkGhostConnection,
		GhostCheckEngine:      s.checkGhostEngine,
		GhostCheckBinlogFmt:   s.checkGhostBinlogFormat,
		GhostCheckBinlogImage: s.checkGhostBinlogRowImage,
		GhostCheckPrivileges:  s.checkGhostPrivileges,
		GhostCheckDisk:        s.checkGhostDiskHeadroom,
		GhostCheckDryRun:      s.checkGhostDryRun,
	}
}

// GhostReadinessReport is the structured result of all ghost checks, run at
// submit-for-review time. AllPassed gates the Draft → Open transition.
type GhostReadinessReport struct {
	AllPassed bool               `json:"all_passed"`
	Checks    []GhostCheckResult `json:"checks"`
}

// RunGhostReadinessChecks executes the full ghost readiness suite against the
// live target and returns per-check results. Used by the submit-for-review
// gate and the preflight endpoint. Dry run is included as the final check.
// ghostCheckOrder fixes the report's check sequence.
var ghostCheckOrder = []string{
	GhostCheckEngine,
	GhostCheckConnection,
	GhostCheckBinlogFmt,
	GhostCheckBinlogImage,
	GhostCheckPrivileges,
	GhostCheckDisk,
	GhostCheckDryRun,
}

func (s *ExecutionService) RunGhostReadinessChecks(ctx context.Context, plan *models.ChangePlan) (*GhostReadinessReport, error) {
	funcs := s.ghostCheckFuncs()
	report := &GhostReadinessReport{AllPassed: true}
	for _, name := range ghostCheckOrder {
		result := funcs[name](ctx, plan)
		report.Checks = append(report.Checks, result)
		if !result.Passed {
			report.AllPassed = false
			if name == GhostCheckEngine || name == GhostCheckConnection {
				break // no point continuing without a valid target
			}
		}
	}
	return report, nil
}

// RunGhostCheck executes a single named check; unknown names return an
// informative failure result (callers validate names first).
func (s *ExecutionService) RunGhostCheck(ctx context.Context, plan *models.ChangePlan, checkName string) (GhostCheckResult, error) {
	funcs := s.ghostCheckFuncs()
	runner, ok := funcs[checkName]
	if !ok {
		return GhostCheckResult{}, fmt.Errorf("unknown ghost check: %s (known: %s)", checkName, strings.Join(ghostCheckOrder, ", "))
	}
	return runner(ctx, plan), nil
}

// checkGhostEngine verifies the target engine is MySQL.
func (s *ExecutionService) checkGhostEngine(ctx context.Context, plan *models.ChangePlan) GhostCheckResult {
	_, dbInstance, _, _, err := s.resolveGhostTarget(ctx, plan)
	if err != nil {
		return GhostCheckResult{Name: GhostCheckEngine, Passed: false, Detail: err.Error()}
	}
	if dbInstance.DatabaseEngine == models.DatabaseEngineMySQL {
		return GhostCheckResult{Name: GhostCheckEngine, Passed: true, Detail: "target is MySQL"}
	}
	return GhostCheckResult{Name: GhostCheckEngine, Passed: false,
		Detail: fmt.Sprintf("ghost currently supports MySQL only (target: %s)", dbInstance.DatabaseEngine)}
}

// checkGhostConnection verifies the target is reachable.
func (s *ExecutionService) checkGhostConnection(ctx context.Context, plan *models.ChangePlan) GhostCheckResult {
	db, dbInstance := s.mustResolveForConn(ctx, plan)
	if db == nil {
		return GhostCheckResult{Name: GhostCheckConnection, Passed: false, Detail: "failed to resolve target database"}
	}
	conn, err := s.connectionMgr.Connect(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		return GhostCheckResult{Name: GhostCheckConnection, Passed: false,
			Detail: fmt.Sprintf("failed to connect: %s", err.Error())}
	}
	_ = conn.Close()
	return GhostCheckResult{Name: GhostCheckConnection, Passed: true, Detail: "connected to target instance"}
}

// checkGhostBinlogFormat verifies binlog_format=ROW.
func (s *ExecutionService) checkGhostBinlogFormat(ctx context.Context, plan *models.ChangePlan) GhostCheckResult {
	return s.ghostVariableCheck(ctx, plan, GhostCheckBinlogFmt, "binlog_format", "ROW")
}

// checkGhostBinlogRowImage verifies binlog_row_image=FULL.
func (s *ExecutionService) checkGhostBinlogRowImage(ctx context.Context, plan *models.ChangePlan) GhostCheckResult {
	return s.ghostVariableCheck(ctx, plan, GhostCheckBinlogImage, "binlog_row_image", "FULL")
}

func (s *ExecutionService) ghostVariableCheck(ctx context.Context, plan *models.ChangePlan, checkName, variable, want string) GhostCheckResult {
	db, dbInstance, _, _, err := s.resolveGhostTarget(ctx, plan)
	if err != nil {
		return GhostCheckResult{Name: checkName, Passed: false, Detail: err.Error()}
	}
	conn, err := s.connectionMgr.Connect(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		return GhostCheckResult{Name: checkName, Passed: false, Detail: fmt.Sprintf("failed to connect: %s", err.Error())}
	}
	defer func() { _ = conn.Close() }()

	var name, value string
	if err := conn.QueryRowContext(ctx, "SHOW VARIABLES LIKE '"+variable+"'").Scan(&name, &value); err != nil {
		return GhostCheckResult{Name: checkName, Passed: false, Detail: err.Error()}
	}
	if !strings.EqualFold(value, want) {
		return GhostCheckResult{Name: checkName, Passed: false,
			Detail: fmt.Sprintf("gh-ost requires %s=%s (current: %s); check the RDS parameter group", variable, want, value)}
	}
	return GhostCheckResult{Name: checkName, Passed: true, Detail: fmt.Sprintf("%s=%s", variable, want)}
}

// checkGhostPrivileges verifies grants needed by gh-ost.
func (s *ExecutionService) checkGhostPrivileges(ctx context.Context, plan *models.ChangePlan) GhostCheckResult {
	db, dbInstance, _, _, err := s.resolveGhostTarget(ctx, plan)
	if err != nil {
		return GhostCheckResult{Name: GhostCheckPrivileges, Passed: false, Detail: err.Error()}
	}
	conn, err := s.connectionMgr.Connect(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		return GhostCheckResult{Name: GhostCheckPrivileges, Passed: false, Detail: fmt.Sprintf("failed to connect: %s", err.Error())}
	}
	defer func() { _ = conn.Close() }()

	rows, qErr := conn.QueryContext(ctx, "SHOW GRANTS")
	if qErr != nil {
		return GhostCheckResult{Name: GhostCheckPrivileges, Passed: false, Detail: "could not read grants"}
	}
	var all string
	for rows.Next() {
		var g string
		if scanErr := rows.Scan(&g); scanErr == nil {
			all += " " + strings.ToUpper(g)
		}
	}
	_ = rows.Close()
	hasReplication := strings.Contains(all, "REPLICATION SLAVE") && strings.Contains(all, "REPLICATION CLIENT")
	dbNameEsc := strings.ToUpper(db.DatabaseName)
	hasDBPrivs := strings.Contains(all, "ALL PRIVILEGES") &&
		(strings.Contains(all, "ON *.*") || strings.Contains(all, "ON `"+dbNameEsc+"`.*") || strings.Contains(all, "ON "+dbNameEsc+".*"))
	passed := hasReplication && hasDBPrivs
	detail := "grants OK"
	if !passed {
		detail = fmt.Sprintf("required: ALL PRIVILEGES ON %s.* plus REPLICATION SLAVE, REPLICATION CLIENT ON *.*", db.DatabaseName)
	}
	return GhostCheckResult{Name: GhostCheckPrivileges, Passed: passed, Detail: detail}
}

// checkGhostDiskHeadroom estimates free-disk need (always passes — operator verifies).
func (s *ExecutionService) checkGhostDiskHeadroom(ctx context.Context, plan *models.ChangePlan) GhostCheckResult {
	db, dbInstance, table, _, err := s.resolveGhostTarget(ctx, plan)
	if err != nil {
		return GhostCheckResult{Name: GhostCheckDisk, Passed: false, Detail: err.Error()}
	}
	conn, err := s.connectionMgr.Connect(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		return GhostCheckResult{Name: GhostCheckDisk, Passed: false, Detail: fmt.Sprintf("failed to connect: %s", err.Error())}
	}
	defer func() { _ = conn.Close() }()

	var dataLen, indexLen sql.NullInt64
	_ = conn.QueryRowContext(ctx,
		`SELECT DATA_LENGTH, INDEX_LENGTH FROM information_schema.TABLES
		 WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`, db.DatabaseName, table,
	).Scan(&dataLen, &indexLen)
	totalBytes := dataLen.Int64 + indexLen.Int64
	estGB := float64(totalBytes) / (1024 * 1024 * 1024)
	return GhostCheckResult{Name: GhostCheckDisk, Passed: true,
		Detail: fmt.Sprintf("table %s is %.2f GB (data+indexes); gh-ost needs roughly that much free disk again plus binlog growth — verify free disk on the instance before executing", table, estGB)}
}

// checkGhostDryRun runs gh-ost without --execute against the live target.
func (s *ExecutionService) checkGhostDryRun(ctx context.Context, plan *models.ChangePlan) GhostCheckResult {
	db, dbInstance, table, password, err := s.resolveGhostTarget(ctx, plan)
	if err != nil {
		return GhostCheckResult{Name: GhostCheckDryRun, Passed: false, Detail: err.Error()}
	}
	if err := s.ghostDryRunWithCreds(ctx, plan, db, dbInstance, table, password); err != nil {
		return GhostCheckResult{Name: GhostCheckDryRun, Passed: false, Detail: err.Error()}
	}
	return GhostCheckResult{Name: GhostCheckDryRun, Passed: true, Detail: "gh-ost dry run completed successfully"}
}

// mustResolveForConn resolves database+instance, returning nils on failure.
func (s *ExecutionService) mustResolveForConn(ctx context.Context, plan *models.ChangePlan) (*models.Database, *models.DatabaseInstance) {
	db, dbInstance, _, _, err := s.resolveGhostTarget(ctx, plan)
	if err != nil {
		return nil, nil
	}
	return db, dbInstance
}

// ghostDryRunWithCreds is the shared dry-run implementation (avoids
// re-resolving the target when the caller already has it).
func (s *ExecutionService) ghostDryRunWithCreds(
	ctx context.Context,
	plan *models.ChangePlan,
	db *models.Database,
	dbInstance *models.DatabaseInstance,
	table, password string,
) error {
	_, alterClause, err := ParseAlterStatement(plan.SQLContent)
	if err != nil {
		return fmt.Errorf("invalid ghost SQL: %w", err)
	}
	binPath, args, err := s.ghostExecutor.BuildArgs(
		dbInstance.DatabaseEngine,
		dbInstance.Host,
		dbInstance.Username,
		password,
		db.DatabaseName,
		table,
		alterClause,
		dbInstance.Port,
		"", "",
	)
	if err != nil {
		return err
	}
	dryArgs := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--execute" {
			continue
		}
		dryArgs = append(dryArgs, a)
	}
	var lastErr string
	callbacks := GhostRunCallbacks{
		OnLogLine: func(line string) {
			s.logger.Info("ghost-dryrun",
				zap.String("change_plan_id", plan.ID.String()),
				zap.String("line", line),
			)
			l := strings.ToLower(line)
			if strings.Contains(l, "error") || strings.Contains(l, "fatal") {
				lastErr = line
			}
		},
	}
	if err := s.ghostExecutor.Run(ctx, binPath, dryArgs, callbacks); err != nil {
		if lastErr != "" {
			return fmt.Errorf("ghost dry-run failed: %s", lastErr)
		}
		return fmt.Errorf("ghost dry-run failed: %w", err)
	}
	return nil
}

// GhostDryRun validates a ghost plan by running gh-ost WITHOUT --execute
// (dry run): it checks privileges, binlog access, and ALTER syntax against
// the live target without migrating. Called at approval time so plans with
// misconfigured targets fail before they can be approved.
func (s *ExecutionService) GhostDryRun(ctx context.Context, changePlanID uuid.UUID) error {
	if s.ghostExecutor == nil {
		return ErrGhostUnsupported
	}
	plan, err := s.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		return err
	}
	return s.GhostDryRunForPlan(ctx, plan)
}

// GhostDryRunForPlan runs the gh-ost dry run for an in-memory plan (used at
// create/update time before the plan is persisted, and at approval time).
func (s *ExecutionService) GhostDryRunForPlan(ctx context.Context, plan *models.ChangePlan) error {
	if s.ghostExecutor == nil {
		return ErrGhostUnsupported
	}
	db, dbInstance, table, password, err := s.resolveGhostTarget(ctx, plan)
	if err != nil {
		return err
	}
	return s.ghostDryRunWithCreds(ctx, plan, db, dbInstance, table, password)
}

// resolveGhostTarget resolves the plan's target database, instance, parsed
// table name, and decrypted password for post-completion ghost operations.
func (s *ExecutionService) resolveGhostTarget(
	ctx context.Context,
	plan *models.ChangePlan,
) (*models.Database, *models.DatabaseInstance, string, string, error) {
	if plan.ExecutionBackend != models.ExecutionBackendGhost {
		return nil, nil, "", "", fmt.Errorf("change plan is not running under the ghost executor")
	}
	table, _, err := ParseAlterStatement(plan.SQLContent)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("invalid ghost SQL: %w", err)
	}
	db, err := s.databaseRepo.GetByID(ctx, plan.TargetDatabaseID)
	if err != nil {
		return nil, nil, "", "", err
	}
	dbInstance, err := s.dbInstanceRepo.GetByID(ctx, db.DatabaseInstanceID)
	if err != nil {
		return nil, nil, "", "", err
	}
	password, err := utils.DecryptPassword(dbInstance.ConnectionDetailsEncrypted, s.encryptionKey)
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("failed to decrypt instance password: %w", err)
	}
	_ = password // retained for callers that build raw tool invocations
	return db, dbInstance, table, password, nil
}

// GhostBackupTableName returns the backup table gh-ost retains post-cutover
// (_<table>_del) for a completed ghost plan, verifying it exists.
func (s *ExecutionService) GhostBackupTableName(ctx context.Context, changePlanID uuid.UUID) (string, error) {
	plan, err := s.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		return "", err
	}
	_, _, table, _, err := s.resolveGhostTarget(ctx, plan)
	if err != nil {
		return "", err
	}
	return "_" + table + "_del", nil
}

// DropGhostBackupTable drops the retained _<table>_del backup table after a
// completed ghost migration (cleanup step once the validation window passes).
// Only allowed on Completed ghost plans; the drop runs outside a transaction
// (DDL) and is audited via the execution log message.
func (s *ExecutionService) DropGhostBackupTable(ctx context.Context, changePlanID uuid.UUID) error {
	plan, err := s.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		return err
	}
	if plan.Status != models.ChangePlanStatusCompleted {
		return fmt.Errorf("backup table can only be dropped after the plan is Completed (current: %s)", plan.Status)
	}
	db, dbInstance, table, _, err := s.resolveGhostTarget(ctx, plan)
	if err != nil {
		return err
	}
	backup := "_" + table + "_del"

	conn, err := s.connectionMgr.Connect(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS `%s`", backup)); err != nil {
		return fmt.Errorf("failed to drop backup table %s: %w", backup, err)
	}
	s.logger.Info("Ghost backup table dropped",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("backup_table", backup),
	)
	return nil
}

// RollbackGhostPlan atomically swaps the backup table back into place after a
// completed ghost migration: RENAME TABLE <table> -> _<table>_new,
// _<table>_del -> <table>. WARNING: writes since cutover live only in the
// current table (preserved as _<table>_new for reconciliation); the caller
// must surface this data-loss risk to the user.
func (s *ExecutionService) RollbackGhostPlan(ctx context.Context, changePlanID uuid.UUID) error {
	plan, err := s.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		return err
	}
	if plan.Status != models.ChangePlanStatusCompleted {
		return fmt.Errorf("rollback is only available for Completed ghost plans (current: %s)", plan.Status)
	}
	db, dbInstance, table, _, err := s.resolveGhostTarget(ctx, plan)
	if err != nil {
		return err
	}
	backup := "_" + table + "_del"
	sidelined := "_" + table + "_new"

	conn, err := s.connectionMgr.Connect(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to target database: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Atomic multi-table rename (ghost-lab 08-rollback.sh equivalent)
	rename := fmt.Sprintf("RENAME TABLE `%s` TO `%s`, `%s` TO `%s`", table, sidelined, backup, table)
	if _, err := conn.ExecContext(ctx, rename); err != nil {
		return fmt.Errorf("rollback rename failed: %w", err)
	}

	// Mark the plan as Failed with a rollback note so the UI reflects the
	// schema was reverted (Completed stays immutable history otherwise).
	plan.Status = models.ChangePlanStatusFailed
	msg := fmt.Sprintf("Rolled back via ghost swap-back; post-cutover writes preserved in %s", sidelined)
	plan.ErrorMessage = &msg
	plan.UpdatedAt = time.Now()
	if err := s.changePlanRepo.Update(ctx, plan); err != nil {
		s.logger.Error("Failed to update plan status after rollback", zap.Error(err))
	}

	s.logger.Info("Ghost plan rolled back",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("table", table),
		zap.String("sidelined_table", sidelined),
	)
	return nil
}

// Executor dispatch errors
var (
	ErrGhostUnsupported = fmt.Errorf("ghost executor not configured")
	ErrGhostRejectedEnv = fmt.Errorf("ghost executor not permitted on production environments")
)

// NewExecutionServiceWithTimeout creates a new execution service with custom timeout
func NewExecutionServiceWithTimeout(
	changePlanRepo repository.ChangePlanRepository,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	changelogRepo repository.ChangelogRepository,
	schemaHistoryRepo repository.SchemaHistoryRepository,
	schemaVersionService *SchemaVersionService,
	lockManager ExecutionLockManager,
	queueService *ExecutionQueueService,
	executionLogRepo repository.ExecutionLogRepository,
	statementResultRepo repository.StatementExecutionResultRepository,
	connectionMgr ConnectionManagerInterface,
	encryptionKey string,
	executionTimeout time.Duration,
	logger *zap.Logger,
) *ExecutionService {
	maxStatementCount := 1000 // Default: 1000 statements max

	return &ExecutionService{
		changePlanRepo:       changePlanRepo,
		dbInstanceRepo:       dbInstanceRepo,
		databaseRepo:         databaseRepo,
		changelogRepo:        changelogRepo,
		schemaHistoryRepo:    schemaHistoryRepo,
		schemaVersionService: schemaVersionService,
		lockManager:          lockManager,
		queueService:         queueService,
		executionLogRepo:     executionLogRepo,
		statementResultRepo:  statementResultRepo,
		connectionMgr:        connectionMgr,
		encryptionKey:        encryptionKey,
		executionTimeout:     executionTimeout,
		maxStatementCount:    maxStatementCount,
		logger:               logger,
	}
}

// Execute executes SQL statements from a change plan within a SERIALIZABLE transaction
// Implements complete workflow: lock → snapshot → execute → snapshot → update status → unlock
func (s *ExecutionService) Execute(
	ctx context.Context,
	changePlanID uuid.UUID,
	executorID uuid.UUID,
) (*ExecutionResult, error) {
	// Log execution start
	overallStartTime := time.Now()
	s.logger.Info("Execution started",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("executor_id", executorID.String()),
	)

	// Detach execution from the caller's context (typically the HTTP request).
	// Database changes legitimately outlive request timeouts (e.g. the 30s
	// server write timeout); if the request context is canceled mid-run,
	// cleanup steps (lock release, changelog/execution-log updates) must
	// still complete. WithoutCancel keeps values (correlation IDs) but
	// removes the deadline/cancel propagation.
	ctx = context.WithoutCancel(ctx)

	// Get change plan
	changePlan, err := s.changePlanRepo.GetByID(ctx, changePlanID)
	if err != nil {
		if err == repository.ErrNotFound {
			s.logger.Error("Change plan not found",
				zap.String("change_plan_id", changePlanID.String()),
			)
			return nil, ErrChangePlanNotFound
		}
		s.logger.Error("Failed to get change plan",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get change plan: %w", err)
	}

	// Validate Change Plan is in Approved status
	if changePlan.Status != models.ChangePlanStatusApproved {
		s.logger.Error("Change plan not in Approved status",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("current_status", string(changePlan.Status)),
		)
		return nil, fmt.Errorf("change plan must be in Approved status to execute, current status: %s", changePlan.Status)
	}

	// Ghost executor dispatch: defense-in-depth validation happens again in
	// ghost executor; when requested by user preference we use it or fail.
	if changePlan.ExecutionBackend == models.ExecutionBackendGhost {
		if s.ghostExecutor == nil {
			return nil, ErrGhostUnsupported
		}
		if err := s.validateGhostPlan(ctx, changePlan); err != nil {
			return nil, err
		}
		// Ghost run handled by a dedicated implementation below; direct path untouched.
		return s.executeGhost(ctx, changePlan, executorID)
	}

	// Get database and database instance
	db, err := s.databaseRepo.GetByID(ctx, changePlan.TargetDatabaseID)
	if err != nil {
		if err == repository.ErrNotFound {
			s.logger.Error("Target database not found",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", changePlan.TargetDatabaseID.String()),
			)
			return nil, ErrDatabaseNotFound
		}
		s.logger.Error("Failed to get database",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", changePlan.TargetDatabaseID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	dbInstance, err := s.dbInstanceRepo.GetByID(ctx, db.DatabaseInstanceID)
	if err != nil {
		if err == repository.ErrNotFound {
			s.logger.Error("Database instance not found",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_instance_id", db.DatabaseInstanceID.String()),
			)
			return nil, ErrDatabaseInstanceNotFound
		}
		s.logger.Error("Failed to get database instance",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_instance_id", db.DatabaseInstanceID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Log database context for monitoring
	sqlContentLength := len(changePlan.SQLContent)
	s.logger.Info("Execution context loaded",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_name", db.DatabaseName),
		zap.String("database_instance_id", dbInstance.ID.String()),
		zap.String("database_instance_name", dbInstance.Name),
		zap.String("database_engine", string(dbInstance.DatabaseEngine)),
		zap.Int("sql_content_length", sqlContentLength),
	)

	// Step 0: Acquire execution lock for database
	lockStartTime := time.Now()
	if err := s.lockManager.AcquireLock(ctx, db.ID, changePlanID, executorID); err != nil {
		if err == ErrDatabaseLocked {
			// Database is locked - enqueue the Change Plan
			if s.queueService != nil {
				queueItem, enqueueErr := s.queueService.Enqueue(ctx, changePlanID, db.ID, executorID)
				if enqueueErr != nil {
					s.logger.Error("Failed to enqueue change plan",
						zap.String("change_plan_id", changePlanID.String()),
						zap.String("database_id", db.ID.String()),
						zap.Error(enqueueErr),
					)
					return nil, fmt.Errorf("database locked and failed to enqueue: %w", enqueueErr)
				}
				s.logger.Info("Change plan enqueued due to database lock",
					zap.String("change_plan_id", changePlanID.String()),
					zap.String("database_id", db.ID.String()),
					zap.Int("position", queueItem.Position),
				)
				return nil, ErrDatabaseLocked // Return lock error so caller knows it was queued
			}
			// No queue service available - return lock error
			return nil, ErrDatabaseLocked
		}
		s.logger.Error("Failed to acquire execution lock",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", db.ID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to acquire execution lock: %w", err)
	}
	lockDuration := time.Since(lockStartTime)
	s.logger.Info("Execution lock acquired",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.Duration("lock_acquisition_duration", lockDuration),
	)

	// Guarantee lock release on exit (success or failure)
	defer func() {
		if releaseErr := s.lockManager.ReleaseLock(ctx, db.ID); releaseErr != nil {
			s.logger.Error("Failed to release execution lock",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
				zap.Error(releaseErr),
			)
		} else {
			s.logger.Debug("Execution lock released",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
			)
		}
	}()

	// Step 1: Update status to InProgress
	changePlan.Status = models.ChangePlanStatusInProgress
	changePlan.UpdatedAt = time.Now()
	if err := s.changePlanRepo.Update(ctx, changePlan); err != nil {
		s.logger.Error("Failed to update change plan status to InProgress",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to update change plan status to InProgress: %w", err)
	}
	s.logger.Info("Change plan status updated to InProgress",
		zap.String("change_plan_id", changePlanID.String()),
	)

	// Step 2: Get latest schema version and create changelog entry
	var prevSchemaHistoryID *uuid.UUID
	if db.Status != models.DatabaseStatusPending {
		// Get the latest schema version for this database (if exists)
		latestSchema, err := s.schemaVersionService.GetLatestSchemaVersion(ctx, db.ID)
		if err != nil {
			s.logger.Error("Failed to get latest schema version",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
				zap.Error(err),
			)
			// Update status to Failed
			changePlan.Status = models.ChangePlanStatusFailed
			errorMsg := fmt.Sprintf("failed to get latest schema version: %v", err)
			changePlan.ErrorMessage = &errorMsg
			changePlan.UpdatedAt = time.Now()
			if updateErr := s.changePlanRepo.Update(ctx, changePlan); updateErr != nil {
				s.logger.Error("Failed to update change plan status to Failed", zap.Error(updateErr))
			}
			return nil, fmt.Errorf("failed to get latest schema version: %w", err)
		}
		if latestSchema != nil {
			prevSchemaHistoryID = &latestSchema.ID
			s.logger.Info("Found previous schema version",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("prev_schema_history_id", prevSchemaHistoryID.String()),
			)
		}
	} else {
		s.logger.Info("No previous schema - database does not exist yet",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", db.ID.String()),
			zap.String("database_status", string(db.Status)),
		)
	}

	// Create changelog entry with status "pending"
	changelog := &models.Changelog{
		ID:                  uuid.New(),
		DatabaseInstanceID:  dbInstance.ID,
		DatabaseID:          db.ID,
		ChangePlanID:        changePlanID,
		Status:              models.ChangelogStatusPending,
		PrevSchemaHistoryID: prevSchemaHistoryID,
		SchemaHistoryID:     nil, // Will be set after execution
		CreatedAt:           time.Now(),
	}
	if err := s.changelogRepo.Create(ctx, changelog); err != nil {
		s.logger.Error("Failed to create changelog entry",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		// Update status to Failed
		changePlan.Status = models.ChangePlanStatusFailed
		errorMsg := fmt.Sprintf("failed to create changelog entry: %v", err)
		changePlan.ErrorMessage = &errorMsg
		changePlan.UpdatedAt = time.Now()
		if updateErr := s.changePlanRepo.Update(ctx, changePlan); updateErr != nil {
			s.logger.Error("Failed to update change plan status to Failed", zap.Error(updateErr))
		}
		return nil, fmt.Errorf("failed to create changelog entry: %w", err)
	}
	s.logger.Info("Changelog entry created",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("changelog_id", changelog.ID.String()),
		zap.String("status", string(changelog.Status)),
	)

	// Step 3: Create execution log entry (pre-execution)
	// We create this before execution so statement results can reference it via foreign key
	executionLogID := uuid.New()
	dbType := string(dbInstance.DatabaseEngine)
	sqlContent := changePlan.SQLContent
	executionLog := &models.ExecutionLog{
		ID:                   executionLogID,
		ChangePlanID:         changePlanID,
		ExecutorID:           executorID,
		Status:               models.ExecutionLogStatusFailed, // Will be updated to Completed on success
		SQLContent:           &sqlContent,
		DatabaseID:           &db.ID,
		DatabaseName:         &db.DatabaseName,
		DatabaseInstanceName: &dbInstance.Name,
		DatabaseEngine:       &dbType,
		StartedAt:            time.Now(),
	}

	// Persist execution log to database so statement results can reference it via foreign key
	if err := s.executionLogRepo.Create(ctx, executionLog); err != nil {
		s.logger.Error("Failed to create execution log entry",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create execution log: %w", err)
	}

	// Step 4: Execute SQL in transaction with timeout
	executionStartTime := time.Now()
	s.logger.Info("Starting SQL execution",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("execution_log_id", executionLogID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_engine", string(dbInstance.DatabaseEngine)),
		zap.Int("sql_content_length", sqlContentLength),
		zap.Duration("execution_timeout", s.executionTimeout),
	)

	// Create timeout context for SQL execution
	execCtx, cancel := context.WithTimeout(ctx, s.executionTimeout)
	defer cancel()

	result, err := s.executeInTransaction(execCtx, dbInstance, db, changePlan.SQLContent, changePlanID, executionLogID)
	duration := time.Since(executionStartTime)

	// Check if timeout occurred
	if err != nil && execCtx.Err() == context.DeadlineExceeded {
		s.logger.Error("Execution timeout exceeded",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Duration("timeout", s.executionTimeout),
			zap.Duration("actual_duration", duration),
		)
		err = fmt.Errorf("execution timeout after %v: %w", s.executionTimeout, err)
	}

	// Store execution results in Change Plan
	now := time.Now()
	changePlan.ExecutedAt = &now
	durationMs := int(duration.Milliseconds())
	changePlan.ExecutionDurationMs = &durationMs
	changePlan.UpdatedAt = time.Now()

	if err != nil {
		// Step 5: Execution failed - update status to Failed
		var errorMsg string
		var userFriendlyMsg string

		// Check if it's a PositionedError (from multi-statement execution)
		if posErr, ok := err.(*PositionedError); ok {
			userFriendlyMsg = posErr.Error()
			errorMsg = userFriendlyMsg

			s.logger.Error("SQL execution failed (positioned)",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
				zap.Int("statement_index", posErr.StatementIndex),
				zap.Int("line", posErr.Line),
				zap.Int("column", posErr.Column),
				zap.String("user_message", userFriendlyMsg),
				zap.Duration("execution_duration", duration),
			)
		} else if sqlErr, ok := err.(*SQLExecutionError); ok {
			// Check if it's a SQLExecutionError for enhanced handling
			// Use the user-friendly message from SQLExecutionError
			userFriendlyMsg = sqlErr.Message
			errorMsg = userFriendlyMsg

			// Enhanced error context logging (already logged in executeInTransaction, but log summary here)
			s.logger.Error("SQL execution failed (summary)",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
				zap.String("database_name", db.DatabaseName),
				zap.String("database_instance_name", dbInstance.Name),
				zap.String("database_engine", string(dbInstance.DatabaseEngine)),
				zap.String("database_error_code", sqlErr.DatabaseErrorCode),
				zap.String("user_message", userFriendlyMsg),
				zap.Duration("execution_duration", duration),
				zap.Int("sql_content_length", sqlContentLength),
			)
		} else {
			// Generic error handling
			errorMsg = err.Error()
			userFriendlyMsg = errorMsg

			// Enhanced error context logging
			sqlPreview := changePlan.SQLContent
			if len(sqlPreview) > 200 {
				sqlPreview = sqlPreview[:200] + "... (truncated)"
			}
			s.logger.Error("SQL execution failed",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
				zap.String("database_name", db.DatabaseName),
				zap.String("database_instance_name", dbInstance.Name),
				zap.String("database_engine", string(dbInstance.DatabaseEngine)),
				zap.Error(err),
				zap.Duration("execution_duration", duration),
				zap.Int("sql_content_length", sqlContentLength),
				zap.String("sql_preview", sqlPreview),
			)
		}

		changePlan.Status = models.ChangePlanStatusFailed
		changePlan.ErrorMessage = &errorMsg
		changePlan.AffectedRows = nil // No rows affected on error

		// Update Change Plan with error results
		if updateErr := s.changePlanRepo.Update(ctx, changePlan); updateErr != nil {
			s.logger.Error("Failed to update change plan with execution results", zap.Error(updateErr))
			// Don't fail the whole operation if update fails, but log it
		}

		// Update changelog status to "failed"
		changelog.Status = models.ChangelogStatusFailed
		if updateErr := s.changelogRepo.Update(ctx, changelog); updateErr != nil {
			s.logger.Error("Failed to update changelog status to failed",
				zap.String("changelog_id", changelog.ID.String()),
				zap.Error(updateErr),
			)
		} else {
			s.logger.Info("Changelog status updated to failed",
				zap.String("changelog_id", changelog.ID.String()),
			)
		}

		// Update execution log entry for failed execution
		if s.executionLogRepo != nil {
			executionLog.Status = models.ExecutionLogStatusFailed
			executionLog.ExecutionDurationMs = int(duration.Milliseconds())
			executionLog.ErrorMessage = &errorMsg
			executionLog.CompletedAt = time.Now()

			if logErr := s.executionLogRepo.Update(ctx, executionLog); logErr != nil {
				s.logger.Warn("Failed to update execution log entry", zap.Error(logErr))
			}
		}

		return &ExecutionResult{
			Success:      false,
			RowsAffected: 0,
			Duration:     duration,
			ErrorMessage: &userFriendlyMsg, // Use user-friendly message
		}, nil // Return result with error, don't propagate error
	}

	// Step 5: Execution succeeded - capture new schema version
	s.logger.Info("SQL execution completed successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.Int("rows_affected", result.RowsAffected),
		zap.Duration("execution_duration", duration),
	)

	schemaVersionStartTime := time.Now()
	s.logger.Info("Capturing schema version",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_engine", string(dbInstance.DatabaseEngine)),
	)

	// Build changelog metadata
	changelogMeta := models.ChangelogMetadata{}
	var schemaVersionDuration time.Duration

	schemaHistory, err := s.schemaVersionService.CaptureSchemaVersion(ctx, db.ID)
	if err != nil {
		s.logger.Warn("Failed to capture schema snapshot - execution will continue",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		// Record snapshot failure in metadata
		failedStatus := models.SnapshotStatusFailed
		errorMsg := err.Error()
		changelogMeta.SnapshotStatus = &failedStatus
		changelogMeta.SnapshotError = &errorMsg
		schemaVersionDuration = time.Since(schemaVersionStartTime)
	} else {
		schemaVersionDuration = time.Since(schemaVersionStartTime)
		s.logger.Info("Schema snapshot captured successfully",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("schema_history_id", schemaHistory.ID.String()),
			zap.Duration("schema_capture_duration", schemaVersionDuration),
		)
		// Record snapshot success in metadata
		successStatus := models.SnapshotStatusSuccess
		changelogMeta.SnapshotStatus = &successStatus
		changelog.SchemaHistoryID = &schemaHistory.ID
	}

	// Serialize metadata to JSON
	metadataJSON, err := json.Marshal(changelogMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize changelog metadata: %w", err)
	}
	changelog.Metadata = metadataJSON

	// Update changelog status to "done" and metadata
	changelog.Status = models.ChangelogStatusDone
	if updateErr := s.changelogRepo.Update(ctx, changelog); updateErr != nil {
		s.logger.Error("Failed to update changelog status to done",
			zap.String("changelog_id", changelog.ID.String()),
			zap.Error(updateErr),
		)
		// Continue anyway - the schema was captured successfully
	} else {
		s.logger.Info("Changelog status updated to done",
			zap.String("changelog_id", changelog.ID.String()),
			zap.String("schema_history_id", schemaHistory.ID.String()),
		)
	}

	// Step 6: Update status to Completed
	changePlan.Status = models.ChangePlanStatusCompleted
	changePlan.AffectedRows = &result.RowsAffected
	changePlan.ErrorMessage = nil // Clear any previous error message
	changePlan.UpdatedAt = time.Now()

	// Update Change Plan with execution results
	if err := s.changePlanRepo.Update(ctx, changePlan); err != nil {
		s.logger.Error("Failed to update change plan with execution results",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to update change plan with execution results: %w", err)
	}

	// Step 7: Update database status from Pending to Active if this was a CREATE DATABASE operation
	if db.Status == models.DatabaseStatusPending {
		db.Status = models.DatabaseStatusActive
		db.UpdatedAt = time.Now()
		if err := s.databaseRepo.Update(ctx, db); err != nil {
			s.logger.Error("Failed to update database status to Active",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_id", db.ID.String()),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to update database status to Active: %w", err)
		}
		s.logger.Info("Database status updated to Active",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", db.ID.String()),
		)
	}

	overallDuration := time.Since(overallStartTime)
	s.logger.Info("Execution workflow completed successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", db.ID.String()),
		zap.String("database_name", db.DatabaseName),
		zap.String("database_instance_name", dbInstance.Name),
		zap.String("database_engine", string(dbInstance.DatabaseEngine)),
		zap.Int("rows_affected", result.RowsAffected),
		zap.Duration("sql_execution_duration", duration),
		zap.Duration("schema_capture_duration", schemaVersionDuration),
		zap.Duration("lock_acquisition_duration", lockDuration),
		zap.Duration("total_execution_duration", overallDuration),
	)

	// Update execution log entry for successful execution
	if s.executionLogRepo != nil {
		rowsAffected := result.RowsAffected
		executionLog.Status = models.ExecutionLogStatusCompleted
		executionLog.ExecutionDurationMs = int(duration.Milliseconds())
		executionLog.RowsAffected = &rowsAffected
		executionLog.CompletedAt = time.Now()

		if logErr := s.executionLogRepo.Update(ctx, executionLog); logErr != nil {
			s.logger.Warn("Failed to update execution log entry", zap.Error(logErr))
		}
	}

	return result, nil
}

// GetQueueStatus returns the queue status for a Change Plan
func (s *ExecutionService) GetQueueStatus(ctx context.Context, changePlanID uuid.UUID) (*models.ExecutionQueue, error) {
	if s.queueService == nil {
		return nil, fmt.Errorf("queue service not available")
	}
	return s.queueService.GetQueueStatus(ctx, changePlanID)
}

// GetExecutionLogHistory returns the execution log history for a Change Plan
func (s *ExecutionService) GetExecutionLogHistory(ctx context.Context, changePlanID uuid.UUID) ([]*models.ExecutionLog, error) {
	if s.executionLogRepo == nil {
		return nil, fmt.Errorf("execution log repository not available")
	}
	return s.executionLogRepo.GetByChangePlanID(ctx, changePlanID)
}

// PositionedError wraps an error with position information for multi-statement execution
type PositionedError struct {
	Message        string
	Line           int
	Column         int
	StatementIndex int
	OriginalError  error
}

func (e *PositionedError) Error() string {
	return fmt.Sprintf("statement %d (line %d, column %d): %s",
		e.StatementIndex+1, e.Line, e.Column, e.Message)
}

func (e *PositionedError) Unwrap() error {
	return e.OriginalError
}

// validateGhostPlan runs defense-in-depth checks before ghost execution.
func (s *ExecutionService) validateGhostPlan(ctx context.Context, changePlan *models.ChangePlan) error {
	if s.ghostExecutor == nil {
		return ErrGhostUnsupported
	}
	if err := s.ghostExecutor.ValidateGhostEligibility(ctx, changePlan); err != nil {
		return err
	}
	if s.envRepo != nil {
		dbRow, err := s.databaseRepo.GetByID(ctx, changePlan.TargetDatabaseID)
		if err == nil && dbRow.EnvironmentID != nil {
			env, err := s.envRepo.GetByID(ctx, *dbRow.EnvironmentID)
			if err == nil && env != nil && env.IsProduction {
				return ErrGhostRejectedEnv
			}
		}
	}
	return nil
}

// executeGhost runs a DDL change plan through the configured ghost tool
// (gh-ost for MySQL, pg-online-schema-change for PostgreSQL). It mirrors the
// direct executor's lifecycle: lock → InProgress → changelog/log → run →
// Completed/Failed. Between copy and cutover the plan sits in
// AwaitingCutover while the tool parks on its postpone flag; the database
// lock is intentionally held until the run finishes so queued plans wait
// for the whole migration (decisions locked).
func (s *ExecutionService) executeGhost(
	ctx context.Context,
	changePlan *models.ChangePlan,
	executorID uuid.UUID,
) (*ExecutionResult, error) {
	changePlanID := changePlan.ID
	startedAt := time.Now()

	// Extract table + alter clause first; ghost supports a single ALTER TABLE
	// per plan and invalid SQL should fail before touching repositories.
	table, alterClause, err := ParseAlterStatement(changePlan.SQLContent)
	if err != nil {
		return nil, fmt.Errorf("invalid ghost SQL: %w", err)
	}

	// Resolve database + instance (same as direct path)
	db, err := s.databaseRepo.GetByID(ctx, changePlan.TargetDatabaseID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrDatabaseNotFound
		}
		return nil, fmt.Errorf("failed to get database: %w", err)
	}
	dbInstance, err := s.dbInstanceRepo.GetByID(ctx, db.DatabaseInstanceID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, ErrDatabaseInstanceNotFound
		}
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Decrypt instance password for the tool invocation.
	password, err := utils.DecryptPassword(dbInstance.ConnectionDetailsEncrypted, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt instance password: %w", err)
	}

	postponeFlag, panicFlag := s.ghostExecutor.ControlFlagPaths(changePlanID)
	binPath, args, err := s.ghostExecutor.BuildArgs(
		dbInstance.DatabaseEngine,
		dbInstance.Host,
		dbInstance.Username,
		password,
		db.DatabaseName,
		table,
		alterClause,
		dbInstance.Port,
		postponeFlag,
		panicFlag,
	)
	if err != nil {
		return nil, err
	}

	// gh-ost requires the postpone flag to exist BEFORE the copy finishes to
	// park awaiting cutover; create it up front (doc'd ghost-lab workflow).
	if dbInstance.DatabaseEngine == models.DatabaseEngineMySQL && postponeFlag != "" {
		if f, err := os.Create(postponeFlag); err == nil {
			_ = f.Close()
			defer func() {
				_ = os.Remove(postponeFlag)
			}()
		}
	}

	// Acquire the execution lock (queued behavior mirrors direct executor).
	if err := s.lockManager.AcquireLock(ctx, db.ID, changePlanID, executorID); err != nil {
		if err == ErrDatabaseLocked && s.queueService != nil {
			if _, enqueueErr := s.queueService.Enqueue(ctx, changePlanID, db.ID, executorID); enqueueErr != nil {
				return nil, fmt.Errorf("database locked and failed to enqueue: %w", enqueueErr)
			}
		}
		return nil, ErrDatabaseLocked
	}
	defer func() {
		if releaseErr := s.lockManager.ReleaseLock(ctx, db.ID); releaseErr != nil {
			s.logger.Error("Failed to release execution lock", zap.Error(releaseErr))
		}
	}()

	// Mark plan InProgress and create changelog + execution log entries.
	changePlan.Status = models.ChangePlanStatusInProgress
	changePlan.UpdatedAt = time.Now()
	if err := s.changePlanRepo.Update(ctx, changePlan); err != nil {
		return nil, fmt.Errorf("failed to update change plan status to InProgress: %w", err)
	}

	changelog := &models.Changelog{
		ID:                 uuid.New(),
		DatabaseInstanceID: dbInstance.ID,
		DatabaseID:         db.ID,
		ChangePlanID:       changePlanID,
		Status:             models.ChangelogStatusPending,
		CreatedAt:          time.Now(),
	}
	if err := s.changelogRepo.Create(ctx, changelog); err != nil {
		return nil, fmt.Errorf("failed to create changelog entry: %w", err)
	}

	executionLogID := uuid.New()
	dbType := string(dbInstance.DatabaseEngine)
	sqlContent := changePlan.SQLContent
	executionLog := &models.ExecutionLog{
		ID:                   executionLogID,
		ChangePlanID:         changePlanID,
		ExecutorID:           executorID,
		Status:               models.ExecutionLogStatusFailed,
		SQLContent:           &sqlContent,
		DatabaseID:           &db.ID,
		DatabaseName:         &db.DatabaseName,
		DatabaseInstanceName: &dbInstance.Name,
		DatabaseEngine:       &dbType,
		StartedAt:            time.Now(),
	}
	if err := s.executionLogRepo.Create(ctx, executionLog); err != nil {
		return nil, fmt.Errorf("failed to create execution log: %w", err)
	}

	// Stream tool output into the zap logger and flip the plan into
	// AwaitingCutover when the tool parks behind the postpone flag.
	awaitingCutoverSet := false
	callbacks := GhostRunCallbacks{
		OnLogLine: func(line string) {
			s.logger.Info("ghost",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("line", line),
			)
		},
		OnStateChange: func(status string) {
			if status == GhostStatusAwaiting && !awaitingCutoverSet {
				awaitingCutoverSet = true
				changePlan.Status = models.ChangePlanStatusAwaitingCutover
				changePlan.UpdatedAt = time.Now()
				if err := s.changePlanRepo.Update(context.Background(), changePlan); err != nil {
					s.logger.Error("Failed to set AwaitingCutover status", zap.Error(err))
				}
				s.logger.Info("Ghost awaiting cutover",
					zap.String("change_plan_id", changePlanID.String()),
				)
			}
		},
	}

	runErr := s.ghostExecutor.Run(ctx, binPath, args, callbacks)
	duration := time.Since(startedAt)

	// Finalize plan/log/changelog based on the tool's outcome.
	if runErr != nil {
		errMsg := runErr.Error()
		changePlan.Status = models.ChangePlanStatusFailed
		changePlan.ErrorMessage = &errMsg
		changePlan.UpdatedAt = time.Now()
		if updateErr := s.changePlanRepo.Update(ctx, changePlan); updateErr != nil {
			s.logger.Error("Failed to update change plan with failure", zap.Error(updateErr))
		}

		changelog.Status = models.ChangelogStatusFailed
		if updateErr := s.changelogRepo.Update(ctx, changelog); updateErr != nil {
			s.logger.Error("Failed to update changelog status to failed", zap.Error(updateErr))
		}

		executionLog.Status = models.ExecutionLogStatusFailed
		executionLog.ErrorMessage = &errMsg
		executionLog.CompletedAt = time.Now()
		executionLog.ExecutionDurationMs = int(duration.Milliseconds())
		if logErr := s.executionLogRepo.Update(ctx, executionLog); logErr != nil {
			s.logger.Warn("Failed to update execution log entry", zap.Error(logErr))
		}

		return &ExecutionResult{Success: false, Duration: duration, ErrorMessage: &errMsg}, runErr
	}

	// Success path
	changePlan.Status = models.ChangePlanStatusCompleted
	executedAt := time.Now()
	changePlan.ExecutedAt = &executedAt
	durationMs := int(duration.Milliseconds())
	changePlan.ExecutionDurationMs = &durationMs
	changePlan.UpdatedAt = time.Now()
	if updateErr := s.changePlanRepo.Update(ctx, changePlan); updateErr != nil {
		s.logger.Error("Failed to update change plan to Completed", zap.Error(updateErr))
	}

	changelog.Status = models.ChangelogStatusDone
	if updateErr := s.changelogRepo.Update(ctx, changelog); updateErr != nil {
		s.logger.Error("Failed to update changelog status to completed", zap.Error(updateErr))
	}

	executionLog.Status = models.ExecutionLogStatusCompleted
	executionLog.CompletedAt = time.Now()
	executionLog.ExecutionDurationMs = durationMs
	if logErr := s.executionLogRepo.Update(ctx, executionLog); logErr != nil {
		s.logger.Warn("Failed to update execution log entry", zap.Error(logErr))
	}

	s.logger.Info("Ghost execution completed",
		zap.String("change_plan_id", changePlanID.String()),
		zap.Duration("duration", duration),
	)

	return &ExecutionResult{Success: true, Duration: duration}, nil
}

// executeInTransaction executes SQL statements within a SERIALIZABLE transaction
// Now supports multi-statement execution with per-statement tracking
func (s *ExecutionService) executeInTransaction(
	ctx context.Context,
	dbInstance *models.DatabaseInstance,
	db *models.Database,
	sqlContent string,
	changePlanID uuid.UUID,
	executionLogID uuid.UUID,
) (*ExecutionResult, error) {
	// Determine dialect for SQL parsing
	var dialect parser.Dialect
	switch dbInstance.DatabaseEngine {
	case models.DatabaseEnginePostgreSQL:
		dialect = parser.DialectPostgres
	case models.DatabaseEngineMySQL:
		dialect = parser.DialectMySQL
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", dbInstance.DatabaseEngine)
	}

	// Parse SQL into individual statements
	parseResult, err := parser.SplitSQL(sqlContent, dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SQL: %w", err)
	}

	// Validate statement count
	if parseResult.TotalCount > s.maxStatementCount {
		return nil, fmt.Errorf("too many statements: %d (max %d)", parseResult.TotalCount, s.maxStatementCount)
	}

	s.logger.Info("Parsed SQL statements",
		zap.String("change_plan_id", changePlanID.String()),
		zap.Int("statement_count", parseResult.TotalCount),
		zap.String("dialect", string(dialect)),
	)

	// Get database connection
	// For CREATE DATABASE operations (Pending status), connect to system database
	connectStartTime := time.Now()
	var dbConn *sqlx.DB

	if db.Status == models.DatabaseStatusPending {
		// Create a temporary database object for system database connection
		systemDB := &models.Database{
			ID:                 db.ID,
			ProjectID:          db.ProjectID,
			DatabaseInstanceID: db.DatabaseInstanceID,
			EnvironmentID:      db.EnvironmentID,
			Status:             db.Status,
			CreatedByID:        db.CreatedByID,
			CreatedAt:          db.CreatedAt,
			UpdatedAt:          db.UpdatedAt,
		}

		// Use system database based on engine type
		switch dbInstance.DatabaseEngine {
		case models.DatabaseEnginePostgreSQL:
			systemDB.DatabaseName = "postgres" // Default PostgreSQL system database
		case models.DatabaseEngineMySQL:
			systemDB.DatabaseName = "" // MySQL can execute CREATE DATABASE without specifying a database
		}

		s.logger.Info("Connecting to system database for CREATE DATABASE operation",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("target_database_name", db.DatabaseName),
			zap.String("system_database_name", systemDB.DatabaseName),
			zap.String("database_engine", string(dbInstance.DatabaseEngine)),
		)

		dbConn, err = s.connectionMgr.Connect(ctx, database.ConnectionParams{
			DatabaseInstance: dbInstance,
			Database:         systemDB,
			IOTimeout:        s.executionTimeout,
		})
	} else {
		dbConn, err = s.connectionMgr.Connect(ctx, database.ConnectionParams{
			DatabaseInstance: dbInstance,
			Database:         db,
			IOTimeout:        s.executionTimeout,
		})
	}

	if err != nil {
		s.logger.Error("Failed to connect to database",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", db.ID.String()),
			zap.String("database_name", db.DatabaseName),
			zap.String("database_instance_name", dbInstance.Name),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	connectDuration := time.Since(connectStartTime)
	s.logger.Debug("Database connection established",
		zap.String("change_plan_id", changePlanID.String()),
		zap.Duration("connection_duration", connectDuration),
	)

	defer func() {
		if err := dbConn.Close(); err != nil {
			s.logger.Warn("Failed to close database connection",
				zap.String("change_plan_id", changePlanID.String()),
				zap.Error(err),
			)
		}
	}()

	// For CREATE DATABASE operations, we cannot use transactions (PostgreSQL restriction)
	// Execute directly without transaction for Pending databases
	if db.Status == models.DatabaseStatusPending {
		s.logger.Info("Executing CREATE DATABASE without transaction",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_name", db.DatabaseName),
		)

		execStartTime := time.Now()
		result, err := dbConn.ExecContext(ctx, sqlContent)
		execDuration := time.Since(execStartTime)

		if err != nil {
			s.logger.Error("Failed to execute CREATE DATABASE",
				zap.String("change_plan_id", changePlanID.String()),
				zap.String("database_name", db.DatabaseName),
				zap.Duration("execution_duration", execDuration),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to execute SQL: %w", err)
		}

		rowsAffected, _ := result.RowsAffected()
		s.logger.Info("CREATE DATABASE executed successfully",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_name", db.DatabaseName),
			zap.Int64("rows_affected", rowsAffected),
			zap.Duration("execution_duration", execDuration),
		)

		return &ExecutionResult{
			Success:      true,
			RowsAffected: int(rowsAffected),
			Duration:     execDuration,
		}, nil
	}

	// Begin transaction with SERIALIZABLE isolation for normal operations
	txStartTime := time.Now()
	tx, err := dbConn.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		s.logger.Error("Failed to begin transaction",
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("isolation_level", "SERIALIZABLE"),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	txBeginDuration := time.Since(txStartTime)
	s.logger.Debug("Transaction begun",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("isolation_level", "SERIALIZABLE"),
		zap.Duration("transaction_begin_duration", txBeginDuration),
	)

	// Track if we should rollback
	shouldRollback := true
	defer func() {
		if shouldRollback {
			rollbackStartTime := time.Now()
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				s.logger.Error("Failed to rollback transaction",
					zap.String("change_plan_id", changePlanID.String()),
					zap.Duration("rollback_duration", time.Since(rollbackStartTime)),
					zap.Error(rollbackErr),
				)
			} else {
				s.logger.Info("Transaction rolled back",
					zap.String("change_plan_id", changePlanID.String()),
					zap.Duration("rollback_duration", time.Since(rollbackStartTime)),
				)
			}
		}
	}()

	// Execute SQL statements sequentially
	execStartTime := time.Now()
	var totalRowsAffected int64
	var statementResults []models.StatementExecutionResult

	s.logger.Debug("Executing SQL statements",
		zap.String("change_plan_id", changePlanID.String()),
		zap.Int("statement_count", parseResult.TotalCount),
	)

	for _, stmt := range parseResult.Statements {
		if stmt.Empty {
			continue // Skip empty/comment-only statements
		}

		stmtStart := time.Now()
		result, err := tx.ExecContext(ctx, stmt.Text)
		stmtDuration := time.Since(stmtStart)

		stmtResult := models.StatementExecutionResult{
			ID:                  uuid.New(),
			ExecutionLogID:      executionLogID,
			StatementIndex:      stmt.Index,
			StatementSQL:        stmt.Text,
			ExecutionDurationMs: int(stmtDuration.Milliseconds()),
			StartLine:           &stmt.StartLine,
			StartColumn:         &stmt.StartColumn,
			EndLine:             &stmt.EndLine,
			EndColumn:           &stmt.EndColumn,
		}

		if err != nil {
			// Execution failed - record error
			errMsg := err.Error()
			stmtResult.Status = models.StatementStatusFailed
			stmtResult.ErrorMessage = &errMsg
			statementResults = append(statementResults, stmtResult)

			// Save statement results before returning
			s.saveStatementResults(ctx, statementResults)

			s.logger.Error("Statement execution failed",
				zap.String("change_plan_id", changePlanID.String()),
				zap.Int("statement_index", stmt.Index),
				zap.Int("start_line", stmt.StartLine),
				zap.Int("start_column", stmt.StartColumn),
				zap.Duration("statement_duration", stmtDuration),
				zap.Error(err),
			)

			// Create structured SQL execution error with position info
			sqlErr := NewSQLExecutionError(
				string(dbInstance.DatabaseEngine),
				db.DatabaseName,
				db.ID.String(),
				dbInstance.Name,
				stmt.Text,
				err,
			)

			// Return positioned error
			return nil, &PositionedError{
				Message:        sqlErr.Message,
				Line:           stmt.StartLine,
				Column:         stmt.StartColumn,
				StatementIndex: stmt.Index,
				OriginalError:  sqlErr,
			}
		}

		// Success - record result
		rowsAffected, _ := result.RowsAffected()
		rows := int(rowsAffected)
		totalRowsAffected += rowsAffected
		stmtResult.Status = models.StatementStatusCompleted
		stmtResult.RowsAffected = &rows
		statementResults = append(statementResults, stmtResult)

		s.logger.Debug("Statement executed successfully",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Int("statement_index", stmt.Index),
			zap.Int64("rows_affected", rowsAffected),
			zap.Duration("statement_duration", stmtDuration),
		)
	}

	// Commit transaction
	commitStartTime := time.Now()
	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction",
			zap.String("change_plan_id", changePlanID.String()),
			zap.Int64("total_rows_affected", totalRowsAffected),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	commitDuration := time.Since(commitStartTime)
	s.logger.Debug("Transaction committed successfully",
		zap.String("change_plan_id", changePlanID.String()),
		zap.Int64("total_rows_affected", totalRowsAffected),
		zap.Duration("commit_duration", commitDuration),
	)

	// Mark that we don't need to rollback (commit succeeded)
	shouldRollback = false

	// Save statement results after successful commit
	s.saveStatementResults(ctx, statementResults)

	duration := time.Since(execStartTime)

	return &ExecutionResult{
		Success:      true,
		RowsAffected: int(totalRowsAffected),
		Duration:     duration,
	}, nil
}

// saveStatementResults saves statement execution results to the repository
func (s *ExecutionService) saveStatementResults(ctx context.Context, results []models.StatementExecutionResult) {
	if s.statementResultRepo == nil || len(results) == 0 {
		return
	}

	if err := s.statementResultRepo.CreateBatch(ctx, results); err != nil {
		s.logger.Warn("Failed to save statement execution results",
			zap.Int("result_count", len(results)),
			zap.Error(err),
		)
	}
}
