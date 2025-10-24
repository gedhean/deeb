# Phase 1: Foundation & Project Setup

**Duration**: ~5 days (40 milestones @ 1h each)
**Dependencies**: None (first phase)
**Status**: 🚧 TODO

## Goal

Establish the foundational infrastructure for the Deeb MVP by setting up the project structure, database schema, core utilities, and authentication framework for both Go backend and React frontend. This phase creates the skeleton that all subsequent phases will build upon.

## Phase-level Acceptance Criteria

- Go project structure follows standard layout (cmd/, internal/, pkg/, api/)
- React frontend initialized with Vite and Shadcn/ui configured
- Database schema designed and migration system implemented
- Authentication system working for email/password and OAuth 2.0 (Google)
- User management (create, update, deactivate) implemented
- Role-based access control (RBAC) foundation in place
- Configuration management system operational (environment variables + config files)
- Logging infrastructure functional with structured logging
- Health check endpoint implemented
- Unit tests for core utilities passing
- Documentation for setup and configuration complete
- Docker development environment configured
- CI/CD pipeline skeleton created

## Milestones

### P01M01: Initialize Go project with standard layout

**Status:** ✅ COMPLETE
**ID:** P01M01

**Goal**
Create the Go project structure following standard Go project layout with cmd/, internal/, pkg/, and api/ directories.

**Acceptance Criteria**
- [x] All backend code should be in the backend/ directory.
- [x] Directory structure created: cmd/server/, cmd/migrate/, internal/, pkg/, api/
- [x] go.mod initialized with module name github.com/nsxbet/deeb
- [x] .gitignore configured for Go projects
- [x] README.md with project overview created
- [x] Directory structure documented in README.md

### P01M02: Initialize React frontend with Vite

**Status:** ✅ COMPLETE
**ID:** P01M02

**Goal**
Set up React frontend project using Vite build tool with TypeScript configuration.

**Acceptance Criteria**
- [x] All frontend code should be in the frontend/ directory.
- [x] Use pnpm as package manager.
- [x] Vite project initialized in frontend/ directory
- [x] TypeScript configured with strict mode
- [x] React 18+ installed and configured
- [x] Development server runs successfully (pnpm run dev)
- [x] Basic App component renders

---

### P01M03: Configure Shadcn/ui component library

**Status:** ✅ COMPLETE
**ID:** P01M03

**Goal**
Install and configure Shadcn/ui component library with Tailwind CSS for consistent UI components.

**Acceptance Criteria**
- [x] Tailwind CSS installed and configured
- [x] Shadcn/ui CLI initialized
- [x] Basic components installed (Button, Card, Input)
- [x] Theme configuration completed
- [x] Sample page using Shadcn components renders correctly

---

### P01M04: Design database schema for metadata database

**Status:** ✅ COMPLETE
**ID:** P01M04

**Goal**
Design comprehensive database schema for storing users, issues, approvals, audit logs, and environment configurations.

**Acceptance Criteria**
- [x] Users table schema defined (id, email, password_hash, role, status, created_at, updated_at)
- [x] Issues table schema defined (id, title, description, sql_content, status, author_id, target_database_id)
- [x] Approvals table schema defined (id, issue_id, approver_id, status, comments, approved_at)
- [x] Audit_logs table schema defined (id, user_id, action, resource_type, resource_id, details, timestamp)
- [x] Environments table schema defined (id, name, database_type, connection_details_encrypted)
- [x] Entity-relationship diagram created
- [x] All tables use proper indexes on foreign keys and query filters

---

### P01M05: Implement database migration system

**Status:** ✅ COMPLETE
**ID:** P01M05

**Goal**
Set up database migration system using golang-migrate or similar tool for versioned schema changes.

**Acceptance Criteria**
- [x] Migration tool dependency added to go.mod
- [x] Migrations directory created: migrations/
- [x] Initial migration files created (up and down)
- [x] Migration command implemented in cmd/migrate/
- [x] Documentation for running migrations added to README
- [x] Migrations run successfully on local database

---

### P01M06: Implement configuration management system

**Status:** ✅ COMPLETE
**ID:** P01M06

**Goal**
Create configuration management using environment variables and config files with validation.

