# TODOS

### P0 (High Priority)

- [x] Change SQL Review engine to use https://github.com/nsxbet/sql-reviewer pkg
- [x] Update the create policy UI to use https://github.com/nsxbet/sql-reviewer rules

### P1 (Medium Priority)

- [x] Update "Create Database" action in Database Instance Detail page to create a new database for the instance (or import if it already exists)
    - [x] Update the CreateDatabaseDialog to use Sheet component instead of Dialog component
    - [x] Update the Create Database button in Databases Page to use the CreateDatabaseDialog (with instance selector)
- [x] Fix UI POV for Project roles (ProjectOwner, ProjectDeveloper, ProjectApprover, ProjectViewer)
- [ ] Integrate with github.com/nsxbet/sql-schema to show database schema metadata in Database Detail page
    - [x] Do not sync schema metadata for inactive or pending databases
    - [x] Engine column should be shown in the tables list only for MySQL databases
    - [x] Upgrade sql-schema pkg to latest version
    - [x] Use this pkg to generate schema metadata
- [x] Environment: Add "Allow Self Approval" checkbox to allow developers to approve their own changes without approval from a reviewer

### P2 (Low Priority)

- [x] Use Monaco Editor for SQL editor (https://github.com/react-monaco-editor/react-monaco-editor)
- [x] Update the Change Plan actions buttons to show a single button for main action based on status (Draft, Open, Approved, Closed, InProgress, Completed, Failed, Cancelled) and a three dots dropdown for other actions.


### P3 (Low Priority)
- [x] Add Start Background to Login Page and Register Page
- [x] Simplify the Change Plan create page with fewer fields and better UI.
- [x] Non-admin users should be able to see only the projects they are members of
- [x] Update Change Plan create page to use useCurrentProject hook to get the current project ID
- [x] Omit project selector from Change Plan create page (set via useCurrentProject hook)
- [x] The database selector should be filtered to only show databases that belong to the current project
- [x] Fix: Create Change Plan is not working for non-admin users.
- [x] Fix: Non-admin users are getting forbidden error when trying to view Change Plan details.
- [x] Fix: Users should be able to see Databases that belong to the project they are members of
- [x] Fix: Changes Plans page should show all changes plans for filter project: all.
- [x] Database should be a project level resource (not organization level)
- [x] Update get request for project level resources use useCurrentProject hook to get the current project ID
- [x] Optimize Frontend APIs calls: reduce the number of API calls and optimize the data fetching logic (use store to reuse data and avoid duplicate API calls).

### Current

- [ ] Update Environments Page to use correct RBAC permissions
- [ ] Update SQL Review: SQL syntax built-in check to use sql-reviewer pkg (DML dry-run validation).
- [ ] Update SQL Review: connection to the database built-in check to use sql-reviewer pkg (connection validation).
- [ ] Create reusable Table component to have pagination by default.
- [ ] Update Tables to use the reusable Table component with pagination.
- [ ] Create Helm Chart for Deeb deployment
- [ ] Create Deeb CI/CD pipeline with GitHub Actions for deploy
- [ ] Create database schema versions for schema change history
    - [ ] UI to view schema history
    - [ ] Button to set current schema as baseline version
    - [ ] Create database schema snapshot only after schema change is applied (not before AND after)
    - [ ] Use this sql-schema pkg to generate schema snapshot and metadata for schema change history
- [ ] Refactor Audit Log to be at the middleware level with request/response and non-blocking run (async and do not block on error)
- [ ] Better handling for updating requests that set values to undefined or null (e.g. update database instance request). Currently, if I set a field to undefined or null, the field is not updated in the database.
- [ ] Update SQL Review rules to not use "both" for engine. Instead, use the specific engine.
- [ ] Fix: Database test connection should work for existing databases instances (not just new ones).
- [ ] Upgrade sql-reviewer pkg to latest version.
- [ ] Upgrade sql-schema pkg to latest version.
- [ ] Think how to handle large number of resource in the UI. Currenlty, we fetch with limit fixed and not offset. We need to handle this better.
