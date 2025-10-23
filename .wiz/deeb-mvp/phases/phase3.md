# Phase 3: Database Integration & Change Execution

**Duration**: ~4 days (35 milestones @ 1h each)
**Dependencies**: Phase 2 (Core Backend Services)
**Status**: 🚧 TODO

## Goal

Implement the database access layer with support for AWS RDS (MySQL and PostgreSQL), connection management, and change execution capabilities. This phase enables the system to securely connect to target databases using both RDS IAM authentication and traditional credentials, execute approved SQL changes with transaction support, and provide rollback capabilities. Environment management for Staging and Production is also implemented.

## Phase-level Acceptance Criteria

- Database connection manager implemented with connection pooling (5-20 connections per database)
- RDS IAM authentication working for MySQL and PostgreSQL
- Traditional username/password authentication supported
- SSL/TLS enforcement for all database connections
- Environment management system supports Staging and Production configurations
- Database credentials encrypted at rest using AES-256
- Connectivity validation during database configuration
- SQL change execution engine operational with transaction support
- Multi-statement execution supported (up to 100 statements per transaction)
- Execution results captured (success/failure, affected rows, execution time)
- Transaction-level rollback capability implemented
- Dry-run mode for previewing changes without execution
- Concurrent execution prevention for same database (proper locking)
- Retry logic with exponential backoff for transient failures
- Multi-stage promotion (Staging -> Production) working
- Scheduled change execution implemented
- Integration tests for database operations passing
- Database connection error handling comprehensive

## Milestones

<!-- Milestones appended by /wiz:milestones -->
### P03M01: Create database connection model

**Status:** 🚧 TODO
**ID:** P03M01

**Goal**
Implement database connection configuration model for storing RDS connection details.

**Acceptance Criteria**
- [ ] DatabaseConnection struct with id, name, type (MySQL/PostgreSQL), host, port, database, auth_method
- [ ] Credentials encrypted at rest using AES-256
- [ ] ConnectionRepository interface and PostgreSQL implementation
- [ ] Unit tests passing

---

### P03M02: Implement RDS IAM authentication

**Status:** 🚧 TODO
**ID:** P03M02

**Goal**
Add RDS IAM token generation for MySQL and PostgreSQL connections.

**Acceptance Criteria**
- [ ] IAM token generator using AWS SDK
- [ ] Token refresh logic before expiration (15 min validity)
- [ ] Support for both MySQL and PostgreSQL protocols
- [ ] Error handling for IAM failures
- [ ] Unit tests passing

---

### P03M03: Implement traditional credentials authentication

**Status:** 🚧 TODO
**ID:** P03M03

**Goal**
Add username/password authentication for database connections.

**Acceptance Criteria**
- [ ] Credential storage with encryption
- [ ] Connection string builder for MySQL and PostgreSQL
- [ ] Password retrieval and decryption
- [ ] Unit tests passing

---

### P03M04: Create connection pool manager

**Status:** 🚧 TODO
**ID:** P03M04

**Goal**
Implement connection pooling with configurable min/max connections (5-20).

**Acceptance Criteria**
- [ ] ConnectionPool struct with min=5, max=20 connections
- [ ] Pool per target database
- [ ] Connection health checks
- [ ] Idle connection cleanup
- [ ] Pool metrics (active, idle connections)
- [ ] Unit tests passing

---

### P03M05: Enforce SSL/TLS for database connections

**Status:** 🚧 TODO
**ID:** P03M05

**Goal**
Configure SSL/TLS enforcement for all database connections.

**Acceptance Criteria**
- [ ] SSL mode set to 'require' for all connections
- [ ] Certificate validation enabled
- [ ] Connection fails without SSL
- [ ] Integration tests verify SSL enforcement

---

### P03M06: Implement database connectivity validator

**Status:** 🚧 TODO
**ID:** P03M06

**Goal**
Create validator that tests database connectivity during configuration.

**Acceptance Criteria**
- [ ] TestConnection method attempts connection and simple query
- [ ] Returns detailed error messages for failures
- [ ] Timeout set to 5 seconds
- [ ] Integration tests with valid and invalid configs

---

### P03M07: Create environment model and repository

**Status:** 🚧 TODO
**ID:** P03M07

