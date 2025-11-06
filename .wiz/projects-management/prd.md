---
title: "Project Member and Owner Management"
slug: "projects-management"
version: "1.0.0"
status: "Draft"
created: "2025-11-05"
owner: "TBD"
primary_language: "Go, TypeScript"
benchmarking_policy: "skip"
fuzzing_policy: "skip"
---

# Product Requirements Document: Project Member and Owner Management

## 1. Background

Deeb is a secure database change management tool that enables developers to manage database changes through an issue-based workflow. Currently, the system supports organization-wide roles and permissions, but lacks fine-grained project-level access control. As the system scales to support multiple projects (target: 50 projects), there is a need to delegate project management responsibilities to project owners while maintaining centralized administrative oversight.

This PRD defines the project member and owner management feature, which will enable:
- **Decentralized Project Management**: Project owners can manage their project members without requiring admin intervention
- **Project-Level Access Control**: Members have specific roles within projects (owner, approver, viewer, developer)
- **Administrative Oversight**: Admins retain the ability to override owner decisions and manage all projects
- **Audit Trail**: Full tracking of membership changes for compliance and troubleshooting

## 2. Problem Statement

### Current State
- Projects exist in the system but lack formal ownership and membership models
- Access control is organization-wide (global roles) rather than project-specific
- Admins must manually manage all project access, creating bottlenecks
- No clear ownership model for who is responsible for project decisions

### Desired State
- Projects have designated owners who can manage project membership
- Project members have project-specific roles that determine their permissions
- Admins can define owners and have override capabilities
- Full audit trail of membership changes
- Self-service project management for owners

### Success Metrics
- Reduce admin workload for project access management by 80%
- Enable project owners to add/remove members within 30 seconds
- Complete audit trail for all membership changes
- Support 50 concurrent projects with clear ownership

## 3. Goals and Non-Goals

### Goals
1. **Enable Project Ownership**: Allow multiple owners per project with full member management capabilities
2. **Project-Level RBAC**: Implement project-specific roles (owner, approver, viewer, developer)
3. **Admin Controls**: Provide admins with override capabilities and owner assignment
4. **Audit Trail**: Track all membership changes with timestamps, actors, and reasons
5. **User-Friendly UI**: Dedicated management page for viewing and modifying project membership
6. **Ownership Continuity**: Prevent orphaned projects by requiring explicit owner reassignment

### Non-Goals
- **Notification System**: No email/push notifications for membership changes (future enhancement)
- **Member Limits**: No restrictions on number of members per project
- **Hierarchical Projects**: No parent-child project relationships
- **External Identity Providers**: Use existing SSO integration (Google, GitHub OAuth)
- **Custom Role Creation**: Use predefined project roles only (extensibility for future)

## 4. User Personas and Use Cases

### Persona 1: System Administrator
**Description**: Organization-wide admin responsible for initial project setup and oversight

**Use Cases**:
- UC1.1: Assign initial owners to new projects
- UC1.2: Add/remove owners from any project (override)
- UC1.3: Add/remove members from any project (override)
- UC1.4: View membership across all projects
- UC1.5: Audit membership changes organization-wide
- UC1.6: Reassign ownership when owners leave organization

### Persona 2: Project Owner
**Description**: Developer responsible for a specific project's membership and decisions

**Use Cases**:
- UC2.1: View all members and their roles in owned projects
- UC2.2: Add members to owned projects with specific roles
- UC2.3: Remove members from owned projects
- UC2.4: Change member roles within owned projects
- UC2.5: View audit history of membership changes
- UC2.6: Cannot remove self if last owner (must reassign first)

### Persona 3: Project Member
**Description**: Developer with project-specific access and permissions

**Use Cases**:
- UC3.1: View projects they are members of
- UC3.2: View their role in each project
- UC3.3: Perform actions based on project role (owner, approver, viewer, developer)

