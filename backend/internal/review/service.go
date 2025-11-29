package review

import (
	"context"
	"database/sql"
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
	policyRepo      repository.PolicyRepository
	environmentRepo repository.EnvironmentRepository
	schemaCache     *SchemaCache
	logger          *zap.Logger

	// Configuration
	defaultTimeout time.Duration
}

// NewReviewService creates a new enhanced review service
func NewReviewService(
	policyRepo repository.PolicyRepository,
	environmentRepo repository.EnvironmentRepository,
	schemaCache *SchemaCache,
	logger *zap.Logger,
) *ReviewService {
	return &ReviewService{
		policyRepo:      policyRepo,
		environmentRepo: environmentRepo,
		schemaCache:     schemaCache,
		logger:          logger,
		defaultTimeout:  30 * time.Second,
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

	// DatabaseEngine is the database engine
	DatabaseEngine models.DatabaseEngine

	// Schema is optional pre-loaded schema metadata
	Schema *types.DatabaseSchemaMetadata

	// Driver is optional database connection for DML dry-run and other checks
	Driver *sql.DB

	// ChangeType specifies the type of database change (DDL, DML, DDL_DML)
	ChangeType *types.PlanCheckRunConfig_ChangeDatabaseType

	// EnableQueryLogging enables detailed SQL query logging for debugging
	EnableQueryLogging bool

	// UsePostgresDatabaseOwner enables using PostgreSQL database owner role
	UsePostgresDatabaseOwner bool

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
	engine := convertDatabaseEngineToEngine(opts.DatabaseEngine)

	// Create reviewer instance
	rev := reviewer.New(engine)

	// Build rule configuration for sql-reviewer
	rules, enabledRuleTypes := s.buildRulesForReviewer(config, opts.DatabaseEngine)

	// Log detailed rule information for debugging
	s.logger.Debug("Built rules for reviewer",
		zap.Int("total_rules", len(rules)),
		zap.Strings("enabled_rule_types", enabledRuleTypes))

	// Log each rule for debugging
	for i, rule := range rules {
		s.logger.Debug("Rule details",
			zap.Int("rule_index", i),
			zap.String("rule_type", rule.Type),
			zap.String("rule_level", string(rule.Level)),
			zap.String("rule_engine", rule.Engine.String()),
			zap.Any("rule_payload", rule.Payload))
	}

	// Configure reviewer with rules
	if len(rules) > 0 {
		// Create Config for the reviewer
		reviewConfig := &reviewerConfig.Config{
			ID:    "policy-" + policy.ID.String(),
			Rules: rules,
		}

		s.logger.Debug("Configuring reviewer with rules",
			zap.String("config_id", reviewConfig.ID),
			zap.Int("config_rules_count", len(reviewConfig.Rules)))

		// WithConfigObject returns the reviewer for method chaining - must capture it!
		rev = rev.WithConfigObject(reviewConfig)

		s.logger.Debug("Reviewer configured successfully")
	} else {
		s.logger.Warn("No rules to configure - reviewer will run without rules")
	}

	// Get schema for schema-aware validation if available
	var schema *types.DatabaseSchemaMetadata
	if opts.Schema != nil {
		schema = opts.Schema
	} else if opts.DatabaseID != nil && s.schemaCache != nil {
		schema = s.schemaCache.Get(ctx, *opts.DatabaseID)
	}

	// Build reviewer options
	reviewOpts := []reviewer.ReviewOption{}

	// Add database driver if provided
	if opts.Driver != nil {
		s.logger.Debug("Adding database driver to reviewer options")
		reviewOpts = append(reviewOpts, reviewer.WithDriver(opts.Driver))
	}

	// Add change type if provided
	if opts.ChangeType != nil {
		s.logger.Debug("Adding change type to reviewer options",
			zap.Int32("change_type", int32(*opts.ChangeType)))
		reviewOpts = append(reviewOpts, reviewer.WithChangeType(*opts.ChangeType))
	}

	// Add query logging if enabled
	if opts.EnableQueryLogging {
		s.logger.Debug("Enabling query logging in reviewer")
		reviewOpts = append(reviewOpts, reviewer.WithQueryLogging(true))
	}

	// Add PostgreSQL database owner option if enabled
	if opts.UsePostgresDatabaseOwner && opts.DatabaseEngine == models.DatabaseEnginePostgreSQL {
		s.logger.Debug("Enabling PostgreSQL database owner role")
		reviewOpts = append(reviewOpts, reviewer.WithPostgresDatabaseOwner(true))
	}

	// Execute review
	var result *reviewer.ReviewResult
	if schema != nil {
		s.logger.Info("Running schema-aware SQL review",
			zap.String("database_id", opts.DatabaseID.String()),
			zap.Int("rules", len(rules)),
			zap.Int("review_options", len(reviewOpts)))
		result, err = rev.ReviewWithSchema(ctx, opts.SQLContent, schema, reviewOpts...)
	} else {
		s.logger.Info("Running SQL review without schema",
			zap.Int("rules", len(rules)),
			zap.String("sql_content", opts.SQLContent),
			zap.String("database_engine", string(opts.DatabaseEngine)),
			zap.Int("review_options", len(reviewOpts)))
		// DEBUG: print each rule being sent to the reviewer and its payload
		for _, rule := range rules {
			s.logger.Debug("Rule being sent to the reviewer",
				zap.String("rule_type", rule.Type),
				zap.String("rule_level", string(rule.Level)),
				zap.String("rule_engine", string(rule.Engine)),
				zap.Any("rule_payload", rule.Payload))
		}
		for _, reviewOpt := range reviewOpts {
			s.logger.Debug("Review option being sent to the reviewer",
				zap.Any("review_opt", reviewOpt))
		}
		result, err = rev.Review(ctx, opts.SQLContent, reviewOpts...)
	}

	if err != nil {
		return nil, fmt.Errorf("review failed: %w", err)
	}

	// Log raw reviewer result for debugging
	s.logger.Debug("SQL Reviewer returned result",
		zap.Int("advice_count", len(result.Advices)),
		zap.Any("advices", result.Advices))

	// Convert result to our format
	reviewResult := s.convertReviewerResult(result, enabledRuleTypes, policy, opts.DatabaseEngine, start)

	return reviewResult, nil
}

// loadPolicyForEnvironment loads the SQL_REVIEW policy for an environment
func (s *ReviewService) loadPolicyForEnvironment(ctx context.Context, environmentID uuid.UUID) (*models.Policy, error) {
	// If environment ID is nil/zero, treat as unassigned (no environment)
	if environmentID == uuid.Nil {
		s.logger.Debug("No environment assigned, skipping policy load")
		return nil, nil
	}

	// Fetch the environment
	environment, err := s.environmentRepo.GetByID(ctx, environmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch environment: %w", err)
	}

	// If environment has no associated policy, return nil (no policy to apply)
	if environment.PolicyID == nil {
		s.logger.Info("No policy associated with environment",
			zap.String("environment_id", environmentID.String()),
			zap.String("environment_name", environment.Name))
		return nil, nil
	}

	// Load the policy
	policy, err := s.policyRepo.GetByID(ctx, *environment.PolicyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load policy: %w", err)
	}

	// Verify it's a SQL_REVIEW policy
	if policy.Type != models.PolicyTypeSQLReview {
		s.logger.Warn("Environment policy is not SQL_REVIEW type",
			zap.String("environment_id", environmentID.String()),
			zap.String("policy_id", policy.ID.String()),
			zap.String("policy_type", string(policy.Type)))
		return nil, nil
	}

	s.logger.Debug("Loaded policy for environment",
		zap.String("environment_id", environmentID.String()),
		zap.String("policy_id", policy.ID.String()),
		zap.String("policy_name", policy.Name))

	return policy, nil
}

// buildRulesForReviewer builds sql-reviewer rules from policy config
func (s *ReviewService) buildRulesForReviewer(
	config *models.SQLReviewPolicyConfig,
	dbType models.DatabaseEngine,
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

		// Convert to sql-reviewer rule (pass target database type for "both" engine rules)
		storeRule := ConvertRuleMetadataToStoreRule(*metadata, ruleConfig.Level, ruleConfig.Payload, dbType)
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
	dbType models.DatabaseEngine,
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

	// Log when we can't classify the rule type
	contentSnippet := content
	if len(contentSnippet) > 100 {
		contentSnippet = contentSnippet[:100] + "..."
	}
	s.logger.Warn("Failed to classify rule type from advice",
		zap.String("advice_title", advice.Title),
		zap.String("content_snippet", contentSnippet),
		zap.Int("enabled_rules_count", len(enabledRuleTypes)),
		zap.Strings("enabled_rule_types", enabledRuleTypes))

	return "unknown"
}

// getAllRuleTypes returns all available rule types for the specified database type
func (s *ReviewService) getAllRuleTypes(policy *models.Policy, dbType models.DatabaseEngine) []string {
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

// convertDatabaseEngineToEngine converts models.DatabaseEngine to sql-reviewer engine
func convertDatabaseEngineToEngine(dbType models.DatabaseEngine) types.Engine {
	switch dbType {
	case models.DatabaseEngineMySQL:
		return types.Engine_MYSQL
	case models.DatabaseEnginePostgreSQL:
		return types.Engine_POSTGRES
	default:
		return types.Engine_MYSQL
	}
}
