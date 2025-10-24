package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db      *sqlx.DB
	logger  *zap.Logger
	version string
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Database  string    `json:"database"`
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *sqlx.DB, logger *zap.Logger, version string) *HealthHandler {
	return &HealthHandler{
		db:      db,
		logger:  logger,
		version: version,
	}
}

// Health handles GET /health
func (h *HealthHandler) Health(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Check database connectivity
	dbStatus := "ok"
	if err := h.db.PingContext(ctx); err != nil {
		h.logger.Error("Database health check failed", zap.Error(err))
		dbStatus = "unhealthy"

		response := &HealthResponse{
			Status:    "unhealthy",
			Version:   h.version,
			Timestamp: time.Now(),
			Database:  dbStatus,
		}

		return c.JSON(http.StatusServiceUnavailable, response)
	}

	response := &HealthResponse{
		Status:    "healthy",
		Version:   h.version,
		Timestamp: time.Now(),
		Database:  dbStatus,
	}

	return c.JSON(http.StatusOK, response)
}
