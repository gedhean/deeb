package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SchemaHistory represents an immutable schema version at a point in time
// It stores both metadata (as JSONB) and the raw SQL dump of the schema
type SchemaHistory struct {
	ID                 uuid.UUID       `json:"id"                   db:"id"`
	DatabaseInstanceID uuid.UUID       `json:"database_instance_id" db:"database_instance_id"`
	DatabaseID         uuid.UUID       `json:"database_id"          db:"database_id"`
	Metadata           json.RawMessage `json:"metadata"             db:"metadata"`
	RawDump            string          `json:"raw_dump"             db:"raw_dump"`
	CreatedAt          time.Time       `json:"created_at"           db:"created_at"`
}

// CreateSchemaHistoryRequest represents the request to create a new schema history entry
type CreateSchemaHistoryRequest struct {
	DatabaseInstanceID uuid.UUID       `json:"database_instance_id" validate:"required"`
	DatabaseID         uuid.UUID       `json:"database_id"          validate:"required"`
	Metadata           json.RawMessage `json:"metadata"             validate:"required"`
	RawDump            string          `json:"raw_dump"             validate:"required"`
}

// SchemaHistoryResponse represents the API response for a schema history entry
type SchemaHistoryResponse struct {
	ID                 uuid.UUID       `json:"id"`
	DatabaseInstanceID uuid.UUID       `json:"database_instance_id"`
	DatabaseID         uuid.UUID       `json:"database_id"`
	Metadata           json.RawMessage `json:"metadata"`
	RawDump            string          `json:"raw_dump"`
	CreatedAt          time.Time       `json:"created_at"`
}

// ToResponse converts a SchemaHistory model to a SchemaHistoryResponse
func (s *SchemaHistory) ToResponse() *SchemaHistoryResponse {
	return &SchemaHistoryResponse{
		ID:                 s.ID,
		DatabaseInstanceID: s.DatabaseInstanceID,
		DatabaseID:         s.DatabaseID,
		Metadata:           s.Metadata,
		RawDump:            s.RawDump,
		CreatedAt:          s.CreatedAt,
	}
}
