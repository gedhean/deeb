# Phase 4: Approval Workflow & Execution Engine

**Duration**: ~5 days (38 milestones @ 1h each)
**Dependencies**: Phase 3
**Status**: 🚧 TODO

## Goal

Implement approval workflow system and change execution engine with transaction management, schema snapshot capture, and concurrency control. This phase enables the complete Change Plan lifecycle from approval through execution with safety guarantees.

## Phase-level Acceptance Criteria

- [ ] Approval workflow engine implemented (single-approval flow)
- [ ] Project Approvers can approve or close Change Plans
- [ ] Approval comments and history tracking functional
- [ ] Optional reviewer assignment working
- [ ] Approval requires no SQL Review errors
- [ ] Change Plan execution engine implemented
- [ ] Only Approved Change Plans can be executed
- [ ] Execution permissions enforced (Organization Admin, Organization DBA, Project Owner, Change Plan Author)
- [ ] Schema snapshot capture before execution (pg_dump for PostgreSQL, mysqldump/information_schema for MySQL)
- [ ] Schema snapshot capture after successful execution
- [ ] Schema snapshots stored securely (BYTEA or JSONB column)
- [ ] SQL execution wrapped in transaction (SERIALIZABLE isolation level)
- [ ] Automatic transaction rollback on error
- [ ] Execution results captured (success/error, rows affected, duration)
- [ ] Execution results stored in Change Plan run log history
- [ ] Concurrency control implemented (one Change Plan per database at a time)
- [ ] Change Plan queuing for concurrent execution requests
- [ ] Concurrent changes on different databases allowed
- [ ] Error handling and logging for execution failures
- [ ] Change Plan status updates (InProgress → Completed/Failed)
- [ ] Unit tests for execution engine passing
- [ ] Integration tests for approval and execution workflow passing
- [ ] All linting checks passing

## Milestones

<!-- Milestones appended by /wiz-milestones -->

### P04M01: Create Approval repository interface and implementation

**Status:** 🚧 TODO
**ID:** P04M01

**Goal**
Create ApprovalRepository interface and PostgreSQL implementation for storing approval records.

**Acceptance Criteria**
- [ ] ApprovalRepository interface defined in `backend/internal/repository/approval_repository.go`
- [ ] Interface methods: Create, GetByChangePlanID, GetByApproverID, GetHistory
- [ ] PostgreSQL implementation created
- [ ] All methods implemented with proper SQL queries
- [ ] Approval history ordering by created_at DESC
- [ ] Error handling implemented

---

### P04M02: Design approval workflow engine architecture

**Status:** 🚧 TODO
**ID:** P04M02

**Goal**
Design single-approval workflow engine architecture for Change Plan approval process.

**Acceptance Criteria**
- [ ] Approval workflow service architecture designed
- [ ] Single-approval flow documented
- [ ] Approval state machine defined
- [ ] Approval requirements documented (SQL Review errors check)
- [ ] Architecture supports approval comments and history

---

### P04M03: Implement approval workflow service

**Status:** 🚧 TODO
**ID:** P04M03

**Goal**
Create approval workflow service that handles approval and rejection logic.

**Acceptance Criteria**
- [ ] Approval service created in `backend/internal/services/approval_service.go`
- [ ] Approve method implemented (checks SQL Review errors, updates status)
- [ ] Close method implemented (rejects Change Plan, updates status)
- [ ] Approval validation logic implemented
- [ ] Error handling for invalid approvals
- [ ] Unit tests for approval service passing

---

### P04M04: Implement approval POST endpoint

**Status:** 🚧 TODO
**ID:** P04M04

**Goal**
Implement POST /api/v1/change-plans/:id/approve endpoint for approving Change Plans.

**Acceptance Criteria**
- [ ] POST /api/v1/change-plans/:id/approve endpoint implemented
- [ ] Request validation (optional comment field)
- [ ] RBAC check: Project Approver role required
- [ ] SQL Review errors check (must have no errors)
- [ ] Change Plan status updated to Approved
- [ ] Approval record created
- [ ] Integration test passing

---

### P04M05: Implement close/reject POST endpoint

