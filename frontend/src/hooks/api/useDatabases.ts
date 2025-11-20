import useSWR from 'swr';
import { useCallback, useMemo } from 'react';
import { databasesService } from '@/lib/api/services/databases';
import type { Database } from '@/lib/api/types';
import {
  useDatabases as useDatabasesStore,
  useDatabasesLoading,
  useDatabasesError,
  useGetDatabaseById as useGetDatabaseByIdStore,
} from '@/store';

/**
 * SWR fetcher function for databases list
 */
const fetchDatabases = async (filters?: {
  project_id?: string;
  environment_id?: string;
  database_instance_id?: string;
  limit?: number;
  offset?: number;
}) => {
  const response = await databasesService.listDatabases({
    limit: filters?.limit || 1000,
    ...filters,
  });
  return { items: response.items, total: response.total };
};

interface UseDatabasesResult {
  databases: Database[];
  total: number;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
  getDatabaseById: (databaseId: string) => Database | undefined;
  getDatabasesByProjectId: (projectId: string) => Database[];
}

interface UseDatabasesOptions {
  project_id?: string;
  environment_id?: string;
  database_instance_id?: string;
  limit?: number;
  offset?: number;
}

/**
 * Hook for fetching databases list with SWR caching and Zustand store integration
 *
 * This hook combines SWR for automatic request deduplication and background revalidation
 * with Zustand store for immediate cache availability across components.
 *
 * @param options - Optional filters for project_id, environment_id, database_instance_id, and limit
 */
export function useDatabases(options: UseDatabasesOptions = {}): UseDatabasesResult {
  const storeDatabases = useDatabasesStore();
  const storeLoading = useDatabasesLoading();
  const storeError = useDatabasesError();
  const getDatabaseByIdFromStore = useGetDatabaseByIdStore();

  // Stabilize options object reference to prevent infinite loops
  const stableOptions = useMemo(
    () => options,
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [
      options.project_id,
      options.environment_id,
      options.database_instance_id,
      options.limit,
      options.offset,
    ],
  );

  // Create a unique SWR key based on filters
  const swrKey = useMemo(() => {
    if (
      stableOptions.project_id ||
      stableOptions.environment_id ||
      stableOptions.database_instance_id ||
      stableOptions.limit ||
      stableOptions.offset
    ) {
      const params: Record<string, string> = {};
      if (stableOptions.project_id) params.project_id = stableOptions.project_id;
      if (stableOptions.environment_id) params.environment_id = stableOptions.environment_id;
      if (stableOptions.database_instance_id)
        params.database_instance_id = stableOptions.database_instance_id;
      if (stableOptions.limit) params.limit = stableOptions.limit.toString();
      if (stableOptions.offset) params.offset = stableOptions.offset.toString();
      return `/api/databases?${new URLSearchParams(params).toString()}`;
    }
    return '/api/databases';
  }, [stableOptions]);

  // SWR hook for automatic caching and revalidation
  const {
    data: swrData,
    error: swrError,
    isLoading: swrLoading,
    mutate,
  } = useSWR<{ items: Database[]; total: number }>(swrKey, () => fetchDatabases(stableOptions), {
    fallbackData:
      storeDatabases.length > 0
        ? { items: storeDatabases, total: storeDatabases.length }
        : undefined,
    revalidateIfStale: true,
    keepPreviousData: true,
  });

  // Manual refetch function
  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  // Helper to get database by ID
  const getDatabaseById = useCallback(
    (databaseId: string) => {
      const databases = swrData?.items || storeDatabases;
      return (
        databases.find((database) => database.id === databaseId) ||
        getDatabaseByIdFromStore(databaseId)
      );
    },
    [swrData, storeDatabases, getDatabaseByIdFromStore],
  );

  // Helper to get databases by project ID
  const getDatabasesByProjectId = useCallback(
    (projectId: string) => {
      const databases = swrData?.items || storeDatabases;
      return databases.filter(
        (database) => database.project_id && database.project_id === projectId,
      );
    },
    [swrData, storeDatabases],
  );

  const databases = swrData?.items || storeDatabases;
  const total = swrData?.total || storeDatabases.length;
  const isLoading = swrLoading || storeLoading;
  const error = swrError
    ? swrError instanceof Error
      ? swrError.message
      : 'Failed to load databases'
    : storeError;

  return {
    databases,
    total,
    isLoading,
    error,
    refetch,
    getDatabaseById,
    getDatabasesByProjectId,
  };
}

/**
 * Hook for getting a single database by ID
 *
 * This hook uses the cached databases list instead of making a separate API call
 */
export function useDatabaseById(databaseId: string | undefined): {
  database: Database | undefined;
  isLoading: boolean;
  error: string | null;
} {
  const { isLoading, error, getDatabaseById } = useDatabases();

  const database = databaseId ? getDatabaseById(databaseId) : undefined;

  return {
    database,
    isLoading,
    error,
  };
}