**Acceptance Criteria**
- [x] Config struct defined in internal/config/
- [x] Environment variable loading implemented
- [x] Config file loading implemented (YAML or JSON)
- [x] Config validation with helpful error messages
- [x] Default values provided for optional settings
- [x] Documentation for all configuration options

---

### P01M07: Set up structured logging with Zap

**Status:** ✅ COMPLETE
**ID:** P01M07

**Goal**
Configure structured logging using go.uber.org/zap with JSON output and log levels.

**Acceptance Criteria**
- [x] Zap logger initialized in main.go
- [x] Log levels configurable via environment variable
- [x] JSON format output for production
- [x] Console format for development
- [x] Logger accessible throughout application
- [x] Sample log statements working correctly

---

### P01M08: Create user model and repository

**Status:** ✅ COMPLETE
**ID:** P01M08

**Goal**
Implement user data model and repository interface for database operations.

**Acceptance Criteria**
- [x] User struct defined with JSON tags
- [x] UserRepository interface defined (Create, GetByID, GetByEmail, Update, Delete)
- [x] PostgreSQL implementation of UserRepository
- [x] Basic CRUD operations working
- [x] Unit tests for repository operations passing

---

### P01M09: Implement password hashing with bcrypt

**Status:** ✅ COMPLETE
**ID:** P01M09

**Goal**
Add secure password hashing and verification using bcrypt with appropriate work factor.

**Acceptance Criteria**
- [x] HashPassword function implemented
- [x] VerifyPassword function implemented
- [x] Bcrypt work factor set to 12 or higher
- [x] Unit tests for password hashing passing
- [x] Error handling for invalid inputs

---

### P01M10: Implement JWT token generation and validation

**Status:** 🚧 TODO
**ID:** P01M10

**Goal**
Create JWT token management for session handling with signing and verification.

**Acceptance Criteria**
- [ ] JWT secret loaded from configuration
- [ ] GenerateToken function implemented with claims (user_id, email, role, exp)
- [ ] ValidateToken function implemented
- [ ] Token expiration set to 24 hours
- [ ] Unit tests for token operations passing

---

### P01M11: Create authentication middleware

**Status:** 🚧 TODO
**ID:** P01M11

**Goal**
Implement HTTP middleware for JWT authentication on protected routes.

**Acceptance Criteria**
- [ ] Authentication middleware extracts JWT from Authorization header
- [ ] Token validation performed before route handler
- [ ] User context stored in request context
- [ ] 401 Unauthorized returned for missing/invalid tokens
- [ ] Unit tests for middleware passing

---

### P01M12: Implement email/password registration endpoint

**Status:** 🚧 TODO
**ID:** P01M12

**Goal**
Create POST /api/v1/auth/register endpoint for user registration with email and password.

**Acceptance Criteria**
- [ ] Endpoint defined: POST /api/v1/auth/register
- [ ] Request validation (email format, password strength)
- [ ] Password hashing before storage
- [ ] User created in database
- [ ] 201 Created response with user data (excluding password)
- [ ] Error handling for duplicate email addresses

---

### P01M13: Implement email/password login endpoint

**Status:** 🚧 TODO
**ID:** P01M13

**Goal**
Create POST /api/v1/auth/login endpoint that validates credentials and returns JWT token.

**Acceptance Criteria**
- [ ] Endpoint defined: POST /api/v1/auth/login
- [ ] Credentials validated against database
- [ ] JWT token generated on successful authentication
- [ ] Token returned in response body
- [ ] 401 Unauthorized for invalid credentials
- [ ] Rate limiting applied (10 attempts per 5 minutes)

---

### P01M14: Implement OAuth 2.0 Google authentication flow

**Status:** 🚧 TODO
**ID:** P01M14

**Goal**
Add OAuth 2.0 authentication with Google including redirect and callback handling.

**Acceptance Criteria**
- [ ] Google OAuth client ID and secret configured
- [ ] GET /api/v1/auth/google endpoint redirects to Google
- [ ] GET /api/v1/auth/google/callback handles OAuth response
- [ ] User created or updated from Google profile
- [ ] JWT token generated for authenticated user
- [ ] Error handling for failed OAuth flow

---

### P01M15: Implement role-based access control (RBAC) foundation

