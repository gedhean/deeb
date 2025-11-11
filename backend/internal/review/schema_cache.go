package review

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/sql-reviewer/pkg/types"
	"go.uber.org/zap"
)

// SchemaCache caches database schema metadata for sql-reviewer
type SchemaCache struct {
	cache  map[uuid.UUID]*cacheEntry
	mu     sync.RWMutex
	ttl    time.Duration
	logger *zap.Logger
}

type cacheEntry struct {
	schema    *types.DatabaseSchemaMetadata
	timestamp time.Time
}

// NewSchemaCache creates a new schema cache with the specified TTL
func NewSchemaCache(ttl time.Duration, logger *zap.Logger) *SchemaCache {
	return &SchemaCache{
		cache:  make(map[uuid.UUID]*cacheEntry),
		ttl:    ttl,
		logger: logger,
	}
}

// Get retrieves schema from cache or returns nil if not found or expired
func (c *SchemaCache) Get(ctx context.Context, databaseID uuid.UUID) *types.DatabaseSchemaMetadata {
	c.mu.RLock()
	entry, exists := c.cache[databaseID]
	c.mu.RUnlock()

	if !exists {
		c.logger.Debug("Schema cache miss", zap.String("database_id", databaseID.String()))
		return nil
	}

	// Check if expired
	if time.Since(entry.timestamp) > c.ttl {
		c.logger.Debug("Schema cache expired", zap.String("database_id", databaseID.String()))
		c.Invalidate(databaseID)
		return nil
	}

	c.logger.Debug("Schema cache hit", zap.String("database_id", databaseID.String()))
	return entry.schema
}

// Set stores schema in cache
func (c *SchemaCache) Set(databaseID uuid.UUID, schema *types.DatabaseSchemaMetadata) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[databaseID] = &cacheEntry{
		schema:    schema,
		timestamp: time.Now(),
	}

	c.logger.Debug("Schema cached",
		zap.String("database_id", databaseID.String()),
		zap.Int("schemas", len(schema.Schemas)))
}

// Invalidate removes schema from cache
func (c *SchemaCache) Invalidate(databaseID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, databaseID)
	c.logger.Debug("Schema cache invalidated", zap.String("database_id", databaseID.String()))
}

// Clear removes all entries from cache
func (c *SchemaCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[uuid.UUID]*cacheEntry)
	c.logger.Info("Schema cache cleared")
}

