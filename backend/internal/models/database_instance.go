package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DatabaseEngine represents supported database engines
type DatabaseEngine string

const (
	DatabaseEngineMySQL      DatabaseEngine = "mysql"
	DatabaseEnginePostgreSQL DatabaseEngine = "postgresql"
)

// DatabaseInstance represents a database server or cluster
type DatabaseInstance struct {
	ID            uuid.UUID      `json:"id"             db:"id"`
	Name          string         `json:"name"           db:"name"`
	Host          string         `json:"host"           db:"host"`
	Port          int            `json:"port"           db:"port"`
	Engine        DatabaseEngine `json:"engine"         db:"engine"`
	EnvironmentID uuid.UUID      `json:"environment_id" db:"environment_id"`
	CreatedAt     time.Time      `json:"created_at"     db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"     db:"updated_at"`
}

// CreateDatabaseInstanceRequest represents the request to create a new database instance
type CreateDatabaseInstanceRequest struct {
	Name          string         `json:"name"           validate:"required,max=100"`
	Host          string         `json:"host"           validate:"required,max=255"`
	Port          int            `json:"port"           validate:"required,min=1,max=65535"`
	Engine        DatabaseEngine `json:"engine"         validate:"required,oneof=mysql postgresql"`
	EnvironmentID uuid.UUID      `json:"environment_id" validate:"required"`
}

// UpdateDatabaseInstanceRequest represents the request to update a database instance
type UpdateDatabaseInstanceRequest struct {
	Name          *string         `json:"name,omitempty"           validate:"omitempty,max=100"`
	Host          *string         `json:"host,omitempty"           validate:"omitempty,max=255"`
	Port          *int            `json:"port,omitempty"           validate:"omitempty,min=1,max=65535"`
	Engine        *DatabaseEngine `json:"engine,omitempty"         validate:"omitempty,oneof=mysql postgresql"`
	EnvironmentID *uuid.UUID      `json:"environment_id,omitempty"`
}

// DatabaseInstanceResponse represents a database instance response
type DatabaseInstanceResponse struct {
	ID            uuid.UUID      `json:"id"`
	Name          string         `json:"name"`
	Host          string         `json:"host"`
	Port          int            `json:"port"`
	Engine        DatabaseEngine `json:"engine"`
	EnvironmentID uuid.UUID      `json:"environment_id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// ToResponse converts a DatabaseInstance to DatabaseInstanceResponse
func (d *DatabaseInstance) ToResponse() *DatabaseInstanceResponse {
	return &DatabaseInstanceResponse{
		ID:            d.ID,
		Name:          d.Name,
		Host:          d.Host,
		Port:          d.Port,
		Engine:        d.Engine,
		EnvironmentID: d.EnvironmentID,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

// Validation errors
var (
	ErrDatabaseInstanceNameRequired = errors.New("database instance name is required")
	ErrDatabaseInstanceNameTooLong  = errors.New("database instance name must not exceed 100 characters")
	ErrHostRequired                 = errors.New("host is required")
	ErrHostTooLong                  = errors.New("host must not exceed 255 characters")
	ErrPortRequired                 = errors.New("port is required")
	ErrPortInvalid                  = errors.New("port must be between 1 and 65535")
	ErrEngineRequired               = errors.New("database engine is required")
	ErrEngineInvalid                = errors.New("database engine must be mysql or postgresql")
	ErrEnvironmentIDRequired        = errors.New("environment ID is required")
)

// Validate validates the database instance model
func (d *DatabaseInstance) Validate() error {
	// Trim whitespace
	d.Name = strings.TrimSpace(d.Name)
	d.Host = strings.TrimSpace(d.Host)

	// Validate name
	if d.Name == "" {
		return ErrDatabaseInstanceNameRequired
	}
	if len(d.Name) > 100 {
		return ErrDatabaseInstanceNameTooLong
	}

	// Validate host
	if d.Host == "" {
		return ErrHostRequired
	}
	if len(d.Host) > 255 {
		return ErrHostTooLong
	}

	// Validate port
	if d.Port < 1 || d.Port > 65535 {
		return ErrPortInvalid
	}

	// Validate engine
	if d.Engine == "" {
		return ErrEngineRequired
	}
	if d.Engine != DatabaseEngineMySQL && d.Engine != DatabaseEnginePostgreSQL {
		return ErrEngineInvalid
	}

	// Validate environment ID
	if d.EnvironmentID == uuid.Nil {
		return ErrEnvironmentIDRequired
	}

	return nil
}

// ValidateCreateRequest validates a create request
func ValidateCreateDatabaseInstanceRequest(req *CreateDatabaseInstanceRequest) error {
	// Trim whitespace
	req.Name = strings.TrimSpace(req.Name)
	req.Host = strings.TrimSpace(req.Host)

	// Validate name
	if req.Name == "" {
		return ErrDatabaseInstanceNameRequired
	}
	if len(req.Name) > 100 {
		return ErrDatabaseInstanceNameTooLong
	}

	// Validate host
	if req.Host == "" {
		return ErrHostRequired
	}
	if len(req.Host) > 255 {
		return ErrHostTooLong
	}

	// Validate port
	if req.Port < 1 || req.Port > 65535 {
		return ErrPortInvalid
	}

	// Validate engine
	if req.Engine == "" {
		return ErrEngineRequired
	}
	if req.Engine != DatabaseEngineMySQL && req.Engine != DatabaseEnginePostgreSQL {
		return ErrEngineInvalid
	}

	// Validate environment ID
	if req.EnvironmentID == uuid.Nil {
		return ErrEnvironmentIDRequired
	}

	return nil
}

// ValidateUpdateRequest validates an update request
func ValidateUpdateDatabaseInstanceRequest(req *UpdateDatabaseInstanceRequest) error {
	// Validate name if provided
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return ErrDatabaseInstanceNameRequired
		}
		if len(name) > 100 {
			return ErrDatabaseInstanceNameTooLong
		}
		req.Name = &name
	}

	// Validate host if provided
	if req.Host != nil {
		host := strings.TrimSpace(*req.Host)
		if host == "" {
			return ErrHostRequired
		}
		if len(host) > 255 {
			return ErrHostTooLong
		}
		req.Host = &host
	}

	// Validate port if provided
	if req.Port != nil {
		if *req.Port < 1 || *req.Port > 65535 {
			return ErrPortInvalid
		}
	}

	// Validate engine if provided
	if req.Engine != nil {
		if *req.Engine != DatabaseEngineMySQL && *req.Engine != DatabaseEnginePostgreSQL {
			return ErrEngineInvalid
		}
	}

	// Validate environment ID if provided
	if req.EnvironmentID != nil && *req.EnvironmentID == uuid.Nil {
		return ErrEnvironmentIDRequired
	}

	return nil
}
