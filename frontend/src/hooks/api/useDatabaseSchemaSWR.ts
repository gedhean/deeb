import { useCallback, useState } from 'react';
import useSWR from 'swr';
import { schemaService } from '@/lib/api/services/schema';
import { APIError, type DatabaseSchemaMetadata, type TableDetail } from '@/lib/api/types';
import { showSuccessToast } from '@/lib/errorHandling';

interface UseDatabaseSchemaResult {
  schemaMetadata: DatabaseSchemaMetadata | null;
  isLoading: boolean;
  error: string;
  refetch: () => Promise<void>;
}

interface UseRefreshDatabaseSchemaResult {
  refreshDatabaseSchema: (databaseId: string) => Promise<DatabaseSchemaMetadata | null>;
  isRefreshing: boolean;
  error: string | null;
}

/**
 * Hook to fetch database schema metadata with SWR caching
 * @param databaseId - The database ID to fetch schema for
 * @returns Schema metadata, loading state, error, and refetch function
 */
export function useDatabaseSchemaSWR(databaseId: string | undefined): UseDatabaseSchemaResult {
  const {
    data: schemaMetadata,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<DatabaseSchemaMetadata>(
    databaseId ? `/api/v1/databases/${databaseId}/schema` : null,
    databaseId ? () => schemaService.getDatabaseSchema(databaseId) : null,
    {
      revalidateIfStale: true,
      keepPreviousData: true,
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
    },
  );

  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  return {
    schemaMetadata: schemaMetadata || null,
    isLoading,
    error: swrError
      ? swrError instanceof APIError
        ? swrError.message
        : 'Failed to load database schema. Please try again.'
      : '',
    refetch,
  };
}

/**
 * Hook to refresh database schema metadata (forces re-extraction from database)
 * @returns Refresh function, loading state, and error
 */
export function useRefreshDatabaseSchema(): UseRefreshDatabaseSchemaResult {
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshDatabaseSchema = useCallback(
    async (databaseId: string): Promise<DatabaseSchemaMetadata | null> => {
      try {
        setIsRefreshing(true);
        setError(null);
        const result = await schemaService.refreshDatabaseSchema(databaseId);
        showSuccessToast('Database schema refreshed successfully');
        return result;
      } catch (err) {
        const errorMsg =
          err instanceof APIError
            ? err.message
            : 'Failed to refresh database schema. Please try again.';
        setError(errorMsg);
        return null;
      } finally {
        setIsRefreshing(false);
      }
    },
    [],
  );

  return { refreshDatabaseSchema, isRefreshing, error };
}

interface UseTableDetailOptions {
  enabled?: boolean;
}

interface UseTableDetailResult {
  tableDetail: TableDetail | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook to fetch table detail with SWR caching
 * @param databaseId - The database ID
 * @param tableName - The table name to fetch details for
 * @param options - Options to control fetching behavior
 * @returns Table detail, loading state, error, and refetch function
 */
export function useTableDetailSWR(
  databaseId: string | undefined,
  tableName: string | null | undefined,
  options: UseTableDetailOptions = {},
): UseTableDetailResult {
  const { enabled = true } = options;

  const shouldFetch = enabled && databaseId && tableName;
  const {
    data: tableDetail,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<TableDetail>(
    shouldFetch ? `/api/v1/databases/${databaseId}/schema/tables/${tableName}` : null,
    shouldFetch && databaseId && tableName
      ? () => schemaService.getTableDetail(databaseId, tableName)
      : null,
    {
      revalidateIfStale: true,
      keepPreviousData: true,
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
    },
  );

  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  const error = swrError
    ? swrError instanceof APIError
      ? swrError.message
      : 'Failed to load table details. Please try again.'
    : null;

  return {
    tableDetail: tableDetail || null,
    isLoading,
    error,
    refetch,
  };
}
