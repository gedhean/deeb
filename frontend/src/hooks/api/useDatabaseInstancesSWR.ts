import useSWR, { mutate as globalMutate } from 'swr';
import { useCallback, useState } from 'react';
import { databaseInstancesService } from '@/lib/api/services/database-instances';
import type {
  DatabaseInstance,
  CreateDatabaseInstanceRequest,
  UpdateDatabaseInstanceRequest,
} from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { showSuccessToast } from '@/lib/errorHandling';

/**
 * Parameters for database instances list fetching.
 *
 * Permissive shape so pages can forward server-side filter params
 * (search, database_engine, environment_id, status, created_after/before, ...).
 */
export type UseDatabaseInstancesParams = Record<string, string | number | undefined>;

/**
 * SWR fetcher function for database instances list
 */
const fetchDatabaseInstances = async (params?: UseDatabaseInstancesParams) => {
  const response = await databaseInstancesService.listDatabaseInstances(params || { limit: 100 });
  return { items: response.items, total: response.total };
};

interface UseDatabaseInstancesResult {
  databaseInstances: DatabaseInstance[];
  total: number;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching database instances with SWR caching
 *
 * This hook uses SWR for automatic request deduplication and background revalidation.
 */
export function useDatabaseInstancesSWR(
  params?: UseDatabaseInstancesParams,
): UseDatabaseInstancesResult {
  // Create cache key that includes pagination params
  const cacheKey = params ? ['/api/database-instances', params] : '/api/database-instances';

  const {
    data: swrData,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<{ items: DatabaseInstance[]; total: number }>(
    cacheKey,
    () => fetchDatabaseInstances(params),
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
      : 'Failed to load database instances'
    : null;

  return {
    databaseInstances: swrData?.items || [],
    total: swrData?.total || 0,
    isLoading,
    error,
    refetch,
  };
}

// Mutation hooks

interface UseCreateDatabaseInstanceResult {
  createDatabaseInstance: (data: CreateDatabaseInstanceRequest) => Promise<DatabaseInstance | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for creating a database instance with automatic cache invalidation
 */
export function useCreateDatabaseInstance(): UseCreateDatabaseInstanceResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createDatabaseInstance = useCallback(
    async (data: CreateDatabaseInstanceRequest): Promise<DatabaseInstance | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await databaseInstancesService.createDatabaseInstance(data);

        // Invalidate database instances cache. The list is cached under an array key
        // (['/api/database-instances', params]); match by predicate so filtered lists refresh.
        await globalMutate(
          (key) =>
            (Array.isArray(key) && key[0] === '/api/database-instances') ||
            key === '/api/database-instances',
          undefined,
          { revalidate: true },
        );

        // Invalidate the new instance cache if we have the ID
        if (result?.id) {
          await globalMutate(`/api/database-instances/${result.id}`);
        }

        // Invalidate related caches that may list instances
        await globalMutate((key) => typeof key === 'string' && key.startsWith('/api/databases'));

        showSuccessToast('Database instance created successfully');
        return result;
      } catch (err) {
        const errorMsg =
          err instanceof APIError ? err.message : 'Failed to create database instance';
        setError(errorMsg);
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  return { createDatabaseInstance, isLoading, error };
}

interface UseUpdateDatabaseInstanceResult {
  updateDatabaseInstance: (
    id: string,
    data: UpdateDatabaseInstanceRequest,
  ) => Promise<DatabaseInstance | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for updating a database instance with automatic cache invalidation
 */
export function useUpdateDatabaseInstance(): UseUpdateDatabaseInstanceResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateDatabaseInstance = useCallback(
    async (id: string, data: UpdateDatabaseInstanceRequest): Promise<DatabaseInstance | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await databaseInstancesService.updateDatabaseInstance(id, data);

        // Invalidate database instances cache. The list is cached under an array key
        // (['/api/database-instances', params]); match by predicate so filtered lists refresh.
        await globalMutate(
          (key) =>
            (Array.isArray(key) && key[0] === '/api/database-instances') ||
            key === '/api/database-instances',
          undefined,
          { revalidate: true },
        );

        // Invalidate the specific instance cache
        await globalMutate(`/api/database-instances/${id}`);

        // Invalidate related caches that may reference this instance
        await globalMutate((key) => typeof key === 'string' && key.startsWith('/api/databases'));

        showSuccessToast('Database instance updated successfully');
        return result;
      } catch (err) {
        const errorMsg =
          err instanceof APIError ? err.message : 'Failed to update database instance';
        setError(errorMsg);
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  return { updateDatabaseInstance, isLoading, error };
}

interface UseDeleteDatabaseInstanceResult {
  deleteDatabaseInstance: (id: string) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for deleting a database instance with automatic cache invalidation
 */
export function useDeleteDatabaseInstance(): UseDeleteDatabaseInstanceResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const deleteDatabaseInstance = useCallback(async (id: string): Promise<boolean> => {
    try {
      setIsLoading(true);
      setError(null);
      await databaseInstancesService.deleteDatabaseInstance(id);

      // Invalidate database instances cache. The list is cached under an array key
      // (['/api/database-instances', params]); match by predicate so filtered lists refresh.
      await globalMutate(
        (key) =>
          (Array.isArray(key) && key[0] === '/api/database-instances') ||
          key === '/api/database-instances',
        undefined,
        { revalidate: true },
      );

      // Invalidate the specific instance cache
      await globalMutate(`/api/database-instances/${id}`);

      // Invalidate related caches that may reference this instance
      // This includes databases that belong to this instance
      await globalMutate((key) => typeof key === 'string' && key.startsWith('/api/databases'));

      showSuccessToast('Database instance deleted successfully');
      return true;
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to delete database instance';
      setError(errorMsg);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return { deleteDatabaseInstance, isLoading, error };
}

// Single instance hook

interface UseDatabaseInstanceByIdResult {
  databaseInstance: DatabaseInstance | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching a single database instance by ID with SWR caching
 */
export function useDatabaseInstanceById(id: string | undefined): UseDatabaseInstanceByIdResult {
  const {
    data: databaseInstance,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<DatabaseInstance>(
    id ? `/api/database-instances/${id}` : null,
    id ? () => databaseInstancesService.getDatabaseInstance(id) : null,
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
      : 'Failed to load database instance'
    : null;

  return {
    databaseInstance: databaseInstance || null,
    isLoading,
    error,
    refetch,
  };
}

// Test connection hook

interface TestConnectionResult {
  success: boolean;
  message?: string;
  error?: string;
  host?: string;
  port?: number;
  database?: string;
  engine?: string;
  username?: string;
}

interface UseTestConnectionResult {
  testConnection: (data: CreateDatabaseInstanceRequest) => Promise<TestConnectionResult>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for testing database instance connection with provided credentials
 * Used for testing connection before creating a new instance
 */
export function useTestConnection(): UseTestConnectionResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const testConnection = useCallback(
    async (data: CreateDatabaseInstanceRequest): Promise<TestConnectionResult> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await databaseInstancesService.testConnectionWithCredentials(data);
        return result;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Connection test failed';
        setError(errorMsg);
        return {
          success: false,
          error: errorMsg,
        };
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  return { testConnection, isLoading, error };
}
