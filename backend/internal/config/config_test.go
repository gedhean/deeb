package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDatabaseURL(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
		want        DatabaseConfig
		wantErr     bool
	}{
		{
			name:        "valid postgresql URL",
			databaseURL: "postgresql://deeb:deeb_dev_password@localhost:5432/deeb?sslmode=disable",
			want: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "deeb",
				Password: "deeb_dev_password",
				Database: "deeb",
				SSLMode:  "disable",
			},
			wantErr: false,
		},
		{
			name:        "valid postgres URL",
			databaseURL: "postgres://testuser:testpass@db.example.com:5433/testdb?sslmode=require",
			want: DatabaseConfig{
				Host:     "db.example.com",
				Port:     5433,
				User:     "testuser",
				Password: "testpass",
				Database: "testdb",
				SSLMode:  "require",
			},
			wantErr: false,
		},
		{
			name:        "URL without password",
			databaseURL: "postgresql://deeb@localhost:5432/deeb?sslmode=disable",
			want: DatabaseConfig{
				Host:     "localhost",
				Port:     5432,
				User:     "deeb",
				Password: "",
				Database: "deeb",
				SSLMode:  "disable",
			},
			wantErr: false,
		},
		{
			name:        "URL without port (defaults handled by config defaults)",
			databaseURL: "postgresql://deeb:pass@localhost/deeb",
			want: DatabaseConfig{
				Host:     "localhost",
				Port:     5432, // Will be from defaults
				User:     "deeb",
				Password: "pass",
				Database: "deeb",
				SSLMode:  "disable", // Will be from defaults
			},
			wantErr: false,
		},
		{
			name:        "invalid scheme",
			databaseURL: "mysql://user:pass@localhost:3306/db",
			wantErr:     true,
		},
		{
			name:        "invalid URL",
			databaseURL: "not a valid url",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			cfg.setDefaults()

			err := cfg.parseDatabaseURL(tt.databaseURL)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.Host, cfg.Database.Host)
			assert.Equal(t, tt.want.Port, cfg.Database.Port)
			assert.Equal(t, tt.want.User, cfg.Database.User)
			assert.Equal(t, tt.want.Password, cfg.Database.Password)
			assert.Equal(t, tt.want.Database, cfg.Database.Database)
			assert.Equal(t, tt.want.SSLMode, cfg.Database.SSLMode)
		})
	}
}

func TestLoadFromEnv_DatabaseURL(t *testing.T) {
	// Save original env vars
	originalVars := map[string]string{
		"DATABASE_URL": os.Getenv("DATABASE_URL"),
		"DB_HOST":      os.Getenv("DB_HOST"),
		"DB_PORT":      os.Getenv("DB_PORT"),
		"DB_USER":      os.Getenv("DB_USER"),
		"DB_PASSWORD":  os.Getenv("DB_PASSWORD"),
		"DB_NAME":      os.Getenv("DB_NAME"),
		"DB_SSLMODE":   os.Getenv("DB_SSLMODE"),
	}

	// Restore env vars after test
	defer func() {
		for k, v := range originalVars {
			if v == "" {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, v)
			}
		}
	}()

	t.Run("DATABASE_URL sets all database config", func(t *testing.T) {
		// Clear individual env vars
		_ = os.Unsetenv("DB_HOST")
		_ = os.Unsetenv("DB_PORT")
		_ = os.Unsetenv("DB_USER")
		_ = os.Unsetenv("DB_PASSWORD")
		_ = os.Unsetenv("DB_NAME")
		_ = os.Unsetenv("DB_SSLMODE")

		// Set DATABASE_URL
		_ = os.Setenv("DATABASE_URL", "postgresql://deeb:deeb_dev_password@localhost:5432/deeb?sslmode=disable")

		cfg := &Config{}
		cfg.setDefaults()
		cfg.loadFromEnv()

		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
		assert.Equal(t, "deeb", cfg.Database.User)
		assert.Equal(t, "deeb_dev_password", cfg.Database.Password)
		assert.Equal(t, "deeb", cfg.Database.Database)
		assert.Equal(t, "disable", cfg.Database.SSLMode)
	})

	t.Run("individual env vars override DATABASE_URL", func(t *testing.T) {
		// Set DATABASE_URL
		_ = os.Setenv("DATABASE_URL", "postgresql://deeb:deeb_dev_password@localhost:5432/deeb?sslmode=disable")

		// Set individual override
		_ = os.Setenv("DB_HOST", "overridden-host")
		_ = os.Setenv("DB_PORT", "9999")

		cfg := &Config{}
		cfg.setDefaults()
		cfg.loadFromEnv()

		// These should be overridden
		assert.Equal(t, "overridden-host", cfg.Database.Host)
		assert.Equal(t, 9999, cfg.Database.Port)

		// These should come from DATABASE_URL
		assert.Equal(t, "deeb", cfg.Database.User)
		assert.Equal(t, "deeb_dev_password", cfg.Database.Password)
		assert.Equal(t, "deeb", cfg.Database.Database)
		assert.Equal(t, "disable", cfg.Database.SSLMode)
	})
}

