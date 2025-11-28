package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockApprovalRepository is a mock implementation of ApprovalRepository
type MockApprovalRepository struct {
	mock.Mock
}

func (m *MockApprovalRepository) Create(ctx context.Context, approval *models.Approval) error {
	args := m.Called(ctx, approval)
	return args.Error(0)
}

func (m *MockApprovalRepository) GetByChangePlanID(ctx context.Context, changePlanID uuid.UUID) ([]*models.Approval, error) {
	args := m.Called(ctx, changePlanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Approval), args.Error(1)
}

func (m *MockApprovalRepository) GetByApproverID(ctx context.Context, approverID uuid.UUID) ([]*models.Approval, error) {
	args := m.Called(ctx, approverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Approval), args.Error(1)
}

func (m *MockApprovalRepository) GetHistory(ctx context.Context, changePlanID uuid.UUID) ([]*models.Approval, error) {
	args := m.Called(ctx, changePlanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Approval), args.Error(1)
}

func (m *MockApprovalRepository) DeleteByChangePlanID(ctx context.Context, changePlanID uuid.UUID) error {
	args := m.Called(ctx, changePlanID)
	return args.Error(0)
}

func (m *MockApprovalRepository) GetByChangePlanIDAndApproverID(
	ctx context.Context,
	changePlanID, approverID uuid.UUID,
) (*models.Approval, error) {
	args := m.Called(ctx, changePlanID, approverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Approval), args.Error(1)
}

func (m *MockApprovalRepository) Update(ctx context.Context, approval *models.Approval) error {
	args := m.Called(ctx, approval)
	return args.Error(0)
}

// MockChangePlanRepository is a mock implementation of ChangePlanRepository
type MockChangePlanRepository struct {
	mock.Mock
}

func (m *MockChangePlanRepository) Create(ctx context.Context, changePlan *models.ChangePlan) error {
	args := m.Called(ctx, changePlan)
	return args.Error(0)
}

func (m *MockChangePlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ChangePlan, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ChangePlan), args.Error(1)
}

func (m *MockChangePlanRepository) Update(ctx context.Context, changePlan *models.ChangePlan) error {
	args := m.Called(ctx, changePlan)
	return args.Error(0)
}

func (m *MockChangePlanRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockChangePlanRepository) List(ctx context.Context, limit, offset int) ([]*models.ChangePlan, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*models.ChangePlan), args.Error(1)
}

func (m *MockChangePlanRepository) ListWithFilters(
	ctx context.Context,
	filters repository.ChangePlanListFilters,
) ([]*models.ChangePlan, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]*models.ChangePlan), args.Error(1)
}

func (m *MockChangePlanRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockChangePlanRepository) CountWithFilters(
	ctx context.Context,
	filters repository.ChangePlanListFilters,
) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockChangePlanRepository) GetByStatus(
	ctx context.Context,
	status models.ChangePlanStatus,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	args := m.Called(ctx, status, limit, offset)
	return args.Get(0).([]*models.ChangePlan), args.Error(1)
}

func (m *MockChangePlanRepository) GetByProjectID(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	args := m.Called(ctx, projectID, limit, offset)
	return args.Get(0).([]*models.ChangePlan), args.Error(1)
}

func (m *MockChangePlanRepository) GetByDatabaseID(
	ctx context.Context,
	databaseID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	args := m.Called(ctx, databaseID, limit, offset)
	return args.Get(0).([]*models.ChangePlan), args.Error(1)
}

func (m *MockChangePlanRepository) GetByAuthorID(
	ctx context.Context,
	authorID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	args := m.Called(ctx, authorID, limit, offset)
	return args.Get(0).([]*models.ChangePlan), args.Error(1)
}

// MockPolicyFindingRepository is a mock implementation of PolicyFindingRepository
type MockPolicyFindingRepository struct {
	mock.Mock
}

func (m *MockPolicyFindingRepository) Create(ctx context.Context, finding *models.PolicyFinding) error {
	args := m.Called(ctx, finding)
	return args.Error(0)
}

func (m *MockPolicyFindingRepository) CreateBatch(ctx context.Context, findings []*models.PolicyFinding) error {
	args := m.Called(ctx, findings)
	return args.Error(0)
}

func (m *MockPolicyFindingRepository) GetByChangePlanID(
	ctx context.Context,
	changePlanID uuid.UUID,
) ([]*models.PolicyFinding, error) {
	args := m.Called(ctx, changePlanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PolicyFinding), args.Error(1)
}

func (m *MockPolicyFindingRepository) DeleteByChangePlanID(ctx context.Context, changePlanID uuid.UUID) error {
	args := m.Called(ctx, changePlanID)
	return args.Error(0)
}

