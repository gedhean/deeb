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
	"github.com/nsxbet/deeb/internal/review"
	"github.com/nsxbet/deeb/internal/review/validators"
	"github.com/nsxbet/deeb/internal/services"
	"go.uber.org/zap"
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
	policyRepo := postgres.NewPolicyRepository(db)
	dbInstanceRepo := postgres.NewDatabaseInstanceRepository(db, cfg.Security.EncryptionKey)
	dbRepo := postgres.NewDatabaseRepository(db)
	issueRepo := postgres.NewIssueRepository(db.DB)
	changePlanRepo := postgres.NewChangePlanRepository(db)
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)

	// Setup API routes
	setupHealthRoute(e, db, version)
	setupSetupRoutes(e, userRepo)
	setupAuthRoutes(e, cfg, userRepo, tokenManager, loginRateLimiter)
	setupUserRoutes(e, userRepo, tokenManager)
	setupProjectRoutes(e, projectRepo, dbRepo, projectMemberRepo, userRepo, tokenManager)
	setupEnvironmentRoutes(e, envRepo, db, tokenManager)
	setupPolicyRoutes(e, policyRepo, envRepo, tokenManager)
	setupDatabaseInstanceRoutes(e, dbInstanceRepo, dbRepo, cfg.Security.EncryptionKey, tokenManager)
	setupDatabaseRoutes(e, dbRepo, projectRepo, envRepo, dbInstanceRepo, changePlanRepo, projectMemberRepo, tokenManager)
	setupIssueRoutes(e, issueRepo, tokenManager)
	setupChangePlanRoutes(e, changePlanRepo, projectRepo, dbRepo, envRepo, dbInstanceRepo, projectMemberRepo, policyFindingRepo, policyRepo, tokenManager)

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
	usersGroup.GET(
		"",
		userHandler.ListUsers,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersRead),
	)
	usersGroup.GET(
		"/:id",
		userHandler.GetUser,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersRead),
	)
	usersGroup.POST(
		"",
		userHandler.CreateUser,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersCreate),
	)
	usersGroup.PUT(
		"/:id",
		userHandler.UpdateUser,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersUpdate),
	)
	usersGroup.DELETE(
		"/:id",
		userHandler.DeleteUser,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersDelete),
	)
}

