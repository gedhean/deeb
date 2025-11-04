# Phase 5: Audit Logging & Error Handling

**Duration**: ~3 days (25 milestones @ 1h each)
**Dependencies**: Phase 4
**Status**: 🚧 TODO

## Goal

Implement comprehensive audit logging system and robust error handling across all operations. This phase ensures full traceability and compliance requirements are met.

## Phase-level Acceptance Criteria

- [ ] Audit logging system implemented
- [ ] All user actions logged (login/logout, resource CRUD, Change Plan operations, approvals, executions)
- [ ] Audit log entries include: timestamp, user, action, target resource, result
- [ ] Change history stored for each Change Plan (detailed migration history)
- [ ] Before/after schema snapshots stored and queryable
- [ ] Audit log querying and filtering implemented
- [ ] Audit log UI available (admin only)
- [ ] Error handling middleware implemented
- [ ] SQL execution errors logged with full context
- [ ] Database connection failures handled gracefully
- [ ] Status transition errors handled with clear messages
- [ ] Input validation errors handled consistently
- [ ] Error responses standardized across API
- [ ] Error context captured for debugging
- [ ] Unit tests for audit logging passing
- [ ] Integration tests for error scenarios passing
- [ ] All linting checks passing

## Milestones

<!-- Milestones appended by /wiz-milestones -->

### P05M01: Create AuditLog repository interface and implementation

**Status:** ✅ COMPLETE
**ID:** P05M01

**Goal**
Create AuditLogRepository interface and PostgreSQL implementation for storing audit log entries.

**Acceptance Criteria**
- [x] AuditLogRepository interface defined in `backend/internal/repository/audit_log_repository.go`
- [x] Interface methods: Create, List, GetByUserID, GetByAction, GetByTargetType, Filter
- [x] PostgreSQL implementation created
- [x] All methods implemented with proper SQL queries
- [x] Filtering support implemented
- [x] Pagination support implemented
- [x] Error handling implemented

**Notes:**
- **Repository Interface**: Created `backend/internal/repository/audit_log_repository.go` with `AuditLogRepository` interface including methods: `Create`, `List`, `GetByUserID`, `GetByAction`, `GetByTargetType`, `Filter`, `Count`, and `CountWithFilters`. Also defined `AuditLogListFilters` struct for flexible filtering.
- **PostgreSQL Implementation**: Created `backend/internal/repository/postgres/audit_log_repository.go` with full implementation of all interface methods. All methods use proper SQL queries with parameterized statements, ordered by timestamp DESC for chronological ordering.
- **Filtering Support**: Implemented dynamic WHERE clause building in `Filter` method that supports filtering by `UserID`, `Action`, `ResourceType`, and `ResourceID`. Filter combinations are properly handled with AND conditions.
- **Pagination Support**: All list methods (`List`, `GetByUserID`, `GetByAction`, `GetByTargetType`, `Filter`) support pagination via `limit` and `offset` parameters. Pagination is implemented using PostgreSQL LIMIT and OFFSET clauses.
- **Error Handling**: All methods include proper error handling with context-aware error messages using `fmt.Errorf`. Errors are wrapped for better debugging context.
- **Code Quality**: Code compiles successfully, all linting checks pass (0 issues), and follows the same patterns as other repositories in the codebase (e.g., `ChangePlanRepository`, `DatabaseRepository`).

---

### P05M02: Design audit logging service architecture

**Status:** ✅ COMPLETE
**ID:** P05M02

**Goal**
Design audit logging service architecture with consistent logging patterns across the application.

**Acceptance Criteria**
- [x] Audit logging service architecture designed
- [x] Log entry structure defined
- [x] Action types enumerated
- [x] Target types enumerated
- [x] Logging patterns documented

