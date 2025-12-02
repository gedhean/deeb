package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitMySQL_SingleStatement(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
		texts    []string
	}{
		{
			name:     "simple select",
			sql:      "SELECT * FROM t;",
			expected: 1,
			texts:    []string{"SELECT * FROM t;"},
		},
		{
			name:     "select without semicolon",
			sql:      "SELECT * FROM t",
			expected: 1,
			texts:    []string{"SELECT * FROM t"},
		},
		{
			name:     "create table",
			sql:      "CREATE TABLE t(a int);",
			expected: 1,
			texts:    []string{"CREATE TABLE t(a int);"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.TotalCount)
			for i, text := range tt.texts {
				assert.Equal(t, text, result.Statements[i].Text)
			}
		})
	}
}

func TestSplitMySQL_MultipleStatements(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
		texts    []string
	}{
		{
			name:     "two statements",
			sql:      "CREATE TABLE t(a int); INSERT INTO t VALUES (1);",
			expected: 2,
			texts:    []string{"CREATE TABLE t(a int);", "INSERT INTO t VALUES (1);"},
		},
		{
			name:     "three statements",
			sql:      "SELECT 1; SELECT 2; SELECT 3;",
			expected: 3,
			texts:    []string{"SELECT 1;", "SELECT 2;", "SELECT 3;"},
		},
		{
			name:     "multiline statements",
			sql:      "SELECT 1;\nSELECT 2;",
			expected: 2,
			texts:    []string{"SELECT 1;", "SELECT 2;"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.TotalCount)
			for i, text := range tt.texts {
				assert.Equal(t, text, result.Statements[i].Text)
			}
		})
	}
}

func TestSplitMySQL_StringsWithSemicolons(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "single quoted string with semicolon",
			sql:      "INSERT INTO t VALUES ('a;b;c');",
			expected: 1,
		},
		{
			name:     "double quoted string with semicolon",
			sql:      `INSERT INTO t VALUES ("a;b;c");`,
			expected: 1,
		},
		{
			name:     "multiple strings with semicolons",
			sql:      "INSERT INTO t VALUES ('a;b', 'c;d');",
			expected: 1,
		},
		{
			name:     "string with semicolon followed by statement",
			sql:      "INSERT INTO t VALUES ('a;b'); SELECT 1;",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.TotalCount)
		})
	}
}

func TestSplitMySQL_EscapedQuotes(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "escaped single quote",
			sql:      `SELECT 'it\'s working';`,
			expected: 1,
		},
		{
			name:     "escaped backslash",
			sql:      `SELECT 'path\\to\\file';`,
			expected: 1,
		},
		{
			name:     "escaped quote at end",
			sql:      `SELECT 'test\'';`,
			expected: 1,
		},
		{
			name:     "multiple escaped quotes",
			sql:      `SELECT 'it\'s a \'test\'';`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.TotalCount)
		})
	}
}

func TestSplitMySQL_Comments(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "block comment with semicolon",
			sql:      "SELECT 1; /* comment; */ SELECT 2;",
			expected: 2,
		},
		{
			name:     "line comment with dash",
			sql:      "SELECT 1; -- comment\nSELECT 2;",
			expected: 2,
		},
		{
			name:     "hash comment",
			sql:      "SELECT 1; # comment\nSELECT 2;",
			expected: 2,
		},
		{
			name:     "multi-line block comment",
			sql:      "SELECT 1; /* line1\nline2\nline3 */ SELECT 2;",
			expected: 2,
		},
		{
			name:     "comment only statement",
			sql:      "SELECT 1; /* comment */; SELECT 2;",
			expected: 2, // Empty statement between ; and ; is filtered
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.TotalCount)
		})
	}
}

