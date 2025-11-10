package review

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	reviewerConfig "github.com/nsxbet/sql-reviewer/pkg/config"
	"github.com/nsxbet/sql-reviewer/pkg/reviewer"
	"github.com/nsxbet/sql-reviewer/pkg/types"
	"go.uber.org/zap"
)

// ReviewService provides enhanced SQL review capabilities
type ReviewService struct {
	policyRepo  repository.PolicyRepository
	schemaCache *SchemaCache
	logger      *zap.Logger

	// Configuration
	defaultTimeout time.Duration
}

// NewReviewService creates a new enhanced review service
func NewReviewService(
	policyRepo repository.PolicyRepository,
	schemaCache *SchemaCache,
	logger *zap.Logger,
) *ReviewService {
	return &ReviewService{
		policyRepo:     policyRepo,
		schemaCache:    schemaCache,
		logger:         logger,
		defaultTimeout: 30 * time.Second,
	}
}

// ReviewOptions contains options for running SQL review
type ReviewOptions struct {
	// ChangePlanID is the ID of the change plan being reviewed
	ChangePlanID uuid.UUID

	// EnvironmentID for loading the associated policy
	EnvironmentID uuid.UUID

	// DatabaseID for schema-aware validation
	DatabaseID *uuid.UUID

	// SQLContent is the SQL to review
	SQLContent string

	// DatabaseType is the database engine
	DatabaseType models.DatabaseType

	// Schema is optional pre-loaded schema metadata
	Schema *types.DatabaseSchemaMetadata

	// Timeout for the review operation
	Timeout time.Duration
}

// Review performs SQL review using the high-level reviewer API
func (s *ReviewService) Review(ctx context.Context, opts ReviewOptions) (*ReviewResult, error) {
	start := time.Now()

	// Set timeout
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = s.defaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Load policy for environment
	policy, err := s.loadPolicyForEnvironment(ctx, opts.EnvironmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}

	// If no policy, return clean result
	if policy == nil {
		return s.reviewWithNoPolicy(start), nil
	}

	// Parse policy configuration
	config, err := models.ParseSQLReviewConfig(policy.PolicyConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse policy config: %w", err)
	}

	// Convert database type to sql-reviewer engine
	engine := convertDatabaseTypeToEngine(opts.DatabaseType)

	// Create reviewer instance
	rev := reviewer.New(engine)

	// Build rule configuration for sql-reviewer
	rules, enabledRuleTypes := s.buildRulesForReviewer(config, opts.DatabaseType)

	// Configure reviewer with rules
	if len(rules) > 0 {
		// Create Config for the reviewer
		reviewConfig := &reviewerConfig.Config{
			ID:    "policy-" + policy.ID.String(),
			Rules: rules,
		}

		rev.WithConfigObject(reviewConfig)
	}

	// Get schema for schema-aware validation if available
	var schema *types.DatabaseSchemaMetadata
	if opts.Schema != nil {
		schema = opts.Schema
	} else if opts.DatabaseID != nil && s.schemaCache != nil {
		schema = s.schemaCache.Get(ctx, *opts.DatabaseID)
	}

	// Execute review
	var result *reviewer.ReviewResult
	if schema != nil {
		s.logger.Info("Running schema-aware SQL review",
			zap.String("database_id", opts.DatabaseID.String()),
			zap.Int("rules", len(rules)))
		result, err = rev.ReviewWithSchema(ctx, opts.SQLContent, schema)
	} else {
		s.logger.Info("Running SQL review without schema",
			zap.Int("rules", len(rules)))
		result, err = rev.Review(ctx, opts.SQLContent)
	}

	if err != nil {
		return nil, fmt.Errorf("review failed: %w", err)
	}

	// Convert result to our format
	reviewResult := s.convertReviewerResult(result, enabledRuleTypes, policy, opts.DatabaseType, start)

	return reviewResult, nil
}

// loadPolicyForEnvironment loads the SQL_REVIEW policy for an environment
func (s *ReviewService) loadPolicyForEnvironment(ctx context.Context, environmentID uuid.UUID) (*models.Policy, error) {
	// For now, load the first SQL_REVIEW policy
	// TODO: In Phase 3, load policy associated with environment
	policies, err := s.policyRepo.GetByType(ctx, models.PolicyTypeSQLReview, 1, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}

	if len(policies) == 0 {
		s.logger.Warn("No SQL_REVIEW policy found")
		return nil, nil
	}

	return policies[0], nil
}

// buildRulesForReviewer builds sql-reviewer rules from policy config
func (s *ReviewService) buildRulesForReviewer(
	config *models.SQLReviewPolicyConfig,
	dbType models.DatabaseType,
) ([]*types.SQLReviewRule, []string) {
	rules := make([]*types.SQLReviewRule, 0)
	enabledTypes := make([]string, 0)

	for ruleKey, ruleConfig := range config.Rules {
		// Only process rules for the current database type
		// Rules with "both" engine or matching engine
		if ruleConfig.Engine != dbType && ruleConfig.Engine != "both" {
			continue
		}

		// Find rule metadata
		metadata := FindRuleMetadata(ruleConfig.Type)
		if metadata == nil {
			s.logger.Warn("Rule metadata not found",
				zap.String("rule_key", ruleKey),
				zap.String("rule_type", ruleConfig.Type))
			continue
		}

		// Convert to sql-reviewer rule
		storeRule := ConvertRuleMetadataToStoreRule(*metadata, ruleConfig.Level, ruleConfig.Payload)
		rules = append(rules, storeRule)
		enabledTypes = append(enabledTypes, ruleConfig.Type)
	}

	return rules, enabledTypes
}

