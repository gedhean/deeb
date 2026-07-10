package handlers

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/review"
	"go.uber.org/zap"
)

// PolicyHandler handles policy management endpoints
type PolicyHandler struct {
	policyRepo repository.PolicyRepository
	envRepo    repository.EnvironmentRepository
	logger     *zap.Logger
}

// NewPolicyHandler creates a new policy handler
func NewPolicyHandler(
	policyRepo repository.PolicyRepository,
	envRepo repository.EnvironmentRepository,
	logger *zap.Logger,
) *PolicyHandler {
	return &PolicyHandler{
		policyRepo: policyRepo,
		envRepo:    envRepo,
		logger:     logger,
	}
}

// CreatePolicy handles POST /api/v1/policies - create policy (Admin only)
func (h *PolicyHandler) CreatePolicy(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse request
	var req models.CreatePolicyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := models.ValidateCreatePolicyRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}

	// Create policy model with created_by_id set to authenticated user
	policy := &models.Policy{
		Name:         req.Name,
		Description:  req.Description,
		Type:         req.Type,
		PolicyConfig: req.PolicyConfig,
		CreatedByID:  claims.UserID,
	}

	// Save to repository
	if err := h.policyRepo.Create(ctx, policy); err != nil {
		h.logger.Error("Failed to create policy", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create policy",
		})
	}

	h.logger.Info("Policy created successfully", zap.String("policy_id", policy.ID.String()))
	return c.JSON(http.StatusCreated, policy.ToResponse())
}

// ListPolicies handles GET /api/v1/policies - list policies (all authenticated users)
func (h *PolicyHandler) ListPolicies(c echo.Context) error {
	ctx := c.Request().Context()

	// Get pagination parameters with defaults
	limit := 100
	offset := 0

	// Parse query parameters if provided
	if limitParam := c.QueryParam("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	if offsetParam := c.QueryParam("offset"); offsetParam != "" {
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Build filters from query parameters
	var filters repository.PolicyFilters
	filters.Search = c.QueryParam("search")

	// Parse type filter if provided
	if typeParam := c.QueryParam("type"); typeParam != "" {
		policyType := models.PolicyType(typeParam)
		// Validate type
		if policyType != models.PolicyTypeSQLReview {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"error": "Invalid policy type. Must be SQL_REVIEW",
			})
		}
		filters.Type = policyType
	}

	createdAfter, err := parseTimeParam(c, "created_after")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid created_after (must be RFC3339)",
		})
	}
	createdBefore, err := parseTimeParam(c, "created_before")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid created_before (must be RFC3339)",
		})
	}
	filters.CreatedAfter = createdAfter
	filters.CreatedBefore = createdBefore

	policies, err := h.policyRepo.List(ctx, filters, limit, offset)
	var totalCount int64
	if err == nil {
		totalCount, err = h.policyRepo.Count(ctx, filters)
	}

	if err != nil {
		h.logger.Error("Failed to list policies", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve policies",
		})
	}

	// Convert to response format
	responses := make([]*models.PolicyResponse, len(policies))
	for i, policy := range policies {
		responses[i] = policy.ToResponse()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"policies": responses,
		"limit":    limit,
		"offset":   offset,
		"total":    totalCount,
	})
}

// GetPolicy handles GET /api/v1/policies/:id - get policy by ID
func (h *PolicyHandler) GetPolicy(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse policy ID from path parameter
	idParam := c.Param("id")
	policyID, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid policy ID format",
		})
	}

	// Fetch policy from repository
	policy, err := h.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Policy not found",
			})
		}
		h.logger.Error("Failed to get policy", zap.Error(err), zap.String("policy_id", policyID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve policy",
		})
	}

	return c.JSON(http.StatusOK, policy.ToResponse())
}

// UpdatePolicy handles PUT /api/v1/policies/:id - update policy
func (h *PolicyHandler) UpdatePolicy(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse policy ID from path parameter
	idParam := c.Param("id")
	policyID, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid policy ID format",
		})
	}

	// Parse update request
	var req models.UpdatePolicyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := models.ValidateUpdatePolicyRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}

	// Fetch existing policy
	policy, err := h.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Policy not found",
			})
		}
		h.logger.Error("Failed to get policy for update", zap.Error(err), zap.String("policy_id", policyID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve policy",
		})
	}

	// Apply updates
	if req.Name != nil {
		policy.Name = *req.Name
	}
	if req.Description != nil {
		policy.Description = req.Description
	}
	if req.PolicyConfig != nil {
		policy.PolicyConfig = *req.PolicyConfig
	}

	// Update policy in repository
	if err := h.policyRepo.Update(ctx, policy); err != nil {
		h.logger.Error("Failed to update policy", zap.Error(err), zap.String("policy_id", policyID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update policy",
		})
	}

	h.logger.Info("Policy updated successfully", zap.String("policy_id", policyID.String()))
	return c.JSON(http.StatusOK, policy.ToResponse())
}

// DeletePolicy handles DELETE /api/v1/policies/:id - delete policy
func (h *PolicyHandler) DeletePolicy(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse policy ID from path parameter
	idParam := c.Param("id")
	policyID, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid policy ID format",
		})
	}

	// Check if policy exists
	_, err = h.policyRepo.GetByID(ctx, policyID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Policy not found",
			})
		}
		h.logger.Error("Failed to get policy for deletion", zap.Error(err), zap.String("policy_id", policyID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve policy",
		})
	}

	// Check if policy is referenced by any environments
	// We need to check all environments to see if any reference this policy
	// Since we don't have a direct method, we'll list all environments and check
	environments, err := h.envRepo.List(ctx, repository.EnvironmentFilters{}, 1000, 0) // Get a large number to check all
	if err != nil {
		h.logger.Error("Failed to check environments for policy", zap.Error(err), zap.String("policy_id", policyID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to check policy dependencies",
		})
	}

	for _, env := range environments {
		if env.PolicyID != nil && *env.PolicyID == policyID {
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"error": "Cannot delete policy: policy is referenced by one or more environments. Please remove or reassign the policy from environments first.",
			})
		}
	}

	// Delete policy
	if err := h.policyRepo.Delete(ctx, policyID); err != nil {
		h.logger.Error("Failed to delete policy", zap.Error(err), zap.String("policy_id", policyID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to delete policy",
		})
	}

	h.logger.Info("Policy deleted successfully", zap.String("policy_id", policyID.String()))
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Policy deleted successfully",
	})
}

// GetSQLReviewRules handles GET /api/v1/policies/sql-review/rules - get available SQL review rules
func (h *PolicyHandler) GetSQLReviewRules(c echo.Context) error {
	// Get all available rules organized by engine and category
	allRules := review.GetAvailableRules()

	// Convert to response format
	response := make(map[string]interface{})
	for engine, categories := range allRules {
		engineRules := make(map[string][]review.RuleMetadata)
		for category, rules := range categories {
			engineRules[string(category)] = rules
		}
		response[string(engine)] = engineRules
	}

	return c.JSON(http.StatusOK, response)
}
