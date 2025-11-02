# Phase 3: Change Plan Management & SQL Review Engine

**Duration**: ~5 days (40 milestones @ 1h each)
**Dependencies**: Phase 2
**Status**: 🚧 TODO

## Goal

Implement Change Plan lifecycle management and comprehensive SQL Review engine with multiple validators. This phase delivers the core workflow for creating, reviewing, and managing database change requests with automated policy validation.

## Phase-level Acceptance Criteria

- [ ] Change Plan CRUD operations fully implemented (create, read, update, delete, list)
- [ ] Change Plan status workflow operational (Draft → Open → Approved/Closed → InProgress → Completed/Failed)
- [ ] Status transition rules enforced with proper validation
- [ ] Change Plan filtering implemented (status, project, database, author)
- [ ] Risk level assignment automated based on SQL content, target database, and environment
- [ ] SQL Review engine architecture implemented (pluggable validator system)
- [ ] SQL syntax validator implemented (MySQL and PostgreSQL parsers)
- [ ] Security validator implemented (detects destructive operations, DROP on tables with data)
- [ ] Performance validator implemented (missing indexes, no primary key)
- [ ] Backwards compatibility validator implemented (breaking changes detection)
- [ ] Best practices validator implemented (naming conventions, data types)
- [ ] SQL Review runs automatically on Change Plan submission (Draft → Open)
- [ ] SQL Review can be triggered manually from Change Plan details page
- [ ] SQL Review results displayed with severity levels (Error, Warning, Pass)
- [ ] SQL Review errors block approval/execution
- [ ] SQL Review warnings shown but don't block
- [ ] Policy findings stored and queryable
- [ ] Change Plan resubmission after error fixes working
- [ ] Unit tests for all validators passing (>85% coverage)
- [ ] Integration tests for Change Plan workflow passing
- [ ] All linting checks passing

## Milestones

<!-- Milestones appended by /wiz-milestones -->

### P03M01: Create ChangePlan repository interface and implementation

**Status:** ✅ COMPLETE
**ID:** P03M01

**Goal**
Create ChangePlanRepository interface and PostgreSQL implementation for Change Plan CRUD operations.

**Acceptance Criteria**
- [x] ChangePlanRepository interface defined in `backend/internal/repository/change_plan_repository.go`
- [x] Interface methods: Create, GetByID, Update, Delete, List, Count, GetByStatus, GetByProjectID, GetByDatabaseID, GetByAuthorID
- [x] PostgreSQL implementation created in `backend/internal/repository/postgres/change_plan_repository.go`
- [x] All methods implemented with proper SQL queries
- [x] Error handling implemented
- [x] Foreign key relationships handled correctly

---

### P03M02: Create ChangePlan model struct aligned with schema

**Status:** ✅ COMPLETE
**ID:** P03M02

**Goal**
Create ChangePlan model struct (renamed from Issue) with all fields matching the new schema.

**Acceptance Criteria**
- [x] ChangePlan struct defined in `backend/internal/models/change_plan.go`
- [x] All fields match schema: id, project_id, target_database_id, title, change_type, sql_content, description, rollback_sql, status, risk_level, author_id, assigned_reviewer_id, created_at, updated_at
- [x] Status constants defined: Draft, Open, Approved, Closed, InProgress, Completed, Failed, Cancelled
- [x] RiskLevel constants defined: Low, Medium, High
- [x] ChangeType constants defined: DDL, DML
- [x] JSON and database tags properly defined
- [x] Validation method implemented

---

### P03M03: Implement Change Plan status workflow logic

**Status:** ✅ COMPLETE
**ID:** P03M03

**Goal**
Implement status transition rules and validation: Draft → Open → Approved/Closed → InProgress → Completed/Failed.

**Acceptance Criteria**
- [x] Status transition rules defined in `backend/internal/workflow/change_plan_status.go`
- [x] Transition validation function implemented
- [x] Valid transitions: Draft→Open, Open→Approved, Open→Closed, Approved→InProgress, InProgress→Completed, InProgress→Failed
- [x] Invalid transitions rejected with clear error messages
- [x] Transition helper methods implemented
- [x] Unit tests for status transitions passing