**Goal**
Implement environment (Staging, Production) configuration model.

**Acceptance Criteria**
- [ ] Environment struct with id, name, database_connection_id, promotion_order
- [ ] EnvironmentRepository interface
- [ ] PostgreSQL implementation
- [ ] Unit tests passing

---

### P03M08: Implement environment management endpoints

**Status:** 🚧 TODO
**ID:** P03M08

**Goal**
Create API endpoints for managing environments (Admin only).

**Acceptance Criteria**
- [ ] POST /api/v1/environments - create environment
- [ ] GET /api/v1/environments - list environments
- [ ] PUT /api/v1/environments/:id - update environment
- [ ] DELETE /api/v1/environments/:id - delete environment
- [ ] Admin permission required
- [ ] Integration tests passing

---

### P03M09: Create SQL execution engine

**Status:** 🚧 TODO
**ID:** P03M09

**Goal**
Implement core SQL execution engine with transaction support.

**Acceptance Criteria**
- [ ] ExecutionEngine struct that executes SQL against target database
- [ ] Transaction BEGIN/COMMIT/ROLLBACK support
- [ ] Multi-statement execution (up to 100 statements)
- [ ] Execution timeout (5 minutes max)
- [ ] Result capture (rows affected, execution time)
- [ ] Unit tests with mock database

---

### P03M10: Implement dry-run execution mode

**Status:** 🚧 TODO
**ID:** P03M10

**Goal**
Add dry-run mode that validates SQL without executing changes.

**Acceptance Criteria**
- [ ] DryRunMode flag in execution request
- [ ] SQL parsed and validated but not executed
- [ ] Returns estimated impact (tables, columns affected)
- [ ] Transaction rolled back after validation
- [ ] Integration tests passing

---

### P03M11: Implement transaction-level rollback

**Status:** 🚧 TODO
**ID:** P03M11

**Goal**
Add automatic rollback on execution errors.

**Acceptance Criteria**
- [ ] All executions wrapped in transactions
- [ ] Automatic ROLLBACK on any statement failure
- [ ] Error details captured and returned
- [ ] Partial success not allowed (all-or-nothing)
- [ ] Integration tests verify rollback

---

### P03M12: Add execution locking mechanism

**Status:** 🚧 TODO
**ID:** P03M12

**Goal**
Prevent concurrent executions on the same database.

**Acceptance Criteria**
- [ ] Database-level locks using advisory locks or table locks
- [ ] Lock acquired before execution
- [ ] Lock released after completion
- [ ] Timeout if lock unavailable (30 seconds)
- [ ] Integration tests verify locking

---

### P03M13: Implement retry logic with exponential backoff

**Status:** 🚧 TODO
**ID:** P03M13

**Goal**
Add retry mechanism for transient database failures.

**Acceptance Criteria**
- [ ] Retry logic for connection failures
- [ ] Exponential backoff (1s, 2s, 4s, 8s)
- [ ] Max 3 retries
- [ ] Only retries transient errors (connection, timeout)
- [ ] Does not retry syntax or permission errors
- [ ] Unit tests passing

---

### P03M14: Create execution result model

**Status:** 🚧 TODO
**ID:** P03M14

**Goal**
Implement model for capturing and storing execution results.

**Acceptance Criteria**
- [ ] ExecutionResult struct (id, issue_id, environment_id, status, rows_affected, execution_time, error_message, executed_at)
- [ ] ExecutionResultRepository interface
- [ ] PostgreSQL implementation
- [ ] Unit tests passing

---

### P03M15: Implement change application endpoint

**Status:** 🚧 TODO
**ID:** P03M15

**Goal**
Create POST /api/v1/issues/:id/apply endpoint for executing approved changes.

**Acceptance Criteria**
- [ ] Endpoint requires Approved issue status
- [ ] Executes SQL against target database
- [ ] Updates issue status to InProgress, then Applied or Failed
- [ ] Stores execution results
- [ ] Returns execution summary
- [ ] Integration tests passing

---

### P03M16: Implement scheduled execution

**Status:** 🚧 TODO
**ID:** P03M16

**Goal**
Add scheduling capability for delayed change execution.

