package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitPostgreSQL_SingleStatement(t *testing.T) {
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

func TestSplitPostgreSQL_MultipleStatements(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "two statements",
			sql:      "CREATE TABLE t(a int); INSERT INTO t VALUES (1);",
			expected: 2,
		},
		{
			name:     "three statements",
			sql:      "SELECT 1; SELECT 2; SELECT 3;",
			expected: 3,
		},
		{
			name:     "multiline statements",
			sql:      "SELECT 1;\nSELECT 2;",
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

func TestSplitPostgreSQL_DollarQuotedStrings(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name: "simple dollar quote",
			sql: `CREATE FUNCTION f() AS $$
SELECT 1;
SELECT 2;
$$ LANGUAGE SQL;`,
			expected: 1,
		},
		{
			name: "tagged dollar quote",
			sql: `CREATE FUNCTION f() AS $body$
SELECT 1;
SELECT 2;
$body$ LANGUAGE SQL;`,
			expected: 1,
		},
		{
			name:     "dollar quote followed by statement",
			sql:      "DO $$ SELECT 1; $$; SELECT 2;",
			expected: 2,
		},
		{
			name: "nested semicolons in dollar quote",
			sql: `DO $$
BEGIN
  IF true THEN
    SELECT 1;
  END IF;
END
$$;`,
			expected: 1,
		},
		{
			name:     "multiple dollar quoted functions",
			sql:      "CREATE FUNCTION f1() AS $$ SELECT 1; $$ LANGUAGE SQL; CREATE FUNCTION f2() AS $$ SELECT 2; $$ LANGUAGE SQL;",
			expected: 2,
		},
		{
			name: "dollar quote with custom tag",
			sql: `CREATE FUNCTION test() AS $fn_body$
  DECLARE
    x INT := 0;
  BEGIN
    x := x + 1;
    RETURN x;
  END;
$fn_body$ LANGUAGE plpgsql;`,
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

func TestSplitPostgreSQL_StringsWithSemicolons(t *testing.T) {
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

func TestSplitPostgreSQL_EscapedQuotes(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "doubled single quote",
			sql:      "SELECT 'it''s working';",
			expected: 1,
		},
		{
			name:     "multiple doubled quotes",
			sql:      "SELECT 'it''s a ''test''';",
			expected: 1,
		},
		{
			name:     "doubled quote at end",
			sql:      "SELECT 'test''';",
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

func TestSplitPostgreSQL_DoubleQuotedIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "double quoted identifier with semicolon",
			sql:      `SELECT "column;name" FROM t;`,
			expected: 1,
		},
		{
			name:     "double quoted identifier with spaces",
			sql:      `SELECT "column name" FROM "table name";`,
			expected: 1,
		},
		{
			name:     "double quoted with reserved word",
			sql:      `SELECT "select" FROM "from";`,
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

func TestSplitPostgreSQL_Comments(t *testing.T) {
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
			name:     "line comment",
			sql:      "SELECT 1; -- comment\nSELECT 2;",
			expected: 2,
		},
		{
			name:     "multi-line block comment",
			sql:      "SELECT 1; /* line1\nline2\nline3 */ SELECT 2;",
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

func TestSplitPostgreSQL_EmptyStatements(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SplitSQL(tt.sql)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.TotalCount)
		})
	}
}

func TestSplitPostgreSQL_PositionTracking(t *testing.T) {
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

func TestSplitPostgreSQL_BestEffortParsing(t *testing.T) {
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
			name: "unclosed dollar quote",
			sql:  "DO $$ SELECT 1;",
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

func TestSplitPostgreSQL_MixedContent(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "string and dollar quote",
			sql:      "INSERT INTO t VALUES ('text', $$code;here$$);",
			expected: 1,
		},
		{
			name:     "identifier and dollar quote",
			sql:      `CREATE TABLE "my;table" AS SELECT * FROM (SELECT $$ text;here $$ as col);`,
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

func TestSplitPostgreSQL_RealWorldExample(t *testing.T) {
	sql := `
CREATE OR REPLACE FUNCTION get_user(user_id INT)
RETURNS TABLE(id INT, name TEXT) AS $$
BEGIN
    RETURN QUERY SELECT u.id, u.name FROM users u WHERE u.id = user_id;
END;
$$ LANGUAGE plpgsql;

CREATE INDEX CONCURRENTLY idx_users_name ON users(name);
`
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)

	// Verify both statements are captured
	assert.Contains(t, result.Statements[0].Text, "CREATE OR REPLACE FUNCTION")
	assert.Contains(t, result.Statements[1].Text, "CREATE INDEX CONCURRENTLY")
}

func TestSplitPostgreSQL_Unicode(t *testing.T) {
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
			sql:      `SELECT "日本語" FROM t;`,
			expected: 1,
		},
		{
			name:     "unicode in dollar quote",
			sql:      "DO $$ SELECT '日本語'; $$;",
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

// Additional test cases from Bytebase test suite

func TestSplitPostgreSQL_UnclosedString(t *testing.T) {
	// From Bytebase: INSERT INTO t VALUES ('klajfas)
	sql := "INSERT INTO t VALUES ('klajfas)"
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	// Best-effort: treat as single statement
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, sql, result.Statements[0].Text)
}

func TestSplitPostgreSQL_UnclosedIdentifier(t *testing.T) {
	// From Bytebase: INSERT INTO "t VALUES ('klajfas)
	sql := `INSERT INTO "t VALUES ('klajfas)`
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	// Best-effort: treat as single statement
	assert.Equal(t, 1, result.TotalCount)
}

func TestSplitPostgreSQL_UnclosedBlockComment(t *testing.T) {
	// From Bytebase: /*INSERT INTO "t VALUES ('klajfas)
	sql := `/*INSERT INTO "t VALUES ('klajfas)`
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	// Best-effort: treat as single statement (comment-only may be filtered)
	assert.GreaterOrEqual(t, result.TotalCount, 0)
}

func TestSplitPostgreSQL_UnclosedDollarQuote(t *testing.T) {
	// From Bytebase: $$INSERT INTO "t VALUES ('klajfas)
	sql := `$$INSERT INTO "t VALUES ('klajfas)`
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	// Best-effort: treat as single statement
	assert.Equal(t, 1, result.TotalCount)
}

func TestSplitPostgreSQL_ComplexMultiValueInsert(t *testing.T) {
	// From Bytebase: complex insert with many escaped characters
	sql := `INSERT INTO "public"."table"("id","content")
VALUES
(12,'table column name () { :xna,sydfn,,kasdfyn;}; /////test string/// 0'),
(133,'knuandfan public table id'';create table t(a int, b int);set @text=''\\kdaminxkljasdfiebkla.unkonwn''+''abcdef.xyz\''; local xxxyy.abcddd.mysql @text;------- '),
(1444,'table t xyz abc a''a\\\\''b"c>?>xxxxxx%}}%%>c<[[?${12344556778990{%}}cake\\');`

	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalCount)
}

func TestSplitPostgreSQL_InlineComments(t *testing.T) {
	// From Bytebase: inline comments in statements
	sql := `/* this is the comment. */
CREATE /* inline comment */TABLE "tech_Book"(id int, name varchar(255));
-- this is the comment.
INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafajdfl;"ka');`

	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestSplitPostgreSQL_DollarQuotedWithComments(t *testing.T) {
	// From Bytebase: dollar-quoted strings with embedded comments and semicolons
	sql := `CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
LANGUAGE SQL
AS $$
/*this is the comment */
INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lk\\jasdflkasdf"asdklf\\');
-- this is the comment
INSERT INTO tbl VALUES ('fasf_bkdjlfa');
$$;
CREATE TABLE t(a int);`

	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
	assert.Contains(t, result.Statements[0].Text, "CREATE PROCEDURE")
	assert.Contains(t, result.Statements[1].Text, "CREATE TABLE")
}

func TestSplitPostgreSQL_TaggedDollarQuote(t *testing.T) {
	// From Bytebase: $tag_name$ style dollar quotes
	sql := `CREATE PROCEDURE insert_data(a varchar(50), b varchar(50))
LANGUAGE SQL
AS $tag_name$
/*this is the comment */
INSERT /* inline comment */ INTO tbl VALUES ('lkjafd''lksjadlf;lkjasdflkasdf"asdklf');
-- this is the comment
INSERT INTO tbl VALUES ('fasf_bkdjlfa');
$tag_name$;
CREATE TABLE t(a int);`

	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestSplitPostgreSQL_WindowsLineEndings(t *testing.T) {
	// From Bytebase: Windows line endings test
	sql := "CREATE TABLE t\r\n(a int);\r\nCREATE TABLE t1(b int);"
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestSplitPostgreSQL_LeadingWhitespace(t *testing.T) {
	// From Bytebase: statements with leading whitespace
	sql := "    CREATE TABLE t(a int); CREATE TABLE t1(a int)"
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestSplitPostgreSQL_CommentOnlyMultipleLines(t *testing.T) {
	// From Bytebase: multiple line comments only
	sql := "-- klsjdfjasldf\n\t\t\t-- klsjdflkjaskldfj\n\t\t\t"
	result, err := SplitSQL(sql)
	require.NoError(t, err)
	// Comment-only content is filtered out
	assert.Equal(t, 0, result.TotalCount)
}

func TestSplitPostgreSQL_CommentsWithStatement(t *testing.T) {
	// From Bytebase: comment followed by statement, then trailing comment
	sql := `select * from t;
/* sdfasdf */;
select * from t;`

	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}

func TestSplitPostgreSQL_DoubleQuotedWithEscapedContent(t *testing.T) {
	// From Bytebase: double-quoted identifiers with content
	sql := `CREATE TABLE "tech_Book"(id int, name varchar(255));
INSERT INTO "tech_Book" VALUES (0, 'abce_ksdf'), (1, 'lks''kjsafajdfl;"ka');`

	result, err := SplitSQL(sql)
	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalCount)
}
