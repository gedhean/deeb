// Package postgres provides PostgreSQL-specific SQL statement splitting.
package postgres

import "github.com/nsxbet/deeb/internal/parser/base"

// SplitSQL splits PostgreSQL SQL content into individual statements.
//
// PostgreSQL-specific handling:
//   - Semicolon (;) as statement delimiter
//   - Single quotes (') for string literals (with ” escape, not \')
//   - Double quotes (") for identifiers
//   - Dollar-quoted strings ($$...$$, $tag$...$tag$)
//   - Comments: /* */, --
//
// Note: BEGIN ATOMIC ... END style procedures are not supported.
// Use dollar-quoted strings instead.
func SplitSQL(sql string) (*base.ParseResult, error) {
	t := base.NewTokenizer(sql)
	statements := split(t)

	// Filter empty statements and assign indices
	result := &base.ParseResult{
		Statements: base.FilterAndIndex(statements),
	}
	result.TotalCount = len(result.Statements)

	return result, nil
}

// split performs the actual PostgreSQL splitting logic.
func split(t *base.Tokenizer) []base.SingleSQL {
	var results []base.SingleSQL

	t.SkipBlank()
	t.SetEmptyStatement(true)
	startPos := t.Pos()
	startLine := t.Line()
	startColumn := t.GetColumn()

	for {
		switch {
		case t.Char(0) == base.EOFRune:
			// Handle final statement (may not end with semicolon)
			text := t.GetString(startPos, t.Pos()-startPos)
			if !base.IsEmptyString(text) {
				endLine := t.Line() - t.AboveNonBlankLineDistance()
				results = append(results, base.SingleSQL{
					Text:        text,
					StartLine:   startLine,
					StartColumn: startColumn,
					EndLine:     endLine,
					EndColumn:   t.GetColumn(),
					ByteStart:   t.GetByteOffset(int(startPos)),
					ByteEnd:     t.GetByteOffset(int(t.Pos())),
					Empty:       t.IsEmptyStatement(),
				})
			}
			return results

		case t.Char(0) == ';':
			// Found statement delimiter
			t.Skip(1)
			text := t.GetString(startPos, t.Pos()-startPos)
			results = append(results, base.SingleSQL{
				Text:        text,
				StartLine:   startLine,
				StartColumn: startColumn,
				EndLine:     t.Line(),
				EndColumn:   t.GetColumn(),
				ByteStart:   t.GetByteOffset(int(startPos)),
				ByteEnd:     t.GetByteOffset(int(t.Pos())),
				Empty:       t.IsEmptyStatement(),
			})
			t.SkipBlank()
			startPos = t.Pos()
			startLine = t.Line()
			startColumn = t.GetColumn()
			t.SetEmptyStatement(true)

		case t.Char(0) == '/' && t.Char(1) == '*':
			// Multi-line comment
			t.ScanComment()
			t.SkipBlank()

		case t.Char(0) == '-' && t.Char(1) == '-':
			// Single line comment
			t.ScanComment()
			t.SkipBlank()

		case t.Char(0) == '\'':
			// String literal (PostgreSQL uses '' for escaping, not \')
			t.ScanPostgreSQLString()
			t.SetEmptyStatement(false)

		case t.Char(0) == '$':
			// Dollar-quoted string ($$...$$ or $tag$...$tag$)
			t.ScanDoubleDollarQuotedString()
			t.SetEmptyStatement(false)

		case t.Char(0) == '"':
			// Double-quoted identifier
			t.ScanIdentifier('"')
			t.SetEmptyStatement(false)

		case t.Char(0) == '\n':
			t.IncrementLine()
			t.Skip(1)

		default:
			t.Skip(1)
			t.SetEmptyStatement(false)
		}
	}
}