## 5. Technical Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────┐
│                      Frontend (React)                    │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Project Management Page                          │  │
│  │  - Member List Component                          │  │
│  │  - Owner List Component                           │  │
│  │  - Add/Remove Actions                             │  │
│  │  - Audit History Component                        │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│              Backend API (Go + Echo)                     │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Project Management Handlers                      │  │
│  │  - RBAC Middleware (check project permissions)    │  │
│  │  - Validation Layer (prevent orphaned projects)   │  │
│  │  - Audit Logger                                   │  │
│  └──────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────┐  │
│  │  Project Member Service                           │  │
│  │  - Add/Remove/List Members                        │  │
│  │  - Role Management                                │  │
│  │  - Ownership Validation                           │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    Database (PostgreSQL)                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   projects   │  │project_owners│  │project_members│ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│  ┌──────────────┐  ┌──────────────┐                   │
│  │    users     │  │  audit_logs  │                   │
│  └──────────────┘  └──────────────┘                   │
└─────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Admin Assigns Owner**: Admin → API → `project_owners` table → Audit log
2. **Owner Adds Member**: Owner → API → RBAC check → `project_members` table → Audit log
3. **Member Views Projects**: Member → API → Join query → Return project list with roles
4. **Audit Query**: User → API → `audit_logs` table → Return filtered history

### Integration Points
- **Existing RBAC System**: Extend permission model to include project-level permissions
- **Existing Audit System**: Use existing audit log infrastructure
- **Existing SSO**: Use current user authentication (no changes)

## 6. Functional Requirements

### FR1: Project Ownership Management (Admin Only)

**FR1.1**: Admins can assign one or more users as owners to any project
- Input: Project ID, User IDs
- Validation: Users must exist in system
- Output: Success/failure, audit log entry

**FR1.2**: Admins can remove owners from any project
- Input: Project ID, User ID
- Validation: Cannot remove last owner without reassignment
- Output: Success/failure, audit log entry

**FR1.3**: System prevents orphaned projects
- Rule: Last owner removal requires explicit reassignment to another user
- Error: "Cannot remove last owner. Please assign a new owner first."

**FR1.4**: Admins can view all owners across all projects
- Output: List of projects with their owners
- Filters: By project, by user

### FR2: Project Member Management (Owners & Admins)

**FR2.1**: Owners can add members to their projects
- Input: Project ID, User ID, Role (approver/viewer/developer)
- Validation: Actor must be owner or admin
- Output: Success/failure, audit log entry

**FR2.2**: Owners can remove members from their projects
- Input: Project ID, User ID
- Validation: Actor must be owner or admin, cannot remove owners
- Output: Success/failure, audit log entry

**FR2.3**: Owners can change member roles
- Input: Project ID, User ID, New Role
- Validation: Actor must be owner or admin, cannot change owner role
- Output: Success/failure, audit log entry

**FR2.4**: Owners cannot remove themselves if they are the last owner
- Validation: Check owner count before removal
- Error: "Cannot remove yourself as last owner. Please assign another owner first."

### FR3: Project Role Model

**FR3.1**: System supports the following project roles:
- **Owner**: Full project control (add/remove members, manage settings, approve changes)
- **Approver**: Can approve database changes for the project
- **Developer**: Can create database change requests
- **Viewer**: Read-only access to project changes and status

**FR3.2**: Role hierarchy and permissions:
```
Owner > Approver > Developer > Viewer
```

**FR3.3**: Role assignment rules:
- Only owners and admins can assign roles
- Users can hold only one role per project
- Users can be members of multiple projects with different roles

**FR3.4**: Default role determination (TBD):
- To be determined during implementation
- Options: Default to "Developer" or require explicit selection

### FR4: Project Membership Queries

**FR4.1**: Users can view projects they are members of
- Output: List of projects with their role in each
- Sorting: By project name, by role, by join date

**FR4.2**: Owners can view all members of their projects
- Output: List of users with roles for each owned project
- Filters: By role, by join date

