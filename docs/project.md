# Deeb: Database Change Management Tool

Deeb is a tool for managing databases changes with a focus on security, auditability, and governance. 💅

## Project Overview

The main goal is to provide a tool for engineers to manage databases changes with **security**, **compliance**, and **improved productivity and reliability**. No need for direct database access by engineers. Everything is audited, changes are policy enforced and the process is fully transparent.

## Architecture

The project is really simple: one Web UI (frontend) and a REST API (backend). Data is stored in a PostgreSQL database. All the SQL queries are executed by the backend with Username and Password or RDS IAM authentication.

The web UI will be used by engineers to manage databases changes. The REST API will act as a central point of truth to manage databases changes.

Instances and databases can be added by admins in the UI.

**Issue's based changes**: the changes are created as Issues like in a traditional issue tracking system with an (muilt-)approval and review process with a set of built-in rules policy analysis and validation to ensure the changes are compliant with the organization's standards. It can have custom approval flows for different risk levels. For example, a change that modifies a production database will require a review from a senior engineer and a DBA. A change that modifies a development database will require a review from a developer.

The application of the changes can be done now and scheduled for a future time. The changes are applied to the database in a controlled way, with a rollback mechanism in case of errors.

**Changes testing/dry-run** can be performed with multi-stage application process `Staging -> Production`.

**Rollback strategy**: Deep will records the detailed migration history and the before/after schema snapshot for each migration it applies. It will use this information to rollback the changes.

**Error handling**: Error while the change is being applied should be logged and audited. The change should be rolled back to the previous state and the error should be reported to the user that applied the change.

Changes are by default wrapped in a transaction. If the change is applied successfully, the transaction is committed. If the change is not applied successfully, the transaction is rolled back. User can opt-out of the transaction by disabling the "Wrap in transaction" option.

Concurrency control: Deeb will apply one change at a time to the database. If a change is already being applied, the new change will be queued and applied after the current change is completed.

Initial database support: 
- MySQL
- PostgreSQL

**Notifications**: Deeb will send notifications to the users when the changes are applied, rolled back, or when an error occurs. Initially, it will support Slack and Webhook integrations.

## Database Changes Definitions and Lifecycle

### Change Types

The database changes can be separated into two categories: 
- **DDL (Data Definition Language)**: changes that modify the database schema.
- **DML (Data Manipulation Language)**: changes that modify data in the database.

### Change Targets

The changes can be applied to different targets:
- single database
- multiple databases

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

Rules can be customized by environment and are created by Admins.

### Lifecycle 3: Approval Process

The approval process is done through the web UI by the approvers. After the review, the approvers can approve or reject the changes. If the changes are rejected, the change is blocked and the approvers are notified. If the changes are approved, the change is (can be) applied to the database.

Some changes can have diffrent approval flows based on the risk level of the changes. For example, a change that modifies a large table in a production database will require a additional approval from a senior engineer.

### Lifecycle 4: Application

The application of the changes is done by the backend through the REST API. The backend will apply the changes to the database in a controlled way, with a rollback mechanism in case of errors.

The application can be done in a multi-step process. For example, it applies one-by-one in this order `Dev -> Staging -> Production`. Also, it can be applied now or scheduled for a future time.

### Lifecycle 5: Monitoring

Automatic schema drift detection and notification. Changes that are made outside of the tool are detected and notified. The drift detection is done by comparing the database schema with the schema defined in the tool.

## Security

### Authentication and Authorization

**Single Sign-On (SSO) with OAuth 2.0**: OAuth 2.0 is an open standard for token-based authentication and authorization which is used to provide Single Sign-On (SSO). Deeb supports popular OAuth 2.0 providers Google and GitHub.

### Users and Groups

Users are the entities that can access and perform actions in Deeb. Groups are collections of users that can share access to resources for easy permission management.

### Roles and Permissions

RBAC (Role-Based Access Control) is used to manage permissions. Each user has a role and a set of permissions. Role can be assigned to a group of users as well.

Roles are organized in two levels: Organization and Project.
- **Organization Roles**:
  - **Organization Admin**: Full system access, manages databases and users
  - **Organization DBA**: Administrator databases across the organization
  - **Organization Developer**: member of the organization (default role)
- **Project Roles**:
  - **Project Owner**: Full control over project resources
  - **Project Developer**: Create and manage change requests
  - **Project Approver**: Approve and reject change requests
  - **Project Viewer**: View only access to project resources

### Two-factor Authentication

Two-factor authentication (2FA) is a security feature that requires users to provide two forms of authentication to access their accounts. Deeb supports popular 2FA providers like Google Authenticator, Authy, 1Password, etc.

2FA can be enforced for all users at the organization level.

### Audit Logs

All actions are logged and audited. The logs are stored in a secure and compliant way.

### Data Encryption

All data is encrypted at rest and in transit.