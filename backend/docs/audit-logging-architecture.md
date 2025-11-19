# Audit Logging Service Architecture

This document defines the architecture, design patterns, and conventions for the audit logging system in Deeb.

## Overview

The audit logging system provides a complete audit trail of all user actions for compliance, security, and debugging purposes. All CRUD operations and important actions are logged with full context.

## Architecture

### Service Layer

The audit logging service (`AuditLogService`) provides a high-level interface for logging user actions. It abstracts the underlying repository implementation and provides consistent logging patterns.

```
┌─────────────────┐
│   Handlers      │
│   Middleware    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ AuditLogService │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ AuditLogRepo    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   PostgreSQL    │
│   audit_logs    │
└─────────────────┘
```

### Log Entry Structure

Each audit log entry contains the following fields:

```go
type AuditLog struct {
    ID           uuid.UUID       // Unique identifier
    UserID       uuid.UUID       // User who performed the action
    Action       string          // Action performed (e.g., "create", "update")
    ResourceType string          // Type of resource affected (e.g., "change_plan")
    ResourceID   uuid.UUID       // ID of the affected resource
    Details      json.RawMessage // Additional metadata (optional)
    IPAddress    *net.IP         // IP address of the request (optional)
    UserAgent    *string         // User agent string (optional)
    Timestamp    time.Time       // Action timestamp
}
```

**Field Details:**

- **ID**: Auto-generated UUID for the log entry
- **UserID**: The user who performed the action (required)
- **Action**: The action performed (see Action Types below)
- **ResourceType**: The type of resource affected (see Resource Types below)
- **ResourceID**: The UUID of the affected resource (required)
- **Details**: JSON object containing additional metadata (e.g., `{"status": "approved", "previous_status": "open"}`)
- **IPAddress**: IP address of the HTTP request (captured from middleware)
- **UserAgent**: User agent string from HTTP request (captured from middleware)
- **Timestamp**: When the action occurred (auto-set to current time)

## Action Types

Actions follow a consistent naming pattern: `{verb}_{resource_type}` or `{verb}_{operation}`.

### Authentication Actions

- `login` - User logged in
- `logout` - User logged out
- `register` - New user registration

### CRUD Actions

- `create` - Create a new resource (e.g., `create_change_plan`, `create_user`)
- `read` - Read/retrieve a resource (typically logged only for sensitive operations)
- `update` - Update an existing resource (e.g., `update_change_plan`, `update_user`)
- `delete` - Delete a resource (e.g., `delete_change_plan`, `delete_user`)

### Change Plan Actions

- `create_change_plan` - Create a new change plan
- `update_change_plan` - Update change plan details
- `delete_change_plan` - Delete a change plan
- `submit_change_plan` - Submit change plan for review
- `approve_change_plan` - Approve a change plan
- `reject_change_plan` - Reject a change plan
- `close_change_plan` - Close a change plan (reject after approval)
- `execute_change_plan` - Execute a change plan
- `assign_reviewer` - Assign reviewer to change plan
- `unassign_reviewer` - Unassign reviewer from change plan

### Project Actions

- `create_project` - Create a new project
- `update_project` - Update project details
- `delete_project` - Delete a project
- `add_project_member` - Add member to project
- `remove_project_member` - Remove member from project
- `update_project_member_role` - Update project member role

### Database Actions

- `create_database` - Create a new database
- `update_database` - Update database details
- `delete_database` - Delete a database
- `create_database_instance` - Create a new database instance
- `update_database_instance` - Update database instance details
- `delete_database_instance` - Delete a database instance

### User Actions

- `create_user` - Create a new user (admin only)
- `update_user` - Update user details
- `delete_user` - Delete a user (admin only)
- `update_user_role` - Update user organization role (admin only)

### Environment Actions

- `create_environment` - Create a new environment
- `update_environment` - Update environment details
- `delete_environment` - Delete an environment

## Resource Types

Resource types represent the entities in the system that can be acted upon.

### Core Resources

- `user` - User account
- `project` - Project
- `environment` - Environment (e.g., development, staging, production)
- `database` - Database
- `database_instance` - Database instance

### Change Plan Resources

- `change_plan` - Change plan
- `approval` - Approval record
- `execution_log` - Execution log entry
- `schema_snapshot` - Schema snapshot

