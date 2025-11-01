# Implementation Guide: Deeb MVP2 - Enhanced Database Change Management System

**PRD**: deeb-mvp2
**Status**: Ready for implementation
**Primary Language(s)**: Go, TypeScript
**Total Phases**: 6
**Estimated Duration**: 4 weeks (~202 milestones @ 1h each)

## Overview

Deeb MVP2 is an enhanced version of deeb-mvp that refines project definitions and clarifies MVP scope. The system enables engineers to manage database changes securely without direct database access, providing a controlled workflow with automated validation, approval processes, and comprehensive audit trails.

MVP2 focuses on delivering clear project definitions and a refined MVP scope within a compressed 4-week timeline, rebuilding the database schema from scratch while leveraging existing codebase components where appropriate.

## How to Use This Guide

This guide provides step-by-step instructions for implementing the project using the Wiz Planner workflow.

### Starting Implementation

If no work has been done so far, start with Phase 1, Milestone 1 (P01M01). Otherwise, continue to the next milestone in ascending order.

To begin implementation:
```bash
/wiz:next
```

This will load the next TODO milestone and guide you through implementation.

### Workflow Commands

- `/wiz:next` - Load next TODO milestone and begin work
- `/wiz:status` - View progress across all phases
- `/wiz:resume` - Resume work on in-progress milestone
- `/wiz:review-milestone <id>` - Review and mark milestone complete
- `/wiz:review-phase <number>` - Review entire phase completion

## Phase Overview

### Phase 1: Foundation & Database Schema Rebuild (~4 days, 37 milestones)

**Goal**: Rebuild the database schema from scratch aligned with refined project definitions, establish foundational infrastructure for authentication and authorization, and set up core utilities and configuration management.

**Key Deliverables**:
- Fresh PostgreSQL schema with all core tables (including policies)
- Database migration system (golang-migrate)
- User authentication (OAuth 2.0 with Google + email/password)
- JWT token generation and validation
- RBAC foundation
- Credential encryption system
- Configuration management
- Logging infrastructure

**Dependencies**: None (first phase)

### Phase 2: Core Resources Management (~3 days, 31 milestones)

**Goal**: Implement core resource management functionality including Projects, Environments, Policies, Database Instances, and Databases.

**Key Deliverables**:
- Projects CRUD operations
- Environments CRUD operations (with Policy assignment)
- Policy CRUD operations (SQL_REVIEW type)
- Database Instance CRUD operations with connection testing
- Databases CRUD operations
- Project membership management
- Authorization middleware for resource access

**Dependencies**: Phase 1

### Phase 3: Change Plan Management & SQL Review Engine (~5 days, 40 milestones)

**Goal**: Implement Change Plan lifecycle management and comprehensive SQL Review engine with multiple validators.

**Key Deliverables**:
- Change Plan CRUD operations
- Status workflow (Draft → Open → Approved/Closed → InProgress → Completed/Failed)
- Automated risk level assignment
- SQL Review engine architecture (pluggable validators)
- 5 validators: syntax, security, performance, backwards compatibility, best practices
- Automatic and manual SQL Review
- Policy violation storage and querying

**Dependencies**: Phase 2

### Phase 4: Approval Workflow & Execution Engine (~5 days, 38 milestones)

**Goal**: Implement approval workflow system and change execution engine with transaction management, schema snapshot capture, and concurrency control.

**Key Deliverables**:
- Approval workflow engine (single-approval flow)
- Schema snapshot capture (before and after execution)
- SQL execution wrapped in transactions (SERIALIZABLE isolation)
- Concurrency control (one Change Plan per database)
- Execution results capture and logging
- Change Plan queuing system

**Dependencies**: Phase 3

### Phase 5: Audit Logging & Error Handling (~3 days, 25 milestones)

**Goal**: Implement comprehensive audit logging system and robust error handling across all operations.

**Key Deliverables**:
- Audit logging system
- All user actions logged
- Change history tracking
- Error handling middleware
- Standardized error responses
- Error context capture

**Dependencies**: Phase 4

### Phase 6: UI Components & Integration (~4 days, 35 milestones)

**Goal**: Build comprehensive React frontend UI components for all user-facing features and integrate with backend APIs.

**Key Deliverables**:
- Dashboard UI
- Authentication UI (login, registration, OAuth)
- Resource management UIs (Projects, Environments, Policies, Databases)
- Change Plan UI (create, list, detail, filtering)
- SQL Review results display
- Approval workflow UI
- Execution trigger UI
- Audit log viewer UI
- RBAC enforcement at UI level
- Frontend unit tests (Vitest)
- E2E tests (Playwright)

**Dependencies**: Phase 5

## Design Guidelines

This project uses the following design guidelines:
- **Go**: Follow Effective Go guidelines and project-specific patterns
- **TypeScript/React**: Follow TypeScript best practices and React patterns
- **Database**: PostgreSQL schema design principles
- **API**: RESTful API design standards

Refer to existing codebase patterns and language-specific style guides during implementation.

## NFR Priority Order

This project follows the Wiz Planner NFR priority order:

1. **P0 - Correctness**: Code must work correctly for all inputs
   - SQL syntax validation before execution
   - SQL injection prevention
   - Status transition validation
   - Data consistency (transaction wrapping)

2. **P1 - Regression Prevention**: Tests prevent future breakage
   - Unit test coverage >85% for business logic
   - Integration tests for API endpoints
   - E2E tests for critical workflows
   - All tests pass before deployment

3. **P2 - Security**: System is secure against threats
   - Credential encryption (AES-256)
   - HTTPS for all communications
   - JWT token expiration
   - RBAC enforcement
   - Input sanitization