**Status:** 🚧 TODO
**ID:** P01M15

**Goal**
Create RBAC system with roles (Developer, Approver, Admin) and permission checking.

**Acceptance Criteria**
- [ ] Role enum defined (Developer, Approver, Admin)
- [ ] Permission constants defined for actions
- [ ] Role-to-permissions mapping implemented
- [ ] HasPermission function checks user role against required permission
- [ ] Authorization middleware created for protected routes
- [ ] Unit tests for permission checking passing

---

### P01M16: Create user management endpoints

**Status:** 🚧 TODO
**ID:** P01M16

**Goal**
Implement REST API endpoints for user CRUD operations (Admin only).

**Acceptance Criteria**
- [ ] GET /api/v1/users - list all users
- [ ] GET /api/v1/users/:id - get user by ID
- [ ] PUT /api/v1/users/:id - update user
- [ ] DELETE /api/v1/users/:id - deactivate user (soft delete)
- [ ] Admin permission required for all endpoints
- [ ] Input validation and error handling

---

### P01M17: Implement health check endpoint

**Status:** 🚧 TODO
**ID:** P01M17

**Goal**
Create GET /health endpoint that returns application and database health status.

**Acceptance Criteria**
- [ ] Endpoint defined: GET /health
- [ ] Database connectivity check included
- [ ] Response includes status, version, timestamp
- [ ] 200 OK when healthy, 503 Service Unavailable when unhealthy
- [ ] No authentication required
- [ ] Response time under 100ms

---

### P01M18: Set up HTTP server with net/http

**Status:** 🚧 TODO
**ID:** P01M18

**Goal**
Configure HTTP server using Go standard library net/http with graceful shutdown.

**Acceptance Criteria**
- [ ] Server initialized in cmd/server/main.go
- [ ] Port configurable via environment variable
- [ ] Graceful shutdown on SIGINT/SIGTERM signals
- [ ] Request logging middleware applied
- [ ] CORS headers configured for frontend origin
- [ ] Server starts and responds to requests

---

### P01M19: Implement request logging middleware

**Status:** 🚧 TODO
**ID:** P01M19

**Goal**
Create middleware that logs all HTTP requests with correlation IDs for tracing.

**Acceptance Criteria**
- [ ] Correlation ID generated for each request
- [ ] Request method, path, status code logged
- [ ] Request duration logged
- [ ] Correlation ID added to response headers
- [ ] Logs in structured JSON format
- [ ] Integration with Zap logger

---

### P01M20: Implement error handling middleware

**Status:** 🚧 TODO
**ID:** P01M20

**Goal**
Create centralized error handling middleware that catches panics and formats error responses.

**Acceptance Criteria**
- [ ] Panic recovery middleware implemented
- [ ] Error responses formatted consistently (JSON)
- [ ] Stack traces logged for 500 errors
- [ ] HTTP status codes mapped correctly
- [ ] Sensitive information not exposed in responses
- [ ] Unit tests for error handling passing

---

### P01M21: Create Docker development environment

**Status:** 🚧 TODO
**ID:** P01M21

**Goal**
Set up Docker Compose for local development with PostgreSQL database.

**Acceptance Criteria**
- [ ] docker-compose.yml created
- [ ] PostgreSQL service configured
- [ ] Database credentials configured
- [ ] Volume mounts for persistent data
- [ ] Development workflow documented in README
- [ ] docker-compose up starts all services successfully

---

### P01M22: Set up frontend API client

**Status:** 🚧 TODO
**ID:** P01M22

**Goal**
Create API client module in frontend for making HTTP requests to backend.

**Acceptance Criteria**
- [ ] API client class/module created
- [ ] Base URL configurable via environment variable
- [ ] Request interceptor adds JWT token to headers
- [ ] Response interceptor handles errors
- [ ] TypeScript interfaces for API requests/responses
- [ ] Sample API call working (e.g., health check)

---

### P01M23: Implement frontend authentication context

**Status:** 🚧 TODO
**ID:** P01M23

**Goal**
Create React context for managing authentication state and user session.

**Acceptance Criteria**
- [ ] AuthContext and AuthProvider created
- [ ] Login, logout, and register functions
- [ ] JWT token stored in localStorage
- [ ] User state available throughout app
- [ ] Token refresh logic implemented
- [ ] Protected route wrapper component

