# Phase 1: Foundation & Database Schema Rebuild

**Duration**: ~4 days (37 milestones @ 1h each)
**Dependencies**: None (first phase)
**Status**: 🚧 TODO

## Goal

Rebuild the database schema from scratch aligned with refined project definitions, establish foundational infrastructure for authentication and authorization, and set up core utilities and configuration management. This phase creates the fresh foundation that all subsequent phases will build upon, ensuring the schema accurately reflects the refined project definitions from docs/project.md.

## Phase-level Acceptance Criteria

- [ ] Fresh PostgreSQL schema designed and implemented with all core tables (users, projects, environments, policies, database_instances, databases, change_plans, approvals, audit_logs, schema_snapshots, policy_findings)
- [ ] Database migration system operational (golang-migrate)
- [ ] User authentication system working (OAuth 2.0 with Google + email/password)
- [ ] JWT token generation and validation implemented
- [ ] RBAC foundation in place (organization and project roles)
- [ ] First user registration creates Organization Admin automatically
- [ ] User management backend endpoints implemented
- [ ] Configuration management system operational (environment variables + config files)
- [ ] Logging infrastructure functional with structured logging (zap)
- [ ] Health check endpoint implemented
- [ ] Database connection pooling configured
- [ ] Credential encryption system implemented (AES-256 for database credentials)
- [ ] Unit tests for core utilities and models passing
- [ ] Integration tests for authentication endpoints passing
- [ ] All linting checks passing

## Milestones

<!-- Milestones appended by /wiz-milestones -->

### P01M01: Design fresh PostgreSQL schema for refined project definitions

**Status:** 🚧 TODO
**ID:** P01M01

**Goal**
Design the complete PostgreSQL database schema aligned with refined project definitions from docs/project.md, including all core tables: users, projects, environments, policies, database_instances, databases, change_plans, approvals, audit_logs, schema_snapshots, policy_findings.

**Acceptance Criteria**
- [ ] Schema ERD created documenting all tables and relationships
- [ ] All core tables defined with columns, constraints, and indexes
- [ ] Foreign key relationships documented
- [ ] Schema aligns with refined project definitions (Change Plans instead of Issues)
- [ ] Schema design document created at `backend/docs/schema.md` (overwrite the existing schema.md)

---

### P01M02: Create initial database migration file

**Status:** 🚧 TODO
**ID:** P01M02

**Goal**
Create the first migration file using golang-migrate for the fresh schema with users table and core authentication fields.

**Acceptance Criteria**
- [ ] Migration file created: `backend/migrations/000001_initial_schema.up.sql`
- [ ] Users table created with all required fields (id, email, password_hash, display_name, organization_role, is_active, oauth_provider, oauth_id, two_factor_enabled, two_factor_secret, last_login_at, created_at, updated_at)
- [ ] Down migration created: `backend/migrations/000001_initial_schema.down.sql`
- [ ] Migration file follows PostgreSQL best practices

---

### P01M03: Create projects table migration

**Status:** 🚧 TODO
**ID:** P01M03

**Goal**
Add projects table to migration with name, description fields (no status field per PRD refinement).

**Acceptance Criteria**
- [ ] Projects table created with columns: id (UUID), name, description, created_by_id, created_at, updated_at
- [ ] Foreign key constraint to users table
- [ ] Unique constraint on name
- [ ] Indexes created for common queries
- [ ] Migration file updated

---

### P01M04: Create policies table migration

**Status:** 🚧 TODO
**ID:** P01M04

**Goal**
Add policies table for storing Policy configurations (SQL_REVIEW type and future types) that can be referenced by Environments.

**Acceptance Criteria**
- [ ] Policies table created with columns: id (UUID), name, description, type (enum: SQL_REVIEW), policy_config (JSONB), created_by_id, created_at, updated_at
- [ ] Foreign key constraint to users table
- [ ] Unique constraint on name
- [ ] Type enum defined (SQL_REVIEW, extensible for future types)
- [ ] Policy config JSONB column for storing validator settings
- [ ] Migration file updated

---

### P01M05: Create environments table migration

**Status:** 🚧 TODO
**ID:** P01M05

**Goal**
Add environments table with name, description, and policy_id (references Policy).

**Acceptance Criteria**
- [ ] Environments table created with columns: id (UUID), name, description, policy_id (UUID nullable), created_by_id, created_at, updated_at
- [ ] Foreign key constraint to users table
- [ ] Foreign key constraint to policies table (nullable)
- [ ] Unique constraint on name
- [ ] Migration file updated

---

### P01M06: Create database_instances table migration

**Status:** 🚧 TODO
**ID:** P01M06

