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
- [ ] Create database schema versions for schema change history
    - [ ] UI to view schema history
    - [ ] Button to set current schema as baseline version
    - [ ] Create database schema snapshot only after schema change is applied (not before AND after)
    - [ ] Use this sql-schema pkg to generate schema snapshot and metadata for schema change history
- [x] Environment: Add "Allow Self Approval" checkbox to allow developers to approve their own changes without approval from a reviewer
- [ ] Environment: Add "Require Approval" checkbox to require approval from a reviewer for all changes

### P2 (Low Priority)

- [ ] Use Monaco Editor for SQL editor (https://github.com/react-monaco-editor/react-monaco-editor)
- [x] Update the Change Plan actions buttons to show a single button for main action based on status (Draft, Open, Approved, Closed, InProgress, Completed, Failed, Cancelled) and a three dots dropdown for other actions.
- [ ] Update Tables to have pagination by default.
- [ ] Create Helm Chart for Deeb deployment
- [ ] Create Deeb CI/CD pipeline with GitHub Actions
- [ ] SQL Review: SQL syntax built-in check
- [ ] SQL Review: connection to the database built-in check
- [ ] Optimize Frontend APIs calls: reduce the number of API calls and optimize the data fetching logic (use store to reuse data and avoid duplicate API calls).

### P3 (Low Priority)
- [x] Add Start Background to Login Page and Register Page
- [ ] Simplify the Change Plan create page with fewer fields and better UI.

### Current

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
