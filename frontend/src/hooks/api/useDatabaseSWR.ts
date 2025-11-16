import useSWR from 'swr';
import { useCallback, useState } from 'react';
import { databasesService } from '@/lib/api/services/databases';
import type { Database, ChangePlan, CreateDatabaseRequest } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';

/**
 * SWR fetcher function for single database
 */
const fetchDatabase = async (id: string) => {
  return await databasesService.getDatabase(id);
};

interface UseDatabaseOptions {
  enabled?: boolean;
}

interface UseDatabaseResult {
  database: Database | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching a single database by ID with SWR
 */
export function useDatabaseSWR(
  id: string | undefined,
  options: UseDatabaseOptions = {},
): UseDatabaseResult {
  const { enabled = true } = options;

  const shouldFetch = enabled && id;
  const {
    data: database,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<Database>(
    shouldFetch ? `/api/databases/${id}` : null,
    shouldFetch && id ? () => fetchDatabase(id) : null,
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
      : 'Failed to load database'
    : null;

  return {
    database: database || null,
    isLoading,
    error,
    refetch,
  };
}

// Mutation hooks

interface UseCreateDatabaseResult {
  createDatabase: (
    instanceId: string,
    data: CreateDatabaseRequest,
  ) => Promise<Database | ChangePlan | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for creating a database with automatic cache invalidation
 *
 * Returns either a Database (if directly created) or a ChangePlan (if requires approval)
 */
export function useCreateDatabase(): UseCreateDatabaseResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createDatabase = useCallback(
    async (
      instanceId: string,
      data: CreateDatabaseRequest,
    ): Promise<Database | ChangePlan | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await databasesService.createDatabaseForInstance(instanceId, data);

        // Invalidate databases cache
        const { mutate: mutateGlobal } = await import('swr');
        mutateGlobal((key: string) => typeof key === 'string' && key.startsWith('/api/databases'));

        return result;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Failed to create database';
        setError(errorMsg);
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  return { createDatabase, isLoading, error };
}
