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
	changelogRepo := postgres.NewChangelogRepository(db)
	schemaHistoryRepo := postgres.NewSchemaHistoryRepository(db)

	// Initialize audit log repository and service
	auditLogRepo := postgres.NewAuditLogRepository(db)
	auditLogService := services.NewAuditLogService(auditLogRepo, logger.Get())

	// Create protected route group with authentication and audit logging
	authMiddleware := middleware.NewAuthMiddleware(tokenManager)
	auditLogMiddleware := middleware.NewAuditLogMiddleware(auditLogService, logger.Get(), &cfg.AuditLogging)

	// Protected routes group - applies auth and audit middleware to all routes under /api/v1
	authenticated := e.Group("/api/v1")
	authenticated.Use(authMiddleware.Authenticate)
	authenticated.Use(auditLogMiddleware.AuditLog)

	// Initialize connection manager
	connectionMgr := database.NewConnectionManager(cfg.Security.EncryptionKey, logger.Get())

	// Initialize database schema repository
	dbSchemaRepo := postgres.NewDatabaseSchemaRepository(db)

	// Initialize schema metadata service
	schemaMetadataService := services.NewSchemaMetadataService(
		dbInstanceRepo,
		dbRepo,
		dbSchemaRepo,
		connectionMgr,
		logger.Get(),
	)

	// Initialize and start schema sync background job
	schemaSyncJob := services.NewSchemaSyncJob(
		dbRepo,
		dbSchemaRepo,
		schemaMetadataService,
		cfg.BackgroundJobs.SchemaSyncInterval,
		logger.Get(),
	)
	// Start the schema sync job in background
	ctx := context.Background()
	schemaSyncJob.Start(ctx)

	// Setup API routes
	// Public routes (no authentication required)
	setupHealthRoute(e, db, version)
	setupSetupRoutes(e, userRepo)
	setupAuthRoutes(e, authenticated, cfg, userRepo, tokenManager, loginRateLimiter, auditLogMiddleware)

	// Protected routes (authentication required via authenticated group)
	setupUserRoutes(authenticated, userRepo)
	setupProjectRoutes(authenticated, projectRepo, dbRepo, projectMemberRepo, userRepo)
	setupEnvironmentRoutes(authenticated, envRepo, db)
	setupPolicyRoutes(authenticated, policyRepo, envRepo)
	setupDatabaseInstanceRoutes(
		authenticated,
		dbInstanceRepo,
		dbRepo,
		projectRepo,
		envRepo,
		changePlanRepo,
		connectionMgr,
		cfg.Security.EncryptionKey,
	)
	setupDatabaseRoutes(
		authenticated,
		dbRepo,
		projectRepo,
		envRepo,
		dbInstanceRepo,
		changePlanRepo,
		projectMemberRepo,
		schemaMetadataService,
	)
	setupChangePlanRoutes(
		authenticated,
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
	)
	setupAuditLogRoutes(authenticated, auditLogRepo)
	setupChangelogRoutes(authenticated, changelogRepo, schemaHistoryRepo, changePlanRepo, projectMemberRepo)

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
	authenticated *echo.Group,
	cfg *config.Config,
	userRepo repository.UserRepository,
	tokenManager *auth.TokenManager,
	loginRateLimiter *handlers.RateLimiter,
	auditLogMiddleware *middleware.AuditLogMiddleware,
) {
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

	// Public routes - no authentication required, but audit logging enabled
	publicAuthGroup := e.Group("/api/v1/auth")
	publicAuthGroup.Use(auditLogMiddleware.AuditLog) // Apply audit logging middleware
	publicAuthGroup.POST("/register", registerHandler.Register)
	publicAuthGroup.POST("/login", loginHandler.Login)
	publicAuthGroup.GET("/google", googleHandler.InitiateGoogleAuth)
	publicAuthGroup.GET("/google/callback", googleHandler.GoogleCallback)

	// Protected routes - use authenticated group (auth already applied)
	authGroup := authenticated.Group("/auth")
	authGroup.POST("/logout", logoutHandler.Logout)
	authGroup.GET("/me", meHandler.GetCurrentUser)
}

