package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/repository"
	"go.uber.org/zap"
)

// ChangelogHandler handles HTTP requests for changelog operations
type ChangelogHandler struct {
	changelogRepo     repository.ChangelogRepository
	schemaHistoryRepo repository.SchemaHistoryRepository
	changePlanRepo    repository.ChangePlanRepository
	logger            *zap.Logger
}

// NewChangelogHandler creates a new changelog handler
func NewChangelogHandler(
	changelogRepo repository.ChangelogRepository,
	schemaHistoryRepo repository.SchemaHistoryRepository,
	changePlanRepo repository.ChangePlanRepository,
	logger *zap.Logger,
) *ChangelogHandler {
	return &ChangelogHandler{
		changelogRepo:     changelogRepo,
		schemaHistoryRepo: schemaHistoryRepo,
		changePlanRepo:    changePlanRepo,
		logger:            logger,
	}
}

// ChangelogListItem represents a changelog entry in the list view
type ChangelogListItem struct {
	ID              string `json:"id"`
	DatabaseID      string `json:"database_id"`
	ChangePlanID    string `json:"change_plan_id"`
	ChangePlanTitle string `json:"change_plan_title"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
}

// ChangePlanSummary represents a summary of a change plan
type ChangePlanSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ChangelogDetailResponse represents detailed changelog information
type ChangelogDetailResponse struct {
	Changelog     interface{}       `json:"changelog"`
	ChangePlan    ChangePlanSummary `json:"change_plan"`
	SchemaHistory interface{}       `json:"schema_history"`
}

// ListDatabaseChangelog returns all changelog entries for a specific database
// GET /api/v1/databases/:id/changelog
func (h *ChangelogHandler) ListDatabaseChangelog(c echo.Context) error {
	databaseIDStr := c.Param("id")
	databaseID, err := uuid.Parse(databaseIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid database ID"})
	}

	// Get all changelog entries for this database
	changelogs, err := h.changelogRepo.GetByDatabaseID(c.Request().Context(), databaseID)
	if err != nil {
		h.logger.Error("Failed to get changelog entries",
			zap.Error(err),
			zap.String("database_id", databaseID.String()),
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve changelog entries"})
	}

	// Build response with change plan titles
	response := make([]ChangelogListItem, 0, len(changelogs))
	for _, changelog := range changelogs {
		// Fetch change plan to get the title
		changePlan, err := h.changePlanRepo.GetByID(c.Request().Context(), changelog.ChangePlanID)
		changePlanTitle := ""
		if err == nil && changePlan != nil {
			changePlanTitle = changePlan.Title
		} else {
			h.logger.Warn("Failed to fetch change plan for changelog",
				zap.Error(err),
				zap.String("change_plan_id", changelog.ChangePlanID.String()),
			)
			changePlanTitle = "Unknown Change Plan"
		}

		response = append(response, ChangelogListItem{
			ID:              changelog.ID.String(),
			DatabaseID:      changelog.DatabaseID.String(),
			ChangePlanID:    changelog.ChangePlanID.String(),
			ChangePlanTitle: changePlanTitle,
			Status:          string(changelog.Status),
			CreatedAt:       changelog.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return c.JSON(http.StatusOK, response)
}

// GetChangelogDetails returns detailed information about a specific changelog entry
// GET /api/v1/changelog/:id
func (h *ChangelogHandler) GetChangelogDetails(c echo.Context) error {
	changelogIDStr := c.Param("id")
	changelogID, err := uuid.Parse(changelogIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid changelog ID"})
	}

	// Get changelog entry
	changelog, err := h.changelogRepo.GetByID(c.Request().Context(), changelogID)
	if err != nil {
		if err == repository.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Changelog not found"})
		}
		h.logger.Error("Failed to get changelog",
			zap.Error(err),
			zap.String("changelog_id", changelogID.String()),
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve changelog"})
	}

	// Get change plan details
	changePlan, err := h.changePlanRepo.GetByID(c.Request().Context(), changelog.ChangePlanID)
	if err != nil {
		h.logger.Error("Failed to get change plan",
			zap.Error(err),
			zap.String("change_plan_id", changelog.ChangePlanID.String()),
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve change plan"})
	}

	changePlanSummary := ChangePlanSummary{
		ID:          changePlan.ID.String(),
		Title:       changePlan.Title,
		Description: changePlan.Description,
	}

	// Get schema history if it exists
	var schemaHistory interface{}
	if changelog.SchemaHistoryID != nil {
		history, err := h.schemaHistoryRepo.GetByID(c.Request().Context(), *changelog.SchemaHistoryID)
		if err != nil {
			if err != repository.ErrNotFound {
				h.logger.Warn("Failed to get schema history",
					zap.Error(err),
					zap.String("schema_history_id", changelog.SchemaHistoryID.String()),
				)
			}
			schemaHistory = nil
		} else {
			schemaHistory = history
		}
	}

	response := ChangelogDetailResponse{
		Changelog:     changelog,
		ChangePlan:    changePlanSummary,
		SchemaHistory: schemaHistory,
	}

	return c.JSON(http.StatusOK, response)
}
