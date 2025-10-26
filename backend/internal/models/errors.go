package models

import "fmt"

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
