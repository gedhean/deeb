# Implementation Guide: Deeb MVP

**Project**: Deeb MVP - Secure Database Change Management System
**Total Duration**: ~22 days (210 milestones @ 1 hour each)
**Phases**: 6 phases from Foundation to Deployment
**Last Updated**: 2025-10-22

---

## Table of Contents

1. [Overview](#overview)
2. [How to Use This Guide](#how-to-use-this-guide)
3. [Phase Roadmap](#phase-roadmap)
4. [NFR Priority Order](#nfr-priority-order)
5. [Testing Strategy](#testing-strategy)
6. [Key Technical Decisions](#key-technical-decisions)
7. [Success Metrics](#success-metrics)
8. [Getting Started](#getting-started)

---

## Overview

This implementation guide provides a comprehensive roadmap for building the Deeb MVP from scratch. The project is organized into 6 sequential phases, with each phase containing detailed milestones that represent ~1 hour of focused work.

### Project Goals

Deeb eliminates unsafe direct database access by providing:
- **Issue-based workflow**: Database changes as issues with multi-approval processes
- **Automated validation**: SQL syntax, security, performance, backwards compatibility, best practices
- **Multi-environment support**: Staging → Production promotion workflow
- **Comprehensive audit logging**: Full tracking for compliance
- **AWS RDS integration**: Native support for MySQL and PostgreSQL

### Target Users

- **Backend Engineers (Developers)**: Submit database change requests
- **Site Reliability Engineers (SREs)**: Review and approve changes, ensure system reliability

### Success Criteria

Zero manual database changes with direct access to databases.

---

## How to Use This Guide

### Workflow

1. **Read the PRD** (`.wiz/deeb-mvp/prd.md`)
   - Understand functional requirements (FR1-FR8)
   - Review NFRs organized by P0-P4 priority
   - Study architecture and technology decisions

2. **Review Phase Goals** (this guide)
   - Understand phase dependencies
   - Identify blocking work and parallel opportunities

3. **Execute Milestones Sequentially**
   - Navigate to phase files (e.g., `.wiz/deeb-mvp/phases/phase1.md`)
   - Work through milestones in order (P01M01, P01M02, etc.)
   - Check off acceptance criteria as you complete them
   - Mark milestone status: 🚧 TODO → 🔄 IN_PROGRESS → ✅ DONE

4. **Verify Phase Completion**
   - Each phase ends with a verification milestone
   - Ensure all acceptance criteria are met before moving to next phase
   - Run tests and validate code quality

5. **Track Progress**
   - Use `/wiz:status` to see overall progress
   - Use `/wiz:next` to find and execute next TODO milestone
   - Use `/wiz:review-phase <slug> <phase-number>` to review completed phase

### Milestone Structure

Each milestone includes:
- **ID**: Unique identifier (e.g., P01M01)
- **Status**: 🚧 TODO, 🔄 IN_PROGRESS, or ✅ DONE
- **Goal**: Clear objective describing what needs to be accomplished
- **Acceptance Criteria**: 3-6 specific, testable requirements

### Dependencies

Phases must be completed sequentially:
- Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6

Within phases, milestones should generally be completed in order, though some may be parallelizable (e.g., independent unit test creation).

---

## Phase Roadmap

### Phase 1: Foundation & Project Setup
**Duration**: ~5 days (40 milestones)
**Dependencies**: None
**File**: `.wiz/deeb-mvp/phases/phase1.md`

**Goal**: Establish project foundation with Go backend structure, React frontend scaffold, core database schema, basic authentication, and CI/CD pipeline.

**Key Deliverables**:
- Go project with standard layout (cmd/, internal/, pkg/, api/)
- React + Vite frontend with Shadcn/ui components
- PostgreSQL metadata schema (users, issues, approvals, audit logs)
- JWT-based authentication with bcrypt password hashing
- GitHub Actions CI/CD pipeline
- Docker development environment

**Milestone Breakdown**:
- M01-M08: Go backend structure and core setup
- M09-M16: React frontend scaffold with routing and layout
- M17-M24: Database schema and migrations
- M25-M32: Authentication and authorization (JWT, RBAC)
- M33-M40: Development infrastructure and CI/CD

**Critical Path**: Database schema (M17-M24) blocks backend service development

---

### Phase 2: Core Backend Services
**Duration**: ~5 days (38 milestones)
**Dependencies**: Phase 1
**File**: `.wiz/deeb-mvp/phases/phase2.md`

**Goal**: Implement core business logic including issue management system, comprehensive SQL validation engine with multiple validators, and approval workflow system.

**Key Deliverables**:
- Issue CRUD operations with status management
- SQL validation engine with 5 validators:
  - Syntax validator (MySQL/PostgreSQL AST parsing)
  - Security validator (injection prevention, dangerous operations)
  - Performance validator (missing indexes, full table scans, Cartesian joins)
  - Backwards compatibility validator (breaking changes detection)
  - Best practices validator (naming conventions, data types, constraints)
- Approval workflow (single and multi-stage)
- Notification system for workflow events
- Comprehensive REST API with OpenAPI documentation

**Milestone Breakdown**:
- M01-M10: Issue management system
- M11-M25: SQL validation engine (5 validators)
- M26-M32: Approval workflow
- M33-M38: API documentation and testing

**Critical Path**: SQL validation engine (M11-M25) is the most complex component

**NFR Highlights**:
- P0: Syntax and security validation prevent dangerous operations
- P1: Unit tests for all validators (>85% coverage)
- P3: golangci-lint passing for code quality

---

### Phase 3: Database Integration & Change Execution
**Duration**: ~4 days (35 milestones)
**Dependencies**: Phase 2
**File**: `.wiz/deeb-mvp/phases/phase3.md`

**Goal**: Implement database access layer with RDS IAM authentication, connection pooling, SQL execution engine with transaction support, and multi-stage promotion workflow.

**Key Deliverables**:
- RDS connection manager (IAM auth + traditional credentials)
- Connection pooling (5-20 connections per database)
- SSL/TLS enforcement for all database connections
- SQL execution engine with transaction support
- Dry-run mode for previewing changes
- Transaction-level rollback capability
- Multi-stage promotion (Staging → Production)
- Scheduled execution
- Execution audit logging

**Milestone Breakdown**:
- M01-M08: Database connection management and authentication
- M09-M13: SQL execution engine with transaction support
- M14-M20: Execution features (locking, retry, rollback)
- M21-M28: Testing and monitoring
- M29-M35: Performance optimization and documentation

**Critical Path**: Connection pooling and execution engine (M04-M13) are foundational

**NFR Highlights**:
- P0: Transaction-level rollback ensures all-or-nothing execution
- P2: SSL/TLS enforcement and credential encryption (AES-256)
- P4: Connection pooling (5-20 connections) for performance

---

### Phase 4: Frontend Implementation
**Duration**: ~5 days (40 milestones)
**Dependencies**: Phase 3
**File**: `.wiz/deeb-mvp/phases/phase4.md`

**Goal**: Build complete React frontend with issue creation/management, Monaco SQL editor, approval dashboard, audit log viewer, and responsive design.

**Key Deliverables**:
- Issue creation wizard with database/environment selection
- Monaco Editor integration for SQL editing
- Approval dashboard with filtering and actions
- Audit log viewer with search and filtering
- Change history timeline
- Responsive design for mobile/tablet
- Accessibility compliance (WCAG 2.1 AA)
- Loading states, error handling, empty states

**Milestone Breakdown**:
- M01-M10: Issue management UI
- M11-M18: SQL editor and validation display
- M19-M26: Approval workflow UI
- M27-M34: Audit and monitoring UI
- M35-M40: UX polish and accessibility

**Critical Path**: Monaco Editor integration (M07) and API integration (M04-M06) are essential

**NFR Highlights**:
- P3: Accessibility (WCAG 2.1 AA), ESLint/Prettier passing
- P4: React.lazy for code splitting, responsive design

---

### Phase 5: Testing, Security & Quality Assurance
**Duration**: ~4 days (32 milestones)
**Dependencies**: Phase 4
**File**: `.wiz/deeb-mvp/phases/phase5.md`

**Goal**: Achieve comprehensive test coverage with integration and e2e tests, implement security hardening, add observability features, and complete documentation.

**Key Deliverables**:
- Integration tests for all REST API endpoints
- End-to-end tests for critical workflows
- Fuzz testing for SQL parsers, user inputs, authentication tokens
- OWASP Top 10 security audit
- Security validations (rate limiting, CSRF, XSS, password encryption, session management)
- Prometheus metrics endpoint
- Structured logging with correlation IDs
- Performance testing (100 concurrent users, P95 <500ms)
- Comprehensive documentation (user guide, API docs, troubleshooting)

**Milestone Breakdown**:
- M01-M09: Integration, e2e, and fuzz testing setup
- M10-M17: Security audit and validation
- M18-M21: Observability (metrics, logging, tracing)
- M22-M24: Performance testing
- M25-M31: Documentation and code quality
- M32: Phase verification

**Critical Path**: Integration tests (M01-M06) validate entire system

**NFR Highlights**:
- P0: All critical workflows pass e2e tests
- P1: Backend coverage >80%, frontend >70%
- P2: OWASP Top 10 audit complete, all security validations passing
- P4: P95 latency <500ms under 100 concurrent users

---

### Phase 6: Deployment & Release
**Duration**: ~3 days (25 milestones)
**Dependencies**: Phase 5
**File**: `.wiz/deeb-mvp/phases/phase6.md`

**Goal**: Package application for production deployment with Docker containers, create Kubernetes manifests, establish CI/CD pipeline, and prepare for MVP release.

**Key Deliverables**:
- Multi-stage Dockerfiles (Go backend, React frontend with nginx)
- Docker Compose configurations (development and production)
- Docker images published to registry
- Kubernetes manifests (deployments, services, ingress, configmaps, secrets)
- Health check probes for zero-downtime deployments
- CI/CD pipeline (automated testing, Docker building, image publishing)
- Database migration strategy for Kubernetes
- Monitoring and logging setup guides
- Operational documentation (backup procedures, rollback, incident response)
- Release notes for MVP

**Milestone Breakdown**:
- M01-M05: Docker containerization
- M06-M13: Kubernetes deployment manifests
- M14-M16: CI/CD pipeline and health checks
- M17-M23: Operational procedures and documentation
- M24-M25: Release preparation and verification

**Critical Path**: Docker images (M01-M05) block Kubernetes deployment

**NFR Highlights**:
- Zero-downtime deployment verified with health probes
- Automated CI/CD pipeline operational
- All operational runbooks complete

---

## NFR Priority Order

Deeb follows a strict NFR priority framework (P0 → P4):

### P0: Correctness
**Priority**: Critical - System must be correct above all else

- Prevent SQL injection attacks through validation engine
- Ensure transaction integrity (all-or-nothing execution)
- Validate SQL syntax before execution
- Prevent breaking changes without explicit approval

**Implementation**: Built into Phase 2 (SQL validators) and Phase 3 (transaction support)

### P1: Regression Prevention (Testing)
**Priority**: Essential - Prevent bugs from reaching production

- Integration tests for all REST API endpoints
- End-to-end tests for critical workflows
- Fuzz testing for SQL parsers and user inputs
- Backend coverage >80%, frontend >70%

**Implementation**: Phase 5 (M01-M09) focuses entirely on testing

### P2: Security
**Priority**: High - Protect sensitive data and prevent unauthorized access

- Encrypt passwords with bcrypt (work factor ≥12)
- Encrypt database credentials at rest (AES-256)
- Enforce SSL/TLS for all database connections
- Implement rate limiting (10 login attempts per 5 minutes)
- CSRF, XSS, session management protections
- OWASP Top 10 audit

**Implementation**: Phase 1 (authentication), Phase 3 (connection security), Phase 5 (security audit)

### P3: Quality
**Priority**: Medium - Maintain code quality and accessibility

- golangci-lint passing for Go code
- ESLint and Prettier passing for JavaScript
- WCAG 2.1 AA accessibility compliance
- Comprehensive documentation

**Implementation**: Phase 5 (M27-M31) covers code quality and documentation

### P4: Performance
**Priority**: Lower - Optimize after correctness, testing, security, quality

- Support 100 concurrent users
- P95 API latency <500ms
- Connection pooling (5-20 connections)
- Response time <2s for SQL validation

**Implementation**: Phase 3 (connection pooling), Phase 5 (M22-M24 performance testing)

**Key Principle**: Never sacrifice higher priority NFRs for lower ones. Correctness trumps performance every time.

---

## Testing Strategy

### Focus Areas

**Integration Tests** (Primary Focus):
- All REST API endpoints (authentication, issues, approvals, execution)
- Database operations (connection, pooling, execution)
- SQL validation engine with real SQL samples
- Approval workflow state transitions

**End-to-End Tests** (Critical Workflows):
- Complete change lifecycle: create issue → approve → apply change
- Multi-stage promotion: Staging → Production
- Authentication flows: email/password and OAuth

**Fuzz Testing** (Core Areas Only):
- SQL parsers (MySQL and PostgreSQL)
- User input handling (issue creation, SQL statements)
- Authentication token processing (JWT parsing and validation)

**Unit Tests** (Supporting):
- Individual validators (syntax, security, performance, compatibility, best practices)
- Business logic functions
- Utility functions

### Coverage Targets

- Backend (Go): >80% overall, >85% for critical packages (validators, execution engine)
- Frontend (React): >70% overall
- Critical paths: 100% coverage

### Testing Tools

- **Go**: `testing` package, `testify` for assertions
- **Frontend**: Jest, React Testing Library
- **E2E**: Playwright or Cypress
- **Fuzz**: `go-fuzz` or native Go fuzzing
- **Load**: k6 or JMeter

### Benchmarking Policy

**Skipped for MVP** - No benchmark tests required. Focus on functional correctness and meeting performance NFRs through load testing instead.

---

## Key Technical Decisions

### Languages and Frameworks

**Backend**: Go 1.21+
- **Web Framework**: `net/http` (standard library)
- **CLI**: `spf13/cobra` for server commands
- **Logging**: `go.uber.org/zap` for structured logging
- **Database**: `lib/pq` (PostgreSQL), `go-sql-driver/mysql`
- **Rationale**: Go's performance, concurrency model, and strong typing fit database operations well

**Frontend**: JavaScript with React 18+
- **Build Tool**: Vite for fast development
- **UI Library**: Shadcn/ui (https://ui.shadcn.com) with Tailwind CSS
- **Code Editor**: Monaco Editor for SQL editing
- **Rationale**: React ecosystem maturity, Vite speed, Shadcn flexibility

### Database Architecture

**Metadata Database**: PostgreSQL 13+ (self-hosted)
- Stores issues, users, approvals, audit logs, execution results
- Why: Strong ACID guarantees, JSON support, mature ecosystem

**Target Databases**: AWS RDS MySQL 8.0+ and PostgreSQL 13+
- Dual authentication: RDS IAM auth (preferred) and traditional credentials (fallback)
- SSL/TLS enforcement mandatory
- Connection pooling (5-20 connections per database)

### Authentication Strategy

**Dual Authentication**:
1. Email/Password (bcrypt hashing, work factor ≥12)
2. OAuth 2.0 with Google only

**Why Dual**: Supports both internal users (email/password) and enterprise SSO scenarios (OAuth)

**Session Management**: JWT tokens with HTTP-only, Secure, SameSite cookies (24h expiration)

### SQL Validation Approach

**Multi-Validator Pipeline**:
1. **Syntax Validator**: Parse SQL using AST parsers (separate for MySQL/PostgreSQL)
2. **Security Validator**: Detect dangerous operations (DROP DATABASE, TRUNCATE without WHERE, SQL injection patterns)
3. **Performance Validator**: Identify missing indexes, full table scans, Cartesian joins
4. **Backwards Compatibility Validator**: Flag breaking changes (DROP COLUMN, RENAME TABLE)
5. **Best Practices Validator**: Enforce naming conventions, proper data types, foreign key constraints

**Rationale**: Layered validation catches issues early in the workflow, reducing production incidents.

### Deployment Model

**Self-Hosted Containerized Deployment**:
- Docker containers for backend and frontend
- Kubernetes manifests for production orchestration
- Docker Compose for local development
- CI/CD via GitHub Actions

**Why Self-Hosted**: Enterprises prefer on-premises control for database security tools.

### AWS Integration

**AWS-Only Focus (MVP)**:
- RDS IAM authentication for MySQL and PostgreSQL
- Potential future: RDS automated backups verification via AWS API

**Why AWS-Only**: Simplifies MVP scope while covering majority of enterprise cloud deployments.

---

## Success Metrics

### Primary Metrics

1. **Zero Direct Database Access**
   - **Target**: 0 manual database changes outside Deeb
   - **Measurement**: Audit all database changes for 30 days post-launch

2. **Change Throughput**
   - **Target**: 50 changes per day with 100 concurrent users
   - **Measurement**: Monitor issue creation and application rates

3. **System Reliability**
   - **Target**: P95 API latency <500ms
   - **Measurement**: Prometheus metrics tracking

4. **Validation Effectiveness**
   - **Target**: >95% of dangerous operations caught by validators
   - **Measurement**: Track validation failures vs. actual incidents

### Secondary Metrics

5. **Approval Time**
   - **Target**: <2 hours median time from submission to approval
   - **Measurement**: Track issue state timestamps

6. **Test Coverage**
   - **Target**: Backend >80%, Frontend >70%
   - **Measurement**: Coverage reports in CI/CD

7. **Security Compliance**
   - **Target**: Pass OWASP Top 10 audit with no critical issues
   - **Measurement**: Phase 5 security audit results

8. **User Adoption**
   - **Target**: 80% of target users (backend engineers + SREs) actively using Deeb within 60 days
   - **Measurement**: Active user count and issue creation rate

---

## Getting Started

### Prerequisites

- **Go**: 1.21 or higher
- **Node.js**: 18+ with npm/yarn
- **PostgreSQL**: 13+ for metadata database
- **Docker**: For containerized development
- **AWS Account**: With RDS instances (MySQL and/or PostgreSQL) for testing
- **GitHub Account**: For CI/CD and OAuth integration

### Initial Setup

1. **Clone Repository Structure**
   ```bash
   # Follow Phase 1, Milestone 1 (P01M01)
   mkdir -p cmd/server cmd/migrate internal pkg api
   go mod init github.com/yourusername/deeb
   ```

2. **Read the PRD**
   - Open `.wiz/deeb-mvp/prd.md`
   - Understand all functional requirements (FR1-FR8)
   - Review NFRs and acceptance criteria

3. **Start Phase 1**
   - Open `.wiz/deeb-mvp/phases/phase1.md`
   - Begin with P01M01: Initialize Go project with standard layout
   - Work through milestones sequentially
   - Check off acceptance criteria as you complete them

4. **Set Up Development Environment**
   - Follow P01M33-M34 for Docker setup
   - Configure PostgreSQL metadata database (P01M17-M24)
   - Set up CI/CD pipeline (P01M35-M37)

5. **Track Progress**
   - Use `/wiz:status` to see overall progress
   - Use `/wiz:next` to find next TODO milestone
   - Mark milestones as complete: 🚧 TODO → ✅ DONE

### Development Workflow

1. **Work in milestone chunks**: Each milestone is ~1 hour of focused work
2. **Test as you go**: Write unit tests alongside implementation
3. **Commit frequently**: Small, atomic commits with descriptive messages
4. **Review acceptance criteria**: Don't mark a milestone complete until all criteria are met
5. **Verify phase completion**: Run the phase verification milestone before moving to the next phase

### Support and Resources

- **PRD**: `.wiz/deeb-mvp/prd.md` - Complete product requirements
- **Phase Files**: `.wiz/deeb-mvp/phases/phase*.md` - Detailed milestone breakdowns
- **Technology Docs**:
  - Go: https://go.dev/doc/
  - React: https://react.dev/
  - Vite: https://vitejs.dev/
  - Shadcn/ui: https://ui.shadcn.com
  - Cobra: https://github.com/spf13/cobra
  - Monaco Editor: https://microsoft.github.io/monaco-editor/

---

## Next Steps

1. **Start Phase 1**: Open `.wiz/deeb-mvp/phases/phase1.md` and begin with P01M01
2. **Set Up Environment**: Configure Go, Node.js, PostgreSQL, Docker
3. **Create Repositories**: Initialize Go module and React project
4. **Follow the Roadmap**: Work through phases sequentially
5. **Maintain Quality**: Follow NFR priority order (P0 → P4)
6. **Track Progress**: Use `/wiz:status` and `/wiz:next` commands

**Ready to start building? Open Phase 1 and begin with the first milestone!**

---

*Generated by Wiz Planner on 2025-10-22*
