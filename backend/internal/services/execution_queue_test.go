package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewExecutionQueueService(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()

	service := NewExecutionQueueService(mockQueueRepo, logger)

	assert.NotNil(t, service)
	assert.Equal(t, mockQueueRepo, service.queueRepo)
	assert.Equal(t, logger, service.logger)
}

func TestExecutionQueueService_Enqueue(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	changePlanID := uuid.New()
	databaseID := uuid.New()
	executorID := uuid.New()

	// Mock Enqueue - it will internally call GetPendingCount, but we're testing the service layer
	// The repository implementation details are tested separately
	mockQueueRepo.On("Enqueue", ctx, mock.MatchedBy(func(item *models.ExecutionQueue) bool {
		return item.ChangePlanID == changePlanID &&
			item.TargetDatabaseID == databaseID &&
			item.ExecutorID == executorID &&
			item.Status == models.QueueStatusPending &&
			item.Priority == 0
	})).Return(nil).Run(func(args mock.Arguments) {
		item := args.Get(1).(*models.ExecutionQueue)
		item.ID = uuid.New()
		item.Position = 0
	})

	queueItem, err := service.Enqueue(ctx, changePlanID, databaseID, executorID)

	require.NoError(t, err)
	assert.NotNil(t, queueItem)
	assert.Equal(t, changePlanID, queueItem.ChangePlanID)
	assert.Equal(t, databaseID, queueItem.TargetDatabaseID)
	assert.Equal(t, executorID, queueItem.ExecutorID)
	assert.Equal(t, models.QueueStatusPending, queueItem.Status)
	assert.Equal(t, 0, queueItem.Position)

	mockQueueRepo.AssertExpectations(t)
}

func TestExecutionQueueService_GetQueueStatus(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	changePlanID := uuid.New()

	expectedQueueItem := &models.ExecutionQueue{
		ID:               uuid.New(),
		ChangePlanID:     changePlanID,
		TargetDatabaseID: uuid.New(),
		ExecutorID:       uuid.New(),
		Status:           models.QueueStatusPending,
		Position:         2,
		Priority:         0,
		QueuedAt:         time.Now(),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	mockQueueRepo.On("GetByChangePlanID", ctx, changePlanID).Return([]*models.ExecutionQueue{expectedQueueItem}, nil)

	queueStatus, err := service.GetQueueStatus(ctx, changePlanID)

	require.NoError(t, err)
	assert.NotNil(t, queueStatus)
	assert.Equal(t, expectedQueueItem.ID, queueStatus.ID)
	assert.Equal(t, expectedQueueItem.Status, queueStatus.Status)
	assert.Equal(t, expectedQueueItem.Position, queueStatus.Position)

	mockQueueRepo.AssertExpectations(t)
}

func TestExecutionQueueService_GetQueueStatus_NotFound(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	changePlanID := uuid.New()

	mockQueueRepo.On("GetByChangePlanID", ctx, changePlanID).Return([]*models.ExecutionQueue{}, nil)

	queueStatus, err := service.GetQueueStatus(ctx, changePlanID)

	assert.Error(t, err)
	assert.Nil(t, queueStatus)
	assert.Equal(t, repository.ErrNotFound, err)

	mockQueueRepo.AssertExpectations(t)
}

func TestExecutionQueueService_GetQueuePosition(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	changePlanID := uuid.New()

	queueItem := &models.ExecutionQueue{
		ID:           uuid.New(),
		ChangePlanID: changePlanID,
		Status:       models.QueueStatusPending,
		Position:     3,
	}

	mockQueueRepo.On("GetByChangePlanID", ctx, changePlanID).Return([]*models.ExecutionQueue{queueItem}, nil)

	position, err := service.GetQueuePosition(ctx, changePlanID)

	require.NoError(t, err)
	assert.Equal(t, 3, position)

	mockQueueRepo.AssertExpectations(t)
}

func TestExecutionQueueService_GetQueueItemsForDatabase(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	databaseID := uuid.New()

	expectedItems := []*models.ExecutionQueue{
		{ID: uuid.New(), TargetDatabaseID: databaseID, Status: models.QueueStatusPending, Position: 0},
		{ID: uuid.New(), TargetDatabaseID: databaseID, Status: models.QueueStatusPending, Position: 1},
	}

	mockQueueRepo.On("GetByDatabaseID", ctx, databaseID).Return(expectedItems, nil)

	items, err := service.GetQueueItemsForDatabase(ctx, databaseID)

	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, expectedItems[0].ID, items[0].ID)
	assert.Equal(t, expectedItems[1].ID, items[1].ID)

	mockQueueRepo.AssertExpectations(t)
}

