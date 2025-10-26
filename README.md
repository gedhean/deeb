# Deeb: Secure Database Change Management System

[![CI](https://github.com/NSXBet/deeb/actions/workflows/ci.yml/badge.svg)](https://github.com/NSXBet/deeb/actions/workflows/ci.yml)

Deeb is a secure database change management tool that enables developers to manage database changes without requiring direct database access. The system provides an issue-based workflow with multi-approval processes, automated policy validation, and comprehensive audit logging.

## Features

- **Issue-based change workflow**: Database changes are created as issues with multi-approval and review processes
- **Automated policy validation**: Built-in rules for SQL syntax, security, performance, backwards compatibility, and best practices
- **Multi-environment support**: Changes can be applied across Dev → Staging → Production environments
- **Full audit logging**: Complete tracking of all database operations for compliance
- **AWS RDS Integration**: Native support for MySQL and PostgreSQL with RDS IAM authentication
- **Self-hosted deployment**: Containerized deployment with Docker and Kubernetes

## Supported Databases

- MySQL 8.0+
- PostgreSQL 16+

## Security Features

- OAuth 2.0 SSO (Google)
- Email/password authentication with bcrypt
- RBAC with users, groups, and permissions
- RDS IAM authentication for database connections
- SSL-enforced database connections
- Data encryption at rest and in transit
- Comprehensive audit logging

## Project Structure

```
deeb/
├── backend/              # Go backend application
│   ├── cmd/             # Command-line applications
│   │   ├── server/      # Main API server
│   │   └── migrate/     # Database migration tool
│   ├── internal/        # Private application code
│   ├── pkg/            # Public library code
│   ├── api/            # OpenAPI/Swagger specifications
│   └── go.mod          # Go module definition
├── frontend/           # React frontend application (to be created)
├── docs/              # Documentation
└── .wiz/              # Wiz Planner project files (PRD, phases, milestones)
```

### Backend Directory Structure

- **cmd/**: Entry points for executable commands
  - `server/`: Main API server application
  - `migrate/`: Database migration utility
- **internal/**: Private application code (cannot be imported by external projects)
  - Business logic, handlers, repositories, services
- **pkg/**: Public library code (can be imported by external projects)
  - Shared utilities, helpers, common types
- **api/**: API specifications (OpenAPI/Swagger)

## Getting Started

### Quick Start

Get up and running in 5 minutes:

```bash
# 1. Clone and navigate to backend
git clone https://github.com/nsxbet/deeb.git
cd deeb/backend

# 2. Configure environment
cp .env.example .env
# Edit .env and set JWT_SECRET and ENCRYPTION_KEY

# 3. Start development environment with live reload
make run-air

# 4. Access the application
# Backend API: http://localhost:8080
# API Documentation: http://localhost:8080/api/docs
```

### Full Setup Guide

For comprehensive setup instructions including:
- Detailed prerequisites and installation steps
- Environment variable configuration
- Database setup and migrations
- Frontend setup
- Docker development workflow
- Troubleshooting common issues

**See [docs/SETUP.md](docs/SETUP.md) for complete setup documentation.**

### Prerequisites

- **Go 1.25.3+** - Backend development
- **Node.js 20+ with pnpm** - Frontend development
- **PostgreSQL 16+** - Metadata database
- **Docker & Docker Compose** - Containerized development (recommended)

### Development Workflow

**Development mode with live reload:**
```bash
cd backend
make run-air
```

**Run tests:**
```bash
make test              # Unit tests
make test-integration  # Integration tests
make test-coverage     # Coverage report
```

**Code quality:**
```bash
make lint              # Run linters
make lint-fix          # Auto-fix issues
make format            # Format code
```

**Other commands:**
```bash
make help              # Show all available commands
```

## Documentation

### Setup & Configuration
- **[Setup Guide](docs/SETUP.md)** - Complete setup and configuration guide
- [Environment Variables](backend/.env.example) - All configuration options

### Development
- [Development Guide](docs/dev-guide.md) - Docker development environment
- [API Documentation](http://localhost:8080/api/docs) - OpenAPI/Swagger (when running)

### Architecture & Design
- [Architecture Overview](docs/architecture.png) - System architecture diagram
- [Project Documentation](docs/project.md) - Technical design and decisions
- [Database Schema](backend/docs/schema-erd.md) - Database structure and ERD

### Planning & Implementation
- [PRD - Product Requirements](/.wiz/deeb-mvp/prd.md) - Product requirements document
- [Implementation Guide](/.wiz/deeb-mvp/IMPLEMENTATION_GUIDE.md) - Phase and milestone breakdown

## Development Status

This project is under active development following the Wiz Planner methodology:

- **Current Phase**: Phase 1 - Foundation & Project Setup
- **Total Phases**: 6 phases
- **Total Milestones**: 210 milestones (~22 days of work)

See [IMPLEMENTATION_GUIDE.md](/.wiz/deeb-mvp/IMPLEMENTATION_GUIDE.md) for detailed phase and milestone breakdown.

## License

(To be determined)

## Contributing

(Coming soon)
