import useSWR, { mutate } from 'swr';
import { useCallback, useState } from 'react';
import { environmentsService } from '@/lib/api/services/environments';
import type {
  Environment,
  CreateEnvironmentRequest,
  UpdateEnvironmentRequest,
} from '@/lib/api/types';
import { APIError } from '@/lib/api/types';

/**
 * Parameters for environments list fetching
 */
export interface UseEnvironmentsParams {
  limit?: number;
  offset?: number;
}

/**
 * SWR fetcher function for environments list
 */
const fetchEnvironments = async (params?: UseEnvironmentsParams) => {
  const response = await environmentsService.listEnvironments(params || { limit: 100 });
  return { items: response.items, total: response.total };
};

interface UseEnvironmentsResult {
  environments: Environment[];
  total: number;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching environments with SWR caching
 *
 * This hook uses SWR for automatic request deduplication and background revalidation.
 */
export function useEnvironmentsSWR(params?: UseEnvironmentsParams): UseEnvironmentsResult {
  // Create cache key that includes pagination params
  const cacheKey = params ? ['/api/environments', params] : '/api/environments';

  const {
    data: swrData,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<{ items: Environment[]; total: number }>(cacheKey, () => fetchEnvironments(params), {
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
    environments: swrData?.items || [],
    total: swrData?.total || 0,
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

  const createEnvironment = useCallback(
    async (data: CreateEnvironmentRequest): Promise<Environment | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await environmentsService.createEnvironment(data);

        // Invalidate environments cache using global mutate
        await mutate('/api/environments', undefined, { revalidate: true });

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
    [],
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

  const updateEnvironment = useCallback(
    async (id: string, data: UpdateEnvironmentRequest): Promise<Environment | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await environmentsService.updateEnvironment(id, data);

        // Invalidate environments cache using global mutate
        await mutate('/api/environments', undefined, { revalidate: true });

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
    [],
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

  const deleteEnvironment = useCallback(async (id: string): Promise<boolean> => {
    try {
      setIsLoading(true);
      setError(null);
      await environmentsService.deleteEnvironment(id);

      // Invalidate environments cache using global mutate
      await mutate('/api/environments', undefined, { revalidate: true });

      return true;
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to delete environment';
      setError(errorMsg);
      console.error('Failed to delete environment:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return {
    deleteEnvironment,
    isLoading,
    error,
  };
}
