// Package services provides business logic services for the Deeb application.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/utils"
	"go.uber.org/zap"
)

// SchemaSnapshotService provides schema snapshot capture functionality
type SchemaSnapshotService struct {
	snapshotRepo   repository.SchemaSnapshotRepository
	dbInstanceRepo repository.DatabaseInstanceRepository
	databaseRepo   repository.DatabaseRepository
	connectionMgr  *database.ConnectionManager
	encryptionKey  string
	logger         *zap.Logger
}

// NewSchemaSnapshotService creates a new schema snapshot service
func NewSchemaSnapshotService(
	snapshotRepo repository.SchemaSnapshotRepository,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	connectionMgr *database.ConnectionManager,
	encryptionKey string,
	logger *zap.Logger,
) *SchemaSnapshotService {
	return &SchemaSnapshotService{
		snapshotRepo:   snapshotRepo,
		dbInstanceRepo: dbInstanceRepo,
		databaseRepo:   databaseRepo,
		connectionMgr:  connectionMgr,
		encryptionKey:  encryptionKey,
		logger:         logger,
	}
}

// CapturePostgreSQL captures a PostgreSQL schema snapshot using pg_dump
func (s *SchemaSnapshotService) CapturePostgreSQL(
	ctx context.Context,
	changePlanID uuid.UUID,
	databaseID uuid.UUID,
	snapshotType models.SnapshotType,
) (*models.SchemaSnapshot, error) {
	// Get database and database instance
	db, err := s.databaseRepo.GetByID(ctx, databaseID)
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

	// Validate database type
	if dbInstance.DatabaseType != models.DatabaseTypePostgreSQL {
		return nil, fmt.Errorf("invalid database type: expected postgresql, got %s", dbInstance.DatabaseType)
	}

	// Decrypt password
	password, err := s.getDecryptedPassword(dbInstance)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Build pg_dump command with performance optimizations
	// pg_dump --schema-only --no-owner --no-privileges --host=<host> --port=<port> --username=<username> <database>
	// Added --jobs=4 for parallel dump (faster for large schemas)
	// Note: --compress is only available for directory/custom formats, not plain format
	cmd := exec.CommandContext(ctx, "pg_dump",
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"--host", dbInstance.Host,
		"--port", fmt.Sprintf("%d", dbInstance.Port),
		"--username", dbInstance.Username,
		"--dbname", db.DatabaseName,
		"--format", "plain", // Plain text SQL format
		"--jobs", "4", // Parallel dump for faster execution (4 parallel workers)
	)

	// Set password via PGPASSWORD environment variable
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", password))

	// Set SSL mode if required
	if dbInstance.RequireSSL {
		cmd.Env = append(cmd.Env, "PGSSLMODE=require")
	} else {
		cmd.Env = append(cmd.Env, "PGSSLMODE=disable")
	}

	s.logger.Info("Capturing PostgreSQL schema snapshot",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", databaseID.String()),
		zap.String("snapshot_type", string(snapshotType)),
		zap.String("host", dbInstance.Host),
		zap.Int("port", dbInstance.Port),
		zap.String("database", db.DatabaseName),
	)

	// Execute pg_dump with timeout
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Error("Failed to capture PostgreSQL schema snapshot",
			zap.Error(err),
			zap.String("output", string(output)),
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", databaseID.String()),
		)
		return nil, fmt.Errorf("failed to execute pg_dump: %w", err)
	}

	// Convert output to JSON for storage
	// Store raw SQL text in snapshot_data_raw
	snapshotDataRaw := string(output)

	// Wrap raw SQL in JSONB format for snapshot_data
	snapshotDataJSON, err := json.Marshal(map[string]string{
		"sql":    snapshotDataRaw,
		"format": "plain",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot data: %w", err)
	}

	// Create schema snapshot
	snapshot := &models.SchemaSnapshot{
		ID:              uuid.New(),
		DatabaseID:      databaseID,
		ChangePlanID:    changePlanID,
		SnapshotType:    snapshotType,
		SnapshotData:    snapshotDataJSON,
		SnapshotDataRaw: &snapshotDataRaw,
		CreatedAt:       time.Now(),
	}

	// Store snapshot in repository
	if err := s.snapshotRepo.Create(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("failed to store schema snapshot: %w", err)
	}

	s.logger.Info("PostgreSQL schema snapshot captured successfully",
		zap.String("snapshot_id", snapshot.ID.String()),
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", databaseID.String()),
		zap.String("snapshot_type", string(snapshotType)),
		zap.Int("snapshot_size_bytes", len(output)),
	)

	return snapshot, nil
}

