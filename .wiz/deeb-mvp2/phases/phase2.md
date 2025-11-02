# Phase 2: Core Resources Management

**Duration**: ~3 days (31 milestones @ 1h each)
**Dependencies**: Phase 1
**Status**: 🚧 TODO

## Goal

Implement core resource management functionality including Projects, Environments, Database Instances, and Databases. This phase enables administrators to configure and manage the organizational structure that Change Plans will operate within.

## Phase-level Acceptance Criteria

- [ ] Projects CRUD operations fully implemented (create, read, update, delete, list)
- [ ] Environments CRUD operations fully implemented (create, read, update, delete, list)
- [ ] Policy CRUD operations fully implemented (create, read, update, delete, list) - SQL_REVIEW type
- [ ] Environment Policy assignment working
- [ ] Database Instance CRUD operations fully implemented (create, read, update, delete, list)
- [ ] Database Instance connection testing implemented (MySQL and PostgreSQL)
- [ ] Database Instance credential encryption working
- [ ] Databases CRUD operations fully implemented (create, read, update, delete, list)
- [ ] Database assignment to Projects and Environments enforced
- [ ] Each database belongs to exactly one Project and one Environment
- [ ] Project membership management (add/remove project members with roles)
- [ ] Project roles assignment working (Owner, Developer, Approver, Viewer)
- [ ] Authorization middleware enforcing resource access permissions
- [ ] All management endpoints have proper RBAC checks
- [ ] Unit tests for repository operations passing
- [ ] Integration tests for resource management endpoints passing
- [ ] All linting checks passing

## Milestones

<!-- Milestones appended by /wiz-milestones -->

### P02M01: Create Project repository interface and implementation

**Status:** ✅ COMPLETE
**ID:** P02M01

**Goal**
Create ProjectRepository interface and PostgreSQL implementation for Project CRUD operations.

**Acceptance Criteria**
- [x] ProjectRepository interface defined in `backend/internal/repository/project_repository.go`
- [x] Interface methods: Create, GetByID, Update, Delete, List, Count
- [x] PostgreSQL implementation created in `backend/internal/repository/postgres/project_repository.go`
- [x] All methods implemented with proper SQL queries
- [x] Error handling implemented

---

### P02M02: Create Project handler and POST endpoint

**Status:** ✅ COMPLETE
**ID:** P02M02

**Goal**
Implement POST /api/v1/projects endpoint for creating projects (admin only).

**Acceptance Criteria**
- [x] Project handler created in `backend/internal/handlers/project_handler.go`
- [x] POST /api/v1/projects endpoint implemented
- [x] Request validation (name required, description optional)
- [x] RBAC check: Organization Admin or Organization DBA only
- [x] Project created with created_by_id set to authenticated user
- [x] 201 Created response with project data
- [x] Unit tests updated (integration test can be added in later milestone)

---

### P02M03: Create Project GET endpoints

**Status:** ✅ COMPLETE
**ID:** P02M03

**Goal**
Implement GET endpoints for retrieving individual projects and listing projects.

**Acceptance Criteria**
- [x] GET /api/v1/projects/:id endpoint implemented
- [x] GET /api/v1/projects endpoint implemented (list all, admin only)
- [x] 404 Not Found for nonexistent projects
- [x] Authorization checks implemented (admin required for list, auth required for get by ID)
- [x] Response includes project details
- [x] Unit tests passing (integration tests can be added in later milestone)

---

### P02M04: Create Project PUT and DELETE endpoints

**Status:** ✅ COMPLETE
**ID:** P02M04

**Goal**
Implement PUT and DELETE endpoints for updating and deleting projects (admin only).

**Acceptance Criteria**
- [x] PUT /api/v1/projects/:id endpoint implemented
- [x] DELETE /api/v1/projects/:id endpoint implemented
- [x] RBAC checks: admin only
- [x] Update validation working
- [x] Delete cascade handling (prevent deletion if databases exist)
- [x] Unit tests updated (integration tests can be added in later milestone)