**FR4.3**: Admins can view membership across all projects
- Output: Global view of all project memberships
- Filters: By project, by user, by role
- Export: CSV for reporting

### FR5: Audit and History

**FR5.1**: System logs all membership changes:
- Event types: `owner_added`, `owner_removed`, `member_added`, `member_removed`, `role_changed`
- Captured data: Actor, target user, project, role (old/new), timestamp, IP address
- Storage: Permanent, immutable audit log

**FR5.2**: Users can view audit history:
- Owners: See history for their projects
- Admins: See history for all projects
- Members: See their own membership history

**FR5.3**: Audit log retention:
- Retention period: Permanent (no deletion)
- Compliance: Support for compliance reporting queries

### FR6: Admin Override Capabilities

**FR6.1**: Admins can perform all owner actions on any project
- Add/remove owners
- Add/remove members
- Change member roles
- View membership and audit logs

**FR6.2**: Admin actions are clearly marked in audit logs
- Flag: `is_admin_override: true`
- Reason field: Optional reason for override (future enhancement)

## 7. Non-Functional Requirements

### P0: Correctness (Critical)

**NFR-P0-1**: Prevent orphaned projects
- Validation: Always check owner count before removal
- Behavior: Reject last owner removal without reassignment
- Error handling: Clear error messages

**NFR-P0-2**: Role permission enforcement
- Validation: All member management actions check actor permissions
- Behavior: Reject unauthorized actions with 403 Forbidden
- Security: Prevent privilege escalation

**NFR-P0-3**: Data consistency
- Transaction: All membership changes in single transaction
- Rollback: Failed operations leave no partial state
- Integrity: Foreign key constraints prevent orphaned records

### P1: Tests (Critical)

**NFR-P1-1**: Backend unit tests
- Coverage: ≥80% for member management service
- Cases: Happy path, error cases, edge cases (last owner, role conflicts)
- Framework: Go testing package + testify

**NFR-P1-2**: Backend integration tests
- Scope: API endpoints with database
- Cases: End-to-end member management flows
- Fixtures: Test projects, users, roles

**NFR-P1-3**: Frontend component tests
- Coverage: ≥70% for member management UI components
- Cases: User interactions, permission checks, error states
- Framework: Vitest + React Testing Library

### P2: Security (Critical)

**NFR-P2-1**: Authorization checks
- Enforcement: Every API endpoint validates actor permissions
- Levels: Organization role (admin) + project role (owner)
- Bypass: No backdoors or permission skips

**NFR-P2-2**: Input validation
- Validation: User IDs, project IDs, roles validated against database
- Sanitization: Prevent SQL injection (use parameterized queries)
- Rate limiting: Prevent abuse (100 requests/minute per user)

**NFR-P2-3**: Audit logging
- Coverage: All membership changes logged
- Integrity: Logs immutable (append-only)
- Security: Logs include actor, target, action, timestamp, IP

**NFR-P2-4**: CSRF protection
- Mechanism: Use existing CSRF token implementation
- Enforcement: All POST/PUT/DELETE endpoints protected

### P3: Quality (High Priority)

**NFR-P3-1**: Code quality
- Linting: Zero Go lint errors (`make lint`)
- Formatting: Consistent style (`make format`)
- Frontend: Zero ESLint errors (`pnpm run lint`)

**NFR-P3-2**: Documentation
- API docs: OpenAPI/Swagger for all endpoints
- Code comments: Document complex logic and security decisions
- User guide: Help text in UI for project management

**NFR-P3-3**: Error handling
- User errors: Clear, actionable error messages
- System errors: Logged with context, generic message to user
- HTTP codes: Standard codes (400, 403, 404, 500)

**NFR-P3-4**: Maintainability
- Structure: Follow existing codebase patterns
- Dependencies: Minimize new dependencies
- Configuration: Use existing config system

### P4: Performance (Medium Priority)

**NFR-P4-1**: API response times
- Target: <200ms for membership queries (p95)
- Target: <500ms for membership mutations (p95)
- Measurement: Add performance tests

