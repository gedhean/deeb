package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

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

func (m *MockDatabaseRepository) List(ctx context.Context, limit, offset int) ([]*models.Database, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*models.Database), args.Error(1)
}

func (m *MockDatabaseRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockDatabaseRepository) GetByProjectID(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	args := m.Called(ctx, projectID, limit, offset)
	return args.Get(0).([]*models.Database), args.Error(1)
}

func (m *MockDatabaseRepository) ListByProject(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	return m.GetByProjectID(ctx, projectID, limit, offset)
}

func (m *MockDatabaseRepository) ListByInstanceID(
	ctx context.Context,
	instanceID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	args := m.Called(ctx, instanceID, limit, offset)
	return args.Get(0).([]*models.Database), args.Error(1)
}

func (m *MockDatabaseRepository) ListByEnvironmentID(
	ctx context.Context,
	environmentID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	args := m.Called(ctx, environmentID, limit, offset)
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

// MockEnvironmentRepository is a mock implementation of EnvironmentRepository
type MockEnvironmentRepository struct {
	mock.Mock
}

func (m *MockEnvironmentRepository) Create(ctx context.Context, environment *models.Environment) error {
	args := m.Called(ctx, environment)
	return args.Error(0)
}

func (m *MockEnvironmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Environment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Environment), args.Error(1)
}

func (m *MockEnvironmentRepository) List(ctx context.Context, limit, offset int) ([]*models.Environment, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*models.Environment), args.Error(1)
}

func (m *MockEnvironmentRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockEnvironmentRepository) Update(ctx context.Context, environment *models.Environment) error {
	args := m.Called(ctx, environment)
	return args.Error(0)
}

func (m *MockEnvironmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockDatabaseInstanceRepository is a mock implementation of DatabaseInstanceRepository
type MockDatabaseInstanceRepository struct {
	mock.Mock
}

func (m *MockDatabaseInstanceRepository) Create(ctx context.Context, instance *models.DatabaseInstance) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *MockDatabaseInstanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.DatabaseInstance, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DatabaseInstance), args.Error(1)
}

func (m *MockDatabaseInstanceRepository) List(ctx context.Context, limit, offset int) ([]*models.DatabaseInstance, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*models.DatabaseInstance), args.Error(1)
}

func (m *MockDatabaseInstanceRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDatabaseInstanceRepository) Update(ctx context.Context, instance *models.DatabaseInstance) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *MockDatabaseInstanceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDatabaseInstanceRepository) ListByEnvironment(ctx context.Context, environmentID uuid.UUID, limit, offset int) ([]*models.DatabaseInstance, error) {
	args := m.Called(ctx, environmentID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DatabaseInstance), args.Error(1)
}

func (m *MockDatabaseInstanceRepository) CountByEnvironment(ctx context.Context, environmentID uuid.UUID) (int64, error) {
	args := m.Called(ctx, environmentID)
	return args.Get(0).(int64), args.Error(1)
}

