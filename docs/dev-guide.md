# Docker Development Environment

Quick guide for using Docker in Deeb development.

## Prerequisites

- Docker Desktop 20.10+
- Docker Compose v2

## Quick Start

```bash
# 1. Copy environment configuration
cp .env.example .env

# 2. Start PostgreSQL
docker compose up -d

# 3. Run migrations
cd backend
export DATABASE_URL="postgresql://deeb:deeb_dev_password@localhost:5432/deeb?sslmode=disable"
go run ./cmd/migrate/

# 4. Start the backend
go run ./cmd/server/

# 5. Verify
curl http://localhost:8080/health
```

**Expected response:**
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "timestamp": "...",
  "database": "ok"
}
```

## Services

### PostgreSQL Database

- **Port**: 5432
- **Database**: deeb
- **Username**: deeb
- **Password**: deeb_dev_password (dev only!)
- **Volume**: postgres_data (persistent)

## Common Commands

### Service Management

```bash
# Start services
docker compose up -d

# Start with logs
docker compose up

# Stop services
docker compose down

# Stop and remove data
docker compose down -v

# Restart
docker compose restart
```

### Logs

```bash
# View all logs
docker compose logs -f

# PostgreSQL logs
docker compose logs -f postgres

# Tail last 100 lines
docker compose logs -f --tail=100 postgres
```

### Database Access

```bash
# Access PostgreSQL CLI
docker compose exec postgres psql -U deeb -d deeb

# Example queries
SELECT * FROM users LIMIT 5;
\dt  # List tables
\q   # Quit

# Run SQL file
docker compose exec -T postgres psql -U deeb -d deeb < script.sql

# Backup
docker compose exec postgres pg_dump -U deeb deeb > backup.sql

# Restore
docker compose exec -T postgres psql -U deeb -d deeb < backup.sql
```

### Service Status

```bash
# Check status
docker compose ps

# View configuration
docker compose config
```

## Development Workflow

### Daily Development

```bash
# 1. Start services
docker compose up -d

# 2. Run migrations (if needed)
cd backend && go run ./cmd/migrate/

# 3. Start backend
go run ./cmd/server/

# 4. Start frontend (new terminal)
cd frontend && pnpm run dev

# 5. When done
docker compose stop
```

### Clean Restart

```bash
# Remove everything
docker compose down -v

# Start fresh
docker compose up -d

# Re-run migrations
cd backend && go run ./cmd/migrate/
```

## Troubleshooting

### Port 5432 Already in Use

**Option 1: Stop local PostgreSQL**
```bash
# macOS
brew services stop postgresql

# Linux
sudo systemctl stop postgresql
```

**Option 2: Change port**
```yaml
# In docker-compose.yml
ports:
  - "5433:5432"  # Use 5433 instead

# Update DATABASE_URL
postgresql://deeb:deeb_dev_password@localhost:5433/deeb?sslmode=disable
```

### Database Connection Issues

```bash
# Check health
docker compose ps postgres

# View logs
docker compose logs postgres

# Test connection
docker compose exec postgres pg_isready -U deeb -d deeb
```

### Reset Database

```bash
docker compose down -v
docker compose up -d
sleep 5
cd backend && go run ./cmd/migrate/
```

## Environment Variables

Key variables in `.env`:

```bash
# Database
DATABASE_URL=postgresql://deeb:deeb_dev_password@localhost:5432/deeb?sslmode=disable

# Server
SERVER_PORT=8080

# Logging
LOG_LEVEL=debug
LOG_FORMAT=console

# Security (generate with: openssl rand -base64 32)
JWT_SECRET=your-secret-here
ENCRYPTION_KEY=your-32-byte-key-here
```

See `.env.example` for complete list.

## Production Notes

**WARNING**: Default credentials are for development only!

For production:
1. Use strong, random passwords
2. Store credentials in secrets manager
3. Enable SSL/TLS for database connections
4. Configure backups and monitoring
5. Use external volumes or managed databases
6. Review security settings

## Networking

- Services use `deeb-network` bridge network
- Services communicate via service names (e.g., `postgres:5432`)
- Host accesses via `localhost:<port>`

## Volumes

- **postgres_data**: Persistent PostgreSQL storage
  - Docker managed volume
  - Survives container restarts
  - Removed with `docker compose down -v`

## Next Steps

- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution guide
- [SETUP.md](SETUP.md) - Complete setup guide
- [Backend Development](../backend/README.md) - Backend docs