---

### P01M24: Create login page UI

**Status:** 🚧 TODO
**ID:** P01M24

**Goal**
Build login page with email/password form and Google OAuth button using Shadcn components.

**Acceptance Criteria**
- [ ] Login form with email and password fields
- [ ] Form validation (email format, required fields)
- [ ] Login button triggers authentication
- [ ] Google OAuth button redirects to OAuth flow
- [ ] Error messages displayed for failed login
- [ ] Successful login redirects to dashboard

---

### P01M25: Create registration page UI

**Status:** 🚧 TODO
**ID:** P01M25

**Goal**
Build registration page with email/password form and validation.

**Acceptance Criteria**
- [ ] Registration form with email, password, confirm password
- [ ] Password strength indicator
- [ ] Form validation (email format, password match, strength)
- [ ] Registration button creates account
- [ ] Success message and redirect to login
- [ ] Error handling for duplicate accounts

---

### P01M26: Implement session management on frontend

**Status:** 🚧 TODO
**ID:** P01M26

**Goal**
Add session persistence and automatic token refresh on frontend.

**Acceptance Criteria**
- [ ] JWT token persisted in localStorage
- [ ] Token loaded on app initialization
- [ ] Expired token detected and user logged out
- [ ] Token refresh before expiration
- [ ] Logout clears session data
- [ ] Unauthorized requests redirect to login

---

### P01M27: Create navigation bar component

**Status:** 🚧 TODO
**ID:** P01M27

**Goal**
Build navigation bar with logo, menu items, and user profile dropdown.

**Acceptance Criteria**
- [ ] Navigation bar component created
- [ ] Logo and app name displayed
- [ ] Menu items for authenticated users (Issues, Approvals, Audit Logs)
- [ ] User profile dropdown with logout option
- [ ] Responsive design for mobile and desktop
- [ ] Active route highlighted

---

### P01M28: Set up React Router for navigation

**Status:** 🚧 TODO
**ID:** P01M28

**Goal**
Configure React Router for client-side routing with protected routes.

**Acceptance Criteria**
- [ ] React Router installed and configured
- [ ] Routes defined for login, register, dashboard, issues
- [ ] Protected routes require authentication
- [ ] Redirect to login for unauthenticated access
- [ ] 404 Not Found page for invalid routes
- [ ] Navigation working between pages

---

### P01M29: Implement CI/CD pipeline skeleton

**Status:** 🚧 TODO
**ID:** P01M29

**Goal**
Create GitHub Actions workflow for automated testing and linting.

**Acceptance Criteria**
- [ ] .github/workflows/ci.yml created
- [ ] Go tests run on push and PR
- [ ] Go linting with golangci-lint
- [ ] Frontend tests run on push and PR
- [ ] Frontend linting with ESLint
- [ ] Workflow badge in README

---

### P01M30: Write unit tests for configuration loading

**Status:** 🚧 TODO
**ID:** P01M30

**Goal**
Create comprehensive unit tests for configuration management system.

**Acceptance Criteria**
- [ ] Test config loading from environment variables
- [ ] Test config loading from file
- [ ] Test config validation with invalid values
- [ ] Test default values applied correctly
- [ ] All tests passing
- [ ] Code coverage for config package >80%

---

### P01M31: Write unit tests for password hashing

**Status:** 🚧 TODO
**ID:** P01M31

**Goal**
Create unit tests for password hashing and verification functions.

**Acceptance Criteria**
- [ ] Test password hashing produces different hashes for same input
- [ ] Test valid password verification returns true
- [ ] Test invalid password verification returns false
- [ ] Test empty password handling
- [ ] All tests passing
- [ ] Code coverage for password package >90%

---

### P01M32: Write unit tests for JWT operations

**Status:** 🚧 TODO
**ID:** P01M32

**Goal**
Create unit tests for JWT token generation and validation.

**Acceptance Criteria**
- [ ] Test token generation includes correct claims
- [ ] Test valid token passes validation
- [ ] Test expired token fails validation
- [ ] Test tampered token fails validation
- [ ] All tests passing
- [ ] Code coverage for JWT package >90%