func TestRiskAssessmentService_AssessRisk(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ctx := context.Background()

	tests := []struct {
		name         string
		sqlContent   string
		changeType   models.ChangeType
		envName      string
		dbType       models.DatabaseType
		expectedRisk models.RiskLevel
		setupMocks   func(*MockDatabaseRepository, *MockEnvironmentRepository, *MockDatabaseInstanceRepository)
	}{
		{
			name:         "Low risk - SELECT in Dev environment",
			sqlContent:   "SELECT * FROM users",
			changeType:   models.ChangeTypeDML,
			envName:      "Development",
			dbType:       models.DatabaseTypePostgreSQL,
			expectedRisk: models.RiskLevelLow,
			setupMocks: func(dbRepo *MockDatabaseRepository, envRepo *MockEnvironmentRepository, instanceRepo *MockDatabaseInstanceRepository) {
				dbID := uuid.New()
				envID := uuid.New()
				instanceID := uuid.New()

				dbRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
					ID:                 dbID,
					EnvironmentID:      envID,
					DatabaseInstanceID: instanceID,
				}, nil)

				envRepo.On("GetByID", ctx, envID).Return(&models.Environment{
					ID:   envID,
					Name: "Development",
				}, nil)

				instanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
					ID:           instanceID,
					DatabaseType: models.DatabaseTypePostgreSQL,
				}, nil)
			},
		},
		{
			name:         "Medium risk - ALTER TABLE in Staging",
			sqlContent:   "ALTER TABLE users ADD COLUMN email VARCHAR(255)",
			changeType:   models.ChangeTypeDDL,
			envName:      "Staging",
			dbType:       models.DatabaseTypeMySQL,
			expectedRisk: models.RiskLevelMedium,
			setupMocks: func(dbRepo *MockDatabaseRepository, envRepo *MockEnvironmentRepository, instanceRepo *MockDatabaseInstanceRepository) {
				dbID := uuid.New()
				envID := uuid.New()
				instanceID := uuid.New()

				dbRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
					ID:                 dbID,
					EnvironmentID:      envID,
					DatabaseInstanceID: instanceID,
				}, nil)

				envRepo.On("GetByID", ctx, envID).Return(&models.Environment{
					ID:   envID,
					Name: "Staging",
				}, nil)

				instanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
					ID:           instanceID,
					DatabaseType: models.DatabaseTypeMySQL,
				}, nil)
			},
		},
		{
			name:         "High risk - DROP TABLE in Production",
			sqlContent:   "DROP TABLE users",
			changeType:   models.ChangeTypeDDL,
			envName:      "Production",
			dbType:       models.DatabaseTypePostgreSQL,
			expectedRisk: models.RiskLevelHigh,
			setupMocks: func(dbRepo *MockDatabaseRepository, envRepo *MockEnvironmentRepository, instanceRepo *MockDatabaseInstanceRepository) {
				dbID := uuid.New()
				envID := uuid.New()
				instanceID := uuid.New()

				dbRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
					ID:                 dbID,
					EnvironmentID:      envID,
					DatabaseInstanceID: instanceID,
				}, nil)

				envRepo.On("GetByID", ctx, envID).Return(&models.Environment{
					ID:   envID,
					Name: "Production",
				}, nil)

				instanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
					ID:           instanceID,
					DatabaseType: models.DatabaseTypePostgreSQL,
				}, nil)
			},
		},
		{
			name:         "High risk - TRUNCATE in Production",
			sqlContent:   "TRUNCATE TABLE orders",
			changeType:   models.ChangeTypeDDL,
			envName:      "Production",
			dbType:       models.DatabaseTypeMySQL,
			expectedRisk: models.RiskLevelHigh,
			setupMocks: func(dbRepo *MockDatabaseRepository, envRepo *MockEnvironmentRepository, instanceRepo *MockDatabaseInstanceRepository) {
				dbID := uuid.New()
				envID := uuid.New()
				instanceID := uuid.New()

				dbRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
					ID:                 dbID,
					EnvironmentID:      envID,
					DatabaseInstanceID: instanceID,
				}, nil)

				envRepo.On("GetByID", ctx, envID).Return(&models.Environment{
					ID:   envID,
					Name: "Production",
				}, nil)

				instanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
					ID:           instanceID,
					DatabaseType: models.DatabaseTypeMySQL,
				}, nil)
			},
		},
		{
			name:         "High risk - DELETE without WHERE in Production",
			sqlContent:   "DELETE FROM users",
			changeType:   models.ChangeTypeDML,
			envName:      "Production",
			dbType:       models.DatabaseTypePostgreSQL,
			expectedRisk: models.RiskLevelHigh,
			setupMocks: func(dbRepo *MockDatabaseRepository, envRepo *MockEnvironmentRepository, instanceRepo *MockDatabaseInstanceRepository) {
				dbID := uuid.New()
				envID := uuid.New()
				instanceID := uuid.New()

				dbRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
					ID:                 dbID,
					EnvironmentID:      envID,
					DatabaseInstanceID: instanceID,
				}, nil)

				envRepo.On("GetByID", ctx, envID).Return(&models.Environment{
					ID:   envID,
					Name: "Production",
				}, nil)

				instanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
					ID:           instanceID,
					DatabaseType: models.DatabaseTypePostgreSQL,
				}, nil)
			},
		},
		{
			name:         "High risk - DELETE with WHERE in Production",
			sqlContent:   "DELETE FROM users WHERE id = 1",
			changeType:   models.ChangeTypeDML,
			envName:      "Production",
			dbType:       models.DatabaseTypePostgreSQL,
			expectedRisk: models.RiskLevelHigh,
			setupMocks: func(dbRepo *MockDatabaseRepository, envRepo *MockEnvironmentRepository, instanceRepo *MockDatabaseInstanceRepository) {
				dbID := uuid.New()
				envID := uuid.New()
				instanceID := uuid.New()

				dbRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
					ID:                 dbID,
					EnvironmentID:      envID,
					DatabaseInstanceID: instanceID,
				}, nil)

				envRepo.On("GetByID", ctx, envID).Return(&models.Environment{
					ID:   envID,
					Name: "Production",
				}, nil)

				instanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
					ID:           instanceID,
					DatabaseType: models.DatabaseTypePostgreSQL,
				}, nil)
			},
		},
		{
			name:         "Medium risk - UPDATE without WHERE in Staging",
			sqlContent:   "UPDATE users SET status = 'active'",
			changeType:   models.ChangeTypeDML,
			envName:      "Staging",
			dbType:       models.DatabaseTypeMySQL,
			expectedRisk: models.RiskLevelMedium,
			setupMocks: func(dbRepo *MockDatabaseRepository, envRepo *MockEnvironmentRepository, instanceRepo *MockDatabaseInstanceRepository) {
				dbID := uuid.New()
				envID := uuid.New()
				instanceID := uuid.New()

				dbRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
					ID:                 dbID,
					EnvironmentID:      envID,
					DatabaseInstanceID: instanceID,
				}, nil)

				envRepo.On("GetByID", ctx, envID).Return(&models.Environment{
					ID:   envID,
					Name: "Staging",
				}, nil)

				instanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
					ID:           instanceID,
					DatabaseType: models.DatabaseTypeMySQL,
				}, nil)
			},
		},
		{
			name:         "Low risk - INSERT in Dev",
			sqlContent:   "INSERT INTO users (name, email) VALUES ('John', 'john@example.com')",
			changeType:   models.ChangeTypeDML,
			envName:      "Development",
			dbType:       models.DatabaseTypePostgreSQL,
			expectedRisk: models.RiskLevelLow,
			setupMocks: func(dbRepo *MockDatabaseRepository, envRepo *MockEnvironmentRepository, instanceRepo *MockDatabaseInstanceRepository) {
				dbID := uuid.New()
				envID := uuid.New()
				instanceID := uuid.New()

				dbRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
					ID:                 dbID,
					EnvironmentID:      envID,
					DatabaseInstanceID: instanceID,
				}, nil)

				envRepo.On("GetByID", ctx, envID).Return(&models.Environment{
					ID:   envID,
					Name: "Development",
				}, nil)

				instanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
					ID:           instanceID,
					DatabaseType: models.DatabaseTypePostgreSQL,
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDBRepo := new(MockDatabaseRepository)
			mockEnvRepo := new(MockEnvironmentRepository)
			mockInstanceRepo := new(MockDatabaseInstanceRepository)

			tt.setupMocks(mockDBRepo, mockEnvRepo, mockInstanceRepo)

			service := NewRiskAssessmentService(
				mockDBRepo,
				mockEnvRepo,
				mockInstanceRepo,
				logger,
			)

			targetDatabaseID := uuid.New()
			riskLevel, err := service.AssessRisk(ctx, tt.sqlContent, tt.changeType, targetDatabaseID)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRisk, riskLevel)

			mockDBRepo.AssertExpectations(t)
			mockEnvRepo.AssertExpectations(t)
			mockInstanceRepo.AssertExpectations(t)
		})
	}
}