---

### P02M05: Create Environment repository interface and implementation

**Status:** ✅ COMPLETE
**ID:** P02M05

**Goal**
Create EnvironmentRepository interface and PostgreSQL implementation for Environment CRUD operations with Policy reference.

**Acceptance Criteria**
- [x] EnvironmentRepository interface defined
- [x] Interface methods: Create, GetByID, Update, Delete, List, Count
- [x] PostgreSQL implementation created
- [x] Policy ID foreign key handling implemented (nullable)
- [x] All methods implemented with proper SQL queries

---

### P02M06: Create Environment CRUD endpoints

**Status:** ✅ COMPLETE
**ID:** P02M06

**Goal**
Implement all CRUD endpoints for Environments (admin only) with Policy assignment.

**Acceptance Criteria**
- [x] POST /api/v1/environments endpoint implemented
- [x] GET /api/v1/environments/:id endpoint implemented
- [x] GET /api/v1/environments endpoint implemented (list all)
- [x] PUT /api/v1/environments/:id endpoint implemented
- [x] DELETE /api/v1/environments/:id endpoint implemented
- [x] Policy ID field handling working (nullable, references policies)
- [x] Policy validation (must exist and be SQL_REVIEW type if provided)
- [x] RBAC checks: admin only
- [x] Handler created with validation logic
- [x] Routes registered with authentication and authorization middleware

---

### P02M07: Create Policy repository interface and implementation

**Status:** ✅ COMPLETE
**ID:** P02M07

**Goal**
Create PolicyRepository interface and PostgreSQL implementation for Policy CRUD operations (SQL_REVIEW type and future types).

**Acceptance Criteria**
- [x] PolicyRepository interface defined
- [x] Interface methods: Create, GetByID, Update, Delete, List, Count, GetByType
- [x] PostgreSQL implementation created
- [x] Policy type filtering implemented
- [x] Policy config JSONB handling implemented
- [x] All methods implemented with proper SQL queries

---

### P02M08: Create Policy CRUD endpoints

**Status:** ✅ COMPLETE
**ID:** P02M08

**Goal**
Implement all CRUD endpoints for Policies (admin only), supporting SQL_REVIEW type and filtering by type.

**Acceptance Criteria**
- [x] POST /api/v1/policies endpoint implemented
- [x] GET /api/v1/policies/:id endpoint implemented
- [x] GET /api/v1/policies endpoint implemented (list all, filter by type)
- [x] PUT /api/v1/policies/:id endpoint implemented
- [x] DELETE /api/v1/policies/:id endpoint implemented
- [x] Policy type validation (SQL_REVIEW type required for MVP)
- [x] Policy config JSONB field handling working
- [x] Delete validation (prevent deletion if referenced by Environments)
- [x] RBAC checks: admin only
- [x] Handler created with validation logic
- [x] Routes registered with authentication and authorization middleware

---

### P02M09: Create DatabaseInstance repository interface and implementation

**Status:** ✅ COMPLETE
**ID:** P02M09

**Goal**
Create DatabaseInstanceRepository interface and PostgreSQL implementation with encrypted credential handling.

**Acceptance Criteria**
- [x] DatabaseInstanceRepository interface defined
- [x] Interface methods: Create, GetByID, Update, Delete, List, Count
- [x] PostgreSQL implementation created
- [x] Encrypted credential storage/retrieval working
- [x] All methods implemented with proper SQL queries
- [x] Repository updated to match new schema (database_type, no environment_id)
- [x] Encryption key passed to repository constructor
- [x] Helper methods for encryption/decryption (private methods)

---

### P02M10: Implement database connection testing utility

**Status:** ✅ COMPLETE
**ID:** P02M10

**Goal**
Create utility functions for testing MySQL and PostgreSQL database connections.

