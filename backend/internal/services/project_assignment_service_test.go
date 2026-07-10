package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestAssignUserToExistingProjects_HappyPath(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := zap.NewNop()

	userID := uuid.New()
	systemUserID := uuid.New()

	systemUser := &models.User{
		ID:               systemUserID,
		Email:            "system@deeb1.dev",
		OrganizationRole: rbac.OrganizationAdmin,
		IsActive:         true,
	}

	projects := []*models.Project{
		{ID: uuid.New(), Name: "Project 1", CreatedByID: uuid.New()},
		{ID: uuid.New(), Name: "Project 2", CreatedByID: uuid.New()},
		{ID: uuid.New(), Name: "Project 3", CreatedByID: uuid.New()},
	}

	mockUserRepo := new(MockUserRepository)
	mockProjectRepo := new(MockProjectRepository)
	mockProjectMemberRepo := new(MockProjectMemberRepository)

	mockUserRepo.On("GetByEmail", ctx, "system@deeb1.dev").Return(systemUser, nil)
	mockProjectRepo.On("List", ctx, mock.Anything, mock.Anything, 0).Return(projects, nil)
	mockProjectMemberRepo.On("AddMembersBatch", ctx, mock.MatchedBy(func(members []*models.ProjectMember) bool {
		return len(members) == 3
	})).Return(nil)

	service := NewProjectAssignmentService(mockProjectRepo, mockProjectMemberRepo, mockUserRepo, true, logger)

	// Execute
	count, err := service.AssignUserToExistingProjects(ctx, userID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 3, count)
	mockUserRepo.AssertExpectations(t)
	mockProjectRepo.AssertExpectations(t)
	mockProjectMemberRepo.AssertExpectations(t)
}

func TestAssignUserToExistingProjects_NoProjects(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := zap.NewNop()

	userID := uuid.New()
	systemUserID := uuid.New()

	systemUser := &models.User{
		ID:               systemUserID,
		Email:            "system@deeb1.dev",
		OrganizationRole: rbac.OrganizationAdmin,
		IsActive:         true,
	}

	mockUserRepo := new(MockUserRepository)
	mockProjectRepo := new(MockProjectRepository)
	mockProjectMemberRepo := new(MockProjectMemberRepository)

	mockUserRepo.On("GetByEmail", ctx, "system@deeb1.dev").Return(systemUser, nil)
	mockProjectRepo.On("List", ctx, mock.Anything, mock.Anything, 0).Return([]*models.Project{}, nil)

	service := NewProjectAssignmentService(mockProjectRepo, mockProjectMemberRepo, mockUserRepo, true, logger)

	// Execute
	count, err := service.AssignUserToExistingProjects(ctx, userID)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	mockUserRepo.AssertExpectations(t)
	mockProjectRepo.AssertExpectations(t)
}

func TestAssignUserToExistingProjects_SystemUserNotFound(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := zap.NewNop()

	userID := uuid.New()

	mockUserRepo := new(MockUserRepository)
	mockProjectRepo := new(MockProjectRepository)
	mockProjectMemberRepo := new(MockProjectMemberRepository)

	mockUserRepo.On("GetByEmail", ctx, "system@deeb1.dev").Return(nil, repository.ErrNotFound)

	service := NewProjectAssignmentService(mockProjectRepo, mockProjectMemberRepo, mockUserRepo, true, logger)

	// Execute
	count, err := service.AssignUserToExistingProjects(ctx, userID)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrSystemUserNotFound, err)
	assert.Equal(t, 0, count)
	mockUserRepo.AssertExpectations(t)
}

