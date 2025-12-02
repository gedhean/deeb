// Package parser provides SQL statement splitting functionality for MySQL and PostgreSQL.
// It accurately detects statement boundaries while handling strings, comments,
// and dialect-specific syntax like PostgreSQL dollar-quoted strings.
package parser

import (
	"fmt"

	"github.com/nsxbet/deeb/internal/parser/base"
	"github.com/nsxbet/deeb/internal/parser/mysql"
	"github.com/nsxbet/deeb/internal/parser/postgres"
)

// Re-export types from base package for convenience.
type (
	// Dialect represents supported database engines.
	Dialect = base.Dialect
	// SingleSQL represents a parsed SQL statement with position information.
	SingleSQL = base.SingleSQL
	// ParseResult contains the result of parsing SQL content.
	ParseResult = base.ParseResult
)

// Dialect constants.
const (
	DialectMySQL    = base.DialectMySQL
	DialectPostgres = base.DialectPostgres
)

// SplitSQL splits SQL content into individual statements based on the dialect.
// It handles strings, comments, and dialect-specific syntax appropriately.
//
// For MySQL:
//   - Semicolon (;) as statement delimiter
//   - Single quotes ('), double quotes (") for strings
//   - Backticks (`) for identifiers
//   - Comments: /* */, --, #
//
// For PostgreSQL:
//   - Semicolon (;) as statement delimiter
//   - Single quotes (') for strings
//   - Double quotes (") for identifiers
//   - Dollar-quoted strings ($$...$$, $tag$...$tag$)
//   - Comments: /* */, --
func SplitSQL(sql string, dialect Dialect) (*ParseResult, error) {
	switch dialect {
	case DialectMySQL:
		return mysql.SplitSQL(sql)
	case DialectPostgres:
		return postgres.SplitSQL(sql)
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialect)
	}
}
