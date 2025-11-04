package workflow

import (
	"testing"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrInvalidStatusTransition_UserMessage(t *testing.T) {
	err := ErrInvalidStatusTransition{
		From:   models.ChangePlanStatusDraft,
		To:     models.ChangePlanStatusApproved,
		Reason: "test reason",
	}

	msg := err.UserMessage()
	require.NotEmpty(t, msg)
	assert.Contains(t, msg, "Cannot transition")
	assert.Contains(t, msg, "Draft")
	assert.Contains(t, msg, "Approved")
}

func TestErrInvalidStatusTransition_Details(t *testing.T) {
	tests := []struct {
		name           string
		err            ErrInvalidStatusTransition
		expectedFields []string
	}{
		{
			name: "error with valid statuses",
			err: ErrInvalidStatusTransition{
				From:   models.ChangePlanStatusDraft,
				To:     models.ChangePlanStatusApproved,
				Reason: "test reason",
			},
			expectedFields: []string{"current_status", "attempted_status", "reason", "valid_transitions", "is_terminal_status"},
		},
		{
			name: "terminal status error",
			err: ErrInvalidStatusTransition{
				From: models.ChangePlanStatusCompleted,
				To:   models.ChangePlanStatusOpen,
			},
			expectedFields: []string{"current_status", "attempted_status", "is_terminal_status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details := tt.err.Details()
			require.NotNil(t, details)

			for _, field := range tt.expectedFields {
				assert.Contains(t, details, field, "Details should contain %s", field)
			}

			// Verify required fields
			assert.Equal(t, string(tt.err.From), details["current_status"])
			assert.Equal(t, string(tt.err.To), details["attempted_status"])

			// Verify valid transitions for non-terminal statuses
			if !IsTerminalStatus(tt.err.From) {
				validTransitions, ok := details["valid_transitions"]
				assert.True(t, ok, "Non-terminal status should have valid_transitions")
				if ok {
					transitions, ok := validTransitions.([]string)
					assert.True(t, ok, "valid_transitions should be []string")
					if ok {
						assert.Greater(t, len(transitions), 0, "Non-terminal status should have valid transitions")
					}
				}
			}
		})
	}
}

func TestErrInvalidStatusTransition_ErrorMessages_ContainsValidTransitions(t *testing.T) {
	err := ValidateTransition(models.ChangePlanStatusDraft, models.ChangePlanStatusApproved)
	require.NotNil(t, err)

	transitionErr, ok := err.(ErrInvalidStatusTransition)
	require.True(t, ok, "Error should be ErrInvalidStatusTransition")

	msg := transitionErr.Error()
	assert.Contains(t, msg, "Draft")
	assert.Contains(t, msg, "Approved")
	assert.Contains(t, msg, "Valid transitions")
	assert.Contains(t, msg, "Open")
}

func TestErrInvalidStatusTransition_TerminalStatusMessage(t *testing.T) {
	err := ValidateTransition(models.ChangePlanStatusCompleted, models.ChangePlanStatusOpen)
	require.NotNil(t, err)

	transitionErr, ok := err.(ErrInvalidStatusTransition)
	require.True(t, ok, "Error should be ErrInvalidStatusTransition")

	msg := transitionErr.Error()
	assert.Contains(t, msg, "terminal state")
	assert.Contains(t, msg, "Completed")
	assert.Contains(t, msg, "cannot be changed")
}

