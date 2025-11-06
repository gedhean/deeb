import { useMemo } from 'react';
import { useAuth } from '@/contexts/AuthContext';
import { isAdmin } from '@/lib/rbac';
import type { ProjectMemberWithUser } from '@/lib/api/types';

/**
 * Hook to check if the current user can manage project members
 * @param projectMembers - List of project members to check if current user is a project owner
 * @returns boolean indicating if user can manage project members
 */
export function useProjectPermissions(projectMembers?: ProjectMemberWithUser[]): {
  canManageMembers: boolean;
  isProjectOwner: boolean;
} {
  const { user } = useAuth();

  return useMemo(() => {
    const userIsAdmin = isAdmin(user?.organization_role);

    // Check if current user is a project owner
    const userIsProjectOwner =
      projectMembers?.some(
        (pm) => pm.member.user_id === user?.id && pm.member.role === 'ProjectOwner',
      ) ?? false;

    return {
      canManageMembers: userIsAdmin || userIsProjectOwner,
      isProjectOwner: userIsProjectOwner,
    };
  }, [user, projectMembers]);
}
