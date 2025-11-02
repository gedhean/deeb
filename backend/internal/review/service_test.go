package review

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
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

func (m *MockPolicyRepository) GetByType(ctx context.Context, policyType models.PolicyType, limit, offset int) ([]*models.Policy, error) {
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

func (m *MockDatabaseRepository) Update(ctx context.Context, database *models.Database) error {
	args := m.Called(ctx, database)
	return args.Error(0)
}

func (m *MockDatabaseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDatabaseRepository) List(ctx context.Context, filters repository.ListFilters) ([]*models.Database, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Database), args.Error(1)
}

func (m *MockDatabaseRepository) Count(ctx context.Context, filters repository.ListFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

func TestReviewService(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	t.Run("creates service successfully", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := NewEngine()

		service := NewReviewService(policyRepo, databaseRepo, engine, logger)
		assert.NotNil(t, service)
	})

	t.Run("registers validators", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := NewEngine()

		service := NewReviewService(policyRepo, databaseRepo, engine, logger)

		validator := validators.NewSecurityValidator()
		err := service.RegisterValidator(validator)
		require.NoError(t, err)

		// Verify validator is registered
		registry := service.engine.GetRegistry()
		registeredValidator, err := registry.Get("security")
		require.NoError(t, err)
		assert.Equal(t, "security", registeredValidator.GetName())
	})

	t.Run("reviews SQL with policy config", func(t *testing.T) {
		policyRepo := new(MockPolicyRepository)
		databaseRepo := new(MockDatabaseRepository)
		engine := NewEngine()

		service := NewReviewService(policyRepo, databaseRepo, engine, logger)

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
		opts := ReviewOptions{
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
		engine := NewEngine()

		service := NewReviewService(policyRepo, databaseRepo, engine, logger)

		// Register validators
		securityValidator := validators.NewSecurityValidator()
		require.NoError(t, service.RegisterValidator(securityValidator))

		// Setup policy mock to return no policies
		policyRepo.On("GetByType", ctx, models.PolicyTypeSQLReview, 1, 0).Return([]*models.Policy{}, nil)

		// Run review
		opts := ReviewOptions{
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
		engine := NewEngine()

		service := NewReviewService(policyRepo, databaseRepo, engine, logger)

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
		opts := ReviewOptions{
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
		engine := NewEngine()

		service := NewReviewService(policyRepo, databaseRepo, engine, logger)

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
		opts := ReviewOptions{
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
		engine := NewEngine()

		service := NewReviewService(policyRepo, databaseRepo, engine, logger)

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
		opts := ReviewOptions{
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
}
