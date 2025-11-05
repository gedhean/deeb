# Phase 6: UI Components & Integration

**Duration**: ~4 days (35 milestones @ 1h each)
**Dependencies**: Phase 5
**Status**: 🚧 TODO

## Goal

Build comprehensive React frontend UI components for all user-facing features and integrate with backend APIs. This phase delivers the complete user experience for managing database changes through the web interface.

## Phase-level Acceptance Criteria

- [ ] Dashboard UI implemented (overview of open/recent Change Plans, pending approvals, system status)
- [ ] Authentication UI implemented (login with OAuth/email-password, user registration)
- [ ] User management UI implemented (admin only: create, update, deactivate users, assign roles)
- [ ] Projects & Environments management UI implemented (admin: CRUD operations)
- [ ] Policy management UI implemented (admin: CRUD operations, SQL_REVIEW type)
- [ ] Database management UI implemented (admin: add instances, databases, test connections)
- [ ] Change Plan creation form implemented (title, change type, SQL content, target database, optional description/rollback)
- [ ] Change Plan detail view implemented (full details, SQL Review results, approval status, execution history)
- [ ] Change Plan list view implemented (filtering by status, project, database, author)
- [ ] SQL Review results display implemented (errors, warnings, pass indicators)
- [ ] Approval workflow UI implemented (approve/close with comments)
- [ ] Execution trigger UI implemented (execute button with confirmation)
- [ ] Execution results display implemented (run log history, success/error details)
- [ ] Audit log viewer UI implemented (admin: filterable audit log)
- [ ] RBAC enforcement at UI level (show/hide features based on roles)
- [ ] Error handling UI implemented (user-friendly error messages)
- [ ] Loading states and feedback implemented
- [ ] Responsive design working (mobile-friendly)
- [ ] Frontend unit tests passing (Vitest)
- [ ] End-to-end tests passing (Playwright: Create → Approve → Execute workflow)
- [ ] All linting checks passing (ESLint)

## Milestones

<!-- Milestones appended by /wiz-milestones -->

### P06M01: Set up API client and authentication utilities

**Status:** ✅ COMPLETE
**ID:** P06M01

**Goal**
Create API client with axios and authentication utilities for HttpOnly cookie-based authentication.

**Acceptance Criteria**
- [x] API client created in `frontend/src/lib/api/client.ts`
- [x] Axios instance configured with base URL
- [x] `withCredentials: true` enabled to send HttpOnly cookies automatically
- [x] Response interceptor handles 401 errors (dispatches logout event)
- [x] Authentication utilities created in `frontend/src/lib/api/auth.ts`
- [x] Utilities document HttpOnly cookie approach (no localStorage token management)
- [x] Unit tests written (tests will pass once Vitest is installed - see P06M29 for test setup)

---

### P06M02: Create authentication store slice

**Status:** ✅ COMPLETE
**ID:** P06M02

**Goal**
Create Zustand store slice for authentication state management.

**Acceptance Criteria**
- [x] Authentication slice created in `frontend/src/store/slices/authSlice.ts`
- [x] State includes: user, isAuthenticated
- [x] Actions: login, logout, setUser
- [x] State managed in memory (no localStorage - tokens in HttpOnly cookies)
- [x] Store integrated with API client

---

### P06M03: Implement login page UI

**Status:** ✅ COMPLETE
**ID:** P06M03

**Goal**
Create login page with email/password form and OAuth Google button.

**Acceptance Criteria**
- [x] Login page component created in `frontend/src/pages/LoginPage.tsx`
- [x] Email/password form implemented
- [x] Google OAuth button implemented
- [x] Form validation working
- [x] Error handling and display
- [x] Responsive design working
- [x] Integration tests will be added in P06M32 (E2E tests for authentication flow)

---

### P06M04: Implement registration page UI

**Status:** ✅ COMPLETE
**ID:** P06M04

**Goal**
Create user registration page with email/password form.

**Acceptance Criteria**
- [x] Registration page component created in `frontend/src/pages/RegisterPage.tsx`
- [x] Registration form implemented (email, password, confirm password)
- [x] Form validation working (email format, password strength, matching passwords)
- [x] Success handling (redirect to login)
- [x] Error handling and display
- [x] Responsive design working
- [x] Integration tests will be added in P06M32 (E2E tests for authentication flow)