---

### P01M33: Write unit tests for RBAC permission checking

**Status:** 🚧 TODO
**ID:** P01M33

**Goal**
Create unit tests for role-based access control logic.

**Acceptance Criteria**
- [ ] Test Developer role has correct permissions
- [ ] Test Approver role has correct permissions
- [ ] Test Admin role has all permissions
- [ ] Test HasPermission function for various scenarios
- [ ] All tests passing
- [ ] Code coverage for RBAC package >85%

---

### P01M34: Write integration tests for authentication endpoints

**Status:** 🚧 TODO
**ID:** P01M34

**Goal**
Create integration tests for registration and login API endpoints.

**Acceptance Criteria**
- [ ] Test POST /api/v1/auth/register creates user
- [ ] Test POST /api/v1/auth/login returns JWT token
- [ ] Test login with invalid credentials fails
- [ ] Test registration with duplicate email fails
- [ ] Test rate limiting on login endpoint
- [ ] All tests passing with test database

---

### P01M35: Document API endpoints with OpenAPI/Swagger

**Status:** 🚧 TODO
**ID:** P01M35

**Goal**
Create OpenAPI 3.0 specification for authentication and user management endpoints.

**Acceptance Criteria**
- [ ] openapi.yaml created in api/ directory
- [ ] Authentication endpoints documented
- [ ] User management endpoints documented
- [ ] Request/response schemas defined
- [ ] Error responses documented
- [ ] Swagger UI accessible at /api/docs

---

### P01M36: Write setup and configuration documentation

**Status:** 🚧 TODO
**ID:** P01M36

**Goal**
Create comprehensive documentation for project setup and configuration.

**Acceptance Criteria**
- [ ] Prerequisites documented (Go version, Node version, PostgreSQL)
- [ ] Installation instructions for backend
- [ ] Installation instructions for frontend
- [ ] Environment variable documentation
- [ ] Database setup instructions
- [ ] Local development workflow documented

---

### P01M37: Create godoc comments for public packages

**Status:** 🚧 TODO
**ID:** P01M37

**Goal**
Add godoc comments to all public functions, types, and packages in Go code.

**Acceptance Criteria**
- [ ] Package-level comments for all packages
- [ ] Function comments for all exported functions
- [ ] Type comments for all exported types
- [ ] Examples for complex functions
- [ ] godoc generates readable documentation
- [ ] golangci-lint godoc checks passing

---

### P01M38: Configure code formatting and linting

**Status:** 🚧 TODO
**ID:** P01M38

**Goal**
Set up automated code formatting and linting for Go and JavaScript code.

**Acceptance Criteria**
- [ ] gofmt configured and run on all Go files
- [ ] golangci-lint configured with strict rules
- [ ] ESLint configured for React/TypeScript
- [ ] Prettier configured for code formatting
- [ ] Pre-commit hooks installed (optional but recommended)
- [ ] All linting checks passing

---

### P01M39: Write troubleshooting guide

**Status:** 🚧 TODO
**ID:** P01M39

**Goal**
Create troubleshooting guide for common development issues.

**Acceptance Criteria**
- [ ] Database connection issues documented
- [ ] Configuration errors documented
- [ ] Build errors documented
- [ ] Runtime errors documented
- [ ] Solutions provided for each issue
- [ ] Guide added to docs/ directory

---

### P01M40: Verify Phase 1 Completion

**Status:** 🚧 TODO
**ID:** P01M40

**Goal**
Verify that all Phase 1 requirements are met, tests pass, and the phase is ready for sign-off before proceeding to Phase 2.

**Acceptance Criteria**
- [ ] All previous milestones in Phase 1 marked as complete
- [ ] All tests pass (unit and integration)
- [ ] Code coverage meets project standards (>80% for critical packages)
- [ ] No outstanding bugs or issues from this phase
- [ ] Code reviewed and meets quality standards (gofmt, golangci-lint, ESLint passing)
- [ ] Documentation complete for all features in this phase
- [ ] Go backend serves API endpoints successfully
- [ ] React frontend renders and authenticates users
- [ ] Database migrations run successfully
- [ ] Docker development environment working
- [ ] Ready to proceed to Phase 2

---
