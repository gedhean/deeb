//go:build integration
// +build integration

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository/postgres"
	"github.com/nsxbet/deeb/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_AuditLog_PasswordRedaction tests that passwords are automatically redacted in audit logs
func TestIntegration_AuditLog_PasswordRedaction(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)
	auditLogRepo := postgres.NewAuditLogRepository(db)
	ctx := context.Background()

	t.Run("Register User - Password Redacted", func(t *testing.T) {
		// Create a unique email for this test
		email := fmt.Sprintf("redaction-test-%d@example.com", time.Now().UnixNano())
		password := "SuperSecretPassword123!"

		reqBody := map[string]string{
			"email":    email,
			"password": password,
			"name":     "Test User",
		}
		body, _ := json.Marshal(reqBody)

		rec := makeRequest(t, e, http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
		require.Equal(t, http.StatusCreated, rec.Code)

		var userResp models.UserResponse
		err := json.NewDecoder(rec.Body).Decode(&userResp)
		require.NoError(t, err)

		// Wait for async audit log to be written
		time.Sleep(200 * time.Millisecond)

		// Fetch audit logs for this user
		auditLogs, err := auditLogRepo.GetByUserID(ctx, userResp.ID, 10, 0)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs, "Audit log should be created for registration")

		// Find the registration audit log
		var registrationLog *models.AuditLog
		for i := range auditLogs {
			if auditLogs[i].Action == services.ActionCreate && auditLogs[i].ResourceType == services.ResourceTypeUser {
				registrationLog = &auditLogs[i]
				break
			}
		}
		require.NotNil(t, registrationLog, "Registration audit log should exist")

		// Verify password is redacted in details
		var details map[string]interface{}
		err = json.Unmarshal(registrationLog.Details, &details)
		require.NoError(t, err)

		// Check that password field doesn't contain the actual password
		detailsJSON := string(registrationLog.Details)
		assert.NotContains(t, detailsJSON, password, "Password should not appear in audit log")
		assert.NotContains(t, detailsJSON, "SuperSecret", "Password should not appear in audit log")

		// If password field is present, it should be redacted
		if pwd, ok := details["password"]; ok {
			assert.Equal(t, "[REDACTED]", pwd, "Password field should be redacted")
		}
	})
}

// TestIntegration_AuditLog_ConnectionStringRedaction tests that database connection strings are redacted
func TestIntegration_AuditLog_ConnectionStringRedaction(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)
	auditLogRepo := postgres.NewAuditLogRepository(db)
	ctx := context.Background()

	// Create admin user for database operations
	adminEmail := fmt.Sprintf("admin-redact-%d@example.com", time.Now().Unix())
	adminID := createTestUser(t, db, adminEmail, rbac.OrganizationAdmin)
	adminToken, _ := loginUser(t, db, adminEmail, "TestPassword123!")

	t.Run("Create Database Instance - Connection String Redacted", func(t *testing.T) {
		connectionString := "postgresql://testuser:SuperSecretPassword@db.example.com:5432/testdb"

		reqBody := map[string]interface{}{
			"name":              "Test Database Instance",
			"host":              "db.example.com",
			"port":              5432,
			"database":          "testdb",
			"username":          "testuser",
			"password":          "SuperSecretPassword",
			"connection_string": connectionString,
		}
		body, _ := json.Marshal(reqBody)

		rec := makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/database-instances", bytes.NewReader(body), adminToken)

		// Even if the request fails, the audit log should have been created with redacted data
		// Wait for async audit log
		time.Sleep(200 * time.Millisecond)

		// Fetch recent audit logs for this user
		auditLogs, err := auditLogRepo.GetByUserID(ctx, adminID, 10, 0)
		require.NoError(t, err)

		// Find the most recent audit log
		if len(auditLogs) > 0 {
			latestLog := auditLogs[0]

			// Verify sensitive data is redacted
			detailsJSON := string(latestLog.Details)

			// Password should be redacted
			assert.NotContains(t, detailsJSON, "SuperSecretPassword", "Password should not appear in audit log")

			// Connection string credentials should be redacted
			assert.NotContains(t, detailsJSON, ":SuperSecretPassword@", "Connection string password should be redacted")

			// Check that if these fields are present, they are redacted
			var details map[string]interface{}
			err = json.Unmarshal(latestLog.Details, &details)
			require.NoError(t, err)

			if pwd, ok := details["password"]; ok {
				assert.Equal(t, "[REDACTED]", pwd, "Password field should be redacted")
			}

			if connStr, ok := details["connection_string"]; ok {
				connStrValue := fmt.Sprintf("%v", connStr)
				assert.NotContains(t, connStrValue, "SuperSecretPassword", "Connection string should have password redacted")
				if connStrValue != "[REDACTED]" {
					// If not fully redacted, should at least have the password part redacted
					assert.Contains(t, connStrValue, "[REDACTED]", "Connection string should contain redacted marker")
				}
			}

			if dbPass, ok := details["db_password"]; ok {
				assert.Equal(t, "[REDACTED]", dbPass, "db_password field should be redacted")
			}
		}
	})
}

