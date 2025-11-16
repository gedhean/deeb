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
 * SWR fetcher function for database instances list
 */
const fetchDatabaseInstances = async () => {
  const response = await databaseInstancesService.listDatabaseInstances({ limit: 100 });
  return response.items;
};

interface UseDatabaseInstancesResult {
  databaseInstances: DatabaseInstance[];
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching database instances with SWR caching
 *
 * This hook uses SWR for automatic request deduplication and background revalidation.
 */
export function useDatabaseInstancesSWR(): UseDatabaseInstancesResult {
  const {
    data: databaseInstances,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<DatabaseInstance[]>('/api/database-instances', fetchDatabaseInstances, {
    revalidateIfStale: true,
    keepPreviousData: true,
  });

  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  const error = swrError
    ? swrError instanceof Error
      ? swrError.message
      : 'Failed to load database instances'
    : null;

  return {
    databaseInstances: databaseInstances || [],
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
  const { mutate } = useSWR('/api/database-instances');

  const createDatabaseInstance = useCallback(
    async (data: CreateDatabaseInstanceRequest): Promise<DatabaseInstance | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await databaseInstancesService.createDatabaseInstance(data);

        // Invalidate list cache
        await mutate();

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
    [mutate],
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
  const { mutate } = useSWR('/api/database-instances');

  const updateDatabaseInstance = useCallback(
    async (id: string, data: UpdateDatabaseInstanceRequest): Promise<DatabaseInstance | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await databaseInstancesService.updateDatabaseInstance(id, data);

        // Invalidate list cache
        await mutate();

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
    [mutate],
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
  const { mutate } = useSWR('/api/database-instances');

  const deleteDatabaseInstance = useCallback(
    async (id: string): Promise<boolean> => {
      try {
        setIsLoading(true);
        setError(null);
        await databaseInstancesService.deleteDatabaseInstance(id);

        // Invalidate list cache
        await mutate();

        // Invalidate the specific instance cache
        await globalMutate(`/api/database-instances/${id}`);

        // Invalidate related caches that may reference this instance
        // This includes databases that belong to this instance
        await globalMutate((key) => typeof key === 'string' && key.startsWith('/api/databases'));

        showSuccessToast('Database instance deleted successfully');
        return true;
      } catch (err) {
        const errorMsg =
          err instanceof APIError ? err.message : 'Failed to delete database instance';
        setError(errorMsg);
        return false;
      } finally {
        setIsLoading(false);
      }
    },
    [mutate],
  );

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
  error?: string;
  details?: string;
  instance?: DatabaseInstance;
}

interface UseTestConnectionResult {
  testConnection: (data: CreateDatabaseInstanceRequest) => Promise<TestConnectionResult>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for testing database instance connection
 *
 * Note: A temporary test instance may be created and should be deleted by the caller
 */
export function useTestConnection(): UseTestConnectionResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const testConnection = useCallback(
    async (data: CreateDatabaseInstanceRequest): Promise<TestConnectionResult> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await databaseInstancesService.testConnection(data);
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
