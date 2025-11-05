/**
 * RBAC (Role-Based Access Control) utilities for frontend
 * Provides role checking and permission checking functions based on backend RBAC system
 */

// Organization roles (matching backend)
export type OrganizationRole = 'OrganizationAdmin' | 'OrganizationDBA' | 'OrganizationDeveloper';

// Project roles (matching backend)
export type ProjectRole = 'ProjectOwner' | 'ProjectDeveloper' | 'ProjectApprover' | 'ProjectViewer';

// Organization-level permissions
export enum OrganizationPermission {
  // User management
  UsersCreate = 'users.create',
  UsersRead = 'users.read',
  UsersUpdate = 'users.update',
  UsersDelete = 'users.delete',

  // Organization management
  OrganizationUpdate = 'organization.update',
  OrganizationDelete = 'organization.delete',

  // Database management
  DatabasesCreate = 'databases.create',
  DatabasesRead = 'databases.read',
  DatabasesUpdate = 'databases.update',
  DatabasesDelete = 'databases.delete',

  // Project management
  ProjectsCreate = 'projects.create',
  ProjectsRead = 'projects.read',
  ProjectsUpdate = 'projects.update',
  ProjectsDelete = 'projects.delete',

  // Audit logs
  AuditLogsRead = 'audit_logs.read',

  // Wildcard permission (admin only)
  All = '*',
}

// Project-level permissions
export enum ProjectPermission {
  // Project settings
  ProjectSettingsUpdate = 'project.settings.update',

  // Project members
  ProjectMembersCreate = 'project.members.create',
  ProjectMembersRead = 'project.members.read',
  ProjectMembersUpdate = 'project.members.update',
  ProjectMembersDelete = 'project.members.delete',

  // Change requests
  ChangesCreate = 'changes.create',
  ChangesRead = 'changes.read',
  ChangesUpdate = 'changes.update',
  ChangesDelete = 'changes.delete',
  ChangesApprove = 'changes.approve',
  ChangesReject = 'changes.reject',
  ChangesExecute = 'changes.execute',

  // Policy rules
  PolicyRulesCreate = 'policy_rules.create',
  PolicyRulesRead = 'policy_rules.read',
  PolicyRulesUpdate = 'policy_rules.update',
  PolicyRulesDelete = 'policy_rules.delete',
}

// Organization role permissions mapping (matching backend)
const organizationRolePermissions: Record<OrganizationRole, OrganizationPermission[]> = {
  OrganizationAdmin: [OrganizationPermission.All], // Admin has all permissions
  OrganizationDBA: [
    OrganizationPermission.UsersRead,
    OrganizationPermission.DatabasesCreate,
    OrganizationPermission.DatabasesRead,
    OrganizationPermission.DatabasesUpdate,
    OrganizationPermission.DatabasesDelete,
    OrganizationPermission.ProjectsRead,
    OrganizationPermission.AuditLogsRead,
  ],
  OrganizationDeveloper: [
    OrganizationPermission.UsersRead,
    OrganizationPermission.DatabasesRead,
    OrganizationPermission.ProjectsCreate,
    OrganizationPermission.ProjectsRead,
    OrganizationPermission.AuditLogsRead,
  ],
};

// Project role permissions mapping (matching backend)
const projectRolePermissions: Record<ProjectRole, ProjectPermission[]> = {
  ProjectOwner: [
    ProjectPermission.ProjectSettingsUpdate,
    ProjectPermission.ProjectMembersCreate,
    ProjectPermission.ProjectMembersRead,
    ProjectPermission.ProjectMembersUpdate,
    ProjectPermission.ProjectMembersDelete,
    ProjectPermission.ChangesCreate,
    ProjectPermission.ChangesRead,
    ProjectPermission.ChangesUpdate,
    ProjectPermission.ChangesDelete,
    ProjectPermission.ChangesApprove,
    ProjectPermission.ChangesReject,
    ProjectPermission.ChangesExecute,
    ProjectPermission.PolicyRulesCreate,
    ProjectPermission.PolicyRulesRead,
    ProjectPermission.PolicyRulesUpdate,
    ProjectPermission.PolicyRulesDelete,
  ],
  ProjectDeveloper: [
    ProjectPermission.ProjectMembersRead,
    ProjectPermission.ChangesCreate,
    ProjectPermission.ChangesRead,
    ProjectPermission.ChangesUpdate,
    ProjectPermission.ChangesDelete,
    ProjectPermission.PolicyRulesRead,
  ],
  ProjectApprover: [
    ProjectPermission.ProjectMembersRead,
    ProjectPermission.ChangesRead,
    ProjectPermission.ChangesApprove,
    ProjectPermission.ChangesReject,
    ProjectPermission.PolicyRulesRead,
  ],
  ProjectViewer: [
    ProjectPermission.ProjectMembersRead,
    ProjectPermission.ChangesRead,
    ProjectPermission.PolicyRulesRead,
  ],
};

