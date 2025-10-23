# Phase 2: Core Backend Services

**Duration**: ~5 days (38 milestones @ 1h each)
**Dependencies**: Phase 1 (Foundation & Project Setup)
**Status**: 🚧 TODO

## Goal

Implement the core backend services including issue management, SQL validation engine, and approval workflow. This phase delivers the essential business logic that enables database change tracking, automated policy validation, and approval processes. The validation engine will support syntax checking, security analysis, performance checks, backwards compatibility validation, and best practices enforcement for both MySQL and PostgreSQL.

## Phase-level Acceptance Criteria

- Issue CRUD operations fully implemented (create, read, update, delete, list)
- Issue status tracking operational (Draft, Pending Review, Approved, In Progress, Applied, Rejected, Failed)
- SQL validation engine implemented with pluggable validator architecture
- MySQL and PostgreSQL syntax validation working using parsers
- Security validator detects SQL injection patterns and dangerous operations
- Performance analyzer identifies missing indexes and full table scans
- Backwards compatibility checker flags breaking schema changes
- Best practices validator enforces naming conventions and data types
- Validation results include severity levels (Error, Warning, Info)
- Approval workflow engine supports single-approval and multi-stage workflows
- Issue authors cannot approve their own changes
- Approval comments and history tracking functional
- Search and filter capabilities for issues implemented
- All API endpoints have comprehensive error handling
- Integration tests for core services passing
- API documentation (OpenAPI/Swagger) generated

## Milestones

<!-- Milestones appended by /wiz:milestones -->
### P02M01: Create issue model and repository

**Status:** 🚧 TODO
**ID:** P02M01

**Goal**
Implement issue data model and repository interface for database operations.

**Acceptance Criteria**
- [ ] Issue struct defined with all fields (id, title, description, sql_content, status, author_id, target_database_id, created_at, updated_at)
- [ ] IssueRepository interface defined (Create, GetByID, Update, Delete, List, Search)
- [ ] PostgreSQL implementation of IssueRepository
- [ ] Status enum defined (Draft, PendingReview, Approved, InProgress, Applied, Rejected, Failed)
- [ ] Unit tests for repository operations passing

---

### P02M02: Implement issue creation endpoint

**Status:** 🚧 TODO
**ID:** P02M02

**Goal**
Create POST /api/v1/issues endpoint for creating database change issues.

**Acceptance Criteria**
- [ ] Endpoint defined: POST /api/v1/issues
- [ ] Request validation (title, description, sql_content, target_database_id required)
- [ ] SQL content size limit enforced (10MB max)
- [ ] Issue created with Draft status
- [ ] Author set to authenticated user
- [ ] 201 Created response with issue data

---

### P02M03: Implement issue retrieval endpoints

**Status:** 🚧 TODO
**ID:** P02M03

**Goal**
Create GET endpoints for retrieving individual issues and listing issues.

**Acceptance Criteria**
- [ ] GET /api/v1/issues/:id endpoint implemented
- [ ] GET /api/v1/issues endpoint with pagination implemented
- [ ] 404 Not Found for nonexistent issues
- [ ] Authorization check (users can only see their own issues or public ones)
- [ ] Response includes issue details and related data
- [ ] Integration tests passing

---

### P02M04: Implement issue update endpoint

**Status:** 🚧 TODO
**ID:** P02M04

**Goal**
Create PUT /api/v1/issues/:id endpoint for updating issues in Draft status.

**Acceptance Criteria**
- [ ] Endpoint defined: PUT /api/v1/issues/:id
- [ ] Only Draft issues can be updated
- [ ] Only issue author can update
- [ ] Validation applied to updated fields
- [ ] 403 Forbidden if user is not author or issue not in Draft
- [ ] Integration tests passing

---

### P02M05: Implement issue search and filter

**Status:** 🚧 TODO
**ID:** P02M05

**Goal**
Add search and filter capabilities to issue listing endpoint.

