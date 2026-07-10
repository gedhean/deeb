package postgres

import (
	"fmt"
	"strings"
)

// conditions incrementally builds a parameterized SQL WHERE clause.
//
// It owns the positional argument index ($1, $2, ...) so callers don't have to
// track it by hand, and guarantees the same conditions can be reused for both a
// List query and its matching Count query. Arguments produced here are ordered
// starting at $1; callers that append trailing arguments (e.g. LIMIT/OFFSET)
// must number them after len(args).
type conditions struct {
	where []string
	args  []interface{}
}

// nextIndex returns the positional placeholder index for the next argument.
func (c *conditions) nextIndex() int {
	return len(c.args) + 1
}

// equal adds a "col = $n" condition unless the value is nil.
func (c *conditions) equal(col string, value interface{}) {
	if value == nil {
		return
	}
	c.where = append(c.where, fmt.Sprintf("%s = $%d", col, c.nextIndex()))
	c.args = append(c.args, value)
}

// gte adds a "col >= $n" condition unless the value is nil.
func (c *conditions) gte(col string, value interface{}) {
	if value == nil {
		return
	}
	c.where = append(c.where, fmt.Sprintf("%s >= $%d", col, c.nextIndex()))
	c.args = append(c.args, value)
}

// lte adds a "col <= $n" condition unless the value is nil.
func (c *conditions) lte(col string, value interface{}) {
	if value == nil {
		return
	}
	c.where = append(c.where, fmt.Sprintf("%s <= $%d", col, c.nextIndex()))
	c.args = append(c.args, value)
}

// in adds a "col IN ($n, $n+1, ...)" condition. A single value produces a
// one-element IN (equivalent to equality); an empty slice adds nothing.
func (c *conditions) in(col string, values []interface{}) {
	if len(values) == 0 {
		return
	}
	placeholders := make([]string, len(values))
	for i, v := range values {
		placeholders[i] = fmt.Sprintf("$%d", c.nextIndex())
		c.args = append(c.args, v)
	}
	c.where = append(c.where, fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")))
}

// inCSV adds an IN condition from a comma-separated string (e.g. "Active,Pending").
// Blank entries are dropped; an empty/blank input adds nothing. Use for
// multi-select filters over text columns.
func (c *conditions) inCSV(col, csv string) {
	values := splitCSV(csv)
	if len(values) == 0 {
		return
	}
	args := make([]interface{}, len(values))
	for i, v := range values {
		args[i] = v
	}
	c.in(col, args)
}

// splitCSV splits a comma-separated string into trimmed, non-empty values.
func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// search adds a case-insensitive substring match across one or more columns,
// e.g. "(email ILIKE $n OR display_name ILIKE $n)". A blank term or empty
// column list adds nothing.
func (c *conditions) search(term string, cols ...string) {
	term = strings.TrimSpace(term)
	if term == "" || len(cols) == 0 {
		return
	}
	idx := c.nextIndex()
	parts := make([]string, len(cols))
	for i, col := range cols {
		parts[i] = fmt.Sprintf("%s ILIKE $%d", col, idx)
	}
	c.where = append(c.where, "("+strings.Join(parts, " OR ")+")")
	c.args = append(c.args, "%"+term+"%")
}

// clause returns the joined WHERE body (without the leading "WHERE") and the
// collected arguments. When there are no conditions the body is "".
func (c *conditions) clause() (string, []interface{}) {
	return strings.Join(c.where, " AND "), c.args
}

// whereClause returns the full " WHERE ..." fragment (with a leading space) or
// an empty string when there are no conditions, ready to concatenate onto a
// base query.
func (c *conditions) whereClause() string {
	if len(c.where) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(c.where, " AND ")
}
