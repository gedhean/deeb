/**
 * RBAC (Role-Based Access Control) utilities for frontend
 * Provides role checking and permission checking functions based on backend RBAC system
 *
 * IMPORTANT: Permission constants and role mappings are generated from YAML configuration.
 * See rbac.generated.ts (auto-generated from backend/config/iam/*.yaml)
 */

// Import and re-export from generated file
import {
  OrganizationPermission as OrgPermission,
  organizationRolePermissions,
  projectRolePermissions,
} from './rbac.generated';

// Re-export types (OrganizationRole and ProjectRole are pure types)
export type { OrganizationRole, ProjectRole } from './rbac.generated';

// Re-export constants, mappings, and permission types (these are values with associated types)
// Note: OrganizationPermission and ProjectPermission are exported as values (const objects)
// Their types are inferred from the const assertion in the generated file
export {
  OrganizationPermission,
  ProjectPermission,
  organizationRolePermissions,
  projectRolePermissions,
} from './rbac.generated';

/**
 * Check if a user has a specific organization-level permission
 */
export function hasOrganizationPermission(role: string | undefined, permission: string): boolean {
  if (!role) {
    return false;
  }

  if (!isValidOrganizationRole(role)) {
    return false;
  }

  const permissions =
    organizationRolePermissions[role as keyof typeof organizationRolePermissions] || [];
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return permissions.includes(OrgPermission.All) || permissions.includes(permission as any);
}

/**
 * Check if a user has a specific project-level permission
 */
export function hasProjectPermission(role: string | undefined, permission: string): boolean {
  if (!role) {
    return false;
  }

  if (!isValidProjectRole(role)) {
    return false;
  }

  const permissions = projectRolePermissions[role as keyof typeof projectRolePermissions] || [];
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return permissions.includes(permission as any);
}

/**
 * Check if a user has a specific organization role
 */
export function hasOrganizationRole(userRole: string | undefined, requiredRole: string): boolean {
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
  requiredRoles: string[],
): boolean {
  if (!userRole) {
    return false;
  }
  return requiredRoles.some((role) => userRole === role);
}

/**
 * Check if a user has a specific project role
 */
export function hasProjectRole(userRole: string | undefined, requiredRole: string): boolean {
  if (!userRole) {
    return false;
  }
  return userRole === requiredRole;
}

/**
 * Check if a user has one of the required project roles
 */
export function hasAnyProjectRole(userRole: string | undefined, requiredRoles: string[]): boolean {
  if (!userRole) {
    return false;
  }
  return requiredRoles.some((role) => userRole === role);
}

/**
 * Check if a role is a valid organization role
 */
export function isValidOrganizationRole(role: string): boolean {
  return ['OrganizationAdmin', 'OrganizationDBA', 'OrganizationDeveloper'].includes(role);
}

/**
 * Check if a role is a valid project role
 */
export function isValidProjectRole(role: string): boolean {
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
export function getOrganizationPermissions(role: string): string[] {
  return organizationRolePermissions[role as keyof typeof organizationRolePermissions] || [];
}

/**
 * Get all project permissions for a role
 */
export function getProjectPermissions(role: string): string[] {
  return projectRolePermissions[role as keyof typeof projectRolePermissions] || [];
}
