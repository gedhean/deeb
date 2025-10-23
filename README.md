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

#### 1. Clone the repository

```bash
git clone https://github.com/nsxbet/deeb.git
cd deeb
```

#### 2. Set up the metadata database

Create a PostgreSQL database for Deeb's metadata:

```bash
createdb deeb_metadata
```

Set the database connection string:

```bash
export DATABASE_URL="postgres://username:password@localhost:5432/deeb_metadata?sslmode=disable"
```

#### 3. Run database migrations

Navigate to the backend directory and run migrations:

```bash
cd backend
go run cmd/migrate/main.go -command up
```

**Migration Commands:**

- `up` - Apply all pending migrations
- `down` - Rollback all migrations
- `version` - Show current migration version
- `force <version>` - Force database to specific version (use with caution)

**Examples:**

```bash
# Apply migrations
go run cmd/migrate/main.go -command up

# Check current version
go run cmd/migrate/main.go -command version

# Rollback all migrations
go run cmd/migrate/main.go -command down

# Force to specific version (recovery)
go run cmd/migrate/main.go -command force 1
```

#### 4. Install frontend dependencies

```bash
cd frontend
pnpm install
```

#### 5. Start development servers

```bash
# Terminal 1: Frontend
cd frontend
pnpm run dev

# Terminal 2: Backend (coming soon)
cd backend
go run cmd/server/main.go
```

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
