package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// SchemaSyncJob runs background synchronization of schema metadata for all databases
type SchemaSyncJob struct {
	databaseRepo          repository.DatabaseRepository
	databaseSchemaRepo    repository.DatabaseSchemaRepository
	schemaMetadataService *SchemaMetadataService
	logger                *zap.Logger
	stopChan              chan struct{}
	wg                    sync.WaitGroup
	syncInterval          time.Duration
	staleThreshold        time.Duration
}

// NewSchemaSyncJob creates a new schema sync job
func NewSchemaSyncJob(
	databaseRepo repository.DatabaseRepository,
	databaseSchemaRepo repository.DatabaseSchemaRepository,
	schemaMetadataService *SchemaMetadataService,
	syncInterval time.Duration,
	logger *zap.Logger,
) *SchemaSyncJob {
	// Default sync interval is 1 hour if not specified
	if syncInterval == 0 {
		syncInterval = 1 * time.Hour
	}

	return &SchemaSyncJob{
		databaseRepo:          databaseRepo,
		databaseSchemaRepo:    databaseSchemaRepo,
		schemaMetadataService: schemaMetadataService,
		logger:                logger,
		stopChan:              make(chan struct{}),
		syncInterval:          syncInterval,
		staleThreshold:        syncInterval, // Consider schemas stale after sync interval
	}
}

// Start starts the schema sync background worker
func (j *SchemaSyncJob) Start(ctx context.Context) {
	j.wg.Add(1)
	go j.syncLoop(ctx)
	j.logger.Info("Schema sync job started",
		zap.Duration("sync_interval", j.syncInterval),
	)
}

// Stop stops the schema sync job
func (j *SchemaSyncJob) Stop() {
	close(j.stopChan)
	j.wg.Wait()
	j.logger.Info("Schema sync job stopped")
}

// syncLoop continuously syncs schema metadata
func (j *SchemaSyncJob) syncLoop(ctx context.Context) {
	defer j.wg.Done()

	// Run initial sync immediately
	if err := j.syncAllStaleSchemas(ctx); err != nil {
		j.logger.Error("Initial schema sync failed", zap.Error(err))
	}

	ticker := time.NewTicker(j.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			j.logger.Info("Schema sync job context cancelled")
			return
		case <-j.stopChan:
			j.logger.Info("Schema sync job stop signal received")
			return
		case <-ticker.C:
			if err := j.syncAllStaleSchemas(ctx); err != nil {
				j.logger.Error("Schema sync failed", zap.Error(err))
			}
		}
	}
}

// syncAllStaleSchemas syncs schema metadata for all stale databases
func (j *SchemaSyncJob) syncAllStaleSchemas(ctx context.Context) error {
	j.logger.Info("Starting schema sync for stale schemas")

	// Get all databases with stale schema cache (older than stale threshold)
	staleDatabaseIDs, err := j.databaseSchemaRepo.GetStaleSchemas(ctx, j.staleThreshold)
	if err != nil {
		return fmt.Errorf("failed to get stale schemas: %w", err)
	}

	// Get only active databases for syncing (excludes Inactive and Pending)
	// We don't sync schema metadata for inactive or pending databases
	allDatabases, err := j.databaseRepo.ListActiveForSync(ctx, 10000, 0)
	if err != nil {
		return fmt.Errorf("failed to list active databases: %w", err)
	}

	// Get list of all cached database IDs
	cachedDatabaseIDs, err := j.databaseSchemaRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to list cached schemas: %w", err)
	}

	// Create a map of cached database IDs for quick lookup
	cachedMap := make(map[uuid.UUID]bool)
	for _, id := range cachedDatabaseIDs {
		cachedMap[id] = true
	}

	// Create a map of active database IDs for filtering
	activeDatabaseMap := make(map[uuid.UUID]bool)
	for _, db := range allDatabases {
		activeDatabaseMap[db.ID] = true
	}

	// Filter stale databases to only include active ones
	var filteredStaleDatabaseIDs []uuid.UUID
	for _, id := range staleDatabaseIDs {
		if activeDatabaseMap[id] {
			filteredStaleDatabaseIDs = append(filteredStaleDatabaseIDs, id)
		}
	}

	// Find active databases that have never been cached
	var unsyncedDatabaseIDs []uuid.UUID
	for _, db := range allDatabases {
		if !cachedMap[db.ID] {
			unsyncedDatabaseIDs = append(unsyncedDatabaseIDs, db.ID)
		}
	}

	// Combine filtered stale and unsynced databases
	databaseIDsToSync := append(filteredStaleDatabaseIDs, unsyncedDatabaseIDs...)

	if len(databaseIDsToSync) == 0 {
		j.logger.Info("No stale or unsynced schemas found")
		return nil
	}

	j.logger.Info("Found active databases requiring schema sync",
		zap.Int("stale_count", len(filteredStaleDatabaseIDs)),
		zap.Int("unsynced_count", len(unsyncedDatabaseIDs)),
		zap.Int("total_count", len(databaseIDsToSync)),
	)

	// Sync each database
	successCount := 0
	errorCount := 0
	for _, databaseID := range databaseIDsToSync {
		if err := j.syncDatabase(ctx, databaseID); err != nil {
			j.logger.Error("Failed to sync database schema",
				zap.String("database_id", databaseID.String()),
				zap.Error(err),
			)
			errorCount++
		} else {
			successCount++
		}
	}

	j.logger.Info("Schema sync completed",
		zap.Int("success_count", successCount),
		zap.Int("error_count", errorCount),
	)

	return nil
}

// syncDatabase syncs schema metadata for a single database
func (j *SchemaSyncJob) syncDatabase(ctx context.Context, databaseID uuid.UUID) error {
	j.logger.Debug("Syncing database schema", zap.String("database_id", databaseID.String()))

	// Extract schema metadata
	schemaMetadata, err := j.schemaMetadataService.ExtractSchemaMetadata(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("failed to extract schema metadata: %w", err)
	}

	// Store in cache
	if err := j.databaseSchemaRepo.Upsert(ctx, databaseID, schemaMetadata); err != nil {
		return fmt.Errorf("failed to upsert schema cache: %w", err)
	}

	j.logger.Debug("Database schema synced successfully",
		zap.String("database_id", databaseID.String()),
		zap.Int("table_count", len(schemaMetadata.Tables)),
	)

	return nil
}
