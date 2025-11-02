package workflow

import (
	"strings"
	"testing"

	"github.com/nsxbet/deeb/internal/models"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name     string
		from     models.ChangePlanStatus
		to       models.ChangePlanStatus
		expected bool
	}{
		// Valid transitions
		{"Draft to Open", models.ChangePlanStatusDraft, models.ChangePlanStatusOpen, true},
		{"Draft to Cancelled", models.ChangePlanStatusDraft, models.ChangePlanStatusCancelled, true},
		{"Open to Approved", models.ChangePlanStatusOpen, models.ChangePlanStatusApproved, true},
		{"Open to Closed", models.ChangePlanStatusOpen, models.ChangePlanStatusClosed, true},
		{"Open to Draft (resubmission)", models.ChangePlanStatusOpen, models.ChangePlanStatusDraft, true},
		{"Approved to InProgress", models.ChangePlanStatusApproved, models.ChangePlanStatusInProgress, true},
		{"Approved to Closed", models.ChangePlanStatusApproved, models.ChangePlanStatusClosed, true},
		{"Approved to Cancelled", models.ChangePlanStatusApproved, models.ChangePlanStatusCancelled, true},
		{"InProgress to Completed", models.ChangePlanStatusInProgress, models.ChangePlanStatusCompleted, true},
		{"InProgress to Failed", models.ChangePlanStatusInProgress, models.ChangePlanStatusFailed, true},
		{"Same status", models.ChangePlanStatusDraft, models.ChangePlanStatusDraft, true},

		// Invalid transitions
		{"Draft to Approved", models.ChangePlanStatusDraft, models.ChangePlanStatusApproved, false},
		{"Draft to InProgress", models.ChangePlanStatusDraft, models.ChangePlanStatusInProgress, false},
		{"Open to InProgress", models.ChangePlanStatusOpen, models.ChangePlanStatusInProgress, false},
		{"Approved to Completed", models.ChangePlanStatusApproved, models.ChangePlanStatusCompleted, false},
		{"Completed to Failed", models.ChangePlanStatusCompleted, models.ChangePlanStatusFailed, false},
		{"Failed to Completed", models.ChangePlanStatusFailed, models.ChangePlanStatusCompleted, false},
		{"Closed to Open", models.ChangePlanStatusClosed, models.ChangePlanStatusOpen, false},
		{"Cancelled to Draft", models.ChangePlanStatusCancelled, models.ChangePlanStatusDraft, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name        string
		from        models.ChangePlanStatus
		to          models.ChangePlanStatus
		expectError bool
	}{
		// Valid transitions
		{"Draft to Open", models.ChangePlanStatusDraft, models.ChangePlanStatusOpen, false},
		{"Open to Approved", models.ChangePlanStatusOpen, models.ChangePlanStatusApproved, false},
		{"Approved to InProgress", models.ChangePlanStatusApproved, models.ChangePlanStatusInProgress, false},
		{"InProgress to Completed", models.ChangePlanStatusInProgress, models.ChangePlanStatusCompleted, false},
		{"Same status", models.ChangePlanStatusDraft, models.ChangePlanStatusDraft, false},

		// Invalid transitions
		{"Draft to Approved", models.ChangePlanStatusDraft, models.ChangePlanStatusApproved, true},
		{"Completed to Failed", models.ChangePlanStatusCompleted, models.ChangePlanStatusFailed, true},
		{"Closed to Open", models.ChangePlanStatusClosed, models.ChangePlanStatusOpen, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.from, tt.to)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateTransition(%s, %s) error = %v, expectError %v", tt.from, tt.to, err, tt.expectError)
			}
		})
	}
}

func TestGetValidTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     models.ChangePlanStatus
		expected int // expected number of valid transitions
	}{
		{"Draft", models.ChangePlanStatusDraft, 2},
		{"Open", models.ChangePlanStatusOpen, 3},
		{"Approved", models.ChangePlanStatusApproved, 3},
		{"InProgress", models.ChangePlanStatusInProgress, 2},
		{"Completed", models.ChangePlanStatusCompleted, 0},
		{"Failed", models.ChangePlanStatusFailed, 0},
		{"Closed", models.ChangePlanStatusClosed, 0},
		{"Cancelled", models.ChangePlanStatusCancelled, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetValidTransitions(tt.from)
			if len(result) != tt.expected {
				t.Errorf("GetValidTransitions(%s) = %d transitions, want %d", tt.from, len(result), tt.expected)
			}
		})
	}
}

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   models.ChangePlanStatus
		expected bool
	}{
		{"Draft", models.ChangePlanStatusDraft, false},
		{"Open", models.ChangePlanStatusOpen, false},
		{"Approved", models.ChangePlanStatusApproved, false},
		{"InProgress", models.ChangePlanStatusInProgress, false},
		{"Completed", models.ChangePlanStatusCompleted, true},
		{"Failed", models.ChangePlanStatusFailed, true},
		{"Closed", models.ChangePlanStatusClosed, true},
		{"Cancelled", models.ChangePlanStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTerminalStatus(tt.status)
			if result != tt.expected {
				t.Errorf("IsTerminalStatus(%s) = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestCanBeModified(t *testing.T) {
	tests := []struct {
		name     string
		status   models.ChangePlanStatus
		expected bool
	}{
		{"Draft", models.ChangePlanStatusDraft, true},
		{"Open", models.ChangePlanStatusOpen, false},
		{"Approved", models.ChangePlanStatusApproved, false},
		{"InProgress", models.ChangePlanStatusInProgress, false},
		{"Completed", models.ChangePlanStatusCompleted, false},
		{"Failed", models.ChangePlanStatusFailed, false},
		{"Closed", models.ChangePlanStatusClosed, false},
		{"Cancelled", models.ChangePlanStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanBeModified(tt.status)
			if result != tt.expected {
				t.Errorf("CanBeModified(%s) = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestCanBeDeleted(t *testing.T) {
	tests := []struct {
		name     string
		status   models.ChangePlanStatus
		expected bool
	}{
		{"Draft", models.ChangePlanStatusDraft, true},
		{"Closed", models.ChangePlanStatusClosed, true},
		{"Open", models.ChangePlanStatusOpen, false},
		{"Approved", models.ChangePlanStatusApproved, false},
		{"InProgress", models.ChangePlanStatusInProgress, false},
		{"Completed", models.ChangePlanStatusCompleted, false},
		{"Failed", models.ChangePlanStatusFailed, false},
		{"Cancelled", models.ChangePlanStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanBeDeleted(tt.status)
			if result != tt.expected {
				t.Errorf("CanBeDeleted(%s) = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestStatusHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		status   models.ChangePlanStatus
		checkers map[string]struct {
			checker func(models.ChangePlanStatus) bool
			expect  bool
		}
	}{
		{
			"Draft",
			models.ChangePlanStatusDraft,
			map[string]struct {
				checker func(models.ChangePlanStatus) bool
				expect  bool
			}{
				"IsDraftStatus":      {IsDraftStatus, true},
				"IsOpenStatus":       {IsOpenStatus, false},
				"IsApprovedStatus":   {IsApprovedStatus, false},
				"IsInProgressStatus": {IsInProgressStatus, false},
			},
		},
		{
			"Open",
			models.ChangePlanStatusOpen,
			map[string]struct {
				checker func(models.ChangePlanStatus) bool
				expect  bool
			}{
				"IsDraftStatus":      {IsDraftStatus, false},
				"IsOpenStatus":       {IsOpenStatus, true},
				"IsApprovedStatus":   {IsApprovedStatus, false},
				"IsInProgressStatus": {IsInProgressStatus, false},
			},
		},
		{
			"Approved",
			models.ChangePlanStatusApproved,
			map[string]struct {
				checker func(models.ChangePlanStatus) bool
				expect  bool
			}{
				"IsDraftStatus":      {IsDraftStatus, false},
				"IsOpenStatus":       {IsOpenStatus, false},
				"IsApprovedStatus":   {IsApprovedStatus, true},
				"IsInProgressStatus": {IsInProgressStatus, false},
			},
		},
		{
			"InProgress",
			models.ChangePlanStatusInProgress,
			map[string]struct {
				checker func(models.ChangePlanStatus) bool
				expect  bool
			}{
				"IsDraftStatus":      {IsDraftStatus, false},
				"IsOpenStatus":       {IsOpenStatus, false},
				"IsApprovedStatus":   {IsApprovedStatus, false},
				"IsInProgressStatus": {IsInProgressStatus, true},
			},
		},
		{
			"Completed",
			models.ChangePlanStatusCompleted,
			map[string]struct {
				checker func(models.ChangePlanStatus) bool
				expect  bool
			}{
				"IsCompletedStatus": {IsCompletedStatus, true},
				"IsFailedStatus":    {IsFailedStatus, false},
				"IsClosedStatus":    {IsClosedStatus, false},
				"IsCancelledStatus": {IsCancelledStatus, false},
			},
		},
		{
			"Failed",
			models.ChangePlanStatusFailed,
			map[string]struct {
				checker func(models.ChangePlanStatus) bool
				expect  bool
			}{
				"IsCompletedStatus": {IsCompletedStatus, false},
				"IsFailedStatus":    {IsFailedStatus, true},
				"IsClosedStatus":    {IsClosedStatus, false},
				"IsCancelledStatus": {IsCancelledStatus, false},
			},
		},
		{
			"Closed",
			models.ChangePlanStatusClosed,
			map[string]struct {
				checker func(models.ChangePlanStatus) bool
				expect  bool
			}{
				"IsCompletedStatus": {IsCompletedStatus, false},
				"IsFailedStatus":    {IsFailedStatus, false},
				"IsClosedStatus":    {IsClosedStatus, true},
				"IsCancelledStatus": {IsCancelledStatus, false},
			},
		},
		{
			"Cancelled",
			models.ChangePlanStatusCancelled,
			map[string]struct {
				checker func(models.ChangePlanStatus) bool
				expect  bool
			}{
				"IsCompletedStatus": {IsCompletedStatus, false},
				"IsFailedStatus":    {IsFailedStatus, false},
				"IsClosedStatus":    {IsClosedStatus, false},
				"IsCancelledStatus": {IsCancelledStatus, true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for checkerName, tc := range tt.checkers {
				result := tc.checker(tt.status)
				if result != tc.expect {
					t.Errorf("%s(%s) = %v, want %v", checkerName, tt.status, result, tc.expect)
				}
			}
		})
	}
}

func TestValidateTransition_InvalidStatusValues(t *testing.T) {
	tests := []struct {
		name        string
		from        models.ChangePlanStatus
		to          models.ChangePlanStatus
		expectError bool
		errorReason string
	}{
		{
			name:        "invalid source status",
			from:        models.ChangePlanStatus("invalid"),
			to:          models.ChangePlanStatusOpen,
			expectError: true,
			errorReason: "source status",
		},
		{
			name:        "invalid target status",
			from:        models.ChangePlanStatusDraft,
			to:          models.ChangePlanStatus("invalid"),
			expectError: true,
			errorReason: "target status",
		},
		{
			name:        "both statuses invalid",
			from:        models.ChangePlanStatus("invalid1"),
			to:          models.ChangePlanStatus("invalid2"),
			expectError: true,
			errorReason: "source status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.from, tt.to)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateTransition(%s, %s) error = %v, expectError %v", tt.from, tt.to, err, tt.expectError)
				return
			}
			if tt.expectError && err != nil {
				if err.Error() == "" {
					t.Errorf("ValidateTransition(%s, %s) error message is empty", tt.from, tt.to)
				}
				// Check error type
				if _, ok := err.(ErrInvalidStatusTransition); !ok {
					t.Errorf("ValidateTransition(%s, %s) error type = %T, want ErrInvalidStatusTransition", tt.from, tt.to, err)
				}
			}
		})
	}
}

func TestValidateTransition_ErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		from          models.ChangePlanStatus
		to            models.ChangePlanStatus
		expectedInMsg string
	}{
		{
			name:          "invalid transition error message",
			from:          models.ChangePlanStatusDraft,
			to:            models.ChangePlanStatusApproved,
			expectedInMsg: "invalid status transition",
		},
		{
			name:          "invalid source status error message",
			from:          models.ChangePlanStatus("invalid"),
			to:            models.ChangePlanStatusOpen,
			expectedInMsg: "source status",
		},
		{
			name:          "invalid target status error message",
			from:          models.ChangePlanStatusDraft,
			to:            models.ChangePlanStatus("invalid"),
			expectedInMsg: "target status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.from, tt.to)
			if err == nil {
				t.Errorf("ValidateTransition(%s, %s) expected error, got nil", tt.from, tt.to)
				return
			}
			if err.Error() == "" {
				t.Errorf("ValidateTransition(%s, %s) error message is empty", tt.from, tt.to)
				return
			}
			if !strings.Contains(err.Error(), tt.expectedInMsg) {
				t.Errorf(
					"ValidateTransition(%s, %s) error message = %q, want to contain %q",
					tt.from,
					tt.to,
					err.Error(),
					tt.expectedInMsg,
				)
			}
		})
	}
}

func TestErrInvalidStatusTransition_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      ErrInvalidStatusTransition
		expected string
	}{
		{
			name: "error with reason",
			err: ErrInvalidStatusTransition{
				From:   models.ChangePlanStatusDraft,
				To:     models.ChangePlanStatusApproved,
				Reason: "transition not allowed",
			},
			expected: "invalid status transition from Draft to Approved: transition not allowed",
		},
		{
			name: "error without reason",
			err: ErrInvalidStatusTransition{
				From: models.ChangePlanStatusDraft,
				To:   models.ChangePlanStatusApproved,
			},
			expected: "invalid status transition from Draft to Approved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if msg != tt.expected {
				t.Errorf("ErrInvalidStatusTransition.Error() = %q, want %q", msg, tt.expected)
			}
		})
	}
}

func TestCanTransition_AllStatusCombinations(t *testing.T) {
	allStatuses := []models.ChangePlanStatus{
		models.ChangePlanStatusDraft,
		models.ChangePlanStatusOpen,
		models.ChangePlanStatusApproved,
		models.ChangePlanStatusInProgress,
		models.ChangePlanStatusCompleted,
		models.ChangePlanStatusFailed,
		models.ChangePlanStatusClosed,
		models.ChangePlanStatusCancelled,
	}

	// Test all combinations
	for _, from := range allStatuses {
		for _, to := range allStatuses {
			if from == to {
				// Same status should always be valid
				if !CanTransition(from, to) {
					t.Errorf("CanTransition(%s, %s) = false, want true (same status)", from, to)
				}
				continue
			}

			// Check if transition is valid
			result := CanTransition(from, to)
			validTransitions := GetValidTransitions(from)

			expected := false
			for _, valid := range validTransitions {
				if valid == to {
					expected = true
					break
				}
			}

			if result != expected {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", from, to, result, expected)
			}
		}
	}
}

func TestGetValidTransitions_AllStatuses(t *testing.T) {
	allStatuses := []models.ChangePlanStatus{
		models.ChangePlanStatusDraft,
		models.ChangePlanStatusOpen,
		models.ChangePlanStatusApproved,
		models.ChangePlanStatusInProgress,
		models.ChangePlanStatusCompleted,
		models.ChangePlanStatusFailed,
		models.ChangePlanStatusClosed,
		models.ChangePlanStatusCancelled,
	}

	for _, status := range allStatuses {
		t.Run(string(status), func(t *testing.T) {
			transitions := GetValidTransitions(status)
			// Should always return a slice (may be empty for terminal statuses)
			if transitions == nil {
				t.Errorf("GetValidTransitions(%s) = nil, want []", status)
			}

			// Verify all returned transitions are valid
			for _, transition := range transitions {
				if !transition.IsValid() {
					t.Errorf("GetValidTransitions(%s) returned invalid status: %s", status, transition)
				}
			}
		})
	}
}

func TestIsTerminalStatus_AllStatuses(t *testing.T) {
	terminalStatuses := []models.ChangePlanStatus{
		models.ChangePlanStatusCompleted,
		models.ChangePlanStatusFailed,
		models.ChangePlanStatusClosed,
		models.ChangePlanStatusCancelled,
	}

	nonTerminalStatuses := []models.ChangePlanStatus{
		models.ChangePlanStatusDraft,
		models.ChangePlanStatusOpen,
		models.ChangePlanStatusApproved,
		models.ChangePlanStatusInProgress,
	}

	for _, status := range terminalStatuses {
		t.Run(string(status)+" terminal", func(t *testing.T) {
			if !IsTerminalStatus(status) {
				t.Errorf("IsTerminalStatus(%s) = false, want true", status)
			}
		})
	}

	for _, status := range nonTerminalStatuses {
		t.Run(string(status)+" non-terminal", func(t *testing.T) {
			if IsTerminalStatus(status) {
				t.Errorf("IsTerminalStatus(%s) = true, want false", status)
			}
		})
	}
}

func TestCanBeModified_AllStatuses(t *testing.T) {
	modifiableStatuses := []models.ChangePlanStatus{
		models.ChangePlanStatusDraft,
	}

	nonModifiableStatuses := []models.ChangePlanStatus{
		models.ChangePlanStatusOpen,
		models.ChangePlanStatusApproved,
		models.ChangePlanStatusInProgress,
		models.ChangePlanStatusCompleted,
		models.ChangePlanStatusFailed,
		models.ChangePlanStatusClosed,
		models.ChangePlanStatusCancelled,
	}

	for _, status := range modifiableStatuses {
		t.Run(string(status)+" modifiable", func(t *testing.T) {
			if !CanBeModified(status) {
				t.Errorf("CanBeModified(%s) = false, want true", status)
			}
		})
	}

	for _, status := range nonModifiableStatuses {
		t.Run(string(status)+" non-modifiable", func(t *testing.T) {
			if CanBeModified(status) {
				t.Errorf("CanBeModified(%s) = true, want false", status)
			}
		})
	}
}

func TestCanBeDeleted_AllStatuses(t *testing.T) {
	deletableStatuses := []models.ChangePlanStatus{
		models.ChangePlanStatusDraft,
		models.ChangePlanStatusClosed,
	}

	nonDeletableStatuses := []models.ChangePlanStatus{
		models.ChangePlanStatusOpen,
		models.ChangePlanStatusApproved,
		models.ChangePlanStatusInProgress,
		models.ChangePlanStatusCompleted,
		models.ChangePlanStatusFailed,
		models.ChangePlanStatusCancelled,
	}

	for _, status := range deletableStatuses {
		t.Run(string(status)+" deletable", func(t *testing.T) {
			if !CanBeDeleted(status) {
				t.Errorf("CanBeDeleted(%s) = false, want true", status)
			}
		})
	}

	for _, status := range nonDeletableStatuses {
		t.Run(string(status)+" non-deletable", func(t *testing.T) {
			if CanBeDeleted(status) {
				t.Errorf("CanBeDeleted(%s) = true, want false", status)
			}
		})
	}
}

func TestCanTransition_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		from     models.ChangePlanStatus
		to       models.ChangePlanStatus
		expected bool
	}{
		{"empty string to Draft", models.ChangePlanStatus(""), models.ChangePlanStatusDraft, false},
		{"Draft to empty string", models.ChangePlanStatusDraft, models.ChangePlanStatus(""), false},
		{"empty string to empty string", models.ChangePlanStatus(""), models.ChangePlanStatus(""), true}, // Same status
		{"unknown status", models.ChangePlanStatus("Unknown"), models.ChangePlanStatusDraft, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CanTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestValidateTransition_AllValidTransitions(t *testing.T) {
	validTransitionTests := []struct {
		name string
		from models.ChangePlanStatus
		to   models.ChangePlanStatus
	}{
		{"Draft to Open", models.ChangePlanStatusDraft, models.ChangePlanStatusOpen},
		{"Draft to Cancelled", models.ChangePlanStatusDraft, models.ChangePlanStatusCancelled},
		{"Open to Approved", models.ChangePlanStatusOpen, models.ChangePlanStatusApproved},
		{"Open to Closed", models.ChangePlanStatusOpen, models.ChangePlanStatusClosed},
		{"Open to Draft", models.ChangePlanStatusOpen, models.ChangePlanStatusDraft},
		{"Approved to InProgress", models.ChangePlanStatusApproved, models.ChangePlanStatusInProgress},
		{"Approved to Closed", models.ChangePlanStatusApproved, models.ChangePlanStatusClosed},
		{"Approved to Cancelled", models.ChangePlanStatusApproved, models.ChangePlanStatusCancelled},
		{"InProgress to Completed", models.ChangePlanStatusInProgress, models.ChangePlanStatusCompleted},
		{"InProgress to Failed", models.ChangePlanStatusInProgress, models.ChangePlanStatusFailed},
	}

	for _, tt := range validTransitionTests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.from, tt.to)
			if err != nil {
				t.Errorf("ValidateTransition(%s, %s) = %v, want nil", tt.from, tt.to, err)
			}
		})
	}
}

