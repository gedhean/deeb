package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DatabaseEngine represents the type of database
type DatabaseEngine string

const (
	DatabaseEngineMySQL      DatabaseEngine = "mysql"
	DatabaseEnginePostgreSQL DatabaseEngine = "postgres"
)

// DatabaseInstanceStatus represents the status of a database instance
type DatabaseInstanceStatus string

const (
	DatabaseInstanceStatusActive      DatabaseInstanceStatus = "Active"
	DatabaseInstanceStatusInactive    DatabaseInstanceStatus = "Inactive"
	DatabaseInstanceStatusMaintenance DatabaseInstanceStatus = "Maintenance"
)

// DatabaseInstance represents a database server or cluster
type DatabaseInstance struct {
	ID                         uuid.UUID              `json:"id"                    db:"id"`
	Name                       string                 `json:"name"                  db:"name"`
	Description                *string                `json:"description,omitempty" db:"description"`
	DatabaseEngine             DatabaseEngine         `json:"database_engine"       db:"database_engine"`
	Host                       string                 `json:"host"                  db:"host"`
	Port                       int                    `json:"port"                  db:"port"`
	Username                   string                 `json:"username"              db:"username"`
	ConnectionDetailsEncrypted *string                `json:"-"                     db:"connection_details_encrypted"`
	UseRDSIAMAuth              bool                   `json:"use_rds_iam_auth"      db:"use_rds_iam_auth"`
	RequireSSL                 bool                   `json:"require_ssl"           db:"require_ssl"`
	Status                     DatabaseInstanceStatus `json:"status"                db:"status"`
	EnvironmentID              *uuid.UUID             `json:"environment_id"        db:"environment_id"`
	CreatedByID                uuid.UUID              `json:"created_by_id"         db:"created_by_id"`
	CreatedAt                  time.Time              `json:"created_at"            db:"created_at"`
	UpdatedAt                  time.Time              `json:"updated_at"            db:"updated_at"`
}

// CreateDatabaseInstanceRequest represents the request to create a new database instance
type CreateDatabaseInstanceRequest struct {
	Name           string         `json:"name"                  validate:"required,max=100"`
	Description    *string        `json:"description,omitempty"`
	DatabaseEngine DatabaseEngine `json:"database_engine"       validate:"required,oneof=mysql postgres"`
	Host           string         `json:"host"                  validate:"required,max=255"`
	Port           int            `json:"port"                  validate:"required,min=1,max=65535"`
	Username       string         `json:"username"              validate:"required,max=255"`
	Password       string         `json:"password"              validate:"required"` // Plain password (will be encrypted)
	UseRDSIAMAuth  bool           `json:"use_rds_iam_auth"`
	RequireSSL     bool           `json:"require_ssl"`
	EnvironmentID  *uuid.UUID     `json:"environment_id"`
}

// UpdateDatabaseInstanceRequest represents the request to update a database instance
type UpdateDatabaseInstanceRequest struct {
	Name           *string                 `json:"name,omitempty"             validate:"omitempty,max=100"`
	Description    *string                 `json:"description,omitempty"`
	DatabaseEngine *DatabaseEngine         `json:"database_engine,omitempty"  validate:"omitempty,oneof=mysql postgres"`
	Host           *string                 `json:"host,omitempty"             validate:"omitempty,max=255"`
	Port           *int                    `json:"port,omitempty"             validate:"omitempty,min=1,max=65535"`
	Username       *string                 `json:"username,omitempty"         validate:"omitempty,max=255"`
	Password       *string                 `json:"password,omitempty"` // Plain password (will be encrypted)
	UseRDSIAMAuth  *bool                   `json:"use_rds_iam_auth,omitempty"`
	RequireSSL     *bool                   `json:"require_ssl,omitempty"`
	Status         *DatabaseInstanceStatus `json:"status,omitempty"           validate:"omitempty,oneof=Active Inactive Maintenance"`
	EnvironmentID  *uuid.UUID              `json:"environment_id"`
}

// DatabaseInstanceResponse represents a database instance response
type DatabaseInstanceResponse struct {
	ID             uuid.UUID              `json:"id"`
	Name           string                 `json:"name"`
	Description    *string                `json:"description,omitempty"`
	DatabaseEngine DatabaseEngine         `json:"database_engine"`
	Host           string                 `json:"host"`
	Port           int                    `json:"port"`
	Username       string                 `json:"username"`
	UseRDSIAMAuth  bool                   `json:"use_rds_iam_auth"`
	RequireSSL     bool                   `json:"require_ssl"`
	Status         DatabaseInstanceStatus `json:"status"`
	EnvironmentID  *uuid.UUID             `json:"environment_id"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// ToResponse converts a DatabaseInstance to DatabaseInstanceResponse
func (d *DatabaseInstance) ToResponse() *DatabaseInstanceResponse {
	return &DatabaseInstanceResponse{
		ID:             d.ID,
		Name:           d.Name,
		Description:    d.Description,
		DatabaseEngine: d.DatabaseEngine,
		Host:           d.Host,
		Port:           d.Port,
		Username:       d.Username,
		UseRDSIAMAuth:  d.UseRDSIAMAuth,
		RequireSSL:     d.RequireSSL,
		Status:         d.Status,
		EnvironmentID:  d.EnvironmentID,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
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
	ErrDatabaseEngineRequired       = errors.New("database type is required")
	ErrDatabaseEngineInvalid        = errors.New("database type must be mysql or postgres")
	ErrUsernameRequired             = errors.New("username is required")
	ErrPasswordRequired             = errors.New("password is required")
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

	// Validate database type
	if d.DatabaseEngine == "" {
		return ErrDatabaseEngineRequired
	}
	if d.DatabaseEngine != DatabaseEngineMySQL && d.DatabaseEngine != DatabaseEnginePostgreSQL {
		return ErrDatabaseEngineInvalid
	}

	// Validate username
	if d.Username == "" {
		return ErrUsernameRequired
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

	// Validate database type
	if req.DatabaseEngine == "" {
		return ErrDatabaseEngineRequired
	}
	if req.DatabaseEngine != DatabaseEngineMySQL && req.DatabaseEngine != DatabaseEnginePostgreSQL {
		return ErrDatabaseEngineInvalid
	}

	// Validate username
	if req.Username == "" {
		return ErrUsernameRequired
	}

	// Validate password
	if req.Password == "" {
		return ErrPasswordRequired
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

	// Validate database type if provided
	if req.DatabaseEngine != nil {
		if *req.DatabaseEngine != DatabaseEngineMySQL && *req.DatabaseEngine != DatabaseEnginePostgreSQL {
			return ErrDatabaseEngineInvalid
		}
	}

	// Validate username if provided
	if req.Username != nil && *req.Username == "" {
		return ErrUsernameRequired
	}

	return nil
}
