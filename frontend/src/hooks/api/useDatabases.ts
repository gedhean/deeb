import useSWR from 'swr';
import { useCallback } from 'react';
import { databasesService } from '@/lib/api/services/databases';
import type { Database } from '@/lib/api/types';
import {
  useDatabases as useDatabasesStore,
  useDatabasesLoading,
  useDatabasesError,
  useGetDatabaseById as useGetDatabaseByIdStore,
} from '@/store';

/**
 * Options for databases list fetching.
 *
 * Permissive shape so pages can forward server-side filter params
 * (search, project_id, environment_id, database_instance_id, status,
 * created_after/before, ...) built by `useServerTableFilters`.
 */
export type UseDatabasesOptions = Record<string, string | number | undefined>;

/**
 * SWR fetcher function for databases list
 */
const fetchDatabases = async (filters?: UseDatabasesOptions) => {
  const response = await databasesService.listDatabases({
    limit: 1000,
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

/**
 * Hook for fetching databases list with SWR caching and Zustand store integration
 *
 * This hook combines SWR for automatic request deduplication and background revalidation
 * with Zustand store for immediate cache availability across components.
 *
 * @param options - Optional server-side filter params (pagination, search, facets, dates)
 */
export function useDatabases(options: UseDatabasesOptions = {}): UseDatabasesResult {
  const storeDatabases = useDatabasesStore();
  const storeLoading = useDatabasesLoading();
  const storeError = useDatabasesError();
  const getDatabaseByIdFromStore = useGetDatabaseByIdStore();

  // Include filters in the cache key so changing a filter refetches. SWR's stable
  // hashing normalizes object keys, so passing `options` directly is safe even when
  // callers hand in a fresh object literal each render.
  const hasFilters = Object.values(options).some(
    (value) => value !== undefined && value !== null && value !== '',
  );
  const swrKey = hasFilters ? ['/api/databases', options] : '/api/databases';

  // SWR hook for automatic caching and revalidation
  const {
    data: swrData,
    error: swrError,
    isLoading: swrLoading,
    mutate,
  } = useSWR<{ items: Database[]; total: number }>(swrKey, () => fetchDatabases(options), {
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
