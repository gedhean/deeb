import useSWR, { mutate } from 'swr';
import { useCallback, useState } from 'react';
import { projectsService } from '@/lib/api/services/projects';
import type { Project, CreateProjectRequest, UpdateProjectRequest } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import {
  useProjects as useProjectsStore,
  useProjectsLoading,
  useProjectsError,
  useGetProjectById as useGetProjectByIdStore,
  useAddProject as useAddProjectStore,
  useUpdateProject as useUpdateProjectStore,
  useRemoveProject as useRemoveProjectStore,
} from '@/store';

/**
 * Parameters for projects list fetching
 */
export interface UseProjectsParams {
  limit?: number;
  offset?: number;
}

/**
 * SWR fetcher function for projects list
 */
const fetchProjects = async (params?: UseProjectsParams) => {
  const response = await projectsService.listProjects(params || { limit: 100 });
  return { items: response.items, total: response.total };
};

/**
 * SWR fetcher function for single project
 */
const fetchProject = async (id: string) => {
  return await projectsService.getProject(id);
};

interface UseProjectsResult {
  projects: Project[];
  total: number;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
  getProjectById: (projectId: string) => Project | undefined;
}

/**
 * Hook for fetching projects list with SWR caching and Zustand store integration
 *
 * This hook combines SWR for automatic request deduplication and background revalidation
 * with Zustand store for immediate cache availability across components.
 */
export function useProjectsSWR(params?: UseProjectsParams): UseProjectsResult {
  const storeProjects = useProjectsStore();
  const storeLoading = useProjectsLoading();
  const storeError = useProjectsError();
  const getProjectByIdFromStore = useGetProjectByIdStore();

  // Create cache key that includes pagination params
  const cacheKey = params ? ['/api/projects', params] : '/api/projects';

  // SWR hook for automatic caching and revalidation
  const {
    data: swrData,
    error: swrError,
    isLoading: swrLoading,
    mutate,
  } = useSWR<{ items: Project[]; total: number }>(cacheKey, () => fetchProjects(params), {
    fallbackData:
      storeProjects.length > 0 ? { items: storeProjects, total: storeProjects.length } : undefined,
    revalidateIfStale: true,
    keepPreviousData: true,
  });

  // Manual refetch function
  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  // Helper to get project by ID
  const getProjectById = useCallback(
    (projectId: string) => {
      const projects = swrData?.items || storeProjects;
      return (
        projects.find((project) => project.id === projectId) || getProjectByIdFromStore(projectId)
      );
    },
    [swrData, storeProjects, getProjectByIdFromStore],
  );

  const projects = swrData?.items || storeProjects;
  const total = swrData?.total || storeProjects.length;
  const isLoading = swrLoading || storeLoading;
  const error = swrError
    ? swrError instanceof Error
      ? swrError.message
      : 'Failed to load projects'
    : storeError;

  return {
    projects,
    total,
    isLoading,
    error,
    refetch,
    getProjectById,
  };
}

interface UseProjectOptions {
  enabled?: boolean;
}

interface UseProjectResult {
  project: Project | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching a single project by ID with SWR
 */
export function useProjectSWR(
  id: string | undefined,
  options: UseProjectOptions = {},
): UseProjectResult {
  const { enabled = true } = options;

  const shouldFetch = enabled && id;
  const {
    data: project,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<Project>(
    shouldFetch ? `/api/projects/${id}` : null,
    shouldFetch && id ? () => fetchProject(id) : null,
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
      : 'Failed to load project'
    : null;

  return {
    project: project || null,
    isLoading,
    error,
    refetch,
  };
}

interface UseCreateProjectResult {
  createProject: (data: CreateProjectRequest) => Promise<Project | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for creating a project with automatic cache update
 */
export function useCreateProjectSWR(): UseCreateProjectResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const addProjectToStore = useAddProjectStore();

  const createProject = useCallback(
    async (data: CreateProjectRequest): Promise<Project | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await projectsService.createProject(data);

        // Update cache immediately
        addProjectToStore(result);

        // Invalidate projects cache using global mutate
        await mutate('/api/projects', undefined, { revalidate: true });

        return result;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Failed to create project';
        setError(errorMsg);
        console.error('Failed to create project:', err);
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [addProjectToStore],
  );

  return {
    createProject,
    isLoading,
    error,
  };
}

interface UseUpdateProjectResult {
  updateProject: (id: string, data: UpdateProjectRequest) => Promise<Project | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for updating a project with automatic cache update
 */
export function useUpdateProjectSWR(): UseUpdateProjectResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const updateProjectInStore = useUpdateProjectStore();

  const updateProject = useCallback(
    async (id: string, data: UpdateProjectRequest): Promise<Project | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await projectsService.updateProject(id, data);

        // Update cache immediately
        updateProjectInStore(id, result);

        // Invalidate projects cache using global mutate
        await mutate('/api/projects', undefined, { revalidate: true });

        return result;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Failed to update project';
        setError(errorMsg);
        console.error('Failed to update project:', err);
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [updateProjectInStore],
  );

  return {
    updateProject,
    isLoading,
    error,
  };
}

interface UseDeleteProjectResult {
  deleteProject: (id: string) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for deleting a project with automatic cache update
 */
export function useDeleteProjectSWR(): UseDeleteProjectResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const removeProjectFromStore = useRemoveProjectStore();

  const deleteProject = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        setIsLoading(true);
        setError(null);
        await projectsService.deleteProject(id);

        // Update cache immediately
        removeProjectFromStore(id);

        // Invalidate projects cache using global mutate
        await mutate('/api/projects', undefined, { revalidate: true });

        return true;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Failed to delete project';
        setError(errorMsg);
        console.error('Failed to delete project:', err);
        return false;
      } finally {
        setIsLoading(false);
      }
    },
    [removeProjectFromStore],
  );

  return {
    deleteProject,
    isLoading,
    error,
  };
}
