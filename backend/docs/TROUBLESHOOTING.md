# Deeb Troubleshooting Guide

This guide helps you diagnose and resolve common issues when developing with Deeb.

## Table of Contents

- [Database Connection Issues](#database-connection-issues)
- [Configuration Errors](#configuration-errors)
- [Build Errors](#build-errors)
- [Runtime Errors](#runtime-errors)
- [Testing Issues](#testing-issues)
- [Development Tools Issues](#development-tools-issues)

---

## Database Connection Issues

### PostgreSQL Connection Failed

**Symptoms:**
```
Error: failed to connect to database: dial tcp [::1]:5432: connect: connection refused
```

**Causes & Solutions:**

1. **PostgreSQL is not running**
   ```bash
   # Check if PostgreSQL is running
   psql -U postgres -c "SELECT version();"

   # Start PostgreSQL (macOS with Homebrew)
   brew services start postgresql@16

   # Start PostgreSQL (Linux)
   sudo systemctl start postgresql

   # Start PostgreSQL (Docker)
   docker-compose up -d postgres
   ```

2. **Wrong host or port**
   ```bash
   # Check your environment variables
   echo $DB_HOST
   echo $DB_PORT

   # Default is localhost:5432
   export DB_HOST=localhost
   export DB_PORT=5432
   ```

3. **Database doesn't exist**
   ```bash
   # Create the database
   createdb -U postgres deeb_metadata

   # Or using psql
   psql -U postgres -c "CREATE DATABASE deeb_metadata;"
   ```

4. **Invalid credentials**
   ```bash
   # Verify your credentials
   psql -U deeb -d deeb_metadata -h localhost -p 5432

   # Set correct credentials
   export DB_USER=deeb
   export DB_PASSWORD=your-password
   export DB_NAME=deeb_metadata
   ```

### SSL Connection Errors

**Symptoms:**
```
Error: pq: SSL is not enabled on the server
```

**Solution:**
```bash
# For local development, disable SSL
export DB_SSLMODE=disable

# For production with RDS, require SSL
export DB_SSLMODE=require
```

### Migration Failures

**Symptoms:**
```
Error: migration failed at version 001: ...
```

**Solutions:**

1. **Check migration status**
   ```bash
   cd backend
   make migrate-status
   ```

2. **Reset migrations (DEVELOPMENT ONLY)**
   ```bash
   # WARNING: This drops all tables
   make migrate-down
   make migrate-up
   ```

3. **Fix dirty migration state**
   ```bash
   # Check current version
   psql -U deeb -d deeb_metadata -c "SELECT * FROM schema_migrations;"

   # Force version (if needed)
   psql -U deeb -d deeb_metadata -c "UPDATE schema_migrations SET version = 1, dirty = false;"

   # Re-run migrations
   make migrate-up
   ```

### Connection Pool Exhausted

**Symptoms:**
```
Error: timeout waiting for connection from pool
```

**Solution:**
```bash
# Increase connection pool size (config.yaml)
database:
  max_open_conns: 50  # Increase from 25
  max_idle_conns: 10  # Increase from 5
  conn_max_lifetime: 5m
```

---

## Configuration Errors

### Missing Required Environment Variables

**Symptoms:**
```
config validation failed: auth.jwt_secret is required (set JWT_SECRET environment variable)
```

**Solution:**
```bash
# Check which variables are missing
cat .env.example

# Set all required variables
export JWT_SECRET="your-super-secret-jwt-key-minimum-32-characters-long"
export ENCRYPTION_KEY="12345678901234567890123456789012"
export DB_USER="deeb"
export DB_PASSWORD="your-password"
export DB_NAME="deeb_metadata"

# Or use .env file
cp .env.example .env
# Edit .env with your values
```

### Invalid JWT Secret Length

**Symptoms:**
```
config validation failed: auth.jwt_secret must be at least 32 characters
```

**Solution:**
```bash
# Generate a secure JWT secret (64 characters)
export JWT_SECRET=$(openssl rand -base64 48)

# Or use a long passphrase
export JWT_SECRET="your-super-secret-jwt-key-minimum-32-characters-long"
```

### Invalid Encryption Key Length

**Symptoms:**
```
config validation failed: security.encryption_key must be exactly 32 bytes
```

**Solution:**
```bash
# Generate a 32-byte encryption key
export ENCRYPTION_KEY=$(openssl rand -base64 32 | cut -c1-32)

# Or use exactly 32 characters
export ENCRYPTION_KEY="12345678901234567890123456789012"
```

### Port Already in Use

**Symptoms:**
```
Error: bind: address already in use
```

**Solutions:**

1. **Change the port**
   ```bash
   export SERVER_PORT=8081
   ```

2. **Kill the process using the port**
   ```bash
   # Find process using port 8080
   lsof -ti:8080

   # Kill the process
   kill -9 $(lsof -ti:8080)
   ```

3. **Stop the development server**
   ```bash
   # If Air is running in background
   pkill -f "air"
   ```

---

## Build Errors

### Go Module Issues

**Symptoms:**
```
go: module lookup disabled by GOPROXY=off
```
or
```
Error: cannot find module providing package github.com/...
```

**Solutions:**

1. **Download dependencies**
   ```bash
   cd backend
   go mod download
   go mod tidy
   ```

2. **Verify Go proxy**
   ```bash
   echo $GOPROXY
   # Should be: https://proxy.golang.org,direct

   export GOPROXY=https://proxy.golang.org,direct
   ```

3. **Clear module cache**
   ```bash
   go clean -modcache
   go mod download
   ```

### Vendor Directory Issues

**Symptoms:**
```
Error: module github.com/... not found in vendor
```

**Solution:**
```bash
cd backend
# Remove vendor directory
rm -rf vendor

# Re-vendor dependencies
go mod vendor
```

### Build Tool Missing

**Symptoms:**
```
make: *** No rule to make target 'build'.  Stop.
```

**Solutions:**

1. **Verify you're in the correct directory**
   ```bash
   cd backend
   pwd  # Should be .../deeb/backend
   ```

2. **Check available make targets**
   ```bash
   make help
   ```

3. **Install build dependencies**
   ```bash
   ./scripts/ci/setup-formatting.sh
   ```

### Go Version Mismatch

**Symptoms:**
```
Error: go version go1.21.0 is required, but go1.20.0 is installed
```

**Solution:**
```bash
# Check current version
go version

# Install Go 1.25.3+ (macOS)
brew install go@1.25

# Install Go 1.25.3+ (Linux)
wget https://go.dev/dl/go1.25.3.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.25.3.linux-amd64.tar.gz

# Verify installation
go version
```

---

## Runtime Errors

### JWT Token Invalid

**Symptoms:**
```
Error: invalid token: token is expired
```
or
```
Error: invalid token: signature is invalid
```

**Solutions:**

1. **Token expired** - Re-authenticate to get a new token
   ```bash
   # Login again via API or UI
   curl -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"user@example.com","password":"password"}'
   ```

2. **Wrong JWT secret** - Ensure JWT_SECRET hasn't changed
   ```bash
   # All services must use the same JWT_SECRET
   echo $JWT_SECRET
   ```

3. **Token format invalid** - Check Authorization header format
   ```bash
   # Correct format
   Authorization: Bearer <your-jwt-token>

   # Example
   curl -H "Authorization: Bearer eyJhbGc..." http://localhost:8080/api/v1/users
   ```

### Permission Denied Errors

**Symptoms:**
```
Error: user does not have permission to perform this action
```

**Solutions:**

1. **Check user permissions**
   ```bash
   # Query user's permissions
   psql -U deeb -d deeb_metadata -c "
     SELECT u.email, p.name as permission
     FROM users u
     JOIN user_groups ug ON u.id = ug.user_id
     JOIN group_permissions gp ON ug.group_id = gp.group_id
     JOIN permissions p ON gp.permission_id = p.id
     WHERE u.email = 'user@example.com';
   "
   ```

2. **Grant required permissions** (see RBAC documentation)

### CORS Errors

**Symptoms:**
```
Access to fetch at 'http://localhost:8080/api/v1/users' from origin 'http://localhost:3000' has been blocked by CORS policy
```

**Solution:**
```bash
# Add frontend URL to CORS origins in config.yaml
server:
  cors_origins:
    - "http://localhost:3000"
    - "http://localhost:5173"  # Vite default

# Or set via environment (not implemented yet, use config file)
```

### Panic: Runtime Error

**Symptoms:**
```
panic: runtime error: invalid memory address or nil pointer dereference
```

**Solutions:**

1. **Check logs for stack trace**
   ```bash
   # Run with debug logging
   export LOG_LEVEL=debug
   make run
   ```

2. **Enable Air for live reload** (to quickly test fixes)
   ```bash
   make run-air
   ```

3. **Check recent code changes** - Likely a nil pointer that wasn't checked

### Database Query Timeout

**Symptoms:**
```
Error: pq: canceling statement due to user request
```

**Solutions:**

1. **Check slow queries**
   ```sql
   -- Enable slow query logging
   SELECT * FROM pg_stat_activity WHERE state = 'active';
   ```

2. **Add database indexes** for frequently queried columns

3. **Increase query timeout** (if justified)
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
   defer cancel()
   ```

---

## Testing Issues

### Tests Failing: Database Issues

**Symptoms:**
```
Error: failed to connect to test database
```

**Solutions:**

1. **Use testcontainers** (automatically starts PostgreSQL)
   ```bash
   cd backend
   make test-integration
   ```

2. **Check test database exists**
   ```bash
   # Tests use a separate database
   createdb -U postgres deeb_test
   ```

3. **Set test environment variables**
   ```bash
   export DB_NAME=deeb_test
   export DB_SSLMODE=disable
   ```

### Tests Failing: Fixtures

**Symptoms:**
```
Error: fixture data not found
```

**Solution:**
```bash
# Ensure fixture files exist
ls backend/internal/test/fixtures/

# Re-run migrations on test database
export DB_NAME=deeb_test
make migrate-up
```

### gotestsum Not Found

**Symptoms:**
```
make: gotestsum: command not found
```

**Solution:**
```bash
# Install gotestsum
go install gotest.tools/gotestsum@latest

# Verify installation
which gotestsum
```

### Race Condition Detected

**Symptoms:**
```
WARNING: DATA RACE
```

**Solution:**
```bash
# Run tests with race detector
go test -race ./...

# Fix the race condition in code (add proper locking)
# See the stack trace for location
```

---

## Development Tools Issues

### Air Not Starting

**Symptoms:**
```
make: air: command not found
```

**Solution:**
```bash
# Install Air
go install github.com/cosmtrek/air@latest

# Verify installation
which air

# Run with Air
cd backend
make run-air
```

### Linting Failures

**Symptoms:**
```
Error: golangci-lint: command not found
```

**Solutions:**

1. **Install golangci-lint**
   ```bash
   # macOS
   brew install golangci-lint

   # Linux
   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

   # Verify
   golangci-lint version
   ```

2. **Run linting**
   ```bash
   cd backend
   make lint
   ```

3. **Auto-fix linting issues**
   ```bash
   cd backend
   make lint-fix
   ```

### Formatting Issues

**Symptoms:**
```
Error: golines: command not found
```

**Solution:**
```bash
# Install formatting tools
cd backend
./scripts/ci/setup-formatting.sh

# Run formatting
make format
```

### Docker Compose Issues

**Symptoms:**
```
Error: Cannot connect to the Docker daemon
```

**Solutions:**

1. **Start Docker daemon**
   ```bash
   # macOS - Start Docker Desktop
   open -a Docker

   # Linux
   sudo systemctl start docker
   ```

2. **Check Docker is running**
   ```bash
   docker ps
   ```

3. **Restart Docker Compose services**
   ```bash
   docker-compose down
   docker-compose up -d
   ```

---

## Getting More Help

If your issue isn't covered here:

1. **Check the logs** with `LOG_LEVEL=debug`
2. **Search existing issues** on GitHub
3. **Ask in the team chat** with:
   - Error message
   - Steps to reproduce
   - Your environment (OS, Go version, etc.)
4. **Create a detailed bug report** with:
   - Full error message
   - Stack trace
   - Configuration (sanitized)
   - Steps to reproduce

## Useful Debugging Commands

```bash
# Check all environment variables
env | grep -E "DB_|JWT_|SERVER_|LOG_"

# Test database connection
psql -U $DB_USER -d $DB_NAME -h $DB_HOST -p $DB_PORT -c "SELECT 1;"

# Check running processes
ps aux | grep -E "deeb|air|postgres"

# Check port usage
lsof -i :8080
lsof -i :5432

# View application logs
tail -f /var/log/deeb/app.log

# Check disk space
df -h

# Check memory usage
free -h

# Check Go environment
go env

# Check Go module status
go list -m all
```