---

### P06M05: Implement protected route wrapper

**Status:** ✅ COMPLETE
**ID:** P06M05

**Goal**
Create protected route component that enforces authentication and redirects unauthenticated users.

**Acceptance Criteria**
- [x] ProtectedRoute component created in `frontend/src/components/protected-route.tsx` (note: uses kebab-case naming consistent with project conventions)
- [x] Authentication check implemented
- [x] Redirect to login if not authenticated
- [x] Loading state while checking authentication
- [x] Integrated with React Router

---

### P06M06: Implement Dashboard UI

**Status:** ✅ COMPLETE
**ID:** P06M06

**Goal**
Create Dashboard page with overview of open/recent Change Plans, pending approvals, and system status.

**Acceptance Criteria**
- [x] Dashboard page component created in `frontend/src/pages/DashboardPage.tsx`
- [x] Overview cards showing: open Change Plans count, pending approvals count, recent Change Plans
- [x] Recent Change Plans list with status indicators
- [x] Pending approvals list with quick actions
- [x] System status indicators
- [x] API integration working
- [x] Responsive design working

---

### P06M07: Create Projects management UI

**Status:** ✅ COMPLETE
**ID:** P06M07

**Goal**
Implement Projects CRUD UI for administrators (admin only).

**Acceptance Criteria**
- [x] Projects list page created in `frontend/src/pages/ProjectsPage.tsx`
- [x] Create Project form implemented
- [x] Edit Project form implemented
- [x] Delete Project confirmation dialog
- [x] RBAC: admin only visibility
- [x] API integration working
- [x] Responsive design working

---

### P06M08: Create Environments management UI

**Status:** ✅ COMPLETE
**ID:** P06M08

**Goal**
Implement Environments CRUD UI for administrators (admin only) with Policy assignment.

**Acceptance Criteria**
- [x] Environments list page created in `frontend/src/pages/EnvironmentsPage.tsx`
- [x] Create Environment form implemented (name, description, policy_id dropdown)
- [x] Edit Environment form implemented (including Policy assignment)
- [x] Delete Environment confirmation dialog
- [x] Policy dropdown populated from API (filtered by SQL_REVIEW type)
- [x] RBAC: admin only visibility
- [x] API integration working
- [x] Responsive design working

---

### P06M09: Create Policy management UI

**Status:** ✅ COMPLETE
**ID:** P06M09

**Goal**
Implement Policy CRUD UI for administrators (admin only), supporting SQL_REVIEW type and filtering by type.

**Acceptance Criteria**
- [x] Policy list page created in `frontend/src/pages/PoliciesPage.tsx`
- [x] Create Policy form implemented (name, description, type (SQL_REVIEW), policy_config JSON editor)
- [x] Edit Policy form implemented
- [x] Delete Policy confirmation dialog (with validation: prevent deletion if referenced by Environments)
- [x] Policy config JSON editor with validation
- [x] Policy config schema validation (validators enabled/disabled settings for SQL_REVIEW type)
- [x] Filter by type (SQL_REVIEW) in list view
- [x] RBAC: admin only visibility
- [x] API integration working
- [x] Responsive design working

---

### P06M10: Create Database Instances management UI

**Status:** ✅ COMPLETE
**ID:** P06M10

**Goal**
Implement Database Instances CRUD UI with connection testing (admin only).

**Acceptance Criteria**
- [x] Database Instances list page created in `frontend/src/pages/DatabaseInstancesPage.tsx`
- [x] Create Database Instance form implemented (name, type, host, port, credentials)
- [x] Connection test button implemented
- [x] Connection test results displayed
- [x] Edit Database Instance form implemented
- [x] Delete confirmation dialog
- [x] RBAC: admin only visibility
- [x] API integration working

---

### P06M11: Create Databases management UI

**Status:** ✅ COMPLETE
**ID:** P06M11

**Goal**
Implement Databases CRUD UI with project and environment assignment (admin only).

**Acceptance Criteria**
- [x] Databases list page created in `frontend/src/pages/DatabasesPage.tsx`
- [x] Create Database form implemented (database_instance, project, environment, database_name)
- [x] Project and environment dropdowns populated
- [x] Edit Database form implemented
- [x] Delete confirmation dialog
- [x] RBAC: admin only visibility
- [x] API integration working

