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

The audit logging middleware automatically captures:
- HTTP method (GET, POST, PUT, DELETE, etc.)
- Request path
- Response status code
- User ID (from authenticated context)
- IP address
- User agent

**Middleware should log:**
- All authenticated requests (except health checks, static assets)
- Include request/response details in `Details` JSON

### 7. Service Method Pattern

The audit logging service should provide helper methods for common patterns:

```go
// Log a simple action
Log(ctx, userID, action, resourceType, resourceID, details)

// Log with HTTP context (captures IP, user agent)
LogWithHTTP(ctx, userID, action, resourceType, resourceID, httpRequest, details)

// Log state transition
LogStateTransition(ctx, userID, resourceType, resourceID, fromStatus, toStatus, details)
```

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

### Example 4: Login (with HTTP context)

```go
auditService.LogWithHTTP(ctx, userID, "login", "user", userID, httpRequest, json.RawMessage(`{
  "result": "success",
  "email": "user@example.com"
}`))
```

## Security Considerations

1. **Never log sensitive data** in Details:
   - ❌ Passwords
   - ❌ API keys
   - ❌ Encryption keys
   - ❌ Full SQL content (use previews/summaries)

2. **Log access to sensitive resources:**
   - Reading audit logs themselves
   - Accessing user data
   - Accessing database credentials

3. **Log failed authentication attempts:**
   - Failed logins
   - Invalid tokens
   - Permission denied errors

## Performance Considerations

1. **Async logging** (future enhancement):
   - Consider async logging for high-volume operations
   - Use a queue/worker pattern for non-blocking audit logs

2. **Batch logging** (future enhancement):
   - Batch multiple log entries for bulk operations

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

