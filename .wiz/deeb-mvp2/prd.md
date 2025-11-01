---
title: "Deeb MVP2 - Enhanced Database Change Management System"
slug: deeb-mvp2
version: "1.0.0"
status: "Draft"
created: "2025-11-01"
updated: "2025-11-01"
owner: "TBD"
primary_language: "Go, TypeScript"
benchmarking_policy: "Skip"
fuzzing_policy: "Skip"
---

# Product Requirements Document: Deeb MVP2 - Enhanced Database Change Management System

## Executive Summary

Deeb MVP2 is an enhanced version of deeb-mvp that refines project definitions and clarifies MVP scope based on lessons learned. The system enables engineers to manage database changes securely without direct database access, providing a controlled workflow with automated validation, approval processes, and comprehensive audit trails. MVP2 focuses on delivering clear project definitions and a refined MVP scope within a compressed 4-week timeline, rebuilding the database schema from scratch while leveraging existing codebase components where appropriate.

## Background

Direct database access by engineers creates significant security, compliance, and operational risks:
- **Security vulnerabilities**: Unrestricted database access increases attack surface
- **Compliance gaps**: Lack of comprehensive audit trails makes regulatory compliance difficult
- **Operational incidents**: Manual changes without proper review can cause outages or data corruption
- **Knowledge silos**: Database changes lack visibility and historical context

Deeb addresses these challenges by centralizing database change management through Change Plans with approval workflows, automated policy validation, and comprehensive audit logging. MVP2 builds upon deeb-mvp's foundation but refines definitions and clarifies scope to address missing features and unclear definitions identified in the original MVP.

## Problem Statement

**Current State**: Engineers manually apply database changes with direct access to databases, leading to security risks, compliance gaps, and operational incidents. The original deeb-mvp had unclear project definitions and MVP scope, resulting in missing features and implementation ambiguity.

**Desired State**: All database changes flow through Deeb's controlled workflow with automated validation, approval gates, and comprehensive audit trails, eliminating the need for direct database access. MVP2 provides clear, well-defined project scope and requirements for successful implementation.

**Top Pain Points**:
1. Unsafe direct database access by multiple team members
2. Lack of change history and audit trails for compliance
3. Missing features and unclear definitions in original MVP
4. Need for refined project definitions and MVP scope
5. Unclear implementation boundaries and feature priorities

## Goals and Non-Goals

### Goals

1. **Clear Project Definitions**: Establish unambiguous definitions for all core concepts (Projects, Environments, Database Instances, Databases, Change Plans, SQL Review, Approval Process, Rollout)
2. **Refined MVP Scope**: Deliver a focused MVP with clear boundaries and prioritized features from docs/mvp.md Phase 1
3. **Fundamental Workflow**: Enable Create → Approve → Execute → Audit workflow for database changes
4. **Security First**: Ensure all database changes are audited, policy-enforced, and transparent
5. **Fresh Schema Foundation**: Rebuild database schema from scratch aligned with refined project definitions
6. **Codebase Evolution**: Leverage existing deeb-mvp codebase while rebuilding/adjusting features to fit new definitions

### Non-Goals

1. **Multi-database changes**: Single database target only (Phase 2+)
2. **Multi-stage deployment**: Single environment execution only (Phase 2+)
3. **Scheduled execution**: Immediate execution only (Phase 2+)
4. **Groups**: User groups deferred to Phase 2+
5. **2FA**: Two-factor authentication deferred to Phase 3+
6. **Multiple OAuth providers**: Google OAuth only (GitHub in Phase 3+)
7. **Risk-based approval workflows**: Simple 1-approver flow only (Phase 2+)
8. **Schema drift detection**: Deferred to Phase 3+
9. **Read-only query execution**: Deferred to Phase 3+
10. **Email/Slack notifications**: Deferred to Phase 2+
11. **Transaction opt-out**: All changes wrapped in transactions (Phase 2+)
12. **Advanced audit log filtering**: Basic filtering only (Phase 2+)
13. **Change history visualization**: Basic history storage only (Phase 2+)