// setupUserRoutes registers user management routes
func setupUserRoutes(authenticated *echo.Group, userRepo repository.UserRepository) {
	userHandler := handlers.NewUserHandler(userRepo, logger.Get())
	usersGroup := authenticated.Group("/users")

	// Authorization middleware
	authzMiddleware := middleware.NewAuthzMiddleware()

	// All user management routes require authentication (via group) and admin role
	usersGroup.GET(
		"",
		userHandler.ListUsers,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersRead),
	)
	usersGroup.GET(
		"/:id",
		userHandler.GetUser,
		// Authorization check is in handler - allows users to view own profile
	)
	usersGroup.POST(
		"",
		userHandler.CreateUser,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersCreate),
	)
	usersGroup.PUT(
		"/:id",
		userHandler.UpdateUser,
		// Authorization check is in handler - allows users to update own profile
	)
	usersGroup.DELETE(
		"/:id",
		userHandler.DeleteUser,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionUsersDelete),
	)
}

// setupProjectRoutes registers project management routes
func setupProjectRoutes(
	authenticated *echo.Group,
	projectRepo repository.ProjectRepository,
	databaseRepo repository.DatabaseRepository,
	projectMemberRepo repository.ProjectMemberRepository,
	userRepo repository.UserRepository,
) {
	projectHandler := handlers.NewProjectHandler(
		projectRepo,
		databaseRepo,
		projectMemberRepo,
		userRepo,
		logger.Get(),
	)
	projectsGroup := authenticated.Group("/projects")

	// Authorization middleware
	authzMiddleware := middleware.NewAuthzMiddlewareWithRepo(projectMemberRepo)

	// POST requires project.create permission
	projectsGroup.POST(
		"",
		projectHandler.CreateProject,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionProjectsCreate),
	)

	// GET endpoints require authentication (via group)
	// List all projects requires project.read permission
	projectsGroup.GET("", projectHandler.ListProjects)
	projectsGroup.GET(
		"/:id",
		projectHandler.GetProject,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionProjectsRead),
	)

	// PUT and DELETE require project.update permission
	projectsGroup.PUT(
		"/:id",
		projectHandler.UpdateProject,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionProjectsUpdate),
	)
	projectsGroup.DELETE(
		"/:id",
		projectHandler.DeleteProject,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionProjectsDelete),
	)

	// Project member routes require authentication (via group) and project-level permissions
	// POST requires project.members.create permission
	projectsGroup.POST(
		"/:id/members",
		projectHandler.AddProjectMember,
		authzMiddleware.RequireProjectPermission(rbac.PermissionProjectMembersCreate),
	)
	// GET requires project.members.read permission
	projectsGroup.GET(
		"/:id/members",
		projectHandler.ListProjectMembers,
		authzMiddleware.RequireProjectPermission(rbac.PermissionProjectMembersRead),
	)
	// DELETE requires project.members.delete permission
	projectsGroup.DELETE(
		"/:id/members/:user_id",
		projectHandler.RemoveProjectMember,
		authzMiddleware.RequireProjectPermission(rbac.PermissionProjectMembersDelete),
	)
	// PUT requires project.members.update permission
	projectsGroup.PUT(
		"/:id/members/:user_id",
		projectHandler.UpdateProjectMemberRole,
		authzMiddleware.RequireProjectPermission(rbac.PermissionProjectMembersUpdate),
	)
}

// setupEnvironmentRoutes registers environment management routes
func setupEnvironmentRoutes(
	authenticated *echo.Group,
	envRepo repository.EnvironmentRepository,
	db *sqlx.DB,
) {
	envHandler := handlers.NewEnvironmentHandler(envRepo, db, logger.Get())
	environmentsGroup := authenticated.Group("/environments")

	// Authorization middleware
	authzMiddleware := middleware.NewAuthzMiddleware()

	// All environment routes require environments.create permission
	environmentsGroup.POST(
		"",
		envHandler.CreateEnvironment,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionEnvironmentsCreate),
	)
	// GET requires environments.read permission
	environmentsGroup.GET(
		"",
		envHandler.ListEnvironments,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionEnvironmentsRead),
	)
	// GET requires environments.read permission
	environmentsGroup.GET(
		"/:id",
		envHandler.GetEnvironment,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionEnvironmentsRead),
	)
	// PUT requires environments.update permission
	environmentsGroup.PUT(
		"/:id",
		envHandler.UpdateEnvironment,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionEnvironmentsUpdate),
	)
	// DELETE requires environments.delete permission
	environmentsGroup.DELETE(
		"/:id",
		envHandler.DeleteEnvironment,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionEnvironmentsDelete),
	)
}

