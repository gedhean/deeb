package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/config"
	"github.com/nsxbet/deeb/internal/handlers"
	"github.com/nsxbet/deeb/internal/logger"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/repository/postgres"
	"go.uber.org/zap"
)

const version = "0.1.0"

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Initialize(logger.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting Deeb server",
		zap.String("version", version),
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
		zap.String("log_level", cfg.Logging.Level),
		zap.String("log_format", cfg.Logging.Format),
	)

	// Connect to database
	db, err := connectDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	logger.Info("Database connected successfully",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("database", cfg.Database.Database),
	)

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db)
	dbRepo := postgres.NewDatabaseRepository(db)

	// Initialize auth components
	tokenManager, err := auth.NewTokenManager(cfg.Auth.JWTSecret)
	if err != nil {
		logger.Fatal("Failed to create token manager", zap.Error(err))
	}

	// Initialize rate limiter for login attempts
	loginRateLimiter := handlers.NewRateLimiter()

	// Create Echo instance
	e := echo.New()

	// Hide Echo banner
	e.HideBanner = true

	// Configure middleware
	setupMiddleware(e, cfg)

	// Setup routes
	setupRoutes(e, cfg, db, userRepo, projectRepo, dbInstanceRepo, dbRepo, tokenManager, loginRateLimiter)

	// Start server in goroutine
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		logger.Info("HTTP server starting", zap.String("addr", addr))
		if err := e.Start(addr); err != nil {
			logger.Info("Server shutting down", zap.Error(err))
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down gracefully")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Server stopped")
}

// connectDatabase establishes database connection with configuration
func connectDatabase(cfg *config.Config) (*sqlx.DB, error) {
	// Build DSN - omit password parameter if empty to avoid PostgreSQL parsing issues
	var dsn string
	if cfg.Database.Password != "" {
		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Database,
			cfg.Database.SSLMode,
		)
	} else {
		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s dbname=%s sslmode=%s",
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.User,
			cfg.Database.Database,
			cfg.Database.SSLMode,
		)
	}

	logger.Info("Connecting to database",
		zap.String("host", cfg.Database.Host),
		zap.String("database", cfg.Database.Database),
		zap.String("user", cfg.Database.User),
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// setupMiddleware configures Echo middleware
func setupMiddleware(e *echo.Echo, cfg *config.Config) {
	// Recovery middleware - recovers from panics
	e.Use(middleware.Recover())

	// Logger middleware - logs HTTP requests
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339}","method":"${method}","uri":"${uri}",` +
			`"status":${status},"latency":"${latency_human}","bytes_in":${bytes_in},` +
			`"bytes_out":${bytes_out},"error":"${error}"}` + "\n",
	}))

	// Request ID middleware - adds correlation ID to each request
	e.Use(middleware.RequestID())

	// CORS middleware - configure cross-origin requests
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.Server.CORSOrigins,
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.PATCH, echo.OPTIONS},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
		MaxAge:           300, // 5 minutes
	}))

	// Rate limiting middleware - prevent abuse
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	// Gzip compression middleware
	e.Use(middleware.Gzip())

	// Secure middleware - adds security headers
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		ContentSecurityPolicy: "default-src 'self'",
	}))
}

// setupRoutes registers all application routes
func setupRoutes(
	e *echo.Echo,
	cfg *config.Config,
	db *sqlx.DB,
	userRepo repository.UserRepository,
	projectRepo repository.ProjectRepository,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	dbRepo repository.DatabaseRepository,
	tokenManager *auth.TokenManager,
	loginRateLimiter *handlers.RateLimiter,
) {
	// Health check endpoint (no auth required)
	healthHandler := handlers.NewHealthHandler(db, logger.Get(), version)
	e.GET("/health", healthHandler.Health)

	// API v1 routes
	v1 := e.Group("/api/v1")

	// Authentication routes (no auth required)
	authGroup := v1.Group("/auth")
	registerHandler := handlers.NewRegisterHandler(userRepo, logger.Get())
	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, loginRateLimiter, logger.Get())
	googleHandler := handlers.NewGoogleOAuthHandler(
		userRepo,
		tokenManager,
		cfg.Auth.GoogleClientID,
		cfg.Auth.GoogleClientSecret,
		cfg.Auth.CallbackURL,
		logger.Get(),
	)

	// Wrap standard HTTP handlers for Echo
	authGroup.POST("/register", echo.WrapHandler(registerHandler))
	authGroup.POST("/login", echo.WrapHandler(loginHandler))
	authGroup.GET("/google", echo.WrapHandler(googleHandler))
	authGroup.GET("/google/callback", echo.WrapHandler(http.HandlerFunc(googleHandler.HandleCallback)))

	// Protected routes (require authentication)
	// TODO: Add authentication middleware for protected routes
	// For now, routes are accessible without authentication for development

	// User management routes
	userHandler := handlers.NewUserHandler(userRepo, logger.Get())
	usersGroup := v1.Group("/users")
	usersGroup.GET("", userHandler.ListUsers)
	usersGroup.GET("/:id", userHandler.GetUser)
	usersGroup.PUT("/:id", userHandler.UpdateUser)
	usersGroup.DELETE("/:id", userHandler.DeleteUser)

	// Project management routes
	projectHandler := handlers.NewProjectHandler(projectRepo, logger.Get())
	projectsGroup := v1.Group("/projects")
	projectsGroup.POST("", projectHandler.CreateProject)
	projectsGroup.GET("", projectHandler.ListProjects)
	projectsGroup.GET("/:id", projectHandler.GetProject)
	projectsGroup.PUT("/:id", projectHandler.UpdateProject)
	projectsGroup.DELETE("/:id", projectHandler.DeleteProject)

	// Database instance management routes
	dbInstanceHandler := handlers.NewDatabaseInstanceHandler(dbInstanceRepo, logger.Get())
	dbInstancesGroup := v1.Group("/database-instances")
	dbInstancesGroup.POST("", dbInstanceHandler.CreateDatabaseInstance)
	dbInstancesGroup.GET("", dbInstanceHandler.ListDatabaseInstances)
	dbInstancesGroup.GET("/:id", dbInstanceHandler.GetDatabaseInstance)
	dbInstancesGroup.PUT("/:id", dbInstanceHandler.UpdateDatabaseInstance)
	dbInstancesGroup.DELETE("/:id", dbInstanceHandler.DeleteDatabaseInstance)

	// Database management routes
	dbHandler := handlers.NewDatabaseHandler(dbRepo, logger.Get())
	databasesGroup := v1.Group("/databases")
	databasesGroup.POST("", dbHandler.CreateDatabase)
	databasesGroup.GET("", dbHandler.ListDatabases)
	databasesGroup.GET("/:id", dbHandler.GetDatabase)
	databasesGroup.PUT("/:id", dbHandler.UpdateDatabase)
	databasesGroup.DELETE("/:id", dbHandler.DeleteDatabase)

	// Project-scoped database routes
	v1.GET("/projects/:project_id/databases", dbHandler.ListProjectDatabases)

	logger.Info("Routes configured successfully")
}
