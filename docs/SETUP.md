# Deeb Setup Guide

This guide provides comprehensive instructions for setting up the Deeb development environment.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Detailed Setup Instructions](#detailed-setup-instructions)
  - [Backend Setup](#backend-setup)
  - [Frontend Setup](#frontend-setup)
  - [Database Setup](#database-setup)
- [Environment Variables](#environment-variables)
- [Development Workflow](#development-workflow)
- [Docker Development](#docker-development)
- [Troubleshooting](#troubleshooting)

## Prerequisites

Before you begin, ensure you have the following installed:

### Required

- **Go 1.25.3 or higher**
  - Download from [golang.org](https://golang.org/dl/)
  - Verify installation: `go version`

- **Node.js 20+ with pnpm**
  - Download from [nodejs.org](https://nodejs.org/)
  - Install pnpm: `npm install -g pnpm`
  - Verify installation: `node --version && pnpm --version`

- **PostgreSQL 16+**
  - Download from [postgresql.org](https://www.postgresql.org/download/)
  - Verify installation: `psql --version`

- **Docker & Docker Compose** (for containerized development)
  - Download from [docker.com](https://www.docker.com/get-started)
  - Verify installation: `docker --version && docker compose version`

### Optional

- **Make** (for using Makefile commands)
  - Linux: `sudo apt-get install build-essential`
  - macOS: Included with Xcode Command Line Tools
  - Windows: Install via [chocolatey](https://chocolatey.org/)

- **Air** (for live reload during development)
  - Automatically installed in Docker dev environment
  - Manual install: `go install github.com/air-verse/air@latest`

## Quick Start

Get up and running quickly with Docker Compose:

```bash
# 1. Clone the repository
git clone https://github.com/nsxbet/deeb.git
cd deeb/backend

# 2. Copy environment file
cp .env.example .env

# 3. Edit .env and set required values (at minimum: JWT_SECRET, ENCRYPTION_KEY)
vim .env

# 4. Start development environment with live reload
make run-air

# 5. Access the application
# - Backend API: http://localhost:8080
# - Health check: http://localhost:8080/health
# - API docs: http://localhost:8080/api/docs
```

## Detailed Setup Instructions

### Backend Setup

#### 1. Clone the Repository

```bash
git clone https://github.com/nsxbet/deeb.git
cd deeb/backend
```

#### 2. Install Go Dependencies

```bash
go mod download
```

#### 3. Configure Environment Variables

Create a `.env` file from the example:

```bash
cp .env.example .env
```

Edit `.env` and configure at minimum:

```bash
# Required: JWT secret (minimum 32 characters)
JWT_SECRET=your-secure-jwt-secret-key-minimum-32-characters

# Required: Encryption key (exactly 32 characters)
ENCRYPTION_KEY=12345678901234567890123456789012

# Required: Database credentials
DB_USER=deeb
DB_PASSWORD=your-secure-password
DB_NAME=deeb_metadata
```

See [Environment Variables](#environment-variables) section for complete configuration options.

#### 4. Set Up Database

Create the PostgreSQL database:

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database and user
CREATE DATABASE deeb_metadata;
CREATE USER deeb WITH ENCRYPTED PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE deeb_metadata TO deeb;

# Exit psql
\q
```

#### 5. Run Database Migrations

```bash
# Apply all migrations
go run main.go migrate up

# Verify migrations
go run main.go migrate version
```

#### 6. Build and Run

**Option A: Direct run (without live reload)**

```bash
go run main.go server
```

**Option B: Docker with live reload (recommended)**

```bash
make run-air
```

**Option C: Production build**

```bash
make build
./target/$(go env GOOS)/bin/deeb server

# Or simply
make build
cd target/$(go env GOOS)/bin
./deeb server
```

#### 7. Verify Backend is Running

```bash
# Health check
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","version":"0.1.0","timestamp":"2024-10-26T...",
database":"ok"}

# API documentation
open http://localhost:8080/api/docs
```

### Frontend Setup

#### 1. Navigate to Frontend Directory

```bash
cd ../frontend
```

#### 2. Install Dependencies

```bash
pnpm install
```

#### 3. Configure Environment

Create `.env.local` for frontend configuration:

```bash
# API Backend URL
VITE_API_URL=http://localhost:8080

# Enable development mode
VITE_DEV_MODE=true
```

#### 4. Start Development Server

```bash
pnpm run dev
```

The frontend will be available at `http://localhost:5173`

#### 5. Build for Production

```bash
pnpm run build
pnpm run preview
```

### Database Setup

#### Local PostgreSQL

**Create Database:**

```bash
createdb -U postgres deeb_metadata
```

**Create User:**

```bash
psql -U postgres -c "CREATE USER deeb WITH ENCRYPTED PASSWORD 'your-secure-password';"
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE deeb_metadata TO deeb;"
```

#### Docker PostgreSQL

PostgreSQL is included in the Docker Compose setup:

```bash
# Start PostgreSQL container
docker compose -f docker/docker-compose.dev.yaml up postgres -d

# Connect to PostgreSQL
docker exec -it deeb-dev-postgres-1 psql -U deeb -d deeb_metadata
```

#### Database Migrations

**Apply Migrations:**

```bash
go run main.go migrate up
```

**Rollback Migrations:**

```bash
go run main.go migrate down
```

**Check Migration Version:**

```bash
go run main.go migrate version
```

**Force Migration Version (use with caution):**

```bash
go run main.go migrate force <version>
```

**Run Specific Number of Migrations:**

```bash
# Run next 2 migrations up
go run main.go migrate steps 2

# Rollback last migration
go run main.go migrate steps -1
```

## Environment Variables

### Server Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SERVER_HOST` | No | `0.0.0.0` | HTTP server host |
| `SERVER_PORT` | No | `8080` | HTTP server port |

### Database Configuration

**Option 1: Connection String**

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes* | - | PostgreSQL connection string |

Example:
```bash
DATABASE_URL=postgres://deeb:password@localhost:5432/deeb_metadata?sslmode=disable
```

**Option 2: Individual Variables** (overrides `DATABASE_URL`)

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DB_HOST` | Yes* | `localhost` | Database host |
| `DB_PORT` | No | `5432` | Database port |
| `DB_USER` | Yes | - | Database user |
| `DB_PASSWORD` | Yes | - | Database password |
| `DB_NAME` | Yes | - | Database name |
| `DB_SSLMODE` | No | `disable` | SSL mode (disable, require, verify-ca, verify-full) |

### Authentication Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | **Yes** | - | JWT signing secret (min 32 chars) |
| `GOOGLE_CLIENT_ID` | No | - | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | No | - | Google OAuth client secret |
| `AUTH_CALLBACK_URL` | No | - | OAuth callback URL |
| `REQUIRE_2FA` | No | `false` | Require 2FA for all users |

### Security Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ENCRYPTION_KEY` | **Yes** | - | Encryption key (exactly 32 chars) |
| `BCRYPT_COST` | No | `12` | Bcrypt cost factor (10-31) |

### Logging Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LOG_LEVEL` | No | `info` | Log level (debug, info, warn, error) |
| `LOG_FORMAT` | No | `json` | Log format (json, console) |

### Application Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `APP_VERSION` | No | `0.1.0` | Application version |

## Development Workflow

### Using Make Commands

The project provides convenient Make targets for common development tasks.

**Development:**

```bash
make run-air          # Start dev server with live reload
make docker-stop-dev  # Stop dev containers
make docker-down-dev  # Stop and remove dev containers
```

**Production:**

```bash
make run              # Start production-like server
make docker-stop      # Stop production containers
make docker-build     # Build production Docker image
```

**Testing:**

```bash
make test             # Run unit tests
make test-integration # Run integration tests
make test-verbose     # Run tests with verbose output
make test-coverage    # Run tests with coverage report
```

**Code Quality:**

```bash
make lint             # Run all linters
make lint-fix         # Auto-fix linting issues
make format           # Format all code
```

**Help:**

```bash
make help             # Show all available commands
```

### Development with Air (Live Reload)

Air automatically rebuilds and restarts the server when code changes:

```bash
# Start Air in Docker (recommended)
make run-air

# Or run Air directly (requires Air installed)
cd backend
air -c .air.toml
```

**Air Configuration:**

- Configuration file: `backend/.air.toml`
- Watched extensions: `.go`, `.yaml`, `.yml`
- Excluded directories: `vendor`, `tmp`, `testdata`, `docker`
- Build delay: 1 second
- Polling enabled for Docker compatibility

### Running Tests

**Unit Tests:**

```bash
make test
```

**Integration Tests:**

```bash
make test-integration
```

**Coverage Report:**

```bash
make test-coverage
open coverage.html
```

### Code Formatting and Linting

**Format Code:**

```bash
make format        # Format all code (Go + YAML)
make format-go     # Format Go code only
make format-yaml   # Format YAML files only
```

**Lint Code:**

```bash
make lint          # Run all linters
make lint-fix      # Auto-fix issues
make lint-golang   # Lint Go code only
make lint-yaml     # Lint YAML files only
```

## Docker Development

### Development Environment

The development environment includes:

- Backend with Air live reload
- PostgreSQL database
- Volume mounts for code changes
- Hot reload on file changes

**Start:**

```bash
make run-air
```

**Architecture:**

```yaml
services:
  backend:
    - Live reload with Air
    - Volume mounted source code
    - Port 8080:8080

  postgres:
    - PostgreSQL 16
    - Volume for data persistence
    - Port 5432:5432
```

**Logs:**

```bash
# View all logs
docker compose -f docker/docker-compose.dev.yaml logs -f

# View backend logs only
docker compose -f docker/docker-compose.dev.yaml logs -f backend

# View database logs only
docker compose -f docker/docker-compose.dev.yaml logs -f postgres
```

**Stop:**

```bash
make docker-stop-dev    # Stop containers
make docker-down-dev    # Stop and remove containers
```

### Production Environment

**Build:**

```bash
make docker-build
```

**Run:**

```bash
make run
```

**Stop:**

```bash
make docker-stop
```

## Troubleshooting

### Database Connection Issues

**Problem:** `connection refused` or `database does not exist`

**Solution:**

```bash
# Check if PostgreSQL is running
docker compose -f docker/docker-compose.dev.yaml ps

# Check database logs
docker compose -f docker/docker-compose.dev.yaml logs postgres

# Recreate database
make docker-down-dev
make run-air
```

### Environment Variable Issues

**Problem:** `config validation failed: auth.jwt_secret is required`

**Solution:**

Ensure `.env` file exists and contains required variables:

```bash
# Check if .env exists
ls -la backend/.env

# If not, copy from example
cp backend/.env.example backend/.env

# Edit and set JWT_SECRET and ENCRYPTION_KEY
nano backend/.env
```

### Air Not Reloading

**Problem:** Code changes not triggering rebuild

**Solution:**

Air uses polling mode for Docker compatibility. Check `.air.toml`:

```toml
poll = true
poll_interval = 1000  # Check every 1 second
```

If still not working, restart Air:

```bash
make docker-down-dev
make run-air
```

### Port Already in Use

**Problem:** `bind: address already in use`

**Solution:**

```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or change port in .env
echo "SERVER_PORT=8081" >> backend/.env
```

### Migration Errors

**Problem:** `dirty database version` or migration fails

**Solution:**

```bash
# Check current version
go run main.go migrate version

# Force to last known good version
go run main.go migrate force <version>

# Reapply migrations
go run main.go migrate up
```

### Build Errors

**Problem:** `cannot find package` or `inconsistent vendoring`

**Solution:**

```bash
# Clean and reinstall dependencies
rm -rf vendor/
go mod tidy
go mod vendor
go mod download

# Rebuild
make build
```

## Next Steps

Once your development environment is set up:

1. **Explore the API**: Visit http://localhost:8080/api/docs
2. **Read the Architecture**: See [project.md](project.md)
3. **Review the PRD**: See [../.wiz/deeb-mvp/prd.md](../.wiz/deeb-mvp/prd.md)
4. **Check Development Status**: Run `make help` to see available commands

## Additional Resources

- [README.md](../README.md) - Project overview
- [project.md](project.md) - Architecture and design
- [backend/docs/schema-erd.md](../backend/docs/schema-erd.md) - Database schema
- [.wiz/deeb-mvp/prd.md](../.wiz/deeb-mvp/prd.md) - Product requirements
- [.wiz/deeb-mvp/IMPLEMENTATION_GUIDE.md](../.wiz/deeb-mvp/IMPLEMENTATION_GUIDE.md) - Implementation phases
