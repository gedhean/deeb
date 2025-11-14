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
    - [ ] Use this pkg to generate schema metadata
    - [ ] Use this pkg to generate schema snapshot
- [ ] Create database schema versions for schema change history
    - [ ] UI to view schema history
    - [ ] Button to set current schema as baseline version
    - [ ] Create database schema snapshot only after schema change is applied (not before AND after)
- [ ] Enhance Environment:
    - [x] Add "Allow Self Approval" checkbox to allow developers to approve their own changes without approval from a reviewer
    - [ ] Add "Require Approval" checkbox to require approval from a reviewer for all changes

### P2 (Low Priority)

- [ ] Use Monaco Editor for SQL editor (https://github.com/react-monaco-editor/react-monaco-editor)
- [x] Update the Change Plan actions buttons to show a single button for main action based on status (Draft, Open, Approved, Closed, InProgress, Completed, Failed, Cancelled) and a three dots dropdown for other actions.
- [ ] Update Tables to have pagination.
- [ ] Create Helm Chart for Deeb deployment
- [ ] Create Deeb CI/CD pipeline with GitHub Actions
- [ ] SQL Review: get syntax errors from sql-reviewer pkg


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