## User Personas and Use Cases

### Primary Personas

1. **Backend Engineers (Developers)**
   - **Role**: Create and manage Change Plans
   - **Needs**: Simple workflow for submitting database changes, visibility into approval status
   - **Constraints**: No direct database access

2. **Site Reliability Engineers (SREs)**
   - **Role**: Approve and execute Change Plans
   - **Needs**: Review changes before execution, ensure system reliability
   - **Constraints**: Must understand impact of changes

3. **Database Administrators (DBAs)**
   - **Role**: Manage database instances, approve high-risk changes
   - **Needs**: Control over database access, visibility into all changes
   - **Constraints**: Ensure security and compliance

### Key Use Cases

**UC1: Create and Submit Change Plan**
- **Actor**: Backend Engineer (Project Developer role)
- **Trigger**: Engineer needs to modify database schema or data
- **Flow**:
  1. Engineer creates Change Plan with title, description, SQL content
  2. Selects target database (single database)
  3. Classifies change type (DDL or DML)
  4. Optionally provides rollback SQL script
  5. Deeb will assign risk level (Low, Medium, High) based on the SQL content and the target database, target environment.
  6. Submits Change Plan (status: Draft → Open)
  7. SQL Review automatically runs
- **Success**: Change Plan in "Open" status with SQL Review results displayed

**UC2: Review and Approve Change Plan**
- **Actor**: SRE or DBA (Project Approver role)
- **Trigger**: Change Plan submitted for review
- **Flow**:
  1. Reviewer views Change Plan details
  2. Reviews SQL Review results (errors block, warnings inform)
  3. Reviews SQL content and change description
  4. Approves or closes Change Plan with comments
  5. If approved and no errors: Change Plan status → Approved
- **Success**: Change Plan approved and ready for execution

**UC3: Execute Change Plan**
- **Actor**: Organization Admin, Organization DBA, Project Owner, or Change Plan Author.
- **Trigger**: Change Plan approved and ready for execution
- **Flow**:
  1. Executor triggers immediate execution
  2. System captures schema snapshot before execution
  3. System executes SQL in transaction
  4. On success: Commit transaction, capture post-execution snapshot, update status → Completed
  5. On failure: Rollback transaction, update status → Failed, report error
- **Success**: Change Plan successfully executed or safely rolled back

**UC4: Audit Change History**
- **Actor**: Organization Admin or DBA
- **Trigger**: Need to review change history for compliance
- **Flow**:
  1. Admin views audit log UI
  2. Filters by user, database, time range, action type
  3. Views detailed change history with before/after snapshots
- **Success**: Complete audit trail visible for compliance

## Technical Architecture

### High-Level Architecture

```
┌─────────────────┐
│   Web UI        │  React + TypeScript + Vite + shadcn/ui
│   (Frontend)    │
└────────┬────────┘
         │ HTTP/REST
┌────────┴────────┐
│   REST API      │  Go (Echo v4)
│   (Backend)     │
└────────┬────────┘
         │
    ┌────┴────┐
    │        │
┌───▼───┐ ┌──▼──────────┐
│ Deeb  │ │ Target DBs  │
│Metadata│ │(MySQL/      │
│Postgres│ │ PostgreSQL) │
└───────┘ └─────────────┘
```

### Component Architecture

1. **Authentication Layer**
   - OAuth 2.0 with Google
   - Email/password authentication
   - JWT token management
   - Session handling

2. **Authorization Layer**
   - RBAC (Role-Based Access Control)
   - Organization-level roles (Admin, DBA, Developer)
   - Project-level roles (Owner, Developer, Approver, Viewer)
   - Permission checking middleware

3. **Change Plan Management**
   - CRUD operations for Change Plans
   - Status workflow management
   - Change Plan assignment and routing