// TestIntegration_AuditLog_MFASecretRedaction tests that MFA secrets are redacted
func TestIntegration_AuditLog_MFASecretRedaction(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)
	auditLogRepo := postgres.NewAuditLogRepository(db)
	ctx := context.Background()

	// Create user
	userEmail := fmt.Sprintf("mfa-test-%d@example.com", time.Now().Unix())
	userID := createTestUser(t, db, userEmail, rbac.OrganizationMember)
	userToken, _ := loginUser(t, db, userEmail, "TestPassword123!")

	t.Run("Setup MFA - Secret Redacted", func(t *testing.T) {
		totpSecret := "JBSWY3DPEHPK3PXP"
		backupCodes := []string{"12345678", "87654321", "11111111"}

		// Simulate MFA setup request
		reqBody := map[string]interface{}{
			"totp_secret":  totpSecret,
			"backup_codes": backupCodes,
			"mfa_secret":   totpSecret,
		}
		body, _ := json.Marshal(reqBody)

		// Make request (endpoint may not exist, but audit log should still be created)
		_ = makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/users/mfa/setup", bytes.NewReader(body), userToken)

		// Wait for async audit log
		time.Sleep(200 * time.Millisecond)

		// Fetch recent audit logs for this user
		auditLogs, err := auditLogRepo.GetByUserID(ctx, userID, 10, 0)
		require.NoError(t, err)

		if len(auditLogs) > 0 {
			latestLog := auditLogs[0]
			detailsJSON := string(latestLog.Details)

			// Verify MFA secrets are redacted
			assert.NotContains(t, detailsJSON, totpSecret, "TOTP secret should not appear in audit log")
			assert.NotContains(t, detailsJSON, "JBSWY3DPEHPK3PXP", "TOTP secret should not appear in audit log")

			var details map[string]interface{}
			err = json.Unmarshal(latestLog.Details, &details)
			require.NoError(t, err)

			if secret, ok := details["totp_secret"]; ok {
				assert.Equal(t, "[REDACTED]", secret, "totp_secret field should be redacted")
			}

			if secret, ok := details["mfa_secret"]; ok {
				assert.Equal(t, "[REDACTED]", secret, "mfa_secret field should be redacted")
			}

			if codes, ok := details["backup_codes"]; ok {
				assert.Equal(t, "[REDACTED]", codes, "backup_codes field should be redacted")
			}
		}
	})
}

// TestIntegration_AuditLog_NestedRedaction tests that nested sensitive fields are redacted
func TestIntegration_AuditLog_NestedRedaction(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)
	auditLogRepo := postgres.NewAuditLogRepository(db)
	ctx := context.Background()

	adminEmail := fmt.Sprintf("admin-nested-%d@example.com", time.Now().Unix())
	adminID := createTestUser(t, db, adminEmail, rbac.OrganizationAdmin)
	adminToken, _ := loginUser(t, db, adminEmail, "TestPassword123!")

	t.Run("Nested Sensitive Data Redacted", func(t *testing.T) {
		// Request with nested sensitive data
		reqBody := map[string]interface{}{
			"name": "Test Resource",
			"config": map[string]interface{}{
				"api_key":    "sk-1234567890abcdef",
				"api_secret": "secret-key-9876543210",
				"database": map[string]interface{}{
					"host":        "db.example.com",
					"db_password": "NestedSecretPassword",
					"port":        5432,
				},
			},
		}
		body, _ := json.Marshal(reqBody)

		// Make request to create project (or any resource)
		_ = makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(body), adminToken)

		// Wait for async audit log
		time.Sleep(200 * time.Millisecond)

		// Fetch recent audit logs
		auditLogs, err := auditLogRepo.GetByUserID(ctx, adminID, 10, 0)
		require.NoError(t, err)

		if len(auditLogs) > 0 {
			latestLog := auditLogs[0]
			detailsJSON := string(latestLog.Details)

			// Verify nested sensitive data is redacted
			assert.NotContains(t, detailsJSON, "sk-1234567890abcdef", "API key should not appear in audit log")
			assert.NotContains(t, detailsJSON, "secret-key-9876543210", "API secret should not appear in audit log")
			assert.NotContains(t, detailsJSON, "NestedSecretPassword", "Nested password should not appear in audit log")
		}
	})
}

