# Deeb MVP Roadmap

This document outlines the phased development approach for Deeb, focusing on delivering core value incrementally while managing complexity.

## Guiding Principles

- **Start simple, iterate based on feedback**
- **Security is non-negotiable** (even in Phase 1)
- **Working software over comprehensive documentation**
- **Manual processes are acceptable early on** (automate later)

---

## Phase 1: Core MVP (Weeks 1-8)

**Goal**: Deliver the fundamental workflow: Create → Approve → Execute → Audit

### Features

#### 1.1 Authentication & Authorization
- ✓ OAuth 2.0 SSO with **single provider** with Google
- ✓ Basic RBAC with **3 roles**:
   **Organization Roles**:
    - **Organization Admin**: Full system access, manages databases and users
    - **Organization DBA**: Administrator databases across the organization
    - **Organization Developer**: Member of the organization (default role)
   **Project Roles**:
    - **Project Owner**: Full control over project resources
    - **Project Developer**: Create and manage Change Plans
    - **Project Approver**: Approve and close Change Plans
    - **Project Viewer**: View only access to project resources
- ✓ User management UI (admin only)
- ✗ Groups (Phase 2)
- ✗ 2FA (Phase 3)

#### 1.2 Projects & Environments
- ✓ Create and manage Projects
- ✓ Create and manage Environments (Dev, Staging, Prod)
- ✓ Assign databases to Projects and Environments
- ✓ Each database belongs to exactly one Project and one Environment
- ✗ Environment-specific policies (Phase 2)

#### 1.3 Database Management
- ✓ Admin can add/remove database instances (name, host, port, credentials)
- ✓ Support **MySQL and PostgreSQL**
- ✓ Connection via **username/password** authentication
- ✓ Test connection before saving
- ✓ List all registered databases

#### 1.4 Change Plan Workflow
- ✓ Create Change Plan with:
  - Title and description
  - SQL query (DDL or DML)
  - **Single target database** (no multi-DB yet)
  - Change type classification (DDL/DML)
  - Optional rollback SQL script
  - Risk level assignment (Low, Medium, High)
- ✓ Status workflow: Draft → Open → [Approved/Closed] → [Scheduled/InProgress] → Completed/Failed/Cancelled

**Status Definitions**:
- **Draft**: Change Plan is being created/edited, not yet submitted for review
- **Open**: Change Plan is submitted and ready for review/approval
- **Approved**: Change Plan has been approved and is ready for execution
- **Closed**: Change Plan was rejected or closed without execution
- **Scheduled**: Change Plan is approved and scheduled for future execution
- **InProgress**: Change Plan is currently being executed
- **Completed**: Change Plan was successfully executed
- **Failed**: Change Plan execution failed with errors
- **Cancelled**: Change Plan was cancelled before execution

**Status Transition Rules**:
- **Draft → Open**: Author (Project Developer) submits Change Plan for review (automatic SQL Review runs)
- **Open → Draft**: Author can reopen if SQL Review has errors or needs changes (Phase 1: manual reopen)
- **Open → Approved**: Project Approver approves the Change Plan
- **Open → Closed**: Project Approver closes/rejects the Change Plan
- **Approved → Scheduled**: Author or Project Owner schedules execution for future time (Phase 2)
- **Approved → InProgress**: Author, Project Owner, Organization Admin, or Organization DBA triggers immediate execution
- **Scheduled → InProgress**: System automatically transitions when scheduled time arrives (Phase 2)
- **InProgress → Completed**: System automatically transitions on successful execution
- **InProgress → Failed**: System automatically transitions on execution failure (automatic rollback)
- **Approved/Scheduled → Cancelled**: Author, Project Owner, Organization Admin, or Organization DBA cancels before execution
- **Closed/Cancelled**: Terminal states (cannot transition further)

**Permissions**:
- Only Change Plan author can edit Change Plan in Draft status
- Only Project Approvers can approve/close Change Plans in Open status
- Only users with rollout permissions can execute Change Plans (Organization Admin, Organization DBA, Project Owner)
- Authors can reopen Closed Change Plans back to Draft (Phase 1: manual, Phase 2+: workflow-based)