4. **SQL Review Engine**
   - Syntax validation (MySQL/PostgreSQL parsers)
   - Security validation (destructive operations)
   - Performance validation (missing indexes, primary keys)
   - Backwards compatibility validation
   - Best practices validation
   - Policy finding reporting

5. **Approval Workflow Engine**
   - Single-approval workflow
   - Approval routing
   - Approval history tracking

6. **Execution Engine**
   - Transaction management
   - Schema snapshot capture (pg_dump, mysqldump)
   - SQL execution with rollback
   - Concurrency control (one change per database)
   - Error handling and reporting

7. **Audit Logging System**
   - Action logging (all user actions)
   - Change history storage
   - Schema snapshot storage
   - Audit log querying and filtering

8. **Database Connection Management**
   - Database instance management
   - Connection pooling
   - Credential storage (encrypted)
   - Connection testing

### Integration Points

- **OAuth Provider**: Google OAuth 2.0 for authentication
- **Target Databases**: MySQL and PostgreSQL via username/password authentication
- **Schema Snapshots**: pg_dump for PostgreSQL, mysqldump/information_schema for MySQL

## Functional Requirements

### FR1: Authentication and Authorization

**FR1.1**: System shall support OAuth 2.0 SSO with Google provider
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Users can authenticate via Google OAuth
  - OAuth tokens are validated and stored securely
  - JWT tokens issued for authenticated sessions

**FR1.2**: System shall support email/password authentication
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Users can register with email/password
  - Passwords are hashed using bcrypt
  - Users can login with email/password

**FR1.3**: System shall implement RBAC with organization and project roles
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Organization roles: Admin, DBA, Developer
  - Project roles: Owner, Developer, Approver, Viewer
  - Permissions enforced at API and UI level
  - Role assignments managed by admins

**FR1.4**: System shall provide user management UI (admin only)
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Admins can create, update, deactivate users
  - Admins can assign organization roles
  - Admins can assign project roles

### FR2: Projects and Environments

**FR2.1**: System shall support Projects for organizing resources
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Admins can create/update/delete Projects
  - Projects have name, description
  - Projects group databases and team members

**FR2.2**: System shall support Environments (Staging, Production)
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Admins can create/update/delete Environments
  - Environments have name, description, policy_id (references Policy resource)
  - Each database belongs to exactly one Environment

**FR2.3**: System shall support Policies
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Admins can create/update/delete Policies
  - Policies have name, description, type, and policy configuration
  - Policy type: SQL_REVIEW (other types can be added in future)
  - Policy configuration defines which validators are enabled and their settings (for SQL_REVIEW type)
  - Policies can be referenced by multiple Environments
  - Policy configuration stored as JSONB

**FR2.4**: System shall assign databases to Projects and Environments
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Each database belongs to exactly one Project
  - Each database belongs to exactly one Environment
  - Database assignment managed by admins

### FR3: Database Management

**FR3.1**: System shall support MySQL and PostgreSQL database instances
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Admins can add database instances (name, host, port, credentials)
  - System supports MySQL and PostgreSQL
  - Connection via username/password authentication
  - System can test connection before saving

**FR3.2**: System shall list all registered databases
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Database management UI shows all instances
  - Database instances show associated databases
  - Database credentials stored encrypted

### FR4: Change Plan Workflow

**FR4.1**: System shall support Change Plan creation
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Users with Project Developer role can create Change Plans
  - Change Plans require: title, change type (DDL/DML), SQL content, target database
  - Change Plans support: optional description, optional rollback SQL
  - Change Plans start in Draft status

**FR4.2**: System shall support Change Plan status workflow
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Status transitions: Draft → Open → Approved/Closed → InProgress → Completed/Failed
  - Draft → Open: Author submits (triggers SQL Review)
  - Open → Approved: Project Approver approves
  - Open → Closed: Project Approver closes/rejects
  - Approved → InProgress: Authorized user triggers execution
  - InProgress → Completed: Execution succeeds
  - InProgress → Failed: Execution fails (rollback)

