# Deeb: Database Change Management Tool

Deeb is a tool for managing databases changes with a focus on security, auditability, and governance. 💅

## Project Overview

The main goal is to provide a tool for engineers to manage databases changes with **security**, **compliance**, and **improved productivity and reliability**. No need for direct database access by engineers. Everything is audited, changes are policy enforced and the process is fully transparent.

## Architecture

The project is really simple: one Web UI (frontend) and a REST API (backend). Data is stored in a PostgreSQL database. All the SQL queries are executed by the backend with Username and Password or RDS IAM authentication.

The web UI will be used by engineers to manage databases changes. The REST API will act as a central point of truth to manage databases changes.

Instances and databases can be added by admins in the UI.

## MVP Scope (Phase 1)

For the initial MVP, the following features are prioritized:
- Single database changes (not multi-database)
- Simple approval flow (1 approver required)
- Basic SQL validation (syntax + destructive operations)
- Immediate execution only (scheduled execution in Phase 2)
- Single database target (multi-stage deployment in Phase 2)
- Basic Review Policies (fixed rules, configurable policies in Phase 2)

See [MVP Roadmap](mvp.md) for detailed phase breakdown.

**Change Plan based changes**: the changes are created as Change Plans like in a traditional issue tracking system with an (multi-)approval and review process with a set of built-in rules policy analysis and validation to ensure the changes are compliant with the organization's standards. It can have custom approval flows for different risk levels. For example, a change that modifies a production database will require a review from a senior engineer and a DBA. A change that modifies a development database will require a review from a developer.

The application of the changes can be done now and scheduled for a future time (Phase 2). The changes are applied to the database in a controlled way, with a rollback mechanism in case of errors.

**Changes testing/dry-run** can be performed with multi-stage application process `Staging -> Production` (Phase 2+).

**Rollback strategy**: Deeb records the detailed migration history and the before/after schema snapshot for each migration it applies. Schema snapshots are captured using pg_dump for PostgreSQL and mysqldump/information_schema for MySQL. It will use this information to rollback the changes (manual rollback UI in Phase 3+).

**Error handling**: Error while the change is being applied should be logged and audited. The change should be rolled back to the previous state and the error should be reported to the user that applied the change.

Changes are by default wrapped in a transaction. If the change is applied successfully, the transaction is committed. If the change is not applied successfully, the transaction is rolled back. User can opt-out of the transaction by disabling the "Wrap in transaction" option (Phase 2+).

Concurrency control: Deeb will apply one change at a time per database. If a change is already being applied, the new change will be queued and applied after the current change is completed. Concurrent changes on different databases are allowed.

Initial database support: 
- MySQL
- PostgreSQL

**Notifications**: Deeb will send notifications to the users when the changes are applied, rolled back, or when an error occurs. Initially, it will support Slack and Webhook integrations (Phase 2+).

## Database Changes Definitions and Lifecycle

### Change Types

The database changes can be separated into two categories: 
- **DDL (Data Definition Language)**: changes that modify the database schema.
- **DML (Data Manipulation Language)**: changes that modify data in the database.

### Change Targets

The changes can be applied to different targets:
- single database (MVP)
- multiple databases (Phase 2+)

### Lifecycle 1: Planning the Changes

In this phase, we define the change type, target database(s) and the details of the change.

### Lifecycle 2: Review 

The review is process is done in two steps:
- **Automatic Review**: run a set of built-in rules analysis and validation to ensure the changes are compliant with the organization's policies.
- **Manual Review**: manual review by a human reviewer.

The automatic review is done by running a set of built-in rules analysis and validation to ensure the changes are compliant with the organization's policies. It includes:
- **SQL syntax validation**: check for syntax errors and warnings.
- **Security validation**: drop tables with data, drop columns with data.
- **Performance validation**: missing indexes, no primary key.
- **Backwards compatibility validation**: ensure the change won't break existing functionality.
- **Bests practices validation**: follow industry standards and best practices.

Rules can be customized by environment and are created by Admins (Phase 2: configurable policies, Phase 1: fixed rules).

### Lifecycle 3: Approval Process

