# Phase 4: Frontend Implementation

**Duration**: ~5 days (40 milestones @ 1h each)
**Dependencies**: Phase 3 (Database Integration & Change Execution)
**Status**: 🚧 TODO

## Goal

Build the complete React frontend application with Vite, Shadcn/ui components, and all user-facing features. This phase delivers the web interface for issue management, SQL editing, approval workflows, and audit log viewing. The UI will be responsive, accessible (WCAG 2.1 Level AA), and provide real-time validation feedback.

## Phase-level Acceptance Criteria

- React application initialized with Vite and TypeScript configuration
- Shadcn/ui component library integrated with Tailwind CSS
- Responsive layout working on desktop and tablet devices
- Authentication pages implemented (login, OAuth callback, registration)
- Issue creation form with SQL editor and validation feedback
- SQL editor with syntax highlighting for MySQL and PostgreSQL (Monaco Editor or CodeMirror)
- Real-time validation results displayed inline in editor
- Issue list page with search and filter capabilities
- Issue detail page with full history timeline and status tracking
- Approval dashboard showing pending approvals
- Approval workflow UI with comment capability
- Multi-stage promotion visualization (Staging -> Production)
- Audit log viewer with filtering and search functionality
- Environment management UI for database configuration
- User management interface for admins
- Notification system UI (Slack and Webhook configuration)
- Dry-run execution preview interface
- Scheduled change UI for setting execution time
- Error boundaries implemented for graceful error handling
- Loading states and optimistic UI updates
- API integration using Fetch API or Axios with error handling
- Frontend bundle size under 500KB (gzipped)
- ESLint and Prettier configured and passing
- Accessibility audit passing (aXe or similar tool)

## Milestones

<!-- Milestones appended by /wiz:milestones -->
### P04M01: Create dashboard layout component

**Status:** 🚧 TODO
**ID:** P04M01

**Goal**
Build main dashboard layout with navigation, header, and content area.

**Acceptance Criteria**
- [ ] DashboardLayout component created
- [ ] Navigation sidebar with routing
- [ ] Header with user profile and logout
- [ ] Responsive layout for desktop and tablet
- [ ] Content area for page components
- [ ] Component renders correctly

---

### P04M02: Create issue list page

**Status:** 🚧 TODO
**ID:** P04M02

**Goal**
Build issue list page with table view and pagination.

**Acceptance Criteria**
- [ ] IssueList component with table display
- [ ] Columns: title, status, author, created date, actions
- [ ] Pagination controls
- [ ] Loading state while fetching data
- [ ] Error handling for API failures
- [ ] Component renders correctly

---

### P04M03: Add issue list filtering

**Status:** 🚧 TODO
**ID:** P04M03

**Goal**
Implement filter controls for issue list (status, author, date range).

**Acceptance Criteria**
- [ ] Filter controls using Shadcn Select and DatePicker
- [ ] Filter by status dropdown
- [ ] Filter by author dropdown
- [ ] Date range picker
- [ ] Apply/Clear filter buttons
- [ ] Filters update query parameters

---

### P04M04: Add issue list search

**Status:** 🚧 TODO
**ID:** P04M04

**Goal**
Implement full-text search for issues by title and description.

**Acceptance Criteria**
- [ ] Search input with debounce (300ms)
- [ ] Search triggers API call with query parameter
- [ ] Search clears on clear button
- [ ] Loading indicator during search
- [ ] Search works with existing filters

---

### P04M05: Create issue detail page

**Status:** 🚧 TODO
**ID:** P04M05

**Goal**
Build issue detail page showing all issue information and history.

**Acceptance Criteria**
- [ ] IssueDetail component displays title, description, SQL content
- [ ] Shows issue status with visual indicator
- [ ] Displays author and timestamps
- [ ] Shows target database and environment
- [ ] Validation results displayed with severity colors
- [ ] Component renders correctly

---

### P04M06: Add issue history timeline

