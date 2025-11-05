import React from 'react';
import { useAuth } from '@/contexts/AuthContext';
import type { OrganizationRole, ProjectRole } from '@/lib/rbac';
import {
  hasOrganizationPermission,
  hasProjectPermission,
  hasAnyOrganizationRole,
  hasAnyProjectRole,
  OrganizationPermission,
  ProjectPermission,
} from '@/lib/rbac';

interface ProtectedComponentProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
  requireOrganizationRole?: OrganizationRole | OrganizationRole[];
  requireOrganizationPermission?: OrganizationPermission;
  requireProjectRole?: ProjectRole | ProjectRole[];
  requireProjectPermission?: ProjectPermission;
  projectRole?: ProjectRole | string;
}

/**
 * ProtectedComponent wrapper that conditionally renders children based on RBAC checks
 */
export const ProtectedComponent: React.FC<ProtectedComponentProps> = ({
  children,
  fallback = null,
  requireOrganizationRole,
  requireOrganizationPermission,
  requireProjectRole,
  requireProjectPermission,
  projectRole,
}) => {
  const { user } = useAuth();

  if (!user) {
    return <>{fallback}</>;
  }

  // Check organization role
  if (requireOrganizationRole) {
    const roles = Array.isArray(requireOrganizationRole)
      ? requireOrganizationRole
      : [requireOrganizationRole];
    if (!hasAnyOrganizationRole(user.organization_role, roles)) {
      return <>{fallback}</>;
    }
  }

  // Check organization permission
  if (requireOrganizationPermission) {
    if (!hasOrganizationPermission(user.organization_role, requireOrganizationPermission)) {
      return <>{fallback}</>;
    }
  }

  // Check project role
  if (requireProjectRole && projectRole) {
    const roles = Array.isArray(requireProjectRole) ? requireProjectRole : [requireProjectRole];
    if (!hasAnyProjectRole(projectRole, roles)) {
      return <>{fallback}</>;
    }
  }

  // Check project permission
  if (requireProjectPermission && projectRole) {
    if (!hasProjectPermission(projectRole, requireProjectPermission)) {
      return <>{fallback}</>;
    }
  }

  return <>{children}</>;
};