// setupPolicyRoutes registers policy management routes
func setupPolicyRoutes(
	authenticated *echo.Group,
	policyRepo repository.PolicyRepository,
	envRepo repository.EnvironmentRepository,
) {
	policyHandler := handlers.NewPolicyHandler(policyRepo, envRepo, logger.Get())
	policiesGroup := authenticated.Group("/policies")

	// Authorization middleware
	authzMiddleware := middleware.NewAuthzMiddleware()

	// All policy routes require admin role
	policiesGroup.POST(
		"",
		policyHandler.CreatePolicy,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionPoliciesCreate),
	)
	policiesGroup.GET(
		"",
		policyHandler.ListPolicies,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionPoliciesRead),
	)
	policiesGroup.GET(
		"/:id",
		policyHandler.GetPolicy,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionPoliciesRead),
	)
	policiesGroup.PUT(
		"/:id",
		policyHandler.UpdatePolicy,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionPoliciesUpdate),
	)
	policiesGroup.DELETE(
		"/:id",
		policyHandler.DeletePolicy,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionPoliciesDelete),
	)
	// SQL Review rules endpoint - accessible to all authenticated users
	policiesGroup.GET("/sql-review/rules", policyHandler.GetSQLReviewRules)
}

// setupDatabaseInstanceRoutes registers database instance management routes
func setupDatabaseInstanceRoutes(
	authenticated *echo.Group,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	databaseRepo repository.DatabaseRepository,
	projectRepo repository.ProjectRepository,
	envRepo repository.EnvironmentRepository,
	changePlanRepo repository.ChangePlanRepository,
	connManager *database.ConnectionManager,
	encryptionKey string,
) {
	dbInstanceHandler := handlers.NewDatabaseInstanceHandler(
		dbInstanceRepo,
		databaseRepo,
		projectRepo,
		envRepo,
		changePlanRepo,
		connManager,
		encryptionKey,
		logger.Get(),
	)
	dbInstancesGroup := authenticated.Group("/database-instances")

	// Authorization middleware
	authzMiddleware := middleware.NewAuthzMiddleware()

	// POST test connection for existing instance - must come before /test-connection
	// Requires database_instances.read permission
	dbInstancesGroup.POST(
		"/:id/test-connection",
		dbInstanceHandler.TestDatabaseInstanceConnection,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabaseInstancesRead),
	)
	// POST test connection with credentials (for new instances before creation)
	// Requires database_instances.create permission
	dbInstancesGroup.POST(
		"/test-connection",
		dbInstanceHandler.TestDatabaseInstanceConnection,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabaseInstancesCreate),
	)
	// POST requires database_instances.create permission
	dbInstancesGroup.POST(
		"",
		dbInstanceHandler.CreateDatabaseInstance,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabaseInstancesCreate),
	)
	// GET requires database_instances.read permission
	dbInstancesGroup.GET(
		"",
		dbInstanceHandler.ListDatabaseInstances,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
	// GET requires database_instances.read permission
	dbInstancesGroup.GET(
		"/:id",
		dbInstanceHandler.GetDatabaseInstance,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
	// PUT requires database_instances.update permission
	dbInstancesGroup.PUT(
		"/:id",
		dbInstanceHandler.UpdateDatabaseInstance,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabaseInstancesUpdate),
	)
	// DELETE requires database_instances.delete permission
	dbInstancesGroup.DELETE(
		"/:id",
		dbInstanceHandler.DeleteDatabaseInstance,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabaseInstancesDelete),
	)
	// POST requires databases.create permission
	dbInstancesGroup.POST(
		"/:id/databases",
		dbInstanceHandler.CreateDatabaseForInstance,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesCreate),
	)
}