// setupProjectRoutes registers project management routes
func setupProjectRoutes(
	e *echo.Echo,
	projectRepo repository.ProjectRepository,
	databaseRepo repository.DatabaseRepository,
	projectMemberRepo repository.ProjectMemberRepository,
	userRepo repository.UserRepository,
	tokenManager *auth.TokenManager,
) {
	projectHandler := handlers.NewProjectHandler(projectRepo, databaseRepo, projectMemberRepo, userRepo, logger.Get())
	projectsGroup := e.Group("/api/v1/projects")

	// Authentication and authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddlewareWithRepo(projectMemberRepo)

	// POST requires Admin or DBA role
	projectsGroup.POST("", projectHandler.CreateProject, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationDBA())

	// GET endpoints require authentication
	// List all projects requires admin role
	projectsGroup.GET("", projectHandler.ListProjects, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	projectsGroup.GET("/:id", projectHandler.GetProject, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationPermission(rbac.PermissionProjectsRead))

	// PUT and DELETE require admin role
	projectsGroup.PUT(
		"/:id",
		projectHandler.UpdateProject,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationAdmin(),
	)
	projectsGroup.DELETE(
		"/:id",
		projectHandler.DeleteProject,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationAdmin(),
	)

	// Project member routes require authentication and project-level permissions
	projectsGroup.POST("/:id/members", projectHandler.AddProjectMember, authMiddleware.Authenticate, authzMiddleware.RequireProjectPermission(rbac.PermissionProjectMembersCreate))
	projectsGroup.GET("/:id/members", projectHandler.ListProjectMembers, authMiddleware.Authenticate, authzMiddleware.RequireProjectPermission(rbac.PermissionProjectMembersRead))
	projectsGroup.DELETE("/:id/members/:user_id", projectHandler.RemoveProjectMember, authMiddleware.Authenticate, authzMiddleware.RequireProjectPermission(rbac.PermissionProjectMembersDelete))
	projectsGroup.PUT("/:id/members/:user_id", projectHandler.UpdateProjectMemberRole, authMiddleware.Authenticate, authzMiddleware.RequireProjectPermission(rbac.PermissionProjectMembersUpdate))
}

// setupEnvironmentRoutes registers environment management routes
func setupEnvironmentRoutes(
	e *echo.Echo,
	envRepo repository.EnvironmentRepository,
	db *sqlx.DB,
	tokenManager *auth.TokenManager,
) {
	envHandler := handlers.NewEnvironmentHandler(envRepo, db, logger.Get())
	environmentsGroup := e.Group("/api/v1/environments")

	// Authentication and authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddleware()

	// All environment routes require admin role
	environmentsGroup.POST(
		"",
		envHandler.CreateEnvironment,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationAdmin(),
	)
	environmentsGroup.GET(
		"",
		envHandler.ListEnvironments,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationAdmin(),
	)
	environmentsGroup.GET(
		"/:id",
		envHandler.GetEnvironment,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationAdmin(),
	)
	environmentsGroup.PUT(
		"/:id",
		envHandler.UpdateEnvironment,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationAdmin(),
	)
	environmentsGroup.DELETE(
		"/:id",
		envHandler.DeleteEnvironment,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationAdmin(),
	)
}

// setupPolicyRoutes registers policy management routes
func setupPolicyRoutes(
	e *echo.Echo,
	policyRepo repository.PolicyRepository,
	envRepo repository.EnvironmentRepository,
	tokenManager *auth.TokenManager,
) {
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
	policiesGroup.DELETE(
		"/:id",
		policyHandler.DeletePolicy,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationAdmin(),
	)
}

// setupDatabaseInstanceRoutes registers database instance management routes
func setupDatabaseInstanceRoutes(
	e *echo.Echo,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	encryptionKey string,
	tokenManager *auth.TokenManager,
) {
	dbInstanceHandler := handlers.NewDatabaseInstanceHandler(dbInstanceRepo, databaseRepo, encryptionKey, logger.Get())
	dbInstancesGroup := e.Group("/api/v1/database-instances")

	// Authentication and authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddleware()

	dbInstancesGroup.POST(
		"",
		dbInstanceHandler.CreateDatabaseInstance,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationAdmin(),
	)
	dbInstancesGroup.GET("", dbInstanceHandler.ListDatabaseInstances, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead))
	dbInstancesGroup.GET("/:id", dbInstanceHandler.GetDatabaseInstance, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead))
	dbInstancesGroup.PUT(
		"/:id",
		dbInstanceHandler.UpdateDatabaseInstance,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationAdmin(),
	)
	dbInstancesGroup.DELETE(
		"/:id",
		dbInstanceHandler.DeleteDatabaseInstance,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationAdmin(),
	)
}

// setupDatabaseRoutes registers database management routes
func setupDatabaseRoutes(
	e *echo.Echo,
	dbRepo repository.DatabaseRepository,
	projectRepo repository.ProjectRepository,
	envRepo repository.EnvironmentRepository,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	changePlanRepo repository.ChangePlanRepository,
	projectMemberRepo repository.ProjectMemberRepository,
	tokenManager *auth.TokenManager,
) {
	dbHandler := handlers.NewDatabaseHandler(dbRepo, projectRepo, envRepo, dbInstanceRepo, changePlanRepo, logger.Get())
	databasesGroup := e.Group("/api/v1/databases")

	// Authentication and authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddlewareWithRepo(projectMemberRepo)

	// POST requires admin role
	databasesGroup.POST("", dbHandler.CreateDatabase, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	// GET endpoints require authentication and read permission
	databasesGroup.GET("", dbHandler.ListDatabases, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead))
	databasesGroup.GET("/:id", dbHandler.GetDatabase, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead))
	// PUT and DELETE require admin role
	databasesGroup.PUT("/:id", dbHandler.UpdateDatabase, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())
	databasesGroup.DELETE("/:id", dbHandler.DeleteDatabase, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationAdmin())

	// Project-scoped database routes require organization-level read permission
	// (Organization admins, DBAs, and Developers have PermissionDatabasesRead)
	e.GET("/api/v1/projects/:project_id/databases", dbHandler.ListProjectDatabases, authMiddleware.Authenticate, authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead))
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

