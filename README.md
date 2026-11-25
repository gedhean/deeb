# Deeb: Secure Database Change Management

[![CI](https://github.com/NSXBet/deeb/actions/workflows/ci.yml/badge.svg)](https://github.com/NSXBet/deeb/actions/workflows/ci.yml)

Deeb enables developers to manage database changes securely without direct database access. A change plan-based workflow with multi-approval processes, automated policy validation, and comprehensive audit logging.

## Key Features

- **Change plan management** - Changes managed as change plans with multi-approval and review
- **Automated validation** - SQL review engine with multiple validators
- **Multi-environment** - Dev → Staging → Production deployment support (Phase 2+)
- **Full audit trail** - Complete tracking of change plan operations
- **Project & Environment isolation** - Each database belongs to one project and environment
- **Self-hosted** - Docker and Kubernetes deployment

## Supported Databases

- MySQL 8.0+
- PostgreSQL 16+

## Security

- OAuth 2.0 SSO (Google, GitHub)
- Two-factor authentication (2FA)
- RBAC with organization and project-level roles
- RDS IAM authentication
- SSL-enforced connections
- Data encryption at rest and in transit

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

## Quick Start

```bash
# Clone the repository
git clone https://github.com/nsxbet/deeb.git
cd deeb

# Setup pnpm global bin directory (first time only)
pnpm setup
source ~/.zshrc  # or restart your terminal

# Copy environment file and configure (first time only)
cp .env.example .env
cp backend/.env.example backend/.env

# Start development server (includes PostgreSQL + live reload)
make dev

# Access
# - Backend API: http://localhost:8080
# - API Docs: http://localhost:8080/api/docs
# - Frontend: http://localhost:3000
```

**See [Getting Started](#getting-started-guide) below for detailed setup.**

## Prerequisites

- Go 1.25.3+
- Node.js 20+ with pnpm (run `pnpm setup` after installing)
- PostgreSQL 16+
- Docker & Docker Compose (recommended)

## Getting Started Guide

### Setup & Configuration
- **[Setup Guide](docs/setup.md)** - Complete installation and configuration
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines and workflow
- [Development Guide](docs/dev-guide.md) - Docker development environment
- [Environment Variables](backend/.env.example) - Configuration options

### API & Frontend
- [API Documentation](http://localhost:8080/api/docs) - OpenAPI/Swagger (when running)
- [Frontend Conventions](docs/frontend-conventions.md) - React & TypeScript patterns
- [shadcn/ui Reference](docs/shadcn-ui-reference.md) - UI component library

### Architecture & Design
- [Architecture Overview](docs/architecture.png) - System architecture diagram
- [Project Documentation](docs/project.md) - Technical design and decisions
- [Database Schema](backend/docs/schema-erd.md) - Database structure and ERD
- [Technologies](docs/technologies.md) - Tech stack overview

### Deployment & Operations
- [RDS/Aurora Auto-Onboarding](docs/deployment/rds-auto-onboarding.md) - Automatically onboard new AWS databases into Deeb (env + project mapping)

### Planning & Roadmap
- [MVP Roadmap](docs/mvp.md) - Phased development approach
- [PRD - Product Requirements](/.wiz/deeb-mvp/prd.md) - Product requirements document

## Development Workflow

**Backend:**
```bash
make -C backend test              # Run tests
make -C backend lint-fix          # Check code quality and fix issues
make -C backend format            # Format code
make -C backend run-air           # Start with live reload
```

**Frontend:**
```bash
pnpm --dir frontend run test       # Run tests
pnpm --dir frontend run lint-fix   # Check code quality and fix issues
pnpm --dir frontend run format     # Format code
pnpm --dir frontend run dev        # Start dev server
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed development workflow.

## Contributing

We welcome contributions! Please read our [Contributing Guide](CONTRIBUTING.md) to get started.

**Quick links:**
- [Development Workflow](CONTRIBUTING.md#development-workflow)
- [Code Standards](CONTRIBUTING.md#code-standards)
- [Testing](CONTRIBUTING.md#testing)
- [Common Tasks](CONTRIBUTING.md#common-tasks)

## License

MIT
