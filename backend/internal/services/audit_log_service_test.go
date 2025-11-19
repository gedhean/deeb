package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockAuditLogRepository is a mock implementation of AuditLogRepository
type MockAuditLogRepository struct {
	mock.Mock
}

func (m *MockAuditLogRepository) Create(ctx context.Context, auditLog *models.AuditLog) error {
	args := m.Called(ctx, auditLog)
	return args.Error(0)
}

func (m *MockAuditLogRepository) List(ctx context.Context, limit, offset int) ([]*models.AuditLog, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AuditLog), args.Error(1)
}

func (m *MockAuditLogRepository) GetByUserID(
	ctx context.Context,
	userID *uuid.UUID,
	limit, offset int,
) ([]*models.AuditLog, error) {
	args := m.Called(ctx, userID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AuditLog), args.Error(1)
}

func (m *MockAuditLogRepository) GetByAction(ctx context.Context, action string, limit, offset int) ([]*models.AuditLog, error) {
	args := m.Called(ctx, action, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AuditLog), args.Error(1)
}

func (m *MockAuditLogRepository) GetByTargetType(
	ctx context.Context,
	resourceType string,
	limit, offset int,
) ([]*models.AuditLog, error) {
	args := m.Called(ctx, resourceType, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AuditLog), args.Error(1)
}

func (m *MockAuditLogRepository) Filter(ctx context.Context, filters repository.AuditLogListFilters) ([]*models.AuditLog, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AuditLog), args.Error(1)
}