**Status:** 🚧 TODO
**ID:** P04M06

**Goal**
Display timeline of all changes and approvals for an issue.

**Acceptance Criteria**
- [ ] Timeline component with chronological events
- [ ] Shows issue creation, updates, approval requests, approvals, executions
- [ ] Each event has icon, timestamp, user, and description
- [ ] Color-coded by event type
- [ ] Sorted newest to oldest

---

### P04M07: Create SQL editor component with Monaco

**Status:** 🚧 TODO
**ID:** P04M07

**Goal**
Integrate Monaco Editor for SQL editing with syntax highlighting.

**Acceptance Criteria**
- [ ] Monaco Editor installed and configured
- [ ] SQL syntax highlighting for MySQL and PostgreSQL
- [ ] Line numbers and minimap enabled
- [ ] Theme matches application design
- [ ] Editor height adjustable
- [ ] Read-only mode for viewing

---

### P04M08: Add real-time validation to SQL editor

**Status:** 🚧 TODO
**ID:** P04M08

**Goal**
Display validation results inline in SQL editor as user types.

**Acceptance Criteria**
- [ ] Debounced validation API calls (500ms)
- [ ] Validation markers in editor gutter
- [ ] Error/warning/info decorations
- [ ] Hover tooltips show validation messages
- [ ] Validation summary below editor
- [ ] Loading indicator during validation

---

### P04M09: Create issue creation form

**Status:** 🚧 TODO
**ID:** P04M09

**Goal**
Build form for creating new database change issues.

**Acceptance Criteria**
- [ ] Form with title, description, SQL content fields
- [ ] Database type selector (MySQL/PostgreSQL)
- [ ] Target environment selector
- [ ] Form validation (required fields)
- [ ] Submit button creates issue
- [ ] Success message and redirect to issue detail

---

### P04M10: Add issue edit functionality

**Status:** 🚧 TODO
**ID:** P04M10

**Goal**
Allow editing of issues in Draft status.

**Acceptance Criteria**
- [ ] Edit button on issue detail page (Draft only)
- [ ] Edit mode enables form fields
- [ ] Save button updates issue via API
- [ ] Cancel button discards changes
- [ ] Only author can edit
- [ ] Status check prevents editing non-Draft issues

---

### P04M11: Create approval dashboard page

**Status:** 🚧 TODO
**ID:** P04M11

**Goal**
Build dashboard showing pending approvals for current user.

**Acceptance Criteria**
- [ ] ApprovalDashboard component lists pending approvals
- [ ] Shows issue title, author, requested date
- [ ] Quick approve/reject actions
- [ ] Filter by date range
- [ ] Pagination for large lists
- [ ] Component renders correctly

---

### P04M12: Create approval detail modal

**Status:** 🚧 TODO
**ID:** P04M12

**Goal**
Build modal for reviewing and approving/rejecting issues.

**Acceptance Criteria**
- [ ] Modal displays full issue details
- [ ] Shows SQL content with syntax highlighting
- [ ] Displays validation results
- [ ] Approve/Reject buttons
- [ ] Comment textarea (required for rejection)
- [ ] Submission updates approval status

---

### P04M13: Add approval comment functionality

**Status:** 🚧 TODO
**ID:** P04M13

**Goal**
Allow approvers to add comments during review.

**Acceptance Criteria**
- [ ] Comment textarea in approval modal
- [ ] Character count (max 1000 chars)
- [ ] Required for rejection
- [ ] Optional for approval
- [ ] Comments displayed in issue history
- [ ] Rich text preview

---

### P04M14: Create environment management page

**Status:** 🚧 TODO
**ID:** P04M14

**Goal**
Build admin page for managing database environments (Admin only).

**Acceptance Criteria**
- [ ] EnvironmentManagement component lists environments
- [ ] Add/Edit/Delete environment actions
- [ ] Environment form: name, database type, connection details
- [ ] Test connection button
- [ ] Admin permission check
- [ ] Component renders correctly

