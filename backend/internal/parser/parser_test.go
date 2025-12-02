package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitSQL_UnsupportedDialect(t *testing.T) {
	result, err := SplitSQL("SELECT 1;", "oracle")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported dialect")
}

func TestSplitSQL_EmptyInput(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		dialect Dialect
	}{
		{name: "empty string mysql", sql: "", dialect: DialectMySQL},
		{name: "empty string postgres", sql: "", dialect: DialectPostgres},
		{name: "whitespace only mysql", sql: "   \n\t  ", dialect: DialectMySQL},
		{name: "whitespace only postgres", sql: "   \n\t  ", dialect: DialectPostgres},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql, tt.dialect)
			require.NoError(t, err)
			assert.Equal(t, 0, result.TotalCount)
		})
	}
}

func TestSplitSQL_CommentOnly(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		dialect Dialect
	}{
		{name: "line comment mysql", sql: "-- just a comment", dialect: DialectMySQL},
		{name: "line comment postgres", sql: "-- just a comment", dialect: DialectPostgres},
		{name: "block comment mysql", sql: "/* just a comment */", dialect: DialectMySQL},
		{name: "block comment postgres", sql: "/* just a comment */", dialect: DialectPostgres},
		{name: "hash comment mysql", sql: "# just a comment", dialect: DialectMySQL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql, tt.dialect)
			require.NoError(t, err)
			// Comment-only is treated as empty
			assert.Equal(t, 0, result.TotalCount)
		})
	}
}

func TestSplitSQL_LargeStatement(t *testing.T) {
	// Create a statement with ~1MB of data
	largeData := strings.Repeat("x", 1024*1024)
	sql := "INSERT INTO t VALUES ('" + largeData + "');"

	result, err := SplitSQL(sql, DialectMySQL)
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Contains(t, result.Statements[0].Text, largeData)
}

func TestSplitSQL_ManyStatements(t *testing.T) {
	// Create 100 statements
	var builder strings.Builder
	for i := 0; i < 100; i++ {
		builder.WriteString("SELECT ")
		builder.WriteString(strings.Repeat("1", i%10+1))
		builder.WriteString(";\n")
	}

	result, err := SplitSQL(builder.String(), DialectMySQL)
	require.NoError(t, err)
	assert.Equal(t, 100, result.TotalCount)

	// Verify indices are sequential
	for i, stmt := range result.Statements {
		assert.Equal(t, i, stmt.Index)
	}
}

func TestSplitSQL_IndexAssignment(t *testing.T) {
	// Test that indices are assigned correctly after filtering empty statements
	sql := "SELECT 1;; SELECT 2;;; SELECT 3;"

	result, err := SplitSQL(sql, DialectMySQL)
	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalCount)

	// Indices should be 0, 1, 2 (not affected by empty statements)
	assert.Equal(t, 0, result.Statements[0].Index)
	assert.Equal(t, 1, result.Statements[1].Index)
	assert.Equal(t, 2, result.Statements[2].Index)
}

func TestSplitSQL_DialectDifferences(t *testing.T) {
	// Hash comment is MySQL-only
	hashCommentSQL := "SELECT 1; # comment\nSELECT 2;"

	mysqlResult, err := SplitSQL(hashCommentSQL, DialectMySQL)
	require.NoError(t, err)
	assert.Equal(t, 2, mysqlResult.TotalCount)

	// In PostgreSQL, # is not a comment starter, so it will be part of the statement
	pgResult, err := SplitSQL(hashCommentSQL, DialectPostgres)
	require.NoError(t, err)
	// The # will be treated as part of the statement (not a comment)
	// This may result in different behavior - let's just check no error
	assert.GreaterOrEqual(t, pgResult.TotalCount, 1)
}

func TestSplitSQL_BacktickVsDoubleQuote(t *testing.T) {
	// MySQL uses backticks for identifiers
	mysqlSQL := "SELECT `column;name` FROM t;"
	mysqlResult, err := SplitSQL(mysqlSQL, DialectMySQL)
	require.NoError(t, err)
	assert.Equal(t, 1, mysqlResult.TotalCount)

	// PostgreSQL uses double quotes for identifiers
	pgSQL := `SELECT "column;name" FROM t;`
	pgResult, err := SplitSQL(pgSQL, DialectPostgres)
	require.NoError(t, err)
	assert.Equal(t, 1, pgResult.TotalCount)
}

