package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ChangePlan), args.Error(1)
}

func (m *MockChangePlanRepository) ListWithFilters(
	ctx context.Context,
	filters repository.ChangePlanListFilters,
) ([]*models.ChangePlan, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ChangePlan), args.Error(1)
}

func (m *MockChangePlanRepository) GetByProjectID(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	args := m.Called(ctx, projectID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ChangePlan), args.Error(1)
}

func (m *MockChangePlanRepository) GetByDatabaseID(
	ctx context.Context,
	databaseID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	args := m.Called(ctx, databaseID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ChangePlan), args.Error(1)
}

func (m *MockChangePlanRepository) GetByAuthorID(
	ctx context.Context,
	authorID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	args := m.Called(ctx, authorID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ChangePlan), args.Error(1)
}

func TestNewChangePlanStatusTransitionMiddleware(t *testing.T) {
	mockRepo := new(MockChangePlanRepository)
	middleware := NewChangePlanStatusTransitionMiddleware(mockRepo)

	assert.NotNil(t, middleware)
	assert.Equal(t, mockRepo, middleware.changePlanRepo)
}

func TestValidateStatusTransition_NonPUTRequest(t *testing.T) {
	mockRepo := new(MockChangePlanRepository)
	middleware := NewChangePlanStatusTransitionMiddleware(mockRepo)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/change-plans/123", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("123")

	handlerCalled := false
	testHandler := func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	}

	err := middleware.ValidateStatusTransition(testHandler)(c)
	require.NoError(t, err)
	assert.True(t, handlerCalled, "handler should be called for non-PUT requests")
}

func TestValidateStatusTransition_InvalidID(t *testing.T) {
	mockRepo := new(MockChangePlanRepository)
	middleware := NewChangePlanStatusTransitionMiddleware(mockRepo)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/change-plans/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	handlerCalled := false
	testHandler := func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	}

	err := middleware.ValidateStatusTransition(testHandler)(c)
	require.NoError(t, err)
	assert.True(t, handlerCalled, "handler should be called when ID is invalid")
}

func TestValidateStatusTransition_ChangePlanNotFound(t *testing.T) {
	mockRepo := new(MockChangePlanRepository)
	middleware := NewChangePlanStatusTransitionMiddleware(mockRepo)

	changePlanID := uuid.New()
	mockRepo.On("GetByID", mock.Anything, changePlanID).Return(nil, repository.ErrNotFound)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/change-plans/"+changePlanID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(changePlanID.String())

	handlerCalled := false
	testHandler := func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	}

	err := middleware.ValidateStatusTransition(testHandler)(c)
	require.NoError(t, err)
	assert.True(t, handlerCalled, "handler should be called when change plan not found")
}

func TestValidateStatusTransition_NoStatusField(t *testing.T) {
	mockRepo := new(MockChangePlanRepository)
	middleware := NewChangePlanStatusTransitionMiddleware(mockRepo)

	changePlanID := uuid.New()
	changePlan := &models.ChangePlan{
		ID:     changePlanID,
		Status: models.ChangePlanStatusDraft,
	}
	mockRepo.On("GetByID", mock.Anything, changePlanID).Return(changePlan, nil)

	reqBody := map[string]interface{}{
		"title": "Updated Title",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/change-plans/"+changePlanID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(changePlanID.String())

	handlerCalled := false
	testHandler := func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	}

	err := middleware.ValidateStatusTransition(testHandler)(c)
	require.NoError(t, err)
	assert.True(t, handlerCalled, "handler should be called when status field is not present")
}

func TestValidateStatusTransition_ValidTransition(t *testing.T) {
	mockRepo := new(MockChangePlanRepository)
	middleware := NewChangePlanStatusTransitionMiddleware(mockRepo)

	changePlanID := uuid.New()
	changePlan := &models.ChangePlan{
		ID:     changePlanID,
		Status: models.ChangePlanStatusDraft,
	}
	mockRepo.On("GetByID", mock.Anything, changePlanID).Return(changePlan, nil)

	reqBody := map[string]interface{}{
		"status": "Open",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/change-plans/"+changePlanID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(changePlanID.String())

	handlerCalled := false
	testHandler := func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	}

	err := middleware.ValidateStatusTransition(testHandler)(c)
	require.NoError(t, err)
	assert.True(t, handlerCalled, "handler should be called when transition is valid")
}