func TestRiskAssessmentService_analyzeSQLContent(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := &RiskAssessmentService{logger: logger}

	tests := []struct {
		name       string
		sqlContent string
		changeType models.ChangeType
		expected   int
	}{
		{
			name:       "SELECT query - no risk",
			sqlContent: "SELECT * FROM users",
			changeType: models.ChangeTypeDML,
			expected:   0,
		},
		{
			name:       "DROP TABLE - high risk",
			sqlContent: "DROP TABLE users",
			changeType: models.ChangeTypeDDL,
			expected:   7, // 2 (DDL) + 5 (DROP TABLE)
		},
		{
			name:       "TRUNCATE TABLE - high risk",
			sqlContent: "TRUNCATE TABLE orders",
			changeType: models.ChangeTypeDDL,
			expected:   7, // 2 (DDL) + 5 (TRUNCATE)
		},
		{
			name:       "DELETE without WHERE - high risk",
			sqlContent: "DELETE FROM users",
			changeType: models.ChangeTypeDML,
			expected:   5, // DELETE without WHERE
		},
		{
			name:       "DELETE with WHERE - moderate risk",
			sqlContent: "DELETE FROM users WHERE id = 1",
			changeType: models.ChangeTypeDML,
			expected:   3, // DELETE with WHERE
		},
		{
			name:       "UPDATE without WHERE - moderate risk",
			sqlContent: "UPDATE users SET status = 'active'",
			changeType: models.ChangeTypeDML,
			expected:   3, // UPDATE without WHERE
		},
		{
			name:       "UPDATE with WHERE - low risk",
			sqlContent: "UPDATE users SET status = 'active' WHERE id = 1",
			changeType: models.ChangeTypeDML,
			expected:   1, // UPDATE with WHERE
		},
		{
			name:       "ALTER TABLE - moderate risk",
			sqlContent: "ALTER TABLE users ADD COLUMN email VARCHAR(255)",
			changeType: models.ChangeTypeDDL,
			expected:   3, // 2 (DDL) + 1 (ALTER TABLE)
		},
		{
			name:       "INSERT - low risk",
			sqlContent: "INSERT INTO users (name) VALUES ('John')",
			changeType: models.ChangeTypeDML,
			expected:   1, // INSERT
		},
		{
			name:       "CREATE INDEX - moderate risk",
			sqlContent: "CREATE INDEX idx_users_email ON users(email)",
			changeType: models.ChangeTypeDDL,
			expected:   3, // 2 (DDL) + 1 (CREATE INDEX)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.analyzeSQLContent(tt.sqlContent, tt.changeType)
			assert.Equal(t, tt.expected, score)
		})
	}
}

