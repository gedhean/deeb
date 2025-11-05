import { useState, useCallback, useEffect } from 'react';
import { projectsService } from '@/lib/api/services/projects';
import type { Project, CreateProjectRequest, UpdateProjectRequest } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';

interface UseProjectsOptions {
  enabled?: boolean;
  limit?: number;
  offset?: number;
}

interface UseProjectsResult {
  projects: Project[];
  total: number;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching projects list
 */
export function useProjects(options: UseProjectsOptions = {}): UseProjectsResult {
  const { enabled = true, limit, offset } = options;
  const [projects, setProjects] = useState<Project[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchProjects = useCallback(async () => {
    if (!enabled) return;

    try {
      setIsLoading(true);
      setError(null);
      const response = await projectsService.listProjects({ limit, offset });
      setProjects(response.items);
      setTotal(response.total);
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to load projects';
      setError(errorMsg);
      console.error('Failed to load projects:', err);
    } finally {
      setIsLoading(false);
    }
  }, [enabled, limit, offset]);

  useEffect(() => {
    fetchProjects();
  }, [fetchProjects]);

  return {
    projects,
    total,
    isLoading,
    error,
    refetch: fetchProjects,
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
 * Hook for fetching a single project by ID
 */
export function useProject(
  id: string | undefined,
  options: UseProjectOptions = {},
): UseProjectResult {
  const { enabled = true } = options;
  const [project, setProject] = useState<Project | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchProject = useCallback(async () => {
    if (!enabled || !id) return;

    try {
      setIsLoading(true);
      setError(null);
      const data = await projectsService.getProject(id);
      setProject(data);
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to load project';
      setError(errorMsg);
      console.error('Failed to load project:', err);
    } finally {
      setIsLoading(false);
    }
  }, [enabled, id]);

  useEffect(() => {
    fetchProject();
  }, [fetchProject]);

  return {
    project,
    isLoading,
    error,
    refetch: fetchProject,
  };
}

interface UseCreateProjectResult {
  createProject: (data: CreateProjectRequest) => Promise<Project | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for creating a project
 */
export function useCreateProject(): UseCreateProjectResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createProject = useCallback(async (data: CreateProjectRequest): Promise<Project | null> => {
    try {
      setIsLoading(true);
      setError(null);
      const result = await projectsService.createProject(data);
      return result;
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to create project';
      setError(errorMsg);
      console.error('Failed to create project:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

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
 * Hook for updating a project
 */
export function useUpdateProject(): UseUpdateProjectResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateProject = useCallback(
    async (id: string, data: UpdateProjectRequest): Promise<Project | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await projectsService.updateProject(id, data);
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
    [],
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
 * Hook for deleting a project
 */
export function useDeleteProject(): UseDeleteProjectResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const deleteProject = useCallback(async (id: string): Promise<boolean> => {
    try {
      setIsLoading(true);
      setError(null);
      await projectsService.deleteProject(id);
      return true;
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to delete project';
      setError(errorMsg);
      console.error('Failed to delete project:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return {
    deleteProject,
    isLoading,
    error,
  };
}