**FR4.3**: System shall support Change Plan viewing and filtering
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Users can view Change Plan details
  - Users can list Change Plans with filters (status, project, database, author)
  - Dashboard shows overview of open/recent Change Plans

### FR5: SQL Review

**FR5.1**: System shall automatically run SQL Review on Change Plan submission
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - SQL Review runs when Change Plan transitions Draft → Open
  - SQL Review validates syntax (MySQL/PostgreSQL parsers)
  - SQL Review detects destructive operations (DROP TABLE, DROP DATABASE, TRUNCATE)

**FR5.2**: System shall display SQL Review results
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - SQL Review errors block execution (must fix before approval)
  - SQL Review warnings shown but don't block
  - Results displayed in Change Plan details page
  - Results include severity (Error, Warning, Pass)

**FR5.3**: System shall support comprehensive SQL validation
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Security validation: drop tables/columns with data
  - Performance validation: missing indexes, no primary key
  - Backwards compatibility validation: breaking changes detection
  - Best practices validation: naming conventions, data types

**FR5.4**: System shall allow authors to fix and resubmit Change Plans with errors
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - If SQL Review has errors, author can fix and resubmit
  - Change Plan returns to Open status after resubmission
  - SQL Review runs again on resubmission
  - User can trigger SQL Review manually from the Change Plan details page.

### FR6: Approval Workflow

**FR6.1**: System shall support simple approval flow (1 approver required)
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Any Project Approver in the project can approve
  - Approvers can approve or close Change Plans
  - Approval includes optional comments
  - Approval history tracked

**FR6.2**: System shall support optional reviewer assignment
- **Priority**: P1 (Should Have)
- **Acceptance Criteria**:
  - Authors can manually assign reviewers
  - Reviewers notified when assigned
  - Reviewer assignment visible in Change Plan details

### FR7: Change Execution

**FR7.1**: System shall execute Change Plans immediately on trigger
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Only Approved Change Plans can be executed
  - Only users with rollout permissions can execute (Organization Admin, Organization DBA, Project Owner, or user with rollout permission)
  - Execution is immediate (no scheduling in MVP)

**FR7.2**: System shall capture schema snapshots before and after execution
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - System captures schema snapshot before execution
  - System captures schema snapshot after successful execution
  - Uses pg_dump for PostgreSQL, mysqldump/information_schema for MySQL
  - Snapshots stored securely

**FR7.3**: System shall execute SQL in transaction with automatic rollback
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - All SQL wrapped in transaction
  - On success: transaction committed
  - On failure: transaction rolled back automatically
  - Transaction wrapping cannot be disabled in MVP

**FR7.4**: System shall capture execution results
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - System captures success/error status
  - System captures rows affected
  - System captures execution duration
  - Results stored in Change Plan run log history.

**FR7.5**: System shall enforce concurrency control
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - One Change Plan executes at a time per database
  - Other Change Plans queue automatically
  - Concurrent changes on different databases allowed

### FR8: Error Handling

**FR8.1**: System shall handle SQL execution errors gracefully
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - All SQL errors logged with full context
  - Automatic transaction rollback on failure
  - Error displayed to user who triggered execution
  - Change Plan marked as "failed" with error details

### FR9: Audit Logging

**FR9.1**: System shall log all user actions
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Logs: user login/logout, database added/removed, Change Plan created/approved/closed/completed, execution success/failure
  - Stores: timestamp, user, action, target resource, result
  - Audit log UI available (admin only)

**FR9.2**: System shall store change history
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Detailed migration history stored for each Change Plan
  - Before/after schema snapshots stored
  - Change history queryable and filterable

### FR10: UI Components

**FR10.1**: System shall provide Dashboard
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Overview of open/recent Change Plans
  - Quick access to pending approvals
  - System status indicators

