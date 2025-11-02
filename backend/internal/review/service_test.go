package review_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/review"
	"github.com/nsxbet/deeb/internal/review/validators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockPolicyRepository is a mock implementation of PolicyRepository
type MockPolicyRepository struct {
	mock.Mock
}

func (m *MockPolicyRepository) Create(ctx context.Context, policy *models.Policy) error {
	args := m.Called(ctx, policy)
	return args.Error(0)
}

func (m *MockPolicyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Policy, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Policy), args.Error(1)
}

func (m *MockPolicyRepository) List(ctx context.Context, limit, offset int) ([]*models.Policy, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Policy), args.Error(1)
}

func (m *MockPolicyRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPolicyRepository) GetByType(
	ctx context.Context,
	policyType models.PolicyType,
	limit, offset int,
) ([]*models.Policy, error) {
	args := m.Called(ctx, policyType, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Policy), args.Error(1)
}

func (m *MockPolicyRepository) CountByType(ctx context.Context, policyType models.PolicyType) (int64, error) {
	args := m.Called(ctx, policyType)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPolicyRepository) Update(ctx context.Context, policy *models.Policy) error {
	args := m.Called(ctx, policy)
	return args.Error(0)
}

func (m *MockPolicyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockDatabaseRepository is a mock implementation of DatabaseRepository
type MockDatabaseRepository struct {
	mock.Mock
}

func (m *MockDatabaseRepository) Create(ctx context.Context, database *models.Database) error {
	args := m.Called(ctx, database)
	return args.Error(0)
}

func (m *MockDatabaseRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Database, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Database), args.Error(1)
}

func (m *MockDatabaseRepository) GetByProjectID(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	args := m.Called(ctx, projectID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Database), args.Error(1)
}

func (m *MockDatabaseRepository) ListByProject(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	return m.GetByProjectID(ctx, projectID, limit, offset)
}

func (m *MockDatabaseRepository) ListByEnvironmentID(
	ctx context.Context,
	environmentID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	args := m.Called(ctx, environmentID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Database), args.Error(1)
}

func (m *MockDatabaseRepository) ListByInstanceID(
	ctx context.Context,
	instanceID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	args := m.Called(ctx, instanceID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Database), args.Error(1)
}

func (m *MockDatabaseRepository) Update(ctx context.Context, database *models.Database) error {
	args := m.Called(ctx, database)
	return args.Error(0)
}

func (m *MockDatabaseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDatabaseRepository) List(ctx context.Context, limit, offset int) ([]*models.Database, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Database), args.Error(1)
}

func (m *MockDatabaseRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Get(0).(int), args.Error(1)
}