### System Resources

- `audit_log` - Audit log entry (for logging audit log access)
- `policy` - Policy rule
- `policy_finding` - Policy finding from SQL review

## Logging Patterns

### 1. Action Verb Consistency

Always use lowercase action verbs:
- ✅ `create_change_plan`
- ❌ `CreateChangePlan`
- ❌ `CREATE_CHANGE_PLAN`

### 2. Resource Type Consistency

Use snake_case for resource types:
- ✅ `change_plan`
- ❌ `changePlan`
- ❌ `ChangePlan`

### 3. Details JSON Structure

The `Details` field should contain relevant context:

```json
{
  "status": "approved",
  "previous_status": "open",
  "approver_id": "uuid-here",
  "comment": "Looks good to me"
}
```

**Common Details Fields:**

- `status` - Current status after action
- `previous_status` - Status before action (for state transitions)
- `error_message` - Error message if action failed
- `result` - Result of action (`success` or `failure`)
- Additional context-specific fields

### 4. Logging Timing

**Log after successful operations:**
- Log `create` actions after successful creation
- Log `update` actions after successful update
- Log `delete` actions after successful deletion

**Log before operations that might fail:**
- Log `login` attempts (both success and failure)
- Log `execute_change_plan` attempts (both success and failure)

**Log state transitions:**
- Log `approve_change_plan` when status changes from `open` to `approved`
- Log `execute_change_plan` when status changes from `approved` to `completed`

### 5. Error Handling

When logging actions that may fail:
- Include `result: "success"` or `result: "failure"` in Details
- Include `error_message` in Details if action failed
- Always log both success and failure cases for security-sensitive operations

### 6. Middleware Integration

The audit logging middleware (`AuditLogMiddleware`) automatically captures and logs all authenticated HTTP requests. It runs **after** the request handler completes, ensuring it captures the actual response state.

**Architecture:**

```
HTTP Request
    ↓
Auth Middleware (sets user in context)
    ↓
Audit Middleware (wraps handler)
    ↓
Handler Execution
    ↓
Response Captured
    ↓
Build Details JSON
    ↓
Apply Redaction (automatic)
    ↓
Async Audit Log (non-blocking goroutine)
    ↓
HTTP Response
```

**Automatically Captured Fields:**
- HTTP method (GET, POST, PUT, DELETE, etc.)
- Request path
- Response status code
- User ID (from authenticated context)
- IP address (supports `X-Forwarded-For` and `X-Real-IP` proxy headers)
- User agent

**HTTP Method to Action Mapping:**

The middleware automatically maps HTTP methods to audit actions:
- `POST` → `create`
- `GET` → `read`
- `PUT/PATCH` → `update`
- `DELETE` → `delete`

**Path to Resource Type Extraction:**

Resource types are extracted from the API path:
```
/api/v1/change-plans/{id} → resource_type: "change_plan"
/api/v1/databases/{id}    → resource_type: "database"
/api/v1/users/{id}        → resource_type: "user"
```

**Special Action Detection:**

The middleware detects special actions from the request path:
- `/approve` → `approve_change_plan`
- `/reject` → `reject_change_plan`
- `/execute` → `execute_change_plan`
- `/close` → `close_change_plan`
- `/login` → `login` (resource type: `user`)
- `/logout` → `logout` (resource type: `user`)

**What Gets Logged:**
- All authenticated requests (user ID ≠ nil)
- All response status codes (including errors)

**What Gets Skipped:**
- Health check endpoints (`/health`, `/healthz`, `/ready`)
- Static assets (`/static/`, `/assets/`, `.css`, `.js`, `.png`, `.jpg`, `.ico`)
- Swagger UI assets (`/api/docs/swagger`)
- Unauthenticated requests (no user in context)

**Async Logging:**

Audit logging is performed asynchronously in a goroutine to avoid blocking the HTTP response:
- Uses `context.Background()` instead of request context to prevent cancellation
- Errors in audit logging are logged but don't fail the request
- Even if the client disconnects, the audit log is still written

### 7. Handler Enrichment Pattern

Handlers can provide additional context to audit logs using the **AuditEnrichment** pattern. This allows handlers to add business-specific details that the middleware alone cannot capture.

