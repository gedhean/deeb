import useSWR from 'swr';
import { useCallback, useMemo } from 'react';
import { auditLogService } from '@/lib/api/services/auditLogs';
import type { AuditLog, ListAuditLogsParams } from '@/lib/api/types';

interface ListAuditLogsResponse {
  audit_logs: AuditLog[];
  total: number;
}

/**
 * SWR fetcher function for audit logs
 */
const fetchAuditLogs = async (params: ListAuditLogsParams): Promise<ListAuditLogsResponse> => {
  return await auditLogService.listAuditLogs(params);
};

interface UseAuditLogsSWRResult {
  auditLogs: AuditLog[];
  total: number;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching audit logs with SWR caching
 *
 * This hook provides automatic request deduplication and caching for audit logs.
 * The cache key includes all filter parameters to ensure proper cache separation.
 *
 * @param params - Filter parameters for audit logs (user_id, action, target_type, etc.)
 * @param enabled - Whether to fetch data (default: true)
 */
export function useAuditLogsSWR(
  params: ListAuditLogsParams,
  enabled: boolean = true,
): UseAuditLogsSWRResult {
  // Create a stable cache key from params
  const cacheKey = useMemo(() => {
    if (!enabled) return null;
    // Sort keys to ensure consistent cache key
    const sortedParams = Object.keys(params)
      .sort()
      .reduce(
        (acc, key) => {
          acc[key] = params[key as keyof ListAuditLogsParams];
          return acc;
        },
        {} as Record<string, unknown>,
      );
    return `/api/audit-logs?${JSON.stringify(sortedParams)}`;
  }, [params, enabled]);

  const {
    data,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<ListAuditLogsResponse>(cacheKey, cacheKey ? () => fetchAuditLogs(params) : null, {
    revalidateIfStale: true,
    keepPreviousData: true,
  });

  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  const error = swrError
    ? swrError instanceof Error
      ? swrError.message
      : 'Failed to load audit logs'
    : null;

  return {
    auditLogs: data?.audit_logs || [],
    total: data?.total || 0,
    isLoading,
    error,
    refetch,
  };
}