func (m *MockPolicyFindingRepository) CountBySeverity(ctx context.Context, changePlanID uuid.UUID) (map[string]int, error) {
	args := m.Called(ctx, changePlanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int), args.Error(1)
}

// Helper function to create a context with user info
func createContextWithUser(userID uuid.UUID, orgRole rbac.OrganizationRole) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, rbac.ContextKeyUserID, userID.String())
	ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationID, "default")
	ctx = context.WithValue(ctx, rbac.ContextKeyOrganizationRole, orgRole)
	return ctx
}

func TestApprovalService_Approve(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("successful approval with org admin", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()
		comments := "Looks good"

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusOpen,
			UpdatedAt: time.Now(),
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationAdmin)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)
		mockPolicyFindingRepo.On("GetByChangePlanID", ctxWithUser, changePlanID).Return([]*models.PolicyFinding{}, nil)
		mockApprovalRepo.On("GetByChangePlanIDAndApproverID", ctxWithUser, changePlanID, approverID).
			Return(nil, repository.ErrNotFound)
		mockApprovalRepo.On("Create", ctxWithUser, mock.AnythingOfType("*models.Approval")).Return(nil)
		mockChangePlanRepo.On("Update", ctxWithUser, mock.MatchedBy(func(cp *models.ChangePlan) bool {
			return cp.ID == changePlanID && cp.Status == models.ChangePlanStatusApproved
		})).Return(nil)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Approve(ctxWithUser, changePlanID, approverID, &comments)

		assert.NoError(t, err)
		mockApprovalRepo.AssertExpectations(t)
		mockChangePlanRepo.AssertExpectations(t)
		mockPolicyFindingRepo.AssertExpectations(t)
	})

	t.Run("successful approval with project approver", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()
		comments := "Approved"

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusOpen,
			UpdatedAt: time.Now(),
		}

		projectMember := &models.ProjectMember{
			ProjectID: projectID,
			UserID:    approverID,
			Role:      rbac.ProjectApprover,
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationDeveloper)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)
		mockProjectMemberRepo.On("GetMemberRole", ctxWithUser, projectID, approverID).Return(projectMember, nil)
		mockPolicyFindingRepo.On("GetByChangePlanID", ctxWithUser, changePlanID).Return([]*models.PolicyFinding{}, nil)
		mockApprovalRepo.On("GetByChangePlanIDAndApproverID", ctxWithUser, changePlanID, approverID).
			Return(nil, repository.ErrNotFound)
		mockApprovalRepo.On("Create", ctxWithUser, mock.AnythingOfType("*models.Approval")).Return(nil)
		mockChangePlanRepo.On("Update", ctxWithUser, mock.MatchedBy(func(cp *models.ChangePlan) bool {
			return cp.ID == changePlanID && cp.Status == models.ChangePlanStatusApproved
		})).Return(nil)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Approve(ctxWithUser, changePlanID, approverID, &comments)

		assert.NoError(t, err)
		mockApprovalRepo.AssertExpectations(t)
		mockChangePlanRepo.AssertExpectations(t)
		mockPolicyFindingRepo.AssertExpectations(t)
		mockProjectMemberRepo.AssertExpectations(t)
	})

	t.Run("fails when change plan not found", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationAdmin)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(nil, repository.ErrNotFound)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Approve(ctxWithUser, changePlanID, approverID, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "change plan not found")
		mockChangePlanRepo.AssertExpectations(t)
	})

	t.Run("fails when change plan not in Open status", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusApproved,
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationAdmin)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Approve(ctxWithUser, changePlanID, approverID, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already approved")
		mockChangePlanRepo.AssertExpectations(t)
	})

	t.Run("fails when approver lacks permission", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusOpen,
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationDeveloper)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)
		mockProjectMemberRepo.On("GetMemberRole", ctxWithUser, projectID, approverID).Return(nil, repository.ErrNotFound)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Approve(ctxWithUser, changePlanID, approverID, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission")
		mockChangePlanRepo.AssertExpectations(t)
		mockProjectMemberRepo.AssertExpectations(t)
	})

	t.Run("fails when SQL Review errors exist", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusOpen,
		}

		errorFindings := []*models.PolicyFinding{
			{
				ID:           uuid.New(),
				ChangePlanID: changePlanID,
				Severity:     "Error",
				RuleName:     "SyntaxError",
				Message:      "SQL syntax error",
			},
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationAdmin)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)
		mockPolicyFindingRepo.On("GetByChangePlanID", ctxWithUser, changePlanID).Return(errorFindings, nil)
		mockApprovalRepo.On("GetByChangePlanIDAndApproverID", ctxWithUser, changePlanID, approverID).
			Return(nil, repository.ErrNotFound)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Approve(ctxWithUser, changePlanID, approverID, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SQL Review errors")
		mockChangePlanRepo.AssertExpectations(t)
		mockPolicyFindingRepo.AssertExpectations(t)
	})

	t.Run("allows warnings but not errors", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusOpen,
			UpdatedAt: time.Now(),
		}

		warningFindings := []*models.PolicyFinding{
			{
				ID:           uuid.New(),
				ChangePlanID: changePlanID,
				Severity:     "Warning",
				RuleName:     "PerformanceWarning",
				Message:      "Slow query detected",
			},
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationAdmin)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)
		mockPolicyFindingRepo.On("GetByChangePlanID", ctxWithUser, changePlanID).Return(warningFindings, nil)
		mockApprovalRepo.On("GetByChangePlanIDAndApproverID", ctxWithUser, changePlanID, approverID).
			Return(nil, repository.ErrNotFound)
		mockApprovalRepo.On("Create", ctxWithUser, mock.AnythingOfType("*models.Approval")).Return(nil)
		mockChangePlanRepo.On("Update", ctxWithUser, mock.MatchedBy(func(cp *models.ChangePlan) bool {
			return cp.ID == changePlanID && cp.Status == models.ChangePlanStatusApproved
		})).Return(nil)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Approve(ctxWithUser, changePlanID, approverID, nil)

		assert.NoError(t, err)
		mockApprovalRepo.AssertExpectations(t)
		mockChangePlanRepo.AssertExpectations(t)
		mockPolicyFindingRepo.AssertExpectations(t)
	})
}

