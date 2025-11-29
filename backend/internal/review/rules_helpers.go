package review

import (
	"strings"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/sql-reviewer/pkg/types"
)

// GetRulesByEngine returns all rules for a specific database engine
func GetRulesByEngine(dbType models.DatabaseEngine) map[string][]RuleMetadata {
	engine := strings.ToLower(string(dbType))

	// Get rules catalog
	catalog := GetAvailableRules()

	// Return rules for the specified engine
	if engineRules, ok := catalog[engine]; ok {
		return engineRules
	}

	return make(map[string][]RuleMetadata)
}

// ConvertRuleMetadataToStoreRule converts RuleMetadata to sql-reviewer's SQLReviewRule type
// When the rule's engine is "both", it will use the targetEngine parameter
func ConvertRuleMetadataToStoreRule(
	metadata RuleMetadata,
	level models.SQLReviewRuleLevel,
	payload map[string]interface{},
	targetEngine models.DatabaseEngine,
) *types.SQLReviewRule {
	// Convert level to sql-reviewer format
	var ruleLevel types.SQLReviewRuleLevel
	switch level {
	case models.SQLReviewRuleLevelError:
		ruleLevel = types.SQLReviewRuleLevel_ERROR
	case models.SQLReviewRuleLevelWarning:
		ruleLevel = types.SQLReviewRuleLevel_WARNING
	default:
		ruleLevel = types.SQLReviewRuleLevel_WARNING
	}

	// Convert engine - if rule engine is "both", use the target engine
	var engine types.Engine
	engineStr := strings.ToLower(metadata.Engine)
	if engineStr == "both" {
		// Use target engine for "both" rules
		engineStr = strings.ToLower(string(targetEngine))
	}

	switch engineStr {
	case "mysql":
		engine = types.Engine_MYSQL
	case "postgres":
		engine = types.Engine_POSTGRES
	default:
		engine = types.Engine_ENGINE_UNSPECIFIED
	}

	// Merge default parameter values from metadata with user-provided payload
	// This ensures required fields like 'format' have values even if not provided
	mergedPayload := mergePayloadWithDefaults(payload, metadata.Parameters)

	return &types.SQLReviewRule{
		Type:    metadata.Type,
		Level:   ruleLevel,
		Payload: mergedPayload,
		Engine:  engine,
	}
}

// mergePayloadWithDefaults merges user-provided payload with default values from parameters
func mergePayloadWithDefaults(payload map[string]interface{}, parameters []RuleParameter) map[string]interface{} {
	// Start with defaults from metadata
	merged := make(map[string]interface{})
	for _, param := range parameters {
		if param.Default != nil {
			merged[param.Name] = param.Default
		}
	}

	// Override with user-provided values
	for key, value := range payload {
		merged[key] = value
	}

	return merged
}

// GetAllRulesForEngine returns a flat list of all rules for an engine
func GetAllRulesForEngine(dbType models.DatabaseEngine) []RuleMetadata {
	rules := []RuleMetadata{}
	rulesByCategory := GetRulesByEngine(dbType)

	for _, categoryRules := range rulesByCategory {
		rules = append(rules, categoryRules...)
	}

	return rules
}

// FindRuleMetadata finds metadata for a rule type
func FindRuleMetadata(ruleType string) *RuleMetadata {
	metadata, ok := GetRuleMetadata(ruleType)
	if !ok {
		return nil
	}
	return metadata
}