// setupChangePlanRoutes registers change plan management routes
func setupChangePlanRoutes(
	e *echo.Echo,
	changePlanRepo repository.ChangePlanRepository,
	projectRepo repository.ProjectRepository,
	databaseRepo repository.DatabaseRepository,
	environmentRepo repository.EnvironmentRepository,
	databaseInstanceRepo repository.DatabaseInstanceRepository,
	projectMemberRepo repository.ProjectMemberRepository,
	policyFindingRepo repository.PolicyFindingRepository,
	policyRepo repository.PolicyRepository,
	tokenManager *auth.TokenManager,
) {
	// Initialize risk assessment service
	riskAssessmentSvc := services.NewRiskAssessmentService(
		databaseRepo,
		environmentRepo,
		databaseInstanceRepo,
		logger.Get(),
	)

	// Initialize SQL Review engine and register validators
	reviewEngine := review.NewEngine()
	reviewService := review.NewReviewService(
		policyRepo,
		databaseRepo,
		reviewEngine,
		logger.Get(),
	)

	// Register all validators
	if err := reviewService.RegisterValidator(validators.NewPostgreSQLSyntaxValidator()); err != nil {
		logger.Get().Warn("Failed to register PostgreSQL syntax validator", zap.Error(err))
	}
	if err := reviewService.RegisterValidator(validators.NewMySQLSyntaxValidator()); err != nil {
		logger.Get().Warn("Failed to register MySQL syntax validator", zap.Error(err))
	}
	if err := reviewService.RegisterValidator(validators.NewSecurityValidator()); err != nil {
		logger.Get().Warn("Failed to register security validator", zap.Error(err))
	}
	if err := reviewService.RegisterValidator(validators.NewPerformanceValidator()); err != nil {
		logger.Get().Warn("Failed to register performance validator", zap.Error(err))
	}
	if err := reviewService.RegisterValidator(validators.NewBackwardsCompatibilityValidator()); err != nil {
		logger.Get().Warn("Failed to register backwards compatibility validator", zap.Error(err))
	}
	if err := reviewService.RegisterValidator(validators.NewBestPracticesValidator()); err != nil {
		logger.Get().Warn("Failed to register best practices validator", zap.Error(err))
	}

	changePlanHandler := handlers.NewChangePlanHandler(
		changePlanRepo,
		projectRepo,
		databaseRepo,
		databaseInstanceRepo,
		projectMemberRepo,
		policyFindingRepo,
		riskAssessmentSvc,
		reviewService,
		logger.Get(),
	)
	changePlansGroup := e.Group("/api/v1/change-plans")

	// Authentication middleware for protected routes
	// RBAC checks are done in handlers since project_id is in request body
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)

	// All change plan routes require authentication
	changePlansGroup.POST("", changePlanHandler.CreateChangePlan, authMiddleware.Authenticate)
	changePlansGroup.GET("/:id", changePlanHandler.GetChangePlan, authMiddleware.Authenticate)
	changePlansGroup.GET("", changePlanHandler.ListChangePlans, authMiddleware.Authenticate)
	changePlansGroup.PUT("/:id", changePlanHandler.UpdateChangePlan, authMiddleware.Authenticate)
	changePlansGroup.DELETE("/:id", changePlanHandler.DeleteChangePlan, authMiddleware.Authenticate)
	changePlansGroup.POST("/:id/review", changePlanHandler.TriggerSQLReview, authMiddleware.Authenticate)
}