---

### P06M12: Create User management UI

**Status:** ✅ COMPLETE
**ID:** P06M12

**Goal**
Implement User management UI for administrators (admin only).

**Acceptance Criteria**
- [x] Users list page created in `frontend/src/pages/UsersPage.tsx`
- [x] Create User form implemented (email, password, organization role)
- [x] Edit User form implemented (organization role, active status)
- [x] Deactivate user functionality
- [x] User list with role badges
- [x] RBAC: admin only visibility
- [x] API integration working

---

### P06M13: Create Change Plan creation form

**Status:** ✅ COMPLETE
**ID:** P06M13

**Goal**
Implement Change Plan creation form with all required and optional fields.

**Acceptance Criteria**
- [x] Change Plan creation page created in `frontend/src/pages/ChangePlanCreatePage.tsx`
- [x] Form fields: title, change_type (DDL/DML), sql_content (textarea), target_database (dropdown), description (optional), rollback_sql (optional)
- [x] Form validation working
- [x] SQL content editor with syntax highlighting (optional enhancement)
- [x] Database dropdown filtered by user's projects
- [x] API integration working
- [x] Success handling (redirect to Change Plan detail)

---

### P06M14: Create Change Plan list view with filtering

**Status:** ✅ COMPLETE
**ID:** P06M14

**Goal**
Implement Change Plan list page with filtering by status, project, database, and author.

**Acceptance Criteria**
- [x] Change Plan list page created in `frontend/src/pages/ChangePlansPage.tsx`
- [x] Filter controls implemented (status, project, database, author)
- [x] Filter state management
- [x] Change Plan cards/list with status badges
- [x] Pagination implemented
- [x] API integration working
- [x] Responsive design working

---

### P06M15: Create Change Plan detail view

**Status:** 🚧 TODO
**ID:** P06M15

**Goal**
Implement Change Plan detail page showing all Change Plan information.

**Acceptance Criteria**
- [ ] Change Plan detail page created in `frontend/src/pages/ChangePlanDetailPage.tsx`
- [ ] Change Plan information displayed (title, status, author, created date, SQL content, etc.)
- [ ] Status badge with color coding
- [ ] SQL content displayed with syntax highlighting (optional)
- [ ] API integration working
- [ ] Loading states implemented
- [ ] Error handling implemented

---

### P06M16: Implement SQL Review results display

**Status:** 🚧 TODO
**ID:** P06M16

**Goal**
Create component displaying SQL Review results with severity indicators (Error, Warning, Pass).

**Acceptance Criteria**
- [ ] SQL Review results component created in `frontend/src/components/SQLReviewResults.tsx`
- [ ] Findings grouped by severity (Error, Warning, Info)
- [ ] Error findings displayed with red indicators
- [ ] Warning findings displayed with yellow indicators
- [ ] Pass indicators for successful checks
- [ ] Finding details displayed (message, rule name, line number)
- [ ] Manual review trigger button implemented

---

### P06M17: Integrate SQL Review results into Change Plan detail

**Status:** 🚧 TODO
**ID:** P06M17

**Goal**
Integrate SQL Review results component into Change Plan detail page.

**Acceptance Criteria**
- [ ] SQL Review results displayed in Change Plan detail page
- [ ] Results shown in dedicated section
- [ ] Error blocking indication displayed
- [ ] Manual review trigger working
- [ ] Review results updated after manual trigger
- [ ] Loading states implemented

---

### P06M18: Implement approval workflow UI

**Status:** 🚧 TODO
**ID:** P06M18

**Goal**
Create UI components for approving or closing Change Plans with comments.

**Acceptance Criteria**
- [ ] Approval actions component created in `frontend/src/components/ApprovalActions.tsx`
- [ ] Approve button with confirmation dialog
- [ ] Close/reject button with confirmation dialog
- [ ] Comment input field
- [ ] Approval button only visible to Project Approvers
- [ ] Error handling for invalid approvals
- [ ] Success handling (refresh Change Plan details)

---

### P06M19: Implement approval history display

**Status:** 🚧 TODO
**ID:** P06M19

**Goal**
Create component displaying approval history for a Change Plan.