---

### P03M04: Implement Change Plan POST endpoint

**Status:** ✅ COMPLETE
**ID:** P03M04

**Goal**
Implement POST /api/v1/change-plans endpoint for creating Change Plans (Project Developer role required).

**Acceptance Criteria**
- [x] Change Plan handler created in `backend/internal/handlers/change_plan_handler.go`
- [x] POST /api/v1/change-plans endpoint implemented
- [x] Request validation (title, change_type, sql_content, target_database_id required)
- [x] Optional fields: description, rollback_sql
- [x] RBAC check: Project Developer role in project
- [x] Change Plan created with Draft status
- [x] Author ID set from authenticated user
- [ ] Integration test passing

---

### P03M05: Implement Change Plan GET endpoints

**Status:** ✅ COMPLETE
**ID:** P03M05

**Goal**
Implement GET endpoints for retrieving individual Change Plans and listing with filters.

**Acceptance Criteria**
- [x] GET /api/v1/change-plans/:id endpoint implemented
- [x] GET /api/v1/change-plans endpoint implemented (list with filters)
- [x] Filter parameters: status, project_id, database_id, author_id
- [x] 404 Not Found for nonexistent Change Plans
- [x] Authorization checks (users can view Change Plans in their projects)
- [x] Response includes Change Plan details
- [ ] Integration tests passing

---

### P03M06: Implement Change Plan PUT endpoint

**Status:** ✅ COMPLETE
**ID:** P03M06

**Goal**
Implement PUT endpoint for updating Change Plans with status transition validation.

**Acceptance Criteria**
- [x] PUT /api/v1/change-plans/:id endpoint implemented
- [x] Update validation (only Draft status can be modified)
- [x] Status transition validation on update
- [x] RBAC checks: Project Developer role required
- [x] Only author can update their Draft Change Plans
- [x] SQL content size limit enforced (10MB)
- [ ] Integration test passing

---

### P03M07: Implement Change Plan DELETE endpoint

**Status:** ✅ COMPLETE
**ID:** P03M07

**Goal**
Implement DELETE endpoint for Change Plans with proper authorization and status checks.

**Acceptance Criteria**
- [x] DELETE /api/v1/change-plans/:id endpoint implemented
- [x] Delete validation (only Draft or Closed Change Plans can be deleted)
- [x] RBAC checks: Project Developer role required
- [x] Only author can delete their Change Plans
- [x] Cascade deletion of policy findings
- [ ] Integration test passing

---

### P03M08: Implement automated risk level assignment

**Status:** ✅ COMPLETE
**ID:** P03M08

**Goal**
Create risk assessment logic that assigns risk level based on SQL content, target database, and environment.

**Acceptance Criteria**
- [x] Risk assessment service created in `backend/internal/services/risk_assessment.go`
- [x] Risk factors analyzed: SQL content (destructive operations), environment (Production = higher risk), database type
- [x] Risk level calculation: Low, Medium, High
- [x] Risk level assigned on Change Plan creation
- [x] Risk level updated on SQL content change
- [x] Unit tests for risk assessment passing (>85% coverage)

---

### P03M09: Design SQL Review engine architecture

**Status:** ✅ COMPLETE
**ID:** P03M09

**Goal**
Design pluggable validator system architecture for SQL Review engine with extensible validator interface.

**Acceptance Criteria**
- [x] Validator interface defined in `backend/internal/review/validator.go`
- [x] Validator interface methods: Validate, GetName, GetSeverity
- [x] Review engine architecture documented
- [x] Validator registration system designed
- [x] Review result structure defined (violations, severity, messages)
- [x] Architecture supports multiple validators running in parallel

---

### P03M10: Create Validator interface and base implementation

**Status:** ✅ COMPLETE
**ID:** P03M10

**Goal**
Implement Validator interface and base validator infrastructure for SQL Review engine.

**Acceptance Criteria**
- [x] Validator interface implemented in `backend/internal/review/validator.go`
- [x] BaseValidator struct created with common fields
- [x] Severity enum defined: Error, Warning, Info
- [x] Violation struct defined with fields: validator_type, severity, message, rule_name, line_number, column_number
- [x] Validator registry pattern implemented
- [x] Unit tests for base infrastructure passing

