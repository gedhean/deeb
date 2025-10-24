package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the Deeb application
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Security SecurityConfig `yaml:"security"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	CORSOrigins     []string      `yaml:"cors_origins"`
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Database        string        `yaml:"database"`
	SSLMode         string        `yaml:"ssl_mode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret          string        `yaml:"jwt_secret"`
	JWTExpiration      time.Duration `yaml:"jwt_expiration"`
	GoogleClientID     string        `yaml:"google_client_id"`
	GoogleClientSecret string        `yaml:"google_client_secret"`
	GitHubClientID     string        `yaml:"github_client_id"`
	GitHubClientSecret string        `yaml:"github_client_secret"`
	CallbackURL        string        `yaml:"callback_url"`
	Require2FA         bool          `yaml:"require_2fa"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	BcryptCost        int           `yaml:"bcrypt_cost"`
	RateLimitEnabled  bool          `yaml:"rate_limit_enabled"`
	RateLimitRequests int           `yaml:"rate_limit_requests"`
	RateLimitWindow   time.Duration `yaml:"rate_limit_window"`
	EncryptionKey     string        `yaml:"encryption_key"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level    string `yaml:"level"`     // debug, info, warn, error
	Format   string `yaml:"format"`    // json, console
	Output   string `yaml:"output"`    // stdout, stderr, file
	FilePath string `yaml:"file_path"` // if output is file
}

// Load loads configuration from environment variables and optionally a YAML file
func Load(configPath string) (*Config, error) {
	cfg := &Config{}

	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	// Load defaults
	cfg.setDefaults()

	// Load from file if provided
	if configPath != "" {
		if err := cfg.loadFromFile(configPath); err != nil {
			return nil, fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Override with environment variables
	cfg.loadFromEnv()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default values for all configuration options
func (c *Config) setDefaults() {
	// Server defaults
	c.Server.Host = "0.0.0.0"
	c.Server.Port = 8080
	c.Server.ReadTimeout = 10 * time.Second
	c.Server.WriteTimeout = 10 * time.Second
	c.Server.ShutdownTimeout = 30 * time.Second
	c.Server.CORSOrigins = []string{"http://localhost:3000"}

	// Database defaults
	c.Database.Host = "localhost"
	c.Database.Port = 5432
	c.Database.SSLMode = "disable"
	c.Database.MaxOpenConns = 25
	c.Database.MaxIdleConns = 5
	c.Database.ConnMaxLifetime = 5 * time.Minute

	// Auth defaults
	c.Auth.JWTExpiration = 24 * time.Hour
	c.Auth.Require2FA = false

	// Security defaults
	c.Security.BcryptCost = 12
	c.Security.RateLimitEnabled = true
	c.Security.RateLimitRequests = 100
	c.Security.RateLimitWindow = 1 * time.Minute

	// Logging defaults
	c.Logging.Level = "info"
	c.Logging.Format = "json"
	c.Logging.Output = "stdout"
}

// loadFromFile loads configuration from a YAML file
func (c *Config) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() {
	// Server
	if v := os.Getenv("SERVER_HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}

	// Database - check DATABASE_URL first, then individual env vars
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		if err := c.parseDatabaseURL(databaseURL); err != nil {
			// Log error but continue - individual env vars might override
			fmt.Fprintf(os.Stderr, "Warning: failed to parse DATABASE_URL: %v\n", err)
		}
	}

	// Individual database env vars override DATABASE_URL
	if v := os.Getenv("DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Database.Port = port
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		c.Database.Database = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		c.Database.SSLMode = v
	}

	// Auth
	if v := os.Getenv("JWT_SECRET"); v != "" {
		c.Auth.JWTSecret = v
	}
	if v := os.Getenv("GOOGLE_CLIENT_ID"); v != "" {
		c.Auth.GoogleClientID = v
	}
	if v := os.Getenv("GOOGLE_CLIENT_SECRET"); v != "" {
		c.Auth.GoogleClientSecret = v
	}
	if v := os.Getenv("GITHUB_CLIENT_ID"); v != "" {
		c.Auth.GitHubClientID = v
	}
	if v := os.Getenv("GITHUB_CLIENT_SECRET"); v != "" {
		c.Auth.GitHubClientSecret = v
	}
	if v := os.Getenv("AUTH_CALLBACK_URL"); v != "" {
		c.Auth.CallbackURL = v
	}
	if v := os.Getenv("REQUIRE_2FA"); v != "" {
		c.Auth.Require2FA = v == "true"
	}

	// Security
	if v := os.Getenv("ENCRYPTION_KEY"); v != "" {
		c.Security.EncryptionKey = v
	}
	if v := os.Getenv("BCRYPT_COST"); v != "" {
		if cost, err := strconv.Atoi(v); err == nil {
			c.Security.BcryptCost = cost
		}
	}

	// Logging
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.Logging.Level = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		c.Logging.Format = v
	}
}

// parseDatabaseURL parses a PostgreSQL connection URL and populates the database config
// Supports formats:
//   - postgresql://user:password@host:port/database?sslmode=disable
//   - postgres://user:password@host:port/database?sslmode=disable
func (c *Config) parseDatabaseURL(databaseURL string) error {
	// Parse the URL
	u, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("invalid database URL: %w", err)
	}

	// Check scheme
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("invalid database URL scheme: %s (expected postgres or postgresql)", u.Scheme)
	}

	// Extract host and port
	host := u.Hostname()
	if host != "" {
		c.Database.Host = host
	}

	port := u.Port()
	if port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Database.Port = p
		}
	}

	// Extract username
	if u.User != nil {
		c.Database.User = u.User.Username()
		if password, ok := u.User.Password(); ok {
			c.Database.Password = password
		}
	}

	// Extract database name (remove leading slash)
	if u.Path != "" {
		c.Database.Database = strings.TrimPrefix(u.Path, "/")
	}

	// Extract query parameters
	query := u.Query()
	if sslmode := query.Get("sslmode"); sslmode != "" {
		c.Database.SSLMode = sslmode
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Server validation
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}

	// Database validation
	if c.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return fmt.Errorf("database.port must be between 1 and 65535, got %d", c.Database.Port)
	}
	if c.Database.User == "" {
		return fmt.Errorf("database.user is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("database.database is required")
	}
	if c.Database.SSLMode != "disable" && c.Database.SSLMode != "require" && c.Database.SSLMode != "verify-ca" &&
		c.Database.SSLMode != "verify-full" {
		return fmt.Errorf("database.ssl_mode must be one of: disable, require, verify-ca, verify-full")
	}

	// Auth validation
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret is required (set JWT_SECRET environment variable)")
	}
	if len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("auth.jwt_secret must be at least 32 characters")
	}

	// Security validation
	if c.Security.BcryptCost < 10 || c.Security.BcryptCost > 31 {
		return fmt.Errorf("security.bcrypt_cost must be between 10 and 31, got %d", c.Security.BcryptCost)
	}
	if c.Security.EncryptionKey == "" {
		return fmt.Errorf("security.encryption_key is required (set ENCRYPTION_KEY environment variable)")
	}
	if len(c.Security.EncryptionKey) != 32 {
		return fmt.Errorf("security.encryption_key must be exactly 32 bytes")
	}

	// Logging validation
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("logging.level must be one of: debug, info, warn, error, got %s", c.Logging.Level)
	}
	if c.Logging.Format != "json" && c.Logging.Format != "console" {
		return fmt.Errorf("logging.format must be one of: json, console, got %s", c.Logging.Format)
	}

	return nil
}

// DatabaseURL returns a PostgreSQL connection string
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Database,
		c.Database.SSLMode,
	)
}