**Goal**
Add database_instances table with connection details and encrypted credential storage.

**Acceptance Criteria**
- [ ] Database_instances table created with columns: id (UUID), name, description, database_type (MySQL/PostgreSQL), host, port, username, connection_details_encrypted (TEXT), created_by_id, created_at, updated_at
- [ ] Foreign key constraint to users table
- [ ] Check constraint for database_type values
- [ ] Indexes for common queries
- [ ] Migration file updated

---

### P01M07: Create databases table migration

**Status:** 🚧 TODO
**ID:** P01M07

**Goal**
Add databases table linking databases to projects and environments with one-to-one constraints.

**Acceptance Criteria**
- [ ] Databases table created with columns: id (UUID), project_id, database_instance_id, environment_id, database_name, description, created_by_id, created_at, updated_at
- [ ] Foreign key constraints to projects, database_instances, environments, users
- [ ] Unique constraint ensuring each database belongs to exactly one project and one environment
- [ ] Indexes for common queries
- [ ] Migration file updated

---

### P01M08: Create change_plans table migration

**Status:** 🚧 TODO
**ID:** P01M08

**Goal**
Add change_plans table (renamed from issues) with all required fields for Change Plan lifecycle.

**Acceptance Criteria**
- [ ] Change_plans table created with columns: id (UUID), project_id, target_database_id, title, change_type (DDL/DML), sql_content (TEXT), description (TEXT nullable), rollback_sql (TEXT nullable), status, risk_level, author_id, assigned_reviewer_id (nullable), created_at, updated_at
- [ ] Foreign key constraints to projects, databases, users
- [ ] Status enum/check constraint: Draft, Open, Approved, Closed, InProgress, Completed, Failed, Cancelled
- [ ] Risk level enum/check constraint: Low, Medium, High
- [ ] Indexes for filtering and queries
- [ ] Migration file updated

---

### P01M09: Create approvals table migration

**Status:** 🚧 TODO
**ID:** P01M09

**Goal**
Add approvals table for tracking approval workflow with comments and history.

**Acceptance Criteria**
- [ ] Approvals table created with columns: id (UUID), change_plan_id, approver_id, action (approve/close), comment (TEXT nullable), created_at
- [ ] Foreign key constraints to change_plans, users
- [ ] Indexes for querying approval history
- [ ] Migration file updated

---

### P01M10: Create audit_logs table migration

**Status:** 🚧 TODO
**ID:** P01M10

**Goal**
Add audit_logs table for comprehensive audit trail of all user actions.

**Acceptance Criteria**
- [ ] Audit_logs table created with columns: id (UUID), user_id, action, target_type, target_id, result, details (JSONB), created_at
- [ ] Foreign key constraint to users
- [ ] Indexes for filtering by user, action, target_type, created_at
- [ ] JSONB column for flexible details storage
- [ ] Migration file updated

---

### P01M11: Create schema_snapshots table migration

**Status:** 🚧 TODO
**ID:** P01M11

**Goal**
Add schema_snapshots table for storing before/after schema snapshots (JSONB per PRD).

**Acceptance Criteria**
- [ ] Schema_snapshots table created with columns: id (UUID), database_id, change_plan_id, snapshot_type (before/after), snapshot_data (JSONB), snapshot_data_raw (TEXT), created_at
- [ ] Foreign key constraints to databases, change_plans
- [ ] Indexes for querying snapshots by database and change plan
- [ ] Migration file updated

---

### P01M12: Create policy_findings table migration

**Status:** 🚧 TODO
**ID:** P01M12

**Goal**
Add policy_findings table for storing SQL Review findings (results) with severity levels from policy evaluation.

**Acceptance Criteria**
- [ ] Policy_findings table created with columns: id (UUID), change_plan_id, validator_type, severity (Error/Warning/Info), message, rule_name, line_number (nullable), column_number (nullable), created_at
- [ ] Foreign key constraint to change_plans table
- [ ] Indexes for querying findings by change plan and severity
- [ ] Migration file updated

---

### P01M13: Create project_members table migration

**Status:** 🚧 TODO
**ID:** P01M13

**Goal**
Add project_members table for managing project membership and project-level roles.

**Acceptance Criteria**
- [ ] Project_members table created with columns: id (UUID), project_id, user_id, project_role (Owner/Developer/Approver/Viewer), added_by_id, added_at
- [ ] Foreign key constraints to projects, users
- [ ] Unique constraint on (project_id, user_id)
- [ ] Project role enum/check constraint
- [ ] Indexes for common queries
- [ ] Migration file updated

---

### P01M14: Add indexes and constraints to schema migration

