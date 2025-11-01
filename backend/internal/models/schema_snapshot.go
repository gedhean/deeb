package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SnapshotType represents the type of schema snapshot
type SnapshotType string

const (
	SnapshotTypeBefore SnapshotType = "before"
	SnapshotTypeAfter  SnapshotType = "after"
)

// SchemaSnapshot represents a schema snapshot
type SchemaSnapshot struct {
	ID              uuid.UUID       `json:"id"               db:"id"`
	DatabaseID      uuid.UUID       `json:"database_id"      db:"database_id"`
	ChangePlanID    uuid.UUID       `json:"change_plan_id"    db:"change_plan_id"`
	SnapshotType    SnapshotType    `json:"snapshot_type"     db:"snapshot_type"`
	SnapshotData    json.RawMessage `json:"snapshot_data"     db:"snapshot_data"`
	SnapshotDataRaw *string         `json:"snapshot_data_raw,omitempty" db:"snapshot_data_raw"`
	CreatedAt       time.Time       `json:"created_at"        db:"created_at"`
}

// CreateSchemaSnapshotRequest represents the request to create a new schema snapshot
type CreateSchemaSnapshotRequest struct {
	DatabaseID      uuid.UUID       `json:"database_id"      validate:"required"`
	ChangePlanID    uuid.UUID       `json:"change_plan_id"   validate:"required"`
	SnapshotType    SnapshotType    `json:"snapshot_type"     validate:"required,oneof=before after"`
	SnapshotData    json.RawMessage `json:"snapshot_data"     validate:"required"`
	SnapshotDataRaw *string         `json:"snapshot_data_raw,omitempty"`
}

// SchemaSnapshotResponse represents a schema snapshot response
type SchemaSnapshotResponse struct {
	ID              uuid.UUID       `json:"id"`
	DatabaseID      uuid.UUID       `json:"database_id"`
	ChangePlanID    uuid.UUID       `json:"change_plan_id"`
	SnapshotType    SnapshotType    `json:"snapshot_type"`
	SnapshotData    json.RawMessage `json:"snapshot_data"`
	SnapshotDataRaw *string         `json:"snapshot_data_raw,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// ToResponse converts a SchemaSnapshot to SchemaSnapshotResponse
func (s *SchemaSnapshot) ToResponse() *SchemaSnapshotResponse {
	return &SchemaSnapshotResponse{
		ID:              s.ID,
		DatabaseID:      s.DatabaseID,
		ChangePlanID:    s.ChangePlanID,
		SnapshotType:    s.SnapshotType,
		SnapshotData:    s.SnapshotData,
		SnapshotDataRaw: s.SnapshotDataRaw,
		CreatedAt:       s.CreatedAt,
	}
}
