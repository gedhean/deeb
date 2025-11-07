package review

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/review/sqlreviewer"
	"github.com/nsxbet/sql-reviewer/pkg/types"
	"go.uber.org/zap"
)

// ReviewService orchestrates SQL Review using sql-reviewer package
type ReviewService struct {
	policyRepo   repository.PolicyRepository
	databaseRepo repository.DatabaseRepository
	advisor      *sqlreviewer.Advisor
	logger       *zap.Logger
}

// NewReviewService creates a new SQL Review service
func NewReviewService(
	policyRepo repository.PolicyRepository,
	databaseRepo repository.DatabaseRepository,
	advisor *sqlreviewer.Advisor,
	logger *zap.Logger,
) *ReviewService {
	return &ReviewService{
		policyRepo:   policyRepo,
		databaseRepo: databaseRepo,
		advisor:      advisor,
		logger:       logger,
	}
}

// ReviewOptions contains options for running SQL Review
type ReviewOptions struct {
	// ChangePlanID is the ID of the Change Plan being reviewed
	ChangePlanID uuid.UUID
	// SQLContent is the SQL content to review
	SQLContent string
	// DatabaseType is the type of database (MySQL or PostgreSQL)
	DatabaseType models.DatabaseType
}

// ReviewResultWithSummary extends ReviewResult with summary information
type ReviewResultWithSummary struct {
	ReviewResult
	// PolicyID is the ID of the policy used for review
	PolicyID *uuid.UUID `json:"policy_id,omitempty"`
	// PolicyName is the name of the policy used for review
	PolicyName string `json:"policy_name,omitempty"`
	// EnabledRules is the list of rule names that were enabled
	EnabledRules []string `json:"enabled_rules"`
	// SkippedRules is the list of rule names that were skipped
	SkippedRules []string `json:"skipped_rules"`
}

// Review performs SQL Review for a Change Plan using sql-reviewer
func (s *ReviewService) Review(ctx context.Context, opts ReviewOptions) (*ReviewResultWithSummary, error) {
	// Step 1: Load Policy (SQL_REVIEW type)
	policies, err := s.policyRepo.GetByType(ctx, models.PolicyTypeSQLReview, 1, 0)
	if err != nil {
		s.logger.Error("Failed to load SQL_REVIEW policy", zap.Error(err))
		return nil, fmt.Errorf("failed to load SQL_REVIEW policy: %w", err)
	}

	if len(policies) == 0 {
		s.logger.Warn("No SQL_REVIEW policy found, running with no rules")
		return s.reviewWithNoRules(ctx, opts)
	}

	policy := policies[0]
	s.logger.Info("Loaded SQL_REVIEW policy",
		zap.String("policy_id", policy.ID.String()),
		zap.String("policy_name", policy.Name))

	// Step 2: Parse policy config to get enabled rules
	config, err := models.ParseSQLReviewConfig(policy.PolicyConfig)
	if err != nil {
		s.logger.Error("Failed to parse policy config", zap.Error(err))
		return nil, fmt.Errorf("failed to parse policy config: %w", err)
	}

	// Step 3: Filter rules by database type and build types.SQLReviewRule list
	rules := s.buildRulesFromConfig(config, opts.DatabaseType)
	enabledRuleNames := s.getEnabledRuleNames(config, opts.DatabaseType)
	allRuleNames := s.getAllRuleNames(opts.DatabaseType)
	skippedRuleNames := s.getSkippedRuleNames(enabledRuleNames, allRuleNames)

	// Step 4: Run sql-reviewer advisor
	adviceList, err := s.advisor.Check(ctx, opts.SQLContent, opts.DatabaseType, rules)
	if err != nil {
		s.logger.Error("Failed to run sql-reviewer check", zap.Error(err))
		return nil, fmt.Errorf("failed to run sql-reviewer check: %w", err)
	}

	// Step 5: Build ReviewResult from Advice list
	reviewResult := s.convertAdviceToReviewResult(adviceList, enabledRuleNames)

	// Step 6: Return results with summary
	return &ReviewResultWithSummary{
		ReviewResult: reviewResult,
		PolicyID:     &policy.ID,
		PolicyName:   policy.Name,
		EnabledRules: enabledRuleNames,
		SkippedRules: skippedRuleNames,
	}, nil
}

