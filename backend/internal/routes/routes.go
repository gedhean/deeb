// Package routes provides HTTP route setup and registration for the Deeb API.
package routes

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/config"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/handlers"
	"github.com/nsxbet/deeb/internal/logger"
	"github.com/nsxbet/deeb/internal/middleware"
	"github.com/nsxbet/deeb/internal/rbac"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/repository/postgres"
	"github.com/nsxbet/deeb/internal/review"
	"github.com/nsxbet/deeb/internal/services"
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
	changePlanRepo := postgres.NewChangePlanRepository(db)
	changeHistoryRepo := postgres.NewChangeHistoryRepository(db)
	projectMemberRepo := postgres.NewProjectMemberRepository(db)
	policyFindingRepo := postgres.NewPolicyFindingRepository(db)
	approvalRepo := postgres.NewApprovalRepository(db)

	// Initialize audit log repository and service
	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditLogService := services.NewAuditLogService(auditLogRepo, logger.Get())

	// Initialize connection manager
	connectionMgr := database.NewConnectionManager(cfg.Security.EncryptionKey, logger.Get())

	// Setup API routes
	setupHealthRoute(e, db, version)
	setupSetupRoutes(e, userRepo)
	setupAuthRoutes(e, cfg, userRepo, tokenManager, loginRateLimiter, auditLogService)
	setupUserRoutes(e, userRepo, tokenManager)
	setupProjectRoutes(e, projectRepo, dbRepo, projectMemberRepo, userRepo, tokenManager, auditLogService)
	setupEnvironmentRoutes(e, envRepo, db, tokenManager, auditLogService)
	setupPolicyRoutes(e, policyRepo, envRepo, tokenManager)
	setupDatabaseInstanceRoutes(
		e,
		dbInstanceRepo,
		dbRepo,
		projectRepo,
		envRepo,
		changePlanRepo,
		connectionMgr,
		cfg.Security.EncryptionKey,
		tokenManager,
		auditLogService,
	)
	setupDatabaseRoutes(
		e,
		dbRepo,
		projectRepo,
		envRepo,
		dbInstanceRepo,
		changePlanRepo,
		projectMemberRepo,
		tokenManager,
		auditLogService,
	)
	setupChangePlanRoutes(
		e,
		changePlanRepo,
		changeHistoryRepo,
		projectRepo,
		dbRepo,
		envRepo,
		dbInstanceRepo,
		projectMemberRepo,
		policyFindingRepo,
		policyRepo,
		approvalRepo,
		db,
		connectionMgr,
		cfg.Security.EncryptionKey,
		tokenManager,
		auditLogService,
	)
	setupAuditLogRoutes(e, auditLogRepo, tokenManager)

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
	auditLogService *services.AuditLogService,
) {
	authGroup := e.Group("/api/v1/auth")

	// Initialize handlers
	registerHandler := handlers.NewRegisterHandler(userRepo, auditLogService, logger.Get())
	loginHandler := handlers.NewLoginHandler(userRepo, tokenManager, loginRateLimiter, auditLogService, logger.Get())
	logoutHandler := handlers.NewLogoutHandler(auditLogService, logger.Get())
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
		// Authorization check is in handler - allows users to view own profile
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
		// Authorization check is in handler - allows users to update own profile
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
	auditLogService *services.AuditLogService,
) {
	projectHandler := handlers.NewProjectHandler(
		projectRepo,
		databaseRepo,
		projectMemberRepo,
		userRepo,
		auditLogService,
		logger.Get(),
	)
	projectsGroup := e.Group("/api/v1/projects")

	// Authentication and authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddlewareWithRepo(projectMemberRepo)

	// POST requires project.create permission
	projectsGroup.POST(
		"",
		projectHandler.CreateProject,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionProjectsCreate),
	)

	// GET endpoints require authentication
	// List all projects requires project.read permission
	projectsGroup.GET("", projectHandler.ListProjects, authMiddleware.Authenticate)
	projectsGroup.GET(
		"/:id",
		projectHandler.GetProject,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionProjectsRead),
	)

	// PUT and DELETE require project.update permission
	projectsGroup.PUT(
		"/:id",
		projectHandler.UpdateProject,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionProjectsUpdate),
	)
	projectsGroup.DELETE(
		"/:id",
		projectHandler.DeleteProject,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionProjectsDelete),
	)

	// Project member routes require authentication and project-level permissions
	// POST requires project.members.create permission
	projectsGroup.POST(
		"/:id/members",
		projectHandler.AddProjectMember,
		authMiddleware.Authenticate,
		authzMiddleware.RequireProjectPermission(rbac.PermissionProjectMembersCreate),
	)
	// GET requires project.members.read permission
	projectsGroup.GET(
		"/:id/members",
		projectHandler.ListProjectMembers,
		authMiddleware.Authenticate,
		authzMiddleware.RequireProjectPermission(rbac.PermissionProjectMembersRead),
	)
	// DELETE requires project.members.delete permission
	projectsGroup.DELETE(
		"/:id/members/:user_id",
		projectHandler.RemoveProjectMember,
		authMiddleware.Authenticate,
		authzMiddleware.RequireProjectPermission(rbac.PermissionProjectMembersDelete),
	)
	// PUT requires project.members.update permission
	projectsGroup.PUT(
		"/:id/members/:user_id",
		projectHandler.UpdateProjectMemberRole,
		authMiddleware.Authenticate,
		authzMiddleware.RequireProjectPermission(rbac.PermissionProjectMembersUpdate),
	)
}

