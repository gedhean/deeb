package workflow

import (
	"fmt"
	"strings"

	"github.com/nsxbet/deeb/internal/models"
)

// ErrInvalidStatusTransition represents an invalid status transition error
type ErrInvalidStatusTransition struct {
	From   models.ChangePlanStatus
	To     models.ChangePlanStatus
	Reason string
}

// Error implements the error interface with a user-friendly message
func (e ErrInvalidStatusTransition) Error() string {
	var msg strings.Builder

	// Base message
	if e.From == "" || !e.From.IsValid() {
		msg.WriteString(fmt.Sprintf("Cannot transition to %s: source status is invalid", e.To))
	} else if e.To == "" || !e.To.IsValid() {
		msg.WriteString(fmt.Sprintf("Cannot transition from %s: target status %s is invalid", e.From, e.To))
	} else if IsTerminalStatus(e.From) {
		msg.WriteString(fmt.Sprintf("Cannot transition from %s to %s: %s is a terminal state", e.From, e.To, e.From))
		msg.WriteString(". Terminal states (Completed, Failed, Closed, Cancelled) cannot be changed.")
	} else {
		msg.WriteString(fmt.Sprintf("Cannot transition from %s to %s", e.From, e.To))

		// Get valid transitions to provide helpful guidance
		validTransitions := GetValidTransitions(e.From)
		if len(validTransitions) > 0 {
			statusStrings := make([]string, len(validTransitions))
			for i, status := range validTransitions {
				statusStrings[i] = string(status)
			}
			msg.WriteString(fmt.Sprintf(". Valid transitions from %s are: %s", e.From, strings.Join(statusStrings, ", ")))
		} else {
			msg.WriteString(fmt.Sprintf(". Status %s is a terminal state and cannot be changed", e.From))
		}
	}

	// Add reason if provided
	if e.Reason != "" {
		msg.WriteString(fmt.Sprintf(" (%s)", e.Reason))
	}

	return msg.String()
}

// UserMessage returns a user-friendly error message
func (e ErrInvalidStatusTransition) UserMessage() string {
	return e.Error()
}

// Details returns structured error details
func (e ErrInvalidStatusTransition) Details() map[string]interface{} {
	details := map[string]interface{}{
		"current_status":   string(e.From),
		"attempted_status": string(e.To),
	}

	if e.Reason != "" {
		details["reason"] = e.Reason
	}

	// Include valid transitions if available
	if e.From.IsValid() {
		validTransitions := GetValidTransitions(e.From)
		if len(validTransitions) > 0 {
			statusStrings := make([]string, len(validTransitions))
			for i, status := range validTransitions {
				statusStrings[i] = string(status)
			}
			details["valid_transitions"] = statusStrings
		}
	}

	// Include whether current status is terminal
	if e.From.IsValid() {
		details["is_terminal_status"] = IsTerminalStatus(e.From)
	}

	return details
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
		reason := ""
		if IsTerminalStatus(from) {
			reason = fmt.Sprintf("status %s is a terminal state", from)
		} else {
			validTransitions := GetValidTransitions(from)
			if len(validTransitions) > 0 {
				statusStrings := make([]string, len(validTransitions))
				for i, status := range validTransitions {
					statusStrings[i] = string(status)
				}
				reason = fmt.Sprintf("only transitions to %s are allowed from %s", strings.Join(statusStrings, ", "), from)
			} else {
				reason = fmt.Sprintf("no valid transitions from %s", from)
			}
		}

		return ErrInvalidStatusTransition{
			From:   from,
			To:     to,
			Reason: reason,
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
