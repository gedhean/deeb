// Package base provides shared types and utilities for SQL statement splitting.
package base

// Dialect represents supported database engines.
type Dialect string

const (
	// DialectMySQL represents MySQL database dialect.
	DialectMySQL Dialect = "mysql"
	// DialectPostgres represents PostgreSQL database dialect.
	DialectPostgres Dialect = "postgres"
)

// SingleSQL represents a parsed SQL statement with position information.
type SingleSQL struct {
	// Text is the SQL statement text (including trailing semicolon if present).
	Text string
	// Index is the 0-based statement index in the original SQL.
	Index int
	// StartLine is the 1-based line number where the statement starts.
	StartLine int
	// StartColumn is the 0-based column number where the statement starts.
	StartColumn int
	// EndLine is the 1-based line number where the statement ends.
	EndLine int
	// EndColumn is the 0-based column number where the statement ends (exclusive).
	EndColumn int
	// ByteStart is the byte offset where the statement starts in the original SQL.
	ByteStart int
	// ByteEnd is the byte offset where the statement ends in the original SQL.
	ByteEnd int
	// Empty is true if the statement contains only whitespace or comments.
	Empty bool
}

// ParseResult contains the result of parsing SQL content.
type ParseResult struct {
	// Statements is the list of parsed SQL statements.
	Statements []SingleSQL
	// TotalCount is the total number of non-empty statements.
	TotalCount int
}

// FilterAndIndex filters out empty statements and assigns sequential indices.
func FilterAndIndex(statements []SingleSQL) []SingleSQL {
	var result []SingleSQL
	index := 0
	for _, stmt := range statements {
		if !stmt.Empty {
			stmt.Index = index
			result = append(result, stmt)
			index++
		}
	}
	return result
}