// setupEnvironmentRoutes registers environment management routes
func setupEnvironmentRoutes(
	e *echo.Echo,
	envRepo repository.EnvironmentRepository,
	db *sqlx.DB,
	tokenManager *auth.TokenManager,
	auditLogService *services.AuditLogService,
) {
	envHandler := handlers.NewEnvironmentHandler(envRepo, db, auditLogService, logger.Get())
	environmentsGroup := e.Group("/api/v1/environments")

	// Authentication and authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddleware()

	// All environment routes require environments.create permission
	environmentsGroup.POST(
		"",
		envHandler.CreateEnvironment,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionEnvironmentsCreate),
	)
	// GET requires environments.read permission
	environmentsGroup.GET(
		"",
		envHandler.ListEnvironments,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionEnvironmentsRead),
	)
	// GET requires environments.read permission
	environmentsGroup.GET(
		"/:id",
		envHandler.GetEnvironment,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionEnvironmentsRead),
	)
	// PUT requires environments.update permission
	environmentsGroup.PUT(
		"/:id",
		envHandler.UpdateEnvironment,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionEnvironmentsUpdate),
	)
	// DELETE requires environments.delete permission
	environmentsGroup.DELETE(
		"/:id",
		envHandler.DeleteEnvironment,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionEnvironmentsDelete),
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
	// SQL Review rules endpoint - accessible to all authenticated users
	policiesGroup.GET("/sql-review/rules", policyHandler.GetSQLReviewRules, authMiddleware.Authenticate)
}

// setupDatabaseInstanceRoutes registers database instance management routes
func setupDatabaseInstanceRoutes(
	e *echo.Echo,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	projectRepo repository.ProjectRepository,
	envRepo repository.EnvironmentRepository,
	changePlanRepo repository.ChangePlanRepository,
	connManager *database.ConnectionManager,
	encryptionKey string,
	tokenManager *auth.TokenManager,
	auditLogService *services.AuditLogService,
) {
	dbInstanceHandler := handlers.NewDatabaseInstanceHandler(
		dbInstanceRepo,
		databaseRepo,
		projectRepo,
		envRepo,
		changePlanRepo,
		connManager,
		encryptionKey,
		auditLogService,
		logger.Get(),
	)
	dbInstancesGroup := e.Group("/api/v1/database-instances")

	// Authentication and authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddleware()

	// POST requires database_instances.create permission
	dbInstancesGroup.POST(
		"",
		dbInstanceHandler.CreateDatabaseInstance,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabaseInstancesCreate),
	)
	// GET requires database_instances.read permission
	dbInstancesGroup.GET(
		"",
		dbInstanceHandler.ListDatabaseInstances,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
	// GET requires database_instances.read permission
	dbInstancesGroup.GET(
		"/:id",
		dbInstanceHandler.GetDatabaseInstance,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
	// PUT requires database_instances.update permission
	dbInstancesGroup.PUT(
		"/:id",
		dbInstanceHandler.UpdateDatabaseInstance,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabaseInstancesUpdate),
	)
	// DELETE requires database_instances.delete permission
	dbInstancesGroup.DELETE(
		"/:id",
		dbInstanceHandler.DeleteDatabaseInstance,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabaseInstancesDelete),
	)
	// POST requires databases.create permission
	dbInstancesGroup.POST(
		"/:id/databases",
		dbInstanceHandler.CreateDatabaseForInstance,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesCreate),
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
	auditLogService *services.AuditLogService,
) {
	dbHandler := handlers.NewDatabaseHandler(
		dbRepo,
		projectRepo,
		envRepo,
		dbInstanceRepo,
		changePlanRepo,
		auditLogService,
		logger.Get(),
	)
	databasesGroup := e.Group("/api/v1/databases")

	// Authentication and authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddlewareWithRepo(projectMemberRepo)

	// POST requires admin role
	databasesGroup.POST(
		"",
		dbHandler.CreateDatabase,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesCreate),
	)
	// GET endpoints require authentication and read permission
	databasesGroup.GET(
		"",
		dbHandler.ListDatabases,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
	databasesGroup.GET(
		"/:id",
		dbHandler.GetDatabase,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
	// PUT and DELETE require admin role
	databasesGroup.PUT(
		"/:id",
		dbHandler.UpdateDatabase,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesUpdate),
	)
	databasesGroup.DELETE(
		"/:id",
		dbHandler.DeleteDatabase,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesDelete),
	)

	// Project-scoped database routes require organization-level read permission
	// (Organization admins, DBAs, and Developers have PermissionDatabasesRead)
	e.GET(
		"/api/v1/projects/:project_id/databases",
		dbHandler.ListProjectDatabases,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
}