// setupDatabaseRoutes registers database management routes
func setupDatabaseRoutes(
	authenticated *echo.Group,
	dbRepo repository.DatabaseRepository,
	projectRepo repository.ProjectRepository,
	envRepo repository.EnvironmentRepository,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	changePlanRepo repository.ChangePlanRepository,
	projectMemberRepo repository.ProjectMemberRepository,
	schemaMetadataService *services.SchemaMetadataService,
) {
	dbHandler := handlers.NewDatabaseHandler(
		dbRepo,
		projectRepo,
		envRepo,
		dbInstanceRepo,
		changePlanRepo,
		schemaMetadataService,
		logger.Get(),
	)
	databasesGroup := authenticated.Group("/databases")

	// Authorization middleware
	authzMiddleware := middleware.NewAuthzMiddlewareWithRepo(projectMemberRepo)

	// POST requires admin role
	databasesGroup.POST(
		"",
		dbHandler.CreateDatabase,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesCreate),
	)
	// GET endpoints require authentication (via group) and read permission
	databasesGroup.GET(
		"",
		dbHandler.ListDatabases,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
	databasesGroup.GET(
		"/:id",
		dbHandler.GetDatabase,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
	// PUT and DELETE require admin role
	databasesGroup.PUT(
		"/:id",
		dbHandler.UpdateDatabase,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesUpdate),
	)
	databasesGroup.DELETE(
		"/:id",
		dbHandler.DeleteDatabase,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesDelete),
	)

	// Project-scoped database routes require organization-level read permission
	// (Organization admins, DBAs, and Developers have PermissionDatabasesRead)
	projectsGroup := authenticated.Group("/projects")
	projectsGroup.GET(
		"/:project_id/databases",
		dbHandler.ListProjectDatabases,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)

	// Schema metadata routes require databases.read permission
	databasesGroup.GET(
		"/:id/schema",
		dbHandler.GetDatabaseSchema,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
	databasesGroup.GET(
		"/:id/tables/:tableName",
		dbHandler.GetTableDetail,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
	databasesGroup.POST(
		"/:id/schema/refresh",
		dbHandler.RefreshDatabaseSchema,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
}

// setupChangePlanRoutes registers change plan management routes
func setupChangePlanRoutes(
	authenticated *echo.Group,
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

	// Initialize changelog repository
	changelogRepo := postgres.NewChangelogRepository(db)

	// Initialize schema history repository
	schemaHistoryRepo := postgres.NewSchemaHistoryRepository(db)

	// Initialize schema version service
	schemaVersionService := services.NewSchemaVersionService(
		schemaHistoryRepo,
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
		changelogRepo,
		schemaHistoryRepo,
		schemaVersionService,
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
		connectionMgr,
		logger.Get(),
	)
	changePlansGroup := authenticated.Group("/change-plans")

	// Authorization middleware
	authzMiddleware := middleware.NewAuthzMiddlewareWithRepo(projectMemberRepo)
	changePlanExtractor := middleware.NewChangePlanProjectExtractor(changePlanRepo)

	// Status transition validation middleware
	statusTransitionMiddleware := middleware.NewChangePlanStatusTransitionMiddleware(changePlanRepo)

	// Execution auth middleware (special permission logic for execution)
	executionAuthMiddleware := middleware.NewExecutionAuthMiddleware(changePlanRepo, projectMemberRepo)

	// CREATE change plan - requires project-level changes.create permission
	changePlansGroup.POST("",
		changePlanHandler.CreateChangePlan,
		middleware.ExtractProjectFromBody(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesCreate),
	)

	// GET change plan by ID - requires changes.read permission
	changePlansGroup.GET("/:id",
		changePlanHandler.GetChangePlan,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesRead),
	)

	// GET approval history - requires changes.read permission
	changePlansGroup.GET("/:id/approvals",
		changePlanHandler.GetApprovalHistory,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesRead),
	)

	// LIST change plans - no project-level check (filtered by handler based on user's projects)
	changePlansGroup.GET("",
		changePlanHandler.ListChangePlans,
	)

	// UPDATE change plan - requires changes.update permission
	changePlansGroup.PUT("/:id",
		changePlanHandler.UpdateChangePlan,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesUpdate),
		statusTransitionMiddleware.ValidateStatusTransition,
	)

	// ASSIGN reviewer - requires changes.update permission
	changePlansGroup.PUT("/:id/reviewer",
		changePlanHandler.AssignReviewer,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesUpdate),
	)

	// DELETE change plan - requires changes.delete permission
	changePlansGroup.DELETE("/:id",
		changePlanHandler.DeleteChangePlan,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesDelete),
	)

	// TRIGGER SQL review - requires changes.create permission
	changePlansGroup.POST("/:id/review",
		changePlanHandler.TriggerSQLReview,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesCreate),
	)

	// GET review findings - requires changes.read permission
	changePlansGroup.GET("/:id/review/findings",
		changePlanHandler.GetReviewFindings,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesRead),
	)

	// APPROVE change plan - requires changes.approve permission
	changePlansGroup.POST("/:id/approve",
		changePlanHandler.ApproveChangePlan,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesApprove),
	)

	// CLOSE change plan - requires changes.update permission
	changePlansGroup.POST("/:id/close",
		changePlanHandler.CloseChangePlan,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesUpdate),
	)

	// EXECUTE change plan - uses special execution middleware with complex permission logic
	changePlansGroup.POST("/:id/execute",
		changePlanHandler.ExecuteChangePlan,
		executionAuthMiddleware.RequireExecutionPermission(),
	)

	// GET queue status - requires changes.read permission
	changePlansGroup.GET("/:id/queue-status",
		changePlanHandler.GetQueueStatus,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesRead),
	)

	// GET execution logs - requires changes.read permission
	changePlansGroup.GET("/:id/execution-logs",
		changePlanHandler.GetExecutionLogHistory,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesRead),
	)

	// RESUBMIT change plan - requires changes.update permission
	changePlansGroup.POST("/:id/resubmit",
		changePlanHandler.ResubmitChangePlan,
		changePlanExtractor.Extract(),
		authzMiddleware.RequireProjectPermission(rbac.PermissionChangesUpdate),
	)
}