---

### P04M15: Create environment configuration form

**Status:** 🚧 TODO
**ID:** P04M15

**Goal**
Build form for configuring database connection details.

**Acceptance Criteria**
- [ ] Form fields: host, port, database name, auth method
- [ ] Conditional fields based on auth method (IAM vs credentials)
- [ ] Password field with show/hide toggle
- [ ] SSL checkbox
- [ ] Test connection validates configuration
- [ ] Success/error feedback

---

### P04M16: Create user management page

**Status:** 🚧 TODO
**ID:** P04M16

**Goal**
Build admin page for managing users (Admin only).

**Acceptance Criteria**
- [ ] UserManagement component lists users
- [ ] Table shows email, role, status, created date
- [ ] Add/Edit/Deactivate user actions
- [ ] Role selector (Developer, Approver, Admin)
- [ ] Admin permission check
- [ ] Pagination for large user lists

---

### P04M17: Create user form component

**Status:** 🚧 TODO
**ID:** P04M17

**Goal**
Build form for creating and editing users.

**Acceptance Criteria**
- [ ] Form fields: email, role, password (create only)
- [ ] Email validation
- [ ] Password strength indicator (creation)
- [ ] Role dropdown
- [ ] Submit creates or updates user
- [ ] Form validation and error handling

---

### P04M18: Create audit log viewer page

**Status:** 🚧 TODO
**ID:** P04M18

**Goal**
Build page for viewing comprehensive audit logs (Admin only).

**Acceptance Criteria**
- [ ] AuditLogViewer component displays logs in table
- [ ] Columns: timestamp, user, action, resource, details
- [ ] Filter by user, action type, date range
- [ ] Search functionality
- [ ] Export to JSON button
- [ ] Pagination

---

### P04M19: Add audit log detail modal

**Status:** 🚧 TODO
**ID:** P04M19

**Goal**
Create modal showing detailed information for an audit log entry.

**Acceptance Criteria**
- [ ] Modal displays full audit log details
- [ ] JSON viewer for details field
- [ ] Formatted timestamp
- [ ] User information
- [ ] Resource type and ID
- [ ] Close button

---

### P04M20: Create notification settings page

**Status:** 🚧 TODO
**ID:** P04M20

**Goal**
Build page for configuring Slack and Webhook notifications.

**Acceptance Criteria**
- [ ] NotificationSettings component
- [ ] Slack webhook URL input
- [ ] Test Slack notification button
- [ ] Webhook URL input
- [ ] Test webhook button
- [ ] Save settings button
- [ ] Success/error feedback

---

### P04M21: Add dry-run execution UI

**Status:** 🚧 TODO
**ID:** P04M21

**Goal**
Create UI for previewing changes without executing them.

**Acceptance Criteria**
- [ ] Dry-run button on issue detail page
- [ ] Modal shows estimated impact
- [ ] Lists affected tables and columns
- [ ] Shows execution plan
- [ ] Close button
- [ ] Loading state during dry-run

---

### P04M22: Create scheduled execution form

**Status:** 🚧 TODO
**ID:** P04M22

**Goal**
Build form for scheduling delayed change execution.

**Acceptance Criteria**
- [ ] DateTimePicker for execution time
- [ ] Timezone selector
- [ ] Minimum delay validation (5 minutes from now)
- [ ] Schedule button
- [ ] Confirmation modal
- [ ] Shows scheduled time on issue detail

---

### P04M23: Add multi-stage promotion visualization

**Status:** 🚧 TODO
**ID:** P04M23

**Goal**
Create visual representation of Staging -> Production promotion flow.

**Acceptance Criteria**
- [ ] Stage flow diagram showing Staging and Production
- [ ] Visual indicators for completed/pending stages
- [ ] Promote button for approved Staging changes
- [ ] Approval required indicator for Production
- [ ] Status updates after promotion
- [ ] Component renders correctly

---