**Status:** 🚧 TODO
**ID:** P01M14

**Goal**
Add performance indexes, unique constraints, and check constraints to complete the schema migration.

**Acceptance Criteria**
- [ ] All foreign key indexes created
- [ ] Unique constraints added where needed (email, project names, etc.)
- [ ] Check constraints for enum values
- [ ] Composite indexes for common query patterns
- [ ] Migration file complete and validated

---

### P01M15: Create User model struct

**Status:** 🚧 TODO
**ID:** P01M15

**Goal**
Create Go User model struct aligned with new schema, including all fields and JSON/database tags.

**Acceptance Criteria**
- [ ] User struct defined in `backend/internal/models/user.go`
- [ ] All fields match schema: id, email, password_hash, display_name, organization_role, is_active, oauth_provider, oauth_id, two_factor_enabled, two_factor_secret, last_login_at, created_at, updated_at
- [ ] JSON tags for API responses (password_hash and two_factor_secret excluded)
- [ ] Database tags match column names
- [ ] OrganizationRole constants defined

---

### P01M16: Create Project and Environment model structs

**Status:** 🚧 TODO
**ID:** P01M16

**Goal**
Create Project and Environment model structs aligned with refined schema. Environment references Policy instead of storing policy config directly.

**Acceptance Criteria**
- [ ] Project struct defined in `backend/internal/models/project.go`
- [ ] Environment struct defined in `backend/internal/models/environment.go`
- [ ] All fields match schema (name, description for Project; name, description, policy_id for Environment)
- [ ] JSON and database tags properly defined
- [ ] Policy ID field is nullable UUID

---

### P01M17: Create Policy model struct

**Status:** 🚧 TODO
**ID:** P01M17

**Goal**
Create Policy model struct for storing Policy configurations (SQL_REVIEW type and future types).

**Acceptance Criteria**
- [ ] Policy struct defined in `backend/internal/models/policy.go`
- [ ] All fields match schema: id, name, description, type (enum: SQL_REVIEW), policy_config (JSONB), created_by_id, created_at, updated_at
- [ ] PolicyType enum defined (SQL_REVIEW, extensible)
- [ ] PolicyConfig type defined for JSONB handling
- [ ] JSON and database tags properly defined

---

### P01M18: Create DatabaseInstance and Database model structs

**Status:** 🚧 TODO
**ID:** P01M18

**Goal**
Create DatabaseInstance and Database model structs with encrypted credential handling.

**Acceptance Criteria**
- [ ] DatabaseInstance struct defined in `backend/internal/models/database_instance.go`
- [ ] Database struct defined in `backend/internal/models/database.go`
- [ ] All fields match schema including encrypted credential field
- [ ] DatabaseType constants defined (MySQL, PostgreSQL)
- [ ] JSON tags exclude sensitive credential fields

---

### P01M19: Create ChangePlan model struct

**Status:** 🚧 TODO
**ID:** P01M19

**Goal**
Create ChangePlan model struct (renamed from Issue) with all status and risk level fields.

**Acceptance Criteria**
- [ ] ChangePlan struct defined in `backend/internal/models/change_plan.go`
- [ ] All fields match schema: id, project_id, target_database_id, title, change_type, sql_content, description, rollback_sql, status, risk_level, author_id, assigned_reviewer_id, created_at, updated_at
- [ ] Status and RiskLevel constants/types defined
- [ ] ChangeType constants defined (DDL, DML)
- [ ] JSON and database tags properly defined

---

### P01M20: Create supporting model structs (Approval, AuditLog, SchemaSnapshot, PolicyFinding)

**Status:** 🚧 TODO
**ID:** P01M20

**Goal**
Create model structs for Approval, AuditLog, SchemaSnapshot, and PolicyFinding tables.

**Acceptance Criteria**
- [ ] Approval struct defined in `backend/internal/models/approval.go`
- [ ] AuditLog struct defined in `backend/internal/models/audit_log.go`
- [ ] SchemaSnapshot struct defined in `backend/internal/models/schema_snapshot.go`
- [ ] PolicyFinding struct defined in `backend/internal/models/policy_finding.go`
- [ ] All fields match schema with proper tags
- [ ] Severity and Action constants defined

---

### P01M21: Implement credential encryption utility (AES-256)

**Status:** 🚧 TODO
**ID:** P01M21

**Goal**
Create utility functions for encrypting and decrypting database credentials using AES-256 encryption.

