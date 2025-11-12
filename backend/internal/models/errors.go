package models

import (
	"errors"
	"fmt"
)

// ErrInvalidIssue represents a validation error for an Issue
type ErrInvalidIssue string

func (e ErrInvalidIssue) Error() string {
	return fmt.Sprintf("invalid issue: %s", string(e))
}

// ErrNotFound represents a resource not found error
type ErrNotFound struct {
	Resource string
	ID       string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

// Database validation errors
var (
	ErrDatabaseNameRequired          = errors.New("database_name is required")
	ErrDatabaseNameTooLong           = errors.New("database_name must be at most 255 characters")
	ErrDatabaseInstanceIDRequired    = errors.New("database_instance_id is required")
	ErrDatabaseProjectIDRequired     = errors.New("project_id is required")
	ErrDatabaseEnvironmentIDRequired = errors.New("environment_id is required")
)
