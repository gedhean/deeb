# TODOS

### P0 (High Priority)

- [x] Change SQL Review engine to use https://github.com/nsxbet/sql-reviewer pkg
- [x] Update the create policy UI to use https://github.com/nsxbet/sql-reviewer rules

### P1 (Medium Priority)

- [x] Update "Create Database" action in Database Instance Detail page to create a new database for the instance (or import if it already exists)
    - [x] Update the CreateDatabaseDialog to use Sheet component instead of Dialog component
    - [x] Update the Create Database button in Databases Page to use the CreateDatabaseDialog (with instance selector)
- [ ] Fix UI POV for Project roles (ProjectOwner, ProjectDeveloper, ProjectApprover, ProjectViewer)
    - [ ] ProjectOwner should be able to add/remove members
    - [ ] ProjectDeveloper should be able to create change requests
    - [ ] ProjectDeveloper should be able to create databases
    - [ ] ProjectApprover should be able to approve/reject change requests
    - [ ] ProjectViewer should be able to view change requests
    - [ ] Only show Organization settings in app-sidebar for OrganizationAdmins
- [ ] Integrate with github.com/nsxbet/sql-schema to show database schema metadata in Database Detail page
    - [ ] Use this pkg to generate schema metadata
    - [ ] Use this pkg to generate schema snapshot
- [ ] Create database schema versions for schema change history
    - [ ] UI to view schema history
    - [ ] Button to set current schema as baseline version
    - [ ] Create database schema snapshot only after schema change is applied (not before AND after)
- [ ] Enhance Environment:
    - [ ] Add "Allow Self Approval" checkbox to allow developers to approve their own changes without approval from a reviewer
    - [ ] Add "Require Approval" checkbox to require approval from a reviewer for all changes

### P2 (Low Priority)

- [ ] Use Monaco Editor for SQL editor (https://github.com/react-monaco-editor/react-monaco-editor)
- [ ] Update the Change Plan actions buttons to have single for main action based on status (Draft, Open, Approved, Closed, InProgress, Completed, Failed, Cancelled) and a three dots dropdown for other actions.
- [ ] Update Tables to have pagination.
- [ ] Create Helm Chart for Deeb deployment
- [ ] Create Deeb CI/CD pipeline with GitHub Actions
