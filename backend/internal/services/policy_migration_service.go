package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/review"
	"go.uber.org/zap"
)

// PolicyMigrationService handles migration of policies between schema versions
type PolicyMigrationService struct {
	policyRepo repository.PolicyRepository
	logger     *zap.Logger
}

// NewPolicyMigrationService creates a new policy migration service
func NewPolicyMigrationService(
	policyRepo repository.PolicyRepository,
	logger *zap.Logger,
) *PolicyMigrationService {
	return &PolicyMigrationService{
		policyRepo: policyRepo,
		logger:     logger,
	}
}

// MigrationResult represents the result of migrating a single policy
type MigrationResult struct {
	PolicyID     uuid.UUID `json:"policy_id"`
	PolicyName   string    `json:"policy_name"`
	FromVersion  string    `json:"from_version"`
	ToVersion    string    `json:"to_version"`
	RulesAdded   int       `json:"rules_added"`
	RulesUpdated int       `json:"rules_updated"`
	RulesRemoved int       `json:"rules_removed"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	MigratedAt   time.Time `json:"migrated_at"`
}

// BatchMigrationResult represents the result of migrating multiple policies
type BatchMigrationResult struct {
	TotalPolicies int                `json:"total_policies"`
	SuccessCount  int                `json:"success_count"`
	FailureCount  int                `json:"failure_count"`
	Results       []*MigrationResult `json:"results"`
	StartedAt     time.Time          `json:"started_at"`
	CompletedAt   time.Time          `json:"completed_at"`
}

// MigratePolicyToV2 migrates a single policy from V1 to V2
func (s *PolicyMigrationService) MigratePolicyToV2(ctx context.Context, policyID uuid.UUID) (*MigrationResult, error) {
	result := &MigrationResult{
		PolicyID:   policyID,
		MigratedAt: time.Now(),
	}

	// Get existing policy
	policy, err := s.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to get policy: %v", err)
		return result, err
	}

	result.PolicyName = policy.Name
	result.FromVersion = policy.SchemaVersion

	// Check if already on V2
	if policy.SchemaVersion == models.PolicySchemaV2 {
		result.Success = true
		result.ToVersion = models.PolicySchemaV2
		result.Error = "already on latest version"
		return result, nil
	}

	// Parse existing config
	oldConfig, err := models.ParseSQLReviewConfig(policy.PolicyConfig)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to parse policy config: %v", err)
		return result, err
	}

	// Create new V2 config with enhanced rules
	newConfig := s.upgradeConfigToV2(oldConfig)

	// Calculate statistics
	result.RulesAdded = len(newConfig.Rules) - len(oldConfig.Rules)
	result.RulesUpdated = 0 // All old rules are preserved, only new ones added
	result.RulesRemoved = 0

	// Update policy with new config and schema version
	policy.PolicyConfig = newConfig.ToPolicyConfig()
	policy.SchemaVersion = models.PolicySchemaV2

	if err := s.policyRepo.Update(ctx, policy); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to update policy: %v", err)
		return result, err
	}

	result.Success = true
	result.ToVersion = models.PolicySchemaV2

	s.logger.Info("Successfully migrated policy to V2",
		zap.String("policy_id", policyID.String()),
		zap.String("policy_name", policy.Name),
		zap.Int("rules_added", result.RulesAdded))

	return result, nil
}

// upgradeConfigToV2 upgrades a V1 policy config to V2 with comprehensive rules
func (s *PolicyMigrationService) upgradeConfigToV2(oldConfig *models.SQLReviewPolicyConfig) *models.SQLReviewPolicyConfig {
	newConfig := &models.SQLReviewPolicyConfig{
		Rules: make(map[string]models.SQLReviewRule),
	}

	// Copy all existing rules (preserve user configuration)
	for key, rule := range oldConfig.Rules {
		newConfig.Rules[key] = rule
	}

	// Add recommended new rules with default configurations
	// These are sensible defaults that can be customized later
	recommendedRules := s.getRecommendedRulesForV2()
	for key, rule := range recommendedRules {
		// Only add if not already configured
		if _, exists := newConfig.Rules[key]; !exists {
			newConfig.Rules[key] = rule
		}
	}

	return newConfig
}

// getRecommendedRulesForV2 returns a set of recommended rules for V2 migration
// These are conservative, commonly-useful rules that won't break existing workflows
func (s *PolicyMigrationService) getRecommendedRulesForV2() map[string]models.SQLReviewRule {
	recommended := make(map[string]models.SQLReviewRule)

	// Get all available rules
	allRules := review.GetAvailableRules()

	// Add common security rules
	for engine, categories := range allRules {
		dbType := models.DatabaseType(engine)

		// Add security rules (usually important)
		if securityRules, ok := categories["Security"]; ok {
			for _, metadata := range securityRules {
				key := fmt.Sprintf("%s_%s", engine, metadata.Type)
				recommended[key] = models.SQLReviewRule{
					Type:    metadata.Type,
					Level:   models.SQLReviewRuleLevelWarning, // Start with warnings to avoid breaking changes
					Engine:  dbType,
					Comment: metadata.Description,
					Payload: s.getDefaultPayloadForRule(metadata),
				}
			}
		}

		// Add critical syntax rules
		if syntaxRules, ok := categories["Engine"]; ok {
			for _, metadata := range syntaxRules {
				// Only add critical syntax validation
				if metadata.Type == "engine.mysql.use-innodb" || metadata.Type == "engine.postgres.require-where" {
					key := fmt.Sprintf("%s_%s", engine, metadata.Type)
					recommended[key] = models.SQLReviewRule{
						Type:    metadata.Type,
						Level:   models.SQLReviewRuleLevelWarning,
						Engine:  dbType,
						Comment: metadata.Description,
						Payload: s.getDefaultPayloadForRule(metadata),
					}
				}
			}
		}
	}

	return recommended
}

// getDefaultPayloadForRule returns sensible default payload for a rule
func (s *PolicyMigrationService) getDefaultPayloadForRule(metadata review.RuleMetadata) map[string]interface{} {
	payload := make(map[string]interface{})

	for _, param := range metadata.Parameters {
		if param.Default != nil {
			payload[param.Name] = param.Default
		}
	}

	return payload
}

// MigrateAllPolicies migrates all policies from V1 to V2
func (s *PolicyMigrationService) MigrateAllPolicies(ctx context.Context) (*BatchMigrationResult, error) {
	startTime := time.Now()

	result := &BatchMigrationResult{
		StartedAt: startTime,
		Results:   make([]*MigrationResult, 0),
	}

	// Get all SQL_REVIEW policies
	policies, err := s.policyRepo.GetByType(ctx, models.PolicyTypeSQLReview, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}

	result.TotalPolicies = len(policies)

	// Migrate each policy
	for _, policy := range policies {
		// Skip if already on V2
		if policy.SchemaVersion == models.PolicySchemaV2 {
			s.logger.Info("Skipping policy already on V2",
				zap.String("policy_id", policy.ID.String()),
				zap.String("policy_name", policy.Name))
			continue
		}

		migrationResult, err := s.MigratePolicyToV2(ctx, policy.ID)
		if err != nil {
			s.logger.Error("Failed to migrate policy",
				zap.String("policy_id", policy.ID.String()),
				zap.Error(err))
			result.FailureCount++
		} else if migrationResult.Success {
			result.SuccessCount++
		} else {
			result.FailureCount++
		}

		result.Results = append(result.Results, migrationResult)
	}

	result.CompletedAt = time.Now()

	s.logger.Info("Batch migration completed",
		zap.Int("total", result.TotalPolicies),
		zap.Int("success", result.SuccessCount),
		zap.Int("failure", result.FailureCount),
		zap.Duration("duration", result.CompletedAt.Sub(result.StartedAt)))

	return result, nil
}

// GetMigrationStatus returns the migration status for all policies
func (s *PolicyMigrationService) GetMigrationStatus(ctx context.Context) (map[string]int, error) {
	policies, err := s.policyRepo.GetByType(ctx, models.PolicyTypeSQLReview, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}

	status := map[string]int{
		"total":               len(policies),
		models.PolicySchemaV1: 0,
		models.PolicySchemaV2: 0,
		"unknown":             0,
		"needs_migration":     0,
	}

	for _, policy := range policies {
		switch policy.SchemaVersion {
		case models.PolicySchemaV1:
			status[models.PolicySchemaV1]++
			status["needs_migration"]++
		case models.PolicySchemaV2:
			status[models.PolicySchemaV2]++
		default:
			status["unknown"]++
		}
	}

	return status, nil
}