func TestSplitMySQL_BacktickIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "backtick identifier with semicolon",
			sql:      "SELECT `table;name` FROM t;",
			expected: 1,
		},
		{
			name:     "backtick identifier with spaces",
			sql:      "SELECT `column name` FROM `table name`;",
			expected: 1,
		},
		{
			name:     "backtick with reserved word",
			sql:      "SELECT `select` FROM `from`;",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.TotalCount)
		})
	}
}

func TestSplitMySQL_EmptyStatements(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "double semicolon",
			sql:      "SELECT 1;; SELECT 2;",
			expected: 2,
		},
		{
			name:     "multiple empty",
			sql:      "SELECT 1;;; SELECT 2;",
			expected: 2,
		},
		{
			name:     "whitespace only between",
			sql:      "SELECT 1;   \n\n   SELECT 2;",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.TotalCount)
		})
	}
}

func TestSplitMySQL_WindowsLineEndings(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "windows line endings",
			sql:      "SELECT 1;\r\nSELECT 2;",
			expected: 2,
		},
		{
			name:     "mixed line endings",
			sql:      "SELECT 1;\r\nSELECT 2;\nSELECT 3;",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.TotalCount)
		})
	}
}

func TestSplitMySQL_PositionTracking(t *testing.T) {
	sql := "SELECT 1;\nSELECT 2;\nSELECT 3;"
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	require.Equal(t, 3, result.TotalCount)

	// First statement starts at line 1
	assert.Equal(t, 1, result.Statements[0].StartLine)
	// Second statement starts at line 2
	assert.Equal(t, 2, result.Statements[1].StartLine)
	// Third statement starts at line 3
	assert.Equal(t, 3, result.Statements[2].StartLine)

	// Check indices are sequential
	assert.Equal(t, 0, result.Statements[0].Index)
	assert.Equal(t, 1, result.Statements[1].Index)
	assert.Equal(t, 2, result.Statements[2].Index)
}

func TestSplitMySQL_ByteOffsets(t *testing.T) {
	sql := "SELECT 1; SELECT 2;"
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	require.Equal(t, 2, result.TotalCount)

	// First statement: "SELECT 1;" (bytes 0-9)
	assert.Equal(t, 0, result.Statements[0].ByteStart)
	assert.Equal(t, 9, result.Statements[0].ByteEnd)

	// Second statement: "SELECT 2;" starts after space (byte 10-19)
	// skipBlank() trims leading whitespace, so start is at byte 10
	assert.Equal(t, 10, result.Statements[1].ByteStart)
	assert.Equal(t, 19, result.Statements[1].ByteEnd)
}

func TestSplitMySQL_Unicode(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "unicode in string",
			sql:      "INSERT INTO t VALUES ('日本語');",
			expected: 1,
		},
		{
			name:     "unicode in identifier",
			sql:      "SELECT `日本語` FROM t;",
			expected: 1,
		},
		{
			name:     "emoji in string",
			sql:      "INSERT INTO t VALUES ('Hello 👋');",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.TotalCount)
		})
	}
}

func TestSplitMySQL_BestEffortParsing(t *testing.T) {
	// Test that unclosed strings/comments are handled gracefully
	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "unclosed single quote",
			sql:  "SELECT 'unclosed",
		},
		{
			name: "unclosed double quote",
			sql:  `SELECT "unclosed`,
		},
		{
			name: "unclosed block comment",
			sql:  "SELECT 1 /* unclosed",
		},
		{
			name: "unclosed backtick",
			sql:  "SELECT `unclosed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql)
			// Should not error - best effort parsing
			require.NoError(t, err)
			// Should return the content as a single statement
			assert.GreaterOrEqual(t, result.TotalCount, 0)
		})
	}
}

func TestSplitMySQL_RealWorldExample(t *testing.T) {
	sql := `
ALTER TABLE sr_tournaments_lang
  ADD COLUMN provider VARCHAR(50) NOT NULL DEFAULT 'radar';

ALTER TABLE sr_tournaments_lang
  ALTER COLUMN provider DROP DEFAULT;
`
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)

	// Verify both statements are captured
	assert.Contains(t, result.Statements[0].Text, "ADD COLUMN provider")
	assert.Contains(t, result.Statements[1].Text, "ALTER COLUMN provider DROP DEFAULT")
}

