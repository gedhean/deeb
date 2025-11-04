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
- Audit logging for execution operations was already implemented in P05M07 in the handler layer (ExecuteChangePlan method logs execution action and status transition)
- ExecuteChangePlan logs execution action with LogWithHTTP (includes success, rows_affected, duration_ms)
- ExecuteChangePlan logs status transition from Approved to Completed/Failed using LogStateTransition
- Execution start is logged via HTTP request middleware (audit log middleware logs all requests)
- Added integration tests to verify audit logging works correctly:
  - TestIntegration_ChangePlan_AuditLogging_ExecuteChangePlan_Success: Verifies audit log entry created for successful execution with executor ID, change plan ID, execution results (success, rows_affected, duration_ms), and IP address
  - TestIntegration_ChangePlan_AuditLogging_ExecuteChangePlan_Failure: Verifies audit log entry created for failed execution with executor ID, change plan ID, execution results (success=false, duration_ms), and IP address
- Tests verify audit log entries include executor, Change Plan details, and execution results
- Tests verify IP address is logged
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
- Added integration test TestIntegration_ChangePlan_ChangeHistoryTracking to verify change history tracking works correctly
- Change history includes field name, old value, new value, changed by ID, and timestamp
- Change history is queryable via GetByChangePlanID and GetByChangePlanIDWithPagination methods
- All tests passing, linting passes (0 issues)

---

### P05M11: Implement audit log querying and filtering

**Status:** 🚧 TODO
**ID:** P05M11

**Goal**
Implement GET endpoint for querying audit logs with filtering options.

**Acceptance Criteria**
- [ ] GET /api/v1/audit-logs endpoint implemented
- [ ] Filter parameters: user_id, action, target_type, target_id, start_date, end_date
- [ ] Pagination support implemented
- [ ] Filtering implemented in repository layer
- [ ] RBAC checks: admin only
- [ ] Integration test passing

---

### P05M12: Design error handling middleware architecture

**Status:** 🚧 TODO
**ID:** P05M12

**Goal**
Design centralized error handling middleware for consistent error responses across API.

**Acceptance Criteria**
- [ ] Error handling middleware architecture designed
- [ ] Error response structure defined
- [ ] Error types enumerated
- [ ] Error handling patterns documented

---

### P05M13: Implement error handling middleware

**Status:** 🚧 TODO
**ID:** P05M13

**Goal**
Create error handling middleware that standardizes error responses and captures error context.

**Acceptance Criteria**
- [ ] Error handling middleware created in `backend/internal/middleware/error_handler.go`
- [ ] Standardized error response format implemented
- [ ] Error context captured (request ID, user, path, method)
- [ ] Error logging implemented
- [ ] HTTP status codes mapped correctly
- [ ] Unit tests passing

---

### P05M14: Implement error response structure

**Status:** 🚧 TODO
**ID:** P05M14

**Goal**
Define consistent error response structure across all API endpoints.

**Acceptance Criteria**
- [ ] Error response struct defined
- [ ] Error response includes: error, message, code, details, request_id
- [ ] Error response format consistent across endpoints
- [ ] Error response examples documented

---

### P05M15: Implement SQL execution error handling

**Status:** 🚧 TODO
**ID:** P05M15

**Goal**
Add comprehensive error handling for SQL execution failures with full context logging.

**Acceptance Criteria**
- [ ] SQL execution errors captured with full context
- [ ] Error logging includes: SQL content, database, error message, stack trace
- [ ] Error responses include helpful error messages
- [ ] Sensitive information masked in error responses
- [ ] Error handling tests passing

---

### P05M16: Implement database connection failure handling

**Status:** 🚧 TODO
**ID:** P05M16

**Goal**
Add graceful error handling for database connection failures.

**Acceptance Criteria**
- [ ] Connection failure errors handled gracefully
- [ ] Error messages clear and actionable
- [ ] Connection retry logic implemented (optional)
- [ ] Error logging includes connection details
- [ ] Error handling tests passing

---

### P05M17: Implement status transition error handling

**Status:** 🚧 TODO
**ID:** P05M17

**Goal**
Add clear error messages for invalid status transition attempts.

**Acceptance Criteria**
- [ ] Status transition errors handled with clear messages
- [ ] Error messages explain why transition is invalid
- [ ] Error responses include current status and attempted transition
- [ ] Error handling tests passing

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
