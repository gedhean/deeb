import useSWR from 'swr';
import { useCallback } from 'react';
import { healthService } from '@/lib/api/services/health';
import type { HealthCheckResponse } from '@/lib/api/types';

/**
 * SWR fetcher function for health check
 */
const fetchHealth = async () => {
  return await healthService.check();
};

interface UseHealthResult {
  health: HealthCheckResponse | null;
  isHealthy: boolean;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for checking system health with SWR caching
 *
 * This hook provides automatic request deduplication and background revalidation
 * for health checks. Use this instead of calling healthService.check() directly.
 */
export function useHealthSWR(): UseHealthResult {
  const {
    data: health,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<HealthCheckResponse>('/health', fetchHealth, {
    revalidateIfStale: true,
    keepPreviousData: true,
    // Revalidate health more frequently since it's important
    refreshInterval: 30000, // Check health every 30 seconds
  });

  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  const error = swrError
    ? swrError instanceof Error
      ? swrError.message
      : 'Failed to check system health'
    : null;

  return {
    health: health || null,
    isHealthy: health?.status === 'healthy',
    isLoading,
    error,
    refetch,
  };
}
