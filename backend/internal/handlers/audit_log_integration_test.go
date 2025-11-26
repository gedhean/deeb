//go:build integration
// +build integration

package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_AuditLogs_ListAuditLogs tests the audit log querying endpoint
func TestIntegration_AuditLogs_ListAuditLogs(t *testing.T) {
	// Setup
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)

	// Initialize repositories
	auditLogRepo := postgres.NewAuditLogRepository(db)

	// Create admin user
	adminEmail := fmt.Sprintf("admin-%d@example.com", time.Now().Unix())
	adminID := createTestUser(t, db, adminEmail, rbac.OrganizationAdmin)
	adminToken, _ := loginUser(t, db, adminEmail, "TestPassword123!")

	// Create developer user (non-admin)
	devEmail := fmt.Sprintf("developer-%d@example.com", time.Now().Unix())
	devID := createTestUser(t, db, devEmail, rbac.OrganizationDeveloper)
	devToken, _ := loginUser(t, db, devEmail, "TestPassword123!")

	ctx := context.Background()

	// Create some audit log entries
	testResourceID := uuid.New()
	auditLog1 := &models.AuditLog{
		ID:           uuid.New(),
		UserID:       &adminID,
		Action:       "create",
		ResourceType: "project",
		ResourceID:   testResourceID,
		Details:      json.RawMessage(`{}`),
		Timestamp:    time.Now().Add(-2 * time.Hour),
	}
	err := auditLogRepo.Create(ctx, auditLog1)
	require.NoError(t, err)

	auditLog2 := &models.AuditLog{
		ID:           uuid.New(),
		UserID:       &devID,
		Action:       "update",
		ResourceType: "change_plan",
		ResourceID:   uuid.New(),
		Details:      json.RawMessage(`{}`),
		Timestamp:    time.Now().Add(-1 * time.Hour),
	}
	err = auditLogRepo.Create(ctx, auditLog2)
	require.NoError(t, err)

	auditLog3 := &models.AuditLog{
		ID:           uuid.New(),
		UserID:       &adminID,
		Action:       "delete",
		ResourceType: "project",
		ResourceID:   testResourceID,
		Details:      json.RawMessage(`{}`),
		Timestamp:    time.Now(),
	}
	err = auditLogRepo.Create(ctx, auditLog3)
	require.NoError(t, err)

	t.Run("List Audit Logs - Admin Success", func(t *testing.T) {
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, "/api/v1/audit-logs", nil, adminToken)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		assert.Contains(t, response, "audit_logs")
		assert.Contains(t, response, "total")
		assert.Contains(t, response, "limit")
		assert.Contains(t, response, "offset")
		assert.Contains(t, response, "count")

		auditLogs := response["audit_logs"].([]interface{})
		assert.GreaterOrEqual(t, len(auditLogs), 3, "should have at least 3 audit logs")
	})

	t.Run("List Audit Logs - Developer Forbidden", func(t *testing.T) {
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, "/api/v1/audit-logs", nil, devToken)
		assert.Equal(t, http.StatusForbidden, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Insufficient permissions")
	})

	t.Run("List Audit Logs - Filter by User ID", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/audit-logs?user_id=%s", adminID.String())
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, url, nil, adminToken)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		auditLogs := response["audit_logs"].([]interface{})
		assert.GreaterOrEqual(t, len(auditLogs), 2, "should have at least 2 audit logs for admin")

		// Verify all logs belong to admin
		for _, logInterface := range auditLogs {
			logMap := logInterface.(map[string]interface{})
			assert.Equal(t, adminID.String(), logMap["user_id"].(string))
		}
	})

	t.Run("List Audit Logs - Filter by Action", func(t *testing.T) {
		url := "/api/v1/audit-logs?action=create"
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, url, nil, adminToken)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		auditLogs := response["audit_logs"].([]interface{})
		assert.GreaterOrEqual(t, len(auditLogs), 1, "should have at least 1 create action")

		// Verify all logs have create action
		for _, logInterface := range auditLogs {
			logMap := logInterface.(map[string]interface{})
			assert.Equal(t, "create", logMap["action"].(string))
		}
	})

	t.Run("List Audit Logs - Filter by Resource Type", func(t *testing.T) {
		url := "/api/v1/audit-logs?target_type=project"
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, url, nil, adminToken)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		auditLogs := response["audit_logs"].([]interface{})
		assert.GreaterOrEqual(t, len(auditLogs), 2, "should have at least 2 project logs")

		// Verify all logs are for project resource type
		for _, logInterface := range auditLogs {
			logMap := logInterface.(map[string]interface{})
			assert.Equal(t, "project", logMap["resource_type"].(string))
		}
	})

	t.Run("List Audit Logs - Filter by Resource ID", func(t *testing.T) {
		url := fmt.Sprintf("/api/v1/audit-logs?target_id=%s", testResourceID.String())
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, url, nil, adminToken)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		auditLogs := response["audit_logs"].([]interface{})
		assert.GreaterOrEqual(t, len(auditLogs), 2, "should have at least 2 logs for this resource")

		// Verify all logs are for the specified resource
		for _, logInterface := range auditLogs {
			logMap := logInterface.(map[string]interface{})
			assert.Equal(t, testResourceID.String(), logMap["resource_id"].(string))
		}
	})

	t.Run("List Audit Logs - Filter by Date Range", func(t *testing.T) {
		startDate := time.Now().Add(-90 * time.Minute).Format(time.RFC3339)
		endDate := time.Now().Add(-30 * time.Minute).Format(time.RFC3339)
		url := fmt.Sprintf("/api/v1/audit-logs?start_date=%s&end_date=%s", startDate, endDate)
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, url, nil, adminToken)
		require.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		auditLogs := response["audit_logs"].([]interface{})
		assert.GreaterOrEqual(t, len(auditLogs), 1, "should have at least 1 log in date range")

		// Verify all logs are within date range
		for _, logInterface := range auditLogs {
			logMap := logInterface.(map[string]interface{})
			timestampStr := logMap["timestamp"].(string)
			timestamp, err := time.Parse(time.RFC3339, timestampStr)
			require.NoError(t, err)
			assert.True(t, timestamp.After(time.Now().Add(-90*time.Minute)) && timestamp.Before(time.Now().Add(-30*time.Minute)))
		}
	})

	t.Run("List Audit Logs - Pagination", func(t *testing.T) {
		// First page
		rec1 := makeAuthenticatedRequest(t, e, http.MethodGet, "/api/v1/audit-logs?limit=2&offset=0", nil, adminToken)
		require.Equal(t, http.StatusOK, rec1.Code)

		var response1 map[string]interface{}
		err := json.NewDecoder(rec1.Body).Decode(&response1)
		require.NoError(t, err)

		auditLogs1 := response1["audit_logs"].([]interface{})
		assert.Equal(t, 2, len(auditLogs1), "should have 2 logs on first page")
		assert.Equal(t, float64(2), response1["limit"].(float64))
		assert.Equal(t, float64(0), response1["offset"].(float64))

		// Second page
		rec2 := makeAuthenticatedRequest(t, e, http.MethodGet, "/api/v1/audit-logs?limit=2&offset=2", nil, adminToken)
		require.Equal(t, http.StatusOK, rec2.Code)

		var response2 map[string]interface{}
		err = json.NewDecoder(rec2.Body).Decode(&response2)
		require.NoError(t, err)

		auditLogs2 := response2["audit_logs"].([]interface{})
		assert.GreaterOrEqual(t, len(auditLogs2), 1, "should have at least 1 log on second page")

		// Verify logs are different
		if len(auditLogs1) > 0 && len(auditLogs2) > 0 {
			log1ID := auditLogs1[0].(map[string]interface{})["id"].(string)
			log2ID := auditLogs2[0].(map[string]interface{})["id"].(string)
			assert.NotEqual(t, log1ID, log2ID, "logs should be different")
		}
	})

	t.Run("List Audit Logs - Invalid Date Format", func(t *testing.T) {
		url := "/api/v1/audit-logs?start_date=invalid-date"
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, url, nil, adminToken)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid start_date")
	})

	t.Run("List Audit Logs - Invalid UUID", func(t *testing.T) {
		url := "/api/v1/audit-logs?user_id=invalid-uuid"
		rec := makeAuthenticatedRequest(t, e, http.MethodGet, url, nil, adminToken)
		assert.Equal(t, http.StatusBadRequest, rec.Code)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)
		assert.Contains(t, response["error"], "Invalid user_id")
	})
}