- ✓ **Simple approval flow**: 1 approver required (any Project Approver in the project)
- ✓ Optional reviewer assignment (Phase 1: manual assignment, Phase 2+: automatic assignment based on rules)
- ✓ View Change Plan details
- ✓ List all Change Plans (with filters: draft, open, approved, closed, completed, failed, cancelled)
- ✓ Approve/close with comments
- ✗ Risk-based approval rules (Phase 2)
- ✗ Multiple approvers (Phase 2)
- ✗ Approval assignment/delegation (Phase 2)

#### 1.5 SQL Review
- ✓ SQL Review runs automatically when Change Plan is submitted (Draft → Open transition)
- ✓ **Basic SQL syntax validation** (using database parser libraries)
- ✓ Detect destructive operations (DROP TABLE, DROP DATABASE, TRUNCATE)
- ✓ SQL Review **errors** block execution, **warnings** shown but don't block
- ✓ SQL Review results displayed in Change Plan details page
- ✓ If SQL Review has errors, author can fix and resubmit (Change Plan returns to Open)
- ✓ SQL Review Policies: Basic policies enabled by default (Phase 1: fixed rules, Phase 2: configurable per environment/project)
- ✗ Advanced validations (Phase 2+)

#### 1.6 Change Execution
- ✓ Execute Change Plans with status "Approved" **immediately** (manual trigger by user)
- ✓ **Rollout permissions**: Organization Admin, Organization DBA, and Project Owner can execute rollouts
- ✓ Capture schema snapshot **before** execution (pg_dump for PostgreSQL, mysqldump/information_schema for MySQL)
- ✓ Execute SQL in **transaction** with automatic rollback on error (transaction wrapping cannot be disabled in Phase 1)
- ✓ Capture execution results (success/error, rows affected, duration)
- ✓ Store schema snapshot **after** execution
- ✓ Update Change Plan status (completed, failed)
- ✓ **Concurrency control**: One Change Plan executes at a time per database (other Change Plans queue automatically)
- ✓ **Automatic rollback**: On execution failure, transaction automatically rolls back to previous state
- ✗ Scheduled execution (Phase 2)
- ✗ Multi-database execution (Phase 2)
- ✗ Manual rollback from UI (Phase 3)
- ✗ Transaction opt-out option (Phase 2+)

#### 1.7 Error Handling
- ✓ Log all SQL errors with full context
- ✓ Automatic transaction rollback on failure
- ✓ Display error to user who triggered execution
- ✓ Mark Change Plan as "failed" with error details

#### 1.8 Audit Logging
- ✓ Log all actions:
  - User login/logout
  - Database added/removed
  - Change Plan created/approved/closed/completed
  - Execution success/failure
- ✓ Store: timestamp, user, action, target resource, result
- ✓ View audit log UI (admin only)
- ✓ **Change History**: Detailed migration history and before/after schema snapshots stored for each Change Plan
- ✗ Advanced log filtering (Phase 2)
- ✗ Log export (Phase 2)

#### 1.9 UI Components
- ✓ Dashboard: Overview of open/recent Change Plans
- ✓ Projects & Environments management (admin)
- ✓ Database management page (admin)
- ✓ Create Change Plan form
- ✓ Change Plan detail view
- ✓ User management (admin)
- ✓ Audit log viewer (admin)

### Technical Stack Decisions (Phase 1)
- **Frontend**: React + TypeScript + Vite + shadcn/ui
- **Backend**: Golang
- **Database**: PostgreSQL (for Deeb metadata)
- **Authentication**: OAuth 2.0 with Google
- **Deployment**: Docker + Docker Compose

### Out of Scope for Phase 1
- Groups
- 2FA
- Multiple OAuth providers
- Risk-based approval workflows
- Schema drift detection

- ✗ Multi-database changes
- ✗ Read-only query execution
- ✗ Change history visualization
- ✗ Email/Slack notifications
- ✗ Transaction opt-out option

### Success Metrics
- ✓ 5 users can authenticate via OAuth
- ✓ 10+ database instances managed
- ✓ 50+ Change Plans created and executed
- ✓ Zero security incidents
- ✓ 100% audit log coverage

**Estimated Timeline**: 8 weeks with 2 full-time developers

---

## Phase 2: Enhanced Workflow (Weeks 9-16)

**Goal**: Add flexibility, multi-environment support, and better validation

### Features

#### 2.1 Database Support Enhancements
- ✓ RDS IAM authentication option
- ✓ SSL/TLS connection enforcement
- ✓ Enhanced connection pooling and monitoring

