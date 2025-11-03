package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// Action types for audit logging
const (
	// Authentication actions
	ActionLogin    = "login"
	ActionLogout   = "logout"
	ActionRegister = "register"

	// CRUD actions (generic)
	ActionCreate = "create"
	ActionRead   = "read"
	ActionUpdate = "update"
	ActionDelete = "delete"

	// Change plan actions
	ActionCreateChangePlan  = "create_change_plan"
	ActionUpdateChangePlan  = "update_change_plan"
	ActionDeleteChangePlan  = "delete_change_plan"
	ActionSubmitChangePlan  = "submit_change_plan"
	ActionApproveChangePlan = "approve_change_plan"
	ActionRejectChangePlan  = "reject_change_plan"
	ActionCloseChangePlan   = "close_change_plan"
	ActionExecuteChangePlan = "execute_change_plan"
	ActionAssignReviewer    = "assign_reviewer"
	ActionUnassignReviewer  = "unassign_reviewer"

	// Project actions
	ActionCreateProject           = "create_project"
	ActionUpdateProject           = "update_project"
	ActionDeleteProject           = "delete_project"
	ActionAddProjectMember        = "add_project_member"
	ActionRemoveProjectMember     = "remove_project_member"
	ActionUpdateProjectMemberRole = "update_project_member_role"

	// Database actions
	ActionCreateDatabase         = "create_database"
	ActionUpdateDatabase         = "update_database"
	ActionDeleteDatabase         = "delete_database"
	ActionCreateDatabaseInstance = "create_database_instance"
	ActionUpdateDatabaseInstance = "update_database_instance"
	ActionDeleteDatabaseInstance = "delete_database_instance"

	// User actions
	ActionCreateUser     = "create_user"
	ActionUpdateUser     = "update_user"
	ActionDeleteUser     = "delete_user"
	ActionUpdateUserRole = "update_user_role"

	// Environment actions
	ActionCreateEnvironment = "create_environment"
	ActionUpdateEnvironment = "update_environment"
	ActionDeleteEnvironment = "delete_environment"
)

// Resource types for audit logging
const (
	ResourceTypeUser             = "user"
	ResourceTypeProject          = "project"
	ResourceTypeEnvironment      = "environment"
	ResourceTypeDatabase         = "database"
	ResourceTypeDatabaseInstance = "database_instance"
	ResourceTypeChangePlan       = "change_plan"
	ResourceTypeApproval         = "approval"
	ResourceTypeExecutionLog     = "execution_log"
	ResourceTypeSchemaSnapshot   = "schema_snapshot"
	ResourceTypeAuditLog         = "audit_log"
	ResourceTypePolicy           = "policy"
	ResourceTypePolicyFinding    = "policy_finding"
)

// AuditLogService provides audit logging functionality
type AuditLogService struct {
	auditLogRepo repository.AuditLogRepository
	logger       *zap.Logger
}

// NewAuditLogService creates a new audit logging service
func NewAuditLogService(
	auditLogRepo repository.AuditLogRepository,
	logger *zap.Logger,
) *AuditLogService {
	return &AuditLogService{
		auditLogRepo: auditLogRepo,
		logger:       logger,
	}
}

