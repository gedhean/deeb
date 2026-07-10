package handlers

import (
	"time"

	"github.com/labstack/echo/v4"
)

// parseTimeParam parses an optional RFC3339 timestamp query parameter.
// It returns (nil, nil) when the parameter is absent, and an error when the
// value is present but malformed.
func parseTimeParam(c echo.Context, name string) (*time.Time, error) {
	raw := c.QueryParam(name)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// parseBoolParam parses an optional boolean query parameter that accepts
// "true"/"false". It returns nil when the parameter is absent or unrecognized.
func parseBoolParam(c echo.Context, name string) *bool {
	switch c.QueryParam(name) {
	case "true":
		v := true
		return &v
	case "false":
		v := false
		return &v
	default:
		return nil
	}
}
