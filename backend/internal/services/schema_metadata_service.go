// Package services provides business logic services for the Deeb application.
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	schemaextract "github.com/nsxbet/sql-schema"
	mysqlextractor "github.com/nsxbet/sql-schema/extractor/mysql"
	postgresextractor "github.com/nsxbet/sql-schema/extractor/postgres"
	"go.uber.org/zap"
)

// SchemaMetadataService provides database schema metadata extraction functionality
type SchemaMetadataService struct {
	dbInstanceRepo     repository.DatabaseInstanceRepository
	databaseRepo       repository.DatabaseRepository
	databaseSchemaRepo repository.DatabaseSchemaRepository
	connectionMgr      *database.ConnectionManager
	logger             *zap.Logger
	cacheExpiration    time.Duration // How long cached schemas are considered valid
}

// NewSchemaMetadataService creates a new schema metadata service
func NewSchemaMetadataService(
	dbInstanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	databaseSchemaRepo repository.DatabaseSchemaRepository,
	connectionMgr *database.ConnectionManager,
	logger *zap.Logger,
) *SchemaMetadataService {
	return &SchemaMetadataService{
		dbInstanceRepo:     dbInstanceRepo,
		databaseRepo:       databaseRepo,
		databaseSchemaRepo: databaseSchemaRepo,
		connectionMgr:      connectionMgr,
		logger:             logger,
		cacheExpiration:    1 * time.Hour, // Default: cache expires after 1 hour
	}
}

// ExtractSchemaMetadata extracts complete schema metadata for a database
// It first checks the cache and returns cached data if fresh. If cache is stale or missing,
// it extracts fresh data from the database.
func (s *SchemaMetadataService) ExtractSchemaMetadata(
	ctx context.Context,
	databaseID uuid.UUID,
) (*models.DatabaseSchemaMetadata, error) {
	// Try to get from cache first
	if s.databaseSchemaRepo != nil {
		cachedSchema, err := s.databaseSchemaRepo.GetByDatabaseID(ctx, databaseID)
		if err == nil && cachedSchema != nil {
			// Check if cache is still fresh
			if cachedSchema.LastSyncedAt != nil {
				age := time.Since(*cachedSchema.LastSyncedAt)
				if age < s.cacheExpiration {
					s.logger.Debug("Returning cached schema metadata",
						zap.String("database_id", databaseID.String()),
						zap.Duration("age", age),
					)
					return cachedSchema, nil
				}
				s.logger.Debug("Cached schema is stale, refreshing",
					zap.String("database_id", databaseID.String()),
					zap.Duration("age", age),
				)
			}
		} else if err != repository.ErrNotFound {
			// Log error but continue to extract fresh data
			s.logger.Warn("Failed to get cached schema, will extract fresh",
				zap.String("database_id", databaseID.String()),
				zap.Error(err),
			)
		}
	}

	// Extract fresh data
	return s.extractSchemaMetadataFresh(ctx, databaseID)
}

// RefreshSchemaMetadata forces a fresh extraction of schema metadata, bypassing cache
func (s *SchemaMetadataService) RefreshSchemaMetadata(
	ctx context.Context,
	databaseID uuid.UUID,
) (*models.DatabaseSchemaMetadata, error) {
	return s.extractSchemaMetadataFresh(ctx, databaseID)
}

