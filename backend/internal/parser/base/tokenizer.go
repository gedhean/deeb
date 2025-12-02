package base

import "unicode"

const (
	// EOFRune is a sentinel rune indicating end of input.
	EOFRune = rune(-1)
)

// Tokenizer is a cursor-based state machine for scanning SQL content.
type Tokenizer struct {
	// buffer contains the SQL as runes for proper Unicode handling.
	buffer []rune
	// bufferByteOffset maps each rune index to its byte offset in the original string.
	bufferByteOffset []int
	// cursor is the current position in the buffer.
	cursor uint
	// len is the length of the buffer (excluding the sentinel EOF rune).
	len uint
	// line is the current line number (1-based).
	line int
	// emptyStatement tracks whether the current statement has any non-whitespace/comment content.
	emptyStatement bool
}

// NewTokenizer creates a new tokenizer for the given SQL string.
// It converts the string to runes for proper Unicode handling and
// appends a sentinel EOF rune.
func NewTokenizer(sql string) *Tokenizer {
	t := &Tokenizer{
		buffer: []rune(sql),
		cursor: 0,
		line:   1,
	}

	// Build byte offset mapping for each rune
	for i := range sql {
		t.bufferByteOffset = append(t.bufferByteOffset, i)
	}

	t.len = uint(len(t.buffer))

	// Append sentinel EOF rune and its byte offset
	t.buffer = append(t.buffer, EOFRune)
	t.bufferByteOffset = append(t.bufferByteOffset, len(sql))

	return t
}

// Char returns the rune at cursor + after position.
// Returns EOFRune if the position is out of bounds.
func (t *Tokenizer) Char(after uint) rune {
	if t.cursor+after >= t.len {
		return EOFRune
	}
	return t.buffer[t.cursor+after]
}

// preChar returns the rune at cursor - before position.
// Returns EOFRune if the position is out of bounds.
func (t *Tokenizer) preChar(before uint) rune {
	if t.cursor < before {
		return EOFRune
	}
	return t.buffer[t.cursor-before]
}

// Skip advances the cursor by step positions.
// If the resulting position exceeds the buffer length, cursor is set to len.
func (t *Tokenizer) Skip(step uint) {
	t.cursor += step
	if t.cursor > t.len {
		t.cursor = t.len
	}
}

// Pos returns the current cursor position.
func (t *Tokenizer) Pos() uint {
	return t.cursor
}

// GetString extracts a substring from startPos with the given length.
func (t *Tokenizer) GetString(startPos, length uint) string {
	return string(t.runeList(startPos, length))
}

// runeList returns a slice of runes from startPos with the given length.
func (t *Tokenizer) runeList(startPos, length uint) []rune {
	endPos := startPos + length
	if endPos > t.len {
		endPos = t.len
	}
	return t.buffer[startPos:endPos]
}

// GetByteOffset returns the byte offset for the given rune index.
func (t *Tokenizer) GetByteOffset(runeIndex int) int {
	if runeIndex < 0 || runeIndex >= len(t.bufferByteOffset) {
		return len(t.bufferByteOffset) - 1
	}
	return t.bufferByteOffset[runeIndex]
}

// GetColumn returns the 0-based column number for the current cursor position.
func (t *Tokenizer) GetColumn() int {
	for i := int(t.cursor) - 1; i >= 0; i-- {
		if t.buffer[i] == '\n' {
			return int(t.cursor) - i - 1
		}
	}
	return int(t.cursor)
}

// Line returns the current line number (1-based).
func (t *Tokenizer) Line() int {
	return t.line
}

// IncrementLine increments the line counter.
func (t *Tokenizer) IncrementLine() {
	t.line++
}

// IsEmptyStatement returns whether the current statement is empty.
func (t *Tokenizer) IsEmptyStatement() bool {
	return t.emptyStatement
}

// SetEmptyStatement sets the empty statement flag.
func (t *Tokenizer) SetEmptyStatement(empty bool) {
	t.emptyStatement = empty
}

// SkipBlank advances the cursor past any whitespace characters.
// Increments the line counter when newlines are encountered.
func (t *Tokenizer) SkipBlank() {
	r := t.Char(0)
	for unicode.IsSpace(r) {
		t.Skip(1)
		if r == '\n' {
			t.line++
		}
		r = t.Char(0)
	}
}

// SkipToNewLine advances the cursor to the end of the current line.
// Increments the line counter when the newline is found.
func (t *Tokenizer) SkipToNewLine() {
	r := t.Char(0)
	for r != '\n' {
		if r == EOFRune {
			return
		}
		t.Skip(1)
		r = t.Char(0)
	}
	t.line++
	t.Skip(1)
}

// AboveNonBlankLineDistance returns the number of blank lines
// between the current position and the last non-blank content.
// Used to adjust end line numbers at EOF.
func (t *Tokenizer) AboveNonBlankLineDistance() int {
	pos := uint(1)
	dis := 0
	for {
		c := t.preChar(pos)
		if c == EOFRune {
			return dis
		}
		if c == '\n' {
			dis++
		} else if !isEmptyRune(c) {
			return dis
		}
		pos++
	}
}