**NFR-P4-2**: Database optimization
- Indexes: On foreign keys and common query patterns
- Queries: Avoid N+1 queries (use joins)
- Connection pooling: Use existing pool configuration

**NFR-P4-3**: UI responsiveness
- Target: <100ms for UI interactions (button clicks, form submissions)
- Loading states: Show spinners for async operations
- Pagination: Paginate member lists if >100 members

**NFR-P4-4**: Scalability
- Target: Support 50 projects with average 20 members each
- Stress test: Simulate 100 concurrent membership changes
- Monitoring: Add metrics for membership operations

## 8. Database Schema Changes

### New Tables

#### `project_owners` Table
```sql
CREATE TABLE project_owners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_by UUID NOT NULL REFERENCES users(id),
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, user_id)
);

CREATE INDEX idx_project_owners_project ON project_owners(project_id);
CREATE INDEX idx_project_owners_user ON project_owners(user_id);
CREATE INDEX idx_project_owners_assigned_by ON project_owners(assigned_by);
```

**Constraints**:
- Composite unique constraint prevents duplicate owner assignments
- Cascade delete removes owners when project/user deleted
- `assigned_by` tracks who granted ownership (admin or other owner)

#### `project_members` Table
```sql
CREATE TABLE project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('approver', 'developer', 'viewer')),
    added_by UUID NOT NULL REFERENCES users(id),
    added_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, user_id)
);

CREATE INDEX idx_project_members_project ON project_members(project_id);
CREATE INDEX idx_project_members_user ON project_members(user_id);
CREATE INDEX idx_project_members_role ON project_members(role);
CREATE INDEX idx_project_members_added_by ON project_members(added_by);
```

**Constraints**:
- Composite unique constraint: One role per user per project
- Role enum: Only valid roles allowed
- Cascade delete removes members when project/user deleted

#### `audit_logs` Table Extension
```sql
-- Extend existing audit_logs table with new event types
-- New event types to add:
-- - 'project.owner.added'
-- - 'project.owner.removed'
-- - 'project.member.added'
-- - 'project.member.removed'
-- - 'project.member.role_changed'

-- Example audit log entry structure:
-- {
--   "event_type": "project.owner.added",
--   "actor_id": "uuid",
--   "target_user_id": "uuid",
--   "project_id": "uuid",
--   "metadata": {
--     "is_admin_override": false,
--     "previous_role": null,
--     "new_role": "owner"
--   },
--   "timestamp": "2025-11-05T12:00:00Z",
--   "ip_address": "192.168.1.1"
-- }
```

### Migration Strategy

**Phase 1: Schema Creation**
- Run migration to create `project_owners` and `project_members` tables
- Add indexes for performance
- Update audit log event types

**Phase 2: Data Migration (if needed)**
- If existing projects have implicit ownership, migrate to `project_owners`
- Backfill initial owners (e.g., project creator)

**Phase 3: Application Deployment**
- Deploy backend with new API endpoints
- Deploy frontend with new UI pages
- Enable feature flag for gradual rollout

## 9. API Design

### Base URL
```
/api/v1/projects/:projectId
```

### Endpoints

#### 1. Owner Management (Admin Only)

**Add Owner**
```http
POST /api/v1/projects/:projectId/owners
Authorization: Bearer <token>
Content-Type: application/json

{
  "userId": "uuid"
}

Response 201 Created:
{
  "id": "uuid",
  "projectId": "uuid",
  "userId": "uuid",
  "assignedBy": "uuid",
  "assignedAt": "2025-11-05T12:00:00Z"
}

Response 400 Bad Request:
{
  "error": "User is already an owner of this project"
}

Response 403 Forbidden:
{
  "error": "Insufficient permissions. Admin role required."
}
```

