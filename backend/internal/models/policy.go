package models

import (
	"encoding/json"
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

// Policy represents a policy configuration (SQL_REVIEW type and future types)
type Policy struct {
	ID           uuid.UUID    `json:"id"          db:"id"`
	Name         string       `json:"name"        db:"name"`
	Description  *string      `json:"description,omitempty" db:"description"`
	Type         PolicyType   `json:"type"       db:"type"`
	PolicyConfig PolicyConfig `json:"policy_config" db:"policy_config"`
	CreatedByID  uuid.UUID    `json:"created_by_id" db:"created_by_id"`
	CreatedAt    time.Time    `json:"created_at"  db:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"  db:"updated_at"`
}

// CreatePolicyRequest represents the request to create a new policy
type CreatePolicyRequest struct {
	Name         string       `json:"name"        validate:"required,max=100"`
	Description  *string      `json:"description,omitempty"`
	Type         PolicyType   `json:"type"       validate:"required,oneof=SQL_REVIEW"`
	PolicyConfig PolicyConfig `json:"policy_config" validate:"required"`
}

// UpdatePolicyRequest represents the request to update a policy
type UpdatePolicyRequest struct {
	Name         *string       `json:"name,omitempty"        validate:"omitempty,max=100"`
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
