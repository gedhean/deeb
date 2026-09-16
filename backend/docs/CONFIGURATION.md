# Deeb Configuration Guide

This document describes all configuration options for the Deeb application.

## Configuration Sources

Deeb supports configuration from multiple sources, loaded in this order:

1. **Default values** - Built-in defaults
2. **Config file** (optional) - YAML configuration file
3. **Environment variables** - Override file and defaults

## Configuration File

Create a `config.yaml` file in the backend directory:

```bash
cp config.example.yaml config.yaml
```

Then edit `config.yaml` with your settings.

## Environment Variables

All configuration options can be set via environment variables. Environment variables **always take precedence** over config file values.

### Required Environment Variables

These must be set for the application to start:

```bash
# Authentication (minimum 32 characters)
export JWT_SECRET="your-super-secret-jwt-key-minimum-32-characters"

# Data encryption (exactly 32 bytes)
export ENCRYPTION_KEY="12345678901234567890123456789012"

# Database connection
export DB_USER="deeb"
export DB_PASSWORD="your-database-password"
export DB_NAME="deeb_metadata"
```

## Configuration Options

### Server Configuration

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| `server.host` | `SERVER_HOST` | `0.0.0.0` | Server bind address |
| `server.port` | `SERVER_PORT` | `8080` | Server port |
| `server.read_timeout` | - | `10s` | HTTP read timeout |
| `server.write_timeout` | - | `10s` | HTTP write timeout |
| `server.shutdown_timeout` | - | `30s` | Graceful shutdown timeout |
| `server.cors_origins` | - | `["http://localhost:3000"]` | Allowed CORS origins |

**Example:**
```bash
export SERVER_HOST="0.0.0.0"
export SERVER_PORT="8080"
```

### Database Configuration

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| `database.host` | `DB_HOST` | `localhost` | PostgreSQL host |
| `database.port` | `DB_PORT` | `5432` | PostgreSQL port |
| `database.user` | `DB_USER` | - | Database user (REQUIRED) |
| `database.password` | `DB_PASSWORD` | - | Database password |
| `database.database` | `DB_NAME` | - | Database name (REQUIRED) |
| `database.ssl_mode` | `DB_SSLMODE` | `disable` | SSL mode: `disable`, `require`, `verify-ca`, `verify-full` |
| `database.max_open_conns` | - | `25` | Maximum open connections |
| `database.max_idle_conns` | - | `5` | Maximum idle connections |
| `database.conn_max_lifetime` | - | `5m` | Connection maximum lifetime |
| `database.auto_migrate` | `DB_AUTO_MIGRATE` | `true` | Automatically run migrations at startup |
| `database.migrations_path` | `DB_MIGRATIONS_PATH` | `migrations` | Path to migration files directory |

**Example:**
```bash
export DB_HOST="localhost"
export DB_PORT="5432"
export DB_USER="deeb"
export DB_PASSWORD="secure-password"
export DB_NAME="deeb_metadata"
export DB_SSLMODE="require"  # For production
export DB_AUTO_MIGRATE="true"  # Run migrations automatically (default)
export DB_MIGRATIONS_PATH="migrations"  # Path to migration files
```

### Authentication Configuration

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| `auth.jwt_secret` | `JWT_SECRET` | - | JWT secret key (REQUIRED, min 32 chars) |
| `auth.jwt_expiration` | - | `24h` | JWT token expiration |
| `auth.google_client_id` | `GOOGLE_CLIENT_ID` | - | Google OAuth client ID |
| `auth.google_client_secret` | `GOOGLE_CLIENT_SECRET` | - | Google OAuth client secret |
| `auth.github_client_id` | `GITHUB_CLIENT_ID` | - | GitHub OAuth client ID |
| `auth.github_client_secret` | `GITHUB_CLIENT_SECRET` | - | GitHub OAuth client secret |
| `auth.callback_url` | `AUTH_CALLBACK_URL` | - | OAuth callback URL |
| `auth.require_2fa` | `REQUIRE_2FA` | `false` | Require 2FA for all users |

**Generating a secure JWT secret:**
```bash
# Generate a random 64-character secret
export JWT_SECRET=$(openssl rand -base64 48)
```

**Example:**
```bash
export JWT_SECRET="your-super-secret-jwt-key-minimum-32-characters-long"
export GOOGLE_CLIENT_ID="123456-abc.apps.googleusercontent.com"
export GOOGLE_CLIENT_SECRET="your-google-client-secret"
export AUTH_CALLBACK_URL="https://deeb.yourdomain.com/api/v1/auth/callback"
export REQUIRE_2FA="true"  # Enforce 2FA organization-wide
```

### Security Configuration

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| `security.bcrypt_cost` | `BCRYPT_COST` | `12` | Bcrypt hashing cost (10-31) |
| `security.rate_limit_enabled` | - | `true` | Enable rate limiting |
| `security.rate_limit_requests` | - | `100` | Requests allowed per window |
| `security.rate_limit_window` | - | `1m` | Rate limit time window |
| `security.encryption_key` | `ENCRYPTION_KEY` | - | Encryption key (REQUIRED, exactly 32 bytes) |

**Generating an encryption key:**
```bash
# Generate a random 32-byte key
export ENCRYPTION_KEY=$(openssl rand -base64 32 | cut -c1-32)
```

