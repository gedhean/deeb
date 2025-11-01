package models

import (
	"time"

	"github.com/google/uuid"
)

// Environment represents an environment stage (Dev, Staging, Production)
type Environment struct {
	ID          uuid.UUID  `json:"id"          db:"id"`
	Name        string     `json:"name"        db:"name"`
	Description *string    `json:"description,omitempty" db:"description"`
	PolicyID    *uuid.UUID `json:"policy_id,omitempty" db:"policy_id"`
	CreatedByID uuid.UUID  `json:"created_by_id" db:"created_by_id"`
	CreatedAt   time.Time  `json:"created_at"  db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"  db:"updated_at"`
}

// CreateEnvironmentRequest represents the request to create a new environment
type CreateEnvironmentRequest struct {
	Name        string     `json:"name"        validate:"required,max=100"`
	Description *string    `json:"description,omitempty" validate:"omitempty"`
	PolicyID    *uuid.UUID `json:"policy_id,omitempty"`
}

// UpdateEnvironmentRequest represents the request to update an environment
type UpdateEnvironmentRequest struct {
	Name        *string    `json:"name,omitempty"        validate:"omitempty,max=100"`
	Description *string    `json:"description,omitempty"`
	PolicyID    *uuid.UUID `json:"policy_id,omitempty"`
}

// EnvironmentResponse represents an environment response
type EnvironmentResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	PolicyID    *uuid.UUID `json:"policy_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ToResponse converts an Environment to EnvironmentResponse
func (e *Environment) ToResponse() *EnvironmentResponse {
	return &EnvironmentResponse{
		ID:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		PolicyID:    e.PolicyID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}