// TestSetDefaults tests that default values are correctly applied
func TestSetDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()

	// Server defaults
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 10*time.Second, cfg.Server.ReadTimeout)
	assert.Equal(t, 10*time.Second, cfg.Server.WriteTimeout)
	assert.Equal(t, 30*time.Second, cfg.Server.ShutdownTimeout)
	assert.Equal(t, []string{"http://localhost:3000"}, cfg.Server.CORSOrigins)

	// Database defaults
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, 25, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.Database.ConnMaxLifetime)

	// Auth defaults
	assert.Equal(t, 24*time.Hour, cfg.Auth.JWTExpiration)
	assert.False(t, cfg.Auth.Require2FA)

	// Security defaults
	assert.Equal(t, 12, cfg.Security.BcryptCost)
	assert.True(t, cfg.Security.RateLimitEnabled)
	assert.Equal(t, 100, cfg.Security.RateLimitRequests)
	assert.Equal(t, 1*time.Minute, cfg.Security.RateLimitWindow)

	// Logging defaults
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, "stdout", cfg.Logging.Output)
}

// TestLoadFromFile tests loading configuration from a YAML file
func TestLoadFromFile(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		// Create temporary config file
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `
server:
  host: "127.0.0.1"
  port: 9090
  read_timeout: 15s
  write_timeout: 15s
  shutdown_timeout: 45s
  cors_origins:
    - "http://localhost:5000"
    - "https://example.com"

database:
  host: "db.example.com"
  port: 5433
  user: "testuser"
  password: "testpass"
  database: "testdb"
  ssl_mode: "require"
  max_open_conns: 50
  max_idle_conns: 10
  conn_max_lifetime: 10m

auth:
  jwt_secret: "test-secret-key-that-is-at-least-32-characters-long"
  jwt_expiration: 48h
  google_client_id: "google-client-id"
  google_client_secret: "google-client-secret"
  github_client_id: "github-client-id"
  github_client_secret: "github-client-secret"
  callback_url: "http://localhost:8080/auth/callback"
  require_2fa: true

security:
  bcrypt_cost: 14
  rate_limit_enabled: false
  rate_limit_requests: 200
  rate_limit_window: 2m
  encryption_key: "12345678901234567890123456789012"

logging:
  level: "debug"
  format: "console"
  output: "stderr"
  file_path: "/var/log/deeb.log"
`

		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err)

		cfg := &Config{}
		cfg.setDefaults()
		err = cfg.loadFromFile(configPath)
		require.NoError(t, err)

		// Verify server config
		assert.Equal(t, "127.0.0.1", cfg.Server.Host)
		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, 15*time.Second, cfg.Server.ReadTimeout)
		assert.Equal(t, []string{"http://localhost:5000", "https://example.com"}, cfg.Server.CORSOrigins)

		// Verify database config
		assert.Equal(t, "db.example.com", cfg.Database.Host)
		assert.Equal(t, 5433, cfg.Database.Port)
		assert.Equal(t, "testuser", cfg.Database.User)
		assert.Equal(t, "require", cfg.Database.SSLMode)
		assert.Equal(t, 50, cfg.Database.MaxOpenConns)

		// Verify auth config
		assert.Equal(t, "test-secret-key-that-is-at-least-32-characters-long", cfg.Auth.JWTSecret)
		assert.Equal(t, 48*time.Hour, cfg.Auth.JWTExpiration)
		assert.True(t, cfg.Auth.Require2FA)

		// Verify security config
		assert.Equal(t, 14, cfg.Security.BcryptCost)
		assert.False(t, cfg.Security.RateLimitEnabled)

		// Verify logging config
		assert.Equal(t, "debug", cfg.Logging.Level)
		assert.Equal(t, "console", cfg.Logging.Format)
		assert.Equal(t, "stderr", cfg.Logging.Output)
	})

	t.Run("file not found", func(t *testing.T) {
		cfg := &Config{}
		err := cfg.loadFromFile("/nonexistent/config.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid.yaml")

		invalidContent := `
server:
  host: "localhost"
  port: not-a-number
`
		err := os.WriteFile(configPath, []byte(invalidContent), 0o600)
		require.NoError(t, err)

		cfg := &Config{}
		err = cfg.loadFromFile(configPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config file")
	})
}