**Acceptance Criteria**
- [ ] Approval history component created in `frontend/src/components/ApprovalHistory.tsx`
- [ ] Approval history list displayed
- [ ] Each entry shows: approver, action (approve/close), comment, timestamp
- [ ] History ordered by timestamp (newest first)
- [ ] Integrated into Change Plan detail page
- [ ] API integration working

---

### P06M20: Implement execution trigger UI

**Status:** 🚧 TODO
**ID:** P06M20

**Goal**
Create UI for triggering Change Plan execution with confirmation dialog.

**Acceptance Criteria**
- [ ] Execute button component created in `frontend/src/components/ExecuteButton.tsx`
- [ ] Execute button only visible to authorized users (Organization Admin, Organization DBA, Project Owner, Author)
- [ ] Execute button only enabled for Approved Change Plans
- [ ] Confirmation dialog before execution
- [ ] Execution status displayed during execution
- [ ] Success/error handling
- [ ] API integration working

---

### P06M21: Implement execution results display

**Status:** 🚧 TODO
**ID:** P06M21

**Goal**
Create component displaying execution results and run log history.

**Acceptance Criteria**
- [ ] Execution results component created in `frontend/src/components/ExecutionResults.tsx`
- [ ] Execution status displayed (Completed/Failed)
- [ ] Execution duration displayed
- [ ] Rows affected displayed
- [ ] Error messages displayed for failures
- [ ] Run log history displayed
- [ ] Integrated into Change Plan detail page
- [ ] API integration working

---

### P06M22: Implement Audit Log viewer UI

**Status:** 🚧 TODO
**ID:** P06M22

**Goal**
Create Audit Log viewer page with filtering options (admin only).

**Acceptance Criteria**
- [ ] Audit Log viewer page created in `frontend/src/pages/AuditLogPage.tsx`
- [ ] Filter controls implemented (user, action, target_type, date range)
- [ ] Audit log entries displayed in table/list
- [ ] Entry details: timestamp, user, action, target, result
- [ ] Pagination implemented
- [ ] RBAC: admin only visibility
- [ ] API integration working

---

### P06M23: Implement RBAC enforcement at UI level

**Status:** 🚧 TODO
**ID:** P06M23

**Goal**
Create utilities and components for RBAC-based UI feature visibility.

**Acceptance Criteria**
- [ ] RBAC utilities created in `frontend/src/lib/rbac.ts`
- [ ] Role checking functions implemented
- [ ] Permission checking helpers
- [ ] Protected component wrapper for feature visibility
- [ ] Navigation menu filtered by roles
- [ ] Buttons/actions hidden based on permissions
- [ ] Unit tests passing

---

### P06M24: Implement error handling UI

**Status:** 🚧 TODO
**ID:** P06M24

**Goal**
Create user-friendly error display components and error handling utilities.

**Acceptance Criteria**
- [ ] Error display component created in `frontend/src/components/ErrorDisplay.tsx`
- [ ] Error toast notifications implemented
- [ ] Error boundaries implemented for React error handling
- [ ] API error handling utilities
- [ ] User-friendly error messages
- [ ] Error retry mechanisms where appropriate

---

### P06M25: Implement loading states and feedback

**Status:** 🚧 TODO
**ID:** P06M25

**Goal**
Add loading indicators and user feedback throughout the UI.

**Acceptance Criteria**
- [ ] Loading spinner component created
- [ ] Loading states implemented for API calls
- [ ] Skeleton loaders for data loading
- [ ] Success notifications/toasts
- [ ] Progress indicators for long-running operations
- [ ] Consistent loading patterns across UI

---

### P06M26: Implement responsive design

**Status:** 🚧 TODO
**ID:** P06M26

**Goal**
Ensure all UI components are responsive and mobile-friendly.

**Acceptance Criteria**
- [ ] Responsive breakpoints defined
- [ ] Mobile navigation implemented
- [ ] Forms optimized for mobile
- [ ] Tables responsive (scroll or card view on mobile)
- [ ] Touch-friendly button sizes
- [ ] Mobile testing completed

---

### P06M27: Create API integration hooks

**Status:** 🚧 TODO
**ID:** P06M27

**Goal**
Create React hooks for API calls with loading and error states.

