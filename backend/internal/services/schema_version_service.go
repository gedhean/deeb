package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/sql-schema/comparer/engine"
	"github.com/nsxbet/sql-schema/exporter"
	"github.com/nsxbet/sql-schema/snapshot"
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

// CaptureSchemaVersion captures the current schema state for a database using sql-schema snapshot.TakeSnapshot
// and stores it in schema_history
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

	s.logger.Info("Capturing schema snapshot using snapshot.TakeSnapshot",
		zap.String("database_id", databaseID.String()),
		zap.String("database_instance_id", dbInstance.ID.String()),
		zap.String("database_name", db.DatabaseName),
		zap.String("database_engine", string(dbInstance.DatabaseEngine)),
	)

	// Get database connection from connection manager
	sqlDB, err := s.connectionMgr.Connect(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			s.logger.Error("Failed to close database connection", zap.Error(closeErr))
		}
	}()

	// Determine engine type
	var eng engine.Engine
	switch dbInstance.DatabaseEngine {
	case models.DatabaseEnginePostgreSQL:
		eng = engine.PostgreSQL
	case models.DatabaseEngineMySQL:
		eng = engine.MySQL
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", dbInstance.DatabaseEngine)
	}

	// Take snapshot using snapshot.TakeSnapshot (use sqlDB.DB to get underlying *sql.DB)
	snap, err := snapshot.TakeSnapshot(sqlDB.DB, db.DatabaseName, eng, &snapshot.SnapshotOptions{
		Version:     "1.0",
		Description: fmt.Sprintf("Schema snapshot for change plan execution on %s", db.DatabaseName),
		Tags: map[string]string{
			"database_id":          databaseID.String(),
			"database_instance_id": dbInstance.ID.String(),
		},
		HostInfo:    fmt.Sprintf("%s:%d", dbInstance.Host, dbInstance.Port),
		ComputeHash: true, // Compute checksums for integrity verification
	})
	if err != nil {
		return nil, fmt.Errorf("failed to take snapshot: %w", err)
	}

	// Determine SQL dialect for DDL generation
	var dialect exporter.SQLDialect
	if eng == engine.PostgreSQL {
		dialect = exporter.DialectPostgreSQL
	} else {
		dialect = exporter.DialectMySQL
	}

	// Generate SQL DDL from the snapshot
	sqlDDL, err := exporter.GenerateDDL(snap.Schema, dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL DDL: %w", err)
	}

	// Create metadata JSON for schema_history (includes snapshot metadata)
	metadata := map[string]interface{}{
		"database_name":   db.DatabaseName,
		"database_engine": string(dbInstance.DatabaseEngine),
		"host":            dbInstance.Host,
		"port":            dbInstance.Port,
		"captured_at":     snap.Metadata.Timestamp.Format(time.RFC3339),
		"version":         snap.Metadata.Version,
		"checksum_md5":    snap.Metadata.ChecksumMD5,
		"checksum_sha256": snap.Metadata.ChecksumSHA256,
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
		RawDump:            sqlDDL, // Store SQL DDL generated from snapshot
		CreatedAt:          time.Now(),
	}

	// Store schema history
	if err := s.schemaHistoryRepo.Create(ctx, schemaHistory); err != nil {
		return nil, fmt.Errorf("failed to store schema history: %w", err)
	}

	// Count tables for logging
	tableCount := 0
	for _, schema := range snap.Schema.Schemas {
		tableCount += len(schema.Tables)
	}

	s.logger.Info("Schema snapshot captured and DDL generated successfully",
		zap.String("schema_history_id", schemaHistory.ID.String()),
		zap.String("database_id", databaseID.String()),
		zap.Int("sql_ddl_size_bytes", len(sqlDDL)),
		zap.Int("table_count", tableCount),
		zap.String("checksum_md5", snap.Metadata.ChecksumMD5),
		zap.String("checksum_sha256", snap.Metadata.ChecksumSHA256),
	)

	return schemaHistory, nil
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
