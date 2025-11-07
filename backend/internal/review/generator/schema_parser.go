package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// SchemaRule represents a rule definition from schema.yaml
type SchemaRule struct {
	Type          string            `yaml:"type"`
	Category      string            `yaml:"category"`
	Engine        string            `yaml:"engine"`
	ComponentList []SchemaComponent `yaml:"componentList,omitempty"`
}

// SchemaComponent represents a parameter in a rule
type SchemaComponent struct {
	Key     string        `yaml:"key"`
	Payload PayloadConfig `yaml:"payload"`
}

// PayloadConfig represents the payload configuration for a parameter
type PayloadConfig struct {
	Type         string      `yaml:"type"`
	Default      interface{} `yaml:"default,omitempty"`
	TemplateList []string    `yaml:"templateList,omitempty"`
}

// ParsedRule represents a processed rule with metadata
type ParsedRule struct {
	Type        string
	Name        string
	Description string
	Category    string
	Engine      string // "mysql", "postgresql", or "both"
	Parameters  []RuleParameter
}

// RuleParameter represents a rule parameter
type RuleParameter struct {
	Name         string
	Type         string // STRING, NUMBER, BOOLEAN, STRING_ARRAY, TEMPLATE
	Description  string
	Required     bool
	Default      interface{}
	TemplateList []string
}

// ParseSchema reads and parses the schema.yaml file
func ParseSchema(path string) ([]ParsedRule, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Parse YAML
	var schemaRules []SchemaRule
	if err := yaml.Unmarshal(data, &schemaRules); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Group rules by type to identify which engines they support
	rulesByType := make(map[string][]SchemaRule)
	for _, rule := range schemaRules {
		rulesByType[rule.Type] = append(rulesByType[rule.Type], rule)
	}

	// Process unique rules
	uniqueRules := make(map[string]*ParsedRule)
	for ruleType, rules := range rulesByType {
		engines := []string{}
		var components []SchemaComponent

		for _, rule := range rules {
			engines = append(engines, strings.ToLower(rule.Engine))
			if len(rule.ComponentList) > 0 {
				components = rule.ComponentList
			}
		}

		// Determine engine support
		engine := "both"
		if len(engines) == 1 {
			engine = engines[0]
		}

		// Use first rule for category
		category := rules[0].Category

		// Convert to parsed rule
		parsed := &ParsedRule{
			Type:        ruleType,
			Name:        generateRuleName(ruleType),
			Description: generateRuleDescription(ruleType),
			Category:    normalizeCategory(category),
			Engine:      engine,
			Parameters:  convertComponents(components),
		}

		uniqueRules[ruleType] = parsed
	}

	// Convert to slice
	result := make([]ParsedRule, 0, len(uniqueRules))
	for _, rule := range uniqueRules {
		result = append(result, *rule)
	}

	return result, nil
}

// convertComponents converts schema components to rule parameters
func convertComponents(components []SchemaComponent) []RuleParameter {
	params := make([]RuleParameter, 0, len(components))

	for _, comp := range components {
		param := RuleParameter{
			Name:         comp.Key,
			Type:         comp.Payload.Type,
			Description:  generateParameterDescription(comp.Key, comp.Payload.Type),
			Required:     false,
			Default:      comp.Payload.Default,
			TemplateList: comp.Payload.TemplateList,
		}
		params = append(params, param)
	}

	return params
}

