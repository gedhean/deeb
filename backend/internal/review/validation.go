package review

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nsxbet/deeb/internal/models"
)

// ValidatePolicyConfig validates a SQL review policy configuration
func ValidatePolicyConfig(config *models.SQLReviewPolicyConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Version is optional for now (will be added in Phase 3)
	// Once versioning is implemented, uncomment this validation
	// if config.Version != models.PolicyConfigV2 {
	// 	return fmt.Errorf("unsupported policy version: %s (expected %s)", config.Version, models.PolicyConfigV2)
	// }

	// Validate rules
	if len(config.Rules) == 0 {
		return fmt.Errorf("policy must have at least one rule")
	}

	for ruleType, rule := range config.Rules {
		if err := ValidateRule(ruleType, rule); err != nil {
			return fmt.Errorf("invalid rule %s: %w", ruleType, err)
		}
	}

	return nil
}

// ValidateRule validates a single SQL review rule
func ValidateRule(ruleType string, rule models.SQLReviewRule) error {
	// Validate rule type exists
	metadata, ok := GetRuleMetadata(ruleType)
	if !ok {
		return fmt.Errorf("unknown rule type: %s", ruleType)
	}

	// Validate level
	if rule.Level != models.SQLReviewRuleLevelError && rule.Level != models.SQLReviewRuleLevelWarning {
		return fmt.Errorf("invalid level: %s (must be 'error' or 'warning')", rule.Level)
	}

	// Validate payload
	if err := ValidatePayload(metadata, rule.Payload); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	return nil
}

// ValidatePayload validates rule payload against parameter schema
func ValidatePayload(metadata *RuleMetadata, payload map[string]interface{}) error {
	if len(metadata.Parameters) == 0 && len(payload) > 0 {
		return fmt.Errorf("rule %s does not accept parameters", metadata.Type)
	}

	// Build parameter map for validation
	paramMap := make(map[string]RuleParameter)
	for _, param := range metadata.Parameters {
		paramMap[param.Name] = param
	}

	// Validate each payload value
	for key, value := range payload {
		param, ok := paramMap[key]
		if !ok {
			return fmt.Errorf("unknown parameter: %s", key)
		}

		if err := ValidateParameterValue(param, value); err != nil {
			return fmt.Errorf("invalid value for parameter %s: %w", key, err)
		}
	}

	// Check required parameters
	for _, param := range metadata.Parameters {
		if param.Required {
			if _, ok := payload[param.Name]; !ok {
				return fmt.Errorf("required parameter missing: %s", param.Name)
			}
		}
	}

	return nil
}

// ValidateParameterValue validates a parameter value against its type
func ValidateParameterValue(param RuleParameter, value interface{}) error {
	if value == nil {
		return nil // Allow nil values
	}

	switch param.Type {
	case "STRING", "TEMPLATE":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
		// For TEMPLATE type, validate format
		if param.Type == "TEMPLATE" {
			strVal := value.(string)
			if err := ValidateTemplate(strVal, param.TemplateList); err != nil {
				return err
			}
		}

	case "NUMBER":
		// Accept both int and float
		switch v := value.(type) {
		case int, int32, int64, float32, float64:
			// Valid number types
		case string:
			// Try to parse as number
			if _, err := fmt.Sscanf(v, "%f", new(float64)); err != nil {
				return fmt.Errorf("expected number, got unparseable string: %s", v)
			}
		default:
			return fmt.Errorf("expected number, got %T", value)
		}

	case "BOOLEAN":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}

	case "STRING_ARRAY":
		// Check if it's a slice
		switch v := value.(type) {
		case []interface{}:
			// Validate each element is a string
			for i, item := range v {
				if _, ok := item.(string); !ok {
					return fmt.Errorf("array element %d: expected string, got %T", i, item)
				}
			}
		case []string:
			// Already valid
		default:
			return fmt.Errorf("expected string array, got %T", value)
		}

	default:
		return fmt.Errorf("unknown parameter type: %s", param.Type)
	}

	return nil
}

// ValidateTemplate validates a template string against template tokens
func ValidateTemplate(template string, templateList []string) error {
	if len(templateList) == 0 {
		return nil // No template validation required
	}

	// Extract tokens from template
	tokenPattern := regexp.MustCompile(`\{\{(\w+)\}\}`)
	matches := tokenPattern.FindAllStringSubmatch(template, -1)

	// Check if all tokens in template are valid
	validTokens := make(map[string]bool)
	for _, token := range templateList {
		validTokens[token] = true
	}

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		token := match[1]
		if !validTokens[token] {
			return fmt.Errorf("invalid template token: {{%s}} (valid tokens: %s)",
				token, strings.Join(templateList, ", "))
		}
	}

	return nil
}

// ValidateRuleTypeForEngine checks if a rule type is valid for a specific engine
func ValidateRuleTypeForEngine(ruleType string, engine models.DatabaseEngine) error {
	metadata, ok := GetRuleMetadata(ruleType)
	if !ok {
		return fmt.Errorf("unknown rule type: %s", ruleType)
	}

	// Check engine compatibility
	if metadata.Engine == "both" {
		return nil // Rule applies to both engines
	}

	engineStr := strings.ToLower(string(engine))
	if metadata.Engine != engineStr {
		return fmt.Errorf("rule %s is not applicable to %s (only for %s)",
			ruleType, engine, metadata.Engine)
	}

	return nil
}

// SuggestSimilarRuleTypes suggests similar rule types when validation fails
func SuggestSimilarRuleTypes(input string, maxSuggestions int) []string {
	allTypes := GetAllRuleTypes()
	suggestions := []string{}

	// Simple similarity: check if input is a substring
	for _, ruleType := range allTypes {
		if strings.Contains(strings.ToLower(ruleType), strings.ToLower(input)) {
			suggestions = append(suggestions, ruleType)
			if len(suggestions) >= maxSuggestions {
				break
			}
		}
	}

	return suggestions
}