**Acceptance Criteria**
- [ ] ScheduledExecution model with execution_time
- [ ] Background job polls for pending scheduled executions
- [ ] Executes changes at specified time (±1 minute tolerance)
- [ ] Updates issue status appropriately
- [ ] Notifications sent on completion
- [ ] Integration tests passing

---

### P03M17: Implement multi-stage promotion (Staging -> Production)

**Status:** 🚧 TODO
**ID:** P03M17

**Goal**
Add workflow for promoting changes from Staging to Production.

**Acceptance Criteria**
- [ ] Changes must be applied to Staging before Production
- [ ] Separate approval required for Production
- [ ] Promotion endpoint POST /api/v1/issues/:id/promote
- [ ] Validation that Staging execution succeeded
- [ ] Integration tests passing

---

### P03M18: Add execution history tracking

**Status:** 🚧 TODO
**ID:** P03M18

**Goal**
Track all execution attempts with results.

**Acceptance Criteria**
- [ ] All executions logged in execution_results table
- [ ] History includes retries and failures
- [ ] GET /api/v1/issues/:id/executions endpoint
- [ ] Pagination for execution history
- [ ] Integration tests passing

---

### P03M19: Implement execution rollback tracking

**Status:** 🚧 TODO
**ID:** P03M19

**Goal**
Generate and store rollback SQL for executed changes.

**Acceptance Criteria**
- [ ] Rollback SQL generated where possible (INSERT -> DELETE, UPDATE with original values)
- [ ] Stored in execution_results
- [ ] Manual rollback endpoint POST /api/v1/executions/:id/rollback
- [ ] Rollback applies inverse operations
- [ ] Integration tests passing

---

### P03M20: Add execution monitoring and alerting

**Status:** 🚧 TODO
**ID:** P03M20

**Goal**
Implement monitoring for long-running and failed executions.

**Acceptance Criteria**
- [ ] Execution duration tracked
- [ ] Alerts for executions >2 minutes
- [ ] Alerts for failed executions
- [ ] Metrics exposed for monitoring systems
- [ ] Integration tests verify alerts

---

### P03M21: Write unit tests for connection pool

**Status:** 🚧 TODO
**ID:** P03M21

**Goal**
Create comprehensive tests for connection pool manager.

**Acceptance Criteria**
- [ ] Test pool initialization
- [ ] Test connection acquisition and release
- [ ] Test connection health checks
- [ ] Test pool exhaustion handling
- [ ] All tests passing
- [ ] Code coverage >85%

---

### P03M22: Write unit tests for execution engine

**Status:** 🚧 TODO
**ID:** P03M22

**Goal**
Create comprehensive tests for SQL execution engine.

**Acceptance Criteria**
- [ ] Test successful execution
- [ ] Test transaction rollback on error
- [ ] Test multi-statement execution
- [ ] Test timeout handling
- [ ] All tests passing
- [ ] Code coverage >90%

---

### P03M23: Write integration tests for RDS IAM auth

**Status:** 🚧 TODO
**ID:** P03M23

**Goal**
Create integration tests for RDS IAM authentication flow.

**Acceptance Criteria**
- [ ] Test IAM token generation
- [ ] Test connection with IAM token to test RDS instance
- [ ] Test token refresh
- [ ] Test error handling for invalid credentials
- [ ] All tests passing

---

### P03M24: Write integration tests for database operations

**Status:** 🚧 TODO
**ID:** P03M24

**Goal**
Create integration tests for database connection and execution.

**Acceptance Criteria**
- [ ] Test MySQL connection and query execution
- [ ] Test PostgreSQL connection and query execution
- [ ] Test SSL/TLS enforcement
- [ ] Test transaction rollback
- [ ] All tests passing with test databases

---

### P03M25: Write integration tests for change application

**Status:** 🚧 TODO
**ID:** P03M25

**Goal**
Create end-to-end tests for change application flow.

**Acceptance Criteria**
- [ ] Test complete flow: issue creation -> approval -> application
- [ ] Test dry-run mode
- [ ] Test execution failures and rollback
- [ ] Test multi-stage promotion
- [ ] All tests passing

---

### P03M26: Add database connection metrics

**Status:** 🚧 TODO
**ID:** P03M26

**Goal**
Expose Prometheus metrics for database connections.

