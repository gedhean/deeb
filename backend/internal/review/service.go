package review

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// ReviewService orchestrates SQL Review by loading policies, determining enabled validators,
// and running them to aggregate results
type ReviewService struct {
	policyRepo     repository.PolicyRepository
	databaseRepo   repository.DatabaseRepository
	engine         Engine
	logger         *zap.Logger
	validatorNames map[string]bool // Available validator names for quick lookup
}

// NewReviewService creates a new SQL Review service
func NewReviewService(
	policyRepo repository.PolicyRepository,
	databaseRepo repository.DatabaseRepository,
	engine Engine,
	logger *zap.Logger,
) *ReviewService {
	return &ReviewService{
		policyRepo:     policyRepo,
		databaseRepo:   databaseRepo,
		engine:         engine,
		logger:         logger,
		validatorNames: make(map[string]bool),
	}
}

// RegisterValidator registers a validator with the service
func (s *ReviewService) RegisterValidator(validator Validator) error {
	name := validator.GetName()
	if err := s.engine.RegisterValidator(validator); err != nil {
		return fmt.Errorf("failed to register validator %s: %w", name, err)
	}
	s.validatorNames[name] = true
	s.logger.Info("Registered validator", zap.String("validator", name))
	return nil
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
	// EnabledValidators is the list of validator names that were enabled
	EnabledValidators []string `json:"enabled_validators"`
	// SkippedValidators is the list of validator names that were skipped
	SkippedValidators []string `json:"skipped_validators"`
}

// Review performs SQL Review for a Change Plan
// It loads the Policy (SQL_REVIEW type) from Environment via Change Plan's target database,
// determines which validators are enabled based on policy config,
// runs only enabled validators, and aggregates results
func (s *ReviewService) Review(ctx context.Context, opts ReviewOptions) (*ReviewResultWithSummary, error) {
	// Step 1: Load Change Plan's target database to get Environment
	// Note: For MVP, we'll get the database type from opts.DatabaseType
	// In a full implementation, we'd load the Change Plan and get TargetDatabaseID
	// For now, we'll use a simplified approach: get the first SQL_REVIEW policy

	// Step 2: Load Policy (SQL_REVIEW type) from Environment
	// For MVP: Get first SQL_REVIEW policy
	// TODO: In Phase 2, add environment_id to policies table or use a junction table
	policies, err := s.policyRepo.GetByType(ctx, models.PolicyTypeSQLReview, 1, 0)
	if err != nil {
		s.logger.Error("Failed to load SQL_REVIEW policy", zap.Error(err))
		return nil, fmt.Errorf("failed to load SQL_REVIEW policy: %w", err)
	}

	if len(policies) == 0 {
		s.logger.Warn("No SQL_REVIEW policy found, running all validators")
		// If no policy found, run all validators
		return s.reviewWithAllValidators(ctx, opts)
	}

	policy := policies[0]
	s.logger.Info("Loaded SQL_REVIEW policy",
		zap.String("policy_id", policy.ID.String()),
		zap.String("policy_name", policy.Name))

	// Step 3: Determine which validators are enabled based on policy config
	enabledValidators, err := s.getEnabledValidators(policy.PolicyConfig)
	if err != nil {
		s.logger.Error("Failed to parse policy config", zap.Error(err))
		return nil, fmt.Errorf("failed to parse policy config: %w", err)
	}

	// Step 4: Create a temporary engine with only enabled validators
	tempEngine := s.createFilteredEngine(enabledValidators)

	// Step 5: Run enabled validators
	reviewResult := tempEngine.Review(ctx, opts.SQLContent, opts.DatabaseType)

	// Step 6: Determine skipped validators
	skippedValidators := s.getSkippedValidators(enabledValidators)

	// Step 7: Return results with summary
	return &ReviewResultWithSummary{
		ReviewResult:      reviewResult,
		PolicyID:          &policy.ID,
		PolicyName:        policy.Name,
		EnabledValidators: enabledValidators,
		SkippedValidators: skippedValidators,
	}, nil
}

// getEnabledValidators extracts enabled validator names from policy config
// Policy config format: {"validators": {"postgresql-syntax": true, "mysql-syntax": false, ...}}
func (s *ReviewService) getEnabledValidators(config models.PolicyConfig) ([]string, error) {
	validatorsConfig, ok := config["validators"]
	if !ok {
		// If no validators config, enable all validators
		return s.getAllValidatorNames(), nil
	}

	validatorsMap, ok := validatorsConfig.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid validators config format: expected map[string]interface{}")
	}

	var enabled []string
	for name, enabledVal := range validatorsMap {
		// Check if validator exists
		if !s.validatorNames[name] {
			s.logger.Warn("Validator not found in registry", zap.String("validator", name))
			continue
		}

		// Check if enabled (accepts bool true or string "true")
		var isEnabled bool
		switch v := enabledVal.(type) {
		case bool:
			isEnabled = v
		case string:
			isEnabled = v == "true" || v == "1"
		default:
			isEnabled = false
		}

		if isEnabled {
			enabled = append(enabled, name)
		}
	}

	// If no validators enabled, return all validators (fallback)
	if len(enabled) == 0 {
		s.logger.Warn("No validators enabled in policy config, enabling all validators")
		return s.getAllValidatorNames(), nil
	}

	return enabled, nil
}

// getAllValidatorNames returns all registered validator names
func (s *ReviewService) getAllValidatorNames() []string {
	validators := s.engine.GetRegistry().GetAll()
	names := make([]string, 0, len(validators))
	for _, v := range validators {
		names = append(names, v.GetName())
	}
	return names
}

// getSkippedValidators returns validator names that were not enabled
func (s *ReviewService) getSkippedValidators(enabled []string) []string {
	allNames := s.getAllValidatorNames()
	enabledMap := make(map[string]bool)
	for _, name := range enabled {
		enabledMap[name] = true
	}

	var skipped []string
	for _, name := range allNames {
		if !enabledMap[name] {
			skipped = append(skipped, name)
		}
	}
	return skipped
}

// createFilteredEngine creates a temporary engine with only enabled validators
func (s *ReviewService) createFilteredEngine(enabledValidators []string) Engine {
	tempRegistry := NewValidatorRegistry()
	registry := s.engine.GetRegistry()

	for _, name := range enabledValidators {
		validator, err := registry.Get(name)
		if err != nil {
			s.logger.Warn("Failed to get validator for filtered engine",
				zap.String("validator", name),
				zap.Error(err))
			continue
		}
		if err := tempRegistry.Register(validator); err != nil {
			s.logger.Warn("Failed to register validator in filtered engine",
				zap.String("validator", name),
				zap.Error(err))
		}
	}

	return NewEngineWithRegistry(tempRegistry)
}

// reviewWithAllValidators runs review with all validators (fallback when no policy found)
func (s *ReviewService) reviewWithAllValidators(ctx context.Context, opts ReviewOptions) (*ReviewResultWithSummary, error) {
	reviewResult := s.engine.Review(ctx, opts.SQLContent, opts.DatabaseType)

	return &ReviewResultWithSummary{
		ReviewResult:      reviewResult,
		PolicyID:          nil,
		PolicyName:        "",
		EnabledValidators: s.getAllValidatorNames(),
		SkippedValidators: []string{},
	}, nil
}