// convertReviewerResult converts sql-reviewer result to our format
func (s *ReviewService) convertReviewerResult(
	result *reviewer.ReviewResult,
	enabledRuleTypes []string,
	policy *models.Policy,
	dbType models.DatabaseType,
	startTime time.Time,
) *ReviewResult {
	violations := []Violation{}
	warnings := []Violation{}
	errorCount := 0
	warningCount := 0

	// Convert advice to violations
	for _, advice := range result.Advices {
		violation := ConvertAdviceToViolation(advice)

		// Extract rule type from advice (if available)
		// The title or code might give us hints about which rule triggered
		violation.RuleType = s.extractRuleTypeFromAdvice(advice, enabledRuleTypes)

		switch advice.Status {
		case types.Advice_ERROR:
			violations = append(violations, violation)
			errorCount++
		case types.Advice_WARNING:
			warnings = append(warnings, violation)
			warningCount++
		}
	}

	// Calculate skipped rules
	allRuleTypes := s.getAllRuleTypes(policy, dbType)
	skippedRules := s.calculateSkippedRules(enabledRuleTypes, allRuleTypes)

	executionTime := time.Since(startTime)

	return &ReviewResult{
		Valid:           errorCount == 0,
		Violations:      violations,
		Warnings:        warnings,
		TotalViolations: len(violations) + len(warnings),
		ErrorCount:      errorCount,
		WarningCount:    warningCount,
		PolicyID:        &policy.ID,
		PolicyName:      policy.Name,
		EnabledRules:    enabledRuleTypes,
		SkippedRules:    skippedRules,
		ExecutedRules:   len(enabledRuleTypes),
		ExecutionTime:   executionTime,
		ReviewedAt:      time.Now(),
	}
}

// extractRuleTypeFromAdvice attempts to extract rule type from advice
func (s *ReviewService) extractRuleTypeFromAdvice(advice *types.Advice, enabledRuleTypes []string) string {
	// Try to match title or content against enabled rule types
	title := strings.ToLower(advice.Title)
	content := strings.ToLower(advice.Content)

	for _, ruleType := range enabledRuleTypes {
		ruleLower := strings.ToLower(ruleType)
		if strings.Contains(title, ruleLower) || strings.Contains(content, ruleLower) {
			return ruleType
		}

		// Try matching last part of rule type (e.g., "no-select-all" from "statement.select.no-select-all")
		parts := strings.Split(ruleType, ".")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			if strings.Contains(title, lastPart) || strings.Contains(content, lastPart) {
				return ruleType
			}
		}
	}

	return "unknown"
}

// getAllRuleTypes returns all available rule types for the specified database type
func (s *ReviewService) getAllRuleTypes(policy *models.Policy, dbType models.DatabaseType) []string {
	// Parse config to get rule keys
	config, err := models.ParseSQLReviewConfig(policy.PolicyConfig)
	if err != nil {
		return []string{}
	}

	types := make([]string, 0, len(config.Rules))
	for _, rule := range config.Rules {
		// Only include rules for the current database type
		// Rules with "both" engine or matching engine
		if rule.Engine != dbType && rule.Engine != "both" {
			continue
		}
		types = append(types, rule.Type)
	}

	return types
}

// calculateSkippedRules determines which rules were not enabled
func (s *ReviewService) calculateSkippedRules(enabled, all []string) []SkippedRule {
	enabledMap := make(map[string]bool)
	for _, ruleType := range enabled {
		enabledMap[ruleType] = true
	}

	skipped := []SkippedRule{}
	for _, ruleType := range all {
		if !enabledMap[ruleType] {
			skipped = append(skipped, SkippedRule{
				RuleType: ruleType,
				Reason:   "not enabled in policy",
			})
		}
	}

	return skipped
}

// reviewWithNoPolicy returns a clean result when no policy is found
func (s *ReviewService) reviewWithNoPolicy(startTime time.Time) *ReviewResult {
	return &ReviewResult{
		Valid:           true,
		Violations:      []Violation{},
		Warnings:        []Violation{},
		TotalViolations: 0,
		ErrorCount:      0,
		WarningCount:    0,
		PolicyID:        nil,
		PolicyName:      "",
		EnabledRules:    []string{},
		SkippedRules:    []SkippedRule{},
		ExecutedRules:   0,
		ExecutionTime:   time.Since(startTime),
		ReviewedAt:      time.Now(),
	}
}

// convertDatabaseTypeToEngine converts models.DatabaseType to sql-reviewer engine
func convertDatabaseTypeToEngine(dbType models.DatabaseType) types.Engine {
	switch dbType {
	case models.DatabaseTypeMySQL:
		return types.Engine_MYSQL
	case models.DatabaseTypePostgreSQL:
		return types.Engine_POSTGRES
	default:
		return types.Engine_MYSQL
	}
}
