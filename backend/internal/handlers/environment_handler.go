package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/services"
	"go.uber.org/zap"
)

// EnvironmentHandler handles environment management endpoints
type EnvironmentHandler struct {
	envRepo      repository.EnvironmentRepository
	db           *sqlx.DB
	auditService *services.AuditLogService
	logger       *zap.Logger
}

// NewEnvironmentHandler creates a new environment handler
func NewEnvironmentHandler(
	envRepo repository.EnvironmentRepository,
	db *sqlx.DB,
	auditService *services.AuditLogService,
	logger *zap.Logger,
) *EnvironmentHandler {
	return &EnvironmentHandler{
		envRepo:      envRepo,
		db:           db,
		auditService: auditService,
		logger:       logger,
	}
}

// validatePolicy validates that a Policy exists and is SQL_REVIEW type
// This is a temporary implementation until PolicyRepository is created in P02M07
func (h *EnvironmentHandler) validatePolicy(ctx context.Context, policyID *uuid.UUID) error {
	if policyID == nil {
		return nil // Policy ID is optional
	}

	var policyType string
	query := `SELECT type FROM policies WHERE id = $1`
	err := h.db.QueryRowContext(ctx, query, *policyID).Scan(&policyType)
	if err != nil {
		if err == sql.ErrNoRows {
			return echo.NewHTTPError(http.StatusBadRequest, "Policy not found")
		}
		h.logger.Error("Failed to validate policy", zap.Error(err), zap.String("policy_id", policyID.String()))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate policy")
	}

	if policyType != string(models.PolicyTypeSQLReview) {
		return echo.NewHTTPError(http.StatusBadRequest, "Policy must be SQL_REVIEW type")
	}

	return nil
}

// CreateEnvironment handles POST /api/v1/environments - create environment (Admin only)
func (h *EnvironmentHandler) CreateEnvironment(c echo.Context) error {
	ctx := c.Request().Context()

	// Get authenticated user from context
	claims, ok := auth.GetUserFromEchoContext(c)
	if !ok || claims == nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Authentication required",
		})
	}

	// Parse request
	var req models.CreateEnvironmentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := models.ValidateCreateEnvironmentRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}

	// Validate Policy if provided
	if req.PolicyID != nil {
		if err := h.validatePolicy(ctx, req.PolicyID); err != nil {
			return err
		}
	}

	// Create environment model with created_by_id set to authenticated user
	environment := &models.Environment{
		Name:        req.Name,
		Description: req.Description,
		PolicyID:    req.PolicyID,
		CreatedByID: claims.UserID,
	}

	// Save to repository
	if err := h.envRepo.Create(ctx, environment); err != nil {
		h.logger.Error("Failed to create environment", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to create environment",
		})
	}

	h.logger.Info("Environment created successfully", zap.String("environment_id", environment.ID.String()))

	// Log audit entry
	if h.auditService != nil {
		detailsJSON, _ := json.Marshal(map[string]interface{}{
			"environment_name": environment.Name,
			"description":      environment.Description,
			"policy_id":        environment.PolicyID,
		})
		if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionCreate, services.ResourceTypeEnvironment, environment.ID, c.Request(), detailsJSON); err != nil {
			h.logger.Error("Failed to log audit entry for environment creation", zap.Error(err))
		}
	}

	return c.JSON(http.StatusCreated, environment.ToResponse())
}

// ListEnvironments handles GET /api/v1/environments - list environments (all authenticated users)
func (h *EnvironmentHandler) ListEnvironments(c echo.Context) error {
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

	// Fetch environments from repository
	environments, err := h.envRepo.List(ctx, limit, offset)
	if err != nil {
		h.logger.Error("Failed to list environments", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve environments",
		})
	}

	// Convert to response format
	responses := make([]*models.EnvironmentResponse, len(environments))
	for i, env := range environments {
		responses[i] = env.ToResponse()
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"environments": responses,
		"limit":        limit,
		"offset":       offset,
		"count":        len(responses),
	})
}

// GetEnvironment handles GET /api/v1/environments/:id - get environment by ID
func (h *EnvironmentHandler) GetEnvironment(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse environment ID from path parameter
	idParam := c.Param("id")
	environmentID, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid environment ID format",
		})
	}

	// Fetch environment from repository
	environment, err := h.envRepo.GetByID(ctx, environmentID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Environment not found",
			})
		}
		h.logger.Error("Failed to get environment", zap.Error(err), zap.String("environment_id", environmentID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve environment",
		})
	}

	return c.JSON(http.StatusOK, environment.ToResponse())
}