func TestApprovalService_Close(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("successful close with org admin", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()
		comments := "Needs work"

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusOpen,
			UpdatedAt: time.Now(),
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationAdmin)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)
		mockApprovalRepo.On("Create", ctxWithUser, mock.MatchedBy(func(a *models.Approval) bool {
			return a.ChangePlanID == changePlanID && a.Status == models.ApprovalStatusRejected
		})).Return(nil)
		mockChangePlanRepo.On("Update", ctxWithUser, mock.MatchedBy(func(cp *models.ChangePlan) bool {
			return cp.ID == changePlanID && cp.Status == models.ChangePlanStatusClosed
		})).Return(nil)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Close(ctxWithUser, changePlanID, approverID, &comments)

		assert.NoError(t, err)
		mockApprovalRepo.AssertExpectations(t)
		mockChangePlanRepo.AssertExpectations(t)
	})

	t.Run("fails when change plan not found", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationAdmin)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(nil, repository.ErrNotFound)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Close(ctxWithUser, changePlanID, approverID, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "change plan not found")
		mockChangePlanRepo.AssertExpectations(t)
	})

	t.Run("fails when change plan not in Open status", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusClosed,
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationAdmin)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Close(ctxWithUser, changePlanID, approverID, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already closed")
		mockChangePlanRepo.AssertExpectations(t)
	})

	t.Run("successful close with Failed status", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()
		comments := "Closing failed change plan"

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusFailed,
			UpdatedAt: time.Now(),
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationAdmin)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)
		mockApprovalRepo.On("Create", ctxWithUser, mock.MatchedBy(func(a *models.Approval) bool {
			return a.ChangePlanID == changePlanID && a.Status == models.ApprovalStatusRejected
		})).Return(nil)
		mockChangePlanRepo.On("Update", ctxWithUser, mock.MatchedBy(func(cp *models.ChangePlan) bool {
			return cp.ID == changePlanID && cp.Status == models.ChangePlanStatusClosed
		})).Return(nil)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Close(ctxWithUser, changePlanID, approverID, &comments)

		assert.NoError(t, err)
		mockApprovalRepo.AssertExpectations(t)
		mockChangePlanRepo.AssertExpectations(t)
	})

	t.Run("successful close with project approver", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()
		comments := "Rejected"

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusOpen,
			UpdatedAt: time.Now(),
		}

		projectMember := &models.ProjectMember{
			ProjectID: projectID,
			UserID:    approverID,
			Role:      rbac.ProjectApprover,
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationDeveloper)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)
		mockProjectMemberRepo.On("GetMemberRole", ctxWithUser, projectID, approverID).Return(projectMember, nil)
		mockApprovalRepo.On("Create", ctxWithUser, mock.MatchedBy(func(a *models.Approval) bool {
			return a.ChangePlanID == changePlanID && a.Status == models.ApprovalStatusRejected
		})).Return(nil)
		mockChangePlanRepo.On("Update", ctxWithUser, mock.MatchedBy(func(cp *models.ChangePlan) bool {
			return cp.ID == changePlanID && cp.Status == models.ChangePlanStatusClosed
		})).Return(nil)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Close(ctxWithUser, changePlanID, approverID, &comments)

		assert.NoError(t, err)
		mockApprovalRepo.AssertExpectations(t)
		mockChangePlanRepo.AssertExpectations(t)
		mockProjectMemberRepo.AssertExpectations(t)
	})

	t.Run("fails when approver lacks permission", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusOpen,
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationDeveloper)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)
		mockProjectMemberRepo.On("GetMemberRole", ctxWithUser, projectID, approverID).Return(nil, repository.ErrNotFound)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Close(ctxWithUser, changePlanID, approverID, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission")
		mockChangePlanRepo.AssertExpectations(t)
		mockProjectMemberRepo.AssertExpectations(t)
	})

	t.Run("handles approval repository error", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()
		comments := "Rejected"

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusOpen,
			UpdatedAt: time.Now(),
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationAdmin)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)
		mockApprovalRepo.On("Create", ctxWithUser, mock.MatchedBy(func(a *models.Approval) bool {
			return a.ChangePlanID == changePlanID && a.Status == models.ApprovalStatusRejected
		})).Return(errors.New("database error"))

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Close(ctxWithUser, changePlanID, approverID, &comments)

		assert.Error(t, err)
		mockApprovalRepo.AssertExpectations(t)
		mockChangePlanRepo.AssertExpectations(t)
	})
}

