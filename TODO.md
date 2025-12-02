# TODOS

## Completed
- [x] Fix Postgres connection issues when SSL is enabled

## Bugs & Fixes

### Database Connections
- [x] Fix: Test connection for Postgres with SSL enabled is not working

### Change Plans
- [x] Feat: Allow run more than one statement per Change Plan
- [ ] Fix: when creating Change Plan with invalid SQL, it shows the creation failed, but actually created it. Improve to show better error feedback.
- [ ] Fix: Do not create a changelog entry for every run of the same Change Plan. Keep only one for each Change Plan and just update the status.

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
