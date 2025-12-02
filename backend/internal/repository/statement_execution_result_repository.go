package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// StatementExecutionResultRepository defines the interface for statement execution result operations
type StatementExecutionResultRepository interface {
	// Create creates a new statement execution result
	Create(ctx context.Context, result *models.StatementExecutionResult) error

	// CreateBatch creates multiple statement execution results in a single transaction
	CreateBatch(ctx context.Context, results []models.StatementExecutionResult) error

	// GetByExecutionLogID retrieves all statement results for an execution log
	// Returns results ordered by statement_index ASC
	GetByExecutionLogID(ctx context.Context, executionLogID uuid.UUID) ([]*models.StatementExecutionResult, error)

	// GetByID retrieves a statement execution result by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.StatementExecutionResult, error)
}
