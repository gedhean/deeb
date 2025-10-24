package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Database represents an individual database within a database instance
type Database struct {
	ID                 uuid.UUID `json:"id"                   db:"id"`
	Name               string    `json:"name"                 db:"name"`
	DatabaseInstanceID uuid.UUID `json:"database_instance_id" db:"database_instance_id"`
	ProjectID          uuid.UUID `json:"project_id"           db:"project_id"`
	EnvironmentID      uuid.UUID `json:"environment_id"       db:"environment_id"`
	CreatedAt          time.Time `json:"created_at"           db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"           db:"updated_at"`
}

// CreateDatabaseRequest represents the request to create a new database
type CreateDatabaseRequest struct {
	Name               string    `json:"name"                 validate:"required,max=100"`
	DatabaseInstanceID uuid.UUID `json:"database_instance_id" validate:"required"`
	ProjectID          uuid.UUID `json:"project_id"           validate:"required"`
	EnvironmentID      uuid.UUID `json:"environment_id"       validate:"required"`
}

// UpdateDatabaseRequest represents the request to update a database
type UpdateDatabaseRequest struct {
	Name               *string    `json:"name,omitempty"                 validate:"omitempty,max=100"`
	DatabaseInstanceID *uuid.UUID `json:"database_instance_id,omitempty"`
	ProjectID          *uuid.UUID `json:"project_id,omitempty"`
	EnvironmentID      *uuid.UUID `json:"environment_id,omitempty"`
}

// DatabaseResponse represents a database response
type DatabaseResponse struct {
	ID                 uuid.UUID `json:"id"`
	Name               string    `json:"name"`
	DatabaseInstanceID uuid.UUID `json:"database_instance_id"`
	ProjectID          uuid.UUID `json:"project_id"`
	EnvironmentID      uuid.UUID `json:"environment_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ToResponse converts a Database to DatabaseResponse
func (d *Database) ToResponse() *DatabaseResponse {
	return &DatabaseResponse{
		ID:                 d.ID,
		Name:               d.Name,
		DatabaseInstanceID: d.DatabaseInstanceID,
		ProjectID:          d.ProjectID,
		EnvironmentID:      d.EnvironmentID,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

// Validation errors
var (
	ErrDatabaseNameRequired          = errors.New("database name is required")
	ErrDatabaseNameTooLong           = errors.New("database name must not exceed 100 characters")
	ErrDatabaseInstanceIDRequired    = errors.New("database instance ID is required")
	ErrDatabaseProjectIDRequired     = errors.New("project ID is required")
	ErrDatabaseEnvironmentIDRequired = errors.New("environment ID is required")
)

// Validate validates the database model
func (d *Database) Validate() error {
	// Trim whitespace
	d.Name = strings.TrimSpace(d.Name)

	// Validate name
	if d.Name == "" {
		return ErrDatabaseNameRequired
	}
	if len(d.Name) > 100 {
		return ErrDatabaseNameTooLong
	}

	// Validate database instance ID
	if d.DatabaseInstanceID == uuid.Nil {
		return ErrDatabaseInstanceIDRequired
	}

	// Validate project ID
	if d.ProjectID == uuid.Nil {
		return ErrDatabaseProjectIDRequired
	}

	// Validate environment ID
	if d.EnvironmentID == uuid.Nil {
		return ErrDatabaseEnvironmentIDRequired
	}

	return nil
}

// ValidateCreateDatabaseRequest validates a create request
func ValidateCreateDatabaseRequest(req *CreateDatabaseRequest) error {
	// Trim whitespace
	req.Name = strings.TrimSpace(req.Name)

	// Validate name
	if req.Name == "" {
		return ErrDatabaseNameRequired
	}
	if len(req.Name) > 100 {
		return ErrDatabaseNameTooLong
	}

	// Validate database instance ID
	if req.DatabaseInstanceID == uuid.Nil {
		return ErrDatabaseInstanceIDRequired
	}

	// Validate project ID
	if req.ProjectID == uuid.Nil {
		return ErrDatabaseProjectIDRequired
	}

	// Validate environment ID
	if req.EnvironmentID == uuid.Nil {
		return ErrDatabaseEnvironmentIDRequired
	}

	return nil
}

// ValidateUpdateDatabaseRequest validates an update request
func ValidateUpdateDatabaseRequest(req *UpdateDatabaseRequest) error {
	// Validate name if provided
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return ErrDatabaseNameRequired
		}
		if len(name) > 100 {
			return ErrDatabaseNameTooLong
		}
		req.Name = &name
	}

	// Validate database instance ID if provided
	if req.DatabaseInstanceID != nil && *req.DatabaseInstanceID == uuid.Nil {
		return ErrDatabaseInstanceIDRequired
	}

	// Validate project ID if provided
	if req.ProjectID != nil && *req.ProjectID == uuid.Nil {
		return ErrDatabaseProjectIDRequired
	}

	// Validate environment ID if provided
	if req.EnvironmentID != nil && *req.EnvironmentID == uuid.Nil {
		return ErrDatabaseEnvironmentIDRequired
	}

	return nil
}