func TestRiskAssessmentService_analyzeEnvironment(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := &RiskAssessmentService{logger: logger}

	tests := []struct {
		name     string
		envName  string
		expected int
	}{
		{
			name:     "Production - high risk",
			envName:  "Production",
			expected: 3,
		},
		{
			name:     "Prod - high risk",
			envName:  "Prod",
			expected: 3,
		},
		{
			name:     "Staging - moderate risk",
			envName:  "Staging",
			expected: 1,
		},
		{
			name:     "Stage - moderate risk",
			envName:  "Stage",
			expected: 1,
		},
		{
			name:     "Development - low risk",
			envName:  "Development",
			expected: 0,
		},
		{
			name:     "Dev - low risk",
			envName:  "Dev",
			expected: 0,
		},
		{
			name:     "Test - low risk",
			envName:  "Test",
			expected: 0,
		},
		{
			name:     "Local - low risk",
			envName:  "Local",
			expected: 0,
		},
		{
			name:     "Unknown environment - moderate risk",
			envName:  "QA",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.analyzeEnvironment(tt.envName)
			assert.Equal(t, tt.expected, score)
		})
	}
}

func TestRiskAssessmentService_analyzeDatabaseType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := &RiskAssessmentService{logger: logger}

	tests := []struct {
		name     string
		dbType   models.DatabaseType
		expected int
	}{
		{
			name:     "PostgreSQL - no additional risk",
			dbType:   models.DatabaseTypePostgreSQL,
			expected: 0,
		},
		{
			name:     "MySQL - no additional risk",
			dbType:   models.DatabaseTypeMySQL,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.analyzeDatabaseType(tt.dbType)
			assert.Equal(t, tt.expected, score)
		})
	}
}

func TestRiskAssessmentService_calculateRiskLevel(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	service := &RiskAssessmentService{logger: logger}

	tests := []struct {
		name         string
		riskScore    int
		expectedRisk models.RiskLevel
	}{
		{
			name:         "Score 0 - Low",
			riskScore:    0,
			expectedRisk: models.RiskLevelLow,
		},
		{
			name:         "Score 1 - Low",
			riskScore:    1,
			expectedRisk: models.RiskLevelLow,
		},
		{
			name:         "Score 2 - Low",
			riskScore:    2,
			expectedRisk: models.RiskLevelLow,
		},
		{
			name:         "Score 3 - Medium",
			riskScore:    3,
			expectedRisk: models.RiskLevelMedium,
		},
		{
			name:         "Score 4 - Medium",
			riskScore:    4,
			expectedRisk: models.RiskLevelMedium,
		},
		{
			name:         "Score 5 - Medium",
			riskScore:    5,
			expectedRisk: models.RiskLevelMedium,
		},
		{
			name:         "Score 6 - High",
			riskScore:    6,
			expectedRisk: models.RiskLevelHigh,
		},
		{
			name:         "Score 10 - High",
			riskScore:    10,
			expectedRisk: models.RiskLevelHigh,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			riskLevel := service.calculateRiskLevel(tt.riskScore)
			assert.Equal(t, tt.expectedRisk, riskLevel)
		})
	}
}