#### 2.2 Advanced Approval Workflows
- ✓ Multiple approvers (e.g., 2 out of 3 required)
- ✓ Risk-based rules based on risk levels (Low, Medium, High):
  - Production databases → 2 approvers
  - Large tables (>1M rows) → Senior engineer approval
  - Destructive operations → Additional confirmation
  - High risk level → Requires DBA approval
- ✓ Custom approval rule configuration UI (admin)

#### 2.3 Multi-Database Changes
- ✓ Apply same change to multiple databases
- ✓ Execution order configuration
- ✓ Rollback strategy: all-or-nothing vs best-effort
- ✓ Partial failure handling

#### 2.4 Scheduled Execution
- ✓ Schedule Change Plan for specific date/time
- ✓ View scheduled Change Plans
- ✓ Cancel scheduled Change Plans (before execution)
- ✓ Timezone handling

#### 2.5 Groups & Permissions
- ✓ Create user groups
- ✓ Assign users to groups
- ✓ Assign permissions to groups
- ✓ Database-level access control (group → database mapping)

#### 2.6 Enhanced Validation
- ✓ **Security validation**: Detect DROP on tables with data
- ✓ **Performance validation**: Missing indexes on new columns, tables without primary keys
- ✓ Configurable validation rules per database/environment
- ✓ Validation override capability (with audit trail)

#### 2.7 Multi-Stage Deployment
- ✓ Multi-stage deployment: Dev → Staging → Production
- ✓ "Promote to next environment" workflow
- ✓ Environment-specific execution

#### 2.8 Notifications
- ✓ Email notifications for:
  - Approval requests
  - Change Plan execution results
  - Scheduled Change Plan reminders
- ✓ In-app notification center
- ✗ Slack/webhook integrations (Phase 3)

#### 2.9 Audit & Reporting
- ✓ Advanced audit log filtering
- ✓ Audit log export (CSV, JSON)
- ✓ Change history timeline view
- ✓ Execution statistics dashboard

### Success Metrics
- ✓ Support for 50+ databases across MySQL and PostgreSQL
- ✓ 200+ Change Plans executed with <5% failure rate
- ✓ Average approval time <1 hour
- ✓ 90% user satisfaction score

**Estimated Timeline**: 8 weeks with 2 full-time developers

---

## Phase 3: Advanced Features (Weeks 17-24)

**Goal**: Production-ready with drift detection, advanced security, and integrations

### Features

#### 3.1 Schema Drift Detection
- ✓ Periodic schema comparison (scheduled job)
- ✓ Detect out-of-band changes
- ✓ Drift notification system
- ✓ Reconciliation workflow (accept drift or revert)

#### 3.2 Two-Factor Authentication
- ✓ TOTP-based 2FA (Google Authenticator, Authy, etc.)
- ✓ Organization-level 2FA enforcement
- ✓ Backup codes
- ✓ 2FA recovery flow

#### 3.3 Advanced Validation
- ✓ **Backwards compatibility**: Detect breaking schema changes
- ✓ **Best practices**: SQL style linting, naming conventions
- ✓ Custom validation rules (regex-based or script-based)

#### 3.4 Read-Only Query Execution
- ✓ Allow SELECT queries without approval
- ✓ Query result export
- ✓ Query timeout enforcement
- ✓ Query result size limits

#### 3.5 Integrations
- ✓ Slack notifications
- ✓ Webhook support for custom integrations
- ✓ API for external tools
- ✓ Terraform provider (infrastructure as code)

#### 3.6 Enhanced Rollback
- ✓ Manual rollback from UI (using stored snapshots)
- ✓ Dry-run rollback (preview changes)
- ✓ Point-in-time schema comparison

#### 3.7 Multiple OAuth Providers
- ✓ Add GitHub OAuth (additional provider)
- ✓ Support multiple providers simultaneously
- ✓ Account linking

#### 3.8 Performance & Scale
- ✓ Background job queue for executions (Redis/BullMQ)
- ✓ Async execution with status polling
- ✓ Concurrent execution management
- ✓ Database connection pooling

### Success Metrics
- ✓ 100+ databases under management
- ✓ 1000+ Change Plans executed per month
- ✓ <1% drift detection false positives
- ✓ 99.9% uptime SLA

**Estimated Timeline**: 8 weeks with 2-3 full-time developers

---

## Phase 4: Enterprise & Polish (Weeks 25+)