### P04M24: Create execution results display

**Status:** 🚧 TODO
**ID:** P04M24

**Goal**
Display execution results on issue detail page.

**Acceptance Criteria**
- [ ] Execution results card shows status, time, rows affected
- [ ] Error message display for failures
- [ ] Rollback button for failed executions
- [ ] Execution history list
- [ ] Timestamp for each execution attempt
- [ ] Component renders correctly

---

### P04M25: Implement error boundary components

**Status:** 🚧 TODO
**ID:** P04M25

**Goal**
Add error boundaries for graceful error handling.

**Acceptance Criteria**
- [ ] ErrorBoundary component catches React errors
- [ ] Fallback UI shows error message
- [ ] Reset button to recover
- [ ] Errors logged to console
- [ ] Applied to major routes
- [ ] Component tested with intentional errors

---

### P04M26: Add loading states throughout app

**Status:** 🚧 TODO
**ID:** P04M26

**Goal**
Implement consistent loading indicators for async operations.

**Acceptance Criteria**
- [ ] Skeleton screens for list pages
- [ ] Spinner for buttons during actions
- [ ] Progress bar for file uploads
- [ ] Loading overlay for modals
- [ ] Consistent styling across app
- [ ] Accessible loading indicators

---

### P04M27: Implement optimistic UI updates

**Status:** 🚧 TODO
**ID:** P04M27

**Goal**
Add optimistic updates for better perceived performance.

**Acceptance Criteria**
- [ ] Issue status updates immediately
- [ ] Approval actions update instantly
- [ ] Rollback on API error
- [ ] Loading indicator during sync
- [ ] Toast notifications for success/error
- [ ] Consistent behavior across actions

---

### P04M28: Add toast notification system

**Status:** 🚧 TODO
**ID:** P04M28

**Goal**
Implement toast notifications for user feedback.

**Acceptance Criteria**
- [ ] Toast component using Shadcn Toast
- [ ] Success, error, warning, info variants
- [ ] Auto-dismiss after 5 seconds
- [ ] Dismiss button
- [ ] Stack multiple toasts
- [ ] Accessible announcements

---

### P04M29: Implement keyboard shortcuts

**Status:** 🚧 TODO
**ID:** P04M29

**Goal**
Add keyboard shortcuts for common actions.

**Acceptance Criteria**
- [ ] Cmd/Ctrl + K for search
- [ ] Cmd/Ctrl + N for new issue
- [ ] Cmd/Ctrl + S to save
- [ ] Esc to close modals
- [ ] Help modal showing shortcuts (Cmd/Ctrl + /)
- [ ] Shortcuts documented

---

### P04M30: Add dark mode support

**Status:** 🚧 TODO
**ID:** P04M30

**Goal**
Implement dark mode theme toggle.

**Acceptance Criteria**
- [ ] Theme provider with light/dark modes
- [ ] Theme toggle button in header
- [ ] Theme preference saved to localStorage
- [ ] All components support both themes
- [ ] Smooth transitions between themes
- [ ] System preference detection

---

### P04M31: Implement accessibility features

**Status:** 🚧 TODO
**ID:** P04M31

**Goal**
Ensure app meets WCAG 2.1 Level AA standards.

**Acceptance Criteria**
- [ ] All interactive elements keyboard accessible
- [ ] Focus indicators visible
- [ ] ARIA labels on all inputs
- [ ] Screen reader announcements for dynamic content
- [ ] Color contrast ratios meet AA standards
- [ ] aXe accessibility audit passing

---

### P04M32: Add form validation helpers

**Status:** 🚧 TODO
**ID:** P04M32

**Goal**
Create reusable form validation utilities.

**Acceptance Criteria**
- [ ] Validation functions for email, password, required fields
- [ ] Error message display component
- [ ] Real-time validation on blur
- [ ] Submit validation before API call
- [ ] Accessible error messages
- [ ] Reusable across forms

---

### P04M33: Optimize bundle size