**Example:**
```bash
export ENCRYPTION_KEY="12345678901234567890123456789012"  # Exactly 32 bytes
export BCRYPT_COST="14"  # Higher cost for more security (slower)
```

**Note on bcrypt cost:**
- `10`: Fast, suitable for development
- `12`: Recommended for production (default)
- `14`: High security, slower
- `16+`: Very slow, use only if security is critical

### Logging Configuration

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| `logging.level` | `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `logging.format` | `LOG_FORMAT` | `json` | Log format: `json`, `console` |
| `logging.output` | - | `stdout` | Log output: `stdout`, `stderr`, `file` |
| `logging.file_path` | - | - | Log file path (if output is `file`) |

### Ghost Executor Configuration

Optional online-schema-change executor for DDL plans on non-production
environments (gh-ost for MySQL, pg-online-schema-change for PostgreSQL).
See [Ghost Executor](../../docs/ghost-executor.md).

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| `features.ghost_executor.ghost_bin` | `GHOST_BIN` | `/app/gh-ost` | Path to the `gh-ost` binary (bundled in image) |
| `features.ghost_executor.pgosc_bin` | `PGOSC_BIN` | _(off)_ | Path to `pg-online-schema-change` |
| `features.ghost_executor.flag_dir` | `GHOST_FLAG_DIR` | `.` | Directory for cutover/panic flag files |
| `features.ghost_executor.max_duration` | `GHOST_MAX_DURATION` | `6h` | Max ghost run duration (sizes queue window) |

**Example:**
```bash
# Development
export LOG_LEVEL="debug"
export LOG_FORMAT="console"

# Production
export LOG_LEVEL="info"
export LOG_FORMAT="json"
```

## Complete Configuration Examples

### Development Environment

`.env` file:
```bash
# Server
SERVER_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=deeb
DB_PASSWORD=dev-password
DB_NAME=deeb_metadata
DB_SSLMODE=disable

# Auth
JWT_SECRET="dev-secret-key-minimum-32-characters-long-for-testing"
GOOGLE_CLIENT_ID=""
GOOGLE_CLIENT_SECRET=""

# Security
ENCRYPTION_KEY="12345678901234567890123456789012"
BCRYPT_COST=10  # Faster for development

# Logging
LOG_LEVEL=debug
LOG_FORMAT=console
```

### Production Environment

Environment variables (set in your deployment system):
```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Database (AWS RDS)
DB_HOST=deeb-prod.xxxx.us-east-1.rds.amazonaws.com
DB_PORT=5432
DB_USER=deeb_prod
DB_PASSWORD=<your-secure-password>
DB_NAME=deeb_metadata
DB_SSLMODE=require

# Auth
JWT_SECRET=<generated-secret-min-32-chars>
GOOGLE_CLIENT_ID=<your-google-client-id>
GOOGLE_CLIENT_SECRET=<your-google-client-secret>
AUTH_CALLBACK_URL=https://deeb.yourdomain.com/api/v1/auth/callback
REQUIRE_2FA=true

# Security
ENCRYPTION_KEY=<generated-32-byte-key>
BCRYPT_COST=12

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

## Loading Configuration

### From Code

```go
import "github.com/nsxbet/deeb/internal/config"

// Load from environment variables only
cfg, err := config.Load("")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// Load from file + environment variables
cfg, err := config.Load("config.yaml")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// Access configuration
fmt.Printf("Server running on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
fmt.Printf("Database: %s\n", cfg.DatabaseURL())
```

### Validation

Configuration is automatically validated when loaded. If validation fails, the application will not start and will show helpful error messages:

```
config validation failed: auth.jwt_secret is required (set JWT_SECRET environment variable)
```

```
config validation failed: security.encryption_key must be exactly 32 bytes
```

```
config validation failed: database.ssl_mode must be one of: disable, require, verify-ca, verify-full
```

## Security Best Practices

1. **Never commit secrets** to version control
2. **Use environment variables** for sensitive values in production
3. **Generate strong secrets**:
   ```bash
   # JWT Secret (64 characters)
   openssl rand -base64 48

   # Encryption Key (32 bytes)
   openssl rand -base64 32 | cut -c1-32
   ```
4. **Enable SSL** for database connections in production (`DB_SSLMODE=require`)
5. **Use strong bcrypt cost** in production (minimum 12)
6. **Enable 2FA** for production environments
7. **Rotate secrets** regularly
8. **Use secrets management** systems (AWS Secrets Manager, HashiCorp Vault, etc.)

## Troubleshooting

### Configuration validation errors

If you see validation errors, check:
1. Required environment variables are set
2. Values are in correct format (e.g., ports are numbers)
3. String lengths meet requirements (JWT secret min 32 chars)
4. Encryption key is exactly 32 bytes

### Database connection issues

If database connection fails, verify:
1. `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` are correct
2. Database server is running and accessible
3. SSL mode matches your database configuration
4. Firewall allows connections

### OAuth configuration issues

For OAuth to work:
1. OAuth client IDs and secrets must be set
2. Callback URL must match OAuth provider configuration
3. Callback URL must be accessible from the internet (for OAuth redirect)

## References

- [PostgreSQL Connection Strings](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING)
- [Google OAuth Setup](https://developers.google.com/identity/protocols/oauth2)
- [GitHub OAuth Setup](https://docs.github.com/en/developers/apps/building-oauth-apps)