**Acceptance Criteria**
- [ ] Encryption utility created in `backend/internal/utils/encryption.go`
- [ ] Encrypt function encrypts credentials using AES-256
- [ ] Decrypt function decrypts credentials
- [ ] Key management handled securely (from environment/config)
- [ ] Unit tests for encryption/decryption passing
- [ ] Error handling for invalid keys or corrupted data

---

### P01M22: Update configuration management system

**Status:** 🚧 TODO
**ID:** P01M22

**Goal**
Update configuration system to support all required settings including encryption key, OAuth config, and database settings.

**Acceptance Criteria**
- [ ] Config struct updated in `backend/internal/config/config.go`
- [ ] Encryption key configuration added
- [ ] OAuth 2.0 Google configuration added
- [ ] Database connection settings configured
- [ ] Environment variable loading working
- [ ] Config file YAML loading working
- [ ] Config validation implemented

---

### P01M23: Update logging infrastructure with structured logging

**Status:** 🚧 TODO
**ID:** P01M23

**Goal**
Ensure structured logging with zap is properly configured and functional across the application.

**Acceptance Criteria**
- [ ] Logger initialized in `backend/internal/logger/logger.go`
- [ ] Structured logging configured with zap
- [ ] Log levels configurable (debug, info, warn, error)
- [ ] Log output format configurable (JSON for production, console for dev)
- [ ] Logger accessible throughout application
- [ ] Logging examples in key locations

---

### P01M24: Implement database connection pooling

**Status:** 🚧 TODO
**ID:** P01M24

**Goal**
Configure PostgreSQL connection pooling with appropriate settings for the application.

**Acceptance Criteria**
- [ ] Database connection pool configured in `backend/internal/database/postgres.go`
- [ ] Max open connections set appropriately
- [ ] Max idle connections configured
- [ ] Connection lifetime settings configured
- [ ] Connection testing on startup
- [ ] Pool stats logging implemented

---

### P01M25: Implement password hashing utility (bcrypt)

**Status:** 🚧 TODO
**ID:** P01M25

**Goal**
Create password hashing and verification utilities using bcrypt for email/password authentication.

**Acceptance Criteria**
- [ ] Password hashing function created in `backend/internal/auth/password.go`
- [ ] HashPassword function hashes passwords using bcrypt
- [ ] VerifyPassword function verifies passwords against hashes
- [ ] Cost factor configured appropriately (minimum 10)
- [ ] Unit tests for hashing/verification passing
- [ ] Error handling for invalid inputs

---

### P01M26: Implement JWT token generation

**Status:** 🚧 TODO
**ID:** P01M26

**Goal**
Create JWT token generation utility with user claims and configurable expiration.

**Acceptance Criteria**
- [ ] JWT token generation implemented in `backend/internal/auth/jwt.go`
- [ ] Token includes user ID, email, organization role
- [ ] Token expiration configurable (default 24 hours)
- [ ] Secure signing algorithm used (HS256 or RS256)
- [ ] Token signing key from configuration
- [ ] Unit tests for token generation passing

---

### P01M27: Implement JWT token validation

**Status:** 🚧 TODO
**ID:** P01M27

**Goal**
Create JWT token validation utility to parse and verify tokens with user claims extraction.

**Acceptance Criteria**
- [ ] JWT token validation implemented in `backend/internal/auth/jwt.go`
- [ ] ValidateToken function parses and verifies tokens
- [ ] User claims extracted from token
- [ ] Expiration validation working
- [ ] Invalid token errors handled gracefully
- [ ] Unit tests for token validation passing

---

### P01M28: Implement OAuth 2.0 Google authentication flow

**Status:** 🚧 TODO
**ID:** P01M28

**Goal**
Implement OAuth 2.0 authentication flow with Google provider including callback handling.

**Acceptance Criteria**
- [ ] OAuth 2.0 Google configuration loaded from config
- [ ] OAuth redirect endpoint implemented
- [ ] OAuth callback handler implemented
- [ ] Google user info fetched and validated
- [ ] User created or updated from OAuth profile
- [ ] JWT token issued after successful OAuth
- [ ] Error handling for OAuth failures

---

### P01M29: Implement email/password registration endpoint

**Status:** 🚧 TODO
**ID:** P01M29

**Goal**
Create user registration endpoint with email/password that automatically assigns Organization Admin role to first user.

**Acceptance Criteria**
- [ ] POST /api/v1/auth/register endpoint implemented
- [ ] Request validation (email format, password strength)
- [ ] First user check: if no users exist, assign Organization Admin role
- [ ] Subsequent users assigned Organization Developer role by default
- [ ] Password hashed before storage
- [ ] Email uniqueness enforced
- [ ] User created with proper fields
- [ ] Integration test passing

---

### P01M30: Implement email/password login endpoint