**Acceptance Criteria**
- [ ] Active connections per database gauge
- [ ] Connection acquisition duration histogram
- [ ] Connection errors counter
- [ ] Query execution duration histogram
- [ ] Metrics endpoint /metrics updated

---

### P03M27: Implement connection health monitoring

**Status:** 🚧 TODO
**ID:** P03M27

**Goal**
Add periodic health checks for database connections.

**Acceptance Criteria**
- [ ] Background job pings databases every 60 seconds
- [ ] Unhealthy connections removed from pool
- [ ] Alerts triggered for connection failures
- [ ] Health status included in /health endpoint
- [ ] Integration tests passing

---

### P03M28: Add execution audit logging

**Status:** 🚧 TODO
**ID:** P03M28

**Goal**
Log all database change executions in audit log.

**Acceptance Criteria**
- [ ] Audit log entry for each execution attempt
- [ ] Includes SQL content, database, user, timestamp, result
- [ ] Immutable audit entries
- [ ] Queryable via audit log API
- [ ] Integration tests passing

---

### P03M29: Implement execution timeout handling

**Status:** 🚧 TODO
**ID:** P03M29

**Goal**
Enforce 5-minute timeout on SQL executions.

**Acceptance Criteria**
- [ ] Context with timeout passed to execution
- [ ] Execution cancelled after 5 minutes
- [ ] Timeout error returned
- [ ] Partial changes rolled back
- [ ] Integration tests verify timeout

---

### P03M30: Add execution error classification

**Status:** 🚧 TODO
**ID:** P03M30

**Goal**
Classify execution errors for better diagnostics.

**Acceptance Criteria**
- [ ] Error types: Syntax, Permission, Timeout, Connection, Constraint, Unknown
- [ ] Error classification logic
- [ ] User-friendly error messages
- [ ] Suggested resolutions for common errors
- [ ] Unit tests passing

---

### P03M31: Implement database backup verification

**Status:** 🚧 TODO
**ID:** P03M31

**Goal**
Verify database backups exist before applying Production changes.

**Acceptance Criteria**
- [ ] Check RDS backup status via AWS API
- [ ] Warn if no backup within 24 hours
- [ ] Allow override with Admin permission
- [ ] Integration tests with mock AWS API

---

### P03M32: Add execution performance optimization

**Status:** 🚧 TODO
**ID:** P03M32

**Goal**
Optimize execution performance for large SQL scripts.

**Acceptance Criteria**
- [ ] Batch statement execution where possible
- [ ] Prepared statement caching
- [ ] Connection reuse within execution
- [ ] Performance tests show improvement
- [ ] No degradation in correctness

---

### P03M33: Document database connection setup

**Status:** 🚧 TODO
**ID:** P03M33

**Goal**
Create comprehensive documentation for database configuration.

**Acceptance Criteria**
- [ ] RDS setup instructions
- [ ] IAM authentication configuration guide
- [ ] SSL certificate setup
- [ ] Connection troubleshooting guide
- [ ] Example configurations for MySQL and PostgreSQL

---

### P03M34: Update OpenAPI documentation for execution endpoints

**Status:** 🚧 TODO
**ID:** P03M34

**Goal**
Add database integration endpoints to OpenAPI spec.

**Acceptance Criteria**
- [ ] Environment management endpoints documented
- [ ] Change application endpoint documented
- [ ] Dry-run endpoint documented
- [ ] Execution history endpoints documented
- [ ] Request/response schemas defined

---

### P03M35: Verify Phase 3 Completion

**Status:** 🚧 TODO
**ID:** P03M35

**Goal**
Verify that all Phase 3 requirements are met, tests pass, and the phase is ready for sign-off before proceeding to Phase 4.

**Acceptance Criteria**
- [ ] All previous milestones in Phase 3 marked as complete
- [ ] All tests pass (unit and integration)
- [ ] Code coverage meets project standards (>85%)
- [ ] No outstanding bugs or issues from this phase
- [ ] Code reviewed and meets quality standards
- [ ] Database connections working (IAM auth and credentials)
- [ ] SQL execution functional with transaction support
- [ ] Dry-run mode working
- [ ] Multi-stage promotion working
- [ ] Connection pooling and health monitoring operational
- [ ] Ready to proceed to Phase 4

---
