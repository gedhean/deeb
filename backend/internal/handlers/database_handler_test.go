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
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/middleware"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

type mockDatabaseRepository struct {
	databases        []*models.Database
	getErr           error
	listErr          error
	listByProjectErr error
	createErr        error
	updateErr        error
	deleteErr        error
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

func (m *mockDatabaseRepository) ListByProject(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
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

func (m *mockDatabaseRepository) ListActiveForSync(ctx context.Context, limit, offset int) ([]*models.Database, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var active []*models.Database
	for _, db := range m.databases {
		if db.Status == models.DatabaseStatusActive {
			active = append(active, db)
		}
	}
	return active, nil
}

func (m *mockDatabaseRepository) GetByInstanceAndName(
	ctx context.Context,
	instanceID uuid.UUID,
	databaseName string,
) (*models.Database, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, database := range m.databases {
		if database.DatabaseInstanceID == instanceID && database.DatabaseName == databaseName {
			return database, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockDatabaseRepository) Count(ctx context.Context) (int, error) {
	return len(m.databases), nil
}

func (m *mockDatabaseRepository) GetByProjectID(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	return m.ListByProject(ctx, projectID, limit, offset)
}

func (m *mockDatabaseRepository) ListByInstanceID(
	ctx context.Context,
	instanceID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	var filtered []*models.Database
	for _, db := range m.databases {
		if db.DatabaseInstanceID == instanceID {
			filtered = append(filtered, db)
		}
	}
	return filtered, nil
}

func (m *mockDatabaseRepository) ListByEnvironmentID(
	ctx context.Context,
	environmentID uuid.UUID,
	limit, offset int,
) ([]*models.Database, error) {
	var filtered []*models.Database
	for _, db := range m.databases {
		if db.EnvironmentID == environmentID {
			filtered = append(filtered, db)
		}
	}
	return filtered, nil
}

// Minimal mocks for other repositories
type mockProjectRepositoryForDB struct{}

func (m *mockProjectRepositoryForDB) Create(ctx context.Context, project *models.Project) error {
	return nil
}

func (m *mockProjectRepositoryForDB) GetByID(ctx context.Context, id uuid.UUID) (*models.Project, error) {
	return &models.Project{ID: id}, nil
}

func (m *mockProjectRepositoryForDB) List(ctx context.Context, limit, offset int) ([]*models.Project, error) {
	return nil, nil
}

func (m *mockProjectRepositoryForDB) ListByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit, offset int,
) ([]*models.Project, error) {
	return nil, nil
}

func (m *mockProjectRepositoryForDB) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockProjectRepositoryForDB) Update(ctx context.Context, project *models.Project) error {
	return nil
}
func (m *mockProjectRepositoryForDB) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockProjectRepositoryForDB) Count(ctx context.Context) (int64, error)       { return 0, nil }

type mockEnvironmentRepositoryForDB struct{}

func (m *mockEnvironmentRepositoryForDB) Create(ctx context.Context, env *models.Environment) error {
	return nil
}

func (m *mockEnvironmentRepositoryForDB) GetByID(ctx context.Context, id uuid.UUID) (*models.Environment, error) {
	return &models.Environment{ID: id}, nil
}

func (m *mockEnvironmentRepositoryForDB) List(ctx context.Context, limit, offset int) ([]*models.Environment, error) {
	return nil, nil
}

func (m *mockEnvironmentRepositoryForDB) Update(ctx context.Context, env *models.Environment) error {
	return nil
}
func (m *mockEnvironmentRepositoryForDB) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockEnvironmentRepositoryForDB) Count(ctx context.Context) (int64, error)       { return 0, nil }

type mockDatabaseInstanceRepositoryForDB struct{}

func (m *mockDatabaseInstanceRepositoryForDB) Create(ctx context.Context, instance *models.DatabaseInstance) error {
	return nil
}

func (m *mockDatabaseInstanceRepositoryForDB) GetByID(ctx context.Context, id uuid.UUID) (*models.DatabaseInstance, error) {
	return &models.DatabaseInstance{ID: id}, nil
}