**Goal**: Enterprise-ready features and operational excellence

### Features

#### 4.1 Multi-Tenancy
- ✓ Organization isolation
- ✓ Org-level billing and usage tracking
- ✓ Cross-org access controls

#### 4.2 Advanced RBAC
- ✓ Custom roles (beyond Admin/Developer/Approver)
- ✓ Fine-grained permissions
- ✓ Resource-level permissions (database, instance, environment)

#### 4.3 Compliance & Governance
- ✓ SOC 2 compliance features
- ✓ GDPR data handling
- ✓ Audit log immutability guarantees
- ✓ Compliance reporting

#### 4.4 Advanced Monitoring
- ✓ Real-time execution monitoring
- ✓ Database health metrics
- ✓ Performance analytics
- ✓ Cost estimation for changes

#### 4.5 Collaboration Features
- ✓ Comments on Change Plans
- ✓ @mentions and notifications
- ✓ Change Plan templates
- ✓ Approval delegation

#### 4.6 Backup & Disaster Recovery
- ✓ Automated backup of schema snapshots
- ✓ Disaster recovery procedures
- ✓ High availability setup

### Success Metrics
- ✓ Enterprise customer acquisition
- ✓ 99.99% uptime
- ✓ SOC 2 Type II certification

**Estimated Timeline**: Ongoing

---

## Technical Debt & Infrastructure

### Ongoing (All Phases)
- ✓ Comprehensive test coverage (unit, integration, e2e)
- ✓ CI/CD pipeline
- ✓ Documentation (API, user guides, admin guides)
- ✓ Security scanning (SAST, dependency checks)
- ✓ Performance monitoring and optimization
- ✓ Database migrations for Deeb's schema
- ✓ Secrets management (vault, encrypted configs)

---

## Open Questions (To Be Decided Before Phase 1)

1. **Database Credentials Storage**: How to securely store target database credentials?
   - Options: AWS Secrets Manager, HashiCorp Vault, encrypted DB column

2. **Schema Snapshot Storage**: Where to store snapshots?
   - Options: S3, database BYTEA column, separate file storage

3. **Transaction Isolation**: What isolation level for changes?
   - Recommendation: SERIALIZABLE for safety

4. **Change Size Limits**: Max SQL size? Max execution time?
   - Recommendation: 10MB SQL, 5min timeout initially

5. **User Onboarding**: How do first admin get created?
   - Options: Bootstrap script, first OAuth user becomes admin, invite system

6. **Concurrent Changes**: Allow simultaneous changes on different databases?
   - Recommendation: Yes, but lock same-database changes

7. **Read-Only Access**: Should read-only queries be part of Phase 1 or Phase 3?
   - Recommendation: Phase 3 (not critical for MVP)

---

## Dependencies & Requirements

### Infrastructure
- PostgreSQL 14+ (for Deeb metadata)
- Redis (Phase 3+ for job queue)
- S3 or compatible storage (for snapshots)
- OAuth app registration (GitHub/Google)

### External Services
- Email delivery service (SendGrid, AWS SES, etc.) - Phase 2
- Monitoring & alerting (DataDog, Sentry, etc.) - All phases

### Development Tools
- CI/CD platform (GitHub Actions, GitLab CI, etc.)
- Secret management solution
- Container registry (Docker Hub, ECR, etc.)

---

## Risk Mitigation

### High-Risk Areas
1. **Database credential leaks**: Encrypt at rest, strict access controls, audit all access
2. **SQL injection**: Parameterized queries only, no dynamic SQL construction
3. **Privilege escalation**: Principle of least privilege, regular permission audits
4. **Data loss from bad migrations**: Mandatory snapshots, transaction wrapping, approval gates

### Mitigation Strategies
- Security review before each phase completion
- Penetration testing before production launch
- Disaster recovery drills
- Regular dependency updates
- Incident response playbook

---

## Success Criteria for MVP (Phase 1)

**Product-Market Fit Indicators**:
- ✓ 10+ active organizations using Deeb
- ✓ 80%+ approval from early adopters
- ✓ 500+ Change Plans executed successfully
- ✓ Zero security breaches
- ✓ Users prefer Deeb over direct database access

**Go/No-Go for Phase 2**:
- All Phase 1 features complete and stable
- No critical bugs in production
- Positive user feedback
- Clear demand for Phase 2 features