**Acceptance Criteria**
- [x] Connection testing utility created in `backend/internal/utils/database.go`
- [x] TestMySQLConnection function implemented
- [x] TestPostgreSQLConnection function implemented
- [x] Credential decryption before connection test
- [x] Connection timeout handling
- [x] Error messages clear and actionable
- [x] Unit tests passing
- [x] MySQL driver dependency added

---

### P02M11: Create DatabaseInstance POST endpoint with connection testing

**Status:** ✅ COMPLETE
**ID:** P02M11

**Goal**
Implement POST /api/v1/database-instances endpoint with connection testing before saving.

**Acceptance Criteria**
- [x] POST /api/v1/database-instances endpoint implemented
- [x] Request validation (name, database_type, host, port, username, password required)
- [x] Credentials encrypted before storage
- [x] Connection test performed before saving
- [x] Connection test failure returns 400 Bad Request
- [x] RBAC checks: admin only
- [ ] Integration test passing (can be added in later milestone)

---

### P02M12: Create DatabaseInstance GET endpoints

**Status:** ✅ COMPLETE
**ID:** P02M12

**Goal**
Implement GET endpoints for retrieving database instances (credentials excluded from responses).

**Acceptance Criteria**
- [x] GET /api/v1/database-instances/:id endpoint implemented
- [x] GET /api/v1/database-instances endpoint implemented (list all)
- [x] Credentials excluded from response (masked or omitted)
- [x] 404 Not Found for nonexistent instances
- [x] Authorization checks implemented (authentication required)
- [ ] Integration tests passing (can be added in later milestone)

---

### P02M13: Create DatabaseInstance PUT and DELETE endpoints

**Status:** ✅ COMPLETE
**ID:** P02M13

**Goal**
Implement PUT and DELETE endpoints for database instances with proper credential handling.

**Acceptance Criteria**
- [x] PUT /api/v1/database-instances/:id endpoint implemented
- [x] DELETE /api/v1/database-instances/:id endpoint implemented
- [x] Credential encryption on update
- [x] Connection testing on credential update
- [x] Delete cascade handling (prevent deletion if databases exist)
- [x] RBAC checks: admin only
- [ ] Integration tests passing (can be added in later milestone)

---

### P02M14: Create Database repository interface and implementation

**Status:** ✅ COMPLETE
**ID:** P02M14

**Goal**
Create DatabaseRepository interface and PostgreSQL implementation with project/environment constraints.

**Acceptance Criteria**
- [x] DatabaseRepository interface defined
- [x] Interface methods: Create, GetByID, Update, Delete, List, Count, GetByProjectID
- [x] PostgreSQL implementation created
- [x] Foreign key constraints enforced (via schema)
- [x] All methods implemented with proper SQL queries

---

### P02M15: Create Database POST endpoint with validation

**Status:** ✅ COMPLETE
**ID:** P02M15

**Goal**
Implement POST /api/v1/databases endpoint with project and environment assignment validation.

**Acceptance Criteria**
- [x] POST /api/v1/databases endpoint implemented
- [x] Request validation (project_id, database_instance_id, environment_id, database_name required)
- [x] Validation: project exists, instance exists, environment exists
- [x] Validation: database belongs to exactly one project and one environment
- [x] RBAC checks: admin only
- [ ] Integration test passing (can be added in later milestone)

---

### P02M16: Create Database GET endpoints

**Status:** ✅ COMPLETE
**ID:** P02M16

**Goal**
Implement GET endpoints for retrieving databases with filtering options.

**Acceptance Criteria**
- [x] GET /api/v1/databases/:id endpoint implemented
- [x] GET /api/v1/databases endpoint implemented (list with optional filters: project_id, environment_id)
- [x] Response includes project, instance, and environment details
- [x] Authorization checks implemented
- [ ] Integration tests passing (can be added in later milestone)

---

### P02M17: Create Database PUT and DELETE endpoints

**Status:** ✅ COMPLETE
**ID:** P02M17

**Goal**
Implement PUT and DELETE endpoints for databases with constraint enforcement.