---

### P03M11: Implement SQL syntax validator for PostgreSQL

**Status:** ✅ COMPLETE
**ID:** P03M11

**Goal**
Create PostgreSQL syntax validator using a SQL parser library (e.g., github.com/pganalyze/pg_query_go or similar).

**Acceptance Criteria**
- [x] PostgreSQL syntax validator created in `backend/internal/review/validators/postgres_syntax.go`
- [x] SQL parser library integrated (add dependency if needed)
- [x] Syntax validation implemented
- [x] Syntax errors detected and reported with line numbers
- [x] Error severity: Error (blocks approval)
- [x] Unit tests for PostgreSQL syntax validation passing (>85% coverage)

**Note:** pg_query_go has compilation issues on macOS due to C code compatibility, but works correctly in Linux containers. Integration tests should be run in a Linux environment or Docker container.

---

### P03M12: Implement SQL syntax validator for MySQL

**Status:** ✅ COMPLETE
**ID:** P03M12

**Goal**
Create MySQL syntax validator using a SQL parser library (e.g., github.com/xwb1989/sqlparser or similar).

**Acceptance Criteria**
- [x] MySQL syntax validator created in `backend/internal/review/validators/mysql_syntax.go`
- [x] SQL parser library integrated (add dependency if needed)
- [x] Syntax validation implemented
- [x] Syntax errors detected and reported with line numbers
- [x] Error severity: Error (blocks approval)
- [x] Unit tests for MySQL syntax validation passing (>85% coverage)

---

### P03M13: Implement security validator

**Status:** ✅ COMPLETE
**ID:** P03M13

**Goal**
Create validator detecting destructive operations and security risks (DROP TABLE, DROP DATABASE, TRUNCATE, etc.).

**Acceptance Criteria**
- [x] Security validator created in `backend/internal/review/validators/security.go`
- [x] Destructive operations detected: DROP TABLE, DROP DATABASE, TRUNCATE TABLE, DROP COLUMN
- [x] DROP operations on tables with data detected (higher risk)
- [x] Security violations reported with severity (Error or Warning)
- [x] Specific security rules implemented (e.g., no DROP without backup)
- [x] Unit tests for security validator passing (>85% coverage)

---

### P03M14: Implement performance validator

**Status:** 🚧 TODO
**ID:** P03M14

**Goal**
Create validator detecting performance issues (missing indexes, tables without primary keys, etc.).

**Acceptance Criteria**
- [ ] Performance validator created in `backend/internal/review/validators/performance.go`
- [ ] Missing indexes detected (CREATE TABLE without indexes)
- [ ] Tables without primary keys detected
- [ ] Large tables without indexes on foreign keys detected
- [ ] Performance violations reported with severity (Warning)
- [ ] Unit tests for performance validator passing (>85% coverage)

---

### P03M15: Implement backwards compatibility validator

**Status:** 🚧 TODO
**ID:** P03M15

**Goal**
Create validator detecting breaking changes (DROP COLUMN, RENAME TABLE, data type changes, etc.).

**Acceptance Criteria**
- [ ] Backwards compatibility validator created in `backend/internal/review/validators/backwards_compatibility.go`
- [ ] Breaking changes detected: DROP COLUMN, RENAME TABLE, RENAME COLUMN, data type changes
- [ ] ALTER TABLE operations analyzed for breaking changes
- [ ] Backwards compatibility violations reported with severity (Warning)
- [ ] Unit tests for backwards compatibility validator passing (>85% coverage)

---

### P03M16: Implement best practices validator

**Status:** 🚧 TODO
**ID:** P03M16

**Goal**
Create validator checking naming conventions and data type best practices.

**Acceptance Criteria**
- [ ] Best practices validator created in `backend/internal/review/validators/best_practices.go`
- [ ] Naming convention checks (table names, column names, index names)
- [ ] Data type best practices (no TEXT for small strings, proper index types)
- [ ] Best practice violations reported with severity (Info or Warning)
- [ ] Configurable rules for naming conventions
- [ ] Unit tests for best practices validator passing (>85% coverage)

---