// setupChangePlanRoutes registers change plan management routes
func setupChangePlanRoutes(
	e *echo.Echo,
	changePlanRepo repository.ChangePlanRepository,
	changeHistoryRepo repository.ChangeHistoryRepository,
	projectRepo repository.ProjectRepository,
	databaseRepo repository.DatabaseRepository,
	environmentRepo repository.EnvironmentRepository,
	databaseInstanceRepo repository.DatabaseInstanceRepository,
	projectMemberRepo repository.ProjectMemberRepository,
	policyFindingRepo repository.PolicyFindingRepository,
	policyRepo repository.PolicyRepository,
	approvalRepo repository.ApprovalRepository,
	db *sqlx.DB,
	connectionMgr *database.ConnectionManager,
	encryptionKey string,
	tokenManager *auth.TokenManager,
	auditLogService *services.AuditLogService,
) {
	// Initialize risk assessment service
	riskAssessmentSvc := services.NewRiskAssessmentService(
		databaseRepo,
		environmentRepo,
		databaseInstanceRepo,
		logger.Get(),
	)

	// Initialize SQL Review service
	schemaCache := review.NewSchemaCache(15*time.Minute, logger.Get())
	reviewService := review.NewReviewService(
		policyRepo,
		environmentRepo,
		schemaCache,
		logger.Get(),
	)

	// Initialize approval service
	rbacChecker := rbac.NewCheckerService()
	approvalService := services.NewApprovalService(
		approvalRepo,
		changePlanRepo,
		policyFindingRepo,
		projectMemberRepo,
		databaseRepo,
		environmentRepo,
		rbacChecker,
		logger.Get(),
	)

	// Initialize schema snapshot repository
	snapshotRepo := postgres.NewSchemaSnapshotRepository(db)

	// Initialize schema snapshot service
	snapshotService := services.NewSchemaSnapshotService(
		snapshotRepo,
		databaseInstanceRepo,
		databaseRepo,
		connectionMgr,
		encryptionKey,
		logger.Get(),
	)

	// Initialize execution lock manager
	lockManager := services.NewExecutionLockManager(db, logger.Get())

	// Initialize execution queue repository
	queueRepo := postgres.NewExecutionQueueRepository(db)

	// Initialize execution queue service
	queueService := services.NewExecutionQueueService(queueRepo, logger.Get())

	// Initialize execution log repository
	executionLogRepo := postgres.NewExecutionLogRepository(db)

	// Initialize execution service
	executionService := services.NewExecutionService(
		changePlanRepo,
		databaseInstanceRepo,
		databaseRepo,
		snapshotService,
		lockManager,
		queueService,
		executionLogRepo,
		connectionMgr,
		encryptionKey,
		logger.Get(),
	)

	// Initialize queue processor (background worker)
	queueProcessor := services.NewQueueProcessor(queueRepo, executionService, logger.Get())
	// Start the queue processor in background
	ctx := context.Background()
	queueProcessor.Start(ctx)

	changePlanHandler := handlers.NewChangePlanHandler(
		changePlanRepo,
		changeHistoryRepo,
		projectRepo,
		databaseRepo,
		databaseInstanceRepo,
		projectMemberRepo,
		policyFindingRepo,
		approvalRepo,
		environmentRepo,
		riskAssessmentSvc,
		reviewService,
		approvalService,
		executionService,
		auditLogService,
		logger.Get(),
	)
	changePlansGroup := e.Group("/api/v1/change-plans")

	// Authentication and authorization middleware
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddlewareWithRepo(projectMemberRepo)
	changePlanExtractor := middleware.NewChangePlanProjectExtractor(changePlanRepo)

	// Status transition validation middleware
	statusTransitionMiddleware := middleware.NewChangePlanStatusTransitionMiddleware(changePlanRepo)

	// Execution auth middleware (special permission logic for execution)
	executionAuthMiddleware := middleware.NewExecutionAuthMiddleware(changePlanRepo, projectMemberRepo)

	// CREATE change plan - requires project-level changes.create permission
	changePlansGroup.POST("",
		changePlanHandler.CreateChangePlan,
		authMiddleware.Authenticate,
		middleware.ExtractProjectFromBody(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesCreate),
	)

	// GET change plan by ID - requires changes.read permission
	changePlansGroup.GET("/:id",
		changePlanHandler.GetChangePlan,
		authMiddleware.Authenticate,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesRead),
	)

	// GET approval history - requires changes.read permission
	changePlansGroup.GET("/:id/approvals",
		changePlanHandler.GetApprovalHistory,
		authMiddleware.Authenticate,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesRead),
	)

	// LIST change plans - no project-level check (filtered by handler based on user's projects)
	changePlansGroup.GET("",
		changePlanHandler.ListChangePlans,
		authMiddleware.Authenticate,
	)

	// UPDATE change plan - requires changes.update permission
	changePlansGroup.PUT("/:id",
		changePlanHandler.UpdateChangePlan,
		authMiddleware.Authenticate,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesUpdate),
		statusTransitionMiddleware.ValidateStatusTransition,
	)

	// ASSIGN reviewer - requires changes.update permission
	changePlansGroup.PUT("/:id/reviewer",
		changePlanHandler.AssignReviewer,
		authMiddleware.Authenticate,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesUpdate),
	)

	// DELETE change plan - requires changes.delete permission
	changePlansGroup.DELETE("/:id",
		changePlanHandler.DeleteChangePlan,
		authMiddleware.Authenticate,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesDelete),
	)

	// TRIGGER SQL review - requires changes.create permission
	changePlansGroup.POST("/:id/review",
		changePlanHandler.TriggerSQLReview,
		authMiddleware.Authenticate,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesCreate),
	)

	// GET review findings - requires changes.read permission
	changePlansGroup.GET("/:id/review/findings",
		changePlanHandler.GetReviewFindings,
		authMiddleware.Authenticate,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesRead),
	)

	// APPROVE change plan - requires changes.approve permission
	changePlansGroup.POST("/:id/approve",
		changePlanHandler.ApproveChangePlan,
		authMiddleware.Authenticate,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesApprove),
	)

	// CLOSE change plan - requires changes.update permission
	changePlansGroup.POST("/:id/close",
		changePlanHandler.CloseChangePlan,
		authMiddleware.Authenticate,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesUpdate),
	)

	// EXECUTE change plan - uses special execution middleware with complex permission logic
	changePlansGroup.POST("/:id/execute",
		changePlanHandler.ExecuteChangePlan,
		authMiddleware.Authenticate,
		executionAuthMiddleware.RequireExecutionPermission(),
	)

	// GET queue status - requires changes.read permission
	changePlansGroup.GET("/:id/queue-status",
		changePlanHandler.GetQueueStatus,
		authMiddleware.Authenticate,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesRead),
	)

	// GET execution logs - requires changes.read permission
	changePlansGroup.GET("/:id/execution-logs",
		changePlanHandler.GetExecutionLogHistory,
		authMiddleware.Authenticate,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesRead),
	)

	// RESUBMIT change plan - requires changes.update permission
	changePlansGroup.POST("/:id/resubmit",
		changePlanHandler.ResubmitChangePlan,
		authMiddleware.Authenticate,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesUpdate),
	)
}

// setupAuditLogRoutes registers audit log querying routes
func setupAuditLogRoutes(
	e *echo.Echo,
	auditLogRepo repository.AuditLogRepository,
	tokenManager *auth.TokenManager,
) {
	auditLogHandler := handlers.NewAuditLogHandler(auditLogRepo, logger.Get())
	auditLogsGroup := e.Group("/api/v1/audit-logs")

	// Authentication middleware for protected routes
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	authzMiddleware := middleware.NewAuthzMiddleware()

	// All audit log routes require authentication and audit_logs.read permission
	auditLogsGroup.GET(
		"",
		auditLogHandler.ListAuditLogs,
		authMiddleware.Authenticate,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionAuditLogsRead),
	)
}
