# Audit Log Design

## Core Architecture

1. Database Storage
- Single table: audit_log with JSONB payload column
- Indexed on: created_at, parent, method, resource, user
- Stores all audit data as JSON (camelCase keys via json.Marshal)

2. Interception Mechanism

API calls are intercepted by middleware to capture request and response details.

**Flow:**
Request → Check if audit enabled → Execute method → Capture response → Redact sensitive data → Store audit log

3. What Gets Audited

API calls should be marked with a tag to indicate that they should be audited. Examples

- SQL queries and admin execution
- User/instance/database management
- Login operations

4. Data Captured

Each audit log records:
- Who: User UID
- What: API method and resource
- When: Timestamp
- Where: Caller IP and user agent
- How: Request/response payloads (redacted)
- Result: Status code and latency

5. Sensitive Data Protection

Automatically redacts:
- Passwords (login, datasource)
- Query results (to avoid data leakage)
- MFA secrets

## Key Design Principles

1. **Response captured AFTER execution** - Always captures actual result state
2. **Non-blocking** - Audit logging runs in goroutine; failures don't block HTTP responses
3. **Context isolation** - Uses `context.Background()` so audit completes even if client disconnects
4. **Privacy by default** - Sensitive data automatically redacted via pattern matching
5. **Error tolerance** - Errors in audit logging don't break API calls
6. **Metadata over data** - Captures what happened (actions, status codes, timing) without sensitive data

## Middleware Architecture

The audit logging middleware (`internal/middleware/audit_log.go`) runs after request handlers complete:

```
HTTP Request → Auth → Audit Middleware → Handler → Response Capture → Redaction → Async Log → Response
```

**Key Features:**

- **Automatic HTTP method mapping**: POST→create, GET→read, PUT→update, DELETE→delete
- **Path-based resource extraction**: `/api/v1/change-plans/{id}` → `change_plan`
- **Special action detection**: `/approve`, `/execute`, `/login`, etc.
- **IP address extraction**: Supports `X-Forwarded-For` and `X-Real-IP` headers
- **Async logging**: Non-blocking goroutine prevents request delays
- **Context isolation**: Uses `context.Background()` to prevent cancellation

## Handler Enrichment Pattern

Handlers can provide additional context using the `AuditEnrichment` pattern:

```go
c.Set("audit_enrichment", &middleware.AuditEnrichment{
    Email: email,
    Result: "success",
    Custom: map[string]interface{}{
        "login_method": "oauth",
    },
})
```

This enrichment is automatically merged into the audit log `Details` JSON and redacted if it contains sensitive fields.