**Notes:**
- **Architecture Design**: Created comprehensive design document `backend/docs/audit-logging-architecture.md` defining the audit logging service architecture with service layer pattern, log entry structure, and integration points.
- **Log Entry Structure**: Documented the complete AuditLog struct with all fields (ID, UserID, Action, ResourceType, ResourceID, Details, IPAddress, UserAgent, Timestamp) and their purposes.
- **Action Types Enumerated**: Defined comprehensive action types including authentication actions (login, logout, register), CRUD actions (create, read, update, delete), change plan actions (submit, approve, reject, execute, etc.), project actions, database actions, user actions, and environment actions. All actions follow consistent naming pattern: `{verb}_{resource_type}`.
- **Resource Types Enumerated**: Defined all resource types including core resources (user, project, environment, database, database_instance), change plan resources (change_plan, approval, execution_log, schema_snapshot), and system resources (audit_log, policy, policy_finding).
- **Logging Patterns Documented**: Documented 7 key logging patterns: action verb consistency, resource type consistency, details JSON structure, logging timing (when to log), error handling patterns, middleware integration, and service method patterns. Included examples for common scenarios (create, approve, execute, login).

---

### P05M03: Implement audit logging service

**Status:** ✅ COMPLETE
**ID:** P05M03

**Goal**
Create audit logging service that logs all user actions with consistent format.

**Acceptance Criteria**
- [x] Audit logging service created in `backend/internal/services/audit_log_service.go`
- [x] Log method implemented (logs action with user, target, result)
- [x] Action types defined: login, logout, create, update, delete, approve, execute, etc.
- [x] Target types defined: user, project, environment, database, change_plan, etc.
- [x] Result tracking (success, failure)
- [x] Unit tests passing

**Notes:**
- **Service Implementation**: Created `AuditLogService` in `backend/internal/services/audit_log_service.go` following existing service patterns. Service includes `auditLogRepo` and `logger` dependencies.
- **Log Methods**: Implemented four logging methods: `Log` (basic logging), `LogWithHTTP` (with IP address and user agent), `LogStateTransition` (for state changes like status transitions), and `LogWithResult` (for success/failure tracking).
- **Action Types Defined**: Defined comprehensive action type constants including authentication (login, logout, register), CRUD (create, read, update, delete), change plan actions (create, update, delete, submit, approve, reject, close, execute, assign_reviewer, unassign_reviewer), project actions, database actions, user actions, and environment actions.
- **Resource Types Defined**: Defined resource type constants including core resources (user, project, environment, database, database_instance), change plan resources (change_plan, approval, execution_log, schema_snapshot), and system resources (audit_log, policy, policy_finding).
- **Result Tracking**: Implemented `LogWithResult` method that accepts `result` parameter ("success" or "failure") and optional `errorMessage`. Results are stored in Details JSON along with any additional context.
- **IP Address Extraction**: Implemented `extractIPAddress` helper that handles X-Forwarded-For, X-Real-IP headers and RemoteAddr for accurate IP address capture behind proxies.
- **Unit Tests**: Created comprehensive unit tests in `audit_log_service_test.go` covering all methods: Log, LogWithHTTP, LogWithHTTP with X-Forwarded-For, LogStateTransition, LogWithResult (success and failure), error handling, and constant verification. All tests passing (9 tests).
- **Code Quality**: Code compiles successfully, all linting checks pass (0 issues), follows Go best practices and existing service patterns.

---

### P05M04: Implement audit logging middleware

**Status:** ✅ COMPLETE
**ID:** P05M04

**Goal**
Create middleware that automatically logs HTTP requests and responses for audit trail.

**Acceptance Criteria**
- [x] Audit logging middleware created in `backend/internal/middleware/audit_log.go`
- [x] Middleware captures request details (method, path, user)
- [x] Middleware captures response status
- [x] Audit log entry created for each request
- [x] Middleware integrated with Echo router
- [x] Unit tests passing

**Notes:**
- Created `AuditLogMiddleware` in `backend/internal/middleware/audit_log.go`
- Middleware captures request method, path, user ID (from context), and response status
- Automatically determines action type (create, read, update, delete, approve, execute) from HTTP method and path
- Extracts resource type and resource ID from path
- Skips health check endpoints and static assets
- Builds details JSON with method, path, status, result (success/failure), and error message
- Integrated with Echo router in `middleware.Setup()` function
- Added audit log service initialization in `app.New()`
- Comprehensive unit tests: 8 test cases covering authenticated/unauthenticated requests, different HTTP methods, resource IDs, error responses, health check skipping, and special actions (approve, execute)
- All tests passing, linting passes (0 issues)