**Acceptance Criteria**
- [ ] Custom hooks created in `frontend/src/hooks/api/`
- [ ] useChangePlans hook implemented
- [ ] useProjects hook implemented
- [ ] usePolicies hook implemented (filter by type)
- [ ] useApprovals hook implemented
- [ ] useExecution hook implemented
- [ ] Hooks include loading and error states
- [ ] Hooks support mutations and queries

---

### P06M28: Set up frontend routing

**Status:** 🚧 TODO
**ID:** P06M28

**Goal**
Configure React Router with all routes and protected route handling.

**Acceptance Criteria**
- [ ] Routes configured in `frontend/src/App.tsx`
- [ ] Public routes: /login, /register
- [ ] Protected routes: /dashboard, /change-plans, /projects, /environments, /policies, etc.
- [ ] Route guards implemented
- [ ] Redirect logic working
- [ ] 404 page implemented

---

### P06M29: Write frontend unit tests

**Status:** 🚧 TODO
**ID:** P06M29

**Goal**
Create unit tests for React components using Vitest and React Testing Library.

**Acceptance Criteria**
- [ ] Vitest configured
- [ ] Test utilities created
- [ ] Unit tests for key components (>85% coverage)
- [ ] API hook tests passing
- [ ] Store slice tests passing
- [ ] All unit tests passing

---

### P06M30: Set up Playwright for E2E testing

**Status:** 🚧 TODO
**ID:** P06M30

**Goal**
Configure Playwright for end-to-end testing and create test utilities.

**Acceptance Criteria**
- [ ] Playwright installed and configured
- [ ] Test configuration file created
- [ ] Test utilities created (login helpers, API mocks)
- [ ] Test fixtures configured
- [ ] E2E test environment setup working

---

### P06M31: Write E2E test for Create → Approve → Execute workflow

**Status:** 🚧 TODO
**ID:** P06M31

**Goal**
Create comprehensive E2E test covering the complete Change Plan workflow.

**Acceptance Criteria**
- [ ] E2E test created: Create Change Plan → SQL Review → Approve → Execute
- [ ] Test covers all workflow steps
- [ ] Test validates UI state at each step
- [ ] Test validates API calls
- [ ] Test handles async operations
- [ ] Test passing consistently

---

### P06M32: Write E2E tests for authentication flow

**Status:** 🚧 TODO
**ID:** P06M32

**Goal**
Create E2E tests for login, registration, and logout flows.

**Acceptance Criteria**
- [ ] E2E test for login flow passing
- [ ] E2E test for registration flow passing
- [ ] E2E test for logout flow passing
- [ ] E2E test for OAuth flow passing (if testable)
- [ ] All authentication E2E tests passing

---

### P06M33: Write E2E tests for RBAC enforcement

**Status:** 🚧 TODO
**ID:** P06M33

**Goal**
Create E2E tests verifying RBAC enforcement at UI level.

**Acceptance Criteria**
- [ ] E2E test for admin-only features passing
- [ ] E2E test for Project Developer features passing
- [ ] E2E test for Project Approver features passing
- [ ] E2E test for unauthorized access blocked
- [ ] All RBAC E2E tests passing

---

### P06M34: Run linting and fix issues

**Status:** 🚧 TODO
**ID:** P06M34

**Goal**
Run ESLint and fix all linting issues across the frontend codebase.

**Acceptance Criteria**
- [ ] ESLint configured correctly
- [ ] All linting errors fixed
- [ ] All linting warnings addressed
- [ ] Code formatted with Prettier
- [ ] Linting CI check passing

---

### P06M35: Verify Phase 6 Completion

**Status:** 🚧 TODO
**ID:** P06M35

**Goal**
Verify that all Phase 6 requirements are met, tests pass, and the phase is ready for sign-off.

**Acceptance Criteria**
- [ ] All previous milestones in Phase 6 marked as complete
- [ ] All tests pass (unit, E2E)
- [ ] Code coverage meets project standards (>85% for components)
- [ ] No outstanding bugs or issues from this phase
- [ ] Code reviewed and meets quality standards
- [ ] All linting checks passing (ESLint)
- [ ] UI working end-to-end
- [ ] All user workflows tested
- [ ] Responsive design verified
- [ ] Ready for deployment

---
