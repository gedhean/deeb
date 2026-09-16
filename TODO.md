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
- [x] Feat: ghost execution backend (gh-ost) for MySQL DDL change plans on non-production environments — user-selected executor, readiness checks (binlog/privileges/dry-run per plan), human-gated cutover (AwaitingCutover), copy progress bar, rollback + drop-backup actions
- [ ] Feat: pg-online-schema-change support for PostgreSQL ghost plans (currently MySQL-only; selection gate rejects postgres)
- [ ] Feat: RDS replica discovery for ghost plans — enumerate RDS replicas via AWS API and let the user choose which instance gh-ost runs against (currently runs on the primary with --allow-on-master)
- [ ] Feat: CloudWatch FreeStorageSpace check for ghost disk headroom (currently an information_schema size estimate shown as a warning; operator verifies actual free disk)
- [ ] Feat: ghost live log tail endpoint — in-memory ring buffer in GhostExecutor exposed via GET /:id/ghost-log (Option 2 from progress discussion)
- [ ] Feat: async ghost readiness checks job — run the check suite (especially the slow dry_run) as a background job with a status table + polling endpoint so the UI can show real progress; today checks are synchronous (spinners only) and dry-run progress is only visible in backend logs
- [ ] Feat: dedicated ghost user per database instance — separate `ghost` user (ALL + replication grants) used only by the ghost executor, so deeb's primary user can go least-privilege later; plan in `.todos/ghost-dedicated-user-plan.md`
- [ ] Feat: least-privilege deeb user pass — once the dedicated ghost user is provisioned everywhere, drop REPLICATION SLAVE/CLIENT (and review DML scope) from the primary deeb user created by the onboarder