**Remove Owner**
```http
DELETE /api/v1/projects/:projectId/owners/:userId
Authorization: Bearer <token>

Response 204 No Content

Response 400 Bad Request:
{
  "error": "Cannot remove last owner. Please assign a new owner first."
}

Response 403 Forbidden:
{
  "error": "Insufficient permissions. Admin role required."
}
```

**List Owners**
```http
GET /api/v1/projects/:projectId/owners
Authorization: Bearer <token>

Response 200 OK:
{
  "owners": [
    {
      "id": "uuid",
      "userId": "uuid",
      "userName": "John Doe",
      "userEmail": "john@example.com",
      "assignedBy": "uuid",
      "assignedAt": "2025-11-05T12:00:00Z"
    }
  ],
  "total": 2
}
```

#### 2. Member Management (Owners & Admins)

**Add Member**
```http
POST /api/v1/projects/:projectId/members
Authorization: Bearer <token>
Content-Type: application/json

{
  "userId": "uuid",
  "role": "developer"
}

Response 201 Created:
{
  "id": "uuid",
  "projectId": "uuid",
  "userId": "uuid",
  "role": "developer",
  "addedBy": "uuid",
  "addedAt": "2025-11-05T12:00:00Z"
}

Response 400 Bad Request:
{
  "error": "Invalid role. Must be one of: approver, developer, viewer"
}

Response 403 Forbidden:
{
  "error": "Insufficient permissions. Owner or admin role required."
}
```

**Remove Member**
```http
DELETE /api/v1/projects/:projectId/members/:userId
Authorization: Bearer <token>

Response 204 No Content

Response 403 Forbidden:
{
  "error": "Insufficient permissions. Owner or admin role required."
}

Response 404 Not Found:
{
  "error": "User is not a member of this project"
}
```

**Update Member Role**
```http
PATCH /api/v1/projects/:projectId/members/:userId
Authorization: Bearer <token>
Content-Type: application/json

{
  "role": "approver"
}

Response 200 OK:
{
  "id": "uuid",
  "projectId": "uuid",
  "userId": "uuid",
  "role": "approver",
  "addedBy": "uuid",
  "addedAt": "2025-11-05T12:00:00Z",
  "updatedAt": "2025-11-05T13:00:00Z"
}

Response 400 Bad Request:
{
  "error": "Cannot change owner role. Use owner management endpoints."
}
```

**List Members**
```http
GET /api/v1/projects/:projectId/members?role=developer&page=1&limit=50
Authorization: Bearer <token>

Response 200 OK:
{
  "members": [
    {
      "id": "uuid",
      "userId": "uuid",
      "userName": "Jane Smith",
      "userEmail": "jane@example.com",
      "role": "developer",
      "addedBy": "uuid",
      "addedAt": "2025-11-05T12:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 15
  }
}
```

#### 3. User Project Queries

**List User's Projects**
```http
GET /api/v1/users/me/projects
Authorization: Bearer <token>

Response 200 OK:
{
  "projects": [
    {
      "projectId": "uuid",
      "projectName": "Production DB",
      "role": "owner",
      "joinedAt": "2025-11-05T12:00:00Z"
    },
    {
      "projectId": "uuid",
      "projectName": "Staging DB",
      "role": "developer",
      "joinedAt": "2025-11-01T10:00:00Z"
    }
  ],
  "total": 2
}
```

#### 4. Audit History

**Get Project Audit History**
```http
GET /api/v1/projects/:projectId/audit?eventType=project.member.added&page=1&limit=50
Authorization: Bearer <token>

Response 200 OK:
{
  "events": [
    {
      "id": "uuid",
      "eventType": "project.member.added",
      "actorId": "uuid",
      "actorName": "Admin User",
      "targetUserId": "uuid",
      "targetUserName": "New Developer",
      "metadata": {
        "role": "developer",
        "isAdminOverride": false
      },
      "timestamp": "2025-11-05T12:00:00Z",
      "ipAddress": "192.168.1.1"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 120
  }
}
```

### Error Response Format

All error responses follow this structure:
```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "specific field that caused error"
  }
}
```

