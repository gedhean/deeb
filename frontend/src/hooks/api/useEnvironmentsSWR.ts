import useSWR from 'swr';
import { useCallback, useState } from 'react';
import { environmentsService } from '@/lib/api/services/environments';
import type {
  Environment,
  CreateEnvironmentRequest,
  UpdateEnvironmentRequest,
} from '@/lib/api/types';
import { APIError } from '@/lib/api/types';

/**
 * SWR fetcher function for environments list
 */
const fetchEnvironments = async () => {
  const response = await environmentsService.listEnvironments({ limit: 100 });
  return response.items;
};

interface UseEnvironmentsResult {
  environments: Environment[];
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching environments with SWR caching
 *
 * This hook uses SWR for automatic request deduplication and background revalidation.
 */
export function useEnvironmentsSWR(): UseEnvironmentsResult {
  const {
    data: environments,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<Environment[]>('/api/environments', fetchEnvironments, {
    revalidateIfStale: true,
    keepPreviousData: true,
  });

  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  const error = swrError
    ? swrError instanceof Error
      ? swrError.message
      : 'Failed to load environments'
    : null;

  return {
    environments: environments || [],
    isLoading,
    error,
    refetch,
  };
}

interface UseCreateEnvironmentResult {
  createEnvironment: (data: CreateEnvironmentRequest) => Promise<Environment | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for creating an environment with automatic cache update
 */
export function useCreateEnvironmentSWR(): UseCreateEnvironmentResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { mutate: mutateEnvironments } = useSWR('/api/environments');

  const createEnvironment = useCallback(
    async (data: CreateEnvironmentRequest): Promise<Environment | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await environmentsService.createEnvironment(data);

        // Update cache immediately
        await mutateEnvironments();

        return result;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Failed to create environment';
        setError(errorMsg);
        console.error('Failed to create environment:', err);
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [mutateEnvironments],
  );

  return {
    createEnvironment,
    isLoading,
    error,
  };
}

interface UseUpdateEnvironmentResult {
  updateEnvironment: (id: string, data: UpdateEnvironmentRequest) => Promise<Environment | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for updating an environment with automatic cache update
 */
export function useUpdateEnvironmentSWR(): UseUpdateEnvironmentResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { mutate: mutateEnvironments } = useSWR('/api/environments');

  const updateEnvironment = useCallback(
    async (id: string, data: UpdateEnvironmentRequest): Promise<Environment | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await environmentsService.updateEnvironment(id, data);

        // Update cache immediately
        await mutateEnvironments();

        return result;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Failed to update environment';
        setError(errorMsg);
        console.error('Failed to update environment:', err);
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [mutateEnvironments],
  );

  return {
    updateEnvironment,
    isLoading,
    error,
  };
}

interface UseDeleteEnvironmentResult {
  deleteEnvironment: (id: string) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for deleting an environment with automatic cache update
 */
export function useDeleteEnvironmentSWR(): UseDeleteEnvironmentResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { mutate: mutateEnvironments } = useSWR('/api/environments');

  const deleteEnvironment = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        setIsLoading(true);
        setError(null);
        await environmentsService.deleteEnvironment(id);

        // Update cache immediately
        await mutateEnvironments();

        return true;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Failed to delete environment';
        setError(errorMsg);
        console.error('Failed to delete environment:', err);
        return false;
      } finally {
        setIsLoading(false);
      }
    },
    [mutateEnvironments],
  );

  return {
    deleteEnvironment,
    isLoading,
    error,
  };
}
