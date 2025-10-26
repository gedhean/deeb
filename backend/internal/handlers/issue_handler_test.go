package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Mock IssueRepository for testing
type mockIssueRepository struct {
	issues    []*models.Issue
	createErr error
	getErr    error
	updateErr error
	deleteErr error
	listErr   error
	searchErr error
	countErr  error
}

func (m *mockIssueRepository) Create(ctx context.Context, issue *models.Issue) error {
	if m.createErr != nil {
		return m.createErr
	}
	if issue.ID == uuid.Nil {
		issue.ID = uuid.New()
	}
	now := time.Now().UTC()
	issue.CreatedAt = now
	issue.UpdatedAt = now
	m.issues = append(m.issues, issue)
	return nil
}

func (m *mockIssueRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Issue, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, i := range m.issues {
		if i.ID == id {
			return i, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockIssueRepository) Update(ctx context.Context, issue *models.Issue) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	for idx, i := range m.issues {
		if i.ID == issue.ID {
			m.issues[idx] = issue
			return nil
		}
	}
	return repository.ErrNotFound
}

func (m *mockIssueRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for idx, i := range m.issues {
		if i.ID == id {
			m.issues = append(m.issues[:idx], m.issues[idx+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

func (m *mockIssueRepository) List(ctx context.Context, filters repository.ListFilters) ([]*models.Issue, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.issues, nil
}

func (m *mockIssueRepository) Search(ctx context.Context, query string, filters repository.ListFilters) ([]*models.Issue, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.issues, nil
}

func (m *mockIssueRepository) Count(ctx context.Context, filters repository.ListFilters) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return int64(len(m.issues)), nil
}

func TestCreateIssue_Success(t *testing.T) {
	// Setup
	e := echo.New()
	mockRepo := &mockIssueRepository{}
	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, logger)

	projectID := uuid.New()
	targetDBID := uuid.New()
	userID := uuid.New()

	reqBody := `{
		"project_id": "` + projectID.String() + `",
		"title": "Add new column",
		"description": "Adding user_status column to users table",
		"change_type": "DDL",
		"sql_content": "ALTER TABLE users ADD COLUMN user_status VARCHAR(50);",
		"target_database_id": "` + targetDBID.String() + `"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID.String())

	// Execute
	err := handler.CreateIssue(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var responseIssue models.Issue
	err = json.Unmarshal(rec.Body.Bytes(), &responseIssue)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, responseIssue.ID)
	assert.Equal(t, "Add new column", responseIssue.Title)
	assert.Equal(t, models.IssueStatusDraft, responseIssue.Status)
	assert.Equal(t, userID, responseIssue.AuthorID)
	assert.Equal(t, projectID, responseIssue.ProjectID)
	assert.Equal(t, targetDBID, responseIssue.TargetDatabaseID)
	assert.Equal(t, models.ChangeTypeDDL, responseIssue.ChangeType)

	// Verify issue was added to repo
	assert.Len(t, mockRepo.issues, 1)
}

func TestCreateIssue_MissingTitle(t *testing.T) {
	// Setup
	e := echo.New()
	mockRepo := &mockIssueRepository{}
	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, logger)

	projectID := uuid.New()
	targetDBID := uuid.New()
	userID := uuid.New()

	reqBody := `{
		"project_id": "` + projectID.String() + `",
		"description": "Test description",
		"sql_content": "SELECT 1;",
		"target_database_id": "` + targetDBID.String() + `"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID.String())

	// Execute
	err := handler.CreateIssue(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "title is required")
}

func TestCreateIssue_MissingDescription(t *testing.T) {
	// Setup
	e := echo.New()
	mockRepo := &mockIssueRepository{}
	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, logger)

	projectID := uuid.New()
	targetDBID := uuid.New()
	userID := uuid.New()

	reqBody := `{
		"project_id": "` + projectID.String() + `",
		"title": "Test title",
		"sql_content": "SELECT 1;",
		"target_database_id": "` + targetDBID.String() + `"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID.String())

	// Execute
	err := handler.CreateIssue(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "description is required")
}

func TestCreateIssue_MissingSQLContent(t *testing.T) {
	// Setup
	e := echo.New()
	mockRepo := &mockIssueRepository{}
	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, logger)

	projectID := uuid.New()
	targetDBID := uuid.New()
	userID := uuid.New()

	reqBody := `{
		"project_id": "` + projectID.String() + `",
		"title": "Test title",
		"description": "Test description",
		"target_database_id": "` + targetDBID.String() + `"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID.String())

	// Execute
	err := handler.CreateIssue(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "sql_content is required")
}

func TestCreateIssue_MissingTargetDatabaseID(t *testing.T) {
	// Setup
	e := echo.New()
	mockRepo := &mockIssueRepository{}
	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, logger)

	projectID := uuid.New()
	userID := uuid.New()

	reqBody := `{
		"project_id": "` + projectID.String() + `",
		"title": "Test title",
		"description": "Test description",
		"sql_content": "SELECT 1;"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID.String())

	// Execute
	err := handler.CreateIssue(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "target_database_id is required")
}

func TestCreateIssue_MissingProjectID(t *testing.T) {
	// Setup
	e := echo.New()
	mockRepo := &mockIssueRepository{}
	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, logger)

	targetDBID := uuid.New()
	userID := uuid.New()

	reqBody := `{
		"title": "Test title",
		"description": "Test description",
		"sql_content": "SELECT 1;",
		"target_database_id": "` + targetDBID.String() + `"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID.String())

	// Execute
	err := handler.CreateIssue(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "project_id is required")
}

func TestCreateIssue_SQLContentTooLarge(t *testing.T) {
	// Setup
	e := echo.New()
	mockRepo := &mockIssueRepository{}
	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, logger)

	projectID := uuid.New()
	targetDBID := uuid.New()
	userID := uuid.New()

	// Create SQL content larger than 10MB
	largeSQLContent := strings.Repeat("A", MaxSQLContentSize+1)

	reqBody := `{
		"project_id": "` + projectID.String() + `",
		"title": "Test title",
		"description": "Test description",
		"sql_content": "` + largeSQLContent + `",
		"target_database_id": "` + targetDBID.String() + `"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID.String())

	// Execute
	err := handler.CreateIssue(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "exceeds maximum size")
}

func TestCreateIssue_Unauthenticated(t *testing.T) {
	// Setup
	e := echo.New()
	mockRepo := &mockIssueRepository{}
	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, logger)

	projectID := uuid.New()
	targetDBID := uuid.New()

	reqBody := `{
		"project_id": "` + projectID.String() + `",
		"title": "Test title",
		"description": "Test description",
		"sql_content": "SELECT 1;",
		"target_database_id": "` + targetDBID.String() + `"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// No user_id set in context

	// Execute
	err := handler.CreateIssue(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "not authenticated")
}

func TestCreateIssue_RepositoryError(t *testing.T) {
	// Setup
	e := echo.New()
	mockRepo := &mockIssueRepository{
		createErr: assert.AnError,
	}
	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, logger)

	projectID := uuid.New()
	targetDBID := uuid.New()
	userID := uuid.New()

	reqBody := `{
		"project_id": "` + projectID.String() + `",
		"title": "Test title",
		"description": "Test description",
		"sql_content": "SELECT 1;",
		"target_database_id": "` + targetDBID.String() + `"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID.String())

	// Execute
	err := handler.CreateIssue(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Failed to create issue")
}

func TestCreateIssue_InvalidJSON(t *testing.T) {
	// Setup
	e := echo.New()
	mockRepo := &mockIssueRepository{}
	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, logger)

	userID := uuid.New()

	reqBody := `{invalid json}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID.String())

	// Execute
	err := handler.CreateIssue(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid request body")
}

func TestCreateIssue_InvalidProjectIDFormat(t *testing.T) {
	// Setup
	e := echo.New()
	mockRepo := &mockIssueRepository{}
	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, logger)

	targetDBID := uuid.New()
	userID := uuid.New()

	reqBody := `{
		"project_id": "invalid-uuid",
		"title": "Test title",
		"description": "Test description",
		"sql_content": "SELECT 1;",
		"target_database_id": "` + targetDBID.String() + `"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID.String())

	// Execute
	err := handler.CreateIssue(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid project_id format")
}

func TestCreateIssue_InvalidTargetDatabaseIDFormat(t *testing.T) {
	// Setup
	e := echo.New()
	mockRepo := &mockIssueRepository{}
	logger := zap.NewNop()
	handler := NewIssueHandler(mockRepo, logger)

	projectID := uuid.New()
	userID := uuid.New()

	reqBody := `{
		"project_id": "` + projectID.String() + `",
		"title": "Test title",
		"description": "Test description",
		"sql_content": "SELECT 1;",
		"target_database_id": "invalid-uuid"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", userID.String())

	// Execute
	err := handler.CreateIssue(c)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid target_database_id format")
}
