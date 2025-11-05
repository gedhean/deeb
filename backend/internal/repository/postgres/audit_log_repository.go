package postgres

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
)

// AuditLogRepository is the PostgreSQL implementation of repository.AuditLogRepository
type AuditLogRepository struct {
	db *sqlx.DB
}

// auditLogRow is a helper struct for scanning audit log rows from PostgreSQL
type auditLogRow struct {
	ID           string  `db:"id"`
	UserID       string  `db:"user_id"`
	Action       string  `db:"action"`
	ResourceType string  `db:"resource_type"`
	ResourceID   string  `db:"resource_id"`
	Details      []byte  `db:"details"`
	IPAddress    *string `db:"ip_address"` // TEXT from host() function
	UserAgent    *string `db:"user_agent"`
	Timestamp    string  `db:"timestamp"`
}

// toAuditLog converts auditLogRow to models.AuditLog
func (r *auditLogRow) toAuditLog() (*models.AuditLog, error) {
	id, err := uuid.Parse(r.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse audit log ID: %w", err)
	}

	userID, err := uuid.Parse(r.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user ID: %w", err)
	}

	resourceID, err := uuid.Parse(r.ResourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource ID: %w", err)
	}

	// Try parsing timestamp in multiple formats
	// PostgreSQL can return different formats depending on how timestamp is cast
	var timestamp time.Time

	// Try RFC3339 format first (most common)
	timestamp, err = time.Parse(time.RFC3339Nano, r.Timestamp)
	if err != nil {
		// Try PostgreSQL text format with space
		timestamp, err = time.Parse("2006-01-02 15:04:05.999999-07", r.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}
	}

	var ipAddress *net.IP
	if r.IPAddress != nil && *r.IPAddress != "" {
		ip := net.ParseIP(*r.IPAddress)
		if ip != nil {
			ipAddress = &ip
		}
	}

	return &models.AuditLog{
		ID:           id,
		UserID:       userID,
		Action:       r.Action,
		ResourceType: r.ResourceType,
		ResourceID:   resourceID,
		Details:      r.Details,
		IPAddress:    ipAddress,
		UserAgent:    r.UserAgent,
		Timestamp:    timestamp,
	}, nil
}