**Status:** 🚧 TODO
**ID:** P04M05

**Goal**
Implement POST /api/v1/change-plans/:id/close endpoint for rejecting Change Plans.

**Acceptance Criteria**
- [ ] POST /api/v1/change-plans/:id/close endpoint implemented
- [ ] Request validation (optional comment field)
- [ ] RBAC check: Project Approver role required
- [ ] Change Plan status updated to Closed
- [ ] Approval record created with action "close"
- [ ] Integration test passing

---

### P04M06: Implement approval history GET endpoint

**Status:** 🚧 TODO
**ID:** P04M06

**Goal**
Implement GET endpoint for retrieving approval history for a Change Plan.

**Acceptance Criteria**
- [ ] GET /api/v1/change-plans/:id/approvals endpoint implemented
- [ ] Returns approval history ordered by created_at DESC
- [ ] Response includes approver details, action, comment, timestamp
- [ ] Authorization checks implemented
- [ ] Integration test passing

---

### P04M07: Implement optional reviewer assignment

**Status:** 🚧 TODO
**ID:** P04M07

**Goal**
Allow Change Plan authors to assign optional reviewers (Project Approvers).

**Acceptance Criteria**
- [ ] PUT /api/v1/change-plans/:id/reviewer endpoint implemented
- [ ] Request validation (reviewer_id must be Project Approver)
- [ ] assigned_reviewer_id field updated
- [ ] RBAC checks: Author can assign reviewer
- [ ] Reviewer validation implemented
- [ ] Integration test passing

---

### P04M08: Design execution engine architecture

**Status:** 🚧 TODO
**ID:** P04M08

**Goal**
Design Change Plan execution engine architecture with transaction management and snapshot capture.

**Acceptance Criteria**
- [ ] Execution engine architecture designed
- [ ] Transaction management strategy documented
- [ ] Schema snapshot capture strategy documented
- [ ] Concurrency control strategy documented
- [ ] Error handling strategy documented

---

### P04M09: Implement database connection manager

**Status:** 🚧 TODO
**ID:** P04M09

**Goal**
Create database connection manager for connecting to target databases (MySQL and PostgreSQL).

**Acceptance Criteria**
- [ ] Database connection manager created in `backend/internal/database/connection_manager.go`
- [ ] MySQL connection function implemented
- [ ] PostgreSQL connection function implemented
- [ ] Credential decryption before connection
- [ ] Connection pooling configured
- [ ] Error handling for connection failures
- [ ] Unit tests passing

---

### P04M10: Implement schema snapshot capture for PostgreSQL

**Status:** 🚧 TODO
**ID:** P04M10

**Goal**
Create schema snapshot capture utility using pg_dump for PostgreSQL databases.

**Acceptance Criteria**
- [ ] Schema snapshot service created in `backend/internal/services/schema_snapshot.go`
- [ ] PostgreSQL snapshot function implemented (pg_dump --schema-only)
- [ ] Snapshot stored as byte array or JSON
- [ ] Snapshot compression considered (optional)
- [ ] Error handling for snapshot failures
- [ ] Unit tests for PostgreSQL snapshot passing

---

### P04M11: Implement schema snapshot capture for MySQL

**Status:** 🚧 TODO
**ID:** P04M11

**Goal**
Create schema snapshot capture utility using mysqldump or information_schema queries for MySQL databases.

**Acceptance Criteria**
- [ ] MySQL snapshot function implemented (mysqldump --no-data or information_schema queries)
- [ ] Snapshot stored as byte array or JSON
- [ ] Snapshot format consistent with PostgreSQL
- [ ] Error handling for snapshot failures
- [ ] Unit tests for MySQL snapshot passing

---

### P04M12: Implement SchemaSnapshot repository

**Status:** 🚧 TODO
**ID:** P04M12

**Goal**
Create repository for storing and retrieving schema snapshots.

**Acceptance Criteria**
- [ ] SchemaSnapshotRepository interface defined
- [ ] Interface methods: Create, GetByChangePlanID, GetByDatabaseID
- [ ] PostgreSQL implementation created
- [ ] BYTEA or JSONB column handling implemented
- [ ] Snapshot retrieval working
- [ ] Unit tests passing