// extractSchemaMetadataFresh extracts schema metadata directly from the database without cache
func (s *SchemaMetadataService) extractSchemaMetadataFresh(
	ctx context.Context,
	databaseID uuid.UUID,
) (*models.DatabaseSchemaMetadata, error) {
	// Get database and database instance
	db, err := s.databaseRepo.GetByID(ctx, databaseID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("database not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	dbInstance, err := s.dbInstanceRepo.GetByID(ctx, db.DatabaseInstanceID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, fmt.Errorf("database instance not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Extract schema based on database engine
	var schemaMetadata *models.DatabaseSchemaMetadata
	switch dbInstance.DatabaseEngine {
	case models.DatabaseEngineMySQL:
		schemaMetadata, err = s.extractMySQLSchema(ctx, db, dbInstance)
	case models.DatabaseEnginePostgreSQL:
		schemaMetadata, err = s.extractPostgreSQLSchema(ctx, db, dbInstance)
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", dbInstance.DatabaseEngine)
	}

	if err != nil {
		return nil, err
	}

	// Store in cache
	if s.databaseSchemaRepo != nil {
		if cacheErr := s.databaseSchemaRepo.Upsert(ctx, databaseID, schemaMetadata); cacheErr != nil {
			// Log error but don't fail the request
			s.logger.Warn("Failed to cache schema metadata",
				zap.String("database_id", databaseID.String()),
				zap.Error(cacheErr),
			)
		}
	}

	return schemaMetadata, nil
}

// ExtractTableDetail extracts detailed information for a specific table
// It uses the cached schema metadata to avoid redundant schema extraction
func (s *SchemaMetadataService) ExtractTableDetail(
	ctx context.Context,
	databaseID uuid.UUID,
	tableName string,
) (*models.TableDetail, error) {
	// Get schema metadata (uses cache if available)
	schemaMetadata, err := s.ExtractSchemaMetadata(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	// If RawSchema is nil (old cache), force a fresh extraction
	if schemaMetadata.RawSchema == nil {
		s.logger.Info("RawSchema is nil in cache, forcing fresh extraction",
			zap.String("database_id", databaseID.String()),
		)
		schemaMetadata, err = s.RefreshSchemaMetadata(ctx, databaseID)
		if err != nil {
			return nil, err
		}
		// Check again after refresh
		if schemaMetadata.RawSchema == nil {
			return nil, fmt.Errorf("no schema data available after refresh")
		}
	}

	// Search for the table in all schemas
	for _, schema := range schemaMetadata.RawSchema.Schemas {
		for _, table := range schema.Tables {
			if table.Name == tableName {
				return convertToTableDetail(table), nil
			}
		}
	}

	return nil, fmt.Errorf("table not found: %s", tableName)
}

// extractMySQLSchema extracts schema metadata from a MySQL database
func (s *SchemaMetadataService) extractMySQLSchema(
	ctx context.Context,
	db *models.Database,
	dbInstance *models.DatabaseInstance,
) (*models.DatabaseSchemaMetadata, error) {
	// Connect to MySQL database
	sqlxDB, err := s.connectionMgr.ConnectMySQL(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	defer func() {
		if closeErr := sqlxDB.Close(); closeErr != nil {
			s.logger.Error("Failed to close MySQL connection", zap.Error(closeErr))
		}
	}()

	s.logger.Info("Extracting MySQL schema metadata",
		zap.String("database_id", db.ID.String()),
		zap.String("database_name", db.DatabaseName),
	)

	// Create extractor and extract schema
	extractor := mysqlextractor.NewExtractor(sqlxDB.DB, db.DatabaseName)
	dbSchema, err := extractor.ExtractSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract MySQL schema: %w", err)
	}

	// Convert to our model format (MySQL has a single unnamed schema)
	var tables []models.TableListItem
	var views []models.ViewListItem
	var functions []models.FunctionListItem
	if len(dbSchema.Schemas) > 0 {
		tables = make([]models.TableListItem, 0, len(dbSchema.Schemas[0].Tables))
		for _, table := range dbSchema.Schemas[0].Tables {
			tables = append(tables, models.TableListItem{
				Name:      table.Name,
				RowCount:  table.RowCount,
				DataSize:  table.DataSize,
				IndexSize: table.IndexSize,
				TotalSize: table.DataSize + table.IndexSize,
				Engine:    table.Engine,
				Comment:   table.Comment,
			})
		}

		// Extract views
		views = make([]models.ViewListItem, 0, len(dbSchema.Schemas[0].Views))
		for _, view := range dbSchema.Schemas[0].Views {
			views = append(views, models.ViewListItem{
				Name:       view.Name,
				Definition: view.Definition,
				Comment:    view.Comment,
			})
		}

		// Extract functions (MySQL supports functions)
		functions = make([]models.FunctionListItem, 0, len(dbSchema.Schemas[0].Functions))
		for _, fn := range dbSchema.Schemas[0].Functions {
			functions = append(functions, models.FunctionListItem{
				Name:       fn.Name,
				Definition: fn.Definition,
				Comment:    fn.Comment,
			})
		}
	}

	now := time.Now()
	return &models.DatabaseSchemaMetadata{
		DatabaseName: db.DatabaseName,
		DatabaseType: string(dbInstance.DatabaseEngine),
		Tables:       tables,
		Views:        views,
		Functions:    functions,
		LastSyncedAt: &now,
		RawSchema:    dbSchema, // Store the complete schema for detailed queries
	}, nil
}

// extractPostgreSQLSchema extracts schema metadata from a PostgreSQL database
func (s *SchemaMetadataService) extractPostgreSQLSchema(
	ctx context.Context,
	db *models.Database,
	dbInstance *models.DatabaseInstance,
) (*models.DatabaseSchemaMetadata, error) {
	// Connect to PostgreSQL database
	sqlxDB, err := s.connectionMgr.ConnectPostgreSQL(ctx, database.ConnectionParams{
		DatabaseInstance: dbInstance,
		Database:         db,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer func() {
		if closeErr := sqlxDB.Close(); closeErr != nil {
			s.logger.Error("Failed to close PostgreSQL connection", zap.Error(closeErr))
		}
	}()

	s.logger.Info("Extracting PostgreSQL schema metadata",
		zap.String("database_id", db.ID.String()),
		zap.String("database_name", db.DatabaseName),
	)

	// Create extractor and extract schema
	extractor := postgresextractor.NewExtractor(sqlxDB.DB, db.DatabaseName)
	dbSchema, err := extractor.ExtractSchema(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract PostgreSQL schema: %w", err)
	}

	// Convert to our model format (collect tables, views, and PostgreSQL-specific objects from all schemas)
	var tables []models.TableListItem
	var views []models.ViewListItem
	var materializedViews []models.MaterializedViewListItem
	var extensions []models.ExtensionListItem
	var functions []models.FunctionListItem
	var sequences []models.SequenceListItem

	for _, schema := range dbSchema.Schemas {
		for _, table := range schema.Tables {
			tables = append(tables, models.TableListItem{
				Name:      table.Name,
				RowCount:  table.RowCount,
				DataSize:  table.DataSize,
				IndexSize: table.IndexSize,
				TotalSize: table.DataSize + table.IndexSize,
				Engine:    "", // PostgreSQL doesn't have storage engines like MySQL
				Comment:   table.Comment,
			})
		}

		// Extract views
		for _, view := range schema.Views {
			views = append(views, models.ViewListItem{
				Name:       view.Name,
				Definition: view.Definition,
				Comment:    view.Comment,
			})
		}

		// Extract materialized views (PostgreSQL only)
		for _, mview := range schema.MaterializedViews {
			materializedViews = append(materializedViews, models.MaterializedViewListItem{
				Name:       mview.Name,
				Definition: mview.Definition,
				Comment:    mview.Comment,
			})
		}

		// Extract extensions (PostgreSQL only)
		for _, ext := range schema.Extensions {
			extensions = append(extensions, models.ExtensionListItem{
				Name:        ext.Name,
				Schema:      ext.Schema,
				Version:     ext.Version,
				Description: ext.Comment,
			})
		}

		// Extract functions
		for _, fn := range schema.Functions {
			functions = append(functions, models.FunctionListItem{
				Name:       fn.Name,
				Definition: fn.Definition,
				Comment:    fn.Comment,
			})
		}

		// Extract sequences (PostgreSQL only)
		for _, seq := range schema.Sequences {
			sequences = append(sequences, models.SequenceListItem{
				Name:      seq.Name,
				DataType:  seq.DataType,
				Start:     seq.Start,
				Increment: seq.Increment,
				Min:       seq.Min,
				Max:       seq.Max,
				Cache:     seq.Cache,
				Cycle:     seq.Cycle,
				Comment:   seq.Comment,
			})
		}
	}

	now := time.Now()
	return &models.DatabaseSchemaMetadata{
		DatabaseName:      db.DatabaseName,
		DatabaseType:      string(dbInstance.DatabaseEngine),
		Tables:            tables,
		Views:             views,
		MaterializedViews: materializedViews,
		Extensions:        extensions,
		Functions:         functions,
		Sequences:         sequences,
		LastSyncedAt:      &now,
		RawSchema:         dbSchema, // Store the complete schema for detailed queries
	}, nil
}

// convertToTableDetail converts sql-schema Table to our TableDetail model
func convertToTableDetail(table *schemaextract.Table) *models.TableDetail {
	// Convert columns
	columns := make([]models.Column, 0, len(table.Columns))
	for _, col := range table.Columns {
		column := models.Column{
			Name:         col.Name,
			Position:     col.Position,
			Type:         col.Type,
			Nullable:     col.Nullable,
			Comment:      col.Comment,
			CharacterSet: col.CharacterSet,
			Collation:    col.Collation,
			OnUpdate:     col.OnUpdate,
		}

		// Handle default value
		if col.Default != "" {
			column.Default = &col.Default
		}

		// Handle generated columns
		if col.Generation != nil {
			column.IsGenerated = true
			column.GeneratedAs = col.Generation.Expression
		}

		// Handle identity columns (PostgreSQL)
		if col.Identity != nil {
			column.IsIdentity = true
		}

		columns = append(columns, column)
	}

	// Convert indexes
	indexes := make([]models.Index, 0, len(table.Indexes))
	for _, idx := range table.Indexes {
		indexes = append(indexes, models.Index{
			Name:        idx.Name,
			Type:        idx.Type,
			Columns:     idx.Expressions,
			Primary:     idx.Primary,
			Unique:      idx.Unique,
			Visible:     idx.Visible,
			Comment:     idx.Comment,
			WhereClause: idx.WhereClause,
		})
	}

	// Convert foreign keys
	foreignKeys := make([]models.ForeignKey, 0, len(table.ForeignKeys))
	for _, fk := range table.ForeignKeys {
		foreignKeys = append(foreignKeys, models.ForeignKey{
			Name:              fk.Name,
			Columns:           fk.Columns,
			ReferencedTable:   fk.ReferencedTable,
			ReferencedColumns: fk.ReferencedColumns,
			OnDelete:          fk.OnDelete,
			OnUpdate:          fk.OnUpdate,
		})
	}

	// Convert check constraints
	checkConstraints := make([]models.CheckConstraint, 0, len(table.CheckConstraints))
	for _, cc := range table.CheckConstraints {
		checkConstraints = append(checkConstraints, models.CheckConstraint{
			Name:       cc.Name,
			Expression: cc.Expression,
		})
	}

	return &models.TableDetail{
		Name:             table.Name,
		Comment:          table.Comment,
		Engine:           table.Engine,
		Collation:        table.Collation,
		Charset:          table.Charset,
		RowCount:         table.RowCount,
		DataSize:         table.DataSize,
		IndexSize:        table.IndexSize,
		TotalSize:        table.DataSize + table.IndexSize,
		Columns:          columns,
		Indexes:          indexes,
		ForeignKeys:      foreignKeys,
		CheckConstraints: checkConstraints,
	}
}