// TestLoadFromEnv tests loading configuration from environment variables
func TestLoadFromEnv(t *testing.T) {
	// Save and restore environment
	originalEnv := map[string]string{
		"SERVER_HOST":          os.Getenv("SERVER_HOST"),
		"SERVER_PORT":          os.Getenv("SERVER_PORT"),
		"DB_HOST":              os.Getenv("DB_HOST"),
		"DB_PORT":              os.Getenv("DB_PORT"),
		"DB_USER":              os.Getenv("DB_USER"),
		"DB_PASSWORD":          os.Getenv("DB_PASSWORD"),
		"DB_NAME":              os.Getenv("DB_NAME"),
		"DB_SSLMODE":           os.Getenv("DB_SSLMODE"),
		"JWT_SECRET":           os.Getenv("JWT_SECRET"),
		"GOOGLE_CLIENT_ID":     os.Getenv("GOOGLE_CLIENT_ID"),
		"GOOGLE_CLIENT_SECRET": os.Getenv("GOOGLE_CLIENT_SECRET"),
		"GITHUB_CLIENT_ID":     os.Getenv("GITHUB_CLIENT_ID"),
		"GITHUB_CLIENT_SECRET": os.Getenv("GITHUB_CLIENT_SECRET"),
		"AUTH_CALLBACK_URL":    os.Getenv("AUTH_CALLBACK_URL"),
		"REQUIRE_2FA":          os.Getenv("REQUIRE_2FA"),
		"ENCRYPTION_KEY":       os.Getenv("ENCRYPTION_KEY"),
		"BCRYPT_COST":          os.Getenv("BCRYPT_COST"),
		"LOG_LEVEL":            os.Getenv("LOG_LEVEL"),
		"LOG_FORMAT":           os.Getenv("LOG_FORMAT"),
	}

	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, v)
			}
		}
	}()

	t.Run("environment variables override defaults", func(t *testing.T) {
		// Clear all env vars
		for k := range originalEnv {
			_ = os.Unsetenv(k)
		}

		// Set specific env vars
		_ = os.Setenv("SERVER_HOST", "192.168.1.1")
		_ = os.Setenv("SERVER_PORT", "7070")
		_ = os.Setenv("DB_HOST", "prod-db.example.com")
		_ = os.Setenv("DB_PORT", "5433")
		_ = os.Setenv("DB_USER", "produser")
		_ = os.Setenv("DB_PASSWORD", "prodpass")
		_ = os.Setenv("DB_NAME", "proddb")
		_ = os.Setenv("DB_SSLMODE", "require")
		_ = os.Setenv("JWT_SECRET", "env-jwt-secret-at-least-32-chars")
		_ = os.Setenv("GOOGLE_CLIENT_ID", "env-google-id")
		_ = os.Setenv("REQUIRE_2FA", "true")
		_ = os.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
		_ = os.Setenv("BCRYPT_COST", "13")
		_ = os.Setenv("LOG_LEVEL", "warn")
		_ = os.Setenv("LOG_FORMAT", "console")

		cfg := &Config{}
		cfg.setDefaults()
		cfg.loadFromEnv()

		// Verify server config
		assert.Equal(t, "192.168.1.1", cfg.Server.Host)
		assert.Equal(t, 7070, cfg.Server.Port)

		// Verify database config
		assert.Equal(t, "prod-db.example.com", cfg.Database.Host)
		assert.Equal(t, 5433, cfg.Database.Port)
		assert.Equal(t, "produser", cfg.Database.User)
		assert.Equal(t, "prodpass", cfg.Database.Password)
		assert.Equal(t, "proddb", cfg.Database.Database)
		assert.Equal(t, "require", cfg.Database.SSLMode)

		// Verify auth config
		assert.Equal(t, "env-jwt-secret-at-least-32-chars", cfg.Auth.JWTSecret)
		assert.Equal(t, "env-google-id", cfg.Auth.GoogleClientID)
		assert.True(t, cfg.Auth.Require2FA)

		// Verify security config
		assert.Equal(t, "12345678901234567890123456789012", cfg.Security.EncryptionKey)
		assert.Equal(t, 13, cfg.Security.BcryptCost)

		// Verify logging config
		assert.Equal(t, "warn", cfg.Logging.Level)
		assert.Equal(t, "console", cfg.Logging.Format)
	})

	t.Run("invalid port number ignored", func(t *testing.T) {
		_ = os.Setenv("SERVER_PORT", "not-a-number")

		cfg := &Config{}
		cfg.setDefaults()
		cfg.loadFromEnv()

		// Should keep default value
		assert.Equal(t, 8080, cfg.Server.Port)
	})
}