---

### P04M13: Implement execution service with transaction management

**Status:** 🚧 TODO
**ID:** P04M13

**Goal**
Create execution service that wraps SQL execution in transactions with SERIALIZABLE isolation level.

**Acceptance Criteria**
- [ ] Execution service created in `backend/internal/services/execution_service.go`
- [ ] Execute method implemented
- [ ] Transaction wrapping implemented (SERIALIZABLE isolation)
- [ ] Transaction commit on success
- [ ] Transaction rollback on error
- [ ] Error handling implemented
- [ ] Unit tests passing

---

### P04M14: Implement execution result capture

**Status:** 🚧 TODO
**ID:** P04M14

**Goal**
Capture execution results including success/error status, rows affected, and duration.

**Acceptance Criteria**
- [ ] Execution result struct defined
- [ ] Result capture implemented (success/error, rows affected, duration)
- [ ] Execution timing implemented
- [ ] Error message capture implemented
- [ ] Results stored in Change Plan
- [ ] Unit tests passing

---

### P04M15: Implement Change Plan execution workflow

**Status:** 🚧 TODO
**ID:** P04M15

**Goal**
Implement complete execution workflow: snapshot → execute → snapshot → update status.

**Acceptance Criteria**
- [ ] Execution workflow implemented in execution service
- [ ] Before snapshot captured
- [ ] SQL executed in transaction
- [ ] After snapshot captured on success
- [ ] Status updated (InProgress → Completed/Failed)
- [ ] Error handling at each step
- [ ] Unit tests passing

---

### P04M16: Implement execution POST endpoint

**Status:** 🚧 TODO
**ID:** P04M16

**Goal**
Implement POST /api/v1/change-plans/:id/execute endpoint for executing Change Plans.

**Acceptance Criteria**
- [ ] POST /api/v1/change-plans/:id/execute endpoint implemented
- [ ] RBAC checks: Organization Admin, Organization DBA, Project Owner, or Change Plan Author
- [ ] Status validation (only Approved Change Plans can be executed)
- [ ] Execution workflow triggered
- [ ] Response includes execution results
- [ ] Integration test passing

---

### P04M17: Implement execution permission checking

**Status:** 🚧 TODO
**ID:** P04M17

**Goal**
Create authorization middleware or helper for checking execution permissions.

**Acceptance Criteria**
- [ ] Execution permission checker created in `backend/internal/middleware/execution_auth.go`
- [ ] Permission check logic: Organization Admin, Organization DBA, Project Owner, or Change Plan Author
- [ ] Permission checker integrated with execution endpoint
- [ ] 403 Forbidden response for unauthorized users
- [ ] Unit tests for permission checking passing

---

### P04M18: Implement concurrency control system

**Status:** 🚧 TODO
**ID:** P04M18

**Goal**
Create concurrency control system ensuring one Change Plan executes per database at a time.

**Acceptance Criteria**
- [ ] Execution lock manager created in `backend/internal/services/execution_lock.go`
- [ ] Lock acquisition per database implemented
- [ ] Lock release on completion/failure
- [ ] Lock timeout handling
- [ ] Concurrent execution on different databases allowed
- [ ] Unit tests for locking passing

---

### P04M19: Implement Change Plan execution queue

**Status:** 🚧 TODO
**ID:** P04M19

**Goal**
Create execution queue system for queuing Change Plans when database is locked.

**Acceptance Criteria**
- [ ] Execution queue service created
- [ ] Queue implementation (in-memory or database-backed)
- [ ] Queue processing logic implemented
- [ ] Queue status tracking (pending, in-progress, completed, failed)
- [ ] Queue position reporting
- [ ] Unit tests passing

---

### P04M20: Implement execution status updates

**Status:** 🚧 TODO
**ID:** P04M20

**Goal**
Update Change Plan status throughout execution lifecycle (Approved → InProgress → Completed/Failed).

**Acceptance Criteria**
- [ ] Status update logic implemented
- [ ] Status updated to InProgress on execution start
- [ ] Status updated to Completed on success
- [ ] Status updated to Failed on error
- [ ] Status updates atomic and consistent
- [ ] Unit tests passing

