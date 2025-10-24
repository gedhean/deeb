package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func setupHealthMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestHealth(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	t.Run("health check returns healthy when database is ok", func(t *testing.T) {
		db, mock := setupHealthMockDB(t)
		defer db.Close()

		handler := NewHealthHandler(db, logger, "1.0.0")

		mock.ExpectPing()

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := handler.Health(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response HealthResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got %s", response.Status)
		}

		if response.Version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got %s", response.Version)
		}

		if response.Database != "ok" {
			t.Errorf("Expected database 'ok', got %s", response.Database)
		}

		if response.Timestamp.IsZero() {
			t.Error("Expected timestamp to be set")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("health check returns unhealthy when database is down", func(t *testing.T) {
		db, mock := setupHealthMockDB(t)
		defer db.Close()

		handler := NewHealthHandler(db, logger, "1.0.0")

		mock.ExpectPing().WillReturnError(sql.ErrConnDone)

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := handler.Health(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", rec.Code)
		}

		var response HealthResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Status != "unhealthy" {
			t.Errorf("Expected status 'unhealthy', got %s", response.Status)
		}

		if response.Database != "unhealthy" {
			t.Errorf("Expected database 'unhealthy', got %s", response.Database)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("health check returns unhealthy on database timeout", func(t *testing.T) {
		db, mock := setupHealthMockDB(t)
		defer db.Close()

		handler := NewHealthHandler(db, logger, "1.0.0")

		mock.ExpectPing().WillReturnError(errors.New("context deadline exceeded"))

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if err := handler.Health(c); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", rec.Code)
		}

		var response HealthResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Status != "unhealthy" {
			t.Errorf("Expected status 'unhealthy', got %s", response.Status)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}