The approval process is done through the web UI by the approvers. After the review, the approvers can approve or close the changes. If the changes are closed/rejected, the change is blocked and the approvers are notified. If the changes are approved AND there is no SQL Review errors, the change is (can be) applied to the database.

Some changes can have different approval flows based on the risk level of the changes (Phase 2+). For example, a change that modifies a large table in a production database will require an additional approval from a senior engineer.

### Lifecycle 4: Application

The application of the changes is done by the backend through the REST API. The backend will apply the changes to the database in a controlled way, with a rollback mechanism in case of errors.

The application can be done in a multi-step process (Phase 2+). For example, it applies one-by-one in this order `Dev -> Staging -> Production`. Also, it can be applied now or scheduled for a future time (scheduled execution in Phase 2).

### Lifecycle 5: Monitoring

Automatic schema drift detection and notification (Phase 3). Changes that are made outside of the tool are detected and notified. The drift detection is done by comparing the database schema with the schema defined in the tool.

## Security

### Authentication and Authorization

**Single Sign-On (SSO) with OAuth 2.0**: OAuth 2.0 is an open standard for token-based authentication and authorization which is used to provide Single Sign-On (SSO). Deeb supports popular OAuth 2.0 providers Google (MVP) and GitHub (Phase 3+).

### Users and Groups

Users are the entities that can access and perform actions in Deeb. Groups are collections of users that can share access to resources for easy permission management (Phase 2+).

### Projects

A `Project` groups databases, team members, and workflows together. It's a logical grouping of resources that are related to a specific application or project.

### Roles and Permissions

RBAC (Role-Based Access Control) is used to manage permissions. Each user has a role and a set of permissions. Role can be assigned to a group of users as well (group assignment in Phase 2+).

Roles are organized in two levels: Organization and Project.
- **Organization Roles**:
  - **Organization Admin**: Full system access, manages databases and users
  - **Organization DBA**: Administrator databases across the organization
  - **Organization Developer**: Member of the organization (default role)
- **Project Roles**:
  - **Project Owner**: Full control over project resources
  - **Project Developer**: Create and manage Change Plans
  - **Project Approver**: Approve and close Change Plans
  - **Project Viewer**: View only access to project resources

### Two-factor Authentication

Two-factor authentication (2FA) is a security feature that requires users to provide two forms of authentication to access their accounts. Deeb supports popular 2FA providers like Google Authenticator, Authy, 1Password, etc. (Phase 3+)

2FA can be enforced for all users at the organization level.

### Audit Logs

All actions are logged and audited. The logs are stored in a secure and compliant way.

### Data Encryption

All data is encrypted at rest and in transit.


## Technical Stack

- **Frontend**: React + TypeScript + Vite + shadcn/ui
- **Backend**: Go (Golang)
- **Database**: PostgreSQL (for Deeb metadata)
- **Authentication**: OAuth 2.0 with Google (MVP), GitHub (Phase 3+)
- **Deployment**: Docker + Docker Compose

See [MVP Roadmap](mvp.md) for detailed technical decisions and [Technologies](technologies.md) for technology references.

## Core Concepts: Organization Resources

### Projects

A `Project` groups databases, team members, and workflows together. It's a logical grouping of resources that are related to a specific application or project.

### Environments

An `Environment` represents a specific stage in your development and deployment pipeline such as Development, Staging, Production. These typically correspond directly to your actual deployment environments. Organization administrators can attach organizational policies to environments, enabling different rules and approval workflows for each stage (environment-specific policies in Phase 2+).

Each database belongs to exactly one environment.

### Database Instance
A `Database Instance` represents a database server or cluster that you connect to using a host address and port. This includes MySQL servers, PostgreSQL clusters, AWS RDS instances, and similar database systems. A single instance can host multiple individual databases. Organization administrators and DBAs are typically responsible for managing database instances.

### Databases

A `Database` is an individual database residing within a database instance. Each database belongs to exactly one project, establishing clear ownership and access boundaries. Project developers primarily interact with databases in their day-to-day work.

### Change Plan