**Status:** 🚧 TODO
**ID:** P01M30

**Goal**
Create user login endpoint with email/password that validates credentials and issues JWT token.

**Acceptance Criteria**
- [ ] POST /api/v1/auth/login endpoint implemented
- [ ] Request validation (email, password required)
- [ ] User lookup by email
- [ ] Password verification against stored hash
- [ ] User active status checked
- [ ] JWT token issued on successful login
- [ ] Last login timestamp updated
- [ ] Error handling for invalid credentials
- [ ] Integration test passing

---

### P01M31: Create authentication middleware

**Status:** 🚧 TODO
**ID:** P01M31

**Goal**
Implement authentication middleware that validates JWT tokens and extracts user context for protected routes.

**Acceptance Criteria**
- [ ] Authentication middleware created in `backend/internal/middleware/auth.go`
- [ ] JWT token extracted from Authorization header
- [ ] Token validated and user claims extracted
- [ ] User context set in request for downstream handlers
- [ ] Unauthorized responses for invalid/missing tokens
- [ ] Middleware integrated with Echo router
- [ ] Unit tests for middleware passing

---

### P01M32: Create RBAC authorization middleware

**Status:** 🚧 TODO
**ID:** P01M32

**Goal**
Implement RBAC authorization middleware that checks user roles and permissions for resource access.

**Acceptance Criteria**
- [ ] Authorization middleware created in `backend/internal/middleware/authz.go`
- [ ] Organization role checking implemented
- [ ] Project role checking implemented (requires project context)
- [ ] Permission checking helper functions
- [ ] Forbidden responses for insufficient permissions
- [ ] Middleware integrated with routes
- [ ] Unit tests for authorization logic passing

---

### P01M33: Implement user management endpoints (GET, POST, PUT)

**Status:** 🚧 TODO
**ID:** P01M33

**Goal**
Create user management REST API endpoints for admins to create, read, and update users.

**Acceptance Criteria**
- [ ] GET /api/v1/users endpoint implemented (list users, admin only)
- [ ] GET /api/v1/users/:id endpoint implemented (get user, admin only)
- [ ] POST /api/v1/users endpoint implemented (create user, admin only)
- [ ] PUT /api/v1/users/:id endpoint implemented (update user, admin only)
- [ ] Request validation and RBAC checks
- [ ] Organization role assignment working
- [ ] Integration tests passing

---

### P01M34: Implement health check endpoint

**Status:** 🚧 TODO
**ID:** P01M34

**Goal**
Create health check endpoint that verifies database connectivity and returns system status.

**Acceptance Criteria**
- [ ] GET /health endpoint implemented
- [ ] Database connectivity check included
- [ ] Response includes version and status
- [ ] Returns 200 OK when healthy, 503 when unhealthy
- [ ] No authentication required
- [ ] Integration test passing

---

### P01M35: Write unit tests for core utilities and models

**Status:** 🚧 TODO
**ID:** P01M35

**Goal**
Create comprehensive unit tests for password hashing, JWT, encryption utilities, and model validation.

**Acceptance Criteria**
- [ ] Unit tests for password.go passing (>85% coverage)
- [ ] Unit tests for jwt.go passing (>85% coverage)
- [ ] Unit tests for encryption.go passing (>85% coverage)
- [ ] Model validation tests passing
- [ ] All tests use table-driven approach
- [ ] Edge cases and error scenarios covered

---

### P01M36: Write integration tests for authentication endpoints

**Status:** 🚧 TODO
**ID:** P01M36

**Goal**
Create integration tests for registration, login, and OAuth endpoints using testcontainers.

**Acceptance Criteria**
- [ ] Integration test for registration endpoint passing
- [ ] Integration test for login endpoint passing
- [ ] Integration test for OAuth callback passing
- [ ] Test database setup using testcontainers
- [ ] Test cleanup and teardown working
- [ ] All integration tests passing

---

### P01M37: Verify Phase 1 Completion

**Status:** 🚧 TODO
**ID:** P01M37

**Goal**
Verify that all Phase 1 requirements are met, tests pass, and the phase is ready for sign-off before proceeding to Phase 2.

**Acceptance Criteria**
- [ ] All previous milestones in Phase 1 marked as complete
- [ ] All tests pass (unit, integration)
- [ ] Code coverage meets project standards (>85% for business logic)
- [ ] No outstanding bugs or issues from this phase
- [ ] Code reviewed and meets quality standards
- [ ] All linting checks passing (golangci-lint)
- [ ] Database schema migration successful
- [ ] Authentication and authorization working end-to-end
- [ ] Ready to proceed to Phase 2

---