// Size returns the number of cached schemas
func (c *SchemaCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// SchemaFetcher fetches database schema metadata
type SchemaFetcher struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewSchemaFetcher creates a new schema fetcher
func NewSchemaFetcher(db *sql.DB, logger *zap.Logger) *SchemaFetcher {
	return &SchemaFetcher{
		db:     db,
		logger: logger,
	}
}

// FetchSchema retrieves complete schema metadata for a database
func (f *SchemaFetcher) FetchSchema(
	ctx context.Context,
	dbType models.DatabaseEngine,
	dbName string,
) (*types.DatabaseSchemaMetadata, error) {
	start := time.Now()
	defer func() {
		f.logger.Info("Schema fetched",
			zap.String("database", dbName),
			zap.String("type", string(dbType)),
			zap.Duration("duration", time.Since(start)))
	}()

	switch dbType {
	case models.DatabaseEngineMySQL:
		return f.fetchMySQLSchema(ctx, dbName)
	case models.DatabaseEnginePostgreSQL:
		return f.fetchPostgreSQLSchema(ctx, dbName)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// fetchMySQLSchema fetches schema for MySQL database
func (f *SchemaFetcher) fetchMySQLSchema(ctx context.Context, dbName string) (*types.DatabaseSchemaMetadata, error) {
	schema := &types.DatabaseSchemaMetadata{
		Name: dbName,
	}

	// Fetch tables
	tables, err := f.fetchMySQLTables(ctx, dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tables: %w", err)
	}

	// For each table, fetch columns and indexes
	for _, table := range tables {
		columns, err := f.fetchMySQLColumns(ctx, dbName, table.Name)
		if err != nil {
			f.logger.Warn("Failed to fetch columns for table",
				zap.String("table", table.Name),
				zap.Error(err))
			continue
		}
		table.Columns = columns

		indexes, err := f.fetchMySQLIndexes(ctx, dbName, table.Name)
		if err != nil {
			f.logger.Warn("Failed to fetch indexes for table",
				zap.String("table", table.Name),
				zap.Error(err))
			continue
		}
		table.Indexes = indexes
	}

	schemaMetadata := &types.SchemaMetadata{
		Name:   dbName,
		Tables: tables,
	}
	schema.Schemas = []*types.SchemaMetadata{schemaMetadata}

	return schema, nil
}

// fetchMySQLTables fetches table list for MySQL
func (f *SchemaFetcher) fetchMySQLTables(ctx context.Context, dbName string) ([]*types.TableMetadata, error) {
	query := `
		SELECT table_name, engine, table_rows, table_comment
		FROM information_schema.tables
		WHERE table_schema = ?
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := f.db.QueryContext(ctx, query, dbName)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	tables := []*types.TableMetadata{}
	for rows.Next() {
		var table types.TableMetadata
		var engine, comment sql.NullString
		var rowCount sql.NullInt64

		if err := rows.Scan(&table.Name, &engine, &rowCount, &comment); err != nil {
			return nil, err
		}

		if engine.Valid {
			table.Engine = engine.String
		}
		if rowCount.Valid {
			table.RowCount = rowCount.Int64
		}
		if comment.Valid {
			table.Comment = comment.String
		}

		tables = append(tables, &table)
	}

	return tables, rows.Err()
}

// fetchMySQLColumns fetches columns for a MySQL table
func (f *SchemaFetcher) fetchMySQLColumns(ctx context.Context, dbName, tableName string) ([]*types.ColumnMetadata, error) {
	query := `
		SELECT column_name, column_type, is_nullable, column_default, column_comment
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position
	`

	rows, err := f.db.QueryContext(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	columns := []*types.ColumnMetadata{}
	for rows.Next() {
		var col types.ColumnMetadata
		var isNullable string
		var columnDefault, comment sql.NullString

		if err := rows.Scan(&col.Name, &col.Type, &isNullable, &columnDefault, &comment); err != nil {
			return nil, err
		}

		col.Nullable = (isNullable == "YES")
		if columnDefault.Valid {
			col.HasDefault = true
			col.DefaultString = columnDefault.String
		}
		if comment.Valid {
			col.Comment = comment.String
		}

		columns = append(columns, &col)
	}

	return columns, rows.Err()
}

// fetchMySQLIndexes fetches indexes for a MySQL table
func (f *SchemaFetcher) fetchMySQLIndexes(ctx context.Context, dbName, tableName string) ([]*types.IndexMetadata, error) {
	query := `
		SELECT DISTINCT index_name, non_unique, index_type
		FROM information_schema.statistics
		WHERE table_schema = ? AND table_name = ?
		ORDER BY index_name
	`

	rows, err := f.db.QueryContext(ctx, query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	indexes := []*types.IndexMetadata{}
	for rows.Next() {
		var idx types.IndexMetadata
		var nonUnique int
		var indexType string

		if err := rows.Scan(&idx.Name, &nonUnique, &indexType); err != nil {
			return nil, err
		}

		idx.Unique = (nonUnique == 0)
		idx.Type = indexType

		indexes = append(indexes, &idx)
	}

	return indexes, rows.Err()
}

// fetchPostgreSQLSchema fetches schema for PostgreSQL database
func (f *SchemaFetcher) fetchPostgreSQLSchema(ctx context.Context, dbName string) (*types.DatabaseSchemaMetadata, error) {
	// TODO: Implement PostgreSQL schema fetching
	// For now, return empty schema
	return &types.DatabaseSchemaMetadata{
		Name:    dbName,
		Schemas: []*types.SchemaMetadata{},
	}, nil
}
