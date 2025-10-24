# Docker Development Environment

This guide covers setting up and using the Docker development environment for Deeb.

## Prerequisites

- Docker Desktop 20.10+ (includes Docker Compose v2)
- Docker Compose v2 (uses `docker compose` command)

## Quick Start

**1. Copy environment configuration**

```bash
cp .env.example .env
```

Edit `.env` with your configuration if needed (the defaults work for local development).

**2. Start services**

```bash
# Start PostgreSQL database
docker compose up -d

# Verify services are running
docker compose ps

# View logs
docker compose logs -f postgres
```

**3. Run database migrations**

```bash
cd backend
export DATABASE_URL="postgresql://deeb:deeb_dev_password@localhost:5432/deeb?sslmode=disable"
go run ./cmd/migrate/
```

**4. Start the backend server**

```bash
# Start the server (from backend directory)
go run ./cmd/server/
```

**5. Verify the setup**

```bash
# Check health endpoint
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","version":"0.1.0","timestamp":"...","database":"ok"}
```

## Docker Compose Services

### PostgreSQL Database

- **Image**: `postgres:16-alpine`
- **Port**: `5432` (mapped to host)
- **Database**: `deeb`
- **Username**: `deeb`
- **Password**: `deeb_dev_password` (for development only!)
- **Volume**: `postgres_data` (persistent storage)

## Common Commands

### Service Management

```bash
# Start all services
docker compose up -d

# Start services with logs visible
docker compose up

# Stop services
docker compose down

# Stop and remove volumes (cleans database)
docker compose down -v

# Restart services
docker compose restart

# Restart specific service
docker compose restart postgres
```

### Viewing Logs

```bash
# View all logs
docker compose logs

# Follow logs
docker compose logs -f

# View logs for specific service
docker compose logs postgres

# Follow logs for specific service with tail
docker compose logs -f --tail=100 postgres
```

### Database Access

```bash
# Access PostgreSQL CLI
docker compose exec postgres psql -U deeb -d deeb

# Run SQL file
docker compose exec -T postgres psql -U deeb -d deeb < script.sql

# Dump database
docker compose exec postgres pg_dump -U deeb deeb > backup.sql

# Restore database
docker compose exec -T postgres psql -U deeb -d deeb < backup.sql
```

### Service Status

```bash
# Check service status
docker compose ps

# Check service health
docker compose ps postgres

# View service configuration
docker compose config
```

## Development Workflow

### Daily Development

1. **Start services**:
   ```bash
   docker compose up -d
   ```

2. **Run migrations** (if schema changed):
   ```bash
   cd backend
   go run ./cmd/migrate/
   ```

3. **Start backend server**:
   ```bash
   go run ./cmd/server/
   ```

4. **When done, stop services**:
   ```bash
   docker compose stop
   ```

### Clean Restart

To start fresh with a clean database:

```bash
# Stop and remove everything including volumes
docker compose down -v

# Start services again
docker compose up -d

# Run migrations
cd backend
go run ./cmd/migrate/
```

## Troubleshooting

### Port 5432 Already in Use

If you have PostgreSQL running locally:

```bash
# Stop local PostgreSQL (macOS)
brew services stop postgresql

# Or change the port in docker-compose.yml
# Change "5432:5432" to "5433:5432"
# Then update DATABASE_URL to use port 5433
```

### Database Connection Issues

```bash
# Check if PostgreSQL is healthy
docker compose ps postgres

# View PostgreSQL logs
docker compose logs postgres

# Test connection
docker compose exec postgres pg_isready -U deeb -d deeb
```

### Reset Database

```bash
# Remove and recreate database
docker compose down -v
docker compose up -d

# Wait for database to be ready
sleep 5

# Run migrations
cd backend
go run ./cmd/migrate/
```

## Environment Variables

The following environment variables can be customized in `.env`:

- `DATABASE_URL`: PostgreSQL connection string
- `SERVER_PORT`: Backend server port (default: 8080)
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `LOG_FORMAT`: Log format (json, console)

See `.env.example` for the complete list of configuration options.

## Production Considerations

**⚠️ Important**: The default credentials in `docker-compose.yml` are for development only!

For production deployments:

1. Use strong, randomly generated passwords
2. Store credentials in a secrets manager
3. Use SSL/TLS for database connections
4. Configure proper backup and monitoring
5. Use external volumes or managed databases
6. Review and harden security settings

## Networking

All services are connected via the `deeb-network` bridge network:

- Services can communicate using service names (e.g., `postgres:5432`)
- Host machine can access services via `localhost:<port>`

## Volumes

- **postgres_data**: Persistent storage for PostgreSQL data
  - Location: Docker managed volume
  - Survives container restarts
  - Removed with `docker compose down -v`

## Next Steps

- [Backend Development Guide](backend/README.md)
- [Database Migrations](backend/cmd/migrate/README.md)
- [Configuration Guide](CONFIGURATION.md)
