package review

import (
	"strings"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/sql-reviewer/pkg/types"
)

// GetRulesByEngine returns all rules for a specific database engine
func GetRulesByEngine(dbType models.DatabaseType) map[string][]RuleMetadata {
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
func ConvertRuleMetadataToStoreRule(
	metadata RuleMetadata,
	level models.SQLReviewRuleLevel,
	payload map[string]interface{},
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

	// Convert engine
	var engine types.Engine
	switch strings.ToLower(metadata.Engine) {
	case "mysql":
		engine = types.Engine_MYSQL
	case "postgresql", "postgres":
		engine = types.Engine_POSTGRES
	default:
		engine = types.Engine_ENGINE_UNSPECIFIED
	}

	return &types.SQLReviewRule{
		Type:    metadata.Type,
		Level:   ruleLevel,
		Payload: payload, // sql-reviewer uses map[string]interface{} directly
		Engine:  engine,
	}
}

// GetAllRulesForEngine returns a flat list of all rules for an engine
func GetAllRulesForEngine(dbType models.DatabaseType) []RuleMetadata {
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
