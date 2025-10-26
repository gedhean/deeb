package routes

import (
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/config"
	"github.com/nsxbet/deeb/internal/handlers"
	"github.com/nsxbet/deeb/internal/logger"
	"github.com/nsxbet/deeb/internal/middleware"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/repository/postgres"
)

// Setup registers all application routes
func Setup(
	e *echo.Echo,
	cfg *config.Config,
	db *sqlx.DB,
	tokenManager *auth.TokenManager,
	loginRateLimiter *handlers.RateLimiter,
	version string,
) {
	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)
	projectRepo := postgres.NewProjectRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db)
	dbRepo := postgres.NewDatabaseRepository(db)

	// Setup API routes
	setupHealthRoute(e, db, version)
	setupAuthRoutes(e, cfg, userRepo, tokenManager, loginRateLimiter)
	setupUserRoutes(e, userRepo)
	setupProjectRoutes(e, projectRepo)
	setupDatabaseInstanceRoutes(e, dbInstanceRepo)
	setupDatabaseRoutes(e, dbRepo)

	// Setup Swagger documentation
	SetupSwaggerRoutes(e)

	logger.Info("Routes configured successfully")
}

// setupHealthRoute registers the health check endpoint
func setupHealthRoute(e *echo.Echo, db *sqlx.DB, version string) {
	healthHandler := handlers.NewHealthHandler(db, logger.Get(), version)
	e.GET("/health", healthHandler.Health)
}

// setupAuthRoutes registers authentication routes
func setupAuthRoutes(
	e *echo.Echo,
	cfg *config.Config,
	userRepo repository.UserRepository,
	tokenManager *auth.TokenManager,
	loginRateLimiter *handlers.RateLimiter,
) {
	authGroup := e.Group("/api/v1/auth")

	// Initialize handlers
	registerHandler := handlers.NewRegisterHandler(userRepo, logger.Get())
	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, loginRateLimiter, logger.Get())
	logoutHandler := handlers.NewLogoutHandler(logger.Get())
	meHandler := handlers.NewMeHandler(userRepo, logger.Get())
	googleHandler := handlers.NewGoogleOAuthHandler(
		userRepo,
		tokenManager,
		cfg.Auth.GoogleClientID,
		cfg.Auth.GoogleClientSecret,
		cfg.Auth.CallbackURL,
		logger.Get(),
	)

	// Register routes - use Echo handlers directly
	authGroup.POST("/register", registerHandler.Register)
	authGroup.POST("/login", loginHandler.Login)
	authGroup.POST("/logout", logoutHandler.Logout)
	authGroup.GET("/google", googleHandler.InitiateGoogleAuth)
	authGroup.GET("/google/callback", googleHandler.GoogleCallback)

	// Protected routes - require authentication
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authGroup.GET("/me", meHandler.GetCurrentUser, authMiddleware.Authenticate)
}

// setupUserRoutes registers user management routes
func setupUserRoutes(e *echo.Echo, userRepo repository.UserRepository) {
	userHandler := handlers.NewUserHandler(userRepo, logger.Get())
	usersGroup := e.Group("/api/v1/users")

	// TODO: Add authentication middleware for protected routes
	// For now, routes are accessible without authentication for development
	usersGroup.GET("", userHandler.ListUsers)
	usersGroup.GET("/:id", userHandler.GetUser)
	usersGroup.PUT("/:id", userHandler.UpdateUser)
	usersGroup.DELETE("/:id", userHandler.DeleteUser)
}

// setupProjectRoutes registers project management routes
func setupProjectRoutes(e *echo.Echo, projectRepo repository.ProjectRepository) {
	projectHandler := handlers.NewProjectHandler(projectRepo, logger.Get())
	projectsGroup := e.Group("/api/v1/projects")

	projectsGroup.POST("", projectHandler.CreateProject)
	projectsGroup.GET("", projectHandler.ListProjects)
	projectsGroup.GET("/:id", projectHandler.GetProject)
	projectsGroup.PUT("/:id", projectHandler.UpdateProject)
	projectsGroup.DELETE("/:id", projectHandler.DeleteProject)
}

// setupDatabaseInstanceRoutes registers database instance management routes
func setupDatabaseInstanceRoutes(e *echo.Echo, dbInstanceRepo repository.DatabaseInstanceRepository) {
	dbInstanceHandler := handlers.NewDatabaseInstanceHandler(dbInstanceRepo, logger.Get())
	dbInstancesGroup := e.Group("/api/v1/database-instances")

	dbInstancesGroup.POST("", dbInstanceHandler.CreateDatabaseInstance)
	dbInstancesGroup.GET("", dbInstanceHandler.ListDatabaseInstances)
	dbInstancesGroup.GET("/:id", dbInstanceHandler.GetDatabaseInstance)
	dbInstancesGroup.PUT("/:id", dbInstanceHandler.UpdateDatabaseInstance)
	dbInstancesGroup.DELETE("/:id", dbInstanceHandler.DeleteDatabaseInstance)
}

// setupDatabaseRoutes registers database management routes
func setupDatabaseRoutes(e *echo.Echo, dbRepo repository.DatabaseRepository) {
	dbHandler := handlers.NewDatabaseHandler(dbRepo, logger.Get())
	databasesGroup := e.Group("/api/v1/databases")

	databasesGroup.POST("", dbHandler.CreateDatabase)
	databasesGroup.GET("", dbHandler.ListDatabases)
	databasesGroup.GET("/:id", dbHandler.GetDatabase)
	databasesGroup.PUT("/:id", dbHandler.UpdateDatabase)
	databasesGroup.DELETE("/:id", dbHandler.DeleteDatabase)

	// Project-scoped database routes
	e.GET("/api/v1/projects/:project_id/databases", dbHandler.ListProjectDatabases)
}