// UpdateEnvironment handles PUT /api/v1/environments/:id - update environment
func (h *EnvironmentHandler) UpdateEnvironment(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse environment ID from path parameter
	idParam := c.Param("id")
	environmentID, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid environment ID format",
		})
	}

	// First, check if policy_id field is present in the request
	var rawReq map[string]interface{}
	if err := c.Bind(&rawReq); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}
	_, policyFieldPresent := rawReq["policy_id"]

	// Parse update request into typed struct
	var req models.UpdateEnvironmentRequest
	reqJSON, _ := json.Marshal(rawReq)
	if err := json.Unmarshal(reqJSON, &req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if err := models.ValidateUpdateEnvironmentRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Validation failed",
			"details": err.Error(),
		})
	}

	// Validate Policy if provided and not explicitly set to null
	if policyFieldPresent {
		if err := h.validatePolicy(ctx, req.PolicyID.UUID); err != nil {
			return err
		}
	}

	// Fetch existing environment
	environment, err := h.envRepo.GetByID(ctx, environmentID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Environment not found",
			})
		}
		h.logger.Error(
			"Failed to get environment for update",
			zap.Error(err),
			zap.String("environment_id", environmentID.String()),
		)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve environment",
		})
	}

	// Apply updates
	if req.Name != nil {
		environment.Name = *req.Name
	}
	if req.Description != nil {
		environment.Description = req.Description
	}
	// Update policy if the field was explicitly provided (even if null)
	if policyFieldPresent {
		environment.PolicyID = req.PolicyID.UUID
	}

	// Update environment in repository
	if err := h.envRepo.Update(ctx, environment); err != nil {
		h.logger.Error("Failed to update environment", zap.Error(err), zap.String("environment_id", environmentID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to update environment",
		})
	}

	h.logger.Info("Environment updated successfully", zap.String("environment_id", environmentID.String()))

	// Log audit entry
	if h.auditService != nil {
		claims, ok := auth.GetUserFromEchoContext(c)
		if ok && claims != nil {
			detailsJSON, _ := json.Marshal(map[string]interface{}{
				"environment_name": environment.Name,
				"description":      environment.Description,
				"policy_id":        environment.PolicyID,
			})
			if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionUpdate, services.ResourceTypeEnvironment, environmentID, c.Request(), detailsJSON); err != nil {
				h.logger.Error("Failed to log audit entry for environment update", zap.Error(err))
			}
		}
	}

	return c.JSON(http.StatusOK, environment.ToResponse())
}

// DeleteEnvironment handles DELETE /api/v1/environments/:id - delete environment
func (h *EnvironmentHandler) DeleteEnvironment(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse environment ID from path parameter
	idParam := c.Param("id")
	environmentID, err := uuid.Parse(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid environment ID format",
		})
	}

	// Check if environment exists
	_, err = h.envRepo.GetByID(ctx, environmentID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"error": "Environment not found",
			})
		}
		h.logger.Error(
			"Failed to get environment for deletion",
			zap.Error(err),
			zap.String("environment_id", environmentID.String()),
		)
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to retrieve environment",
		})
	}

	// Delete environment
	if err := h.envRepo.Delete(ctx, environmentID); err != nil {
		h.logger.Error("Failed to delete environment", zap.Error(err), zap.String("environment_id", environmentID.String()))
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Failed to delete environment",
		})
	}

	h.logger.Info("Environment deleted successfully", zap.String("environment_id", environmentID.String()))

	// Log audit entry
	if h.auditService != nil {
		claims, ok := auth.GetUserFromEchoContext(c)
		if ok && claims != nil {
			detailsJSON, _ := json.Marshal(map[string]interface{}{
				"environment_id": environmentID.String(),
			})
			if err := h.auditService.LogWithHTTP(ctx, claims.UserID, services.ActionDelete, services.ResourceTypeEnvironment, environmentID, c.Request(), detailsJSON); err != nil {
				h.logger.Error("Failed to log audit entry for environment deletion", zap.Error(err))
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Environment deleted successfully",
	})
}
