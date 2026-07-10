package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/mock"
)

// MockProjectRepository is a mock implementation of ProjectRepository
type MockProjectRepository struct {
	mock.Mock
}

func (m *MockProjectRepository) Create(ctx context.Context, project *models.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Project), args.Error(1)
}

func (m *MockProjectRepository) List(
	ctx context.Context,
	filters repository.ProjectFilters,
	limit, offset int,
) ([]*models.Project, error) {
	args := m.Called(ctx, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Project), args.Error(1)
}

func (m *MockProjectRepository) ListByUserID(
	ctx context.Context,
	userID uuid.UUID,
	filters repository.ProjectFilters,
	limit, offset int,
) ([]*models.Project, error) {
	args := m.Called(ctx, userID, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Project), args.Error(1)
}

func (m *MockProjectRepository) Count(ctx context.Context, filters repository.ProjectFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProjectRepository) CountByUserID(
	ctx context.Context,
	userID uuid.UUID,
	filters repository.ProjectFilters,
) (int64, error) {
	args := m.Called(ctx, userID, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProjectRepository) Update(ctx context.Context, project *models.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *MockProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockProjectMemberRepository is a mock implementation of ProjectMemberRepository
type MockProjectMemberRepository struct {
	mock.Mock
}

func (m *MockProjectMemberRepository) AddMember(ctx context.Context, member *models.ProjectMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *MockProjectMemberRepository) AddMembersBatch(ctx context.Context, members []*models.ProjectMember) error {
	args := m.Called(ctx, members)
	return args.Error(0)
}

func (m *MockProjectMemberRepository) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	args := m.Called(ctx, projectID, userID)
	return args.Error(0)
}

func (m *MockProjectMemberRepository) GetMembers(ctx context.Context, projectID uuid.UUID) ([]*models.ProjectMember, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ProjectMember), args.Error(1)
}

func (m *MockProjectMemberRepository) GetMemberRole(
	ctx context.Context,
	projectID, userID uuid.UUID,
) (*models.ProjectMember, error) {
	args := m.Called(ctx, projectID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProjectMember), args.Error(1)
}

func (m *MockProjectMemberRepository) UpdateRole(ctx context.Context, projectID, userID uuid.UUID, role rbac.ProjectRole) error {
	args := m.Called(ctx, projectID, userID, role)
	return args.Error(0)
}

func (m *MockProjectMemberRepository) GetUserProjectIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(
	ctx context.Context,
	filters repository.UserFilters,
	limit, offset int,
) ([]*models.User, error) {
	args := m.Called(ctx, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) Count(ctx context.Context, filters repository.UserFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
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

func (m *MockDatabaseRepository) List(
	ctx context.Context,
	filters repository.DatabaseFilters,
	limit, offset int,
) ([]*models.Database, error) {
	args := m.Called(ctx, filters, limit, offset)
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

func (m *MockDatabaseRepository) Count(ctx context.Context, filters repository.DatabaseFilters) (int, error) {
	args := m.Called(ctx, filters)
	return args.Int(0), args.Error(1)
}

func (m *MockDatabaseRepository) CountByProject(ctx context.Context, projectID uuid.UUID) (int, error) {
	args := m.Called(ctx, projectID)
	return args.Int(0), args.Error(1)
}

func (m *MockDatabaseRepository) CountByInstanceID(ctx context.Context, instanceID uuid.UUID) (int, error) {
	args := m.Called(ctx, instanceID)
	return args.Int(0), args.Error(1)
}

func (m *MockDatabaseRepository) CountByEnvironmentID(ctx context.Context, environmentID uuid.UUID) (int, error) {
	args := m.Called(ctx, environmentID)
	return args.Int(0), args.Error(1)
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

func (m *MockDatabaseRepository) GetByInstanceAndName(
	ctx context.Context,
	instanceID uuid.UUID,
	databaseName string,
) (*models.Database, error) {
	args := m.Called(ctx, instanceID, databaseName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Database), args.Error(1)
}

func (m *MockDatabaseRepository) ListActiveForSync(ctx context.Context, limit, offset int) ([]*models.Database, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Database), args.Error(1)
}

// MockEnvironmentRepository is a mock implementation of EnvironmentRepository
type MockEnvironmentRepository struct {
	mock.Mock
}

func (m *MockEnvironmentRepository) Create(ctx context.Context, env *models.Environment) error {
	args := m.Called(ctx, env)
	return args.Error(0)
}

func (m *MockEnvironmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Environment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Environment), args.Error(1)
}

func (m *MockEnvironmentRepository) ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]*models.Environment, error) {
	args := m.Called(ctx, projectID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Environment), args.Error(1)
}

func (m *MockEnvironmentRepository) Update(ctx context.Context, env *models.Environment) error {
	args := m.Called(ctx, env)
	return args.Error(0)
}

func (m *MockEnvironmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockEnvironmentRepository) List(
	ctx context.Context,
	filters repository.EnvironmentFilters,
	limit, offset int,
) ([]*models.Environment, error) {
	args := m.Called(ctx, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Environment), args.Error(1)
}

func (m *MockEnvironmentRepository) Count(ctx context.Context, filters repository.EnvironmentFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}
