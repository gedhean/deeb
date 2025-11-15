package models

import (
	"time"

	schemaextract "github.com/nsxbet/sql-schema"
)

// TableListItem represents a table in the database with basic statistics
type TableListItem struct {
	Name      string     `json:"name"`
	RowCount  int64      `json:"row_count"`
	DataSize  int64      `json:"data_size"`  // Size in bytes
	IndexSize int64      `json:"index_size"` // Size in bytes
	TotalSize int64      `json:"total_size"` // DataSize + IndexSize
	Engine    string     `json:"engine"`     // MySQL: InnoDB, MyISAM; PostgreSQL: N/A
	Comment   string     `json:"comment"`
	CreatedAt *time.Time `json:"created_at,omitempty"` // Table creation time (if available)
	UpdatedAt *time.Time `json:"updated_at,omitempty"` // Table last update time (if available)
}

// TableDetail represents detailed information about a table
type TableDetail struct {
	Name             string            `json:"name"`
	Comment          string            `json:"comment"`
	Engine           string            `json:"engine"` // MySQL: InnoDB, MyISAM; PostgreSQL: N/A
	Collation        string            `json:"collation"`
	Charset          string            `json:"charset"`
	RowCount         int64             `json:"row_count"`
	DataSize         int64             `json:"data_size"`  // Size in bytes
	IndexSize        int64             `json:"index_size"` // Size in bytes
	TotalSize        int64             `json:"total_size"` // DataSize + IndexSize
	CreatedAt        *time.Time        `json:"created_at,omitempty"`
	UpdatedAt        *time.Time        `json:"updated_at,omitempty"`
	Columns          []Column          `json:"columns"`
	Indexes          []Index           `json:"indexes"`
	ForeignKeys      []ForeignKey      `json:"foreign_keys"`
	CheckConstraints []CheckConstraint `json:"check_constraints"`
}

// Column represents a table column
type Column struct {
	Name         string  `json:"name"`
	Position     int32   `json:"position"`
	Type         string  `json:"type"`
	Nullable     bool    `json:"nullable"`
	Default      *string `json:"default,omitempty"`
	Comment      string  `json:"comment"`
	CharacterSet string  `json:"character_set,omitempty"`
	Collation    string  `json:"collation,omitempty"`
	OnUpdate     string  `json:"on_update,omitempty"`    // MySQL: ON UPDATE CURRENT_TIMESTAMP
	IsGenerated  bool    `json:"is_generated,omitempty"` // Generated/computed column
	GeneratedAs  string  `json:"generated_as,omitempty"` // Generation expression
	IsIdentity   bool    `json:"is_identity,omitempty"`  // PostgreSQL IDENTITY column
}

// Index represents a table index
type Index struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`    // BTREE, HASH, FULLTEXT, SPATIAL, GIN, GIST, etc.
	Columns     []string `json:"columns"` // Column names in index
	Primary     bool     `json:"primary"`
	Unique      bool     `json:"unique"`
	Visible     bool     `json:"visible"`
	Comment     string   `json:"comment"`
	Cardinality int64    `json:"cardinality,omitempty"`  // Approximate number of unique values
	WhereClause string   `json:"where_clause,omitempty"` // Partial index condition (PostgreSQL)
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	Name              string   `json:"name"`
	Columns           []string `json:"columns"` // Columns in this table
	ReferencedTable   string   `json:"referenced_table"`
	ReferencedColumns []string `json:"referenced_columns"`
	OnDelete          string   `json:"on_delete"` // CASCADE, SET NULL, RESTRICT, NO ACTION
	OnUpdate          string   `json:"on_update"` // CASCADE, SET NULL, RESTRICT, NO ACTION
}

// CheckConstraint represents a check constraint
type CheckConstraint struct {
	Name       string   `json:"name"`
	Expression string   `json:"expression"`        // The check constraint expression
	Columns    []string `json:"columns,omitempty"` // Columns referenced in the constraint
}

// ViewListItem represents a view in the database
type ViewListItem struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
	Comment    string `json:"comment"`
}

// MaterializedViewListItem represents a materialized view in the database (PostgreSQL only)
type MaterializedViewListItem struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
	Comment    string `json:"comment"`
}

// ExtensionListItem represents a PostgreSQL extension
type ExtensionListItem struct {
	Name        string `json:"name"`
	Schema      string `json:"schema"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// FunctionListItem represents a database function
type FunctionListItem struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
	Comment    string `json:"comment"`
}

// SequenceListItem represents a PostgreSQL sequence
type SequenceListItem struct {
	Name      string `json:"name"`
	DataType  string `json:"data_type"`
	Start     int64  `json:"start"`
	Increment int64  `json:"increment"`
	Min       int64  `json:"min"`
	Max       int64  `json:"max"`
	Cache     int64  `json:"cache"`
	Cycle     bool   `json:"cycle"`
	Comment   string `json:"comment"`
}

// DatabaseSchemaMetadata represents the complete schema metadata response
type DatabaseSchemaMetadata struct {
	DatabaseName      string                     `json:"database_name"`
	DatabaseType      string                     `json:"database_type"` // mysql or postgresql
	Tables            []TableListItem            `json:"tables"`
	Views             []ViewListItem             `json:"views"`
	MaterializedViews []MaterializedViewListItem `json:"materialized_views,omitempty"` // PostgreSQL only
	Extensions        []ExtensionListItem        `json:"extensions,omitempty"`         // PostgreSQL only
	Functions         []FunctionListItem         `json:"functions,omitempty"`
	Sequences         []SequenceListItem         `json:"sequences,omitempty"` // PostgreSQL only
	LastSyncedAt      *time.Time                 `json:"last_synced_at,omitempty"`

	// RawSchema stores the complete schema extraction result from sql-schema library
	// This includes full table details (columns, indexes, foreign keys, etc.)
	// It's stored in the cache but not exposed in the API response (json:"-")
	RawSchema *schemaextract.DatabaseSchema `json:"-"`
}
