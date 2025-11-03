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

**Status:** 🚧 TODO
**ID:** P05M02

**Goal**
Design audit logging service architecture with consistent logging patterns across the application.

**Acceptance Criteria**
- [ ] Audit logging service architecture designed
- [ ] Log entry structure defined
- [ ] Action types enumerated
- [ ] Target types enumerated
- [ ] Logging patterns documented

---

### P05M03: Implement audit logging service

**Status:** 🚧 TODO
**ID:** P05M03

**Goal**
Create audit logging service that logs all user actions with consistent format.

**Acceptance Criteria**
- [ ] Audit logging service created in `backend/internal/services/audit_log_service.go`
- [ ] Log method implemented (logs action with user, target, result)
- [ ] Action types defined: login, logout, create, update, delete, approve, execute, etc.
- [ ] Target types defined: user, project, environment, database, change_plan, etc.
- [ ] Result tracking (success, failure)
- [ ] Unit tests passing

---

### P05M04: Implement audit logging middleware

**Status:** 🚧 TODO
**ID:** P05M04

**Goal**
Create middleware that automatically logs HTTP requests and responses for audit trail.

**Acceptance Criteria**
- [ ] Audit logging middleware created in `backend/internal/middleware/audit_log.go`
- [ ] Middleware captures request details (method, path, user)
- [ ] Middleware captures response status
- [ ] Audit log entry created for each request
- [ ] Middleware integrated with Echo router
- [ ] Unit tests passing

---

### P05M05: Add audit logging to authentication endpoints

**Status:** 🚧 TODO
**ID:** P05M05

**Goal**
Add audit logging for login, logout, and registration actions.

**Acceptance Criteria**
- [ ] Login action logged with success/failure
- [ ] Logout action logged
- [ ] Registration action logged
- [ ] Failed login attempts logged
- [ ] Audit log entries include user ID and IP address
- [ ] Integration tests updated

---

### P05M06: Add audit logging to resource management endpoints

**Status:** 🚧 TODO
**ID:** P05M06

**Goal**
Add audit logging for Projects, Environments, Database Instances, and Databases CRUD operations.

**Acceptance Criteria**
- [ ] Create operations logged (project, environment, database_instance, database)
- [ ] Update operations logged
- [ ] Delete operations logged
- [ ] Audit log entries include resource details
- [ ] Integration tests updated

---

### P05M07: Add audit logging to Change Plan operations

**Status:** 🚧 TODO
**ID:** P05M07

**Goal**
Add audit logging for Change Plan creation, updates, status transitions, and deletions.

**Acceptance Criteria**
- [ ] Change Plan creation logged
- [ ] Change Plan updates logged
- [ ] Status transitions logged
- [ ] Change Plan deletion logged
- [ ] Audit log entries include Change Plan details
- [ ] Integration tests updated

---

### P05M08: Add audit logging to approval operations

**Status:** 🚧 TODO
**ID:** P05M08

**Goal**
Add audit logging for approval and rejection actions.

**Acceptance Criteria**
- [ ] Approval actions logged
- [ ] Rejection/close actions logged
- [ ] Audit log entries include approver, Change Plan, and comment
- [ ] Integration tests updated

---

### P05M09: Add audit logging to execution operations

**Status:** 🚧 TODO
**ID:** P05M09

**Goal**
Add audit logging for Change Plan execution attempts and results.

**Acceptance Criteria**
- [ ] Execution start logged
- [ ] Execution success logged
- [ ] Execution failure logged
- [ ] Audit log entries include executor, Change Plan, and execution results
- [ ] Integration tests updated

---

### P05M10: Implement Change Plan change history tracking

**Status:** 🚧 TODO
**ID:** P05M10

**Goal**
Implement detailed change history tracking for Change Plans including all modifications.

**Acceptance Criteria**
- [ ] Change history model defined
- [ ] Change history stored for each Change Plan modification
- [ ] History includes: field changed, old value, new value, changed by, timestamp
- [ ] Change history queryable
- [ ] Integration test passing

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