---

### P04M21: Implement execution error handling

**Status:** 🚧 TODO
**ID:** P04M21

**Goal**
Create comprehensive error handling for execution failures with rollback and logging.

**Acceptance Criteria**
- [ ] Execution error handling implemented
- [ ] Transaction rollback on error
- [ ] Error logging with full context
- [ ] Error messages captured and stored
- [ ] Error notifications (logged)
- [ ] Unit tests for error scenarios passing

---

### P04M22: Implement execution run log history

**Status:** 🚧 TODO
**ID:** P04M22

**Goal**
Create execution run log system for tracking multiple execution attempts.

**Acceptance Criteria**
- [ ] Execution log model defined
- [ ] Execution log storage implemented
- [ ] Run log history tracked per Change Plan
- [ ] Log includes: timestamp, status, duration, rows affected, error message
- [ ] GET endpoint for run log history
- [ ] Integration test passing

---

### P04M23: Add execution results to Change Plan response

**Status:** 🚧 TODO
**ID:** P04M23

**Goal**
Include execution results and run log history in Change Plan detail response.

**Acceptance Criteria**
- [ ] Change Plan detail response includes execution results
- [ ] Response includes last execution status
- [ ] Response includes execution run log history
- [ ] Response includes execution duration and rows affected
- [ ] Integration test updated

---

### P04M24: Implement execution timeout handling

**Status:** 🚧 TODO
**ID:** P04M24

**Goal**
Add timeout handling for SQL execution (5 minutes maximum per PRD).

**Acceptance Criteria**
- [ ] Execution timeout configured (5 minutes default, configurable)
- [ ] Timeout enforced during SQL execution
- [ ] Timeout error handled gracefully
- [ ] Change Plan marked as Failed on timeout
- [ ] Unit tests for timeout passing

---

### P04M25: Implement SQL content size validation

**Status:** 🚧 TODO
**ID:** P04M25

**Goal**
Add validation for SQL content size limit (10MB maximum per PRD).

**Acceptance Criteria**
- [ ] SQL content size validation implemented
- [ ] 10MB limit enforced
- [ ] Validation error returned for oversized SQL
- [ ] Size validation on Change Plan creation and update
- [ ] Unit tests passing

---

### P04M26: Write unit tests for Approval repository

**Status:** 🚧 TODO
**ID:** P04M26

**Goal**
Create comprehensive unit tests for Approval repository operations.

**Acceptance Criteria**
- [ ] Unit tests for all Approval repository methods
- [ ] Test coverage >85%
- [ ] Edge cases covered
- [ ] Error scenarios tested
- [ ] All tests passing

---

### P04M27: Write unit tests for approval workflow service

**Status:** 🚧 TODO
**ID:** P04M27

**Goal**
Create comprehensive unit tests for approval workflow logic.

**Acceptance Criteria**
- [ ] Unit tests for approve method (>85% coverage)
- [ ] Unit tests for close method (>85% coverage)
- [ ] SQL Review error check tests
- [ ] Status transition tests
- [ ] Edge cases covered
- [ ] All tests passing

---

### P04M28: Write unit tests for execution service

**Status:** 🚧 TODO
**ID:** P04M28

**Goal**
Create comprehensive unit tests for execution service including transaction management.

**Acceptance Criteria**
- [ ] Unit tests for execution service (>85% coverage)
- [ ] Transaction wrapping tests
- [ ] Rollback tests
- [ ] Success scenario tests
- [ ] Error scenario tests
- [ ] All tests passing

---

### P04M29: Write unit tests for schema snapshot capture

**Status:** 🚧 TODO
**ID:** P04M29

**Goal**
Create comprehensive unit tests for schema snapshot capture utilities.

**Acceptance Criteria**
- [ ] Unit tests for PostgreSQL snapshot (>85% coverage)
- [ ] Unit tests for MySQL snapshot (>85% coverage)
- [ ] Snapshot format validation tests
- [ ] Error handling tests
- [ ] All tests passing

---

### P04M30: Write unit tests for concurrency control

**Status:** 🚧 TODO
**ID:** P04M30