**Acceptance Criteria**
- [ ] Filter by status query parameter
- [ ] Filter by author query parameter
- [ ] Filter by date range query parameters
- [ ] Filter by target database query parameter
- [ ] Full-text search on title and description
- [ ] Proper SQL indexes for filter fields

---

### P02M06: Create SQL validator interface

**Status:** 🚧 TODO
**ID:** P02M06

**Goal**
Design pluggable validator architecture for SQL validation.

**Acceptance Criteria**
- [ ] Validator interface defined with Validate method
- [ ] ValidationResult struct with severity levels (Error, Warning, Info)
- [ ] ValidationEngine that orchestrates multiple validators
- [ ] Validators can be registered dynamically
- [ ] Unit tests for validation engine passing

---

### P02M07: Implement MySQL syntax validator

**Status:** 🚧 TODO
**ID:** P02M07

**Goal**
Create syntax validator for MySQL using TiDB parser.

**Acceptance Criteria**
- [ ] TiDB parser library integrated
- [ ] MySQLSyntaxValidator implements Validator interface
- [ ] Parses SQL and detects syntax errors
- [ ] Returns Error severity for syntax violations
- [ ] Supports DDL and DML statements
- [ ] Unit tests with valid and invalid SQL passing

---

### P02M08: Implement PostgreSQL syntax validator

**Status:** 🚧 TODO
**ID:** P02M08

**Goal**
Create syntax validator for PostgreSQL using pg_query_go parser.

**Acceptance Criteria**
- [ ] pg_query_go library integrated
- [ ] PostgreSQLSyntaxValidator implements Validator interface
- [ ] Parses SQL and detects syntax errors
- [ ] Returns Error severity for syntax violations
- [ ] Supports DDL and DML statements
- [ ] Unit tests with valid and invalid SQL passing

---

### P02M09: Implement security validator for SQL injection

**Status:** 🚧 TODO
**ID:** P02M09

**Goal**
Create security validator that detects SQL injection patterns.

