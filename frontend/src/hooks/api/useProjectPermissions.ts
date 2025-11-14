import { useMemo } from 'react';
import { useAuth } from '@/contexts/AuthContext';
import { isAdmin, hasProjectPermission, ProjectPermission } from '@/lib/rbac';
import type { ProjectMemberWithUser, ProjectRole } from '@/lib/api/types';

export interface ProjectPermissionsResult {
  // Member management
  canManageMembers: boolean;

  // Settings management
  canManageSettings: boolean;

  // Change operations
  canCreateChanges: boolean;
  canUpdateChanges: boolean;
  canDeleteChanges: boolean;
  canApproveChanges: boolean;
  canRejectChanges: boolean;
  canExecuteChanges: boolean;

  // Policy rules management
  canManagePolicyRules: boolean;

  // Role information
  isProjectOwner: boolean;
  userProjectRole: ProjectRole | null;
}

/**
 * Hook to check project-level permissions for the current user
 * @param projectMembers - List of project members to determine current user's project role
 * @returns Object containing permission flags and role information
 */
export function useProjectPermissions(
  projectMembers?: ProjectMemberWithUser[],
): ProjectPermissionsResult {
  const { user } = useAuth();

  return useMemo(() => {
    const userIsAdmin = isAdmin(user?.organization_role);

    // Find current user's project membership and role
    const userProjectMembership = projectMembers?.find((pm) => pm.member.user_id === user?.id);
    const userProjectRole = (userProjectMembership?.member.role as ProjectRole) || null;

    // Check if current user is a project owner
    const userIsProjectOwner = userProjectRole === 'ProjectOwner';

    // Admin users bypass all project-level permission checks
    // For non-admin users, check their project role permissions
    const canManageMembers =
      userIsAdmin || hasProjectPermission(userProjectRole, ProjectPermission.ProjectMembersCreate);

    const canManageSettings =
      userIsAdmin || hasProjectPermission(userProjectRole, ProjectPermission.ProjectSettingsUpdate);

    const canCreateChanges =
      userIsAdmin || hasProjectPermission(userProjectRole, ProjectPermission.ChangesCreate);

    const canUpdateChanges =
      userIsAdmin || hasProjectPermission(userProjectRole, ProjectPermission.ChangesUpdate);

    const canDeleteChanges =
      userIsAdmin || hasProjectPermission(userProjectRole, ProjectPermission.ChangesDelete);

    const canApproveChanges =
      userIsAdmin || hasProjectPermission(userProjectRole, ProjectPermission.ChangesApprove);

    const canRejectChanges =
      userIsAdmin || hasProjectPermission(userProjectRole, ProjectPermission.ChangesReject);

    const canExecuteChanges =
      userIsAdmin || hasProjectPermission(userProjectRole, ProjectPermission.ChangesExecute);

    const canManagePolicyRules =
      userIsAdmin || hasProjectPermission(userProjectRole, ProjectPermission.PolicyRulesCreate);

    return {
      canManageMembers,
      canManageSettings,
      canCreateChanges,
      canUpdateChanges,
      canDeleteChanges,
      canApproveChanges,
      canRejectChanges,
      canExecuteChanges,
      canManagePolicyRules,
      isProjectOwner: userIsProjectOwner,
      userProjectRole,
    };
  }, [user, projectMembers]);
}
