---
description: Create git worktree, plan task, implement with todos, run quality checks, create PR, and cleanup worktree
tags: [project]
---

# Parallel Task Workflow Command

You are helping the user complete a development task in an isolated git worktree following the Deeb project workflow.

## Task Description
The user wants to: {{{ input }}}

## Workflow Steps

Follow these steps in order:

### 1. Create Git Worktree with New Branch
- Generate a descriptive branch name based on the task (e.g., `feature/add-user-auth`, `fix/sql-validation-bug`, `refactor/api-cleanup`)
- Create a worktree directory name based on the branch (e.g., `../deeb-feature-add-user-auth`)
- Create the worktree with new branch: `git worktree add -b <branch-name> <worktree-path>`
- Change to the worktree directory: `cd <worktree-path>`
- Confirm the worktree was created successfully with: `git worktree list`

### 2. Gather Requirements & Plan
- **Ask clarifying questions** using the AskUserQuestion tool to understand:
  - Task scope and requirements
  - Which parts of the codebase are affected (frontend, backend, both)
  - Any specific implementation preferences
  - Expected behavior and edge cases
  - Whether this affects databases, APIs, UI, or all layers
- **Create a phased implementation plan** that breaks down the work into logical phases
- **Save the plan** to `.todos/<branch-name>-plan.md` (in the worktree directory) using this format:

```markdown
# Task: <Task Description>

## Overview
<Brief description of what will be accomplished>

### Phase 1: <What needs to be done>
**Status**: Open

**Todo**: <Description of work to be done in this phase>

**Acceptance Criteria**
- [ ] Acceptance criterion 1
- [ ] Acceptance criterion 2
- [ ] Acceptance criterion 3

### Phase 2: <What needs to be done>
**Status**: Open

**Todo**: <Description of work to be done in this phase>

**Acceptance Criteria**
- [ ] Acceptance criterion 1
- [ ] Acceptance criterion 2

### Phase 3: <What needs to be done>
**Status**: Open

**Todo**: <Description of work to be done in this phase>

**Acceptance Criteria**
- [ ] Acceptance criterion 1
- [ ] Acceptance criterion 2
```

- **Inform the user** that the plan has been saved to `.todos/<branch-name>-plan.md` and they can review/edit it before you proceed
- **Wait for user approval** on the plan before proceeding
- **Do NOT use TodoWrite until the user confirms the plan is ready to implement**

### 3. Execute Phases with Todo Tracking
Once the user approves the plan:
- Use the TodoWrite tool to create todos based on all phases in the approved plan
- Each todo should reference its phase and include the phase's acceptance criteria

### 4. Implementation
- Work through each phase systematically in order
- For each phase:
  - Update the phase status to `InProgress` in the plan file
  - Mark the corresponding todo as in_progress when starting the phase
  - Complete the work according to the acceptance criteria
  - Update all acceptance criteria checkboxes to `[x]` when completed in the plan file
  - Mark the todo as completed when the phase is done
  - Update the phase status to `Complete` in the plan file
  - Move to the next phase
- If you encounter blockers or need clarification, ask the user before proceeding
- Follow the project's development practices from CLAUDE.md
- Keep the `.todos/<branch-name>-plan.md` file updated as you progress through phases

### 5. Quality Checks
After implementation is complete, run the appropriate quality tools based on what was changed:

**If Backend (Go) code was modified:**
- Run tests: `make -C backend test` (or specific test files)
- Run linting: `make -C backend lint`
- Run autofix: `make -C backend lint-fix`
- Run formatting: `make -C backend format`

**If Frontend code was modified:**
- Run tests: `pnpm --dir frontend run test`
- Run linting: `pnpm --dir frontend run lint`
- Run autofix: `pnpm --dir frontend run lint-fix`
- Run formatting: `pnpm --dir frontend run format`
- Build check: `pnpm --dir frontend build`

**Fix any issues found** by the quality tools before proceeding.

### 6. Create Pull Request
- Review all changes made: `git status` and `git diff`
- Stage all changes: `git add .`
- Create a commit with a descriptive message following this format:
  ```
  <type>: <short description>

  <detailed description of changes>
  <why these changes were made>
  <any breaking changes or important notes>

  🤖 Generated with [Claude Code](https://claude.com/claude-code)

  Co-Authored-By: Claude <noreply@anthropic.com>
  ```
- Push the branch: `git push -u origin <branch-name>`
- Create PR using gh CLI with:
  - Clear title summarizing the task
  - Body including:
    - ## Summary: What was done and why
    - ## Changes: List of key changes
    - ## Testing: How to test the changes
    - ## Acceptance Criteria: Checklist of requirements met
    - Generated with Claude Code footer
- Return the PR URL to the user

### 7. Cleanup Worktree
- Return to the main repository: `cd <original-directory>`
- Remove the worktree: `git worktree remove <worktree-path>`
- Confirm cleanup with: `git worktree list`
- Inform the user that the worktree has been cleaned up and the PR is ready for review

## Important Notes

- **Always ask questions** before implementing if anything is unclear
- **Use TodoWrite tool** to track progress throughout
- **Mark todos as completed** immediately after finishing each one
- **Run all quality checks** appropriate for the code changed
- **Follow project conventions** from CLAUDE.md
- **Test thoroughly** before creating the PR
- **Use gh CLI** for GitHub operations (per CLAUDE.md instructions)
- **Work in isolation**: The worktree provides a clean workspace separate from your main repository
- **Clean up properly**: Always remove the worktree after PR creation to avoid clutter

## Error Handling

- If tests fail: Fix the issues before proceeding to PR creation
- If linting fails: Run autofix, then manual fixes if needed
- If git operations fail: Check branch status and resolve conflicts
- If worktree creation fails: Check if branch name already exists or worktree path conflicts
- If unclear about requirements: Always ask the user rather than making assumptions
- If worktree removal fails: Check if there are uncommitted changes or if the path is in use

## Worktree Benefits

- **Parallel Development**: Work on multiple features simultaneously without stashing or switching branches
- **Clean Isolation**: Each task has its own workspace without affecting your main repository
- **No Context Switching**: Keep your main workspace clean while experimenting in worktrees
- **Easy Cleanup**: Simply remove the worktree when done

Begin by creating the git worktree with a new branch and then asking clarifying questions about the task.