// NewAuditLogRepository creates a new PostgreSQL audit log repository
func NewAuditLogRepository(db *sqlx.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create creates a new audit log entry in the database
func (r *AuditLogRepository) Create(ctx context.Context, auditLog *models.AuditLog) error {
	// Set ID if not set
	if auditLog.ID == uuid.Nil {
		auditLog.ID = uuid.New()
	}

	// Set timestamp if not set
	if auditLog.Timestamp.IsZero() {
		auditLog.Timestamp = time.Now()
	}

	query := `
		INSERT INTO audit_logs (
			id, user_id, action, resource_type, resource_id,
			details, ip_address, user_agent, timestamp
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	// Convert net.IP to string for PostgreSQL INET type
	var ipAddress interface{}
	if auditLog.IPAddress != nil {
		ipAddress = auditLog.IPAddress.String()
	}

	_, err := r.db.ExecContext(ctx, query,
		auditLog.ID,
		auditLog.UserID,
		auditLog.Action,
		auditLog.ResourceType,
		auditLog.ResourceID,
		auditLog.Details,
		ipAddress,
		auditLog.UserAgent,
		auditLog.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// List retrieves all audit logs with pagination
func (r *AuditLogRepository) List(ctx context.Context, limit, offset int) ([]*models.AuditLog, error) {
	query := `
		SELECT id::text, user_id::text, action, resource_type, resource_id::text,
			details, host(ip_address) as ip_address, user_agent, timestamp::text
		FROM audit_logs
		ORDER BY timestamp DESC
		LIMIT $1 OFFSET $2
	`

	var rows []auditLogRow
	err := r.db.SelectContext(ctx, &rows, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}

	auditLogs := make([]*models.AuditLog, 0, len(rows))
	for _, row := range rows {
		auditLog, err := row.toAuditLog()
		if err != nil {
			return nil, err
		}
		auditLogs = append(auditLogs, auditLog)
	}

	return auditLogs, nil
}

// GetByUserID retrieves audit logs filtered by user ID
func (r *AuditLogRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.AuditLog, error) {
	query := `
		SELECT id::text, user_id::text, action, resource_type, resource_id::text,
			details, host(ip_address) as ip_address, user_agent, timestamp::text
		FROM audit_logs
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	var rows []auditLogRow
	err := r.db.SelectContext(ctx, &rows, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by user ID: %w", err)
	}

	auditLogs := make([]*models.AuditLog, 0, len(rows))
	for _, row := range rows {
		auditLog, err := row.toAuditLog()
		if err != nil {
			return nil, err
		}
		auditLogs = append(auditLogs, auditLog)
	}

	return auditLogs, nil
}

// GetByAction retrieves audit logs filtered by action
func (r *AuditLogRepository) GetByAction(ctx context.Context, action string, limit, offset int) ([]*models.AuditLog, error) {
	query := `
		SELECT id::text, user_id::text, action, resource_type, resource_id::text,
			details, host(ip_address) as ip_address, user_agent, timestamp::text
		FROM audit_logs
		WHERE action = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	var rows []auditLogRow
	err := r.db.SelectContext(ctx, &rows, query, action, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by action: %w", err)
	}

	auditLogs := make([]*models.AuditLog, 0, len(rows))
	for _, row := range rows {
		auditLog, err := row.toAuditLog()
		if err != nil {
			return nil, err
		}
		auditLogs = append(auditLogs, auditLog)
	}

	return auditLogs, nil
}

// GetByTargetType retrieves audit logs filtered by resource type
func (r *AuditLogRepository) GetByTargetType(
	ctx context.Context,
	resourceType string,
	limit, offset int,
) ([]*models.AuditLog, error) {
	query := `
		SELECT id::text, user_id::text, action, resource_type, resource_id::text,
			details, host(ip_address) as ip_address, user_agent, timestamp::text
		FROM audit_logs
		WHERE resource_type = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	var rows []auditLogRow
	err := r.db.SelectContext(ctx, &rows, query, resourceType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by resource type: %w", err)
	}

	auditLogs := make([]*models.AuditLog, 0, len(rows))
	for _, row := range rows {
		auditLog, err := row.toAuditLog()
		if err != nil {
			return nil, err
		}
		auditLogs = append(auditLogs, auditLog)
	}

	return auditLogs, nil
}

// Filter retrieves audit logs with combined filters
func (r *AuditLogRepository) Filter(ctx context.Context, filters repository.AuditLogListFilters) ([]*models.AuditLog, error) {
	// Build WHERE clause dynamically based on filters
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if filters.UserID != uuid.Nil {
		whereConditions = append(whereConditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, filters.UserID)
		argIndex++
	}

	if filters.Action != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("action = $%d", argIndex))
		args = append(args, filters.Action)
		argIndex++
	}

	if filters.ResourceType != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("resource_type = $%d", argIndex))
		args = append(args, filters.ResourceType)
		argIndex++
	}

	if filters.ResourceID != uuid.Nil {
		whereConditions = append(whereConditions, fmt.Sprintf("resource_id = $%d", argIndex))
		args = append(args, filters.ResourceID)
		argIndex++
	}

	if filters.StartDate != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *filters.StartDate)
		argIndex++
	}

	if filters.EndDate != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *filters.EndDate)
		argIndex++
	}

	// Build query
	query := `
		SELECT id::text, user_id::text, action, resource_type, resource_id::text,
			details, host(ip_address) as ip_address, user_agent, timestamp::text
		FROM audit_logs
	`

	if len(whereConditions) > 0 {
		query += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	query += " ORDER BY timestamp DESC"

	// Add pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filters.Limit)
		argIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filters.Offset)
	}

	var rows []auditLogRow
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to filter audit logs: %w", err)
	}

	auditLogs := make([]*models.AuditLog, 0, len(rows))
	for _, row := range rows {
		auditLog, err := row.toAuditLog()
		if err != nil {
			return nil, err
		}
		auditLogs = append(auditLogs, auditLog)
	}

	return auditLogs, nil
}

// Count returns the total number of audit logs
func (r *AuditLogRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM audit_logs`

	var count int64
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	return count, nil
}

// CountWithFilters returns the total number of audit logs matching filters
func (r *AuditLogRepository) CountWithFilters(ctx context.Context, filters repository.AuditLogListFilters) (int64, error) {
	// Build WHERE clause dynamically based on filters (same as Filter method)
	whereConditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if filters.UserID != uuid.Nil {
		whereConditions = append(whereConditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, filters.UserID)
		argIndex++
	}

	if filters.Action != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("action = $%d", argIndex))
		args = append(args, filters.Action)
		argIndex++
	}

	if filters.ResourceType != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("resource_type = $%d", argIndex))
		args = append(args, filters.ResourceType)
		argIndex++
	}

	if filters.ResourceID != uuid.Nil {
		whereConditions = append(whereConditions, fmt.Sprintf("resource_id = $%d", argIndex))
		args = append(args, filters.ResourceID)
		argIndex++
	}

	if filters.StartDate != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("timestamp >= $%d", argIndex))
		args = append(args, *filters.StartDate)
		argIndex++
	}

	if filters.EndDate != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("timestamp <= $%d", argIndex))
		args = append(args, *filters.EndDate)
	}

	// Build query
	query := `SELECT COUNT(*) FROM audit_logs`

	if len(whereConditions) > 0 {
		query += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	var count int64
	err := r.db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs with filters: %w", err)
	}

	return count, nil
}
