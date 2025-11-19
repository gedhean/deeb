// Package app provides application initialization and lifecycle management for the Deeb server.
package app

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/config"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/handlers"
	"github.com/nsxbet/deeb/internal/logger"
	"github.com/nsxbet/deeb/internal/middleware"
	"github.com/nsxbet/deeb/internal/migrate"
	"github.com/nsxbet/deeb/internal/routes"
	"github.com/nsxbet/deeb/internal/validator"
	"go.uber.org/zap"
)

// App represents the application with all its dependencies and configuration.
type App struct {
	Config           *config.Config
	DB               *sqlx.DB
	Echo             *echo.Echo
	TokenManager     *auth.TokenManager
	LoginRateLimiter *handlers.RateLimiter
	Version          string
}

// New creates and initializes a new application instance.
// Loads configuration, initializes logger, connects to database, and sets up routes.
func New(version string, configPath string) (*App, error) {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	if err := logger.Initialize(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	}); err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	logger.Info("Starting Deeb server",
		zap.String("version", version),
		zap.String("config_path", configPath),
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
		zap.String("log_level", cfg.Logging.Level),
		zap.String("log_format", cfg.Logging.Format),
	)

	// Log audit logging configuration
	logger.Info("Audit logging configuration loaded",
		zap.Bool("enabled", cfg.AuditLogging.Enabled),
		zap.String("default_policy", cfg.AuditLogging.DefaultPolicy),
		zap.Int("configured_resources", len(cfg.AuditLogging.Resources)),
		zap.Bool("redaction_enabled", cfg.AuditLogging.RedactionEnabled),
	)

	// Log each configured resource (debug level)
	for resourceType, rule := range cfg.AuditLogging.Resources {
		logger.Debug("Audit resource configured",
			zap.String("resource_type", resourceType),
			zap.Bool("enabled", rule.Enabled),
			zap.Int("actions_count", len(rule.Actions)),
			zap.Strings("actions", rule.Actions),
		)
	}

	// Connect to database
	db, err := database.Connect(cfg, logger.Get())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run database migrations if auto_migrate is enabled
	if cfg.Database.AutoMigrate {
		logger.Info("Running database migrations", zap.String("path", cfg.Database.MigrationsPath))

		migrator, err := migrate.New(migrate.Config{
			MigrationsPath: cfg.Database.MigrationsPath,
			DatabaseURL:    cfg.DatabaseURL(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create migrator: %w", err)
		}

		if err := migrator.Up(); err != nil {
			// Close migrator before returning error
			_ = migrator.Close()
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}

		// Close migrator after successful migration
		if err := migrator.Close(); err != nil {
			logger.Warn("Failed to close migrator", zap.Error(err))
		}

		logger.Info("Database migrations completed successfully")
	} else {
		logger.Info("Auto-migration disabled, skipping database migrations")
	}

	// Initialize auth components
	tokenManager, err := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to create token manager: %w", err)
	}

	// Initialize rate limiter for login attempts
	loginRateLimiter := handlers.NewRateLimiter()

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.Validator = validator.New()

	// Configure middleware
	middleware.Setup(e, cfg)

	app := &App{
		Config:           cfg,
		DB:               db,
		Echo:             e,
		TokenManager:     tokenManager,
		LoginRateLimiter: loginRateLimiter,
		Version:          version,
	}

	// Setup routes
	routes.Setup(e, cfg, db, tokenManager, loginRateLimiter, version)

	return app, nil
}

// Start starts the HTTP server on the configured host and port.
func (a *App) Start() error {
	addr := fmt.Sprintf("%s:%d", a.Config.Server.Host, a.Config.Server.Port)
	logger.Info("HTTP server starting", zap.String("addr", addr))
	return a.Echo.Start(addr)
}

// Shutdown gracefully shuts down the application.
// Closes HTTP server, database connections, and flushes logs.
func (a *App) Shutdown(timeout time.Duration) error {
	logger.Info("Server shutting down gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Shutdown HTTP server
	if err := a.Echo.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	// Close database connection
	if a.DB != nil {
		if err := a.DB.Close(); err != nil {
			logger.Error("Error closing database", zap.Error(err))
			return err
		}
	}

	// Sync logger
	if err := logger.Sync(); err != nil {
		// Ignore sync errors on stdout/stderr
		return nil
	}

	logger.Info("Server stopped")
	return nil
}