func (m *MockAuditLogRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAuditLogRepository) CountWithFilters(ctx context.Context, filters repository.AuditLogListFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

func TestAuditLogService_Log(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionCreateChangePlan
	resourceType := ResourceTypeChangePlan

	details := json.RawMessage(`{"title": "Test Change Plan"}`)

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.UserID != nil && *log.UserID == userID &&
			log.Action == action &&
			log.ResourceType == resourceType &&
			log.ResourceID == resourceID &&
			string(log.Details) == string(details)
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err := service.Log(ctx, &userID, action, resourceType, resourceID, details)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithHTTP(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionLogin
	resourceType := ResourceTypeUser

	details := json.RawMessage(`{"result": "success"}`)

	// Create HTTP request with IP address
	req, err := http.NewRequest("POST", "/api/v1/login", nil)
	require.NoError(t, err)
	req.RemoteAddr = "192.168.1.1:54321"
	req.Header.Set("User-Agent", "Mozilla/5.0")

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.UserID != nil && *log.UserID == userID &&
			log.Action == action &&
			log.ResourceType == resourceType &&
			log.ResourceID == resourceID &&
			log.IPAddress != nil &&
			log.UserAgent != nil &&
			*log.UserAgent == "Mozilla/5.0"
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err = service.LogWithHTTP(ctx, &userID, action, resourceType, resourceID, req, details)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithHTTP_XForwardedFor(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionLogin
	resourceType := ResourceTypeUser

	details := json.RawMessage(`{"result": "success"}`)

	// Create HTTP request with X-Forwarded-For header
	req, err := http.NewRequest("POST", "/api/v1/login", nil)
	require.NoError(t, err)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")
	req.Header.Set("User-Agent", "TestAgent")

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.UserID != nil && *log.UserID == userID &&
			log.IPAddress != nil &&
			log.IPAddress.String() == "10.0.0.1"
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err = service.LogWithHTTP(ctx, &userID, action, resourceType, resourceID, req, details)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogStateTransition(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	resourceType := ResourceTypeChangePlan
	fromStatus := "open"
	toStatus := "approved"

	additionalDetails := json.RawMessage(`{"comment": "Looks good"}`)

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		if log.Action != ActionApproveChangePlan {
			return false
		}
		if log.ResourceType != resourceType {
			return false
		}

		// Verify details contain state transition info
		var detailsMap map[string]interface{}
		if err := json.Unmarshal(log.Details, &detailsMap); err != nil {
			return false
		}

		return detailsMap["previous_status"] == fromStatus &&
			detailsMap["status"] == toStatus &&
			detailsMap["comment"] == "Looks good"
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err := service.LogStateTransition(ctx, &userID, resourceType, resourceID, fromStatus, toStatus, additionalDetails)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithResult_Success(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionExecuteChangePlan
	resourceType := ResourceTypeChangePlan
	result := "success"

	additionalDetails := json.RawMessage(`{"duration_ms": 1234}`)

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		var detailsMap map[string]interface{}
		if err := json.Unmarshal(log.Details, &detailsMap); err != nil {
			return false
		}

		return detailsMap["result"] == result &&
			detailsMap["duration_ms"] == float64(1234) &&
			detailsMap["error_message"] == nil
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err := service.LogWithResult(ctx, &userID, action, resourceType, resourceID, result, nil, additionalDetails)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithResult_Failure(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionExecuteChangePlan
	resourceType := ResourceTypeChangePlan
	result := "failure"
	errorMessage := "SQL syntax error"

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		var detailsMap map[string]interface{}
		if err := json.Unmarshal(log.Details, &detailsMap); err != nil {
			return false
		}

		return detailsMap["result"] == result &&
			detailsMap["error_message"] == errorMessage
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err := service.LogWithResult(ctx, &userID, action, resourceType, resourceID, result, &errorMessage, nil)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_Log_RepositoryError(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionCreateChangePlan
	resourceType := ResourceTypeChangePlan

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.Anything).Return(assert.AnError)

	service := NewAuditLogService(mockRepo, logger)

	err := service.Log(ctx, &userID, action, resourceType, resourceID, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create audit log")
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_ActionTypes(t *testing.T) {
	// Verify action type constants are defined
	assert.NotEmpty(t, ActionLogin)
	assert.NotEmpty(t, ActionLogout)
	assert.NotEmpty(t, ActionCreateChangePlan)
	assert.NotEmpty(t, ActionApproveChangePlan)
	assert.NotEmpty(t, ActionExecuteChangePlan)
}

func TestAuditLogService_ResourceTypes(t *testing.T) {
	// Verify resource type constants are defined
	assert.NotEmpty(t, ResourceTypeUser)
	assert.NotEmpty(t, ResourceTypeProject)
	assert.NotEmpty(t, ResourceTypeChangePlan)
	assert.NotEmpty(t, ResourceTypeDatabase)
}

func TestAuditLogService_LogWithHTTP_NilRequest(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionLogin
	resourceType := ResourceTypeUser

	details := json.RawMessage(`{"result": "success"}`)

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.UserID != nil && *log.UserID == userID &&
			log.Action == action &&
			log.ResourceType == resourceType &&
			log.ResourceID == resourceID &&
			log.IPAddress == nil &&
			log.UserAgent == nil
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err := service.LogWithHTTP(ctx, &userID, action, resourceType, resourceID, nil, details)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithHTTP_XRealIP(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionLogin
	resourceType := ResourceTypeUser

	details := json.RawMessage(`{"result": "success"}`)

	req, err := http.NewRequest("POST", "/api/v1/login", nil)
	require.NoError(t, err)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	req.Header.Set("User-Agent", "TestAgent")

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.UserID != nil && *log.UserID == userID &&
			log.IPAddress != nil &&
			log.IPAddress.String() == "10.0.0.1"
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err = service.LogWithHTTP(ctx, &userID, action, resourceType, resourceID, req, details)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithHTTP_RemoteAddr(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionLogin
	resourceType := ResourceTypeUser

	details := json.RawMessage(`{"result": "success"}`)

	req, err := http.NewRequest("POST", "/api/v1/login", nil)
	require.NoError(t, err)
	req.RemoteAddr = "192.168.1.1:54321"

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.UserID != nil && *log.UserID == userID &&
			log.IPAddress != nil &&
			log.IPAddress.String() == "192.168.1.1"
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err = service.LogWithHTTP(ctx, &userID, action, resourceType, resourceID, req, details)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithHTTP_RemoteAddrNoPort(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionLogin
	resourceType := ResourceTypeUser

	details := json.RawMessage(`{"result": "success"}`)

	req, err := http.NewRequest("POST", "/api/v1/login", nil)
	require.NoError(t, err)
	req.RemoteAddr = "192.168.1.1"

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.UserID != nil && *log.UserID == userID &&
			log.IPAddress != nil &&
			log.IPAddress.String() == "192.168.1.1"
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err = service.LogWithHTTP(ctx, &userID, action, resourceType, resourceID, req, details)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithHTTP_EmptyUserAgent(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionLogin
	resourceType := ResourceTypeUser

	details := json.RawMessage(`{"result": "success"}`)

	req, err := http.NewRequest("POST", "/api/v1/login", nil)
	require.NoError(t, err)
	req.RemoteAddr = "192.168.1.1:54321"
	// No User-Agent header set

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.UserID != nil && *log.UserID == userID &&
			log.UserAgent == nil
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err = service.LogWithHTTP(ctx, &userID, action, resourceType, resourceID, req, details)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogStateTransition_NilAdditionalDetails(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	resourceType := ResourceTypeChangePlan
	fromStatus := "open"
	toStatus := "approved"

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		if log.Action != ActionApproveChangePlan {
			return false
		}

		var detailsMap map[string]interface{}
		if err := json.Unmarshal(log.Details, &detailsMap); err != nil {
			return false
		}

		return detailsMap["previous_status"] == fromStatus &&
			detailsMap["status"] == toStatus
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err := service.LogStateTransition(ctx, &userID, resourceType, resourceID, fromStatus, toStatus, nil)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogStateTransition_AllChangePlanStatuses(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	resourceType := ResourceTypeChangePlan

	testCases := []struct {
		name           string
		toStatus       string
		expectedAction string
	}{
		{"approved", "approved", ActionApproveChangePlan},
		{"closed", "closed", ActionCloseChangePlan},
		{"rejected", "rejected", ActionRejectChangePlan},
		{"in_progress", "in_progress", ActionExecuteChangePlan},
		{"completed", "completed", ActionExecuteChangePlan},
		{"failed", "failed", ActionExecuteChangePlan},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(MockAuditLogRepository)
			mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
				return log.Action == tc.expectedAction &&
					log.ResourceType == resourceType &&
					log.ResourceID == resourceID
			})).Return(nil)

			service := NewAuditLogService(mockRepo, logger)

			err := service.LogStateTransition(ctx, &userID, resourceType, resourceID, "open", tc.toStatus, nil)
			require.NoError(t, err)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAuditLogService_LogStateTransition_GenericResourceType(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	resourceType := ResourceTypeProject
	fromStatus := "active"
	toStatus := "inactive"

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		expectedAction := fmt.Sprintf("%s_%s", ActionUpdate, resourceType)
		return log.Action == expectedAction &&
			log.ResourceType == resourceType
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err := service.LogStateTransition(ctx, &userID, resourceType, resourceID, fromStatus, toStatus, nil)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogStateTransition_InvalidJSONDetails(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	resourceType := ResourceTypeChangePlan
	fromStatus := "open"
	toStatus := "approved"

	// Invalid JSON - should still work, just won't merge
	invalidJSON := json.RawMessage(`{invalid json}`)

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		if log.Action != ActionApproveChangePlan {
			return false
		}

		var detailsMap map[string]interface{}
		if err := json.Unmarshal(log.Details, &detailsMap); err != nil {
			return false
		}

		// Should still have state transition info even if merge failed
		return detailsMap["previous_status"] == fromStatus &&
			detailsMap["status"] == toStatus
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err := service.LogStateTransition(ctx, &userID, resourceType, resourceID, fromStatus, toStatus, invalidJSON)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithResult_NilAdditionalDetails(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionExecuteChangePlan
	resourceType := ResourceTypeChangePlan
	result := "success"

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		var detailsMap map[string]interface{}
		if err := json.Unmarshal(log.Details, &detailsMap); err != nil {
			return false
		}

		return detailsMap["result"] == result &&
			detailsMap["error_message"] == nil
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err := service.LogWithResult(ctx, &userID, action, resourceType, resourceID, result, nil, nil)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithResult_InvalidJSONDetails(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionExecuteChangePlan
	resourceType := ResourceTypeChangePlan
	result := "success"

	// Invalid JSON - should still work, just won't merge
	invalidJSON := json.RawMessage(`{invalid json}`)

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		var detailsMap map[string]interface{}
		if err := json.Unmarshal(log.Details, &detailsMap); err != nil {
			return false
		}

		// Should still have result even if merge failed
		return detailsMap["result"] == result
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err := service.LogWithResult(ctx, &userID, action, resourceType, resourceID, result, nil, invalidJSON)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithResult_WithAdditionalDetails(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionExecuteChangePlan
	resourceType := ResourceTypeChangePlan
	result := "success"

	additionalDetails := json.RawMessage(`{"duration_ms": 5000, "rows_affected": 10}`)

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		var detailsMap map[string]interface{}
		if err := json.Unmarshal(log.Details, &detailsMap); err != nil {
			return false
		}

		return detailsMap["result"] == result &&
			detailsMap["duration_ms"] == float64(5000) &&
			detailsMap["rows_affected"] == float64(10)
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err := service.LogWithResult(ctx, &userID, action, resourceType, resourceID, result, nil, additionalDetails)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithHTTP_InvalidIPAddress(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionLogin
	resourceType := ResourceTypeUser

	details := json.RawMessage(`{"result": "success"}`)

	req, err := http.NewRequest("POST", "/api/v1/login", nil)
	require.NoError(t, err)
	req.Header.Set("X-Forwarded-For", "invalid-ip-address")
	req.Header.Set("User-Agent", "TestAgent")

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.UserID != nil && *log.UserID == userID &&
			log.IPAddress == nil // Invalid IP should result in nil
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err = service.LogWithHTTP(ctx, &userID, action, resourceType, resourceID, req, details)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithHTTP_XForwardedForWithWhitespace(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionLogin
	resourceType := ResourceTypeUser

	details := json.RawMessage(`{"result": "success"}`)

	req, err := http.NewRequest("POST", "/api/v1/login", nil)
	require.NoError(t, err)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")
	req.Header.Set("User-Agent", "TestAgent")

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.UserID != nil && *log.UserID == userID &&
			log.IPAddress != nil &&
			log.IPAddress.String() == "10.0.0.1"
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err = service.LogWithHTTP(ctx, &userID, action, resourceType, resourceID, req, details)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithHTTP_IPv6Address(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionLogin
	resourceType := ResourceTypeUser

	details := json.RawMessage(`{"result": "success"}`)

	req, err := http.NewRequest("POST", "/api/v1/login", nil)
	require.NoError(t, err)
	req.Header.Set("X-Real-IP", "2001:0db8:85a3:0000:0000:8a2e:0370:7334")
	req.Header.Set("User-Agent", "TestAgent")

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.UserID != nil && *log.UserID == userID &&
			log.IPAddress != nil &&
			log.IPAddress.String() == "2001:db8:85a3::8a2e:370:7334"
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err = service.LogWithHTTP(ctx, &userID, action, resourceType, resourceID, req, details)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_Log_EmptyDetails(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionCreateChangePlan
	resourceType := ResourceTypeChangePlan

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.MatchedBy(func(log *models.AuditLog) bool {
		return log.UserID != nil && *log.UserID == userID &&
			log.Action == action &&
			log.ResourceType == resourceType &&
			log.ResourceID == resourceID &&
			len(log.Details) == 0
	})).Return(nil)

	service := NewAuditLogService(mockRepo, logger)

	err := service.Log(ctx, &userID, action, resourceType, resourceID, nil)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithHTTP_RepositoryError(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionLogin
	resourceType := ResourceTypeUser

	details := json.RawMessage(`{"result": "success"}`)

	req, err := http.NewRequest("POST", "/api/v1/login", nil)
	require.NoError(t, err)
	req.RemoteAddr = "192.168.1.1:54321"

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.Anything).Return(assert.AnError)

	service := NewAuditLogService(mockRepo, logger)

	err = service.LogWithHTTP(ctx, &userID, action, resourceType, resourceID, req, details)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create audit log")
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogStateTransition_RepositoryError(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	resourceType := ResourceTypeChangePlan
	fromStatus := "open"
	toStatus := "approved"

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.Anything).Return(assert.AnError)

	service := NewAuditLogService(mockRepo, logger)

	err := service.LogStateTransition(ctx, &userID, resourceType, resourceID, fromStatus, toStatus, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create audit log")
	mockRepo.AssertExpectations(t)
}

func TestAuditLogService_LogWithResult_RepositoryError(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	userID := uuid.New()
	resourceID := uuid.New()
	action := ActionExecuteChangePlan
	resourceType := ResourceTypeChangePlan
	result := "success"

	mockRepo := new(MockAuditLogRepository)
	mockRepo.On("Create", ctx, mock.Anything).Return(assert.AnError)

	service := NewAuditLogService(mockRepo, logger)

	err := service.LogWithResult(ctx, &userID, action, resourceType, resourceID, result, nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create audit log")
	mockRepo.AssertExpectations(t)
}
