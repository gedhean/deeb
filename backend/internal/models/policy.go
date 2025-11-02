package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PolicyType represents the type of policy
type PolicyType string

const (
	PolicyTypeSQLReview PolicyType = "SQL_REVIEW"
	// Future policy types can be added here
)

// PolicyConfig represents the JSONB configuration for a policy
type PolicyConfig map[string]interface{}

// MarshalJSON implements json.Marshaler for PolicyConfig
func (pc PolicyConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}(pc))
}

// UnmarshalJSON implements json.Unmarshaler for PolicyConfig
func (pc *PolicyConfig) UnmarshalJSON(data []byte) error {
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*pc = PolicyConfig(m)
	return nil
}

// Value implements driver.Valuer for PolicyConfig (for writing to database)
func (pc PolicyConfig) Value() (driver.Value, error) {
	if pc == nil || len(pc) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(pc)
}

// Scan implements sql.Scanner for PolicyConfig (for reading from database)
func (pc *PolicyConfig) Scan(value interface{}) error {
	if value == nil {
		*pc = PolicyConfig{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("failed to scan PolicyConfig: unsupported type")
	}

	if len(bytes) == 0 {
		*pc = PolicyConfig{}
		return nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(bytes, &m); err != nil {
		return err
	}
	*pc = PolicyConfig(m)
	return nil
}

// Policy represents a policy configuration (SQL_REVIEW type and future types)
type Policy struct {
	ID           uuid.UUID    `json:"id"                    db:"id"`
	Name         string       `json:"name"                  db:"name"`
	Description  *string      `json:"description,omitempty" db:"description"`
	Type         PolicyType   `json:"type"                  db:"type"`
	PolicyConfig PolicyConfig `json:"policy_config"         db:"policy_config"`
	CreatedByID  uuid.UUID    `json:"created_by_id"         db:"created_by_id"`
	CreatedAt    time.Time    `json:"created_at"            db:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"            db:"updated_at"`
}

// CreatePolicyRequest represents the request to create a new policy
type CreatePolicyRequest struct {
	Name         string       `json:"name"                  validate:"required,max=100"`
	Description  *string      `json:"description,omitempty"`
	Type         PolicyType   `json:"type"                  validate:"required,oneof=SQL_REVIEW"`
	PolicyConfig PolicyConfig `json:"policy_config"         validate:"required"`
}

// UpdatePolicyRequest represents the request to update a policy
type UpdatePolicyRequest struct {
	Name         *string       `json:"name,omitempty"          validate:"omitempty,max=100"`
	Description  *string       `json:"description,omitempty"`
	PolicyConfig *PolicyConfig `json:"policy_config,omitempty"`
}

// PolicyResponse represents a policy response
type PolicyResponse struct {
	ID           uuid.UUID    `json:"id"`
	Name         string       `json:"name"`
	Description  *string      `json:"description,omitempty"`
	Type         PolicyType   `json:"type"`
	PolicyConfig PolicyConfig `json:"policy_config"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// ToResponse converts a Policy to PolicyResponse
func (p *Policy) ToResponse() *PolicyResponse {
	return &PolicyResponse{
		ID:           p.ID,
		Name:         p.Name,
		Description:  p.Description,
		Type:         p.Type,
		PolicyConfig: p.PolicyConfig,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

// Validation errors
var (
	ErrPolicyNameRequired       = errors.New("policy name is required")
	ErrPolicyNameTooLong        = errors.New("policy name must not exceed 100 characters")
	ErrPolicyDescriptionTooLong = errors.New("description must not exceed 500 characters")
	ErrPolicyTypeRequired       = errors.New("policy type is required")
	ErrPolicyTypeInvalid        = errors.New("policy type must be SQL_REVIEW")
	ErrPolicyConfigRequired     = errors.New("policy config is required")
)

// Validate validates the policy model
func (p *Policy) Validate() error {
	// Trim whitespace
	p.Name = strings.TrimSpace(p.Name)
	if p.Description != nil {
		trimmed := strings.TrimSpace(*p.Description)
		p.Description = &trimmed
	}

	// Validate name
	if p.Name == "" {
		return ErrPolicyNameRequired
	}
	if len(p.Name) > 100 {
		return ErrPolicyNameTooLong
	}

	// Validate description
	if p.Description != nil && len(*p.Description) > 500 {
		return ErrPolicyDescriptionTooLong
	}

	// Validate type
	if p.Type == "" {
		return ErrPolicyTypeRequired
	}
	if p.Type != PolicyTypeSQLReview {
		return ErrPolicyTypeInvalid
	}

	// Validate config
	if p.PolicyConfig == nil || len(p.PolicyConfig) == 0 {
		return ErrPolicyConfigRequired
	}

	return nil
}

// ValidateCreatePolicyRequest validates a create request
func ValidateCreatePolicyRequest(req *CreatePolicyRequest) error {
	// Trim whitespace
	req.Name = strings.TrimSpace(req.Name)
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		req.Description = &trimmed
	}

	// Validate name
	if req.Name == "" {
		return ErrPolicyNameRequired
	}
	if len(req.Name) > 100 {
		return ErrPolicyNameTooLong
	}

	// Validate description
	if req.Description != nil && len(*req.Description) > 500 {
		return ErrPolicyDescriptionTooLong
	}

	// Validate type
	if req.Type == "" {
		return ErrPolicyTypeRequired
	}
	if req.Type != PolicyTypeSQLReview {
		return ErrPolicyTypeInvalid
	}

	// Validate config
	if req.PolicyConfig == nil || len(req.PolicyConfig) == 0 {
		return ErrPolicyConfigRequired
	}

	return nil
}

// ValidateUpdatePolicyRequest validates an update request
func ValidateUpdatePolicyRequest(req *UpdatePolicyRequest) error {
	// Validate name if provided
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return ErrPolicyNameRequired
		}
		if len(name) > 100 {
			return ErrPolicyNameTooLong
		}
		req.Name = &name
	}

	// Validate description if provided
	if req.Description != nil {
		desc := strings.TrimSpace(*req.Description)
		if len(desc) > 500 {
			return ErrPolicyDescriptionTooLong
		}
		req.Description = &desc
	}

	// Validate config if provided
	if req.PolicyConfig != nil && len(*req.PolicyConfig) == 0 {
		return ErrPolicyConfigRequired
	}

	return nil
}