### Rate Limiting

- **Rate limit**: 100 requests per minute per user
- **Response header**: `X-RateLimit-Remaining: 95`
- **Response on limit**: `429 Too Many Requests`

## 10. UI/UX Requirements

### Project Management Page

**Location**: `/projects/:projectId/members`

**Page Layout**:
```
┌─────────────────────────────────────────────────────────┐
│  Project: Production Database                           │
│  [Overview] [Changes] [Members] [Settings]              │
└─────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────┐
│  Project Owners (2)                     [+ Add Owner]    │
│  ┌───────────────────────────────────────────────────┐ │
│  │ 👤 John Doe (john@example.com)                    │ │
│  │    Assigned by: Admin User                        │ │
│  │    Assigned on: Nov 1, 2025                 [Remove]│ │
│  └───────────────────────────────────────────────────┘ │
│  ┌───────────────────────────────────────────────────┐ │
│  │ 👤 Jane Smith (jane@example.com)                  │ │
│  │    Assigned by: John Doe                          │ │
│  │    Assigned on: Nov 3, 2025                 [Remove]│ │
│  └───────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────┐
│  Project Members (15)                   [+ Add Member]   │
│  Filters: [All Roles ▼] [Search...]                     │
│  ┌───────────────────────────────────────────────────┐ │
│  │ 👤 Alice Johnson (alice@example.com)              │ │
│  │    Role: Approver        Added: Nov 1, 2025       │ │
│  │    [Change Role ▼] [Remove]                       │ │
│  └───────────────────────────────────────────────────┘ │
│  ┌───────────────────────────────────────────────────┐ │
│  │ 👤 Bob Wilson (bob@example.com)                   │ │
│  │    Role: Developer       Added: Nov 2, 2025       │ │
│  │    [Change Role ▼] [Remove]                       │ │
│  └───────────────────────────────────────────────────┘ │
│  [Show More...]                                         │
└─────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────┐
│  Audit History                          [Export CSV]     │
│  ┌───────────────────────────────────────────────────┐ │
│  │ Nov 5, 12:00 PM - Admin User added Bob Wilson    │ │
│  │                   as Developer                     │ │
│  └───────────────────────────────────────────────────┘ │
│  ┌───────────────────────────────────────────────────┐ │
│  │ Nov 3, 3:00 PM  - John Doe assigned Jane Smith   │ │
│  │                   as Owner                         │ │
│  └───────────────────────────────────────────────────┘ │
│  [Load More History...]                                 │
└─────────────────────────────────────────────────────────┘
```

### Add Owner Modal (Admin Only)

```
┌─────────────────────────────────────────┐
│  Add Project Owner                   [X]│
├─────────────────────────────────────────┤
│  Select user to add as owner:          │
│  ┌──────────────────────────────────┐  │
│  │ [Search users...]               🔍│  │
│  └──────────────────────────────────┘  │
│                                         │
│  Suggestions:                           │
│  ☐ John Doe (john@example.com)         │
│  ☐ Jane Smith (jane@example.com)       │
│  ☐ Bob Wilson (bob@example.com)        │
│                                         │
│         [Cancel]  [Add Owner]           │
└─────────────────────────────────────────┘
```

### Add Member Modal (Owners & Admins)

```
┌─────────────────────────────────────────┐
│  Add Project Member                  [X]│
├─────────────────────────────────────────┤
│  Select user:                           │
│  ┌──────────────────────────────────┐  │
│  │ [Search users...]               🔍│  │
│  └──────────────────────────────────┘  │
│                                         │
│  Assign role:                           │
│  ◯ Approver - Can approve changes       │
│  ◉ Developer - Can create change requests│
│  ◯ Viewer - Read-only access            │
│                                         │
│  Note: Default role is Developer        │
│                                         │
│         [Cancel]  [Add Member]          │
└─────────────────────────────────────────┘
```

### Confirmation Modal (Remove Last Owner)

