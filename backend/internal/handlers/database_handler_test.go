package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

type mockDatabaseRepository struct {
	databases         []*models.Database
	getErr            error
	listErr           error
	listByProjectErr  error
	createErr         error
	updateErr         error
	deleteErr         error
}

func (m *mockDatabaseRepository) Create(ctx context.Context, database *models.Database) error {
	if m.createErr != nil {
		return m.createErr
	}
	database.ID = uuid.New()
	m.databases = append(m.databases, database)
	return nil
}

func (m *mockDatabaseRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Database, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, database := range m.databases {
		if database.ID == id {
			return database, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockDatabaseRepository) List(ctx context.Context, limit, offset int) ([]*models.Database, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.databases, nil
}

func (m *mockDatabaseRepository) ListByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*models.Database, error) {
	if m.listByProjectErr != nil {
		return nil, m.listByProjectErr
	}
	var filtered []*models.Database
	for _, db := range m.databases {
		if db.ProjectID == projectID {
			filtered = append(filtered, db)
		}
	}
	return filtered, nil
}

func (m *mockDatabaseRepository) Update(ctx context.Context, database *models.Database) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	for i, existing := range m.databases {
		if existing.ID == database.ID {
			m.databases[i] = database
			return nil
		}
	}
	return repository.ErrNotFound
}

