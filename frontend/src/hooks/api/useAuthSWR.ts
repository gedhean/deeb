import useSWR from 'swr';
import { useCallback } from 'react';
import { authService } from '@/lib/api/services/auth';
import type { User } from '@/lib/api/types';

/**
 * SWR fetcher function for current user
 */
const fetchCurrentUser = async () => {
  return await authService.getCurrentUser();
};

interface UseCurrentUserResult {
  user: User | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
  isAuthenticated: boolean;
}

/**
 * Hook for fetching current user with SWR caching
 *
 * This hook provides automatic request deduplication and caching
 * for the authenticated user. Use this instead of calling authService.getCurrentUser() directly.
 *
 * Benefits:
 * - Automatic request deduplication (multiple calls = single API request)
 * - Cached data available immediately on subsequent mounts
 * - Works correctly with React StrictMode (no duplicate API calls)
 */
export function useCurrentUserSWR(): UseCurrentUserResult {
  const {
    data: user,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<User>('/api/auth/me', fetchCurrentUser, {
    revalidateIfStale: true,
    keepPreviousData: true,
    revalidateOnFocus: false, // Disable focus revalidation to prevent unnecessary refetches
    dedupingInterval: 2000, // Deduplicate requests within 2 seconds
    shouldRetryOnError: false, // Don't retry on auth errors (401/403)
    onError: (error) => {
      // Silently handle authentication errors - user is just not logged in
      if (error?.statusCode === 401 || error?.statusCode === 403) {
        return;
      }
      console.error('Failed to load current user:', error);
    },
  });

  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  const error = swrError
    ? swrError instanceof Error
      ? swrError.message
      : 'Failed to load user'
    : null;

  return {
    user: user || null,
    isLoading,
    error,
    refetch,
    isAuthenticated: !!user,
  };
}
