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
//
// AUTHORITATIVE LIST: These constants define all valid action types for audit logging.
// When adding new actions, ensure they are also added to the audit_logging configuration
// in config.example.yaml to document them for users.
//
// Action Naming Convention:
//   - Generic CRUD actions use simple verbs: "create", "read", "update", "delete"
//   - Resource-specific actions use format: "{verb}_{resource}" (e.g., "approve_change_plan")
//   - Actions should be in snake_case and lowercase
const (
	// Authentication actions
	ActionLogin    = "login"    // User login attempt
	ActionLogout   = "logout"   // User logout
	ActionRegister = "register" // User registration

	// CRUD actions (generic)
	// These are automatically mapped from HTTP methods: POST→create, GET→read, PUT/PATCH→update, DELETE→delete
	ActionCreate = "create" // Generic create action (POST)
	ActionRead   = "read"   // Generic read action (GET)
	ActionUpdate = "update" // Generic update action (PUT/PATCH)
	ActionDelete = "delete" // Generic delete action (DELETE)

	// Change plan actions
	ActionCreateChangePlan  = "create_change_plan"  // Create a new change plan
	ActionUpdateChangePlan  = "update_change_plan"  // Update change plan details
	ActionDeleteChangePlan  = "delete_change_plan"  // Delete a change plan
	ActionSubmitChangePlan  = "submit_change_plan"  // Submit change plan for review
	ActionApproveChangePlan = "approve_change_plan" // Approve change plan (POST /change-plans/{id}/approve)
	ActionRejectChangePlan  = "reject_change_plan"  // Reject change plan (POST /change-plans/{id}/reject)
	ActionCloseChangePlan   = "close_change_plan"   // Close change plan (POST /change-plans/{id}/close)
	ActionExecuteChangePlan = "execute_change_plan" // Execute change plan (POST /change-plans/{id}/execute)
	ActionAssignReviewer    = "assign_reviewer"     // Assign reviewer to change plan
	ActionUnassignReviewer  = "unassign_reviewer"   // Remove reviewer from change plan

	// Project actions
	ActionCreateProject           = "create_project"             // Create new project
	ActionUpdateProject           = "update_project"             // Update project settings
	ActionDeleteProject           = "delete_project"             // Delete project
	ActionAddProjectMember        = "add_project_member"         // Add member to project
	ActionRemoveProjectMember     = "remove_project_member"      // Remove member from project
	ActionUpdateProjectMemberRole = "update_project_member_role" // Update project member's role

	// Database actions
	ActionCreateDatabase         = "create_database"          // Create database configuration
	ActionUpdateDatabase         = "update_database"          // Update database settings
	ActionDeleteDatabase         = "delete_database"          // Delete database configuration
	ActionCreateDatabaseInstance = "create_database_instance" // Create database instance
	ActionUpdateDatabaseInstance = "update_database_instance" // Update database instance
	ActionDeleteDatabaseInstance = "delete_database_instance" // Delete database instance

	// User actions
	ActionCreateUser     = "create_user"      // Create new user
	ActionUpdateUser     = "update_user"      // Update user details
	ActionDeleteUser     = "delete_user"      // Delete user
	ActionUpdateUserRole = "update_user_role" // Change user's role/permissions

	// Environment actions
	ActionCreateEnvironment = "create_environment" // Create environment (dev/staging/prod)
	ActionUpdateEnvironment = "update_environment" // Update environment settings
	ActionDeleteEnvironment = "delete_environment" // Delete environment

	// Policy actions
	ActionCreatePolicy = "create_policy" // Create validation policy
	ActionUpdatePolicy = "update_policy" // Update policy rules
	ActionDeletePolicy = "delete_policy" // Delete policy
)

// Resource types for audit logging
//
// AUTHORITATIVE LIST: These constants define all valid resource types for audit logging.
// When adding new resource types:
//  1. Add the constant here
//  2. Map it in middleware/audit_log.go extractResourceType() function
//  3. Add it to the audit_logging configuration in config.example.yaml
//
// Resource Naming Convention:
//   - Use snake_case and lowercase
//   - Be specific and descriptive
//   - Match the database table name where applicable
const (
	ResourceTypeUser             = "user"              // User accounts and authentication
	ResourceTypeProject          = "project"           // Project management
	ResourceTypeEnvironment      = "environment"       // Environment configurations (dev/staging/prod)
	ResourceTypeDatabase         = "database"          // Database configurations
	ResourceTypeDatabaseInstance = "database_instance" // Database instance connections
	ResourceTypeChangePlan       = "change_plan"       // Database change requests and workflow
	ResourceTypeApproval         = "approval"          // Change plan approvals
	ResourceTypeExecutionLog     = "execution_log"     // Change plan execution logs
	ResourceTypeChangelog        = "changelog"         // Database change history
	ResourceTypeSchemaHistory    = "schema_history"    // Schema version history
	ResourceTypeAuditLog         = "audit_log"         // Audit log entries (self-referential)
	ResourceTypePolicy           = "policy"            // Validation policies
	ResourceTypePolicyFinding    = "policy_finding"    // Policy violation findings
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
// userID can be nil for unauthenticated events like failed login attempts
func (s *AuditLogService) Log(
	ctx context.Context,
	userID *uuid.UUID,
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
		userIDStr := "nil"
		if userID != nil {
			userIDStr = userID.String()
		}
		s.logger.Error("Failed to create audit log",
			zap.String("user_id", userIDStr),
			zap.String("action", action),
			zap.String("resource_type", resourceType),
			zap.String("resource_id", resourceID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	userIDStr := "nil"
	if userID != nil {
		userIDStr = userID.String()
	}
	s.logger.Debug("Audit log created",
		zap.String("audit_log_id", auditLog.ID.String()),
		zap.String("user_id", userIDStr),
		zap.String("action", action),
		zap.String("resource_type", resourceType),
	)

	return nil
}

// LogWithHTTP logs an audit entry with HTTP request context (captures IP address and user agent)
// userID can be nil for unauthenticated events like failed login attempts
func (s *AuditLogService) LogWithHTTP(
	ctx context.Context,
	userID *uuid.UUID,
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
		userIDStr := "nil"
		if userID != nil {
			userIDStr = userID.String()
		}
		s.logger.Error("Failed to create audit log with HTTP context",
			zap.String("user_id", userIDStr),
			zap.String("action", action),
			zap.String("resource_type", resourceType),
			zap.String("resource_id", resourceID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	userIDStr := "nil"
	if userID != nil {
		userIDStr = userID.String()
	}
	s.logger.Debug("Audit log created with HTTP context",
		zap.String("audit_log_id", auditLog.ID.String()),
		zap.String("user_id", userIDStr),
		zap.String("action", action),
		zap.String("resource_type", resourceType),
	)

	return nil
}

// LogStateTransition logs a state transition for a resource (e.g., status change)
// userID can be nil for unauthenticated events
func (s *AuditLogService) LogStateTransition(
	ctx context.Context,
	userID *uuid.UUID,
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
// userID can be nil for unauthenticated events
func (s *AuditLogService) LogWithResult(
	ctx context.Context,
	userID *uuid.UUID,
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
