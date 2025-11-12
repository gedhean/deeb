# TODOS

### P0 (High Priority)

- [x] Change SQL Review engine to use https://github.com/nsxbet/sql-reviewer pkg
- [x] Update the create policy UI to use https://github.com/nsxbet/sql-reviewer rules

### P1 (Medium Priority)

- [ ] Add "Create Database" button in Database Instance Detail page to create a new database for the instance
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
