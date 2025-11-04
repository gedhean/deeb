# Error Handling Middleware Architecture

This document defines the architecture, design patterns, and conventions for error handling in the Deeb API.

## Overview

The error handling middleware provides centralized error handling for consistent error responses across all API endpoints. It ensures that all errors are properly formatted, logged, and returned to clients in a standardized format.

## Architecture

### Error Handler Middleware

The error handling middleware (`CustomErrorHandler`) is registered as Echo's HTTP error handler and intercepts all errors that occur during request processing.

```
┌─────────────────┐
│  HTTP Request   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Echo Router   │
│   + Handlers    │
└────────┬────────┘
         │
         │ (on error)
         ▼
┌─────────────────┐
│CustomErrorHandler│
│  (middleware)    │
└────────┬────────┘
         │
         ├─► Log Error (zap)
         │
         └─► JSON Response
```

### Error Handler Registration

The error handler is registered during middleware setup:

```go
// In middleware.Setup()
e.HTTPErrorHandler = CustomErrorHandler
```

This ensures all errors from handlers, middleware, and Echo itself are processed consistently.

## Error Response Structure

### Standard Error Response Format

All API errors are returned in the following JSON structure:

```json
{
  "error": "Bad Request",
  "message": "Invalid request body",
  "code": "INVALID_REQUEST"
}
```

### ErrorResponse Type

```go
type ErrorResponse struct {
    Error   string `json:"error"`             // HTTP status text (e.g., "Bad Request")
    Message string `json:"message,omitempty"` // Detailed error message
    Code    string `json:"code,omitempty"`    // Application-specific error code
}
```

**Field Details:**

- **Error**: HTTP status text corresponding to the status code (e.g., "Bad Request", "Internal Server Error")
- **Message**: Human-readable error message describing what went wrong
- **Code**: Optional application-specific error code for programmatic error handling

### Error Response Examples

#### 400 Bad Request
```json
{
  "error": "Bad Request",
  "message": "Invalid change plan ID format"
}
```

#### 401 Unauthorized
```json
{
  "error": "Unauthorized",
  "message": "Authentication required"
}
```

#### 403 Forbidden
```json
{
  "error": "Forbidden",
  "message": "Admin access required"
}
```

#### 404 Not Found
```json
{
  "error": "Not Found",
  "message": "Change plan not found"
}
```

#### 500 Internal Server Error
```json
{
  "error": "Internal Server Error",
  "message": "Internal server error"
}
```

**Note:** Server errors (5xx) do not expose internal error details to clients for security reasons.

## Error Types

### Repository Layer Errors

#### Common Repository Errors

- **`repository.ErrNotFound`**: Resource not found in database
  - **HTTP Status**: 404 Not Found
  - **Usage**: Returned when querying by ID returns no results

### Service Layer Errors

Service layer errors are typically wrapped and returned as `echo.HTTPError`:

#### Execution Service Errors

- **`services.ErrChangePlanNotFound`**: Change plan doesn't exist
- **`services.ErrInvalidChangePlanStatus`**: Invalid status for operation
- **`services.ErrUnauthorizedApprover`**: User lacks approval permission
- **`services.ErrChangePlanAlreadyApproved`**: Change plan already approved
- **`services.ErrChangePlanAlreadyClosed`**: Change plan already closed

#### Approval Service Errors

- Validation errors for approval operations
- Status transition errors
- Authorization errors

### Model Validation Errors

#### Custom Error Types

- **`models.ErrInvalidIssue`**: Issue validation failed
- **`models.ErrInvalidChangePlan`**: Change plan validation failed

These are typically converted to `echo.HTTPError` with status 400 Bad Request.

### HTTP Errors

#### Echo HTTPError

Most errors in handlers are returned as `echo.HTTPError`:

```go
return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
```

The error handler automatically:
- Extracts the HTTP status code
- Formats the error message
- Returns standardized JSON response

## Error Handling Patterns

### 1. Handler Error Pattern

Handlers should return errors using `echo.HTTPError`:

```go
func (h *ChangePlanHandler) GetChangePlan(c echo.Context) error {
    changePlanID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "Invalid change plan ID format")
    }

    changePlan, err := h.changePlanRepo.GetByID(ctx, changePlanID)
    if err != nil {
        if err == repository.ErrNotFound {
            return echo.NewHTTPError(http.StatusNotFound, "Change plan not found")
        }
        h.logger.Error("Failed to get change plan", zap.Error(err))
        return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve change plan")
    }

    return c.JSON(http.StatusOK, changePlan.ToResponse())
}
```

### 2. Repository Error Pattern

Repositories return domain errors that handlers convert to HTTP errors:

