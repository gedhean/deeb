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

// SchemaVersionService provides schema versioning functionality
type SchemaVersionService struct {
	schemaHistoryRepo repository.SchemaHistoryRepository
	dbInstanceRepo    repository.DatabaseInstanceRepository
	databaseRepo      repository.DatabaseRepository
	connectionMgr     *database.ConnectionManager
	encryptionKey     string
	logger            *zap.Logger
}

// NewSchemaVersionService creates a new schema version service
func NewSchemaVersionService(
	schemaHistoryRepo repository.SchemaHistoryRepository,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	connectionMgr *database.ConnectionManager,
	encryptionKey string,
	logger *zap.Logger,
) *SchemaVersionService {
	return &SchemaVersionService{
		schemaHistoryRepo: schemaHistoryRepo,
		dbInstanceRepo:    dbInstanceRepo,
		databaseRepo:      databaseRepo,
		connectionMgr:     connectionMgr,
		encryptionKey:     encryptionKey,
		logger:            logger,
	}
}

// CaptureSchemaVersion captures the current schema state for a database and stores it in schema_history
func (s *SchemaVersionService) CaptureSchemaVersion(ctx context.Context, databaseID uuid.UUID) (*models.SchemaHistory, error) {
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

	s.logger.Info("Capturing schema version",
		zap.String("database_id", databaseID.String()),
		zap.String("database_instance_id", dbInstance.ID.String()),
		zap.String("database_name", db.DatabaseName),
		zap.String("database_engine", string(dbInstance.DatabaseEngine)),
	)

	// Capture schema based on database engine
	var rawDump string
	switch dbInstance.DatabaseEngine {
	case models.DatabaseEnginePostgreSQL:
		rawDump, err = s.capturePostgreSQLSchema(ctx, db, dbInstance)
	case models.DatabaseEngineMySQL:
		rawDump, err = s.captureMySQLSchema(ctx, db, dbInstance)
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", dbInstance.DatabaseEngine)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to capture schema: %w", err)
	}

	// Create metadata JSON
	metadata := map[string]interface{}{
		"database_name":   db.DatabaseName,
		"database_engine": string(dbInstance.DatabaseEngine),
		"host":            dbInstance.Host,
		"port":            dbInstance.Port,
		"captured_at":     time.Now().Format(time.RFC3339),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create schema history entry
	schemaHistory := &models.SchemaHistory{
		ID:                 uuid.New(),
		DatabaseInstanceID: dbInstance.ID,
		DatabaseID:         databaseID,
		Metadata:           metadataJSON,
		RawDump:            rawDump,
		CreatedAt:          time.Now(),
	}

	// Store schema history
	if err := s.schemaHistoryRepo.Create(ctx, schemaHistory); err != nil {
		return nil, fmt.Errorf("failed to store schema history: %w", err)
	}

	s.logger.Info("Schema version captured successfully",
		zap.String("schema_history_id", schemaHistory.ID.String()),
		zap.String("database_id", databaseID.String()),
		zap.Int("raw_dump_size_bytes", len(rawDump)),
	)

	return schemaHistory, nil
}

// capturePostgreSQLSchema captures a PostgreSQL schema using pg_dump
func (s *SchemaVersionService) capturePostgreSQLSchema(
	ctx context.Context,
	db *models.Database,
	dbInstance *models.DatabaseInstance,
) (string, error) {
	// Decrypt password
	password, err := s.getDecryptedPassword(dbInstance)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Build pg_dump command
	cmd := exec.CommandContext(ctx, "pg_dump",
		"--schema-only",
		"--no-owner",
		"--no-privileges",
		"--host", dbInstance.Host,
		"--port", fmt.Sprintf("%d", dbInstance.Port),
		"--username", dbInstance.Username,
		"--dbname", db.DatabaseName,
		"--format", "plain",
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

	s.logger.Debug("Executing pg_dump",
		zap.String("host", dbInstance.Host),
		zap.Int("port", dbInstance.Port),
		zap.String("database", db.DatabaseName),
	)

	// Execute pg_dump
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Error("Failed to execute pg_dump",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return "", fmt.Errorf("failed to execute pg_dump: %w", err)
	}

	return string(output), nil
}

// captureMySQLSchema captures a MySQL schema using mysqldump
func (s *SchemaVersionService) captureMySQLSchema(
	ctx context.Context,
	db *models.Database,
	dbInstance *models.DatabaseInstance,
) (string, error) {
	// Decrypt password
	password, err := s.getDecryptedPassword(dbInstance)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}

	// Build mysqldump command
	cmd := exec.CommandContext(ctx, "mysqldump",
		"--no-data",
		"--host", dbInstance.Host,
		"--port", fmt.Sprintf("%d", dbInstance.Port),
		"--user", dbInstance.Username,
		fmt.Sprintf("--password=%s", password),
		"--skip-comments",
		"--skip-add-drop-table",
		"--single-transaction",
		"--quick",
		"--compress",
		db.DatabaseName,
	)

	// Set SSL mode if required
	if dbInstance.RequireSSL {
		cmd.Args = append(cmd.Args, "--ssl-mode=REQUIRED")
	} else {
		cmd.Args = append(cmd.Args, "--ssl-mode=DISABLED")
	}

	cmd.Env = os.Environ()

	s.logger.Debug("Executing mysqldump",
		zap.String("host", dbInstance.Host),
		zap.Int("port", dbInstance.Port),
		zap.String("database", db.DatabaseName),
	)

	// Execute mysqldump
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Error("Failed to execute mysqldump",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return "", fmt.Errorf("failed to execute mysqldump: %w", err)
	}

	return string(output), nil
}

// GetLatestSchemaVersion retrieves the most recent schema version for a database
func (s *SchemaVersionService) GetLatestSchemaVersion(ctx context.Context, databaseID uuid.UUID) (*models.SchemaHistory, error) {
	schemaHistory, err := s.schemaHistoryRepo.GetLatestByDatabaseID(ctx, databaseID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, nil // No schema history exists yet
		}
		return nil, fmt.Errorf("failed to get latest schema version: %w", err)
	}

	return schemaHistory, nil
}

// getDecryptedPassword decrypts the database instance password
func (s *SchemaVersionService) getDecryptedPassword(dbInstance *models.DatabaseInstance) (string, error) {
	password, err := utils.DecryptPassword(dbInstance.ConnectionDetailsEncrypted, s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt password: %w", err)
	}
	return password, nil
}
