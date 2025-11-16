import useSWR from 'swr';
import { useCallback } from 'react';
import { setupService, type SetupStatusResponse } from '@/lib/api/services/setup';

/**
 * SWR fetcher function for setup status
 */
const fetchSetupStatus = async () => {
  return await setupService.getSetupStatus();
};

interface UseSetupResult {
  setupStatus: SetupStatusResponse | null;
  requiresSetup: boolean;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for checking setup status with SWR caching
 *
 * This hook provides automatic request deduplication and caching for setup status.
 * Multiple components can call this hook and only one API call will be made.
 *
 * The setup status is cached for a long time since it rarely changes (only when
 * the first user is created).
 */
export function useSetupSWR(): UseSetupResult {
  const {
    data: setupStatus,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<SetupStatusResponse>('/api/setup/status', fetchSetupStatus, {
    revalidateIfStale: false, // Don't revalidate automatically - setup status rarely changes
    revalidateOnFocus: false, // Don't revalidate on window focus
    revalidateOnReconnect: false, // Don't revalidate on reconnect
    dedupingInterval: 60000, // Deduplicate requests for 60 seconds
  });

  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  const error = swrError
    ? swrError instanceof Error
      ? swrError.message
      : 'Failed to check setup status'
    : null;

  return {
    setupStatus: setupStatus || null,
    requiresSetup: setupStatus?.requires_setup || false,
    isLoading,
    error,
    refetch,
  };
}
