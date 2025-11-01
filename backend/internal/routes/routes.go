// Package routes provides HTTP route setup and registration for the Deeb API.
package routes

import (
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/config"
	"github.com/nsxbet/deeb/internal/handlers"
	"github.com/nsxbet/deeb/internal/logger"
	"github.com/nsxbet/deeb/internal/middleware"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/repository/postgres"
)

// Setup registers all application routes including health checks, authentication,
// user management, projects, databases, and database instances.
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
	envRepo := postgres.NewEnvironmentRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db)
	dbRepo := postgres.NewDatabaseRepository(db)
	issueRepo := postgres.NewIssueRepository(db.DB)

	// Setup API routes
	setupHealthRoute(e, db, version)
	setupSetupRoutes(e, userRepo)
	setupAuthRoutes(e, cfg, userRepo, tokenManager, loginRateLimiter)
	setupUserRoutes(e, userRepo, tokenManager)
	setupProjectRoutes(e, projectRepo, dbRepo, tokenManager)
	setupEnvironmentRoutes(e, envRepo, db, tokenManager)
	setupDatabaseInstanceRoutes(e, dbInstanceRepo)
	setupDatabaseRoutes(e, dbRepo)
	setupIssueRoutes(e, issueRepo, tokenManager)

	// Setup Swagger documentation
	SetupSwaggerRoutes(e)

	logger.Info("Routes configured successfully")
}

// setupHealthRoute registers the health check endpoint
func setupHealthRoute(e *echo.Echo, db *sqlx.DB, version string) {
	healthHandler := handlers.NewHealthHandler(db, logger.Get(), version)
	e.GET("/health", healthHandler.Health)
}

// setupSetupRoutes registers setup and initialization routes
func setupSetupRoutes(e *echo.Echo, userRepo repository.UserRepository) {
	setupHandler := handlers.NewSetupHandler(userRepo)
	setupGroup := e.Group("/api/v1/setup")

	// Public route - no authentication required
	setupGroup.GET("/status", setupHandler.GetSetupStatus)
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
		cfg.Server.FrontendURL,
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
func setupUserRoutes(e *echo.Echo, userRepo repository.UserRepository, tokenManager *auth.TokenManager) {
	userHandler := handlers.NewUserHandler(userRepo, logger.Get())
	usersGroup := e.Group("/api/v1/users")

	// Authentication middleware required for all user routes
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddleware()

	// All user management routes require authentication and admin role
	usersGroup.GET("", userHandler.ListUsers, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersRead))
	usersGroup.GET("/:id", userHandler.GetUser, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersRead))
	usersGroup.POST("", userHandler.CreateUser, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersCreate))
	usersGroup.PUT("/:id", userHandler.UpdateUser, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersUpdate))
	usersGroup.DELETE("/:id", userHandler.DeleteUser, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersDelete))
}

// setupProjectRoutes registers project management routes
func setupProjectRoutes(e *echo.Echo, projectRepo repository.ProjectRepository, databaseRepo repository.DatabaseRepository, tokenManager *auth.TokenManager) {
	projectHandler := handlers.NewProjectHandler(projectRepo, databaseRepo, logger.Get())
	projectsGroup := e.Group("/api/v1/projects")

	// Authentication and authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddleware()

	// POST requires Admin or DBA role
	projectsGroup.POST("", projectHandler.CreateProject, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationDBA())

	// GET endpoints require authentication
	// List all projects requires admin role
	projectsGroup.GET("", projectHandler.ListProjects, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	projectsGroup.GET("/:id", projectHandler.GetProject, authMiddleware.Authenticate)

	// PUT and DELETE require admin role
	projectsGroup.PUT("/:id", projectHandler.UpdateProject, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	projectsGroup.DELETE("/:id", projectHandler.DeleteProject, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
}

// setupEnvironmentRoutes registers environment management routes
func setupEnvironmentRoutes(e *echo.Echo, envRepo repository.EnvironmentRepository, db *sqlx.DB, tokenManager *auth.TokenManager) {
	envHandler := handlers.NewEnvironmentHandler(envRepo, db, logger.Get())
	environmentsGroup := e.Group("/api/v1/environments")

	// Authentication and authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddleware()

	// All environment routes require admin role
	environmentsGroup.POST("", envHandler.CreateEnvironment, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	environmentsGroup.GET("", envHandler.ListEnvironments, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	environmentsGroup.GET("/:id", envHandler.GetEnvironment, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	environmentsGroup.PUT("/:id", envHandler.UpdateEnvironment, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	environmentsGroup.DELETE("/:id", envHandler.DeleteEnvironment, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
}

// setupPolicyRoutes registers policy management routes
func setupPolicyRoutes(e *echo.Echo, policyRepo repository.PolicyRepository, envRepo repository.EnvironmentRepository, tokenManager *auth.TokenManager) {
	policyHandler := handlers.NewPolicyHandler(policyRepo, envRepo, logger.Get())
	policiesGroup := e.Group("/api/v1/policies")

	// Authentication and authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddleware()

	// All policy routes require admin role
	policiesGroup.POST("", policyHandler.CreatePolicy, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	policiesGroup.GET("", policyHandler.ListPolicies, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	policiesGroup.GET("/:id", policyHandler.GetPolicy, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	policiesGroup.PUT("/:id", policyHandler.UpdatePolicy, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	policiesGroup.DELETE("/:id", policyHandler.DeletePolicy, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
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

// setupIssueRoutes registers issue management routes
func setupIssueRoutes(e *echo.Echo, issueRepo repository.IssueRepository, tokenManager *auth.TokenManager) {
	issueHandler := handlers.NewIssueHandler(issueRepo, logger.Get())
	issuesGroup := e.Group("/api/v1/issues")

	// Authentication middleware for protected routes
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)

	// All issue routes require authentication
	issuesGroup.POST("", issueHandler.CreateIssue, authMiddleware.Authenticate)
	issuesGroup.GET("/:id", issueHandler.GetIssue, authMiddleware.Authenticate)
	issuesGroup.GET("", issueHandler.ListIssues, authMiddleware.Authenticate)
	issuesGroup.PUT("/:id", issueHandler.UpdateIssue, authMiddleware.Authenticate)
}
