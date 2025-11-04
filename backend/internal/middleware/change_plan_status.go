package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/workflow"
)

// ChangePlanStatusTransitionMiddleware validates status transitions for Change Plan PUT requests
type ChangePlanStatusTransitionMiddleware struct {
	changePlanRepo repository.ChangePlanRepository
}

// NewChangePlanStatusTransitionMiddleware creates a new status transition validation middleware
func NewChangePlanStatusTransitionMiddleware(
	changePlanRepo repository.ChangePlanRepository,
) *ChangePlanStatusTransitionMiddleware {
	return &ChangePlanStatusTransitionMiddleware{
		changePlanRepo: changePlanRepo,
	}
}

// ValidateStatusTransition validates status transitions for Change Plan PUT requests
func (m *ChangePlanStatusTransitionMiddleware) ValidateStatusTransition(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Only validate PUT requests
		if c.Request().Method != http.MethodPut {
			return next(c)
		}

		// Parse change plan ID from URL
		changePlanIDStr := c.Param("id")
		changePlanID, err := uuid.Parse(changePlanIDStr)
		if err != nil {
			// Invalid ID format - let handler deal with it
			return next(c)
		}

		// Get existing change plan to check current status
		ctx := c.Request().Context()
		changePlan, err := m.changePlanRepo.GetByID(ctx, changePlanID)
		if err != nil {
			// Change plan not found - let handler deal with it
			return next(c)
		}

		// Read request body to check for status field
		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			// Failed to read body - let handler deal with it
			return next(c)
		}

		// Restore body for handler to read
		c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Try to parse JSON to check for status field
		var reqBody map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &reqBody); err != nil {
			// Invalid JSON - let handler deal with it
			return next(c)
		}

		// Check if status is being updated
		statusValue, ok := reqBody["status"]
		if !ok || statusValue == nil {
			// No status field - no transition to validate
			return next(c)
		}

		// Parse new status
		statusStr, ok := statusValue.(string)
		if !ok {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid status format",
			})
		}

		newStatus := models.ChangePlanStatus(statusStr)
		if !newStatus.IsValid() {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid status value",
			})
		}

		// Validate transition
		if err := workflow.ValidateTransition(changePlan.Status, newStatus); err != nil {
			// Check if it's a status transition error to return structured response
			if transitionErr, ok := err.(workflow.ErrInvalidStatusTransition); ok {
				return c.JSON(http.StatusBadRequest, map[string]interface{}{
					"error":   transitionErr.UserMessage(),
					"details": transitionErr.Details(),
				})
			}

			// Fallback for other error types
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": err.Error(),
				"details": map[string]interface{}{
					"current_status":   changePlan.Status,
					"requested_status": newStatus,
				},
			})
		}

		// Transition is valid - continue
		return next(c)
	}
}