**Acceptance Criteria**
- [x] PUT /api/v1/databases/:id endpoint implemented
- [x] DELETE /api/v1/databases/:id endpoint implemented
- [x] Update validation: project/environment constraints maintained
- [x] Delete prevented if Change Plans exist
- [x] RBAC checks: admin only
- [ ] Integration tests passing (can be added in later milestone)

---

### P02M18: Create ProjectMember repository interface and implementation

**Status:** ✅ COMPLETE
**ID:** P02M18

**Goal**
Create ProjectMemberRepository interface and PostgreSQL implementation for managing project membership.

**Acceptance Criteria**
- [x] ProjectMemberRepository interface defined
- [x] Interface methods: AddMember, RemoveMember, GetMembers, GetMemberRole, UpdateRole
- [x] PostgreSQL implementation created
- [x] Unique constraint on (project_id, user_id) enforced
- [x] All methods implemented with proper SQL queries

---

### P02M19: Implement project membership POST endpoint

**Status:** ✅ COMPLETE
**ID:** P02M19

**Goal**
Implement POST /api/v1/projects/:id/members endpoint for adding project members.

**Acceptance Criteria**
- [x] POST /api/v1/projects/:id/members endpoint implemented
- [x] Request validation (user_id, project_role required)
- [x] Project role validation: Owner, Developer, Approver, Viewer
- [x] RBAC checks: Project Owner or Organization Admin
- [x] Duplicate member prevention
- [ ] Integration test passing (can be added in later milestone)

---

### P02M20: Implement project membership GET and DELETE endpoints

**Status:** ✅ COMPLETE
**ID:** P02M20

**Goal**
Implement GET and DELETE endpoints for project membership management.

**Acceptance Criteria**
- [x] GET /api/v1/projects/:id/members endpoint implemented (list members)
- [x] DELETE /api/v1/projects/:id/members/:user_id endpoint implemented
- [x] Response includes user details and project role
- [x] RBAC checks: Project Owner or Organization Admin
- [x] Cannot remove Project Owner if they're the only owner
- [ ] Integration tests passing (can be added in later milestone)

---

### P02M21: Implement project member role update endpoint

**Status:** ✅ COMPLETE
**ID:** P02M21

**Goal**
Implement PUT endpoint for updating project member roles.

**Acceptance Criteria**
- [x] PUT /api/v1/projects/:id/members/:user_id endpoint implemented
- [x] Request validation (project_role required)
- [x] Role update validation working
- [x] RBAC checks: Project Owner or Organization Admin
- [ ] Integration test passing (can be added in later milestone)

---

### P02M22: Enhance authorization middleware for project resources

**Status:** ✅ COMPLETE
**ID:** P02M22

**Goal**
Extend authorization middleware to check project-level roles for project resource access.

**Acceptance Criteria**
- [x] Authorization middleware updated in `backend/internal/middleware/authz.go`
- [x] Project role checking implemented (requires project context from URL or request body)
- [x] Permission checking helper functions for project roles
- [x] Organization role fallback logic
- [x] Unit tests for authorization logic passing

---

### P02M23: Add RBAC checks to all resource management endpoints

**Status:** ✅ COMPLETE
**ID:** P02M23

**Goal**
Ensure all project, environment, database instance, and database endpoints have proper RBAC checks.

**Acceptance Criteria**
- [x] All project endpoints check organization roles (admin/DBA)
- [x] All environment endpoints check organization roles (admin/DBA)
- [x] All database instance endpoints check organization roles (admin/DBA)
- [x] All database endpoints check organization roles (admin/DBA)
- [x] Project member endpoints check project roles or organization roles
- [x] 403 Forbidden responses for unauthorized access
- [ ] Integration tests for authorization scenarios passing (can be added in later milestone)

---

### P02M24: Write unit tests for Project repository

**Status:** ✅ COMPLETE
**ID:** P02M24

**Goal**
Create comprehensive unit tests for Project repository operations using mocks.