4. **P3 - Quality**: Code is maintainable and documented
   - Code passes linting checks (golangci-lint, ESLint)
   - Follows language-specific style guides
   - API documentation (OpenAPI/Swagger)
   - User documentation

5. **P4 - Performance**: System meets performance targets
   - API response time < 200ms (95th percentile)
   - SQL execution timeout: 5 minutes maximum
   - Support for 50+ concurrent users
   - Optimized database queries

Address priorities in this order during implementation.

## Testing Strategy

### Unit Testing
- **Backend**: Go testing package with table-driven tests
- **Frontend**: Vitest with React Testing Library
- **Coverage Target**: >85% for business logic
- **Framework**: Comprehensive unit tests for all validators, services, and components

### Integration Testing
- **Backend**: Go integration tests with testcontainers
- **Coverage**: All REST API endpoints
- **Database**: Test database setup using testcontainers

### End-to-End Testing
- **Framework**: Playwright
- **Coverage**: Critical workflows (Create → Approve → Execute)
- **Scenarios**: Authentication flow, RBAC enforcement, Change Plan workflow

### Benchmarking Policy
**Status**: Skip (per answers.json)

### Fuzzing Policy
**Status**: Skip (per answers.json)

## Key Technical Decisions

### Architecture
- **Backend**: Go (Echo v4) REST API
- **Frontend**: React + TypeScript + Vite + shadcn/ui
- **Database**: PostgreSQL (metadata), MySQL/PostgreSQL (target databases)
- **Authentication**: OAuth 2.0 (Google) + email/password
- **Authorization**: RBAC (Organization and Project roles)

### Database Schema
- **Strategy**: Fresh schema rebuild from scratch
- **Migration Tool**: golang-migrate
- **Schema Storage**: PostgreSQL with proper indexes and constraints

### Credential Storage
- **Strategy**: AES-256 encryption for database credentials
- **Storage**: Encrypted database column (MVP), AWS Secrets Manager (production)

### Schema Snapshots
- **Strategy**: BYTEA or JSONB column in database
- **PostgreSQL**: pg_dump --schema-only
- **MySQL**: mysqldump --no-data or information_schema queries

### Transaction Isolation
- **Level**: SERIALIZABLE for safety
- **Policy**: All SQL execution wrapped in transactions (no opt-out in MVP)

### SQL Size Limits
- **Maximum SQL Size**: 10MB (configurable)
- **Execution Timeout**: 5 minutes (configurable)

### User Onboarding
- **Strategy**: First user to sign up becomes Organization Admin
- **Subsequent Users**: Assigned Organization Developer role by default
- **Management**: Admins can create and manage users

### Policy Management
- **Strategy**: Policies are separate resources managed by Admins with type field
- **Storage**: Policies stored in policies table with JSONB policy_config and type enum
- **Type System**: Policy type enum (SQL_REVIEW for MVP, extensible for future types)
- **Assignment**: Environments reference Policies via policy_id (nullable FK)
- **Usage**: SQL Review engine loads Policy (SQL_REVIEW type) from Environment when reviewing Change Plans
- **Policy Config**: JSONB field defines which validators are enabled and their settings (for SQL_REVIEW type)

## Success Criteria

### MVP2 Launch Criteria
- All Phase 1 features from docs/mvp.md implemented
- Clear project definitions documented and validated
- Database schema rebuilt and aligned with definitions
- Zero security incidents
- 100% audit log coverage
- Comprehensive test coverage (>85%)
- All linting passing

### Product-Market Fit Indicators
- 5+ users can authenticate and use system
- 10+ database instances managed
- 50+ Change Plans created and executed successfully
- Users prefer Deeb over direct database access
- Clear understanding of project definitions and scope

## Next Steps

1. **Review this guide and the PRD**: Read `.wiz/deeb-mvp2/prd.md` for complete requirements
2. **Familiarize yourself with phase structure**: Review `.wiz/deeb-mvp2/phases/` directory
3. **Start with Phase 1**: Run `/wiz:next` to begin Phase 1, Milestone 1 (P01M01)
4. **Follow milestones sequentially**: Complete milestones in order (P01M01 → P01M02 → ...)
5. **Use milestone acceptance criteria**: Each milestone has specific acceptance criteria to verify completion
6. **Review milestones when complete**: Use `/wiz:review-milestone P01M01` when a milestone is done

## Resources

- **PRD**: `.wiz/deeb-mvp2/prd.md` - Complete product requirements
- **Q&A Session**: `.wiz/deeb-mvp2/intake/qa.md` - Clarifying questions and answers
- **Project Definitions**: `docs/project.md` - Core concept definitions
- **MVP Roadmap**: `docs/mvp.md` - MVP scope and features
- **Phase Documents**: `.wiz/deeb-mvp2/phases/` - Detailed phase breakdowns
- **Existing Codebase**: `backend/` and `frontend/` - Reference implementation

## Milestone Summary

**Total Milestones**: 202 across 6 phases

- **Phase 1**: 37 milestones (P01M01-P01M37)
- **Phase 2**: 31 milestones (P02M01-P02M31)
- **Phase 3**: 40 milestones (P03M01-P03M40)
- **Phase 4**: 38 milestones (P04M01-P04M38)
- **Phase 5**: 25 milestones (P05M01-P05M25)
- **Phase 6**: 35 milestones (P06M01-P06M35)

**Total Estimated Duration**: ~25 days (~202 milestones @ 1h each) within 4-week timeline

---

Generated by Wiz Planner • 2025-11-01
