package models

import (
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           uuid.UUID       `json:"id"                   db:"id"`
	UserID       *uuid.UUID      `json:"user_id"              db:"user_id"` // Nullable for unauthenticated events
	Action       string          `json:"action"               db:"action"`
	ResourceType string          `json:"resource_type"        db:"resource_type"`
	ResourceID   uuid.UUID       `json:"resource_id"          db:"resource_id"`
	Details      json.RawMessage `json:"details,omitempty"    db:"details"`
	IPAddress    *net.IP         `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent    *string         `json:"user_agent,omitempty" db:"user_agent"`
	Timestamp    time.Time       `json:"timestamp"            db:"timestamp"`
}

// CreateAuditLogRequest represents the request to create a new audit log entry
type CreateAuditLogRequest struct {
	UserID       *uuid.UUID      `json:"user_id"` // Nullable for unauthenticated events
	Action       string          `json:"action"               validate:"required,max=100"`
	ResourceType string          `json:"resource_type"        validate:"required,max=100"`
	ResourceID   uuid.UUID       `json:"resource_id"          validate:"required"`
	Details      json.RawMessage `json:"details,omitempty"`
	IPAddress    *net.IP         `json:"ip_address,omitempty"`
	UserAgent    *string         `json:"user_agent,omitempty"`
}

// AuditLogResponse represents an audit log response
type AuditLogResponse struct {
	ID           uuid.UUID       `json:"id"`
	UserID       *uuid.UUID      `json:"user_id"` // Nullable for unauthenticated events
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   uuid.UUID       `json:"resource_id"`
	Details      json.RawMessage `json:"details,omitempty"`
	IPAddress    *net.IP         `json:"ip_address,omitempty"`
	UserAgent    *string         `json:"user_agent,omitempty"`
	Timestamp    time.Time       `json:"timestamp"`
}

// ToResponse converts an AuditLog to AuditLogResponse
func (a *AuditLog) ToResponse() *AuditLogResponse {
	return &AuditLogResponse{
		ID:           a.ID,
		UserID:       a.UserID,
		Action:       a.Action,
		ResourceType: a.ResourceType,
		ResourceID:   a.ResourceID,
		Details:      a.Details,
		IPAddress:    a.IPAddress,
		UserAgent:    a.UserAgent,
		Timestamp:    a.Timestamp,
	}
}
