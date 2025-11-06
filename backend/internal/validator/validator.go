// Package validator provides request validation for the Deeb server.
package validator

import (
	"github.com/go-playground/validator/v10"
)

// CustomValidator wraps the go-playground validator to implement Echo's validator interface
type CustomValidator struct {
	validator *validator.Validate
}

// New creates a new CustomValidator instance
func New() *CustomValidator {
	return &CustomValidator{
		validator: validator.New(),
	}
}

// Validate validates a struct and returns validation errors
func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return err
	}
	return nil
}