**Acceptance Criteria**
- [ ] SQLInjectionValidator implements Validator interface
- [ ] Detects common injection patterns (UNION, ; --, `)
- [ ] Checks for dynamic SQL construction
- [ ] Returns Error severity for injection risks
- [ ] Unit tests with malicious SQL passing

---

### P02M10: Implement security validator for dangerous operations

**Status:** 🚧 TODO
**ID:** P02M10

**Goal**
Create security validator that flags dangerous SQL operations.

**Acceptance Criteria**
- [ ] DangerousOperationsValidator implements Validator interface
- [ ] Flags DROP DATABASE, TRUNCATE without WHERE clause
- [ ] Flags DELETE/UPDATE without WHERE clause
- [ ] Flags privilege escalation attempts (GRANT, REVOKE)
- [ ] Returns Warning or Error severity based on operation
- [ ] Unit tests passing

---

### P02M11: Implement performance validator for missing indexes

**Status:** 🚧 TODO
**ID:** P02M11

**Goal**
Create performance validator that identifies queries without proper indexes.

**Acceptance Criteria**
- [ ] MissingIndexValidator implements Validator interface
- [ ] Analyzes SELECT queries for index usage
- [ ] Flags full table scans
- [ ] Returns Warning severity for performance concerns
- [ ] Unit tests with indexed and non-indexed queries passing

---

### P02M12: Implement performance validator for large operations

**Status:** 🚧 TODO
**ID:** P02M12

**Goal**
Create performance validator that flags potentially slow operations.

**Acceptance Criteria**
- [ ] LargeOperationsValidator implements Validator interface
- [ ] Detects large INSERT/UPDATE/DELETE operations
- [ ] Flags queries with excessive joins
- [ ] Flags SELECT * on large tables
- [ ] Returns Warning severity
- [ ] Unit tests passing

---

### P02M13: Implement backwards compatibility validator

**Status:** 🚧 TODO
**ID:** P02M13

**Goal**
Create validator that detects breaking schema changes.

**Acceptance Criteria**
- [ ] BreakingChangesValidator implements Validator interface
- [ ] Flags DROP TABLE/COLUMN operations
- [ ] Flags ALTER COLUMN that changes type incompatibly
- [ ] Flags removal of NOT NULL constraints to add them
- [ ] Returns Error or Warning severity
- [ ] Unit tests passing

---

### P02M14: Implement best practices validator for naming

**Status:** 🚧 TODO
**ID:** P02M14

**Goal**
Create validator that enforces naming conventions.

**Acceptance Criteria**
- [ ] NamingConventionsValidator implements Validator interface
- [ ] Checks table names (snake_case, plural)
- [ ] Checks column names (snake_case)
- [ ] Checks index names follow convention
- [ ] Returns Info severity for violations
- [ ] Unit tests passing

---

### P02M15: Implement best practices validator for data types

**Status:** 🚧 TODO
**ID:** P02M15

**Goal**
Create validator that recommends appropriate data types.

**Acceptance Criteria**
- [ ] DataTypeValidator implements Validator interface
- [ ] Flags TEXT when VARCHAR would suffice
- [ ] Recommends UUID over VARCHAR for IDs
- [ ] Recommends TIMESTAMP over DATETIME
- [ ] Returns Info severity
- [ ] Unit tests passing

---

### P02M16: Create validation service

**Status:** 🚧 TODO
**ID:** P02M16

**Goal**
Implement service layer that orchestrates all validators.

**Acceptance Criteria**
- [ ] ValidationService struct created
- [ ] ValidateSQL method accepts SQL and database type
- [ ] Runs all applicable validators
- [ ] Aggregates validation results
- [ ] Returns sorted results (Error, Warning, Info)
- [ ] Unit tests for service passing

---

### P02M17: Add validation endpoint

**Status:** 🚧 TODO
**ID:** P02M17

**Goal**
Create POST /api/v1/validate endpoint for on-demand SQL validation.

**Acceptance Criteria**
- [ ] Endpoint defined: POST /api/v1/validate
- [ ] Accepts SQL content and database type
- [ ] Returns validation results with severity levels
- [ ] No database persistence (validation only)
- [ ] Timeout set to 2 seconds
- [ ] Integration tests passing

---

### P02M18: Integrate validation into issue creation

**Status:** 🚧 TODO
**ID:** P02M18

**Goal**
Add automatic validation when issues are created or updated.

**Acceptance Criteria**
- [ ] Validation runs automatically on issue creation
- [ ] Validation results stored in database
- [ ] Validation results included in issue response
- [ ] Errors prevent issue from moving to PendingReview
- [ ] Warnings and Info shown but don't block
- [ ] Integration tests passing

---

### P02M19: Create approval model and repository

**Status:** 🚧 TODO
**ID:** P02M19

**Goal**
Implement approval data model and repository for tracking approvals.

**Acceptance Criteria**
- [ ] Approval struct defined (id, issue_id, approver_id, status, comments, approved_at)
- [ ] ApprovalRepository interface defined
- [ ] PostgreSQL implementation
- [ ] Approval status enum (Pending, Approved, Rejected)
- [ ] Unit tests passing

---

### P02M20: Implement approval workflow engine

**Status:** 🚧 TODO
**ID:** P02M20

**Goal**
Create workflow engine that manages approval processes.

**Acceptance Criteria**
- [ ] WorkflowEngine struct created
- [ ] Single-approval workflow implemented
- [ ] Multi-stage workflow (Staging, Production) implemented
- [ ] Self-approval prevention enforced
- [ ] Workflow state machine implemented
- [ ] Unit tests for workflow logic passing

---

### P02M21: Create approval request endpoint

**Status:** 🚧 TODO
**ID:** P02M21

**Goal**
Implement POST /api/v1/issues/:id/request-approval endpoint.

**Acceptance Criteria**
- [ ] Endpoint moves issue from Draft to PendingReview
- [ ] Validation errors must be resolved first
- [ ] Approval request created in database
- [ ] Author cannot be approver
- [ ] 200 OK with updated issue status
- [ ] Integration tests passing

---

### P02M22: Create approval submission endpoint

**Status:** 🚧 TODO
**ID:** P02M22

**Goal**
Implement POST /api/v1/approvals/:id/submit endpoint for approving/rejecting.

**Acceptance Criteria**
- [ ] Endpoint defined: POST /api/v1/approvals/:id/submit
- [ ] Approver role required
- [ ] Self-approval prevented
- [ ] Comments required for rejection
- [ ] Issue status updated based on approval
- [ ] Approval timestamp recorded

---

### P02M23: Implement approval listing endpoint

**Status:** 🚧 TODO
**ID:** P02M23

**Goal**
Create GET /api/v1/approvals endpoint for pending approvals dashboard.

**Acceptance Criteria**
- [ ] Endpoint defined: GET /api/v1/approvals
- [ ] Lists pending approvals for authenticated user
- [ ] Pagination implemented
- [ ] Filter by issue, status, date range
- [ ] Includes issue details in response
- [ ] Integration tests passing

---

### P02M24: Add approval history to issue details

**Status:** 🚧 TODO
**ID:** P02M24

**Goal**
Include approval history in issue detail endpoint.

**Acceptance Criteria**
- [ ] GET /api/v1/issues/:id includes approvals array
- [ ] Approval history shows approver, status, comments, timestamp
- [ ] Sorted by timestamp descending
- [ ] Includes all approvals (approved, rejected, pending)
- [ ] Integration tests passing

---

### P02M25: Implement notification hooks for approvals

**Status:** 🚧 TODO
**ID:** P02M25

**Goal**
Create webhook notification system for approval events.

**Acceptance Criteria**
- [ ] NotificationService interface defined
- [ ] Webhook implementation (POST to configured URL)
- [ ] Slack implementation (POST to Slack webhook)
- [ ] Triggered on approval request, approval, rejection
- [ ] Configurable per user or organization
- [ ] Unit tests for notification sending passing

---

### P02M26: Add audit logging for all operations

**Status:** 🚧 TODO
**ID:** P02M26

**Goal**
Implement comprehensive audit logging for issue and approval operations.

**Acceptance Criteria**
- [ ] Audit log entry created for issue creation, update, deletion
- [ ] Audit log entry for approval request, approval, rejection
- [ ] Audit log includes user ID, action, resource type, resource ID, details, timestamp
- [ ] Audit logs immutable (write-once)
- [ ] Query endpoint for audit logs (Admin only)
- [ ] Integration tests passing

---

### P02M27: Implement audit log viewer endpoint

**Status:** 🚧 TODO
**ID:** P02M27

**Goal**
Create GET /api/v1/audit-logs endpoint for viewing audit trail.

**Acceptance Criteria**
- [ ] Endpoint defined: GET /api/v1/audit-logs (Admin only)
- [ ] Pagination implemented
- [ ] Filter by user, action, resource type, date range
- [ ] Export capability (JSON format)
- [ ] 90-day retention enforced
- [ ] Integration tests passing

---

### P02M28: Write unit tests for issue repository

**Status:** 🚧 TODO
**ID:** P02M28

**Goal**
Create comprehensive unit tests for issue repository operations.

**Acceptance Criteria**
- [ ] Test Create, GetByID, Update, Delete, List operations
- [ ] Test search and filter functionality
- [ ] Test pagination
- [ ] Test error handling for invalid inputs
- [ ] All tests passing
- [ ] Code coverage for repository >85%

---

### P02M29: Write unit tests for validation engine

**Status:** 🚧 TODO
**ID:** P02M29

**Goal**
Create comprehensive unit tests for SQL validation system.

**Acceptance Criteria**
- [ ] Test each validator individually
- [ ] Test validation engine orchestration
- [ ] Test severity level ordering
- [ ] Test validation result aggregation
- [ ] All tests passing
- [ ] Code coverage for validation >90%

---

### P02M30: Write unit tests for approval workflow

**Status:** 🚧 TODO
**ID:** P02M30

**Goal**
Create comprehensive unit tests for approval workflow engine.

**Acceptance Criteria**
- [ ] Test single-approval workflow
- [ ] Test multi-stage workflow
- [ ] Test self-approval prevention
- [ ] Test state transitions
- [ ] All tests passing
- [ ] Code coverage for workflow >85%

---

### P02M31: Write integration tests for issue endpoints

**Status:** 🚧 TODO
**ID:** P02M31

**Goal**
Create integration tests for all issue management endpoints.

**Acceptance Criteria**
- [ ] Test create, retrieve, update, search endpoints
- [ ] Test authorization checks
- [ ] Test validation integration
- [ ] Test error responses
- [ ] All tests passing with test database

---

### P02M32: Write integration tests for validation endpoint

**Status:** 🚧 TODO
**ID:** P02M32

**Goal**
Create integration tests for SQL validation endpoint.

**Acceptance Criteria**
- [ ] Test validation with valid SQL
- [ ] Test validation with syntax errors
- [ ] Test validation with security issues
- [ ] Test validation with performance warnings
- [ ] Test timeout handling
- [ ] All tests passing

---

### P02M33: Write integration tests for approval endpoints

**Status:** 🚧 TODO
**ID:** P02M33

**Goal**
Create integration tests for approval workflow endpoints.

**Acceptance Criteria**
- [ ] Test approval request creation
- [ ] Test approval submission (approve/reject)
- [ ] Test approval listing
- [ ] Test self-approval prevention
- [ ] All tests passing with test database

---

### P02M34: Update OpenAPI documentation

**Status:** 🚧 TODO
**ID:** P02M34

**Goal**
Add issue management and validation endpoints to OpenAPI spec.

**Acceptance Criteria**
- [ ] Issue endpoints documented
- [ ] Validation endpoint documented
- [ ] Approval endpoints documented
- [ ] Request/response schemas defined
- [ ] Error responses documented
- [ ] Swagger UI displays correctly

---

### P02M35: Add request rate limiting

**Status:** 🚧 TODO
**ID:** P02M35

**Goal**
Implement rate limiting for validation and issue creation endpoints.

**Acceptance Criteria**
- [ ] Rate limiter middleware created
- [ ] Validation endpoint limited to 10 req/min per user
- [ ] Issue creation limited to 20 req/min per user
- [ ] 429 Too Many Requests response
- [ ] Rate limit headers in response
- [ ] Integration tests passing

---

### P02M36: Implement correlation ID propagation

**Status:** 🚧 TODO
**ID:** P02M36

**Goal**
Ensure correlation IDs propagate through all service calls.

**Acceptance Criteria**
- [ ] Correlation ID passed from middleware to services
- [ ] Correlation ID included in audit logs
- [ ] Correlation ID included in error logs
- [ ] Correlation ID in database queries (as comment)
- [ ] End-to-end tracing working

---

### P02M37: Add slow query logging

**Status:** 🚧 TODO
**ID:** P02M37

**Goal**
Implement logging for slow API requests and database queries.

**Acceptance Criteria**
- [ ] API requests >2s logged with details
- [ ] Database queries >1s logged with details
- [ ] Log includes correlation ID, user, endpoint, duration
- [ ] Configurable thresholds via environment variables
- [ ] Integration tests verify logging

---

### P02M38: Verify Phase 2 Completion

**Status:** 🚧 TODO
**ID:** P02M38

**Goal**
Verify that all Phase 2 requirements are met, tests pass, and the phase is ready for sign-off before proceeding to Phase 3.

**Acceptance Criteria**
- [ ] All previous milestones in Phase 2 marked as complete
- [ ] All tests pass (unit and integration)
- [ ] Code coverage meets project standards (>85% for core services)
- [ ] No outstanding bugs or issues from this phase
- [ ] Code reviewed and meets quality standards
- [ ] Issue CRUD operations working end-to-end
- [ ] SQL validation working for MySQL and PostgreSQL
- [ ] Approval workflow functional
- [ ] Audit logging capturing all operations
- [ ] API documentation up-to-date
- [ ] Ready to proceed to Phase 3

---