**AuditEnrichment Structure:**

```go
type AuditEnrichment struct {
    Email         string                 `json:"email,omitempty"`
    FailureReason string                 `json:"failure_reason,omitempty"`
    ErrorMessage  string                 `json:"error_message,omitempty"`
    Result        string                 `json:"result,omitempty"` // "success" or "failure"
    Custom        map[string]interface{} `json:"-"`                // Merged inline
}
```

**How to Use in Handlers:**

```go
func LoginHandler(c echo.Context) error {
    // ... authentication logic ...

    // Add enrichment for audit log
    c.Set("audit_enrichment", &middleware.AuditEnrichment{
        Email: email,
        Result: "success",
        Custom: map[string]interface{}{
            "login_method": "oauth",
            "provider": "google",
        },
    })

    return c.JSON(http.StatusOK, response)
}
```

**Enrichment for Failed Operations:**

```go
func CreateChangePlanHandler(c echo.Context) error {
    // ... validation fails ...

    c.Set("audit_enrichment", &middleware.AuditEnrichment{
        Result: "failure",
        ErrorMessage: "Invalid SQL syntax",
        FailureReason: "validation_error",
    })

    return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
}
```

**How Enrichment Works:**

1. Handler sets enrichment in Echo context using `c.Set("audit_enrichment", ...)`
2. Middleware checks for enrichment after handler completes
3. Enrichment fields are merged into the audit log `Details` JSON
4. Custom fields are merged inline (not nested)
5. All fields go through automatic redaction before storage

**Enrichment vs. Direct Logging:**

- ✅ **Use enrichment** when the middleware logs the request (most cases)
- ✅ **Use enrichment** to add business context (email, operation result, etc.)
- ❌ **Don't use direct logging** in handlers unless the operation is not an HTTP request
- ❌ **Don't bypass** the middleware - it ensures consistent logging

### 8. Service Method Pattern

The audit logging service provides helper methods for different logging scenarios:

```go
// Log a simple action (use when not in HTTP context)
Log(ctx, userID, action, resourceType, resourceID, details)

// Log with HTTP context (captures IP, user agent) - used by middleware
LogWithHTTP(ctx, userID, action, resourceType, resourceID, httpRequest, details)

// Log state transition (convenience method)
LogStateTransition(ctx, userID, resourceType, resourceID, fromStatus, toStatus, details)
```

**When to Use Each Method:**