func TestValidateStatusTransition_InvalidTransition(t *testing.T) {
	mockRepo := new(MockChangePlanRepository)
	middleware := NewChangePlanStatusTransitionMiddleware(mockRepo)

	changePlanID := uuid.New()
	changePlan := &models.ChangePlan{
		ID:     changePlanID,
		Status: models.ChangePlanStatusDraft,
	}
	mockRepo.On("GetByID", mock.Anything, changePlanID).Return(changePlan, nil)

	reqBody := map[string]interface{}{
		"status": "Approved", // Invalid: Draft → Approved is not allowed
	}
	bodyBytes, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/change-plans/"+changePlanID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(changePlanID.String())

	handlerCalled := false
	testHandler := func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	}

	err := middleware.ValidateStatusTransition(testHandler)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, handlerCalled, "handler should not be called when transition is invalid")

	var errorResp map[string]interface{}
	err = json.NewDecoder(rec.Body).Decode(&errorResp)
	require.NoError(t, err)
	// Verify error message explains why transition is invalid
	assert.Contains(t, errorResp["error"].(string), "Cannot transition")
	assert.Contains(t, errorResp["error"].(string), "Draft")
	assert.Contains(t, errorResp["error"].(string), "Approved")
	// Verify error response includes current status and attempted transition
	assert.NotNil(t, errorResp["details"])
	details, ok := errorResp["details"].(map[string]interface{})
	require.True(t, ok, "details should be a map")
	assert.Equal(t, "Draft", details["current_status"])
	assert.Equal(t, "Approved", details["attempted_status"])
}

func TestValidateStatusTransition_InvalidStatusValue(t *testing.T) {
	mockRepo := new(MockChangePlanRepository)
	middleware := NewChangePlanStatusTransitionMiddleware(mockRepo)

	changePlanID := uuid.New()
	changePlan := &models.ChangePlan{
		ID:     changePlanID,
		Status: models.ChangePlanStatusDraft,
	}
	mockRepo.On("GetByID", mock.Anything, changePlanID).Return(changePlan, nil)

	reqBody := map[string]interface{}{
		"status": "InvalidStatus",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/change-plans/"+changePlanID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(changePlanID.String())

	handlerCalled := false
	testHandler := func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	}

	err := middleware.ValidateStatusTransition(testHandler)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, handlerCalled, "handler should not be called when status value is invalid")

	var errorResp map[string]interface{}
	err = json.NewDecoder(rec.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"].(string), "Invalid status value")
}

func TestValidateStatusTransition_InvalidStatusFormat(t *testing.T) {
	mockRepo := new(MockChangePlanRepository)
	middleware := NewChangePlanStatusTransitionMiddleware(mockRepo)

	changePlanID := uuid.New()
	changePlan := &models.ChangePlan{
		ID:     changePlanID,
		Status: models.ChangePlanStatusDraft,
	}
	mockRepo.On("GetByID", mock.Anything, changePlanID).Return(changePlan, nil)

	reqBody := map[string]interface{}{
		"status": 123, // Invalid: status should be a string
	}
	bodyBytes, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/change-plans/"+changePlanID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(changePlanID.String())

	handlerCalled := false
	testHandler := func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	}

	err := middleware.ValidateStatusTransition(testHandler)(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.False(t, handlerCalled, "handler should not be called when status format is invalid")

	var errorResp map[string]interface{}
	err = json.NewDecoder(rec.Body).Decode(&errorResp)
	require.NoError(t, err)
	assert.Contains(t, errorResp["error"].(string), "Invalid status format")
}
