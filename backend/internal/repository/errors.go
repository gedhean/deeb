package repository

import "errors"

// ErrNotFound is returned when a requested resource is not found in the database.
var ErrNotFound = errors.New("resource not found")
