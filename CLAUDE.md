# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Deeb is a secure database change management tool that enables developers to manage database changes without requiring direct database access. The system provides:

- **Issue-based change workflow**: Database changes are created as issues with multi-approval and review processes
- **Automated policy validation**: Built-in rules for SQL syntax, security, performance, backwards compatibility, and best practices
- **Multi-environment support**: Changes can be applied across Dev -> Staging -> Production environments
- **Schema drift detection**: Automatic detection and notification of out-of-band schema changes
- **Full audit logging**: Complete tracking of all database operations for compliance

### Supported Databases
- MySQL
- PostgreSQL

### Security Features
- OAuth 2.0 SSO (Google, GitHub)
- RBAC with users, groups, and permissions
- Two-factor authentication (2FA) with optional organization-level enforcement
- RDS IAM authentication for database connections
- SSL-enforced database connections
- Data encryption at rest and in transit

## Architecture

The system follows a centralized architecture:

- **Web UI**: Interface for engineers to create and manage database change requests
- **REST API**: Central service that manages change lifecycle and database operations
- **Database Access Layer**: Connects to MySQL/Postgres using RDS IAM Auth with SSL enforcement

Refer to [docs/architecture.png](docs/architecture.png) for visual representation.

## Development Workflow

**ALWAYS follow these steps after making code changes**:

#### After Go Code Changes

- **Run backend tests**: Run relevant tests before committing
- **Run backend linting**: `cd backend && make lint`
- **Run backend autofix**: `cd backend && make lint-fix`
- **Run backend formatting**: `cd backend && make format`

#### After Frontend Code Changes

- **Run frontend tests**: Run relevant tests before committing with `pnpm --dir frontend run test`
- **Run frontend linting**: `pnpm --dir frontend run lint`
- **Run frontend autofix**: `pnpm --dir frontend run lint-fix`
- **Run frontend formatting**: `pnpm --dir frontend run format`

#### Start the Development Servers

- **Start the Frontend**: `pnpm --dir frontend run dev`
- **Start the Backend**: `cd backend && make run`

#### Design Principles

**KISS (Keep It Simple, Stupid)**
- Encourages Claude to write straightforward, uncomplicated solutions
- Avoids over-engineering and unnecessary complexity
- Results in more readable and maintainable code

**YAGNI (You Aren't Gonna Need It)**
- Prevents Claude from adding speculative features
- Focuses on implementing only what's currently needed
- Reduces code bloat and maintenance overhead

**SOLID Principles**
- **Single Responsibility Principle**
- **Open-Closed Principle**
- **Liskov Substitution Principle**
- **Interface Segregation Principle**
- **Dependency Inversion Principle**

## Database Change Lifecycle

1. **Planning**: Define change type (DDL/DML), target database(s), and change details
2. **Review**:
   - Automatic validation (syntax, security, performance, backwards compatibility, best practices)
   - Manual human review
3. **Approval**: Multi-step approval process with risk-based workflows (e.g., production changes require senior engineer approval)
4. **Application**: Changes applied in controlled manner with rollback capability, supports immediate or scheduled execution
5. **Monitoring**: Continuous schema drift detection
