# TODOS

## Completed
- [x] Fix Postgres connection issues when SSL is enabled

## Bugs & Fixes

### Database Connections
- [x] Fix: Test connection for Postgres with SSL enabled is not working

### Change Plans
- [ ] Fix: when creating Change Plan with invalid SQL, it shows the creation failed, but actually created it. Improve to show better error feedback.
- [ ] Fix: Do not create a changelog entry for every run of the same Change Plan. Keep only one for each Change Plan and just update the status.
- [x] Fix: Gracefully handle Change Plan execution timeouts and show proper error message to users. Queue and Locks executions seems to be in deadlock state when timeouts or unexpected errors happen. (Fixed via context.WithoutCancel detachment — execution/cleanup no longer dies with the HTTP request context; ghost executor additionally gets its own generous timeout window)

### Schema Analysis (External: nsxbet/sql-schema)
- [ ] Fix: `failed to ANALYZE table schema=public` (https://github.com/nsxbet/sql-schema)
- [ ] Fix: failing to parse constraints (https://github.com/nsxbet/sql-schema)

### RBAC & Permissions
- [x] Fix: project roles permissions (ProjectDeveloper now has approve/reject/execute)
- [x] Fix: Approve button not showing for developers role in projects

### UI & Data Refresh
- [ ] Fix: refresh databases list on current project change
- [ ] Fix: refetch resources on create/update actions
- [x] Fix: Policy is showing none in environments even when is set.
- [x] Fix: Policy rules not being sent to API when editing (mutation bug)
- [ ] Fix: Search database by name filter not working
- [ ] Fix: Show proper error message when deleting a database instance

## Features

### Ghost / Online Migrator
- [ ] Fix: progress bar regresses 100% → 99.9% at cutover — when the copy
  finishes, gh-ost's final "Copy: ... 100.0%" line is persisted (bar shows
  100%), then the plan flips to AwaitingCutover and the UI overrides to
  99.9% (semantic "not done yet"). The visible backwards jump confuses
  users. Fix direction: never render a snapshot pct ≥ our status ceiling —
  clamp the bar to 99.9 max while copying (or treat copy-100 as "waiting
  for cutover" visually, e.g. keep bar at 99.9 with a 'copy complete'
  label), so the bar never goes backwards.
- [ ] Feat: livelier ghost progress bar — the static blue fill can look
  frozen during slow phases. Add subtle motion (striped/indeterminate-style
  shimmer or animated gradient) while InProgress, and richer color
  transitions across phases (copy blue → cutover amber → cleanup orange →
  completed green) so the user always sees the process is alive.
- [ ] Feat: replace 1s progress polling with server-push — current design
  polls GET /change-plans/:id every 1s during active phases; move to SSE
  (text/event-stream per plan, in-process publish hub fed by the gh-ost log
  callback, initial snapshot from ghost_progress, heartbeat for ALB idle
  timeouts). Design notes: .todos/ghost-progress-server-push.md
- [ ] Feat: one-click cleanup of stale gh-ost artifacts for admins — when a
  ghost run fails (or is retried) and gh-ost aborts with "table `_<table>_gho`
  already exists" (or a leftover `_<table>_del` backup), an admin should be
  able to drop/rename the artifact from the UI with a single button instead
  of calling SRE. Scope: new endpoint (drop `_<table>_gho` and/or
  `_<table>_del` on the target instance, gated by an admin permission +
  typed confirmation), button surfaced near the Failed banner when
  error_message matches the known gh-ost artifact patterns. Complements the
  existing ghost-drop-backup action (which only covers post-completion).

### Filters & Navigation
- [ ] Feat: add filters to URL query params like current project, tables filter, etc.
- [ ] Feat: update data table filters logic to persist selected filters in URL params so they are preserved on page refresh

### User Management
- [ ] Feat: Set default projects users should join on sign up

### Admin Capabilities
- [ ] Feat: allow admin to change db project and environment

### UX 

- [ ] Feat: add copy-button component to copy text to clipboard (`pnpm dlx shadcn@latest add https://www.shadcn.io/registry/copy-button.json`) and create a component for copyable text fields.

### Integrations
- [ ] Feat: integrate Catalog with SQL Reviewer to provide schema analysis for SQL Reviewer
- [x] Feat: ghost execution backend (gh-ost) for MySQL DDL change plans on non-production environments — rebranded "Online Migrator": executor picker with per-rule ineligibility reasons and dynamic descriptions, executor warnings (no-ALTER/mixed SQL), readiness checks (binlog/privileges/dry-run/MDL instrument per plan), human-gated cutover (AwaitingCutover), async execution with live progress bar (rows/elapsed/%), AwaitingCleanup completion gate (drop-backup completes the plan), rollback, dry-run artifact cleanup, execution log InProgress status
- [ ] Feat: pg-online-schema-change support for PostgreSQL ghost plans (currently MySQL-only; selection gate rejects postgres)
- [ ] Feat: RDS replica discovery for ghost plans — enumerate RDS replicas via AWS API and let the user choose which instance gh-ost runs against (currently runs on the primary with --allow-on-master)
- [ ] Feat: CloudWatch FreeStorageSpace check for ghost disk headroom (currently an information_schema size estimate shown as a warning; operator verifies actual free disk)
- [ ] Feat: ghost live log tail endpoint — in-memory ring buffer in GhostExecutor exposed via GET /:id/ghost-log (Option 2 from progress discussion)
- [ ] Feat: async ghost readiness checks job — run the check suite (especially the slow dry_run) as a background job with a status table + polling endpoint so the UI can show real progress; today checks are synchronous (spinners only) and dry-run progress is only visible in backend logs
- [ ] Feat: dedicated ghost user per database instance — separate `ghost` user (ALL + replication grants) used only by the ghost executor, so deeb's primary user can go least-privilege later; plan in `.todos/ghost-dedicated-user-plan.md`
- [ ] Feat: least-privilege deeb user pass — once the dedicated ghost user is provisioned everywhere, drop REPLICATION SLAVE/CLIENT (and review DML scope) from the primary deeb user created by the onboarder