func TestRiskAssessmentService_AssessRisk_ErrorHandling(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ctx := context.Background()

	t.Run("handles database not found error", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		targetDatabaseID := uuid.New()
		mockDBRepo.On("GetByID", ctx, targetDatabaseID).Return(nil, assert.AnError)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		riskLevel, err := service.AssessRisk(ctx, "SELECT * FROM users", models.ChangeTypeDML, targetDatabaseID)

		assert.Error(t, err)
		assert.Equal(t, models.RiskLevelMedium, riskLevel) // Returns Medium on error

		mockDBRepo.AssertExpectations(t)
	})

	t.Run("handles environment not found error", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		targetDatabaseID := uuid.New()
		envID := uuid.New()
		instanceID := uuid.New()

		mockDBRepo.On("GetByID", ctx, targetDatabaseID).Return(&models.Database{
			ID:                 targetDatabaseID,
			EnvironmentID:      envID,
			DatabaseInstanceID: instanceID,
		}, nil)

		mockEnvRepo.On("GetByID", ctx, envID).Return(nil, assert.AnError)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		riskLevel, err := service.AssessRisk(ctx, "SELECT * FROM users", models.ChangeTypeDML, targetDatabaseID)

		assert.Error(t, err)
		assert.Equal(t, models.RiskLevelMedium, riskLevel) // Returns Medium on error

		mockDBRepo.AssertExpectations(t)
		mockEnvRepo.AssertExpectations(t)
	})

	t.Run("handles database instance not found error", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		targetDatabaseID := uuid.New()
		envID := uuid.New()
		instanceID := uuid.New()

		mockDBRepo.On("GetByID", ctx, targetDatabaseID).Return(&models.Database{
			ID:                 targetDatabaseID,
			EnvironmentID:      envID,
			DatabaseInstanceID: instanceID,
		}, nil)

		mockEnvRepo.On("GetByID", ctx, envID).Return(&models.Environment{
			ID:   envID,
			Name: "Development",
		}, nil)

		mockInstanceRepo.On("GetByID", ctx, instanceID).Return(nil, assert.AnError)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		riskLevel, err := service.AssessRisk(ctx, "SELECT * FROM users", models.ChangeTypeDML, targetDatabaseID)

		assert.Error(t, err)
		assert.Equal(t, models.RiskLevelMedium, riskLevel) // Returns Medium on error

		mockDBRepo.AssertExpectations(t)
		mockEnvRepo.AssertExpectations(t)
		mockInstanceRepo.AssertExpectations(t)
	})
}