func TestReviewService(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	t.Run("creates service successfully", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)
		assert.NotNil(t, service)
	})

	t.Run("registers validators", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		validator := validators.NewSecurityValidator()
		err := service.RegisterValidator(validator)
		require.NoError(t, err)

		// The validator is registered if no error is returned
		// We can verify it works by running a review
	})

	t.Run("reviews SQL with policy config", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test SQL Review Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security": true,
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		// Run review
		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, policyID, *result.PolicyID)
		assert.Equal(t, "Test SQL Review Policy", result.PolicyName)
		assert.Contains(t, result.EnabledValidators, "security")
		assert.False(t, result.Valid) // Should have violations
		assert.Greater(t, result.Errors, 0)

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles missing policy gracefully", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock to return no policies
		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{}, nil)

		// Run review
		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, result.PolicyID)
		assert.Empty(t, result.PolicyName)
		assert.Contains(t, result.EnabledValidators, "security")
		assert.False(t, result.Valid) // Should have violations

		policyRepo.AssertExpectations(t)
	})

	t.Run("filters validators based on policy config", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register multiple validators
		securityValidator := validators.NewSecurityValidator()
		performanceValidator := validators.NewPerformanceValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))
		require.NoError(t, service.RegisterValidator(performanceValidator))

		// Setup policy mock - only enable security validator
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security":    true,
					"performance": false,
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		// Run review
		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "CREATE TABLE users (id INT)",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.EnabledValidators, "security")
		assert.NotContains(t, result.EnabledValidators, "performance")
		assert.Contains(t, result.SkippedValidators, "performance")

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles policy config with no validators", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock with empty validators config
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		// Run review - should fallback to all validators
		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.EnabledValidators, "security") // Should enable all

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles policy config without validators key", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock without validators key
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"other_config": "value",
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		// Run review - should enable all validators
		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.EnabledValidators, "security") // Should enable all

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles policy repository error", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Setup policy mock to return error
		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return(nil, assert.AnError)

		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to load SQL_REVIEW policy")

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles invalid policy config format", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock with invalid validators config (not a map)
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": "invalid_format", // Should be map[string]interface{}
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid validators config format")

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles validator registration error", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validator with nil engine (should handle gracefully)
		validator := validators.NewSecurityValidator()
		err := service.RegisterValidator(validator)
		// Should succeed normally
		require.NoError(t, err)
	})

	t.Run("handles multiple validators execution", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register multiple validators
		securityValidator := validators.NewSecurityValidator()
		performanceValidator := validators.NewPerformanceValidator()
		bestPracticesValidator := validators.NewBestPracticesValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))
		require.NoError(t, service.RegisterValidator(performanceValidator))
		require.NoError(t, service.RegisterValidator(bestPracticesValidator))

		// Setup policy mock - enable all validators
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security":       true,
					"performance":    true,
					"best-practices": true,
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		// Run review with SQL that triggers multiple validators
		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.EnabledValidators, 3)
		assert.Contains(t, result.EnabledValidators, "security")
		assert.Contains(t, result.EnabledValidators, "performance")
		assert.Contains(t, result.EnabledValidators, "best-practices")
		assert.False(t, result.Valid) // Should have violations

		policyRepo.AssertExpectations(t)
	})

	t.Run("aggregates results from multiple validators", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register multiple validators
		securityValidator := validators.NewSecurityValidator()
		performanceValidator := validators.NewPerformanceValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))
		require.NoError(t, service.RegisterValidator(performanceValidator))

		// Setup policy mock - enable all validators
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security":    true,
					"performance": true,
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		// Run review with SQL that triggers both validators
		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "CREATE TABLE users (id INT, name VARCHAR(100))",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Valid)             // Should have violations from both validators
		assert.Greater(t, len(result.Results), 0) // Should have results from multiple validators

		// Verify results are aggregated
		totalViolations := 0
		for _, r := range result.Results {
			totalViolations += len(r.Violations)
		}
		assert.Greater(t, totalViolations, 0)

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles policy config with string boolean values", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		performanceValidator := validators.NewPerformanceValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))
		require.NoError(t, service.RegisterValidator(performanceValidator))

		// Setup policy mock with string boolean values
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security":    "true",  // String instead of bool
					"performance": "false", // String instead of bool
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.EnabledValidators, "security")
		assert.NotContains(t, result.EnabledValidators, "performance")
		assert.Contains(t, result.SkippedValidators, "performance")

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles policy config with numeric boolean values", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock with numeric string "1" for true
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security": "1", // Numeric string should be treated as true
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.EnabledValidators, "security")

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles unknown validator names in policy config", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock with unknown validator name
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security":          true,
					"unknown-validator": true, // Should be ignored
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.EnabledValidators, "security")
		assert.NotContains(t, result.EnabledValidators, "unknown-validator")

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles empty SQL content", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security": true,
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid) // Empty SQL should be valid

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles whitespace-only SQL content", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security": true,
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "   \n\t  ",
			DatabaseType: models.DatabaseTypePostgreSQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid) // Whitespace-only SQL should be valid

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles both MySQL and PostgreSQL database types", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security": true,
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil).Times(2)

		// Test MySQL
		optsMySQL := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		resultMySQL, err := service.Review(ctx, optsMySQL)
		require.NoError(t, err)
		assert.NotNil(t, resultMySQL)
		assert.False(t, resultMySQL.Valid)

		// Test PostgreSQL
		optsPostgreSQL := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypePostgreSQL,
		}

		resultPostgreSQL, err := service.Review(ctx, optsPostgreSQL)
		require.NoError(t, err)
		assert.NotNil(t, resultPostgreSQL)
		assert.False(t, resultPostgreSQL.Valid)

		policyRepo.AssertExpectations(t)
	})

	t.Run("correctly calculates skipped validators", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register multiple validators
		securityValidator := validators.NewSecurityValidator()
		performanceValidator := validators.NewPerformanceValidator()
		bestPracticesValidator := validators.NewBestPracticesValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))
		require.NoError(t, service.RegisterValidator(performanceValidator))
		require.NoError(t, service.RegisterValidator(bestPracticesValidator))

		// Setup policy mock - only enable security
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security":       true,
					"performance":    false,
					"best-practices": false,
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.EnabledValidators, "security")
		assert.Len(t, result.EnabledValidators, 1)
		assert.Contains(t, result.SkippedValidators, "performance")
		assert.Contains(t, result.SkippedValidators, "best-practices")
		assert.Len(t, result.SkippedValidators, 2)

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles all validators enabled scenario", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register multiple validators
		securityValidator := validators.NewSecurityValidator()
		performanceValidator := validators.NewPerformanceValidator()
		backwardsCompatibilityValidator := validators.NewBackwardsCompatibilityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))
		require.NoError(t, service.RegisterValidator(performanceValidator))
		require.NoError(t, service.RegisterValidator(backwardsCompatibilityValidator))

		// Setup policy mock - enable all validators
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security":                true,
					"performance":             true,
					"backwards-compatibility": true,
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.EnabledValidators, 3)
		assert.Empty(t, result.SkippedValidators)

		policyRepo.AssertExpectations(t)
	})

	t.Run("handles no validators registered scenario", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Don't register any validators

		// Setup policy mock
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security": true,
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		// Should have no enabled validators since none are registered
		assert.Empty(t, result.EnabledValidators)
		assert.True(t, result.Valid) // No validators = no violations

		policyRepo.AssertExpectations(t)
	})

	t.Run("correctly populates PolicyID and PolicyName in result", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock
		policyID := uuid.New()
		policyName := "Custom SQL Review Policy"
		policy := &models.Policy{
			ID:   policyID,
			Name: policyName,
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security": true,
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.PolicyID)
		assert.Equal(t, policyID, *result.PolicyID)
		assert.Equal(t, policyName, result.PolicyName)

		policyRepo.AssertExpectations(t)
	})

	t.Run("correctly sets PolicyID and PolicyName to nil/empty when no policy found", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock to return no policies
		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{}, nil)

		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "DROP TABLE users",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, result.PolicyID)
		assert.Empty(t, result.PolicyName)

		policyRepo.AssertExpectations(t)
	})

	t.Run("result aggregation includes errors and warnings counts", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := review.NewEngine()

		service := review.NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		performanceValidator := validators.NewPerformanceValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))
		require.NoError(t, service.RegisterValidator(performanceValidator))

		// Setup policy mock - enable all validators
		policyID := uuid.New()
		policy := &models.Policy{
			ID:   policyID,
			Name: "Test Policy",
			Type: models.PolicyTypeSQLReview,
			PolicyConfig: models.PolicyConfig{
				"validators": map[string]interface{}{
					"security":    true,
					"performance": true,
				},
			},
		}

		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{policy}, nil)

		// Run review with SQL that triggers both validators
		opts := review.ReviewOptions{
			ChangePlanID: uuid.New(),
			SQLContent:   "CREATE TABLE users (id INT)",
			DatabaseType: models.DatabaseTypeMySQL,
		}

		result, err := service.Review(ctx, opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Valid)
		// Should have aggregated counts
		assert.Greater(t, result.Errors+result.Warnings+result.Infos, 0)

		policyRepo.AssertExpectations(t)
	})
}