func TestAssignUserToExistingProjects_UniqueConstraintViolation(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := zap.NewNop()

	userID := uuid.New()
	systemUserID := uuid.New()

	systemUser := &models.User{
		ID:               systemUserID,
		Email:            "system@deeb1.dev",
		OrganizationRole: rbac.OrganizationAdmin,
		IsActive:         true,
	}

	projects := []*models.Project{
		{ID: uuid.New(), Name: "Project 1", CreatedByID: uuid.New()},
		{ID: uuid.New(), Name: "Project 2", CreatedByID: uuid.New()},
	}

	mockUserRepo := new(MockUserRepository)
	mockProjectRepo := new(MockProjectRepository)
	mockProjectMemberRepo := new(MockProjectMemberRepository)

	mockUserRepo.On("GetByEmail", ctx, "system@deeb1.dev").Return(systemUser, nil)
	mockProjectRepo.On("List", ctx, mock.Anything, mock.Anything, 0).Return(projects, nil)

	// Batch call returns unique constraint violation (transaction will rollback)
	uniqueErr := &pq.Error{Code: "23505", Constraint: "unique_project_user"}
	mockProjectMemberRepo.On("AddMembersBatch", ctx, mock.AnythingOfType("[]*models.ProjectMember")).Return(uniqueErr)

	service := NewProjectAssignmentService(mockProjectRepo, mockProjectMemberRepo, mockUserRepo, true, logger)

	// Execute
	count, err := service.AssignUserToExistingProjects(ctx, userID)

	// Assert - should return error with count 0 (transaction rolled back)
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	mockUserRepo.AssertExpectations(t)
	mockProjectRepo.AssertExpectations(t)
	mockProjectMemberRepo.AssertExpectations(t)
}

func TestAssignUserToExistingProjects_PartialFailure(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := zap.NewNop()

	userID := uuid.New()
	systemUserID := uuid.New()

	systemUser := &models.User{
		ID:               systemUserID,
		Email:            "system@deeb1.dev",
		OrganizationRole: rbac.OrganizationAdmin,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	projects := []*models.Project{
		{ID: uuid.New(), Name: "Project 1", CreatedByID: uuid.New()},
		{ID: uuid.New(), Name: "Project 2", CreatedByID: uuid.New()},
		{ID: uuid.New(), Name: "Project 3", CreatedByID: uuid.New()},
	}

	mockUserRepo := new(MockUserRepository)
	mockProjectRepo := new(MockProjectRepository)
	mockProjectMemberRepo := new(MockProjectMemberRepository)

	mockUserRepo.On("GetByEmail", ctx, "system@deeb1.dev").Return(systemUser, nil)
	mockProjectRepo.On("List", ctx, mock.Anything, mock.Anything, 0).Return(projects, nil)

	// Batch call fails with generic error (transaction will rollback, all or nothing)
	mockProjectMemberRepo.On("AddMembersBatch", ctx, mock.AnythingOfType("[]*models.ProjectMember")).Return(assert.AnError)

	service := NewProjectAssignmentService(mockProjectRepo, mockProjectMemberRepo, mockUserRepo, true, logger)

	// Execute
	count, err := service.AssignUserToExistingProjects(ctx, userID)

	// Assert - should return error with count 0 (transaction rolled back - atomic behavior)
	assert.Error(t, err)
	assert.Equal(t, 0, count)
	mockUserRepo.AssertExpectations(t)
	mockProjectRepo.AssertExpectations(t)
	mockProjectMemberRepo.AssertExpectations(t)
}

func TestAssignUserToExistingProjects_FeatureDisabled(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := zap.NewNop()

	userID := uuid.New()

	mockUserRepo := new(MockUserRepository)
	mockProjectRepo := new(MockProjectRepository)
	mockProjectMemberRepo := new(MockProjectMemberRepository)

	// Service with feature DISABLED
	service := NewProjectAssignmentService(mockProjectRepo, mockProjectMemberRepo, mockUserRepo, false, logger)

	// Execute
	count, err := service.AssignUserToExistingProjects(ctx, userID)

	// Assert - should return immediately without calling any repos
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	// No repository methods should be called when feature is disabled
	mockUserRepo.AssertNotCalled(t, "GetByEmail")
	mockProjectRepo.AssertNotCalled(t, "List")
	mockProjectMemberRepo.AssertNotCalled(t, "AddMembersBatch")
}