```go
// Repository layer
func (r *ChangePlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ChangePlan, error) {
    // ...
    if errors.Is(err, sql.ErrNoRows) {
        return nil, repository.ErrNotFound
    }
    return nil, err
}

// Handler layer
changePlan, err := h.changePlanRepo.GetByID(ctx, changePlanID)
if err != nil {
    if err == repository.ErrNotFound {
        return echo.NewHTTPError(http.StatusNotFound, "Change plan not found")
    }
    // Handle other errors
}
```

### 3. Validation Error Pattern

Validation errors are returned as 400 Bad Request:

```go
if err := changePlan.Validate(); err != nil {
    return echo.NewHTTPError(http.StatusBadRequest, err.Error())
}
```

### 4. Authorization Error Pattern

Authorization errors return appropriate status codes:

- **401 Unauthorized**: Authentication required
- **403 Forbidden**: Insufficient permissions

```go
claims, ok := auth.GetUserFromEchoContext(c)
if !ok || claims == nil {
    return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required")
}

if claims.OrganizationRole != string(models.OrganizationRoleAdmin) {
    return echo.NewHTTPError(http.StatusForbidden, "Admin access required")
}
```

### 5. Service Error Pattern

Service layer errors are wrapped and returned:

```go
err = h.approvalService.Approve(ctx, changePlanID, claims.UserID, req.Comments)
if err != nil {
    if err == services.ErrChangePlanNotFound {
        return echo.NewHTTPError(http.StatusNotFound, "Change plan not found")
    }
    if err == services.ErrInvalidChangePlanStatus {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }
    if err == services.ErrUnauthorizedApprover {
        return echo.NewHTTPError(http.StatusForbidden, err.Error())
    }
    // Generic error
    h.logger.Error("Failed to approve change plan", zap.Error(err))
    return echo.NewHTTPError(http.StatusInternalServerError, "Failed to approve change plan")
}
```

## HTTP Status Code Mapping

### Client Errors (4xx)

| Status Code | Meaning | Example Use Case |
|------------|---------|----------------|
| 400 Bad Request | Invalid request format or parameters | Invalid UUID format, missing required fields |
| 401 Unauthorized | Authentication required | Missing or invalid token |
| 403 Forbidden | Insufficient permissions | Non-admin accessing admin endpoint |
| 404 Not Found | Resource doesn't exist | Change plan ID not found |
| 409 Conflict | Resource conflict | Database locked, duplicate email |
| 422 Unprocessable Entity | Validation failed | Invalid data format, business rule violation |

### Server Errors (5xx)

| Status Code | Meaning | Example Use Case |
|------------|---------|----------------|
| 500 Internal Server Error | Unexpected server error | Database connection failed, unhandled exception |
| 502 Bad Gateway | Upstream service error | External service unavailable |
| 503 Service Unavailable | Service temporarily unavailable | Maintenance mode, rate limit exceeded |

## Error Logging

### Logging Strategy

The error handler logs errors based on severity:

#### Server Errors (5xx)
- **Log Level**: Error
- **Includes**: Stack trace, correlation ID, request details
- **Purpose**: Debugging and monitoring

```go
logger.Error("Server error",
    zap.String("correlation_id", correlationID),
    zap.String("method", c.Request().Method),
    zap.String("path", c.Request().URL.Path),
    zap.Int("status", code),
    zap.Error(err),
    zap.String("stack", string(debug.Stack())),
)
```

#### Client Errors (4xx)
- **Log Level**: Info
- **Includes**: Correlation ID, request details, error message
- **Purpose**: Request tracking and auditing

```go
logger.Info("Client error",
    zap.String("correlation_id", correlationID),
    zap.String("method", c.Request().Method),
    zap.String("path", c.Request().URL.Path),
    zap.Int("status", code),
    zap.String("error", errorDetail),
)
```

### Correlation IDs

Correlation IDs are included in error logs for request tracing:

- Generated by request logger middleware
- Included in all error log entries
- Helps correlate errors across distributed systems

## Error Handler Behavior

### Response Already Committed

If the response has already been committed, the error handler does nothing:

```go
if c.Response().Committed {
    return
}
```

This prevents attempting to write to an already-sent response.

### Error Message Extraction

The error handler extracts messages from different error types:

1. **Echo HTTPError**: Extracts code and message
2. **Standard errors**: Uses error message as-is
3. **Custom error types**: Converts to string

### Security Considerations

#### Server Error Sanitization

Server errors (5xx) are sanitized before returning to clients:

- Internal error details are not exposed
- Generic error message returned instead
- Full details logged server-side for debugging

```go
if code >= 500 {
    // Don't expose internal error details to client
    errorDetail = message
}
```

#### Client Error Details

Client errors (4xx) include detailed messages:

- Validation errors: Specific field and reason
- Authorization errors: Clear permission requirements
- Not found errors: Resource type and ID

