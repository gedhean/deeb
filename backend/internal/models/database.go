package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DatabaseStatus represents the status of a database
type DatabaseStatus string

const (
	DatabaseStatusActive   DatabaseStatus = "Active"
	DatabaseStatusInactive DatabaseStatus = "Inactive"
)

// Database represents an individual database within a database instance
type Database struct {
	ID                 uuid.UUID     `json:"id"                   db:"id"`
	ProjectID          uuid.UUID     `json:"project_id"            db:"project_id"`
	DatabaseInstanceID uuid.UUID     `json:"database_instance_id"  db:"database_instance_id"`
	EnvironmentID      uuid.UUID     `json:"environment_id"        db:"environment_id"`
	DatabaseName       string         `json:"database_name"        db:"database_name"`
	Description        *string        `json:"description,omitempty" db:"description"`
	Status             DatabaseStatus `json:"status"               db:"status"`
	CreatedByID        uuid.UUID     `json:"created_by_id"        db:"created_by_id"`
	CreatedAt          time.Time     `json:"created_at"            db:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"            db:"updated_at"`
}

// CreateDatabaseRequest represents the request to create a new database
type CreateDatabaseRequest struct {
	ProjectID          uuid.UUID `json:"project_id"           validate:"required"`
	DatabaseInstanceID uuid.UUID `json:"database_instance_id" validate:"required"`
	EnvironmentID      uuid.UUID `json:"environment_id"       validate:"required"`
	DatabaseName       string    `json:"database_name"        validate:"required,max=255"`
	Description        *string   `json:"description,omitempty"`
}

// UpdateDatabaseRequest represents the request to update a database
type UpdateDatabaseRequest struct {
	DatabaseName       *string      `json:"database_name,omitempty"        validate:"omitempty,max=255"`
	Description        *string      `json:"description,omitempty"`
	Status             *DatabaseStatus `json:"status,omitempty"            validate:"omitempty,oneof=Active Inactive"`
}

// DatabaseResponse represents a database response
type DatabaseResponse struct {
	ID                 uuid.UUID     `json:"id"`
	ProjectID          uuid.UUID     `json:"project_id"`
	DatabaseInstanceID uuid.UUID     `json:"database_instance_id"`
	EnvironmentID      uuid.UUID     `json:"environment_id"`
	DatabaseName       string         `json:"database_name"`
	Description        *string       `json:"description,omitempty"`
	Status             DatabaseStatus `json:"status"`
	CreatedAt          time.Time     `json:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at"`
}

// ToResponse converts a Database to DatabaseResponse
func (d *Database) ToResponse() *DatabaseResponse {
	return &DatabaseResponse{
		ID:                 d.ID,
		ProjectID:          d.ProjectID,
		DatabaseInstanceID: d.DatabaseInstanceID,
		EnvironmentID:      d.EnvironmentID,
		DatabaseName:       d.DatabaseName,
		Description:        d.Description,
		Status:             d.Status,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          d.UpdatedAt,
	}
}

// Validation errors
var (
	ErrDatabaseNameRequired          = errors.New("database name is required")
	ErrDatabaseNameTooLong           = errors.New("database name must not exceed 255 characters")
	ErrDatabaseInstanceIDRequired    = errors.New("database instance ID is required")
	ErrDatabaseProjectIDRequired     = errors.New("project ID is required")
	ErrDatabaseEnvironmentIDRequired = errors.New("environment ID is required")
)

// Validate validates the database model
func (d *Database) Validate() error {
	// Trim whitespace
	d.DatabaseName = strings.TrimSpace(d.DatabaseName)

	// Validate database name
	if d.DatabaseName == "" {
		return ErrDatabaseNameRequired
	}
	if len(d.DatabaseName) > 255 {
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
	req.DatabaseName = strings.TrimSpace(req.DatabaseName)

	// Validate database name
	if req.DatabaseName == "" {
		return ErrDatabaseNameRequired
	}
	if len(req.DatabaseName) > 255 {
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
	// Validate database name if provided
	if req.DatabaseName != nil {
		name := strings.TrimSpace(*req.DatabaseName)
		if name == "" {
			return ErrDatabaseNameRequired
		}
		if len(name) > 255 {
			return ErrDatabaseNameTooLong
		}
		req.DatabaseName = &name
	}

	return nil
}