func (m *mockDatabaseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, database := range m.databases {
		if database.ID == id {
			m.databases = append(m.databases[:i], m.databases[i+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

func TestCreateDatabase(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	t.Run("create database successfully", func(t *testing.T) {
		repo := &mockDatabaseRepository{}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()
		e.Validator = &noOpValidator{}

		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		reqBody := `{
			"name": "test_database",
			"database_instance_id": "` + dbInstanceID.String() + `",
			"project_id": "` + projectID.String() + `",
			"environment_id": "` + envID.String() + `"
		}`

		req := httptest.NewRequest(http.MethodPost, "/api/v1/databases", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := handler.CreateDatabase(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", rec.Code)
		}

		var response models.DatabaseResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Name != "test_database" {
			t.Errorf("Expected name 'test_database', got %s", response.Name)
		}
	})

	t.Run("create database with validation error", func(t *testing.T) {
		repo := &mockDatabaseRepository{}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()

		reqBody := `{
			"name": "",
			"database_instance_id": "` + uuid.New().String() + `",
			"project_id": "` + uuid.New().String() + `",
			"environment_id": "` + uuid.New().String() + `"
		}`

		req := httptest.NewRequest(http.MethodPost, "/api/v1/databases", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := handler.CreateDatabase(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

func TestListDatabases(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	t.Run("list databases successfully", func(t *testing.T) {
		projectID := uuid.New()
		repo := &mockDatabaseRepository{
			databases: []*models.Database{
				{
					ID:                 uuid.New(),
					Name:               "database1",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          projectID,
					EnvironmentID:      uuid.New(),
				},
				{
					ID:                 uuid.New(),
					Name:               "database2",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          projectID,
					EnvironmentID:      uuid.New(),
				},
			},
		}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/databases", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := handler.ListDatabases(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		databases := response["databases"].([]interface{})
		if len(databases) != 2 {
			t.Errorf("Expected 2 databases, got %d", len(databases))
		}
	})

	t.Run("list databases with project filter", func(t *testing.T) {
		projectID := uuid.New()
		otherProjectID := uuid.New()
		repo := &mockDatabaseRepository{
			databases: []*models.Database{
				{
					ID:                 uuid.New(),
					Name:               "database1",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          projectID,
					EnvironmentID:      uuid.New(),
				},
				{
					ID:                 uuid.New(),
					Name:               "database2",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          otherProjectID,
					EnvironmentID:      uuid.New(),
				},
			},
		}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/databases?project_id="+projectID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := handler.ListDatabases(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		databases := response["databases"].([]interface{})
		if len(databases) != 1 {
			t.Errorf("Expected 1 database, got %d", len(databases))
		}
	})

	t.Run("list databases with invalid project filter", func(t *testing.T) {
		repo := &mockDatabaseRepository{}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/databases?project_id=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := handler.ListDatabases(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

func TestGetDatabase(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	t.Run("get database successfully", func(t *testing.T) {
		databaseID := uuid.New()
		repo := &mockDatabaseRepository{
			databases: []*models.Database{
				{
					ID:                 databaseID,
					Name:               "test_database",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          uuid.New(),
					EnvironmentID:      uuid.New(),
				},
			},
		}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/"+databaseID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(databaseID.String())

		if err := handler.GetDatabase(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response models.DatabaseResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.ID != databaseID {
			t.Errorf("Expected ID %s, got %s", databaseID, response.ID)
		}
	})

	t.Run("get database not found", func(t *testing.T) {
		repo := &mockDatabaseRepository{}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()

		databaseID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/"+databaseID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(databaseID.String())

		if err := handler.GetDatabase(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

func TestListProjectDatabases(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	t.Run("list project databases successfully", func(t *testing.T) {
		projectID := uuid.New()
		repo := &mockDatabaseRepository{
			databases: []*models.Database{
				{
					ID:                 uuid.New(),
					Name:               "database1",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          projectID,
					EnvironmentID:      uuid.New(),
				},
				{
					ID:                 uuid.New(),
					Name:               "database2",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          projectID,
					EnvironmentID:      uuid.New(),
				},
			},
		}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/databases", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("project_id")
		c.SetParamValues(projectID.String())

		if err := handler.ListProjectDatabases(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		databases := response["databases"].([]interface{})
		if len(databases) != 2 {
			t.Errorf("Expected 2 databases, got %d", len(databases))
		}
	})

	t.Run("list project databases with invalid project ID", func(t *testing.T) {
		repo := &mockDatabaseRepository{}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/invalid/databases", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("project_id")
		c.SetParamValues("invalid")

		if err := handler.ListProjectDatabases(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}
	})
}

func TestUpdateDatabase(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	t.Run("update database successfully", func(t *testing.T) {
		databaseID := uuid.New()
		repo := &mockDatabaseRepository{
			databases: []*models.Database{
				{
					ID:                 databaseID,
					Name:               "original_name",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          uuid.New(),
					EnvironmentID:      uuid.New(),
				},
			},
		}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()
		e.Validator = &noOpValidator{}

		reqBody := `{
			"name": "updated_name"
		}`

		req := httptest.NewRequest(http.MethodPut, "/api/v1/databases/"+databaseID.String(), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(databaseID.String())

		if err := handler.UpdateDatabase(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response models.DatabaseResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Name != "updated_name" {
			t.Errorf("Expected name 'updated_name', got %s", response.Name)
		}
	})

	t.Run("update database not found", func(t *testing.T) {
		repo := &mockDatabaseRepository{}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()
		e.Validator = &noOpValidator{}

		databaseID := uuid.New()
		reqBody := `{
			"name": "updated_name"
		}`

		req := httptest.NewRequest(http.MethodPut, "/api/v1/databases/"+databaseID.String(), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(databaseID.String())

		if err := handler.UpdateDatabase(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}

func TestDeleteDatabase(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	t.Run("delete database successfully", func(t *testing.T) {
		databaseID := uuid.New()
		repo := &mockDatabaseRepository{
			databases: []*models.Database{
				{
					ID:                 databaseID,
					Name:               "test_database",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          uuid.New(),
					EnvironmentID:      uuid.New(),
				},
			},
		}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/databases/"+databaseID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(databaseID.String())

		if err := handler.DeleteDatabase(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["message"] != "Database deleted successfully" {
			t.Errorf("Expected success message, got %v", response["message"])
		}
	})

	t.Run("delete database not found", func(t *testing.T) {
		repo := &mockDatabaseRepository{}
		handler := NewDatabaseHandler(repo, logger)

		e := echo.New()

		databaseID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/databases/"+databaseID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(databaseID.String())

		if err := handler.DeleteDatabase(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}