func TestValidateTransition_AllInvalidTransitions(t *testing.T) {
	invalidTransitionTests := []struct {
		name string
		from models.ChangePlanStatus
		to   models.ChangePlanStatus
	}{
		{"Draft to Approved", models.ChangePlanStatusDraft, models.ChangePlanStatusApproved},
		{"Draft to InProgress", models.ChangePlanStatusDraft, models.ChangePlanStatusInProgress},
		{"Draft to Completed", models.ChangePlanStatusDraft, models.ChangePlanStatusCompleted},
		{"Open to InProgress", models.ChangePlanStatusOpen, models.ChangePlanStatusInProgress},
		{"Open to Completed", models.ChangePlanStatusOpen, models.ChangePlanStatusCompleted},
		{"Approved to Completed", models.ChangePlanStatusApproved, models.ChangePlanStatusCompleted},
		{"Approved to Failed", models.ChangePlanStatusApproved, models.ChangePlanStatusFailed},
		{"Completed to Failed", models.ChangePlanStatusCompleted, models.ChangePlanStatusFailed},
		{"Failed to Completed", models.ChangePlanStatusFailed, models.ChangePlanStatusCompleted},
		{"Completed to Draft", models.ChangePlanStatusCompleted, models.ChangePlanStatusDraft},
		{"Failed to Draft", models.ChangePlanStatusFailed, models.ChangePlanStatusDraft},
		{"Closed to Open", models.ChangePlanStatusClosed, models.ChangePlanStatusOpen},
		{"Closed to Draft", models.ChangePlanStatusClosed, models.ChangePlanStatusDraft},
		{"Cancelled to Draft", models.ChangePlanStatusCancelled, models.ChangePlanStatusDraft},
		{"Cancelled to Open", models.ChangePlanStatusCancelled, models.ChangePlanStatusOpen},
	}

	for _, tt := range invalidTransitionTests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.from, tt.to)
			if err == nil {
				t.Errorf("ValidateTransition(%s, %s) = nil, want error", tt.from, tt.to)
			}
			// Verify error type
			if _, ok := err.(ErrInvalidStatusTransition); !ok {
				t.Errorf("ValidateTransition(%s, %s) error type = %T, want ErrInvalidStatusTransition", tt.from, tt.to, err)
			}
		})
	}
}
