package models

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NullableUUID is a wrapper that can distinguish between not provided, null, and a value
type NullableUUID struct {
	UUID     *uuid.UUID
	Provided bool
}

// UnmarshalJSON implements json.Unmarshaler for NullableUUID
func (n *NullableUUID) UnmarshalJSON(data []byte) error {
	n.Provided = true

	// Check for explicit null
	if string(data) == "null" {
		n.UUID = nil
		return nil
	}

	// Try to parse as UUID
	var uuidStr string
	if err := json.Unmarshal(data, &uuidStr); err != nil {
		return err
	}

	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return err
	}

	n.UUID = &parsed
	return nil
}

// Environment represents an environment stage (Dev, Staging, Production)
type Environment struct {
	ID                uuid.UUID  `json:"id"                    db:"id"`
	Name              string     `json:"name"                  db:"name"`
	Description       *string    `json:"description,omitempty" db:"description"`
	PolicyID          *uuid.UUID `json:"policy_id,omitempty"   db:"policy_id"`
	AllowSelfApproval bool       `json:"allow_self_approval"   db:"allow_self_approval"`
	CreatedByID       uuid.UUID  `json:"created_by_id"         db:"created_by_id"`
	CreatedAt         time.Time  `json:"created_at"            db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"            db:"updated_at"`
}

// CreateEnvironmentRequest represents the request to create a new environment
type CreateEnvironmentRequest struct {
	Name              string     `json:"name"                          validate:"required,max=100"`
	Description       *string    `json:"description,omitempty"         validate:"omitempty"`
	PolicyID          *uuid.UUID `json:"policy_id,omitempty"`
	AllowSelfApproval *bool      `json:"allow_self_approval,omitempty"`
}

// UpdateEnvironmentRequest represents the request to update an environment
type UpdateEnvironmentRequest struct {
	Name              *string      `json:"name,omitempty"                validate:"omitempty,max=100"`
	Description       *string      `json:"description,omitempty"`
	PolicyID          NullableUUID `json:"policy_id,omitempty"`
	AllowSelfApproval *bool        `json:"allow_self_approval,omitempty"`
}

// EnvironmentResponse represents an environment response
type EnvironmentResponse struct {
	ID                uuid.UUID  `json:"id"`
	Name              string     `json:"name"`
	Description       *string    `json:"description,omitempty"`
	PolicyID          *uuid.UUID `json:"policy_id,omitempty"`
	AllowSelfApproval bool       `json:"allow_self_approval"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// ToResponse converts an Environment to EnvironmentResponse
func (e *Environment) ToResponse() *EnvironmentResponse {
	return &EnvironmentResponse{
		ID:                e.ID,
		Name:              e.Name,
		Description:       e.Description,
		PolicyID:          e.PolicyID,
		AllowSelfApproval: e.AllowSelfApproval,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}
}

// Validation errors
var (
	ErrEnvironmentNameRequired       = errors.New("environment name is required")
	ErrEnvironmentNameTooLong        = errors.New("environment name must not exceed 100 characters")
	ErrEnvironmentDescriptionTooLong = errors.New("description must not exceed 500 characters")
)

// Validate validates the environment model
func (e *Environment) Validate() error {
	// Trim whitespace
	e.Name = strings.TrimSpace(e.Name)
	if e.Description != nil {
		trimmed := strings.TrimSpace(*e.Description)
		e.Description = &trimmed
	}

	// Validate name
	if e.Name == "" {
		return ErrEnvironmentNameRequired
	}
	if len(e.Name) > 100 {
		return ErrEnvironmentNameTooLong
	}

	// Validate description
	if e.Description != nil && len(*e.Description) > 500 {
		return ErrEnvironmentDescriptionTooLong
	}

	return nil
}

// ValidateCreateEnvironmentRequest validates a create request
func ValidateCreateEnvironmentRequest(req *CreateEnvironmentRequest) error {
	// Trim whitespace
	req.Name = strings.TrimSpace(req.Name)
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		req.Description = &trimmed
	}

	// Validate name
	if req.Name == "" {
		return ErrEnvironmentNameRequired
	}
	if len(req.Name) > 100 {
		return ErrEnvironmentNameTooLong
	}

	// Validate description
	if req.Description != nil && len(*req.Description) > 500 {
		return ErrEnvironmentDescriptionTooLong
	}

	return nil
}

// ValidateUpdateEnvironmentRequest validates an update request
func ValidateUpdateEnvironmentRequest(req *UpdateEnvironmentRequest) error {
	// Validate name if provided
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return ErrEnvironmentNameRequired
		}
		if len(name) > 100 {
			return ErrEnvironmentNameTooLong
		}
		req.Name = &name
	}

	// Validate description if provided
	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		if len(desc) > 500 {
			return ErrEnvironmentDescriptionTooLong
		}
		req.Description = &desc
	}

	return nil
}
