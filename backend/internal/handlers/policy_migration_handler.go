package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/services"
	"go.uber.org/zap"
)

// PolicyMigrationHandler handles policy migration operations
type PolicyMigrationHandler struct {
	migrationService *services.PolicyMigrationService
	logger           *zap.Logger
}

// NewPolicyMigrationHandler creates a new policy migration handler
func NewPolicyMigrationHandler(
	migrationService *services.PolicyMigrationService,
	logger *zap.Logger,
) *PolicyMigrationHandler {
	return &PolicyMigrationHandler{
		migrationService: migrationService,
		logger:           logger,
	}
}

// GetMigrationStatus handles GET /api/v1/policies/migration/status
// Returns the current migration status for all policies
func (h *PolicyMigrationHandler) GetMigrationStatus(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Only organization admins can view migration status
	// TODO: Add RBAC check

	status, err := h.migrationService.GetMigrationStatus(ctx)
	if err != nil {
		h.logger.Error("Failed to get migration status", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to get migration status",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": status,
	})
}

// MigratePolicy handles POST /api/v1/policies/:id/migrate
// Migrates a single policy to the latest schema version
func (h *PolicyMigrationHandler) MigratePolicy(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Only organization admins can migrate policies
	// TODO: Add RBAC check

	// Parse policy ID
	policyIDStr := c.Param("id")
	policyID, err := uuid.Parse(policyIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid policy ID format",
		})
	}

	result, err := h.migrationService.MigratePolicyToV2(ctx, policyID)
	if err != nil {
		h.logger.Error("Failed to migrate policy",
			zap.String("policy_id", policyID.String()),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to migrate policy: " + err.Error(),
		})
	}

	if !result.Success {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":  result.Error,
			"result": result,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"result": result,
	})
}

// MigrateAllPolicies handles POST /api/v1/policies/migration/migrate-all
// Migrates all policies to the latest schema version
func (h *PolicyMigrationHandler) MigrateAllPolicies(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Only organization admins can perform batch migrations
	// TODO: Add RBAC check

	result, err := h.migrationService.MigrateAllPolicies(ctx)
	if err != nil {
		h.logger.Error("Failed to migrate all policies", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to migrate all policies: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"result": result,
	})
}