// TestValidate tests configuration validation
func TestValidate(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"

		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("invalid server port - too low", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Server.Port = 0
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server.port must be between 1 and 65535")
	})

	t.Run("invalid server port - too high", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Server.Port = 70000
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server.port must be between 1 and 65535")
	})

	t.Run("missing database host", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.Host = ""
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database.host is required")
	})

	t.Run("invalid database port", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.Port = 0
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database.port must be between 1 and 65535")
	})

	t.Run("missing database user", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.User = ""
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database.user is required")
	})

	t.Run("missing database name", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.User = "testuser"
		cfg.Database.Database = ""
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database.database is required")
	})

	t.Run("invalid ssl mode", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Database.SSLMode = "invalid"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database.ssl_mode must be one of")
	})

	t.Run("missing JWT secret", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = ""
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "auth.jwt_secret is required")
	})

	t.Run("JWT secret too short", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "short"
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "auth.jwt_secret must be at least 32 characters")
	})

	t.Run("invalid bcrypt cost - too low", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.BcryptCost = 5
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "security.bcrypt_cost must be between 10 and 31")
	})

	t.Run("invalid bcrypt cost - too high", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.BcryptCost = 32
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "security.bcrypt_cost must be between 10 and 31")
	})

	t.Run("missing encryption key", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.EncryptionKey = ""

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "security.encryption_key is required")
	})

	t.Run("encryption key wrong length", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.EncryptionKey = "too-short"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "security.encryption_key must be exactly 32 bytes")
	})

	t.Run("invalid log level", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"
		cfg.Logging.Level = "invalid"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "logging.level must be one of")
	})

	t.Run("invalid log format", func(t *testing.T) {
		cfg := &Config{}
		cfg.setDefaults()
		cfg.Database.User = "testuser"
		cfg.Database.Database = "testdb"
		cfg.Auth.JWTSecret = "valid-jwt-secret-that-is-at-least-32-characters"
		cfg.Security.EncryptionKey = "12345678901234567890123456789012"
		cfg.Logging.Format = "invalid"

		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "logging.format must be one of")
	})
}