// buildRulesFromConfig builds types.SQLReviewRule list from policy config
func (s *ReviewService) buildRulesFromConfig(
	config *models.SQLReviewPolicyConfig,
	dbType models.DatabaseType,
) []*types.SQLReviewRule {
	rules := make([]*types.SQLReviewRule, 0)
	allRules := GetRulesByEngine(dbType)

	for ruleKey, ruleConfig := range config.Rules {
		// Only process rules for the current database type
		if ruleConfig.Engine != dbType {
			continue
		}

		// Find rule metadata
		var ruleMetadata *RuleMetadata
		for _, categoryRules := range allRules {
			for _, meta := range categoryRules {
				if meta.Type == ruleConfig.Type {
					ruleMetadata = &meta
					break
				}
			}
			if ruleMetadata != nil {
				break
			}
		}

		if ruleMetadata == nil {
			s.logger.Warn("Rule metadata not found", zap.String("rule", ruleKey), zap.String("type", ruleConfig.Type))
			continue
		}

		// Convert to types.SQLReviewRule
		storeRule := ConvertRuleMetadataToStoreRule(*ruleMetadata, ruleConfig.Level, ruleConfig.Payload)
		rules = append(rules, storeRule)
	}

	return rules
}

// getEnabledRuleNames returns list of enabled rule names from config
func (s *ReviewService) getEnabledRuleNames(config *models.SQLReviewPolicyConfig, dbType models.DatabaseType) []string {
	enabled := make([]string, 0)
	for ruleKey, ruleConfig := range config.Rules {
		if ruleConfig.Engine == dbType {
			enabled = append(enabled, ruleKey)
		}
	}
	return enabled
}

// getAllRuleNames returns all available rule names for the database type
func (s *ReviewService) getAllRuleNames(dbType models.DatabaseType) []string {
	allRules := GetRulesByEngine(dbType)
	names := make([]string, 0)
	for _, categoryRules := range allRules {
		for _, meta := range categoryRules {
			names = append(names, meta.Type)
		}
	}
	return names
}

// getSkippedRuleNames returns rule names that were not enabled
func (s *ReviewService) getSkippedRuleNames(enabled []string, all []string) []string {
	enabledMap := make(map[string]bool)
	for _, name := range enabled {
		enabledMap[name] = true
	}

	skipped := make([]string, 0)
	for _, name := range all {
		if !enabledMap[name] {
			skipped = append(skipped, name)
		}
	}
	return skipped
}

// convertAdviceToReviewResult converts sql-reviewer Advice list to ReviewResult
func (s *ReviewService) convertAdviceToReviewResult(adviceList []*types.Advice, enabledRuleNames []string) ReviewResult {
	errors := 0
	warnings := 0
	infos := 0

	for _, advice := range adviceList {
		switch advice.Status {
		case types.Advice_ERROR:
			errors++
		case types.Advice_WARNING:
			warnings++
		case types.Advice_SUCCESS, types.Advice_STATUS_UNSPECIFIED:
			// Treat SUCCESS and UNSPECIFIED as Info level
			infos++
		default:
			infos++
		}
	}

	valid := errors == 0 && len(adviceList) == 0

	return ReviewResult{
		Valid:           valid,
		Results:         []Result{{ValidatorName: "sql-reviewer", Valid: valid, Violations: adviceList}},
		TotalViolations: len(adviceList),
		Errors:          errors,
		Warnings:        warnings,
		Infos:           infos,
	}
}

// reviewWithNoRules runs review with no rules (fallback when no policy found)
func (s *ReviewService) reviewWithNoRules(ctx context.Context, opts ReviewOptions) (*ReviewResultWithSummary, error) {
	return &ReviewResultWithSummary{
		ReviewResult: ReviewResult{
			Valid:           true,
			Results:         []Result{},
			TotalViolations: 0,
			Errors:          0,
			Warnings:        0,
			Infos:           0,
		},
		PolicyID:     nil,
		PolicyName:   "",
		EnabledRules: []string{},
		SkippedRules: []string{},
	}, nil
}