func TestRiskAssessmentService_EdgeCases(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ctx := context.Background()

	t.Run("handles empty SQL content", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		dbID := uuid.New()
		envID := uuid.New()
		instanceID := uuid.New()

		mockDBRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
			ID:                 dbID,
			EnvironmentID:      envID,
			DatabaseInstanceID: instanceID,
		}, nil)

		mockEnvRepo.On("GetByID", ctx, envID).Return(&models.Environment{
			ID:   envID,
			Name: "Development",
		}, nil)

		mockInstanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
			ID:           instanceID,
			DatabaseType: models.DatabaseTypePostgreSQL,
		}, nil)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		riskLevel, err := service.AssessRisk(ctx, "", models.ChangeTypeDML, dbID)

		assert.NoError(t, err)
		assert.Equal(t, models.RiskLevelLow, riskLevel) // Empty SQL = low risk

		mockDBRepo.AssertExpectations(t)
		mockEnvRepo.AssertExpectations(t)
		mockInstanceRepo.AssertExpectations(t)
	})

	t.Run("handles whitespace-only SQL content", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		dbID := uuid.New()
		envID := uuid.New()
		instanceID := uuid.New()

		mockDBRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
			ID:                 dbID,
			EnvironmentID:      envID,
			DatabaseInstanceID: instanceID,
		}, nil)

		mockEnvRepo.On("GetByID", ctx, envID).Return(&models.Environment{
			ID:   envID,
			Name: "Development",
		}, nil)

		mockInstanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
			ID:           instanceID,
			DatabaseType: models.DatabaseTypePostgreSQL,
		}, nil)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		riskLevel, err := service.AssessRisk(ctx, "   \n\t  ", models.ChangeTypeDML, dbID)

		assert.NoError(t, err)
		assert.Equal(t, models.RiskLevelLow, riskLevel) // Whitespace-only SQL = low risk

		mockDBRepo.AssertExpectations(t)
		mockEnvRepo.AssertExpectations(t)
		mockInstanceRepo.AssertExpectations(t)
	})

	t.Run("handles SQL with comments", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		dbID := uuid.New()
		envID := uuid.New()
		instanceID := uuid.New()

		mockDBRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
			ID:                 dbID,
			EnvironmentID:      envID,
			DatabaseInstanceID: instanceID,
		}, nil)

		mockEnvRepo.On("GetByID", ctx, envID).Return(&models.Environment{
			ID:   envID,
			Name: "Production",
		}, nil)

		mockInstanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
			ID:           instanceID,
			DatabaseType: models.DatabaseTypePostgreSQL,
		}, nil)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		// SQL with comments should still detect DROP TABLE
		riskLevel, err := service.AssessRisk(ctx, "-- Drop users table\nDROP TABLE users", models.ChangeTypeDDL, dbID)

		assert.NoError(t, err)
		assert.Equal(t, models.RiskLevelHigh, riskLevel) // DROP TABLE in Production = high risk

		mockDBRepo.AssertExpectations(t)
		mockEnvRepo.AssertExpectations(t)
		mockInstanceRepo.AssertExpectations(t)
	})

	t.Run("handles case-insensitive SQL matching", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		dbID := uuid.New()
		envID := uuid.New()
		instanceID := uuid.New()

		mockDBRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
			ID:                 dbID,
			EnvironmentID:      envID,
			DatabaseInstanceID: instanceID,
		}, nil)

		mockEnvRepo.On("GetByID", ctx, envID).Return(&models.Environment{
			ID:   envID,
			Name: "Production",
		}, nil)

		mockInstanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
			ID:           instanceID,
			DatabaseType: models.DatabaseTypePostgreSQL,
		}, nil)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		// Lowercase SQL should still be detected
		riskLevel, err := service.AssessRisk(ctx, "drop table users", models.ChangeTypeDDL, dbID)

		assert.NoError(t, err)
		assert.Equal(t, models.RiskLevelHigh, riskLevel) // DROP TABLE in Production = high risk

		mockDBRepo.AssertExpectations(t)
		mockEnvRepo.AssertExpectations(t)
		mockInstanceRepo.AssertExpectations(t)
	})

	t.Run("handles multiple destructive operations (takes first)", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		dbID := uuid.New()
		envID := uuid.New()
		instanceID := uuid.New()

		mockDBRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
			ID:                 dbID,
			EnvironmentID:      envID,
			DatabaseInstanceID: instanceID,
		}, nil)

		mockEnvRepo.On("GetByID", ctx, envID).Return(&models.Environment{
			ID:   envID,
			Name: "Staging",
		}, nil)

		mockInstanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
			ID:           instanceID,
			DatabaseType: models.DatabaseTypePostgreSQL,
		}, nil)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		// Multiple destructive operations - should detect first one
		riskLevel, err := service.AssessRisk(ctx, "DROP TABLE users; TRUNCATE TABLE orders", models.ChangeTypeDDL, dbID)

		assert.NoError(t, err)
		assert.Equal(t, models.RiskLevelHigh, riskLevel) // DROP TABLE (score 7) + Staging (1) = 8 = High

		mockDBRepo.AssertExpectations(t)
		mockEnvRepo.AssertExpectations(t)
		mockInstanceRepo.AssertExpectations(t)
	})

	t.Run("handles complex SQL statements", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		dbID := uuid.New()
		envID := uuid.New()
		instanceID := uuid.New()

		mockDBRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
			ID:                 dbID,
			EnvironmentID:      envID,
			DatabaseInstanceID: instanceID,
		}, nil)

		mockEnvRepo.On("GetByID", ctx, envID).Return(&models.Environment{
			ID:   envID,
			Name: "Development",
		}, nil)

		mockInstanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
			ID:           instanceID,
			DatabaseType: models.DatabaseTypePostgreSQL,
		}, nil)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		// Complex ALTER TABLE with multiple operations
		complexSQL := `ALTER TABLE users
			ADD COLUMN email VARCHAR(255),
			ADD COLUMN phone VARCHAR(20),
			ALTER COLUMN name TYPE VARCHAR(100)`

		riskLevel, err := service.AssessRisk(ctx, complexSQL, models.ChangeTypeDDL, dbID)

		assert.NoError(t, err)
		assert.Equal(t, models.RiskLevelMedium, riskLevel) // ALTER TABLE (score 3) + Dev (0) = 3 = Medium

		mockDBRepo.AssertExpectations(t)
		mockEnvRepo.AssertExpectations(t)
		mockInstanceRepo.AssertExpectations(t)
	})

	t.Run("handles environment name variations", func(t *testing.T) {
		testCases := []struct {
			name         string
			envName      string
			expectedRisk models.RiskLevel
		}{
			{"Production", "Production", models.RiskLevelHigh},     // ALTER TABLE (3) + Production (3) = 6 = High
			{"Prod", "Prod", models.RiskLevelHigh},                 // ALTER TABLE (3) + Prod (3) = 6 = High
			{"PRODUCTION", "PRODUCTION", models.RiskLevelHigh},     // ALTER TABLE (3) + PRODUCTION (3) = 6 = High
			{"prod", "prod", models.RiskLevelHigh},                 // ALTER TABLE (3) + prod (3) = 6 = High
			{"Staging", "Staging", models.RiskLevelMedium},         // ALTER TABLE (3) + Staging (1) = 4 = Medium
			{"Stage", "Stage", models.RiskLevelMedium},             // ALTER TABLE (3) + Stage (1) = 4 = Medium
			{"Development", "Development", models.RiskLevelMedium}, // ALTER TABLE (3) + Dev (0) = 3 = Medium
			{"Dev", "Dev", models.RiskLevelMedium},                 // ALTER TABLE (3) + Dev (0) = 3 = Medium
			{"Test", "Test", models.RiskLevelMedium},               // ALTER TABLE (3) + Test (0) = 3 = Medium
			{"Local", "Local", models.RiskLevelMedium},             // ALTER TABLE (3) + Local (0) = 3 = Medium
			{"QA", "QA", models.RiskLevelMedium},                   // ALTER TABLE (3) + Unknown (1) = 4 = Medium
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				mockDBRepo := new(MockDatabaseRepository)
				mockEnvRepo := new(MockEnvironmentRepository)
				mockInstanceRepo := new(MockDatabaseInstanceRepository)

				dbID := uuid.New()
				envID := uuid.New()
				instanceID := uuid.New()

				mockDBRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
					ID:                 dbID,
					EnvironmentID:      envID,
					DatabaseInstanceID: instanceID,
				}, nil)

				mockEnvRepo.On("GetByID", ctx, envID).Return(&models.Environment{
					ID:   envID,
					Name: tc.envName,
				}, nil)

				mockInstanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
					ID:           instanceID,
					DatabaseType: models.DatabaseTypePostgreSQL,
				}, nil)

				service := NewRiskAssessmentService(
					mockDBRepo,
					mockEnvRepo,
					mockInstanceRepo,
					logger,
				)

				// Use ALTER TABLE which gives score 3
				riskLevel, err := service.AssessRisk(
					ctx,
					"ALTER TABLE users ADD COLUMN email VARCHAR(255)",
					models.ChangeTypeDDL,
					dbID,
				)

				assert.NoError(t, err)
				assert.Equal(t, tc.expectedRisk, riskLevel)

				mockDBRepo.AssertExpectations(t)
				mockEnvRepo.AssertExpectations(t)
				mockInstanceRepo.AssertExpectations(t)
			})
		}
	})

	t.Run("handles combined risk scenarios", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		dbID := uuid.New()
		envID := uuid.New()
		instanceID := uuid.New()

		mockDBRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
			ID:                 dbID,
			EnvironmentID:      envID,
			DatabaseInstanceID: instanceID,
		}, nil)

		mockEnvRepo.On("GetByID", ctx, envID).Return(&models.Environment{
			ID:   envID,
			Name: "Production",
		}, nil)

		mockInstanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
			ID:           instanceID,
			DatabaseType: models.DatabaseTypePostgreSQL,
		}, nil)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		// DROP TABLE (score 7) + Production (score 3) = 10 = High
		riskLevel, err := service.AssessRisk(ctx, "DROP TABLE users", models.ChangeTypeDDL, dbID)

		assert.NoError(t, err)
		assert.Equal(t, models.RiskLevelHigh, riskLevel)

		mockDBRepo.AssertExpectations(t)
		mockEnvRepo.AssertExpectations(t)
		mockInstanceRepo.AssertExpectations(t)
	})

	t.Run("handles UPDATE with WHERE clause correctly", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		dbID := uuid.New()
		envID := uuid.New()
		instanceID := uuid.New()

		mockDBRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
			ID:                 dbID,
			EnvironmentID:      envID,
			DatabaseInstanceID: instanceID,
		}, nil)

		mockEnvRepo.On("GetByID", ctx, envID).Return(&models.Environment{
			ID:   envID,
			Name: "Development",
		}, nil)

		mockInstanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
			ID:           instanceID,
			DatabaseType: models.DatabaseTypePostgreSQL,
		}, nil)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		// UPDATE with WHERE should have lower risk than UPDATE without WHERE
		riskLevel, err := service.AssessRisk(ctx, "UPDATE users SET status = 'active' WHERE id = 1", models.ChangeTypeDML, dbID)

		assert.NoError(t, err)
		assert.Equal(t, models.RiskLevelLow, riskLevel) // UPDATE with WHERE (1) + Dev (0) = 1 = Low

		mockDBRepo.AssertExpectations(t)
		mockEnvRepo.AssertExpectations(t)
		mockInstanceRepo.AssertExpectations(t)
	})

	t.Run("handles DROP INDEX operations", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		dbID := uuid.New()
		envID := uuid.New()
		instanceID := uuid.New()

		mockDBRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
			ID:                 dbID,
			EnvironmentID:      envID,
			DatabaseInstanceID: instanceID,
		}, nil)

		mockEnvRepo.On("GetByID", ctx, envID).Return(&models.Environment{
			ID:   envID,
			Name: "Production",
		}, nil)

		mockInstanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
			ID:           instanceID,
			DatabaseType: models.DatabaseTypePostgreSQL,
		}, nil)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		// DROP INDEX is high risk
		riskLevel, err := service.AssessRisk(ctx, "DROP INDEX idx_users_email", models.ChangeTypeDDL, dbID)

		assert.NoError(t, err)
		assert.Equal(t, models.RiskLevelHigh, riskLevel) // DROP INDEX (7) + Production (3) = 10 = High

		mockDBRepo.AssertExpectations(t)
		mockEnvRepo.AssertExpectations(t)
		mockInstanceRepo.AssertExpectations(t)
	})

	t.Run("handles TRUNCATE with CASCADE", func(t *testing.T) {
		mockDBRepo := new(MockDatabaseRepository)
		mockEnvRepo := new(MockEnvironmentRepository)
		mockInstanceRepo := new(MockDatabaseInstanceRepository)

		dbID := uuid.New()
		envID := uuid.New()
		instanceID := uuid.New()

		mockDBRepo.On("GetByID", ctx, mock.Anything).Return(&models.Database{
			ID:                 dbID,
			EnvironmentID:      envID,
			DatabaseInstanceID: instanceID,
		}, nil)

		mockEnvRepo.On("GetByID", ctx, envID).Return(&models.Environment{
			ID:   envID,
			Name: "Staging",
		}, nil)

		mockInstanceRepo.On("GetByID", ctx, instanceID).Return(&models.DatabaseInstance{
			ID:           instanceID,
			DatabaseType: models.DatabaseTypePostgreSQL,
		}, nil)

		service := NewRiskAssessmentService(
			mockDBRepo,
			mockEnvRepo,
			mockInstanceRepo,
			logger,
		)

		// TRUNCATE with CASCADE should still be detected
		riskLevel, err := service.AssessRisk(ctx, "TRUNCATE TABLE users CASCADE", models.ChangeTypeDDL, dbID)

		assert.NoError(t, err)
		assert.Equal(t, models.RiskLevelHigh, riskLevel) // TRUNCATE (7) + Staging (1) = 8 = High

		mockDBRepo.AssertExpectations(t)
		mockEnvRepo.AssertExpectations(t)
		mockInstanceRepo.AssertExpectations(t)
	})
}