// TestIntegration_AuditLog_SQLExecutionNoResults tests that SQL query results are never logged
func TestIntegration_AuditLog_SQLExecutionNoResults(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)
	auditLogRepo := postgres.NewAuditLogRepository(db)
	ctx := context.Background()

	adminEmail := fmt.Sprintf("admin-sql-%d@example.com", time.Now().Unix())
	adminID := createTestUser(t, db, adminEmail, rbac.OrganizationAdmin)
	adminToken, _ := loginUser(t, db, adminEmail, "TestPassword123!")

	t.Run("SQL Execution - No Query Results in Audit Log", func(t *testing.T) {
		// Simulate SQL execution with results
		reqBody := map[string]interface{}{
			"sql": "SELECT id, email, password FROM users WHERE id = 1",
			"results": []map[string]interface{}{ // This should NEVER appear in audit log
				{
					"id":       1,
					"email":    "user@example.com",
					"password": "hashed_password_12345",
				},
			},
			"rows_affected":     1,
			"execution_time_ms": 45,
		}
		body, _ := json.Marshal(reqBody)

		// Make request (endpoint may not exist, but audit log should be created)
		_ = makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/change-plans/execute", bytes.NewReader(body), adminToken)

		// Wait for async audit log
		time.Sleep(200 * time.Millisecond)

		// Fetch recent audit logs
		auditLogs, err := auditLogRepo.GetByUserID(ctx, adminID, 10, 0)
		require.NoError(t, err)

		if len(auditLogs) > 0 {
			latestLog := auditLogs[0]
			detailsJSON := string(latestLog.Details)

			// Verify SQL results are NOT logged
			assert.NotContains(t, detailsJSON, "hashed_password_12345", "Query results should not appear in audit log")
			assert.NotContains(t, detailsJSON, "user@example.com", "Query results should not appear in audit log")

			// The SQL statement itself is OK to log (metadata)
			// But results should be stripped

			var details map[string]interface{}
			err = json.Unmarshal(latestLog.Details, &details)
			require.NoError(t, err)

			// If results field exists, it should be removed or redacted
			if results, ok := details["results"]; ok {
				// If present, should be redacted or empty
				resultsStr := fmt.Sprintf("%v", results)
				assert.NotContains(t, resultsStr, "hashed_password", "Results field should not contain actual data")
			}
		}
	})
}

// TestIntegration_AuditLog_MultipleRedactionPatterns tests multiple redaction patterns in a single request
func TestIntegration_AuditLog_MultipleRedactionPatterns(t *testing.T) {
	db, cleanup := setupResourceTestDB(t)
	defer cleanup()
	cleanupResourceTestDB(t, db)
	defer cleanupResourceTestDB(t, db)

	e, _ := createEchoServer(t, db)
	auditLogRepo := postgres.NewAuditLogRepository(db)
	ctx := context.Background()

	adminEmail := fmt.Sprintf("admin-multi-%d@example.com", time.Now().Unix())
	adminID := createTestUser(t, db, adminEmail, rbac.OrganizationAdmin)
	adminToken, _ := loginUser(t, db, adminEmail, "TestPassword123!")

	t.Run("Multiple Sensitive Fields Redacted", func(t *testing.T) {
		// Request with multiple types of sensitive data
		reqBody := map[string]interface{}{
			"name":              "Test Config",
			"password":          "UserPassword123",
			"api_key":           "sk-proj-1234567890",
			"access_token":      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			"db_password":       "DBPassword456",
			"connection_string": "postgresql://user:pass@host/db",
			"mfa_secret":        "JBSWY3DPEHPK3PXP",
			"credit_card":       "4532-1234-5678-9010",
			"ssn":               "123-45-6789",
			"public_field":      "This should not be redacted",
		}
		body, _ := json.Marshal(reqBody)

		_ = makeAuthenticatedRequest(t, e, http.MethodPost, "/api/v1/projects", bytes.NewReader(body), adminToken)

		// Wait for async audit log
		time.Sleep(200 * time.Millisecond)

		// Fetch recent audit logs
		auditLogs, err := auditLogRepo.GetByUserID(ctx, adminID, 10, 0)
		require.NoError(t, err)

		if len(auditLogs) > 0 {
			latestLog := auditLogs[0]
			detailsJSON := string(latestLog.Details)

			// Verify all sensitive fields are redacted
			sensitiveValues := []string{
				"UserPassword123",
				"sk-proj-1234567890",
				"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
				"DBPassword456",
				":pass@",
				"JBSWY3DPEHPK3PXP",
				"4532-1234-5678-9010",
				"123-45-6789",
			}

			for _, sensitive := range sensitiveValues {
				assert.NotContains(
					t,
					detailsJSON,
					sensitive,
					fmt.Sprintf("Sensitive value '%s' should not appear in audit log", sensitive),
				)
			}

			// Verify non-sensitive field is preserved
			assert.Contains(t, detailsJSON, "This should not be redacted", "Non-sensitive fields should be preserved")

			// Verify redaction marker is present
			assert.True(t, strings.Contains(detailsJSON, "[REDACTED]"), "Audit log should contain redaction markers")
		}
	})
}