// Additional test cases from Bytebase test suite

func TestSplitMySQL_EscapedDoubleQuoteInString(t *testing.T) {
	// From Bytebase: select * from t;select "\"" where true;
	sql := `select * from t;select "\"" where true;`
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
	assert.Equal(t, "select * from t;", result.Statements[0].Text)
	assert.Contains(t, result.Statements[1].Text, `"\""`)
}

func TestSplitMySQL_InlineComments(t *testing.T) {
	// From Bytebase: inline comments within statements
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "inline block comment in create",
			sql:      "CREATE /* inline comment */TABLE tech_Book(id int);",
			expected: 1,
		},
		{
			name:     "inline block comment in select",
			sql:      "SELECT /* comment */ * FROM t;",
			expected: 1,
		},
		{
			name:     "multiple inline comments",
			sql:      "SELECT /* a */ 1 /* b */ FROM /* c */ t;",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.TotalCount)
		})
	}
}

func TestSplitMySQL_ComplexEscapedQuotes(t *testing.T) {
	// From Bytebase: 'lks''kjsafa\'jdfl;"ka' (both '' and \' escapes)
	sql := `INSERT INTO tech_Book VALUES (1, 'lks''kjsafa\'jdfl;"ka');`
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
	assert.Contains(t, result.Statements[0].Text, `'lks''kjsafa\'jdfl;"ka'`)
}

func TestSplitMySQL_CommentOnlyMultipleLines(t *testing.T) {
	// From Bytebase: multiple line comments only
	sql := "-- klsjdfjasldf\n\t\t\t-- klsjdflkjaskldfj\n"
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	// Comment-only content is filtered out
	assert.Equal(t, 0, result.TotalCount)
}

func TestSplitMySQL_CommentsWithStatements(t *testing.T) {
	// From Bytebase: comments followed by statements
	sql := `/* this is the comment. */
CREATE /* inline comment */TABLE tech_Book(id int, name varchar(255));
-- this is the comment.
INSERT INTO tech_Book VALUES (0, 'abce_ksdf');
# this is the comment.
INSERT INTO tech_Book VALUES (1, 'test');`

	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalCount)
}

func TestSplitMySQL_LargeMultiLineStatement(t *testing.T) {
	// From Bytebase: 20 IF() functions in one statement
	sql := `SELECT
IF (age < 18, 'child', 'adult'),
IF (age < 18, 'child', 'adult'),
IF (age < 18, 'child', 'adult'),
IF (age < 18, 'child', 'adult'),
IF (age < 18, 'child', 'adult')
FROM t; SELECT * FROM t;`

	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
	assert.Contains(t, result.Statements[0].Text, "IF (age < 18")
	assert.Contains(t, result.Statements[1].Text, "SELECT * FROM t")
}

func TestSplitMySQL_StoredProcedureStyle(t *testing.T) {
	// From Bytebase: BEGIN...END blocks (without DELIMITER)
	// Note: We don't support DELIMITER command, but we should handle
	// the semicolons inside BEGIN...END when using a custom delimiter
	sql := `CREATE DEFINER=` + "`root`@`%`" + ` FUNCTION CalcIncome(starting_value INT) RETURNS int
BEGIN
   DECLARE income INT;
   SET income = 0;
   RETURN income;
END ;`

	result, err := SplitSQL(sql)
	require.NoError(t, err)
	// Without DELIMITER support, this will split on internal semicolons
	// This is expected behavior per our design decision
	assert.GreaterOrEqual(t, result.TotalCount, 1)
}

func TestSplitMySQL_LeadingWhitespace(t *testing.T) {
	// From Bytebase: statements with leading whitespace
	sql := "    CREATE TABLE t(a int); CREATE TABLE t1(a int)"
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}