**Status:** 🚧 TODO
**ID:** P04M33

**Goal**
Ensure frontend bundle size is under 500KB (gzipped).

**Acceptance Criteria**
- [ ] Code splitting for routes
- [ ] Lazy loading for heavy components
- [ ] Tree shaking configured
- [ ] Bundle analysis shows breakdown
- [ ] Total gzipped size < 500KB
- [ ] Performance metrics captured

---

### P04M34: Configure ESLint and Prettier

**Status:** 🚧 TODO
**ID:** P04M34

**Goal**
Set up code quality tools for TypeScript and React.

**Acceptance Criteria**
- [ ] ESLint configured with React and TypeScript rules
- [ ] Prettier configured for consistent formatting
- [ ] Pre-commit hook runs linting
- [ ] No linting errors in codebase
- [ ] Editor integration documented
- [ ] Scripts added to package.json

---

### P04M35: Write component unit tests

**Status:** 🚧 TODO
**ID:** P04M35

**Goal**
Create unit tests for React components using Vitest or Jest.

**Acceptance Criteria**
- [ ] Test framework configured
- [ ] Tests for major components (IssueList, IssueDetail, SQLEditor)
- [ ] Mock API calls
- [ ] Test user interactions
- [ ] All tests passing
- [ ] Code coverage > 70%

---

### P04M36: Write integration tests for user flows

**Status:** 🚧 TODO
**ID:** P04M36

**Goal**
Create end-to-end tests for critical user workflows.

**Acceptance Criteria**
- [ ] Test framework configured (Playwright or Cypress)
- [ ] Test: Login -> Create Issue -> Submit for Approval
- [ ] Test: Approve Issue -> Execute Change
- [ ] Test: Search and Filter Issues
- [ ] All tests passing
- [ ] CI integration

---

### P04M37: Add responsive design for mobile

**Status:** 🚧 TODO
**ID:** P04M37

**Goal**
Ensure app is usable on tablet devices (mobile optional).

**Acceptance Criteria**
- [ ] Responsive breakpoints configured
- [ ] Navigation collapses on small screens
- [ ] Tables scroll horizontally on narrow viewports
- [ ] Forms stack vertically on mobile
- [ ] Touch targets minimum 44x44px
- [ ] Tested on iPad and Android tablet

---

### P04M38: Create frontend documentation

**Status:** 🚧 TODO
**ID:** P04M38

**Goal**
Document frontend architecture, components, and development workflow.

**Acceptance Criteria**
- [ ] README with setup instructions
- [ ] Component documentation (Storybook optional)
- [ ] State management documentation
- [ ] API integration guide
- [ ] Styling guide
- [ ] Contribution guidelines

---

### P04M39: Add performance monitoring

**Status:** 🚧 TODO
**ID:** P04M39

**Goal**
Implement performance monitoring for frontend metrics.

**Acceptance Criteria**
- [ ] Web Vitals tracking (LCP, FID, CLS)
- [ ] API request duration tracking
- [ ] Error tracking
- [ ] Metrics sent to backend or external service
- [ ] Performance dashboard (optional)
- [ ] Alerts for degradation

---

### P04M40: Verify Phase 4 Completion

**Status:** 🚧 TODO
**ID:** P04M40

**Goal**
Verify that all Phase 4 requirements are met, tests pass, and the phase is ready for sign-off before proceeding to Phase 5.

**Acceptance Criteria**
- [ ] All previous milestones in Phase 4 marked as complete
- [ ] All tests pass (unit and e2e)
- [ ] Code coverage meets project standards (>70%)
- [ ] No outstanding bugs or issues from this phase
- [ ] ESLint and Prettier checks passing
- [ ] Bundle size under 500KB (gzipped)
- [ ] Accessibility audit passing (aXe)
- [ ] All major features working end-to-end
- [ ] Responsive design verified on desktop and tablet
- [ ] Ready to proceed to Phase 5

---