// setupAuditLogRoutes registers audit log querying routes
func setupAuditLogRoutes(
	authenticated *echo.Group,
	auditLogRepo repository.AuditLogRepository,
) {
	auditLogHandler := handlers.NewAuditLogHandler(auditLogRepo, logger.Get())
	auditLogsGroup := authenticated.Group("/audit-logs")

	// Authorization middleware
	authzMiddleware := middleware.NewAuthzMiddleware()

	// All audit log routes require authentication (via group) and audit_logs.read permission
	auditLogsGroup.GET(
		"",
		auditLogHandler.ListAuditLogs,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionAuditLogsRead),
	)
}

// setupChangelogRoutes registers changelog querying routes
func setupChangelogRoutes(
	authenticated *echo.Group,
	changelogRepo repository.ChangelogRepository,
	schemaHistoryRepo repository.SchemaHistoryRepository,
	changePlanRepo repository.ChangePlanRepository,
	projectMemberRepo repository.ProjectMemberRepository,
) {
	changelogHandler := handlers.NewChangelogHandler(
		changelogRepo,
		schemaHistoryRepo,
		changePlanRepo,
		logger.Get(),
	)

	// Authorization middleware
	authzMiddleware := middleware.NewAuthzMiddlewareWithRepo(projectMemberRepo)

	// Database-specific changelog route - requires databases.read permission
	databasesGroup := authenticated.Group("/databases")
	databasesGroup.GET(
		"/:id/changelog",
		changelogHandler.ListDatabaseChangelog,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)

	// Changelog detail route - requires databases.read permission
	changelogGroup := authenticated.Group("/changelog")
	changelogGroup.GET(
		"/:id",
		changelogHandler.GetChangelogDetails,
		authzMiddleware.RequireOrganizationPermission(rbac.PermissionDatabasesRead),
	)
}
