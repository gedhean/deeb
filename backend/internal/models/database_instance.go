package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	DatabaseTypeMySQL      DatabaseType = "mysql"
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
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
	ID                         uuid.UUID              `json:"id"                         db:"id"`
	Name                       string                 `json:"name"                       db:"name"`
	Description                *string                `json:"description,omitempty"      db:"description"`
	DatabaseType               DatabaseType           `json:"database_type"              db:"database_type"`
	Host                       string                 `json:"host"                       db:"host"`
	Port                       int                    `json:"port"                       db:"port"`
	Username                   string                 `json:"username"                   db:"username"`
	ConnectionDetailsEncrypted *string                `json:"-"                          db:"connection_details_encrypted"`
	UseRDSIAMAuth              bool                   `json:"use_rds_iam_auth"          db:"use_rds_iam_auth"`
	RequireSSL                 bool                   `json:"require_ssl"                db:"require_ssl"`
	Status                     DatabaseInstanceStatus `json:"status"                    db:"status"`
	CreatedByID                uuid.UUID              `json:"created_by_id"              db:"created_by_id"`
	CreatedAt                  time.Time              `json:"created_at"                 db:"created_at"`
	UpdatedAt                  time.Time              `json:"updated_at"                 db:"updated_at"`
}

// CreateDatabaseInstanceRequest represents the request to create a new database instance
type CreateDatabaseInstanceRequest struct {
	Name         string       `json:"name"                       validate:"required,max=100"`
	Description  *string      `json:"description,omitempty"`
	DatabaseType DatabaseType `json:"database_type"            validate:"required,oneof=mysql postgresql"`
	Host         string       `json:"host"                       validate:"required,max=255"`
	Port         int          `json:"port"                       validate:"required,min=1,max=65535"`
	Username     string       `json:"username"                   validate:"required,max=255"`
	Password     string       `json:"password"                   validate:"required"` // Plain password (will be encrypted)
	UseRDSIAMAuth bool        `json:"use_rds_iam_auth"`
	RequireSSL   bool         `json:"require_ssl"`
}

// UpdateDatabaseInstanceRequest represents the request to update a database instance
type UpdateDatabaseInstanceRequest struct {
	Name                       *string                 `json:"name,omitempty"                       validate:"omitempty,max=100"`
	Description                *string                 `json:"description,omitempty"`
	DatabaseType               *DatabaseType           `json:"database_type,omitempty"              validate:"omitempty,oneof=mysql postgresql"`
	Host                       *string                 `json:"host,omitempty"                       validate:"omitempty,max=255"`
	Port                       *int                    `json:"port,omitempty"                       validate:"omitempty,min=1,max=65535"`
	Username                   *string                 `json:"username,omitempty"                   validate:"omitempty,max=255"`
	ConnectionDetailsEncrypted *string                 `json:"connection_details_encrypted,omitempty"`
	UseRDSIAMAuth              *bool                   `json:"use_rds_iam_auth,omitempty"`
	RequireSSL                 *bool                   `json:"require_ssl,omitempty"`
	Status                     *DatabaseInstanceStatus `json:"status,omitempty"                   validate:"omitempty,oneof=Active Inactive Maintenance"`
}

// DatabaseInstanceResponse represents a database instance response
type DatabaseInstanceResponse struct {
	ID            uuid.UUID              `json:"id"`
	Name          string                 `json:"name"`
	Description   *string                `json:"description,omitempty"`
	DatabaseType  DatabaseType           `json:"database_type"`
	Host          string                 `json:"host"`
	Port          int                    `json:"port"`
	Username      string                 `json:"username"`
	UseRDSIAMAuth bool                   `json:"use_rds_iam_auth"`
	RequireSSL    bool                   `json:"require_ssl"`
	Status        DatabaseInstanceStatus `json:"status"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// ToResponse converts a DatabaseInstance to DatabaseInstanceResponse
func (d *DatabaseInstance) ToResponse() *DatabaseInstanceResponse {
	return &DatabaseInstanceResponse{
		ID:            d.ID,
		Name:          d.Name,
		Description:   d.Description,
		DatabaseType:  d.DatabaseType,
		Host:          d.Host,
		Port:          d.Port,
		Username:      d.Username,
		UseRDSIAMAuth: d.UseRDSIAMAuth,
		RequireSSL:    d.RequireSSL,
		Status:        d.Status,
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
	ErrDatabaseTypeRequired         = errors.New("database type is required")
	ErrDatabaseTypeInvalid          = errors.New("database type must be mysql or postgresql")
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
	if d.DatabaseType == "" {
		return ErrDatabaseTypeRequired
	}
	if d.DatabaseType != DatabaseTypeMySQL && d.DatabaseType != DatabaseTypePostgreSQL {
		return ErrDatabaseTypeInvalid
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
	if req.DatabaseType == "" {
		return ErrDatabaseTypeRequired
	}
	if req.DatabaseType != DatabaseTypeMySQL && req.DatabaseType != DatabaseTypePostgreSQL {
		return ErrDatabaseTypeInvalid
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
	if req.DatabaseType != nil {
		if *req.DatabaseType != DatabaseTypeMySQL && *req.DatabaseType != DatabaseTypePostgreSQL {
			return ErrDatabaseTypeInvalid
		}
	}

	// Validate username if provided
	if req.Username != nil && *req.Username == "" {
		return ErrUsernameRequired
	}

	return nil
}