**FR10.2**: System shall provide management UIs
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Projects & Environments management (admin)
  - Database management page (admin)
  - User management (admin)
  - Audit log viewer (admin)

**FR10.3**: System shall provide Change Plan UIs
- **Priority**: P0 (Must Have)
- **Acceptance Criteria**:
  - Create Change Plan form
  - Change Plan detail view
  - Change Plan list with filters

## Non-Functional Requirements

### P0: Correctness

**NFR-P0-1**: System shall validate all SQL syntax before execution
- **Validation**: SQL parser libraries for MySQL and PostgreSQL
- **Error Handling**: Syntax errors must prevent execution

**NFR-P0-2**: System shall prevent SQL injection attacks
- **Validation**: Parameterized queries only, no dynamic SQL construction
- **Error Handling**: Invalid SQL patterns rejected

**NFR-P0-3**: System shall handle edge cases in status transitions
- **Validation**: Status transition rules enforced
- **Error Handling**: Invalid transitions rejected with clear error messages

**NFR-P0-4**: System shall handle database connection failures gracefully
- **Validation**: Connection testing before execution
- **Error Handling**: Connection failures logged and reported

**NFR-P0-5**: System shall ensure data consistency
- **Validation**: Transaction wrapping mandatory
- **Error Handling**: Rollback on any error

### P1: Tests

**NFR-P1-1**: System shall have comprehensive unit test coverage (>85%)
- **Coverage**: All business logic, validators, workflow engines
- **Framework**: Go testing package, https://vitest.dev/ for TypeScript

**NFR-P1-2**: System shall have integration tests for API endpoints
- **Coverage**: All REST API endpoints
- **Framework**: Go integration tests with testcontainers

**NFR-P1-3**: System shall have end-to-end tests for critical workflows
- **Coverage**: Create → Approve → Execute workflow
- **Framework**: Playwright for frontend E2E

**NFR-P1-4**: All tests shall pass before deployment
- **CI/CD**: Tests run in CI pipeline
- **Requirement**: Zero failing tests, zero skipped tests

### P2: Security

**NFR-P2-1**: System shall encrypt database credentials at rest
- **Encryption**: AES-256 encryption for stored credentials
- **Key Management**: Secure key storage

**NFR-P2-2**: System shall use HTTPS for all communications
- **TLS**: TLS 1.2+ required
- **Validation**: Certificate validation enforced

**NFR-P2-3**: System shall implement JWT token expiration
- **Expiration**: Tokens expire after 24 hours (configurable)
- **Refresh**: Refresh token mechanism

**NFR-P2-4**: System shall enforce RBAC at API and UI level
- **Authorization**: Middleware checks permissions
- **Principle**: Least privilege access

**NFR-P2-5**: System shall log all authentication attempts
- **Audit**: Failed login attempts logged
- **Security**: Rate limiting on failed attempts

**NFR-P2-6**: System shall sanitize all user inputs
- **Validation**: Input validation on all endpoints
- **Sanitization**: SQL injection prevention

### P3: Quality

**NFR-P3-1**: Code shall pass linting checks
- **Go**: golangci-lint passing with zero errors
- **TypeScript**: ESLint passing with zero errors

**NFR-P3-2**: Code shall follow language-specific style guides
- **Go**: Effective Go guidelines
- **TypeScript**: TypeScript style guide

**NFR-P3-3**: System shall have comprehensive API documentation
- **Documentation**: OpenAPI/Swagger specification
- **Coverage**: All endpoints documented

**NFR-P3-4**: System shall have user documentation
- **Documentation**: User guides, admin guides
- **Coverage**: Setup, configuration, usage

**NFR-P3-5**: Code shall be maintainable and well-structured
- **Structure**: Clear separation of concerns
- **Documentation**: Inline comments for complex logic

### P4: Performance