### P03M17: Implement SQL Review orchestration service

**Status:** 🚧 TODO
**ID:** P03M17

**Goal**
Create service that runs all validators and aggregates results into policy findings. Service loads Policy (SQL_REVIEW type) from Environment via Change Plan's target database.

**Acceptance Criteria**
- [ ] Review service created in `backend/internal/review/service.go`
- [ ] Review method loads Policy (SQL_REVIEW type) from Environment (via Change Plan's database)
- [ ] Policy config determines which validators are enabled
- [ ] Review method runs enabled validators based on policy
- [ ] Validator results aggregated
- [ ] Policy findings created from validator results
- [ ] Review results include summary (errors count, warnings count)
- [ ] Error handling for validator failures
- [ ] Unit tests for review orchestration passing

---

### P03M18: Implement PolicyFinding repository

**Status:** 🚧 TODO
**ID:** P03M18

**Goal**
Create repository for storing and querying policy findings from SQL Review.

**Acceptance Criteria**
- [ ] PolicyFindingRepository interface defined
- [ ] Interface methods: Create, GetByChangePlanID, DeleteByChangePlanID, CountBySeverity
- [ ] PostgreSQL implementation created
- [ ] Batch insert for multiple findings
- [ ] Querying by change plan ID and severity working
- [ ] Unit tests for repository passing

---

### P03M19: Implement automatic SQL Review on submission

**Status:** 🚧 TODO
**ID:** P03M19

**Goal**
Trigger SQL Review automatically when Change Plan transitions from Draft to Open status.

**Acceptance Criteria**
- [ ] Status transition handler updated
- [ ] SQL Review triggered on Draft → Open transition
- [ ] Review results stored as policy findings
- [ ] Change Plan status updated to Open only if no errors
- [ ] Review errors prevent status transition
- [ ] Integration test for automatic review passing

---

### P03M20: Implement manual SQL Review trigger endpoint

**Status:** 🚧 TODO
**ID:** P03M20

**Goal**
Create POST /api/v1/change-plans/:id/review endpoint for manually triggering SQL Review.

**Acceptance Criteria**
- [ ] POST /api/v1/change-plans/:id/review endpoint implemented
- [ ] Endpoint triggers SQL Review for existing Change Plan
- [ ] Review results returned in response
- [ ] Policy findings updated/created
- [ ] RBAC checks: Project Developer role required
- [ ] Integration test passing

---

### P03M21: Implement Change Plan filtering

**Status:** 🚧 TODO
**ID:** P03M21

**Goal**
Add filtering support for status, project, database, author in Change Plan list endpoint.

**Acceptance Criteria**
- [ ] GET /api/v1/change-plans endpoint updated with filter parameters
- [ ] Filter parameters: status, project_id, database_id, author_id
- [ ] Filtering implemented in repository layer
- [ ] Multiple filters can be combined
- [ ] Filtered results paginated correctly
- [ ] Integration tests for filtering passing

---

### P03M22: Implement Change Plan status transition validation middleware

**Status:** 🚧 TODO
**ID:** P03M22

**Goal**
Add validation middleware that prevents invalid status transitions at API level.

**Acceptance Criteria**
- [ ] Status transition validation middleware created
- [ ] Middleware validates status transitions on PUT requests
- [ ] Invalid transitions return 400 Bad Request with clear error
- [ ] Middleware integrated with Change Plan routes
- [ ] Unit tests for middleware passing

---

### P03M23: Add SQL Review results to Change Plan response

**Status:** 🚧 TODO
**ID:** P03M23

**Goal**
Include policy findings in Change Plan detail response with severity summary.

**Acceptance Criteria**
- [ ] Change Plan detail response includes policy findings
- [ ] Findings grouped by severity (Error, Warning, Info)
- [ ] Finding count summary included
- [ ] Response includes has_errors flag
- [ ] Response includes has_warnings flag
- [ ] Integration test updated

---

### P03M24: Verify SQL Review errors block approval

**Status:** 🚧 TODO
**ID:** P03M24

**Goal**
Ensure Change Plans with SQL Review errors cannot be approved or executed.

**Acceptance Criteria**
- [ ] Approval endpoint checks for SQL Review errors
- [ ] Change Plans with errors cannot transition to Approved
- [ ] Error response returned with list of errors
- [ ] Execution endpoint checks for errors (safety check)
- [ ] Integration tests for error blocking passing

---

### P03M25: Verify SQL Review warnings don't block

**Status:** 🚧 TODO
**ID:** P03M25

**Goal**
Ensure Change Plans with only warnings can be approved and executed.

**Acceptance Criteria**
- [ ] Approval endpoint allows approval when only warnings exist
- [ ] Warnings displayed but don't prevent approval
- [ ] Change Plans with warnings can transition to Approved
- [ ] Warnings logged but execution proceeds
- [ ] Integration tests for warning handling passing

---

### P03M26: Implement Change Plan resubmission logic

**Status:** 🚧 TODO
**ID:** P03M26

**Goal**
Allow authors to fix errors and resubmit Change Plans (Open → Draft → Open again).

**Acceptance Criteria**
- [ ] Resubmission endpoint or logic implemented
- [ ] Authors can move Open Change Plans back to Draft
- [ ] SQL Review runs again on resubmission (Draft → Open)
- [ ] Old policy findings cleared on resubmission
- [ ] Resubmission history tracked
- [ ] Integration test passing

---

### P03M27: Write unit tests for ChangePlan repository

**Status:** 🚧 TODO
**ID:** P03M27

**Goal**
Create comprehensive unit tests for all Change Plan repository operations.

**Acceptance Criteria**
- [ ] Unit tests for Create, GetByID, Update, Delete, List methods
- [ ] Test coverage >85%
- [ ] Edge cases covered (not found, duplicate, etc.)
- [ ] Filter tests passing
- [ ] Error scenarios tested
- [ ] All tests passing

---

### P03M28: Write unit tests for SQL syntax validators

**Status:** 🚧 TODO
**ID:** P03M28

**Goal**
Create comprehensive unit tests for PostgreSQL and MySQL syntax validators.

**Acceptance Criteria**
- [ ] Unit tests for PostgreSQL syntax validator (>85% coverage)
- [ ] Unit tests for MySQL syntax validator (>85% coverage)
- [ ] Valid SQL test cases passing
- [ ] Invalid SQL test cases passing
- [ ] Error message validation tests
- [ ] All tests passing

---

### P03M29: Write unit tests for security validator

**Status:** 🚧 TODO
**ID:** P03M29

**Goal**
Create comprehensive unit tests covering all security violation scenarios.

**Acceptance Criteria**
- [ ] Unit tests for security validator (>85% coverage)
- [ ] DROP operations test cases
- [ ] TRUNCATE operations test cases
- [ ] Security rule violation tests
- [ ] Edge cases covered
- [ ] All tests passing

---

### P03M30: Write unit tests for performance validator

**Status:** 🚧 TODO
**ID:** P03M30

**Goal**
Create comprehensive unit tests for performance violation detection.

**Acceptance Criteria**
- [ ] Unit tests for performance validator (>85% coverage)
- [ ] Missing index detection tests
- [ ] Missing primary key tests
- [ ] Performance rule violation tests
- [ ] Edge cases covered
- [ ] All tests passing

---

### P03M31: Write unit tests for backwards compatibility validator

**Status:** 🚧 TODO
**ID:** P03M31

**Goal**
Create comprehensive unit tests for breaking change detection.

**Acceptance Criteria**
- [ ] Unit tests for backwards compatibility validator (>85% coverage)
- [ ] DROP COLUMN detection tests
- [ ] RENAME operations tests
- [ ] Data type change tests
- [ ] Breaking change detection tests
- [ ] All tests passing

---

### P03M32: Write unit tests for best practices validator

**Status:** 🚧 TODO
**ID:** P03M32

**Goal**
Create comprehensive unit tests for naming convention and best practice checks.

**Acceptance Criteria**
- [ ] Unit tests for best practices validator (>85% coverage)
- [ ] Naming convention test cases
- [ ] Data type best practice tests
- [ ] Rule violation tests
- [ ] Edge cases covered
- [ ] All tests passing

---

### P03M33: Write unit tests for SQL Review orchestration

**Status:** 🚧 TODO
**ID:** P03M33

**Goal**
Create comprehensive unit tests for validator orchestration and result aggregation.

**Acceptance Criteria**
- [ ] Unit tests for review service (>85% coverage)
- [ ] Multiple validator execution tests
- [ ] Result aggregation tests
- [ ] Error handling tests
- [ ] Edge cases covered
- [ ] All tests passing

---

### P03M34: Write unit tests for risk level assignment

**Status:** 🚧 TODO
**ID:** P03M34

**Goal**
Create comprehensive unit tests for automated risk level calculation.

**Acceptance Criteria**
- [ ] Unit tests for risk assessment service (>85% coverage)
- [ ] Risk level calculation test cases
- [ ] Environment-based risk tests
- [ ] SQL content-based risk tests
- [ ] Edge cases covered
- [ ] All tests passing

---

### P03M35: Write unit tests for status workflow

**Status:** 🚧 TODO
**ID:** P03M35

**Goal**
Create comprehensive unit tests for status transition validation.

**Acceptance Criteria**
- [ ] Unit tests for status transitions (>85% coverage)
- [ ] Valid transition test cases
- [ ] Invalid transition test cases
- [ ] Status validation tests
- [ ] Edge cases covered
- [ ] All tests passing

---

### P03M36: Write integration tests for Change Plan CRUD

**Status:** 🚧 TODO
**ID:** P03M36

**Goal**
Create integration tests for all Change Plan endpoints using testcontainers.

**Acceptance Criteria**
- [ ] Integration test for POST /api/v1/change-plans passing
- [ ] Integration test for GET /api/v1/change-plans/:id passing
- [ ] Integration test for GET /api/v1/change-plans (list) passing
- [ ] Integration test for PUT /api/v1/change-plans/:id passing
- [ ] Integration test for DELETE /api/v1/change-plans/:id passing
- [ ] RBAC tests included
- [ ] All integration tests passing

---

### P03M37: Write integration tests for SQL Review flow

**Status:** 🚧 TODO
**ID:** P03M37

**Goal**
Create integration tests for automatic and manual SQL Review workflows.

**Acceptance Criteria**
- [ ] Integration test for automatic SQL Review on submission passing
- [ ] Integration test for manual SQL Review trigger passing
- [ ] Integration test for error blocking approval passing
- [ ] Integration test for warning handling passing
- [ ] Policy finding storage tests passing
- [ ] All integration tests passing

---

### P03M38: Write integration tests for Change Plan filtering

**Status:** 🚧 TODO
**ID:** P03M38

**Goal**
Create integration tests for Change Plan filtering functionality.

**Acceptance Criteria**
- [ ] Integration test for status filtering passing
- [ ] Integration test for project filtering passing
- [ ] Integration test for database filtering passing
- [ ] Integration test for author filtering passing
- [ ] Integration test for combined filters passing
- [ ] All integration tests passing

---

### P03M39: Update API documentation for Change Plans

**Status:** 🚧 TODO
**ID:** P03M39

**Goal**
Update OpenAPI specification with all Change Plan endpoints and SQL Review endpoints.

**Acceptance Criteria**
- [ ] OpenAPI spec updated with Change Plan CRUD endpoints
- [ ] OpenAPI spec updated with SQL Review endpoint
- [ ] Request/response schemas documented
- [ ] Policy finding schema documented
- [ ] Swagger UI displays all endpoints correctly

---

### P03M40: Verify Phase 3 Completion

**Status:** 🚧 TODO
**ID:** P03M40

**Goal**
Verify that all Phase 3 requirements are met, tests pass, and the phase is ready for sign-off before proceeding to Phase 4.

**Acceptance Criteria**
- [ ] All previous milestones in Phase 3 marked as complete
- [ ] All tests pass (unit, integration)
- [ ] Code coverage meets project standards (>85% for business logic)
- [ ] No outstanding bugs or issues from this phase
- [ ] Code reviewed and meets quality standards
- [ ] All linting checks passing (golangci-lint)
- [ ] SQL Review engine working end-to-end
- [ ] All validators functioning correctly
- [ ] Ready to proceed to Phase 4

---