- **LogWithHTTP**: Used by middleware automatically (don't call from handlers)
- **Log**: Use for background jobs, scheduled tasks, or non-HTTP operations
- **LogStateTransition**: Use for explicit state changes (rare - middleware handles most cases)

## Examples

### Example 1: Create Change Plan

```go
auditService.Log(ctx, userID, "create_change_plan", "change_plan", changePlanID, json.RawMessage(`{
  "title": "Add users table",
  "status": "draft",
  "project_id": "project-uuid"
}`))
```

### Example 2: Approve Change Plan

```go
auditService.LogStateTransition(ctx, userID, "change_plan", changePlanID, 
  "open", "approved", json.RawMessage(`{
  "approver_id": "approver-uuid",
  "comment": "Looks good"
}`))
```

### Example 3: Execute Change Plan (with failure)

```go
details := json.RawMessage(`{
  "result": "failure",
  "error_message": "SQL syntax error at line 5",
  "execution_duration_ms": 1234
}`)
auditService.Log(ctx, executorID, "execute_change_plan", "change_plan", changePlanID, details)
```

### Example 4: Login with Enrichment (Middleware Pattern)

```go
func LoginHandler(c echo.Context) error {
    var req LoginRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
    }

    // Authenticate user
    user, err := authService.Authenticate(req.Email, req.Password)
    if err != nil {
        // Enrichment for failed login
        c.Set("audit_enrichment", &middleware.AuditEnrichment{
            Email: req.Email,
            Result: "failure",
            FailureReason: "invalid_credentials",
        })
        return echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
    }

    // Enrichment for successful login
    c.Set("audit_enrichment", &middleware.AuditEnrichment{
        Email: user.Email,
        Result: "success",
        Custom: map[string]interface{}{
            "login_method": "password",
        },
    })

    return c.JSON(http.StatusOK, user)
}
```

**Resulting Audit Log (automatically redacted):**
```json
{
  "user_id": "user-uuid",
  "action": "login",
  "resource_type": "user",
  "resource_id": "user-uuid",
  "details": {
    "method": "POST",
    "path": "/api/v1/auth/login",
    "status": 200,
    "status_code": 200,
    "result": "success",
    "email": "user@example.com",
    "login_method": "password"
  },
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "timestamp": "2025-01-18T10:30:00Z"
}
```

### Example 5: Create Database Instance with Credentials (Redacted)

```go
func CreateDatabaseInstanceHandler(c echo.Context) error {
    var req CreateDatabaseInstanceRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
    }

    // Create database instance
    instance, err := dbService.CreateInstance(ctx, req)
    if err != nil {
        c.Set("audit_enrichment", &middleware.AuditEnrichment{
            Result: "failure",
            ErrorMessage: err.Error(),
        })
        return err
    }

    // Enrichment with connection details
    c.Set("audit_enrichment", &middleware.AuditEnrichment{
        Result: "success",
        Custom: map[string]interface{}{
            "database_name": req.DatabaseName,
            "host": req.Host,
            "port": req.Port,
            // Note: Even if you accidentally include sensitive data,
            // it will be automatically redacted
            "connection_string": req.ConnectionString,
            "db_password": req.Password,
        },
    })

    return c.JSON(http.StatusCreated, instance)
}
```

**Resulting Audit Log (with automatic redaction):**
```json
{
  "user_id": "user-uuid",
  "action": "create",
  "resource_type": "database_instance",
  "resource_id": "instance-uuid",
  "details": {
    "method": "POST",
    "path": "/api/v1/database-instances",
    "status": 201,
    "status_code": 201,
    "result": "success",
    "database_name": "myapp_db",
    "host": "db.example.com",
    "port": 5432,
    "connection_string": "[REDACTED]",
    "db_password": "[REDACTED]"
  },
  "ip_address": "192.168.1.1",
  "user_agent": "curl/7.68.0",
  "timestamp": "2025-01-18T10:35:00Z"
}
```

### Example 6: SQL Execution with Risk Assessment

```go
// In change plan execution handler
sqlDetails := audit.SQLExecutionDetails{
    Statement:       "DROP TABLE old_users",
    StatementType:   audit.SQLTypeDDL,
    RiskLevel:       audit.SQLRiskHigh,
    DatabaseName:    "production_db",
    Environment:     "production",
    RowsAffected:    0,
    ExecutionTimeMS: 125,
    Success:         true,
}

detailsJSON, _ := json.Marshal(sqlDetails)

c.Set("audit_enrichment", &middleware.AuditEnrichment{
    Result: "success",
    Custom: map[string]interface{}{
        "sql_execution": sqlDetails,
    },
})
```

**Resulting Audit Log:**
```json
{
  "user_id": "user-uuid",
  "action": "execute_change_plan",
  "resource_type": "change_plan",
  "resource_id": "plan-uuid",
  "details": {
    "method": "POST",
    "path": "/api/v1/change-plans/{id}/execute",
    "status": 200,
    "status_code": 200,
    "result": "success",
    "sql_execution": {
      "statement": "DROP TABLE old_users",
      "statement_type": "DDL",
      "risk_level": "high",
      "database_name": "production_db",
      "environment": "production",
      "rows_affected": 0,
      "execution_time_ms": 125,
      "success": true
    }
  },
  "ip_address": "192.168.1.1",
  "user_agent": "DeebCLI/1.0",
  "timestamp": "2025-01-18T10:40:00Z"
}
```

**Note:** Query results are **never** included in SQL execution audit logs.

## Security Considerations

### Automatic Sensitive Data Redaction

The audit logging system includes a comprehensive redaction system that **automatically** strips sensitive data from audit logs before storage. This ensures security by default without requiring developers to manually redact data.

**Redacted Field Patterns (case-insensitive):**

1. **Passwords:**
   - `password`, `passwd`, `pwd`
   - `new_password`, `old_password`, `current_password`

2. **Database Credentials:**
   - `db_password`, `datasource_password`, `database_password`
   - `connection_string` (credentials are stripped from connection URIs)

3. **MFA/2FA Secrets:**
   - `mfa_secret`, `totp_secret`, `otp_secret`
   - `backup_codes`, `recovery_keys`, `two_factor_secret`

4. **Authentication Tokens:**
   - `api_key`, `api_secret`, `access_token`, `refresh_token`
   - `auth_token`, `bearer_token`, `secret_key`, `private_key`, `encryption_key`

5. **Financial Data:**
   - `credit_card`, `card_number`, `cvv`, `ssn`, `social_security`

6. **Personal Identifiable Information:**
   - `passport`, `tax_id`, `license_number`

**How Redaction Works:**

- Redaction is applied in the middleware's `buildDetails()` method
- All sensitive field values are replaced with `[REDACTED]`
- Redaction works recursively on nested JSON objects and arrays
- Redaction is **enabled by default** for security
- Can be disabled for testing with `SetRedactionEnabled(false)`

**Connection String Redaction:**

Connection strings are specially handled to strip embedded credentials:
```
postgresql://user:password@host:5432/db
→ postgresql://user:[REDACTED]@host:5432/db
```

### SQL Execution Logging

SQL execution audit logs capture metadata but **never log query results** to prevent data leakage:

**What Gets Logged:**
- ✅ Full SQL statement text
- ✅ Statement type (DDL, DML, DCL, TCL, Query)
- ✅ Risk level (high/medium/low based on statement type)
- ✅ Execution metadata: `rows_affected`, `execution_time_ms`, `database_name`, `environment`
- ✅ Success/failure status and error messages

**What NEVER Gets Logged:**
- ❌ Query results
- ❌ Returned rows
- ❌ Data values from SELECT statements

**Risk Level Classification:**
- **High risk**: DROP, TRUNCATE, UPDATE/DELETE without WHERE clause
- **Medium risk**: CREATE, ALTER, UPDATE/DELETE with WHERE clause, DCL operations
- **Low risk**: SELECT, INSERT, SHOW, DESCRIBE, TCL operations

### Best Practices for Developers

1. **Never manually include sensitive data** in Details JSON:
   - ❌ Don't pass raw passwords
   - ❌ Don't pass API keys
   - ❌ Don't pass SQL query results
   - ✅ Use the enrichment pattern (see below) and let redaction handle it

2. **Use the AuditEnrichment pattern** for adding context:
   ```go
   c.Set("audit_enrichment", &middleware.AuditEnrichment{
       Email: email,
       Result: "success",
       Custom: map[string]interface{}{
           "login_method": "oauth",
       },
   })
   ```

3. **Log failed authentication attempts:**
   - Failed logins
   - Invalid tokens
   - Permission denied errors

4. **Adding new sensitive field patterns:**
   ```go
   redactor := audit.NewRedactor()
   redactor.AddPattern(`(?i)custom_secret`, "Custom secret field")
   ```

## Performance Considerations

1. **Async logging** (currently implemented):
   - Audit logging is performed asynchronously in goroutines to avoid blocking HTTP responses
   - Uses `context.Background()` instead of request context to prevent cancellation
   - Errors in audit logging are logged but don't fail requests
   - Non-blocking behavior ensures client requests complete quickly regardless of audit log performance

2. **Queue/batching optimizations** (future enhancement):
   - Consider implementing a queue/worker pattern for higher throughput
   - Batch multiple log entries for bulk operations to reduce database roundtrips

3. **Indexing:**
   - All filterable fields are indexed (user_id, action, resource_type, resource_id, timestamp)
   - GIN index on details JSONB for efficient JSON queries

## Compliance

The audit logging system supports compliance requirements by:

1. **Immutable logs**: Audit logs are append-only (no updates/deletes)
2. **Complete audit trail**: All user actions are logged
3. **Timestamp accuracy**: All logs include precise timestamps
4. **User attribution**: All actions are attributed to specific users
5. **Context preservation**: IP addresses and user agents captured for forensics

## Future Enhancements

1. **Log retention policies**: Automatic archival/deletion of old logs
2. **Log compression**: Compress old logs for storage efficiency
3. **Real-time alerts**: Alert on suspicious patterns
4. **Export functionality**: Export logs for external analysis
5. **Log analysis dashboard**: UI for analyzing audit logs