---

### P05M05: Add audit logging to authentication endpoints

**Status:** ✅ COMPLETE
**ID:** P05M05

**Goal**
Add audit logging for login, logout, and registration actions.

**Acceptance Criteria**
- [x] Login action logged with success/failure
- [x] Logout action logged
- [x] Registration action logged
- [x] Failed login attempts logged
- [x] Audit log entries include user ID and IP address
- [x] Integration tests updated

**Notes:**
- Audit logging already implemented in LoginHandler, LogoutHandler, and RegisterHandler
- LoginHandler: Logs successful logins with `logAuditLoginSuccess` and failed logins with `logAuditLoginFailure` (covers rate limit, user not found, invalid password, inactive account, database errors)
- LogoutHandler: Logs logout actions for authenticated users
- RegisterHandler: Logs successful registrations and failed registrations (duplicate email)
- All audit log entries include user ID and IP address (via LogWithHTTP which extracts IP from request headers)
- Updated integration tests to pass auditService to all handlers
- Added 4 new integration tests: TestIntegration_Auth_AuditLogging_LoginSuccess, TestIntegration_Auth_AuditLogging_LoginFailure, TestIntegration_Auth_AuditLogging_Registration, TestIntegration_Auth_AuditLogging_Logout
- Tests verify audit log entries are created with correct action, resource type, user ID, IP address, and timestamp
- Updated unit tests to pass nil auditService (since they don't verify audit logging)
- All tests passing, linting passes (0 issues)

---

### P05M06: Add audit logging to resource management endpoints

**Status:** ✅ COMPLETE
**ID:** P05M06

**Goal**
Add audit logging for Projects, Environments, Database Instances, and Databases CRUD operations.

**Acceptance Criteria**
- [x] Create operations logged (project, environment, database_instance, database)
- [x] Update operations logged
- [x] Delete operations logged
- [x] Audit log entries include resource details
- [x] Integration tests updated

**Notes:**
- Added auditService field to ProjectHandler, EnvironmentHandler, DatabaseInstanceHandler, and DatabaseHandler
- Updated all handler constructors to accept auditService parameter
- Added audit logging calls in Create, Update, and Delete methods for all handlers:
  - ProjectHandler: CreateProject, UpdateProject, DeleteProject
  - EnvironmentHandler: CreateEnvironment, UpdateEnvironment, DeleteEnvironment
  - DatabaseInstanceHandler: CreateDatabaseInstance, UpdateDatabaseInstance, DeleteDatabaseInstance
  - DatabaseHandler: CreateDatabase, UpdateDatabase, DeleteDatabase
- Audit log entries include resource details (name, description, IDs, etc.) in JSON format
- Updated routes.go to pass auditService to all handler constructors
- Updated unit tests to pass nil auditService (since they don't verify audit logging)
- All tests passing, linting passes (0 issues)

---

### P05M07: Add audit logging to Change Plan operations

**Status:** ✅ COMPLETE
**ID:** P05M07

**Goal**
Add audit logging for Change Plan creation, updates, status transitions, and deletions.

**Acceptance Criteria**
- [x] Change Plan creation logged
- [x] Change Plan updates logged
- [x] Status transitions logged
- [x] Change Plan deletion logged
- [x] Audit log entries include Change Plan details
- [x] Integration tests updated

**Notes:**
- Added auditService field to ChangePlanHandler
- Updated NewChangePlanHandler constructor to accept auditService parameter
- Added audit logging calls in CreateChangePlan, UpdateChangePlan, DeleteChangePlan
- Added status transition logging in ApproveChangePlan (Open → Approved), CloseChangePlan (Open → Closed), ExecuteChangePlan (Approved → Completed/Failed)
- UpdateChangePlan logs both update action and status transitions when status changes
- ExecuteChangePlan logs execution action and status transition to final status
- Updated routes.go to pass auditService to ChangePlanHandler constructor
- All tests passing, linting passes (0 issues)

---

### P05M08: Add audit logging to approval operations

**Status:** ✅ COMPLETE
**ID:** P05M08

**Goal**
Add audit logging for approval and rejection actions.

**Acceptance Criteria**
- [x] Approval actions logged
- [x] Rejection/close actions logged
- [x] Audit log entries include approver, Change Plan, and comment
- [x] Integration tests updated

**Notes:**
- Audit logging for approval operations was already implemented in P05M07 in the handler layer (ApproveChangePlan and CloseChangePlan methods log status transitions)
- Added integration tests to verify audit logging works correctly:
  - TestIntegration_ChangePlan_AuditLogging_ApproveChangePlan: Verifies audit log entry created for approval with approver ID, change plan ID, comments, and IP address
  - TestIntegration_ChangePlan_AuditLogging_CloseChangePlan: Verifies audit log entry created for close/rejection with approver ID, change plan ID, comments, and IP address
- Updated setupChangePlanHandler and setupChangePlanHandlerWithTimeout to initialize auditLogService
- Added auditLogService to ChangePlanHandler constructor calls in test setup functions
- Tests verify audit log entries include approver, Change Plan details, and comments
- Tests verify IP address is logged
- All tests passing, linting passes (0 issues)

---

### P05M09: Add audit logging to execution operations

**Status:** ✅ COMPLETE
**ID:** P05M09

**Goal**
Add audit logging for Change Plan execution attempts and results.

**Acceptance Criteria**
- [x] Execution start logged
- [x] Execution success logged
- [x] Execution failure logged
- [x] Audit log entries include executor, Change Plan, and execution results
- [x] Integration tests updated

**Notes:**
- Execution start is logged automatically via audit logging middleware which captures all POST /api/v1/change-plans/:id/execute requests
- Execution success/failure is logged in `ExecuteChangePlan` handler after execution completes (lines 1953-1969 in `change_plan_handler.go`)
- Execution audit log entries include: executor ID, change plan ID, success status, rows affected, duration_ms
- Status transition from Approved to Completed/Failed is logged via `LogStateTransition`
- Added integration tests: `TestIntegration_ChangePlan_AuditLogging_ExecuteChangePlan_Success` and `TestIntegration_ChangePlan_AuditLogging_ExecuteChangePlan_Failure`
- Tests verify audit log entries include executor, Change Plan details, and execution results (success, rows_affected, duration_ms)
- All tests passing, linting passes (0 issues)

---

### P05M10: Implement Change Plan change history tracking

**Status:** ✅ COMPLETE
**ID:** P05M10

**Goal**
Implement detailed change history tracking for Change Plans including all modifications.

**Acceptance Criteria**
- [x] Change history model defined
- [x] Change history stored for each Change Plan modification
- [x] History includes: field changed, old value, new value, changed by, timestamp
- [x] Change history queryable
- [x] Integration test passing

**Notes:**
- Created ChangeHistory model in `backend/internal/models/change_history.go`
- Created migration `000017_add_change_history_table.up.sql` for change_history table
- Created ChangeHistoryRepository interface in `backend/internal/repository/change_history_repository.go`
- Created PostgreSQL implementation in `backend/internal/repository/postgres/change_history_repository.go`
- Added changeHistoryRepo to ChangePlanHandler struct and constructor
- Integrated change history tracking into UpdateChangePlan handler method
- Added trackChangeHistory helper function to track field-level changes (title, description, sql_content, rollback_sql, status, risk_level, assigned_reviewer_id, scheduled_for)
- Updated routes.go to initialize changeHistoryRepo and pass it to ChangePlanHandler
- Updated setupChangePlanHandler and setupChangePlanHandlerWithTimeout to initialize changeHistoryRepo
- Added integration test `TestIntegration_ChangePlan_ChangeHistoryTracking` to verify change history tracking works correctly
- Change history includes field name, old value, new value, changed by ID, and timestamp
- Change history is queryable via GetByChangePlanID and GetByChangePlanIDWithPagination methods
- All tests passing, linting passes (0 issues)

---

### P05M11: Implement audit log querying and filtering

**Status:** ✅ COMPLETE
**ID:** P05M11

**Goal**
Implement GET endpoint for querying audit logs with filtering options.

**Acceptance Criteria**
- [x] GET /api/v1/audit-logs endpoint implemented
- [x] Filter parameters: user_id, action, target_type, target_id, start_date, end_date
- [x] Pagination support implemented
- [x] Filtering implemented in repository layer
- [x] RBAC checks: admin only
- [x] Integration test passing

**Notes:**
- Added StartDate and EndDate filters to AuditLogListFilters in repository interface
- Updated Filter and CountWithFilters methods in PostgreSQL repository to support date filters
- Created AuditLogHandler with ListAuditLogs method that handles GET /api/v1/audit-logs
- Handler parses query parameters: user_id, action, target_type, target_id, start_date, end_date, limit, offset
- RBAC check enforced: only OrganizationRoleAdmin can access audit logs
- Added setupAuditLogRoutes function in routes.go to register the endpoint
- Endpoint returns paginated results with total count
- Added comprehensive integration test `TestIntegration_AuditLogs_ListAuditLogs` covering:
  - Admin access success
  - Developer access forbidden
  - Filter by user_id
  - Filter by action
  - Filter by resource_type (target_type)
  - Filter by resource_id (target_id)
  - Filter by date range (start_date, end_date)
  - Pagination (limit, offset)
  - Invalid date format handling
  - Invalid UUID handling
- All tests passing, linting passes (0 issues)

---

### P05M12: Design error handling middleware architecture

**Status:** ✅ COMPLETE
**ID:** P05M12

**Goal**
Design centralized error handling middleware for consistent error responses across API.

**Acceptance Criteria**
- [x] Error handling middleware architecture designed
- [x] Error response structure defined
- [x] Error types enumerated
- [x] Error handling patterns documented

**Notes:**
- Created comprehensive error handling architecture documentation in `backend/docs/error-handling-architecture.md`
- Documented existing CustomErrorHandler middleware architecture and registration
- Defined ErrorResponse structure: Error (HTTP status text), Message (detailed message), Code (optional application code)
- Enumerated error types:
  - Repository errors: ErrNotFound
  - Service errors: ErrChangePlanNotFound, ErrInvalidChangePlanStatus, ErrUnauthorizedApprover, ErrChangePlanAlreadyApproved, ErrChangePlanAlreadyClosed, ErrCannotApproveWithErrors
  - Model validation errors: ErrInvalidIssue, ErrInvalidChangePlan
  - Review errors: ErrValidatorNotFound, ErrValidatorAlreadyRegistered
- Documented error handling patterns:
  - Handler error pattern (using echo.HTTPError)
  - Repository error pattern (domain errors converted to HTTP errors)
  - Validation error pattern (400 Bad Request)
  - Authorization error pattern (401/403)
  - Service error pattern (error wrapping and conversion)
- Documented HTTP status code mapping (4xx and 5xx)
- Documented error logging strategy (server errors with stack trace, client errors at info level)
- Documented security considerations (error sanitization, no internal details exposed)
- Documented integration with other middleware (request logger, audit logging, recovery)
- Included best practices and examples
- All documentation complete, linting passes (0 issues)

---

### P05M13: Implement error handling middleware

**Status:** ✅ COMPLETE
**ID:** P05M13

**Goal**
Create error handling middleware that standardizes error responses and captures error context.

**Acceptance Criteria**
- [x] Error handling middleware created in `backend/internal/middleware/error_handler.go`
- [x] Standardized error response format implemented
- [x] Error context captured (request ID, user, path, method)
- [x] Error logging implemented
- [x] HTTP status codes mapped correctly
- [x] Unit tests passing

**Notes:**
- Enhanced existing CustomErrorHandler middleware in `backend/internal/middleware/error_handler.go`
- Updated ErrorResponse struct to include RequestID field
- Added user context capture (user ID and email) from Echo context using auth.GetUserFromEchoContext
- Enhanced error logging to include user context (user_id and user_email) when available
- Request ID (correlation ID) is now included in error responses for client reference
- Error context captured: request ID (correlation ID), user ID, user email, HTTP method, path
- Error logging implemented with different levels:
  - Server errors (5xx): Error level with stack trace
  - Client errors (4xx): Info level with error details
- HTTP status codes mapped correctly via echo.HTTPError
- Updated unit tests:
  - TestCustomErrorHandler_WithCorrelationID: Verifies request ID in response
  - TestCustomErrorHandler_WithUserContext: Verifies user context capture in logs
  - TestCustomErrorHandler_RequestIDInResponse: Verifies request ID inclusion
  - TestCustomErrorHandler_NoRequestIDWhenMissing: Verifies empty request ID when missing
- All existing tests still pass, new tests verify enhanced functionality
- All tests passing, linting passes (0 issues)

---

### P05M14: Implement error response structure

**Status:** ✅ COMPLETE
**ID:** P05M14

**Goal**
Define consistent error response structure across all API endpoints.

**Acceptance Criteria**
- [x] Error response struct defined
- [x] Error response includes: error, message, code, details, request_id
- [x] Error response format consistent across endpoints
- [x] Error response examples documented

**Notes:**
- Enhanced ErrorResponse struct to include Details field (map[string]interface{})
- Error response now includes all required fields: error, message, code, details, request_id
- Created HTTPErrorWithCode type and NewHTTPErrorWithCode helper function for errors with codes and details
- Defined error code constants in `backend/internal/middleware/error_codes.go`:
  - Validation errors: VALIDATION_ERROR, INVALID_INPUT, INVALID_FORMAT, MISSING_REQUIRED_FIELD
  - Authentication/Authorization: UNAUTHORIZED, FORBIDDEN, INVALID_TOKEN, EXPIRED_TOKEN, INSUFFICIENT_PERMISSIONS
  - Resource errors: RESOURCE_NOT_FOUND, RESOURCE_ALREADY_EXISTS, RESOURCE_CONFLICT, INVALID_STATE
  - Server errors: INTERNAL_ERROR, DATABASE_ERROR, EXTERNAL_SERVICE_ERROR, TIMEOUT
- Error response format is consistent across all endpoints via CustomErrorHandler middleware
- Updated error handling architecture documentation with:
  - Complete error response structure definition
  - Error code enumeration
  - Error response examples for all common scenarios (400, 401, 403, 404, 409, 422, 500)
  - Handler error pattern examples using NewHTTPErrorWithCode
- Added unit test TestCustomErrorHandler_WithCodeAndDetails to verify code and details handling
- All tests passing, linting passes (0 issues)

---

### P05M15: Implement SQL execution error handling

**Status:** ✅ COMPLETE
**ID:** P05M15

**Goal**
Add comprehensive error handling for SQL execution failures with full context logging.

**Acceptance Criteria**
- [x] SQL execution errors captured with full context
- [x] Error logging includes: SQL content, database, error message, stack trace
- [x] Error responses include helpful error messages
- [x] Sensitive information masked in error responses
- [x] Error handling tests passing

**Notes:**
- Created `SQLExecutionError` type in `backend/internal/services/sql_execution_error.go` to capture database-specific error details
- `SQLExecutionError` includes: database type/name/ID/instance, SQL content, database error code/message/detail/hint/position, stack trace, and user-friendly message
- Enhanced `executeInTransaction` in `backend/internal/services/execution_service.go` to create `SQLExecutionError` when SQL execution fails
- Comprehensive error logging added with all database-specific fields (error code, message, detail, hint, position)
- SQL content masking implemented via `MaskSQLContent` function to mask passwords, credit cards, SSNs, and API keys
- User-friendly error messages generated based on PostgreSQL and MySQL error codes (e.g., unique violations, foreign key violations, syntax errors)
- Enhanced `ExecuteChangePlan` error handling to use `SQLExecutionError` and return user-friendly messages
- Created comprehensive unit tests in `backend/internal/services/sql_execution_error_test.go` covering:
  - PostgreSQL error handling (extracts error code, message, detail, hint, position)
  - MySQL error handling (extracts error number and message)
  - Generic error handling
  - User message generation for common error codes
  - SQL content masking
- All tests passing, linting passes (0 issues)

---

### P05M16: Implement database connection failure handling

**Status:** ✅ COMPLETE
**ID:** P05M16

**Goal**
Add graceful error handling for database connection failures.

**Acceptance Criteria**
- [x] Connection failure errors handled gracefully
- [x] Error messages clear and actionable
- [x] Connection retry logic implemented (optional)
- [x] Error logging includes connection details
- [x] Error handling tests passing

**Notes:**
- Created `ConnectionError` type in `backend/internal/database/connection_error.go` to capture structured connection error details
- `ConnectionError` includes: database type/name/ID/instance, host, port, username, SSL settings, error classification, error codes, stack trace, and retry information
- Error classification implemented: network, authentication, timeout, database_not_found, permission, other
- Database-specific error code extraction for PostgreSQL (pq.Error) and MySQL (MySQLError)
- User-friendly error messages with actionable hints based on error type:
  - Network errors: hints about server running, host/port, network connectivity, firewall
  - Authentication errors: hints about username/password and permissions
  - Timeout errors: hints about server response and network latency
  - Database not found: hints about database name and existence
  - Permission errors: hints about user permissions
- Connection retry logic implemented in `connectWithRetry` function with exponential backoff (available for future use, marked as optional)
- RetryConfig struct with configurable parameters (MaxAttempts, InitialDelay, MaxDelay, BackoffMultiplier)
- Comprehensive error logging in `ConnectionManager` includes: database_id, database_name, database_instance_name, database_type, host, port, username, ssl_required, error_type, error_code, stack_trace
- Error logging implemented for both MySQL and PostgreSQL connection failures (connection and ping failures)
- Created comprehensive unit tests in `backend/internal/database/connection_error_test.go` covering:
  - Network error classification
  - Timeout error classification
  - PostgreSQL error classification
  - MySQL error classification
  - Database not found error classification
  - Error message generation
  - Retry information tracking
- All tests passing, linting passes (0 issues)

---

### P05M17: Implement status transition error handling

**Status:** ✅ COMPLETE
**ID:** P05M17

**Goal**
Add clear error messages for invalid status transition attempts.

**Acceptance Criteria**
- [x] Status transition errors handled with clear messages
- [x] Error messages explain why transition is invalid
- [x] Error responses include current status and attempted transition
- [x] Error handling tests passing

**Notes:**
- Created `ErrInvalidStatusTransition` error type in `backend/internal/workflow/change_plan_status.go`
- Error messages clearly explain why transitions are invalid:
  - Invalid source/target status messages
  - Terminal state messages (explains that terminal states cannot be changed)
  - Valid transitions guidance (lists all valid transitions from current status)
  - Reason field for additional context
- Error messages include helpful information:
  - Current status and attempted status
  - List of valid transitions from current status
  - Whether current status is terminal
  - Specific reason for the error
- Error responses include structured details:
  - `current_status`: Current status of the Change Plan
  - `attempted_status`: The status that was attempted
  - `valid_transitions`: List of valid transitions from current status (if applicable)
  - `is_terminal_status`: Whether current status is terminal
  - `reason`: Additional reason for the error
- Error handling implemented in both handler (`change_plan_handler.go` lines 700-717) and middleware (`change_plan_status.go` lines 94-100)
- Handler returns structured JSON response with error message and details
- Middleware validates status transitions before handler execution
- Created comprehensive unit tests in `backend/internal/workflow/change_plan_status_test.go` and `change_plan_status_error_test.go` covering:
  - Error message generation for invalid transitions
  - Error message generation for invalid source/target statuses
  - Terminal status error messages
  - Valid transitions guidance in error messages
  - Error details structure
- Created middleware tests in `backend/internal/middleware/change_plan_status_test.go` verifying:
  - Invalid transition error responses include current and attempted status
  - Error messages explain why transition is invalid
  - Error details are properly structured
- All tests passing, linting passes (0 issues)

---

### P05M18: Implement input validation error handling

**Status:** 🚧 TODO
**ID:** P05M18

**Goal**
Add consistent error handling for input validation errors across all endpoints.

**Acceptance Criteria**
- [ ] Input validation errors handled consistently
- [ ] Error messages include field names and validation rules
- [ ] Error responses include validation details
- [ ] Error handling tests passing

---

### P05M19: Implement request ID tracking

**Status:** 🚧 TODO
**ID:** P05M19

**Goal**
Add request ID tracking for correlating logs and errors across the application.

**Acceptance Criteria**
- [ ] Request ID middleware created
- [ ] Request ID generated for each request
- [ ] Request ID included in logs and error responses
- [ ] Request ID correlation working
- [ ] Unit tests passing

---

### P05M20: Add error context to execution logs

**Status:** 🚧 TODO
**ID:** P05M20

**Goal**
Ensure execution errors include full context for debugging.

**Acceptance Criteria**
- [ ] Execution error logging includes full context
- [ ] Context includes: Change Plan ID, SQL content, database, user, timestamp
- [ ] Error context stored in execution logs
- [ ] Error context queryable

---

### P05M21: Write unit tests for AuditLog repository

**Status:** 🚧 TODO
**ID:** P05M21

**Goal**
Create comprehensive unit tests for AuditLog repository operations.

**Acceptance Criteria**
- [ ] Unit tests for all AuditLog repository methods
- [ ] Test coverage >85%
- [ ] Filtering tests passing
- [ ] Pagination tests passing
- [ ] Edge cases covered
- [ ] All tests passing

---

### P05M22: Write unit tests for audit logging service

**Status:** 🚧 TODO
**ID:** P05M22

**Goal**
Create comprehensive unit tests for audit logging service.

**Acceptance Criteria**
- [ ] Unit tests for audit logging service (>85% coverage)
- [ ] Action logging tests
- [ ] Target type tests
- [ ] Result tracking tests
- [ ] Edge cases covered
- [ ] All tests passing

---

### P05M23: Write unit tests for error handling middleware

**Status:** 🚧 TODO
**ID:** P05M23

**Goal**
Create comprehensive unit tests for error handling middleware.

**Acceptance Criteria**
- [ ] Unit tests for error handling middleware (>85% coverage)
- [ ] Error response format tests
- [ ] Error context capture tests
- [ ] Error logging tests
- [ ] Edge cases covered
- [ ] All tests passing

---

### P05M24: Write integration tests for audit logging

**Status:** 🚧 TODO
**ID:** P05M24

**Goal**
Create integration tests for audit logging across all endpoints.

**Acceptance Criteria**
- [ ] Integration test for authentication audit logging passing
- [ ] Integration test for resource CRUD audit logging passing
- [ ] Integration test for Change Plan audit logging passing
- [ ] Integration test for approval audit logging passing
- [ ] Integration test for execution audit logging passing
- [ ] Audit log querying tests passing
- [ ] All integration tests passing

---

### P05M25: Verify Phase 5 Completion

**Status:** 🚧 TODO
**ID:** P05M25

**Goal**
Verify that all Phase 5 requirements are met, tests pass, and the phase is ready for sign-off before proceeding to Phase 6.

**Acceptance Criteria**
- [ ] All previous milestones in Phase 5 marked as complete
- [ ] All tests pass (unit, integration)
- [ ] Code coverage meets project standards (>85% for business logic)
- [ ] No outstanding bugs or issues from this phase
- [ ] Code reviewed and meets quality standards
- [ ] All linting checks passing (golangci-lint)
- [ ] Audit logging working end-to-end
- [ ] Error handling consistent across API
- [ ] Ready to proceed to Phase 6

---