func (m *mockDatabaseInstanceRepositoryForDB) List(ctx context.Context, limit, offset int) ([]*models.DatabaseInstance, error) {
	return nil, nil
}

func (m *mockDatabaseInstanceRepositoryForDB) Update(ctx context.Context, instance *models.DatabaseInstance) error {
	return nil
}

func (m *mockDatabaseInstanceRepositoryForDB) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockDatabaseInstanceRepositoryForDB) Count(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *mockDatabaseInstanceRepositoryForDB) ListByEnvironment(
	ctx context.Context,
	environmentID uuid.UUID,
	limit, offset int,
) ([]*models.DatabaseInstance, error) {
	return nil, nil
}

func (m *mockDatabaseInstanceRepositoryForDB) CountByEnvironment(ctx context.Context, environmentID uuid.UUID) (int64, error) {
	return 0, nil
}

type mockChangePlanRepositoryForDB struct{}

func (m *mockChangePlanRepositoryForDB) Create(ctx context.Context, plan *models.ChangePlan) error {
	return nil
}

func (m *mockChangePlanRepositoryForDB) GetByID(ctx context.Context, id uuid.UUID) (*models.ChangePlan, error) {
	return nil, nil
}

func (m *mockChangePlanRepositoryForDB) List(ctx context.Context, limit, offset int) ([]*models.ChangePlan, error) {
	return nil, nil
}

func (m *mockChangePlanRepositoryForDB) GetByDatabaseID(
	ctx context.Context,
	databaseID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	return nil, nil
}

func (m *mockChangePlanRepositoryForDB) Update(ctx context.Context, plan *models.ChangePlan) error {
	return nil
}
func (m *mockChangePlanRepositoryForDB) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockChangePlanRepositoryForDB) Count(ctx context.Context) (int64, error)       { return 0, nil }

func (m *mockChangePlanRepositoryForDB) GetByStatus(
	ctx context.Context,
	status models.ChangePlanStatus,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	return nil, nil
}

func (m *mockChangePlanRepositoryForDB) GetByProjectID(
	ctx context.Context,
	projectID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	return nil, nil
}

func (m *mockChangePlanRepositoryForDB) GetByAuthorID(
	ctx context.Context,
	authorID uuid.UUID,
	limit, offset int,
) ([]*models.ChangePlan, error) {
	return nil, nil
}

func (m *mockChangePlanRepositoryForDB) ListWithFilters(
	ctx context.Context,
	filters repository.ChangePlanListFilters,
) ([]*models.ChangePlan, error) {
	return nil, nil
}

func (m *mockChangePlanRepositoryForDB) CountWithFilters(
	ctx context.Context,
	filters repository.ChangePlanListFilters,
) (int64, error) {
	return 0, nil
}

