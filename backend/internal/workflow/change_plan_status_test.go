package workflow

import (
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
				"IsOpenStatus":        {IsOpenStatus, true},
				"IsApprovedStatus":    {IsApprovedStatus, false},
				"IsInProgressStatus":  {IsInProgressStatus, false},
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

