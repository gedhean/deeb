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
	DatabaseStatusActive   DatabaseStatus = "Active"   // Database is active and available for use. Change plans can be executed against it.
	DatabaseStatusInactive DatabaseStatus = "Inactive" // Database is inactive and not available for use. Change plans cannot be executed against it.
	DatabaseStatusPending  DatabaseStatus = "Pending"  // Database is pending and awaiting creation. Change plans can be executed against it after it is created.
)

// Database represents an individual database within a database instance
type Database struct {
	ID                 uuid.UUID      `json:"id"                    db:"id"`
	ProjectID          uuid.UUID      `json:"project_id"            db:"project_id"`
	DatabaseInstanceID uuid.UUID      `json:"database_instance_id"  db:"database_instance_id"`
	EnvironmentID      uuid.UUID      `json:"environment_id"        db:"environment_id"`
	DatabaseName       string         `json:"database_name"         db:"database_name"`
	Description        *string        `json:"description,omitempty" db:"description"`
	Status             DatabaseStatus `json:"status"                db:"status"`
	CreatedByID        uuid.UUID      `json:"created_by_id"         db:"created_by_id"`
	CreatedAt          time.Time      `json:"created_at"            db:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"            db:"updated_at"`

	// Related entities (populated with joins)
	Project          *Project          `json:"project,omitempty"           db:"-"`
	DatabaseInstance *DatabaseInstance `json:"database_instance,omitempty" db:"-"`
	Environment      *Environment      `json:"environment,omitempty"       db:"-"`
}

// CreateDatabaseRequest represents a request to create a new database
type CreateDatabaseRequest struct {
	ProjectID          uuid.UUID `json:"project_id"            validate:"required,uuid"`
	DatabaseInstanceID uuid.UUID `json:"database_instance_id"  validate:"required,uuid"`
	EnvironmentID      uuid.UUID `json:"environment_id"        validate:"required,uuid"`
	DatabaseName       string    `json:"database_name"         validate:"required,max=255"`
	Description        *string   `json:"description,omitempty"`
}

// UpdateDatabaseRequest represents a request to update a database
type UpdateDatabaseRequest struct {
	DatabaseName *string         `json:"database_name,omitempty" validate:"omitempty,max=255"`
	Description  *string         `json:"description,omitempty"`
	Status       *DatabaseStatus `json:"status,omitempty"        validate:"omitempty,oneof=Active Inactive Pending"`
}

// DatabaseResponse represents a database response
type DatabaseResponse struct {
	ID                 uuid.UUID      `json:"id"`
	ProjectID          uuid.UUID      `json:"project_id"`
	DatabaseInstanceID uuid.UUID      `json:"database_instance_id"`
	EnvironmentID      uuid.UUID      `json:"environment_id"`
	DatabaseName       string         `json:"database_name"`
	Description        *string        `json:"description,omitempty"`
	Status             DatabaseStatus `json:"status"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	// Related entities
	Project          *Project          `json:"project,omitempty"`
	DatabaseInstance *DatabaseInstance `json:"database_instance,omitempty"`
	Environment      *Environment      `json:"environment,omitempty"`
}

// Validate validates the database fields
func (d *Database) Validate() error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.ProjectID == uuid.Nil {
		return ErrDatabaseProjectIDRequired
	}
	if d.DatabaseInstanceID == uuid.Nil {
		return ErrDatabaseInstanceIDRequired
	}
	if d.EnvironmentID == uuid.Nil {
		return ErrDatabaseEnvironmentIDRequired
	}
	if strings.TrimSpace(d.DatabaseName) == "" {
		return ErrDatabaseNameRequired
	}
	if len(d.DatabaseName) > 255 {
		return ErrDatabaseNameTooLong
	}
	if d.Status == "" {
		d.Status = DatabaseStatusActive
	}
	if d.Status != DatabaseStatusActive && d.Status != DatabaseStatusInactive && d.Status != DatabaseStatusPending {
		return errors.New("status must be Active, Inactive, or Pending")
	}
	if d.CreatedByID == uuid.Nil {
		return errors.New("created_by_id is required")
	}
	return nil
}

// ValidateCreateDatabaseRequest validates a create database request
func ValidateCreateDatabaseRequest(req *CreateDatabaseRequest) error {
	if req.ProjectID == uuid.Nil {
		return ErrDatabaseProjectIDRequired
	}
	if req.DatabaseInstanceID == uuid.Nil {
		return ErrDatabaseInstanceIDRequired
	}
	if req.EnvironmentID == uuid.Nil {
		return ErrDatabaseEnvironmentIDRequired
	}
	if strings.TrimSpace(req.DatabaseName) == "" {
		return ErrDatabaseNameRequired
	}
	if len(req.DatabaseName) > 255 {
		return ErrDatabaseNameTooLong
	}
	return nil
}

// ValidateUpdateDatabaseRequest validates an update database request
func ValidateUpdateDatabaseRequest(req *UpdateDatabaseRequest) error {
	if req.DatabaseName != nil {
		if strings.TrimSpace(*req.DatabaseName) == "" {
			return ErrDatabaseNameRequired
		}
		if len(*req.DatabaseName) > 255 {
			return ErrDatabaseNameTooLong
		}
	}
	if req.Status != nil {
		if *req.Status != DatabaseStatusActive && *req.Status != DatabaseStatusInactive && *req.Status != DatabaseStatusPending {
			return errors.New("status must be Active, Inactive, or Pending")
		}
	}
	return nil
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
		Project:            d.Project,
		DatabaseInstance:   d.DatabaseInstance,
		Environment:        d.Environment,
	}
}