func TestApprovalService_GetApprovalHistory(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ctx := context.Background()

	t.Run("successful retrieval", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		approvals := []*models.Approval{
			{
				ID:           uuid.New(),
				ChangePlanID: changePlanID,
				ApproverID:   approverID,
				Status:       models.ApprovalStatusApproved,
				CreatedAt:    time.Now(),
			},
		}

		mockApprovalRepo.On("GetHistory", ctx, changePlanID).Return(approvals, nil)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		result, err := service.GetApprovalHistory(ctx, changePlanID)

		assert.NoError(t, err)
		assert.Equal(t, approvals, result)
		mockApprovalRepo.AssertExpectations(t)
	})

	t.Run("returns empty list when no approvals", func(t *testing.T) {
		changePlanID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		mockApprovalRepo.On("GetHistory", ctx, changePlanID).Return([]*models.Approval{}, nil)

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		result, err := service.GetApprovalHistory(ctx, changePlanID)

		assert.NoError(t, err)
		assert.Empty(t, result)
		mockApprovalRepo.AssertExpectations(t)
	})

	t.Run("handles repository error", func(t *testing.T) {
		changePlanID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		mockApprovalRepo.On("GetHistory", ctx, changePlanID).Return(nil, errors.New("database error"))

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		result, err := service.GetApprovalHistory(ctx, changePlanID)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockApprovalRepo.AssertExpectations(t)
	})
}

func TestApprovalService_ErrorHandling(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("handles repository errors", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationAdmin)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(nil, errors.New("database error"))

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Approve(ctxWithUser, changePlanID, approverID, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get change plan")
		mockChangePlanRepo.AssertExpectations(t)
	})

	t.Run("handles approval creation error", func(t *testing.T) {
		changePlanID := uuid.New()
		approverID := uuid.New()
		projectID := uuid.New()

		mockApprovalRepo := new(MockApprovalRepository)
		mockChangePlanRepo := new(MockChangePlanRepository)
		mockPolicyFindingRepo := new(MockPolicyFindingRepository)
		mockProjectMemberRepo := new(MockProjectMemberRepository)
		rbacChecker := rbac.NewCheckerService()

		changePlan := &models.ChangePlan{
			ID:        changePlanID,
			ProjectID: projectID,
			Status:    models.ChangePlanStatusOpen,
		}

		ctxWithUser := createContextWithUser(approverID, rbac.OrganizationAdmin)

		mockChangePlanRepo.On("GetByID", ctxWithUser, changePlanID).Return(changePlan, nil)
		mockPolicyFindingRepo.On("GetByChangePlanID", ctxWithUser, changePlanID).Return([]*models.PolicyFinding{}, nil)
		mockApprovalRepo.On("GetByChangePlanIDAndApproverID", ctxWithUser, changePlanID, approverID).
			Return(nil, repository.ErrNotFound)
		mockApprovalRepo.On("Create", ctxWithUser, mock.AnythingOfType("*models.Approval")).Return(errors.New("database error"))

		service := NewApprovalService(
			mockApprovalRepo,
			mockChangePlanRepo,
			mockPolicyFindingRepo,
			mockProjectMemberRepo,
			new(MockDatabaseRepository),
			new(MockEnvironmentRepository),
			rbacChecker,
			logger,
		)

		err := service.Approve(ctxWithUser, changePlanID, approverID, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create approval record")
		mockApprovalRepo.AssertExpectations(t)
		mockChangePlanRepo.AssertExpectations(t)
		mockPolicyFindingRepo.AssertExpectations(t)
	})
}