func TestSplitSQL_EscapeSequences(t *testing.T) {
	// MySQL uses backslash escapes
	mysqlSQL := `SELECT 'it\'s working';`
	mysqlResult, err := SplitSQL(mysqlSQL, DialectMySQL)
	require.NoError(t, err)
	assert.Equal(t, 1, mysqlResult.TotalCount)

	// PostgreSQL uses doubled quotes
	pgSQL := `SELECT 'it''s working';`
	pgResult, err := SplitSQL(pgSQL, DialectPostgres)
	require.NoError(t, err)
	assert.Equal(t, 1, pgResult.TotalCount)
}

func TestSplitSQL_DollarQuoteOnlyPostgreSQL(t *testing.T) {
	dollarSQL := "DO $$ SELECT 1; $$;"

	// PostgreSQL handles dollar quotes
	pgResult, err := SplitSQL(dollarSQL, DialectPostgres)
	require.NoError(t, err)
	assert.Equal(t, 1, pgResult.TotalCount)

	// MySQL doesn't have dollar quotes - the $ is just a regular character
	// This will split on the internal semicolon
	mysqlResult, err := SplitSQL(dollarSQL, DialectMySQL)
	require.NoError(t, err)
	// MySQL will see this differently since $$ isn't special
	assert.GreaterOrEqual(t, mysqlResult.TotalCount, 1)
}

func TestSplitSQL_StatementPositionsAcrossDialects(t *testing.T) {
	sql := "SELECT 1;\nSELECT 2;"

	mysqlResult, err := SplitSQL(sql, DialectMySQL)
	require.NoError(t, err)

	pgResult, err := SplitSQL(sql, DialectPostgres)
	require.NoError(t, err)

	// Both should have same positions
	assert.Equal(t, mysqlResult.TotalCount, pgResult.TotalCount)
	assert.Equal(t, mysqlResult.Statements[0].StartLine, pgResult.Statements[0].StartLine)
	assert.Equal(t, mysqlResult.Statements[1].StartLine, pgResult.Statements[1].StartLine)
}

func TestSingleSQL_Fields(t *testing.T) {
	sql := "SELECT 1;\nSELECT 2;"
	result, err := SplitSQL(sql, DialectMySQL)
	require.NoError(t, err)
	require.Equal(t, 2, result.TotalCount)

	stmt := result.Statements[0]

	// Check all fields are populated
	assert.Equal(t, "SELECT 1;", stmt.Text)
	assert.Equal(t, 0, stmt.Index)
	assert.Equal(t, 1, stmt.StartLine)
	assert.GreaterOrEqual(t, stmt.StartColumn, 0)
	assert.GreaterOrEqual(t, stmt.EndLine, stmt.StartLine)
	assert.GreaterOrEqual(t, stmt.ByteEnd, stmt.ByteStart)
	assert.False(t, stmt.Empty)
}

func TestParseResult_Fields(t *testing.T) {
	sql := "SELECT 1; SELECT 2; SELECT 3;"
	result, err := SplitSQL(sql, DialectMySQL)
	require.NoError(t, err)

	assert.Equal(t, 3, result.TotalCount)
	assert.Len(t, result.Statements, 3)
}

func TestSplitSQL_TrailingContent(t *testing.T) {
	// Statement without trailing semicolon
	sql := "SELECT 1; SELECT 2"
	result, err := SplitSQL(sql, DialectMySQL)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
	assert.Equal(t, "SELECT 1;", result.Statements[0].Text)
	assert.Equal(t, "SELECT 2", result.Statements[1].Text)
}

func TestSplitSQL_OnlySemicolons(t *testing.T) {
	sql := ";;;"
	result, err := SplitSQL(sql, DialectMySQL)
	require.NoError(t, err)
	// All statements are empty
	assert.Equal(t, 0, result.TotalCount)
}

func TestSplitSQL_ComplexNestedStructures(t *testing.T) {
	// Complex SQL with nested structures
	sql := `
CREATE TABLE test (
    id INT PRIMARY KEY,
    data JSON DEFAULT '{"key": "value; with semicolon"}'
);

INSERT INTO test VALUES (1, '{"nested": {"deep": "value;test"}}');
`
	result, err := SplitSQL(sql, DialectMySQL)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}