/**
 * Check if a user has a specific organization-level permission
 */
export function hasOrganizationPermission(
  role: OrganizationRole | string | undefined,
  permission: OrganizationPermission,
): boolean {
  if (!role) {
    return false;
  }

  const orgRole = role as OrganizationRole;
  if (!isValidOrganizationRole(orgRole)) {
    return false;
  }

  const permissions = organizationRolePermissions[orgRole] || [];
  return permissions.includes(OrganizationPermission.All) || permissions.includes(permission);
}

/**
 * Check if a user has a specific project-level permission
 */
export function hasProjectPermission(
  role: ProjectRole | string | undefined,
  permission: ProjectPermission,
): boolean {
  if (!role) {
    return false;
  }

  const projRole = role as ProjectRole;
  if (!isValidProjectRole(projRole)) {
    return false;
  }

  const permissions = projectRolePermissions[projRole] || [];
  return permissions.includes(permission);
}

/**
 * Check if a user has a specific organization role
 */
export function hasOrganizationRole(
  userRole: string | undefined,
  requiredRole: OrganizationRole,
): boolean {
  if (!userRole) {
    return false;
  }
  return userRole === requiredRole;
}

/**
 * Check if a user has one of the required organization roles
 */
export function hasAnyOrganizationRole(
  userRole: string | undefined,
  requiredRoles: OrganizationRole[],
): boolean {
  if (!userRole) {
    return false;
  }
  return requiredRoles.some((role) => userRole === role);
}

/**
 * Check if a user has a specific project role
 */
export function hasProjectRole(userRole: string | undefined, requiredRole: ProjectRole): boolean {
  if (!userRole) {
    return false;
  }
  return userRole === requiredRole;
}

/**
 * Check if a user has one of the required project roles
 */
export function hasAnyProjectRole(
  userRole: string | undefined,
  requiredRoles: ProjectRole[],
): boolean {
  if (!userRole) {
    return false;
  }
  return requiredRoles.some((role) => userRole === role);
}

/**
 * Check if a role is a valid organization role
 */
export function isValidOrganizationRole(role: string): role is OrganizationRole {
  return ['OrganizationAdmin', 'OrganizationDBA', 'OrganizationDeveloper'].includes(role);
}

/**
 * Check if a role is a valid project role
 */
export function isValidProjectRole(role: string): role is ProjectRole {
  return ['ProjectOwner', 'ProjectDeveloper', 'ProjectApprover', 'ProjectViewer'].includes(role);
}

/**
 * Check if user is admin
 */
export function isAdmin(userRole: string | undefined): boolean {
  return hasOrganizationRole(userRole, 'OrganizationAdmin');
}

/**
 * Check if user is DBA
 */
export function isDBA(userRole: string | undefined): boolean {
  return hasOrganizationRole(userRole, 'OrganizationDBA');
}

/**
 * Check if user is developer
 */
export function isDeveloper(userRole: string | undefined): boolean {
  return hasOrganizationRole(userRole, 'OrganizationDeveloper');
}

/**
 * Get all organization permissions for a role
 */
export function getOrganizationPermissions(role: OrganizationRole): OrganizationPermission[] {
  return organizationRolePermissions[role] || [];
}

/**
 * Get all project permissions for a role
 */
export function getProjectPermissions(role: ProjectRole): ProjectPermission[] {
  return projectRolePermissions[role] || [];
}
