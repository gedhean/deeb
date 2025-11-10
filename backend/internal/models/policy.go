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

// SQLReviewRuleLevel represents the severity level of a SQL review rule
type SQLReviewRuleLevel string

const (
	SQLReviewRuleLevelError   SQLReviewRuleLevel = "error"
	SQLReviewRuleLevelWarning SQLReviewRuleLevel = "warning"
)

// SQLReviewRule represents a SQL review rule configuration
// See: github.com/nsxbet/sql-reviewer/pkg/types
type SQLReviewRule struct {
	Type    string                 `json:"type"              yaml:"type"`
	Level   SQLReviewRuleLevel     `json:"level"             yaml:"level"` // error, warning
	Payload map[string]interface{} `json:"payload,omitempty" yaml:"payload,omitempty"`
	Engine  DatabaseType           `json:"engine"            yaml:"engine"`
	Comment string                 `json:"comment,omitempty" yaml:"comment,omitempty"`
}

// SQLReviewPolicyConfig represents the policy configuration for SQL_REVIEW type
type SQLReviewPolicyConfig struct {
	Rules map[string]SQLReviewRule `json:"rules"`
}

// ParseSQLReviewConfig parses PolicyConfig into SQLReviewPolicyConfig
func ParseSQLReviewConfig(config PolicyConfig) (*SQLReviewPolicyConfig, error) {
	rulesRaw, ok := config["rules"]
	if !ok {
		return &SQLReviewPolicyConfig{Rules: make(map[string]SQLReviewRule)}, nil
	}

	rulesMap, ok := rulesRaw.(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid rules format: expected map")
	}

	rules := make(map[string]SQLReviewRule)
	for key, value := range rulesMap {
		ruleMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		rule := SQLReviewRule{}
		if typeVal, ok := ruleMap["type"].(string); ok {
			rule.Type = typeVal
		}
		if levelVal, ok := ruleMap["level"].(string); ok {
			rule.Level = SQLReviewRuleLevel(levelVal)
		}
		if payloadVal, ok := ruleMap["payload"].(map[string]interface{}); ok {
			rule.Payload = payloadVal
		}
		if engineVal, ok := ruleMap["engine"].(string); ok {
			rule.Engine = DatabaseType(engineVal)
		}
		if commentVal, ok := ruleMap["comment"].(string); ok {
			rule.Comment = commentVal
		}

		rules[key] = rule
	}

	return &SQLReviewPolicyConfig{Rules: rules}, nil
}

// ToPolicyConfig converts SQLReviewPolicyConfig to PolicyConfig
func (c *SQLReviewPolicyConfig) ToPolicyConfig() PolicyConfig {
	rulesMap := make(map[string]interface{})
	for key, rule := range c.Rules {
		ruleMap := map[string]interface{}{
			"type":   rule.Type,
			"level":  string(rule.Level),
			"engine": string(rule.Engine),
		}
		if len(rule.Payload) > 0 {
			ruleMap["payload"] = rule.Payload
		}
		if rule.Comment != "" {
			ruleMap["comment"] = rule.Comment
		}
		rulesMap[key] = ruleMap
	}
	return PolicyConfig{"rules": rulesMap}
}

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
	if len(pc) == 0 {
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
	if len(p.PolicyConfig) == 0 {
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
	if len(req.PolicyConfig) == 0 {
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