```
┌─────────────────────────────────────────┐
│  ⚠️  Cannot Remove Last Owner        [X]│
├─────────────────────────────────────────┤
│  You are attempting to remove the last  │
│  owner from this project.               │
│                                         │
│  Projects must have at least one owner. │
│  Please assign a new owner before       │
│  removing this one.                     │
│                                         │
│            [Assign New Owner]           │
│                 [Cancel]                │
└─────────────────────────────────────────┘
```

### Permission States

**Admin View**:
- Can add/remove owners
- Can add/remove members
- Can change member roles
- "Admin Override" badge visible when not an owner

**Owner View**:
- Cannot add/remove owners (disabled)
- Can add/remove members
- Can change member roles
- Cannot remove self if last owner (disabled with tooltip)

**Non-Owner View**:
- Cannot add/remove owners (not visible)
- Cannot add/remove members (not visible)
- Can view member list (read-only)
- Can view audit history (read-only)

### Responsive Design
- **Desktop**: Three-column layout (owners | members | audit)
- **Tablet**: Two-column layout (owners/members | audit)
- **Mobile**: Single column, collapsible sections

### Accessibility
- Keyboard navigation for all interactive elements
- Screen reader labels for all buttons and inputs
- ARIA attributes for dynamic content
- Focus management for modals

## 11. Security Considerations

### Authorization Model

**Permission Hierarchy**:
```
System Admin
  └─ Can manage all projects
     └─ Can override any owner decision
        └─ Can view all audit logs

Project Owner
  └─ Can manage members of owned projects
     └─ Can view audit logs for owned projects
        └─ Cannot manage owners (admin only)

Project Member (Approver/Developer/Viewer)
  └─ Role-specific permissions within project
     └─ Can view own membership status
        └─ No member management capabilities
```

### Permission Checks

**Every API call must validate**:
1. **Authentication**: Valid JWT token
2. **Organization membership**: User belongs to organization
3. **Project access**: User has project-level permission
4. **Action authorization**: User role allows the action

**Example permission check logic**:
```go
func CanManageMembers(ctx context.Context, actorID, projectID uuid.UUID) bool {
    // Check if admin (org-level role)
    if IsAdmin(ctx, actorID) {
        return true
    }

    // Check if project owner
    isOwner, err := IsProjectOwner(ctx, actorID, projectID)
    if err != nil {
        return false
    }

    return isOwner
}
```

### Attack Vectors & Mitigations

**AV1: Privilege Escalation**
- **Attack**: Non-owner attempts to add themselves as owner
- **Mitigation**: Owner management restricted to admins only
- **Validation**: Check `IsAdmin()` before processing owner changes

**AV2: Last Owner Removal**
- **Attack**: Malicious actor removes all owners to lock project
- **Mitigation**: Prevent last owner removal without reassignment
- **Validation**: Count owners before allowing removal

**AV3: Mass Member Addition**
- **Attack**: Bulk add all organization users as members
- **Mitigation**: Rate limiting (100 requests/min)
- **Validation**: Require explicit approval for bulk operations (future)

**AV4: Audit Log Tampering**
- **Attack**: Modify audit logs to hide malicious activity
- **Mitigation**: Append-only audit logs with database constraints
- **Validation**: No UPDATE or DELETE permissions on audit_logs table

**AV5: Session Hijacking**
- **Attack**: Steal admin token to gain project control
- **Mitigation**: Short-lived tokens (1 hour), refresh token rotation
- **Validation**: IP address tracking in audit logs

### Data Protection

**Sensitive Data**:
- User emails (PII) - Only show to authorized users
- IP addresses in audit logs - Restricted to admins
- Audit history - Project-scoped access control

**Encryption**:
- All API traffic over HTTPS (TLS 1.3)
- Database connections encrypted (SSL)
- Passwords never stored (OAuth only)

### Compliance

**GDPR Considerations**:
- User data in `project_owners` and `project_members` tables
- Right to be forgotten: CASCADE DELETE handles user removal
- Audit logs contain user actions (lawful basis: legitimate interest)

