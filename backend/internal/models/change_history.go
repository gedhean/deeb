package models

import (
	"time"

	"github.com/google/uuid"
)

// ChangeHistory represents a single field change in a Change Plan's history
type ChangeHistory struct {
	ID           uuid.UUID `json:"id"                  db:"id"`
	ChangePlanID uuid.UUID `json:"change_plan_id"      db:"change_plan_id"`
	FieldName    string    `json:"field_name"          db:"field_name"`
	OldValue     *string   `json:"old_value,omitempty" db:"old_value"`
	NewValue     *string   `json:"new_value,omitempty" db:"new_value"`
	ChangedByID  uuid.UUID `json:"changed_by_id"       db:"changed_by_id"`
	ChangedAt    time.Time `json:"changed_at"          db:"changed_at"`
}

// ChangeHistoryResponse represents a change history response
type ChangeHistoryResponse struct {
	ID           uuid.UUID `json:"id"`
	ChangePlanID uuid.UUID `json:"change_plan_id"`
	FieldName    string    `json:"field_name"`
	OldValue     *string   `json:"old_value,omitempty"`
	NewValue     *string   `json:"new_value,omitempty"`
	ChangedByID  uuid.UUID `json:"changed_by_id"`
	ChangedAt    time.Time `json:"changed_at"`
}

// ToResponse converts a ChangeHistory to ChangeHistoryResponse
func (ch *ChangeHistory) ToResponse() *ChangeHistoryResponse {
	return &ChangeHistoryResponse{
		ID:           ch.ID,
		ChangePlanID: ch.ChangePlanID,
		FieldName:    ch.FieldName,
		OldValue:     ch.OldValue,
		NewValue:     ch.NewValue,
		ChangedByID:  ch.ChangedByID,
		ChangedAt:    ch.ChangedAt,
	}
}