**Goal**
Create comprehensive unit tests for execution locking and queue system.

**Acceptance Criteria**
- [ ] Unit tests for execution lock manager (>85% coverage)
- [ ] Lock acquisition/release tests
- [ ] Concurrent lock tests
- [ ] Queue tests
- [ ] Edge cases covered
- [ ] All tests passing

---

### P04M31: Write integration tests for approval workflow

**Status:** 🚧 TODO
**ID:** P04M31

**Goal**
Create integration tests for approval endpoints and workflow.

**Acceptance Criteria**
- [ ] Integration test for approve endpoint passing
- [ ] Integration test for close endpoint passing
- [ ] Integration test for approval history passing
- [ ] SQL Review error blocking tests
- [ ] RBAC tests included
- [ ] All integration tests passing

---

### P04M32: Write integration tests for execution workflow

**Status:** 🚧 TODO
**ID:** P04M32

**Goal**
Create integration tests for Change Plan execution including snapshot capture.

**Acceptance Criteria**
- [ ] Integration test for execution endpoint passing
- [ ] Execution success scenario test
- [ ] Execution failure scenario test
- [ ] Snapshot capture tests
- [ ] Transaction rollback tests
- [ ] All integration tests passing

---

### P04M33: Write integration tests for concurrency control

**Status:** 🚧 TODO
**ID:** P04M33

**Goal**
Create integration tests for concurrent execution requests and locking.

**Acceptance Criteria**
- [ ] Integration test for concurrent execution blocking passing
- [ ] Integration test for different database concurrent execution passing
- [ ] Queue handling tests
- [ ] Lock release tests
- [ ] All integration tests passing

---

### P04M34: Write integration tests for error scenarios

**Status:** 🚧 TODO
**ID:** P04M34

**Goal**
Create integration tests for execution error scenarios and rollback.

**Acceptance Criteria**
- [ ] Integration test for SQL syntax error handling passing
- [ ] Integration test for timeout handling passing
- [ ] Integration test for connection failure handling passing
- [ ] Rollback verification tests
- [ ] Error logging tests
- [ ] All integration tests passing

---

### P04M35: Update API documentation for approval and execution

**Status:** 🚧 TODO
**ID:** P04M35

**Goal**
Update OpenAPI specification with all approval and execution endpoints.

**Acceptance Criteria**
- [ ] OpenAPI spec updated with approval endpoints
- [ ] OpenAPI spec updated with execution endpoints
- [ ] Request/response schemas documented
- [ ] Execution result schema documented
- [ ] Swagger UI displays all endpoints correctly

---

### P04M36: Implement execution monitoring and logging

**Status:** 🚧 TODO
**ID:** P04M36

**Goal**
Add comprehensive logging for execution operations for debugging and monitoring.

**Acceptance Criteria**
- [ ] Execution logging implemented
- [ ] Log entries for execution start, progress, completion, failure
- [ ] Execution metrics logged (duration, rows affected)
- [ ] Error context logged
- [ ] Structured logging format used

---

### P04M37: Performance optimization for execution

**Status:** 🚧 TODO
**ID:** P04M37

**Goal**
Optimize execution performance for large SQL files and concurrent requests.

**Acceptance Criteria**
- [ ] Execution performance analyzed
- [ ] Bottlenecks identified and addressed
- [ ] Connection pooling optimized
- [ ] Snapshot capture optimized
- [ ] Performance tests passing

---

### P04M38: Verify Phase 4 Completion

**Status:** 🚧 TODO
**ID:** P04M38

**Goal**
Verify that all Phase 4 requirements are met, tests pass, and the phase is ready for sign-off before proceeding to Phase 5.

**Acceptance Criteria**
- [ ] All previous milestones in Phase 4 marked as complete
- [ ] All tests pass (unit, integration)
- [ ] Code coverage meets project standards (>85% for business logic)
- [ ] No outstanding bugs or issues from this phase
- [ ] Code reviewed and meets quality standards
- [ ] All linting checks passing (golangci-lint)
- [ ] Approval workflow working end-to-end
- [ ] Execution engine working end-to-end
- [ ] Concurrency control verified
- [ ] Ready to proceed to Phase 5

---