func TestExecutionQueueService_GetPendingCount(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	databaseID := uuid.New()

	mockQueueRepo.On("GetPendingCount", ctx, databaseID).Return(5, nil)

	count, err := service.GetPendingCount(ctx, databaseID)

	require.NoError(t, err)
	assert.Equal(t, 5, count)

	mockQueueRepo.AssertExpectations(t)
}

// TestExecutionQueueService_Enqueue_Error tests Enqueue repository error
func TestExecutionQueueService_Enqueue_Error(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	changePlanID := uuid.New()
	databaseID := uuid.New()
	executorID := uuid.New()

	mockQueueRepo.On("Enqueue", ctx, mock.Anything).Return(errors.New("database error"))

	queueItem, err := service.Enqueue(ctx, changePlanID, databaseID, executorID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to enqueue change plan")
	assert.Nil(t, queueItem)

	mockQueueRepo.AssertExpectations(t)
}

// TestExecutionQueueService_GetQueuePosition_NoPendingItem tests GetQueuePosition when no pending item found
func TestExecutionQueueService_GetQueuePosition_NoPendingItem(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	changePlanID := uuid.New()

	// Return items but none are pending
	queueItems := []*models.ExecutionQueue{
		{
			ID:           uuid.New(),
			ChangePlanID: changePlanID,
			Status:       models.QueueStatusCompleted,
			Position:     0,
		},
		{
			ID:           uuid.New(),
			ChangePlanID: changePlanID,
			Status:       models.QueueStatusFailed,
			Position:     1,
		},
	}

	mockQueueRepo.On("GetByChangePlanID", ctx, changePlanID).Return(queueItems, nil)

	position, err := service.GetQueuePosition(ctx, changePlanID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no pending queue item found")
	assert.Equal(t, 0, position)

	mockQueueRepo.AssertExpectations(t)
}

// TestExecutionQueueService_GetQueuePosition_RepositoryError tests GetQueuePosition repository error
func TestExecutionQueueService_GetQueuePosition_RepositoryError(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	changePlanID := uuid.New()

	mockQueueRepo.On("GetByChangePlanID", ctx, changePlanID).Return(nil, errors.New("database error"))

	position, err := service.GetQueuePosition(ctx, changePlanID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get queue items")
	assert.Equal(t, 0, position)

	mockQueueRepo.AssertExpectations(t)
}

// TestExecutionQueueService_GetQueuePosition_MultipleItems tests GetQueuePosition with multiple items (finds first pending)
func TestExecutionQueueService_GetQueuePosition_MultipleItems(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	changePlanID := uuid.New()

	queueItems := []*models.ExecutionQueue{
		{
			ID:           uuid.New(),
			ChangePlanID: changePlanID,
			Status:       models.QueueStatusCompleted,
			Position:     0,
		},
		{
			ID:           uuid.New(),
			ChangePlanID: changePlanID,
			Status:       models.QueueStatusPending,
			Position:     2,
		},
		{
			ID:           uuid.New(),
			ChangePlanID: changePlanID,
			Status:       models.QueueStatusPending,
			Position:     3,
		},
	}

	mockQueueRepo.On("GetByChangePlanID", ctx, changePlanID).Return(queueItems, nil)

	position, err := service.GetQueuePosition(ctx, changePlanID)

	require.NoError(t, err)
	assert.Equal(t, 2, position) // Should return first pending item's position

	mockQueueRepo.AssertExpectations(t)
}

// TestExecutionQueueService_GetQueueStatus_RepositoryError tests GetQueueStatus repository error
func TestExecutionQueueService_GetQueueStatus_RepositoryError(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	changePlanID := uuid.New()

	mockQueueRepo.On("GetByChangePlanID", ctx, changePlanID).Return(nil, errors.New("database error"))

	queueStatus, err := service.GetQueueStatus(ctx, changePlanID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get queue items")
	assert.Nil(t, queueStatus)

	mockQueueRepo.AssertExpectations(t)
}

// TestExecutionQueueService_GetQueueItemsForDatabase_Error tests GetQueueItemsForDatabase error
func TestExecutionQueueService_GetQueueItemsForDatabase_Error(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	databaseID := uuid.New()

	mockQueueRepo.On("GetByDatabaseID", ctx, databaseID).Return(nil, errors.New("database error"))

	items, err := service.GetQueueItemsForDatabase(ctx, databaseID)

	assert.Error(t, err)
	assert.Nil(t, items)

	mockQueueRepo.AssertExpectations(t)
}

// TestExecutionQueueService_GetPendingCount_Error tests GetPendingCount error
func TestExecutionQueueService_GetPendingCount_Error(t *testing.T) {
	mockQueueRepo := new(MockExecutionQueueRepository)
	logger := zap.NewNop()
	service := NewExecutionQueueService(mockQueueRepo, logger)

	ctx := context.Background()
	databaseID := uuid.New()

	mockQueueRepo.On("GetPendingCount", ctx, databaseID).Return(0, errors.New("database error"))

	count, err := service.GetPendingCount(ctx, databaseID)

	assert.Error(t, err)
	assert.Equal(t, 0, count)

	mockQueueRepo.AssertExpectations(t)
}

// MockExecutionQueueRepository is a mock implementation of ExecutionQueueRepository
type MockExecutionQueueRepository struct {
	mock.Mock
}

func (m *MockExecutionQueueRepository) Enqueue(ctx context.Context, queueItem *models.ExecutionQueue) error {
	args := m.Called(ctx, queueItem)
	return args.Error(0)
}

func (m *MockExecutionQueueRepository) GetNextPending(ctx context.Context, databaseID uuid.UUID) (*models.ExecutionQueue, error) {
	args := m.Called(ctx, databaseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExecutionQueue), args.Error(1)
}

func (m *MockExecutionQueueRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ExecutionQueue, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExecutionQueue), args.Error(1)
}

func (m *MockExecutionQueueRepository) GetByChangePlanID(
	ctx context.Context,
	changePlanID uuid.UUID,
) ([]*models.ExecutionQueue, error) {
	args := m.Called(ctx, changePlanID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ExecutionQueue), args.Error(1)
}

func (m *MockExecutionQueueRepository) GetByDatabaseID(
	ctx context.Context,
	databaseID uuid.UUID,
) ([]*models.ExecutionQueue, error) {
	args := m.Called(ctx, databaseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ExecutionQueue), args.Error(1)
}

func (m *MockExecutionQueueRepository) GetPendingCount(ctx context.Context, databaseID uuid.UUID) (int, error) {
	args := m.Called(ctx, databaseID)
	return args.Int(0), args.Error(1)
}

func (m *MockExecutionQueueRepository) GetPosition(ctx context.Context, queueItemID uuid.UUID) (int, error) {
	args := m.Called(ctx, queueItemID)
	return args.Int(0), args.Error(1)
}

func (m *MockExecutionQueueRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.QueueStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockExecutionQueueRepository) Update(ctx context.Context, queueItem *models.ExecutionQueue) error {
	args := m.Called(ctx, queueItem)
	return args.Error(0)
}

func (m *MockExecutionQueueRepository) RecalculatePositions(ctx context.Context, databaseID uuid.UUID) error {
	args := m.Called(ctx, databaseID)
	return args.Error(0)
}

func (m *MockExecutionQueueRepository) MarkInProgress(ctx context.Context, databaseID uuid.UUID) (*models.ExecutionQueue, error) {
	args := m.Called(ctx, databaseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ExecutionQueue), args.Error(1)
}

func (m *MockExecutionQueueRepository) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockExecutionQueueRepository) MarkFailed(ctx context.Context, id uuid.UUID, errorMessage string) error {
	args := m.Called(ctx, id, errorMessage)
	return args.Error(0)
}

func (m *MockExecutionQueueRepository) GetDatabasesWithPending(ctx context.Context) ([]uuid.UUID, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}
