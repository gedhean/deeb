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
 * It uses a stable key based on all project IDs to avoid hooks violations.
 * Per-project errors are surfaced individually so callers can detect and handle individual failures.
 */
export function useMultipleProjectMembersSWR(
  projectIds: string[],
): Record<string, UseProjectMembersResult> {
  // Create a stable key for all projects
  const projectIdsKey = projectIds.sort().join(',');

  // Fetch all project members in parallel using a single SWR hook
  const {
    data: allMembersData,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<Record<string, { members: ProjectMemberWithUser[]; error?: unknown }>>(
    projectIds.length > 0 ? `/api/projects/members?projects=${projectIdsKey}` : null,
    async () => {
      // Fetch members for all projects in parallel
      const fetchPromises = projectIds.map(async (projectId) => {
        try {
          const members = await fetchProjectMembers(projectId);
          return { projectId, members, error: undefined };
        } catch (err) {
          return { projectId, members: [], error: err };
        }
      });

      const results = await Promise.all(fetchPromises);

      // Convert to Record format, preserving both members and errors
      const membersMap: Record<string, { members: ProjectMemberWithUser[]; error?: unknown }> = {};
      for (const result of results) {
        membersMap[result.projectId] = {
          members: result.members,
          error: result.error,
        };
      }

      return membersMap;
    },
    {
      revalidateIfStale: true,
      keepPreviousData: true,
    },
  );

  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  const globalError = swrError
    ? swrError instanceof Error
      ? swrError.message
      : 'Failed to load project members'
    : null;

  // Convert to the expected return format
  const results: Record<string, UseProjectMembersResult> = {};

  for (const projectId of projectIds) {
    const projectData = allMembersData?.[projectId];
    const projectError = projectData?.error
      ? projectData.error instanceof Error
        ? projectData.error.message
        : 'Failed to load members for this project'
      : null;

    results[projectId] = {
      members: projectData?.members || [],
      isLoading,
      error: projectError || globalError,
      refetch,
    };
  }

  return results;
}