// TestLoad tests the full Load function
func TestLoad(t *testing.T) {
	// Save and restore environment
	originalEnv := map[string]string{
		"JWT_SECRET":     os.Getenv("JWT_SECRET"),
		"ENCRYPTION_KEY": os.Getenv("ENCRYPTION_KEY"),
		"DB_USER":        os.Getenv("DB_USER"),
		"DB_NAME":        os.Getenv("DB_NAME"),
	}

	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, v)
			}
		}
	}()

	t.Run("load with valid environment", func(t *testing.T) {
		_ = os.Setenv("JWT_SECRET", "valid-jwt-secret-that-is-at-least-32-characters")
		_ = os.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
		_ = os.Setenv("DB_USER", "testuser")
		_ = os.Setenv("DB_NAME", "testdb")

		cfg, err := Load("")
		require.NoError(t, err)
		assert.NotNil(t, cfg)

		// Verify defaults were applied
		assert.Equal(t, 8080, cfg.Server.Port)
		// Verify env vars were loaded
		assert.Equal(t, "testuser", cfg.Database.User)
		assert.Equal(t, "testdb", cfg.Database.Database)
	})

	t.Run("load with config file and env override", func(t *testing.T) {
		// Clear all database env vars to avoid conflicts
		_ = os.Unsetenv("DATABASE_URL")
		_ = os.Unsetenv("DB_NAME")
		_ = os.Unsetenv("DB_HOST")
		_ = os.Unsetenv("DB_PORT")
		_ = os.Unsetenv("DB_PASSWORD")
		_ = os.Unsetenv("SERVER_PORT")

		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		configContent := `
server:
  port: 9090

database:
  host: "file-db-host"
  user: "fileuser"
  database: "filedb"
`
		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err)

		// Set env vars that should override file
		_ = os.Setenv("DB_USER", "envuser")
		_ = os.Setenv("JWT_SECRET", "valid-jwt-secret-that-is-at-least-32-characters")
		_ = os.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")

		cfg, err := Load(configPath)
		require.NoError(t, err)

		// From file
		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, "file-db-host", cfg.Database.Host)
		assert.Equal(t, "filedb", cfg.Database.Database)

		// From env (overrides file)
		assert.Equal(t, "envuser", cfg.Database.User)
	})

	t.Run("load fails with invalid config", func(t *testing.T) {
		// Clear required env vars
		_ = os.Unsetenv("JWT_SECRET")
		_ = os.Unsetenv("ENCRYPTION_KEY")
		_ = os.Unsetenv("DB_USER")
		_ = os.Unsetenv("DB_NAME")

		_, err := Load("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config validation failed")
	})

	t.Run("load fails with nonexistent file", func(t *testing.T) {
		_, err := Load("/nonexistent/config.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load config from file")
	})
}

// TestDatabaseURL tests the DatabaseURL helper function
func TestDatabaseURL(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()
	cfg.Database.Host = "db.example.com"
	cfg.Database.Port = 5433
	cfg.Database.User = "testuser"
	cfg.Database.Password = "testpass"
	cfg.Database.Database = "testdb"
	cfg.Database.SSLMode = "require"

	expected := "host=db.example.com port=5433 user=testuser password=testpass dbname=testdb sslmode=require"
	assert.Equal(t, expected, cfg.DatabaseURL())
}
