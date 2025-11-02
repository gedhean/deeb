package workflow

import (
	"fmt"

	"github.com/nsxbet/deeb/internal/models"
)

// ErrInvalidStatusTransition represents an invalid status transition error
type ErrInvalidStatusTransition struct {
	From   models.ChangePlanStatus
	To     models.ChangePlanStatus
	Reason string
}

func (e ErrInvalidStatusTransition) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("invalid status transition from %s to %s: %s", e.From, e.To, e.Reason)
	}
	return fmt.Sprintf("invalid status transition from %s to %s", e.From, e.To)
}

// Valid status transitions according to Phase 3 requirements:
// Draft → Open
// Open → Approved
// Open → Closed
// Approved → InProgress
// InProgress → Completed
// InProgress → Failed

// validTransitions defines the allowed status transitions
var validTransitions = map[models.ChangePlanStatus][]models.ChangePlanStatus{
	models.ChangePlanStatusDraft: {
		models.ChangePlanStatusOpen,
		models.ChangePlanStatusCancelled,
	},
	models.ChangePlanStatusOpen: {
		models.ChangePlanStatusApproved,
		models.ChangePlanStatusClosed,
		models.ChangePlanStatusDraft, // Allow resubmission (Open → Draft → Open)
	},
	models.ChangePlanStatusApproved: {
		models.ChangePlanStatusInProgress,
		models.ChangePlanStatusClosed,
		models.ChangePlanStatusCancelled,
	},
	models.ChangePlanStatusInProgress: {
		models.ChangePlanStatusCompleted,
		models.ChangePlanStatusFailed,
	},
	models.ChangePlanStatusCompleted: {
		// Terminal state - no transitions allowed
	},
	models.ChangePlanStatusFailed: {
		// Terminal state - no transitions allowed
	},
	models.ChangePlanStatusClosed: {
		// Terminal state - no transitions allowed
	},
	models.ChangePlanStatusCancelled: {
		// Terminal state - no transitions allowed
	},
}

// CanTransition checks if a status transition is valid
func CanTransition(from, to models.ChangePlanStatus) bool {
	// Same status is always valid (no-op)
	if from == to {
		return true
	}

	// Check if transition is allowed
	allowedTransitions, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == to {
			return true
		}
	}

	return false
}

// ValidateTransition validates a status transition and returns an error if invalid
func ValidateTransition(from, to models.ChangePlanStatus) error {
	// Same status is always valid (no-op)
	if from == to {
		return nil
	}

	// Check if both statuses are valid
	if !from.IsValid() {
		return ErrInvalidStatusTransition{
			From:   from,
			To:     to,
			Reason: fmt.Sprintf("source status %s is invalid", from),
		}
	}

	if !to.IsValid() {
		return ErrInvalidStatusTransition{
			From:   from,
			To:     to,
			Reason: fmt.Sprintf("target status %s is invalid", to),
		}
	}

	// Check if transition is allowed
	if !CanTransition(from, to) {
		return ErrInvalidStatusTransition{
			From:   from,
			To:     to,
			Reason: fmt.Sprintf("transition from %s to %s is not allowed", from, to),
		}
	}

	return nil
}

// GetValidTransitions returns all valid transitions from a given status
func GetValidTransitions(from models.ChangePlanStatus) []models.ChangePlanStatus {
	transitions, exists := validTransitions[from]
	if !exists {
		return []models.ChangePlanStatus{}
	}
	return transitions
}

// IsTerminalStatus checks if a status is a terminal state (no further transitions allowed)
func IsTerminalStatus(status models.ChangePlanStatus) bool {
	transitions := GetValidTransitions(status)
	return len(transitions) == 0
}

// IsDraftStatus checks if the status is Draft
func IsDraftStatus(status models.ChangePlanStatus) bool {
	return status == models.ChangePlanStatusDraft
}

// IsOpenStatus checks if the status is Open
func IsOpenStatus(status models.ChangePlanStatus) bool {
	return status == models.ChangePlanStatusOpen
}

// IsApprovedStatus checks if the status is Approved
func IsApprovedStatus(status models.ChangePlanStatus) bool {
	return status == models.ChangePlanStatusApproved
}

// IsInProgressStatus checks if the status is InProgress
func IsInProgressStatus(status models.ChangePlanStatus) bool {
	return status == models.ChangePlanStatusInProgress
}

// IsCompletedStatus checks if the status is Completed
func IsCompletedStatus(status models.ChangePlanStatus) bool {
	return status == models.ChangePlanStatusCompleted
}

// IsFailedStatus checks if the status is Failed
func IsFailedStatus(status models.ChangePlanStatus) bool {
	return status == models.ChangePlanStatusFailed
}

// IsClosedStatus checks if the status is Closed
func IsClosedStatus(status models.ChangePlanStatus) bool {
	return status == models.ChangePlanStatusClosed
}

// IsCancelledStatus checks if the status is Cancelled
func IsCancelledStatus(status models.ChangePlanStatus) bool {
	return status == models.ChangePlanStatusCancelled
}

// CanBeModified checks if a Change Plan can be modified (only Draft status)
func CanBeModified(status models.ChangePlanStatus) bool {
	return status == models.ChangePlanStatusDraft
}

// CanBeDeleted checks if a Change Plan can be deleted (only Draft or Closed)
func CanBeDeleted(status models.ChangePlanStatus) bool {
	return status == models.ChangePlanStatusDraft || status == models.ChangePlanStatusClosed
}