// Log logs an audit entry for a user action
func (s *AuditLogService) Log(
	ctx context.Context,
	userID uuid.UUID,
	action string,
	resourceType string,
	resourceID uuid.UUID,
	details json.RawMessage,
) error {
	auditLog := &models.AuditLog{
		ID:           uuid.New(),
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		Timestamp:    time.Now(),
	}

	if err := s.auditLogRepo.Create(ctx, auditLog); err != nil {
		s.logger.Error("Failed to create audit log",
			zap.String("user_id", userID.String()),
			zap.String("action", action),
			zap.String("resource_type", resourceType),
			zap.String("resource_id", resourceID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	s.logger.Debug("Audit log created",
		zap.String("audit_log_id", auditLog.ID.String()),
		zap.String("user_id", userID.String()),
		zap.String("action", action),
		zap.String("resource_type", resourceType),
	)

	return nil
}

// LogWithHTTP logs an audit entry with HTTP request context (captures IP address and user agent)
func (s *AuditLogService) LogWithHTTP(
	ctx context.Context,
	userID uuid.UUID,
	action string,
	resourceType string,
	resourceID uuid.UUID,
	httpRequest *http.Request,
	details json.RawMessage,
) error {
	auditLog := &models.AuditLog{
		ID:           uuid.New(),
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		Timestamp:    time.Now(),
	}

	// Extract IP address from request
	if httpRequest != nil {
		ip := extractIPAddress(httpRequest)
		if ip != nil {
			auditLog.IPAddress = ip
		}

		userAgent := httpRequest.UserAgent()
		if userAgent != "" {
			auditLog.UserAgent = &userAgent
		}
	}

	if err := s.auditLogRepo.Create(ctx, auditLog); err != nil {
		s.logger.Error("Failed to create audit log with HTTP context",
			zap.String("user_id", userID.String()),
			zap.String("action", action),
			zap.String("resource_type", resourceType),
			zap.String("resource_id", resourceID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	s.logger.Debug("Audit log created with HTTP context",
		zap.String("audit_log_id", auditLog.ID.String()),
		zap.String("user_id", userID.String()),
		zap.String("action", action),
		zap.String("resource_type", resourceType),
	)

	return nil
}

// LogStateTransition logs a state transition for a resource (e.g., status change)
func (s *AuditLogService) LogStateTransition(
	ctx context.Context,
	userID uuid.UUID,
	resourceType string,
	resourceID uuid.UUID,
	fromStatus string,
	toStatus string,
	additionalDetails json.RawMessage,
) error {
	// Build details JSON with state transition information
	detailsMap := map[string]interface{}{
		"previous_status": fromStatus,
		"status":          toStatus,
	}

	// Merge additional details if provided
	if len(additionalDetails) > 0 {
		var additional map[string]interface{}
		if err := json.Unmarshal(additionalDetails, &additional); err == nil {
			for k, v := range additional {
				detailsMap[k] = v
			}
		}
	}

	detailsJSON, err := json.Marshal(detailsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal state transition details: %w", err)
	}

	// Determine action based on state transition
	action := determineActionFromTransition(resourceType, fromStatus, toStatus)

	return s.Log(ctx, userID, action, resourceType, resourceID, detailsJSON)
}

// LogWithResult logs an action with success/failure result
func (s *AuditLogService) LogWithResult(
	ctx context.Context,
	userID uuid.UUID,
	action string,
	resourceType string,
	resourceID uuid.UUID,
	result string, // "success" or "failure"
	errorMessage *string,
	additionalDetails json.RawMessage,
) error {
	detailsMap := map[string]interface{}{
		"result": result,
	}

	if errorMessage != nil {
		detailsMap["error_message"] = *errorMessage
	}

	// Merge additional details if provided
	if len(additionalDetails) > 0 {
		var additional map[string]interface{}
		if err := json.Unmarshal(additionalDetails, &additional); err == nil {
			for k, v := range additional {
				detailsMap[k] = v
			}
		}
	}

	detailsJSON, err := json.Marshal(detailsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal result details: %w", err)
	}

	return s.Log(ctx, userID, action, resourceType, resourceID, detailsJSON)
}

// extractIPAddress extracts the real IP address from HTTP request
// Handles X-Forwarded-For, X-Real-IP headers for proxies
func extractIPAddress(r *http.Request) *net.IP {
	// Check X-Forwarded-For header (used by proxies)
	// X-Forwarded-For can contain multiple IPs separated by commas, take the first one
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Split by comma and take first IP
		for i, char := range forwarded {
			if char == ',' {
				forwarded = forwarded[:i]
				break
			}
		}
		// Trim whitespace
		for len(forwarded) > 0 && forwarded[0] == ' ' {
			forwarded = forwarded[1:]
		}
		if ip := parseIP(forwarded); ip != nil {
			return ip
		}
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		if ip := parseIP(realIP); ip != nil {
			return ip
		}
	}

	// Fall back to RemoteAddr
	if r.RemoteAddr != "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			if ip := parseIP(host); ip != nil {
				return ip
			}
		} else {
			// Try parsing as IP without port
			if ip := parseIP(r.RemoteAddr); ip != nil {
				return ip
			}
		}
	}

	return nil
}

// parseIP parses an IP address string and returns a net.IP
func parseIP(ipStr string) *net.IP {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil
	}
	return &ip
}

// determineActionFromTransition determines the action type from a state transition
// This helps create meaningful action names for state changes
func determineActionFromTransition(resourceType, fromStatus, toStatus string) string {
	// For change plans, use specific action types
	if resourceType == ResourceTypeChangePlan {
		switch toStatus {
		case "approved":
			return ActionApproveChangePlan
		case "closed":
			return ActionCloseChangePlan
		case "rejected":
			return ActionRejectChangePlan
		case "in_progress":
			return ActionExecuteChangePlan
		case "completed":
			return ActionExecuteChangePlan
		case "failed":
			return ActionExecuteChangePlan
		}
	}

	// Generic state transition
	return fmt.Sprintf("%s_%s", ActionUpdate, resourceType)
}