A `Change Plan` defines a set of schema changes to apply to the database. It is created by a user and can be assigned to a reviewer (a different user). The reviewer can approve or close the change. If the change is approved, the change is applied to the database. If the change is closed/rejected, the change is blocked and the user is notified.

Change plans can be created by users with the `Project Developer` role.

A change plan can have more than one `Stage`. Where each stage is a different environment. For example, a change plan can be applied to a database in a multi-step process (Phase 2+). For example, it applies one-by-one in this order `Dev -> Staging -> Production`.

Change plans can be created for different targets:
- single database (MVP)
- multiple databases (one in each stage) (Phase 2+)

Change plans can be created for different types of changes:
- DDL (Data Definition Language)
- DML (Data Manipulation Language)

### SQL Review

SQL Review automatically validates your database changes against configured policies before they’re executed. This ensures consistency and catches potential issues early in the change process.

When a Change Plan is submitted for review (Draft → Open transition), the SQL Review is automatically run against the SQL content of the change plan and flags Policy violations.

SQL Review results are shown in the Change Plan details page:
- **Errors**: Critical issues that must be fixed before the change can be applied (blocks the change).
- **Warnings**: Issues that should be reviewed but don't block the change.
- **Pass**: The change is compliant with the policies.

### Review Policy

Review policies determine which rules are enforced for your database changes. You can configure these policies at the environment or project level (Phase 2: configurable policies, Phase 1: fixed rules).

Review policies can be configured by Admins (Phase 2+).

### Approval Process 

The approval process is done through the web UI by the approvers. After the review, the approvers can approve or close the changes. If the changes are closed/rejected, the change is blocked and the approvers are notified. If the changes are approved AND there is no SQL Review errors, the change is (can be) applied to the database.

Process of approval:
1. **Risk**: When a Change Plan is created, the risk level is assigned (Low, Medium, High). Risk-based approval rules are configured in Phase 2+.
2. **Flow**: The approval flow is determined based on the Risk Level (Phase 2: configurable flows, Phase 1: simple 1-approver flow).
3. **Approvers**: The approvers review and approve each stage of the change plan (Phase 2: multi-stage approval).
4. **Rollout**: After approval and no SQL Review errors, the change is applied to the database.

**Risk Levels** defines the severity of the change and can be configured based on:
- Operation type
- Target database
- Target environment
- Custom rules

The **Risk Levels** are:
- **Low**: Low risk changes that can be applied without a review.
- **Medium**: Medium risk changes that require a review from a reviewer.
- **High**: High risk changes that require a review from a senior engineer.

**Approval Flows** are defined based on the Risk Level and can be configured by Admins (Phase 2+).

**Approval Flows** are defined as:
- **Single Approval**: One approval is required for the change (MVP).
- **Multi-stage Approval**: Multiple approvals are required for the change (Phase 2+).
- **Specific Role Approval**: A specific role is required to approve the change (Phase 2+).

### Rollout

A Rollout is a process of applying a change plan to the database. If the change plan is approved and there is no SQL Review errors, the change is applied to the database.

Deeb provides transaction management to ensure safe database changes. By default, all SQL statements are wrapped in a transaction, allowing automatic rollback if errors occur during execution.

Roles that can run the rollout are by default:
- **Organization Admin**: Full system access, can run the rollout.
- **Organization DBA**: Administrator databases across the organization, can run the rollout.
- **Project Owner**: Full control over project resources, can run the rollout.

Also, it can be configured to allow other roles to run the rollout based on the environment or project (Phase 2+).

Rollout can be applied now or scheduled for a future time (scheduled execution in Phase 2).

### Change History

Deeb records the detailed migration history and the before/after schema snapshot for each migration it applies. Schema snapshots are stored securely and can be used for rollback and schema drift detection. It also leverages these records to detect schema drifts (Phase 3).

### Schema Rollback

Deeb provides a rollback mechanism to revert the changes to the previous state. It uses the detailed migration history and the before/after schema snapshot for each migration to revert the changes. Manual rollback from UI is available in Phase 3+. Phase 1 provides automatic rollback on execution failure via transaction rollback.
