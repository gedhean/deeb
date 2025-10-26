package models

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Project represents a logical grouping of databases and team members.
type Project struct {
	ID          uuid.UUID `json:"id"          db:"id"`
	Name        string    `json:"name"        db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at"  db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"  db:"updated_at"`
}

// CreateProjectRequest represents the request to create a new project with validation tags.
type CreateProjectRequest struct {
	Name        string `json:"name"        validate:"required,max=100"`
	Description string `json:"description" validate:"max=500"`
}

// UpdateProjectRequest represents the request to update a project with optional fields.
type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty"        validate:"omitempty,max=100"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
}

// ProjectResponse represents a project response for API output.
type ProjectResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ToResponse converts a Project to ProjectResponse.
func (p *Project) ToResponse() *ProjectResponse {
	return &ProjectResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// Validation errors
var (
	ErrProjectNameRequired = errors.New("project name is required")
	ErrProjectNameTooLong  = errors.New("project name must not exceed 100 characters")
	ErrDescriptionTooLong  = errors.New("description must not exceed 500 characters")
)

// Validate validates the project model, checking name and description constraints.
func (p *Project) Validate() error {
	// Trim whitespace
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)

	// Validate name
	if p.Name == "" {
		return ErrProjectNameRequired
	}
	if len(p.Name) > 100 {
		return ErrProjectNameTooLong
	}

	// Validate description
	if len(p.Description) > 500 {
		return ErrDescriptionTooLong
	}

	return nil
}

// ValidateCreateRequest validates a create request, trimming whitespace and checking constraints.
func ValidateCreateRequest(req *CreateProjectRequest) error {
	// Trim whitespace
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)

	// Validate name
	if req.Name == "" {
		return ErrProjectNameRequired
	}
	if len(req.Name) > 100 {
		return ErrProjectNameTooLong
	}

	// Validate description
	if len(req.Description) > 500 {
		return ErrDescriptionTooLong
	}

	return nil
}

// ValidateUpdateRequest validates an update request with optional field validation.
func ValidateUpdateRequest(req *UpdateProjectRequest) error {
	// Validate name if provided
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return ErrProjectNameRequired
		}
		if len(name) > 100 {
			return ErrProjectNameTooLong
		}
		req.Name = &name
	}

	// Validate description if provided
	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		if len(desc) > 500 {
			return ErrDescriptionTooLong
		}
		req.Description = &desc
	}

	return nil
}