// ScanString scans a string literal enclosed by the given delimiter.
// Handles escape sequences (\' and \\) and tracks newlines inside strings.
// For best-effort parsing, returns on EOF instead of error.
func (t *Tokenizer) ScanString(delimiter rune) {
	if t.Char(0) != delimiter {
		return // Not a string, skip
	}

	t.Skip(1)
	for {
		switch t.Char(0) {
		case delimiter:
			t.Skip(1)
			return
		case EOFRune:
			// Best-effort: treat unclosed string as end of statement
			return
		case '\\':
			// Skip both backslash and next character (handles \' and \\)
			t.Skip(2)
		case '\n':
			t.line++
			t.Skip(1)
		default:
			t.Skip(1)
		}
	}
}

// ScanIdentifier scans a quoted identifier enclosed by the given delimiter.
// Unlike strings, identifiers don't use escape sequences.
// For best-effort parsing, returns on EOF instead of error.
func (t *Tokenizer) ScanIdentifier(delimiter rune) {
	if t.Char(0) != delimiter {
		return // Not an identifier, skip
	}

	t.Skip(1)
	for {
		switch t.Char(0) {
		case delimiter:
			t.Skip(1)
			return
		case EOFRune:
			// Best-effort: treat unclosed identifier as end of statement
			return
		case '\n':
			t.line++
			t.Skip(1)
		default:
			t.Skip(1)
		}
	}
}

// ScanComment scans a comment.
// Supports three styles:
//   - /* multi-line comments */
//   - -- single line comments
//   - # MySQL single line comments
//
// For best-effort parsing, returns on EOF instead of error.
func (t *Tokenizer) ScanComment() {
	switch {
	case t.Char(0) == '/' && t.Char(1) == '*':
		// Multi-line comment
		t.Skip(2)
		for {
			switch {
			case t.Char(0) == '*' && t.Char(1) == '/':
				t.Skip(2)
				return
			case t.Char(0) == EOFRune:
				// Best-effort: treat unclosed comment as end
				return
			case t.Char(0) == '\n':
				t.line++
				t.Skip(1)
			default:
				t.Skip(1)
			}
		}
	case t.Char(0) == '-' && t.Char(1) == '-':
		// Single line comment (standard SQL)
		t.Skip(2)
		t.SkipToNewLine()
	case t.Char(0) == '#':
		// Single line comment (MySQL style)
		t.Skip(1)
		t.SkipToNewLine()
	}
}

// ScanDoubleDollarQuotedString scans a PostgreSQL dollar-quoted string.
// Supports both $$ and $tag$ syntax.
// Uses KMP algorithm for efficient delimiter matching.
func (t *Tokenizer) ScanDoubleDollarQuotedString() {
	startPos := t.Pos()

	// Scan the opening delimiter ($$ or $tag$)
	t.scanDollarTag()

	// Extract the tag (e.g., $$ or $tag$)
	tag := t.runeList(startPos, t.Pos()-startPos)

	// Search for the closing tag using KMP
	t.ScanTo(tag)
}

// scanDollarTag scans a dollar-quote tag ($$ or $tag$).
func (t *Tokenizer) scanDollarTag() {
	if t.Char(0) != '$' {
		return
	}

	t.Skip(1)
	for {
		switch t.Char(0) {
		case '$':
			t.Skip(1)
			return
		case EOFRune:
			return
		case '\n':
			t.line++
			t.Skip(1)
		default:
			t.Skip(1)
		}
	}
}

// ScanTo scans forward until the delimiter is found using KMP algorithm.
// This provides O(n) complexity for finding multi-character delimiters.
// For best-effort parsing, returns on EOF instead of error.
func (t *Tokenizer) ScanTo(delimiter []rune) {
	if len(delimiter) == 0 {
		return
	}

	// Build KMP failure function (next array)
	next := make([]int, len(delimiter))
	next[0] = 0
	now := 0
	for i := 1; i < len(delimiter); i++ {
		for now > 0 && delimiter[i] != delimiter[now] {
			now = next[now-1]
		}
		if delimiter[i] == delimiter[now] {
			now++
		}
		next[i] = now
	}

	// Search for delimiter
	pos := 0
	for {
		// Found delimiter successfully
		if pos == len(delimiter) {
			return
		}

		if t.Char(0) == EOFRune {
			// Best-effort: treat unclosed dollar-quote as end
			return
		}

		if t.Char(0) == '\n' {
			t.line++
		}

		if t.Char(0) == delimiter[pos] {
			pos++
			t.Skip(1)
		} else {
			if pos == 0 {
				t.Skip(1)
			} else {
				pos = next[pos-1]
			}
		}
	}
}

// ScanPostgreSQLString scans a PostgreSQL string literal.
// PostgreSQL uses ” (two single quotes) to escape a single quote,
// not the backslash escape (\') used by MySQL.
func (t *Tokenizer) ScanPostgreSQLString() {
	if t.Char(0) != '\'' {
		return
	}

	t.Skip(1)
	for {
		switch t.Char(0) {
		case '\'':
			// Check for escaped quote ('')
			if t.Char(1) == '\'' {
				t.Skip(2) // Skip both quotes
				continue
			}
			// End of string
			t.Skip(1)
			return
		case EOFRune:
			// Best-effort: treat unclosed string as end
			return
		case '\n':
			t.line++
			t.Skip(1)
		default:
			t.Skip(1)
		}
	}
}

// isEmptyRune returns true if the rune is whitespace.
func isEmptyRune(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == '\r'
}

// IsEmptyString returns true if the string contains only whitespace.
func IsEmptyString(s string) bool {
	for _, c := range s {
		if !isEmptyRune(c) {
			return false
		}
	}
	return true
}