// CaptureMySQL captures a MySQL schema snapshot using mysqldump
func (s *SchemaSnapshotService) CaptureMySQL(
	ctx context.Context,
	changePlanID uuid.UUID,
	databaseID uuid.UUID,
	snapshotType models.SnapshotType,
) (*models.SchemaSnapshot, error) {
	// Get database and database instance
	db, err := s.databaseRepo.GetByID(ctx, databaseID)
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

	// Validate database type
	if dbInstance.DatabaseType != models.DatabaseTypeMySQL {
		return nil, fmt.Errorf("invalid database type: expected mysql, got %s", dbInstance.DatabaseType)
	}

	// Decrypt password
	password, err := s.getDecryptedPassword(dbInstance)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Build mysqldump command with performance optimizations
	// mysqldump --no-data --host=<host> --port=<port> --user=<username> --password=<password> <database>
	// Added --single-transaction and --quick for faster dumps
	cmd := exec.CommandContext(ctx, "mysqldump",
		"--no-data",
		"--host", dbInstance.Host,
		"--port", fmt.Sprintf("%d", dbInstance.Port),
		"--user", dbInstance.Username,
		fmt.Sprintf("--password=%s", password),
		"--skip-comments",       // Skip comments for cleaner output
		"--skip-add-drop-table", // Don't include DROP TABLE statements
		"--single-transaction",  // Faster for InnoDB tables (no lock)
		"--quick",               // Retrieve rows one at a time (faster for large schemas)
		"--compress",            // Compress data transfer
		db.DatabaseName,
	)

	// Set SSL mode if required
	if dbInstance.RequireSSL {
		cmd.Args = append(cmd.Args, "--ssl-mode=REQUIRED")
	} else {
		cmd.Args = append(cmd.Args, "--ssl-mode=DISABLED")
	}

	// Preserve environment variables
	cmd.Env = os.Environ()

	s.logger.Info("Capturing MySQL schema snapshot",
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", databaseID.String()),
		zap.String("snapshot_type", string(snapshotType)),
		zap.String("host", dbInstance.Host),
		zap.Int("port", dbInstance.Port),
		zap.String("database", db.DatabaseName),
	)

	// Execute mysqldump with timeout
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Error("Failed to capture MySQL schema snapshot",
			zap.Error(err),
			zap.String("output", string(output)),
			zap.String("change_plan_id", changePlanID.String()),
			zap.String("database_id", databaseID.String()),
		)
		return nil, fmt.Errorf("failed to execute mysqldump: %w", err)
	}

	// Convert output to JSON for storage (consistent with PostgreSQL format)
	// Store raw SQL text in snapshot_data_raw
	snapshotDataRaw := string(output)

	// Wrap raw SQL in JSONB format for snapshot_data (consistent with PostgreSQL)
	snapshotDataJSON, err := json.Marshal(map[string]string{
		"sql":    snapshotDataRaw,
		"format": "plain",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot data: %w", err)
	}

	// Create schema snapshot
	snapshot := &models.SchemaSnapshot{
		ID:              uuid.New(),
		DatabaseID:      databaseID,
		ChangePlanID:    changePlanID,
		SnapshotType:    snapshotType,
		SnapshotData:    snapshotDataJSON,
		SnapshotDataRaw: &snapshotDataRaw,
		CreatedAt:       time.Now(),
	}

	// Store snapshot in repository
	if err := s.snapshotRepo.Create(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("failed to store schema snapshot: %w", err)
	}

	s.logger.Info("MySQL schema snapshot captured successfully",
		zap.String("snapshot_id", snapshot.ID.String()),
		zap.String("change_plan_id", changePlanID.String()),
		zap.String("database_id", databaseID.String()),
		zap.String("snapshot_type", string(snapshotType)),
		zap.Int("snapshot_size_bytes", len(output)),
	)

	return snapshot, nil
}

// getDecryptedPassword decrypts the password from database instance
func (s *SchemaSnapshotService) getDecryptedPassword(dbInstance *models.DatabaseInstance) (string, error) {
	password, err := utils.DecryptPassword(dbInstance.ConnectionDetailsEncrypted, s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}
	return password, nil
}

// Schema snapshot errors
var (
	ErrDatabaseNotFound         = fmt.Errorf("database not found")
	ErrDatabaseInstanceNotFound = fmt.Errorf("database instance not found")
	ErrSnapshotConnectionFailed = fmt.Errorf("snapshot connection failed")
	ErrSnapshotPermissionDenied = fmt.Errorf("snapshot permission denied")
	ErrSnapshotTimeout          = fmt.Errorf("snapshot timeout")
)