// generateRuleName generates a human-readable name from rule type
func generateRuleName(ruleType string) string {
	// Remove prefix and convert to title case
	parts := strings.Split(ruleType, ".")
	name := parts[len(parts)-1]

	// Convert kebab-case to Title Case
	words := strings.Split(name, "-")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

// generateRuleDescription generates a description from rule type
func generateRuleDescription(ruleType string) string {
	descriptions := map[string]string{
		"engine.mysql.use-innodb":                             "Require InnoDB storage engine for all tables",
		"table.require-pk":                                    "Require primary key on all tables",
		"table.no-foreign-key":                                "Disallow foreign key constraints",
		"table.drop-naming-convention":                        "Enforce DROP table naming pattern",
		"table.comment":                                       "Require or validate table comments",
		"table.disallow-partition":                            "Disallow table partitions",
		"table.disallow-trigger":                              "Disallow triggers on tables",
		"table.no-duplicate-index":                            "Prevent duplicate indexes",
		"table.text-fields-total-length":                      "Limit total TEXT field length in table",
		"table.disallow-set-charset":                          "Disallow charset changes on tables",
		"table.disallow-ddl":                                  "Restrict DDL operations on specific tables",
		"table.disallow-dml":                                  "Restrict DML operations on specific tables",
		"table.limit-size":                                    "Restrict operations on large tables",
		"table.require-charset":                               "Enforce specific charset for tables",
		"table.require-collation":                             "Enforce specific collation for tables",
		"statement.select.no-select-all":                      "Disallow SELECT * in queries",
		"statement.where.require.select":                      "Require WHERE clause in SELECT statements",
		"statement.where.require.update-delete":               "Require WHERE clause in UPDATE/DELETE statements",
		"statement.where.no-leading-wildcard-like":            "Disallow leading wildcard in LIKE patterns",
		"statement.disallow-on-del-cascade":                   "Disallow ON DELETE CASCADE",
		"statement.disallow-rm-tbl-cascade":                   "Disallow DROP TABLE CASCADE",
		"statement.disallow-commit":                           "Disallow explicit COMMIT statements",
		"statement.disallow-limit":                            "Disallow LIMIT in UPDATE/DELETE statements",
		"statement.disallow-order-by":                         "Disallow ORDER BY in UPDATE/DELETE statements",
		"statement.merge-alter-table":                         "Suggest merging multiple ALTER TABLE statements",
		"statement.insert.row-limit":                          "Limit number of rows in INSERT statements",
		"statement.insert.must-specify-column":                "Require column specification in INSERT statements",
		"statement.insert.disallow-order-by-rand":             "Disallow ORDER BY RAND() in queries",
		"statement.affected-row-limit":                        "Limit number of affected rows in DML statements",
		"statement.dml-dry-run":                               "Execute DML dry-run validation",
		"statement.disallow-add-column-with-default":          "Prevent ADD COLUMN WITH DEFAULT in PostgreSQL",
		"statement.add-check-not-valid":                       "Require NOT VALID for CHECK constraints",
		"statement.disallow-add-not-null":                     "Prevent adding NOT NULL constraint directly",
		"statement.select-full-table-scan":                    "Detect full table scans in SELECT queries",
		"statement.create-specify-schema":                     "Require schema specification in CREATE statements",
		"statement.check-set-role-variable":                   "Check SET ROLE variable usage",
		"statement.disallow-using-temporary":                  "Disallow queries using temporary tables",
		"statement.disallow-using-filesort":                   "Disallow queries using filesort",
		"statement.where.no-equal-null":                       "Disallow = NULL (use IS NULL instead)",
		"statement.where.disallow-functions-and-calculations": "Disallow functions and calculations in WHERE clause",
		"statement.query.minimum-plan-level":                  "Require minimum execution plan level",
		"statement.where.maximum-logical-operator-count":      "Limit number of AND/OR operators in WHERE clause",
		"statement.maximum-limit-value":                       "Limit maximum value in LIMIT clause",
		"statement.maximum-join-table-count":                  "Limit number of tables in JOIN",
		"statement.maximum-statements-in-transaction":         "Limit number of statements per transaction",
		"statement.join-strict-column-attrs":                  "Enforce matching column attributes in JOINs",
		"statement.disallow-mix-in-ddl":                       "Disallow mixing DML in DDL statements",
		"statement.disallow-mix-in-dml":                       "Disallow mixing DDL in DML statements",
		"statement.max-execution-time":                        "Enforce maximum execution time for queries",
		"statement.require-algorithm-option":                  "Require ALGORITHM option in ALTER TABLE",
		"statement.require-lock-option":                       "Require LOCK option in ALTER TABLE",
		"naming.fully-qualified":                              "Require fully qualified object names",
		"naming.table":                                        "Enforce table naming convention",
		"naming.column":                                       "Enforce column naming convention",
		"naming.index.uk":                                     "Enforce unique key naming convention",
		"naming.index.idx":                                    "Enforce index naming convention",
		"naming.index.fk":                                     "Enforce foreign key naming convention",
		"naming.index.pk":                                     "Enforce primary key naming convention",
		"naming.column.auto-increment":                        "Enforce auto-increment column naming convention",
		"naming.identifier.no-keyword":                        "Disallow SQL keywords in identifiers",
		"column.required":                                     "Require specific columns in tables",
		"column.no-null":                                      "Disallow NULL values in columns",
		"column.disallow-change-type":                         "Prevent column type changes",
		"column.set-default-for-not-null":                     "Require default value for NOT NULL columns",
		"column.disallow-change":                              "Disallow CHANGE COLUMN operations",
		"column.disallow-changing-order":                      "Prevent column reordering",
		"column.disallow-drop-in-index":                       "Prevent dropping columns that are in indexes",
		"column.comment":                                      "Require or validate column comments",
		"column.auto-increment-must-integer":                  "Require auto-increment columns to be integer type",
		"column.type-disallow-list":                           "Disallow specific column types",
		"column.disallow-set-charset":                         "Disallow charset changes on columns",
		"column.maximum-character-length":                     "Limit maximum character length for columns",
		"column.maximum-varchar-length":                       "Limit maximum VARCHAR length",
		"column.auto-increment-initial-value":                 "Set initial value for auto-increment columns",
		"column.auto-increment-must-unsigned":                 "Require auto-increment columns to be UNSIGNED",
		"column.current-time-count-limit":                     "Limit number of CURRENT_TIMESTAMP columns",
		"column.require-default":                              "Require default values for columns",
		"column.default-disallow-volatile":                    "Disallow volatile default values",
		"column.require-charset":                              "Enforce specific charset for columns",
		"column.require-collation":                            "Enforce specific collation for columns",
		"schema.backward-compatibility":                       "Ensure schema changes are backward compatible",
		"database.drop-empty-database":                        "Only allow dropping empty databases",
		"index.no-duplicate-column":                           "Prevent duplicate columns in indexes",
		"index.key-number-limit":                              "Limit number of keys in index",
		"index.pk-type-limit":                                 "Restrict primary key column types",
		"index.type-no-blob":                                  "Disallow BLOB columns in indexes",
		"index.total-number-limit":                            "Limit total number of indexes per table",
		"index.primary-key-type-allowlist":                    "Allow only specific types for primary keys",
		"index.create-concurrently":                           "Require CREATE INDEX CONCURRENTLY",
		"index.type-allow-list":                               "Allow only specific index types",
		"system.charset.allowlist":                            "Allow only specific character sets",
		"system.collation.allowlist":                          "Allow only specific collations",
		"system.comment.length":                               "Limit comment length",
		"system.procedure.disallow-create":                    "Disallow creating stored procedures",
		"system.event.disallow-create":                        "Disallow creating events",
		"system.view.disallow-create":                         "Disallow creating views",
		"system.function.disallow-create":                     "Disallow creating functions",
		"system.function.disallowed-list":                     "Disallow specific functions",
		"statement.add-foreign-key-not-valid":                 "Require NOT VALID for foreign key constraints",
		"statement.non-transactional":                         "Detect non-transactional operations",
		"statement.object-owner-check":                        "Verify object ownership before operations",
	}

	if desc, ok := descriptions[ruleType]; ok {
		return desc
	}

	// Fallback: generate from rule name
	return fmt.Sprintf("Rule: %s", generateRuleName(ruleType))
}

// generateParameterDescription generates a description for a parameter
func generateParameterDescription(name, paramType string) string {
	descriptions := map[string]string{
		"format":                 "Regular expression pattern for naming validation",
		"maxLength":              "Maximum length for identifiers",
		"number":                 "Numeric limit or threshold",
		"list":                   "List of allowed or disallowed values",
		"required":               "Whether the field is required",
		"requiredClassification": "Whether classification is required in comments",
		"level":                  "Minimum execution plan level",
	}

	if desc, ok := descriptions[name]; ok {
		return desc
	}

	// Fallback
	return fmt.Sprintf("Parameter: %s (type: %s)", name, paramType)
}

// normalizeCategory converts YAML category to lowercase
func normalizeCategory(category string) string {
	return strings.ToLower(category)
}