**NFR-P4-1**: API response time shall be < 200ms (95th percentile)
- **Target**: Fast response for UI interactions
- **Measurement**: Response time monitoring

**NFR-P4-2**: SQL execution shall complete within reasonable time
- **Timeout**: 5 minutes maximum execution time
- **Monitoring**: Execution duration tracked

**NFR-P4-3**: System shall support concurrent users
- **Target**: 50+ concurrent users
- **Load Testing**: Load tests validate capacity

**NFR-P4-4**: Database queries shall be optimized
- **Indexing**: Appropriate indexes on metadata database
- **Query Optimization**: Efficient queries

## Design Guidelines

Detailed design guidelines will be applied during implementation (Phase 1). Guidelines will cover:
- Go backend patterns and best practices
- React/TypeScript frontend patterns
- Database schema design principles
- API design standards
- Testing strategies

## Milestones and Phasing

Milestones and phases will be generated by `/wiz-phases` command after PRD approval.

**Rough Timeline Estimate**: 4 weeks with focused development effort

## Open Questions

1. **Database Credentials Storage**: How to securely store target database credentials?
   - Options: AWS Secrets Manager, HashiCorp Vault, encrypted DB column
   - Recommendation: Encrypted DB column for MVP, external secrets manager for production
   - Answer: Encrypted DB column for MVP and AWS Secrets Manager for production.

2. **Schema Snapshot Storage**: Where to store snapshots?
   - Options: S3, database BYTEA column, separate file storage
   - Recommendation: Database BYTEA column for MVP, S3 for production
   - Answer: Database BYTEA or JSONB column for MVP.

3. **Transaction Isolation**: What isolation level for changes?
   - Recommendation: SERIALIZABLE for safety
   - Answer: SERIALIZABLE for safety.

4. **Change Size Limits**: Max SQL size? Max execution time?
   - Recommendation: 10MB SQL, 5min timeout initially
   - Answer: 10MB SQL, 5min timeout initially and configurable.

5. **User Onboarding**: How do first admin get created?
   - Options: Bootstrap script, first OAuth user becomes admin, invite system
   - Recommendation: Bootstrap script for MVP
   - Answer: First user to sign up is the admin. After that, users can be added by the admin or Project Owner.

6. **Concurrent Changes**: Allow simultaneous changes on different databases?
   - Recommendation: Yes, but lock same-database changes (already specified)
   - Answer: Yes, but lock same-database changes.

## Success Criteria

**MVP2 Launch Criteria**:
- All Phase 1 features from docs/mvp.md implemented
- Clear project definitions documented and validated
- Database schema rebuilt and aligned with definitions
- Zero security incidents
- 100% audit log coverage
- Comprehensive test coverage (>85%)
- All linting passing

**Product-Market Fit Indicators**:
- 5+ users can authenticate and use system
- 10+ database instances managed
- 50+ Change Plans created and executed successfully
- Users prefer Deeb over direct database access
- Clear understanding of project definitions and scope

## Appendix

### References

- **Project Definitions**: `docs/project.md`
- **MVP Roadmap**: `docs/mvp.md`
- **Existing deeb-mvp PRD**: `.wiz/deeb-mvp/prd.md`

### Research Sources

- Go Echo Framework: https://echo.labstack.com/
- React Best Practices: https://react.dev/
- PostgreSQL Documentation: https://www.postgresql.org/docs/
- MySQL Documentation: https://dev.mysql.com/doc/
- OAuth 2.0 Specification: https://oauth.net/2/
- Database Migration Best Practices: Industry standards for safe database changes

### Glossary

- **Change Plan**: A request to modify a database, including SQL content, metadata, and approval workflow
- **SQL Review**: Automated validation of SQL against configured policies
- **Rollout**: The process of executing a Change Plan against a target database
- **Schema Snapshot**: A captured representation of database schema at a point in time
- **Environment**: A deployment stage (Dev, Staging, Production)
- **Project**: A logical grouping of databases, team members, and workflows
