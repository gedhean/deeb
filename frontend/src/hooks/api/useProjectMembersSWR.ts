import useSWR from 'swr';
import { useCallback } from 'react';
import { projectMembersService } from '@/lib/api/services/projects';
import type { ProjectMemberWithUser } from '@/lib/api/types';

/**
 * SWR fetcher function for project members
 */
const fetchProjectMembers = async (projectId: string) => {
  return await projectMembersService.listProjectMembers(projectId);
};

interface UseProjectMembersResult {
  members: ProjectMemberWithUser[];
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching project members with SWR caching
 *
 * This hook provides automatic request deduplication and caching for project members.
 * Multiple components can call this hook with the same projectId and only one API call will be made.
 */
export function useProjectMembersSWR(projectId: string | undefined): UseProjectMembersResult {
  const shouldFetch = !!projectId;

  const {
    data: members,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<ProjectMemberWithUser[]>(
    shouldFetch ? `/api/projects/${projectId}/members` : null,
    shouldFetch && projectId ? () => fetchProjectMembers(projectId) : null,
    {
      revalidateIfStale: true,
      keepPreviousData: true,
    },
  );

  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  const error = swrError
    ? swrError instanceof Error
      ? swrError.message
      : 'Failed to load project members'
    : null;

  return {
    members: members || [],
    isLoading,
    error,
    refetch,
  };
}

/**
 * Hook for fetching members for multiple projects with automatic deduplication
 *
 * This hook fetches members for all provided project IDs and returns them grouped by project.
 * It leverages SWR's automatic request deduplication to avoid duplicate calls.
 */
export function useMultipleProjectMembersSWR(
  projectIds: string[],
): Record<string, UseProjectMembersResult> {
  const results: Record<string, UseProjectMembersResult> = {};

  // Use individual hooks for each project - SWR will deduplicate automatically
  for (const projectId of projectIds) {
    // eslint-disable-next-line react-hooks/rules-of-hooks
    results[projectId] = useProjectMembersSWR(projectId);
  }

  return results;
}
