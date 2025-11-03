package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
)

// AuditLogListFilters defines filtering options for listing audit logs
type AuditLogListFilters struct {
	Limit        int       // Maximum number of results to return
	Offset       int       // Number of results to skip
	UserID       uuid.UUID // Filter by user ID (Nil for all)
	Action       string    // Filter by action (empty for all)
	ResourceType string    // Filter by resource type (empty for all)
	ResourceID   uuid.UUID // Filter by resource ID (Nil for all)
}

// AuditLogRepository defines the interface for audit log data operations
type AuditLogRepository interface {
	// Create creates a new audit log entry in the database
	Create(ctx context.Context, auditLog *models.AuditLog) error

	// List retrieves all audit logs with pagination
	List(ctx context.Context, limit, offset int) ([]*models.AuditLog, error)

	// GetByUserID retrieves audit logs filtered by user ID
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.AuditLog, error)

	// GetByAction retrieves audit logs filtered by action
	GetByAction(ctx context.Context, action string, limit, offset int) ([]*models.AuditLog, error)

	// GetByTargetType retrieves audit logs filtered by resource type
	GetByTargetType(ctx context.Context, resourceType string, limit, offset int) ([]*models.AuditLog, error)

	// Filter retrieves audit logs with combined filters
	Filter(ctx context.Context, filters AuditLogListFilters) ([]*models.AuditLog, error)

	// Count returns the total number of audit logs
	Count(ctx context.Context) (int64, error)

	// CountWithFilters returns the total number of audit logs matching filters
	CountWithFilters(ctx context.Context, filters AuditLogListFilters) (int64, error)
}