func TestCreateDatabase(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	t.Run("create database successfully", func(t *testing.T) {
		repo := &mockDatabaseRepository{}
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()
		e.Validator = &noOpValidator{}

		dbInstanceID := uuid.New()
		projectID := uuid.New()
		envID := uuid.New()
		reqBody := `{
			"database_name": "test_database",
			"database_instance_id": "` + dbInstanceID.String() + `",
			"project_id": "` + projectID.String() + `",
			"environment_id": "` + envID.String() + `"
		}`

		req := httptest.NewRequest(http.MethodPost, "/api/v1/databases", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

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

		if response.DatabaseName != "test_database" {
			t.Errorf("Expected name 'test_database', got %s", response.DatabaseName)
		}
	})

	t.Run("create database with validation error", func(t *testing.T) {
		repo := &mockDatabaseRepository{}
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()
		e.HTTPErrorHandler = middleware.CustomErrorHandler

		reqBody := `{
			"database_name": "",
			"database_instance_id": "` + uuid.New().String() + `",
			"project_id": "` + uuid.New().String() + `",
			"environment_id": "` + uuid.New().String() + `"
		}`

		req := httptest.NewRequest(http.MethodPost, "/api/v1/databases", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

		err := handler.CreateDatabase(c)
		if err != nil {
			// Error handler should have written the response
			e.HTTPErrorHandler(err, c)
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d. Response: %s", rec.Code, rec.Body.String())
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
					DatabaseName:       "database1",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          projectID,
					EnvironmentID:      uuid.New(),
				},
				{
					ID:                 uuid.New(),
					DatabaseName:       "database2",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          projectID,
					EnvironmentID:      uuid.New(),
				},
			},
		}
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/databases", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

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
					DatabaseName:       "database1",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          projectID,
					EnvironmentID:      uuid.New(),
				},
				{
					ID:                 uuid.New(),
					DatabaseName:       "database2",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          otherProjectID,
					EnvironmentID:      uuid.New(),
				},
			},
		}
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/databases?project_id="+projectID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

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

		databases, ok := response["databases"].([]interface{})
		if !ok {
			t.Fatalf("Expected databases array, got %v", response)
		}
		if len(databases) != 1 {
			t.Errorf("Expected 1 database, got %d", len(databases))
		}
	})

	t.Run("list databases with invalid project filter", func(t *testing.T) {
		repo := &mockDatabaseRepository{}
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/databases?project_id=invalid", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

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
					DatabaseName:       "test_database",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          uuid.New(),
					EnvironmentID:      uuid.New(),
				},
			},
		}
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/"+databaseID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(databaseID.String())

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

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
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()

		databaseID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/databases/"+databaseID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(databaseID.String())

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

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
					DatabaseName:       "database1",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          projectID,
					EnvironmentID:      uuid.New(),
				},
				{
					ID:                 uuid.New(),
					DatabaseName:       "database2",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          projectID,
					EnvironmentID:      uuid.New(),
				},
			},
		}
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID.String()+"/databases", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("project_id")
		c.SetParamValues(projectID.String())

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

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

		databases, ok := response["databases"].([]interface{})
		if !ok {
			t.Fatalf("Expected databases array, got %v", response)
		}
		if len(databases) != 2 {
			t.Errorf("Expected 2 databases, got %d", len(databases))
		}
	})

	t.Run("list project databases with invalid project ID", func(t *testing.T) {
		repo := &mockDatabaseRepository{}
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/invalid/databases", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("project_id")
		c.SetParamValues("invalid")

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

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
					DatabaseName:       "original_name",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          uuid.New(),
					EnvironmentID:      uuid.New(),
				},
			},
		}
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()
		e.Validator = &noOpValidator{}

		reqBody := `{
			"database_name": "updated_name"
		}`

		req := httptest.NewRequest(http.MethodPut, "/api/v1/databases/"+databaseID.String(), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(databaseID.String())

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

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

		if response.DatabaseName != "updated_name" {
			t.Errorf("Expected name 'updated_name', got %s", response.DatabaseName)
		}
	})

	t.Run("update database not found", func(t *testing.T) {
		repo := &mockDatabaseRepository{}
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()
		e.Validator = &noOpValidator{}

		databaseID := uuid.New()
		reqBody := `{
			"database_name": "updated_name"
		}`

		req := httptest.NewRequest(http.MethodPut, "/api/v1/databases/"+databaseID.String(), strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(databaseID.String())

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

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
					DatabaseName:       "test_database",
					DatabaseInstanceID: uuid.New(),
					ProjectID:          uuid.New(),
					EnvironmentID:      uuid.New(),
				},
			},
		}
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/databases/"+databaseID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(databaseID.String())

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

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
		handler := NewDatabaseHandler(
			repo,
			&mockProjectRepositoryForDB{},
			&mockEnvironmentRepositoryForDB{},
			&mockDatabaseInstanceRepositoryForDB{},
			&mockChangePlanRepositoryForDB{},
			nil,
			logger,
		)

		e := echo.New()

		databaseID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/databases/"+databaseID.String(), nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues(databaseID.String())

		// Set authenticated user context
		userID := uuid.New()
		c.Set("user", &auth.Claims{
			UserID:           userID,
			Email:            "admin@example.com",
			OrganizationRole: string(rbac.OrganizationAdmin),
		})

		if err := handler.DeleteDatabase(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", rec.Code)
		}
	})
}