## Error Context Capture

### Request Context

The error handler captures request context for logging:

- **Method**: HTTP method (GET, POST, PUT, DELETE)
- **Path**: Request path
- **Correlation ID**: Request correlation ID
- **Status Code**: HTTP status code

### User Context

User context is captured when available:

- User ID from authentication context
- Organization role
- IP address (from audit logging middleware)

## Error Response Consistency

### Current State

The error handler provides consistent error responses:

- ✅ All errors return JSON format
- ✅ Standard error structure (Error, Message, Code)
- ✅ Appropriate HTTP status codes
- ✅ Security-conscious error messages

### Future Enhancements

Potential improvements for P05M13:

1. **Request ID**: Include request ID in error responses
2. **Error Codes**: Standardize application error codes
3. **Error Details**: Structured error details for validation errors
4. **Multi-Language**: Support for localized error messages

## Integration with Other Middleware

### Request Logger Middleware

- Generates correlation IDs
- Logs request/response details
- Provides context for error logging

### Audit Logging Middleware

- Logs error responses to audit trail
- Captures error context for compliance
- Tracks error patterns

### Recovery Middleware

- Catches panics
- Prevents server crashes
- Routes panics to error handler

## Best Practices

### 1. Use Specific Error Types

Prefer specific error types over generic errors:

```go
// Good
if err == repository.ErrNotFound {
    return echo.NewHTTPError(http.StatusNotFound, "Change plan not found")
}

// Avoid
if err != nil {
    return echo.NewHTTPError(http.StatusInternalServerError, "Error occurred")
}
```

### 2. Include Context in Errors

Include relevant context in error messages:

```go
// Good
return echo.NewHTTPError(http.StatusBadRequest, 
    fmt.Sprintf("Invalid change plan ID format: %s", changePlanIDStr))

// Avoid
return echo.NewHTTPError(http.StatusBadRequest, "Invalid ID")
```

### 3. Log Before Returning

Log errors before returning them:

```go
if err != nil {
    h.logger.Error("Failed to process request", zap.Error(err))
    return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process request")
}
```

### 4. Use Appropriate Status Codes

Match HTTP status codes to error types:

- Validation errors → 400 Bad Request
- Authentication errors → 401 Unauthorized
- Permission errors → 403 Forbidden
- Not found → 404 Not Found
- Conflicts → 409 Conflict
- Server errors → 500 Internal Server Error

### 5. Don't Expose Internal Details

Never expose internal system details:

```go
// Good
return echo.NewHTTPError(http.StatusInternalServerError, "Failed to process request")

// Bad
return echo.NewHTTPError(http.StatusInternalServerError, 
    fmt.Sprintf("Database error: %v", dbErr))
```

## Examples

### Example 1: Resource Not Found

```go
changePlan, err := h.changePlanRepo.GetByID(ctx, changePlanID)
if err != nil {
    if err == repository.ErrNotFound {
        return echo.NewHTTPError(http.StatusNotFound, "Change plan not found")
    }
    h.logger.Error("Failed to get change plan", zap.Error(err))
    return echo.NewHTTPError(http.StatusInternalServerError, "Failed to retrieve change plan")
}
```

**Error Response:**
```json
{
  "error": "Not Found",
  "message": "Change plan not found"
}
```

### Example 2: Validation Error

```go
if err := changePlan.Validate(); err != nil {
    return echo.NewHTTPError(http.StatusBadRequest, err.Error())
}
```

**Error Response:**
```json
{
  "error": "Bad Request",
  "message": "title is required"
}
```

### Example 3: Authorization Error

```go
if claims.OrganizationRole != string(models.OrganizationRoleAdmin) {
    return echo.NewHTTPError(http.StatusForbidden, "Admin access required")
}
```

**Error Response:**
```json
{
  "error": "Forbidden",
  "message": "Admin access required"
}
```

### Example 4: Service Error

```go
err = h.approvalService.Approve(ctx, changePlanID, claims.UserID, req.Comments)
if err != nil {
    if err == services.ErrChangePlanNotFound {
        return echo.NewHTTPError(http.StatusNotFound, "Change plan not found")
    }
    if err == services.ErrInvalidChangePlanStatus {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }
    h.logger.Error("Failed to approve change plan", zap.Error(err))
    return echo.NewHTTPError(http.StatusInternalServerError, "Failed to approve change plan")
}
```

## Conclusion

The error handling middleware provides a centralized, consistent approach to error handling across the API. It ensures:

1. **Consistency**: All errors follow the same format
2. **Security**: Internal errors are not exposed
3. **Observability**: All errors are logged with context
4. **Maintainability**: Error handling logic is centralized

Future enhancements (P05M13) will expand on this foundation to add request IDs, standardized error codes, and structured error details.