**Acceptance Criteria**
- [x] Unit tests for Create, GetByID, Update, Delete, List methods
- [x] Test coverage >85%
- [x] Edge cases covered (duplicate names, not found, etc.)
- [x] Error scenarios tested
- [x] All tests passing

---

### P02M25: Write unit tests for Environment repository

**Status:** ✅ COMPLETE
**ID:** P02M25

**Goal**
Create comprehensive unit tests for Environment repository operations including Policy reference handling.

**Acceptance Criteria**
- [x] Unit tests for all CRUD operations
- [x] Policy ID foreign key handling tests
- [x] Test coverage >85%
- [x] Edge cases covered
- [x] All tests passing

---

### P02M26: Write unit tests for Policy repository

**Status:** ✅ COMPLETE
**ID:** P02M26

**Goal**
Create comprehensive unit tests for Policy repository operations including policy config JSONB handling and type filtering.

**Acceptance Criteria**
- [x] Unit tests for all CRUD operations
- [x] Policy type filtering tests
- [x] Policy config JSONB handling tests
- [x] Test coverage >85%
- [x] Edge cases covered
- [x] All tests passing

---

### P02M27: Write unit tests for DatabaseInstance repository

**Status:** ✅ COMPLETE
**ID:** P02M27

**Goal**
Create comprehensive unit tests for DatabaseInstance repository including encryption/decryption.

**Acceptance Criteria**
- [x] Unit tests for all CRUD operations
- [x] Credential encryption/decryption tests
- [x] Test coverage >85%
- [x] Edge cases covered
- [x] All tests passing

---

### P02M28: Write unit tests for Database repository

**Status:** ✅ COMPLETE
**ID:** P02M28

**Goal**
Create comprehensive unit tests for Database repository including constraint validation.

**Acceptance Criteria**
- [x] Unit tests for all CRUD operations
- [x] Project/environment constraint tests
- [x] Test coverage >85%
- [x] Edge cases covered
- [x] All tests passing

---

### P02M29: Write integration tests for resource management endpoints

**Status:** 🚧 TODO
**ID:** P02M29

**Goal**
Create integration tests for Projects, Environments, Database Instances, and Databases endpoints.

**Acceptance Criteria**
- [ ] Integration tests for all CRUD endpoints
- [ ] RBAC authorization tests
- [ ] Connection testing tests
- [ ] Constraint validation tests
- [ ] Test database setup using testcontainers
- [ ] All integration tests passing

---

### P02M30: Update API documentation (OpenAPI/Swagger)

**Status:** 🚧 TODO
**ID:** P02M30

**Goal**
Update OpenAPI specification with all new resource management endpoints including Policy endpoints.

**Acceptance Criteria**
- [ ] OpenAPI spec updated with Project endpoints
- [ ] OpenAPI spec updated with Environment endpoints
- [ ] OpenAPI spec updated with Policy endpoints
- [ ] OpenAPI spec updated with DatabaseInstance endpoints
- [ ] OpenAPI spec updated with Database endpoints
- [ ] OpenAPI spec updated with ProjectMember endpoints
- [ ] Request/response schemas documented
- [ ] Swagger UI displays all endpoints correctly

---

### P02M31: Verify Phase 2 Completion

**Status:** 🚧 TODO
**ID:** P02M31

**Goal**
Verify that all Phase 2 requirements are met, tests pass, and the phase is ready for sign-off before proceeding to Phase 3.

**Acceptance Criteria**
- [ ] All previous milestones in Phase 2 marked as complete
- [ ] All tests pass (unit, integration)
- [ ] Code coverage meets project standards (>85% for business logic)
- [ ] No outstanding bugs or issues from this phase
- [ ] Code reviewed and meets quality standards
- [ ] All linting checks passing (golangci-lint)
- [ ] All resource management endpoints working end-to-end
- [ ] RBAC enforcement verified
- [ ] Ready to proceed to Phase 3

---