**SOC2 Controls**:
- Access control enforced at API and database layers
- Audit trail for all membership changes
- Regular security reviews of permission logic

## 12. Milestones and Phasing

This section will be populated by the `/wiz:phases` command after PRD approval.

**Placeholder Phase Structure**:
- **Phase 1**: Database schema and migrations
- **Phase 2**: Backend API and services
- **Phase 3**: Frontend UI components
- **Phase 4**: Integration, testing, and documentation
- **Phase 5**: Deployment and rollout

Detailed milestones (~1 hour tasks) will be defined during phase planning.

## 13. Open Questions

### Q1: Default Member Role
**Question**: What should be the default role assigned to new members if not explicitly specified?

**Options**:
1. **Developer** (most permissive non-owner role)
2. **Viewer** (most restrictive, safest default)
3. **Require explicit selection** (no default)

**Decision needed by**: Phase 2 (Backend API implementation)

**Recommendation**: Require explicit selection to avoid accidental over-permissioning

### Q2: Bulk Operations
**Question**: Should we support bulk member addition/removal (e.g., add 10 users at once)?

**Options**:
1. Add bulk endpoint (`POST /projects/:id/members/bulk`)
2. Defer to future enhancement
3. Allow via multiple single-add API calls (current approach)

**Decision needed by**: Phase 2 (Backend API implementation)

**Recommendation**: Defer to future enhancement unless user feedback indicates need

### Q3: Member Invitation Workflow
**Question**: Should members be automatically added, or should they accept an invitation?

**Current approach**: Immediate addition (no invitation workflow)

**Future consideration**: Email invitation flow where users must accept membership

**Decision needed by**: Phase 3 (Frontend UI implementation)

**Recommendation**: Start with immediate addition, add invitation workflow in v2.0

### Q4: Owner Self-Removal
**Question**: Should owners be allowed to remove themselves if not the last owner?

**Current spec**: Owners can remove themselves if at least one other owner exists

**Risk**: Accidental self-removal locks owner out of management

**Mitigation**: Add confirmation modal "Are you sure you want to remove yourself as owner?"

**Decision needed by**: Phase 3 (Frontend UI implementation)

## 14. Appendix

### References

**RBAC Design Patterns**:
- [AWS IAM Best Practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html)
- [NIST RBAC Model](https://csrc.nist.gov/projects/role-based-access-control)
- [OWASP Access Control Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Access_Control_Cheat_Sheet.html)

**Audit Logging Standards**:
- [ISO 27001 Audit Requirements](https://www.iso.org/standard/27001)
- [GDPR Article 30: Records of Processing](https://gdpr-info.eu/art-30-gdpr/)

**Database Design**:
- [Postgres Best Practices for Multi-Tenancy](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [Database Index Design](https://use-the-index-luke.com/)

### Existing System Context

**Related Tables**:
- `users`: Contains user profiles, authentication, and organization membership
- `projects`: Existing project table (name, description, database connection details)
- `audit_logs`: Existing audit system for tracking all system events

**Related Permissions** (to extend):
- Organization-level: `admin`, `user`
- Project-level (new): `owner`, `approver`, `developer`, `viewer`

**Authentication Flow**:
1. User logs in via OAuth (Google/GitHub)
2. JWT token issued with organization role
3. API calls include JWT in Authorization header
4. Backend validates token and checks project-level permissions

### Glossary

- **Project Owner**: User with full management control over a specific project
- **Project Member**: User with role-based access to a specific project
- **Admin Override**: Action taken by system admin outside normal ownership rules
- **Orphaned Project**: Project with no owners (prevented by system constraints)
- **Project Role**: Permission level within a specific project (owner, approver, developer, viewer)
- **Organization Role**: System-wide permission level (admin, user)

### Version History

- **v1.0.0** (2025-11-05): Initial PRD created from user intake answers
