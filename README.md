# Deeb: Secure Database Change Management System

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
- PostgreSQL 13+

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

### Prerequisites

- Go 1.21 or higher
- Node.js 18+ with pnpm
- PostgreSQL 13+ (for metadata database)
- Docker (for containerized development)
- AWS Account with RDS instances

### Development Setup

(Coming soon - see Phase 1 milestones in `.wiz/deeb-mvp/phases/phase1.md`)

## Documentation

- [Architecture Overview](docs/architecture.png)
- [Project Documentation](docs/project.md)
- [PRD - Product Requirements](/.wiz/deeb-mvp/prd.md)
- [Implementation Guide](/.wiz/deeb-mvp/IMPLEMENTATION_GUIDE.md)

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
