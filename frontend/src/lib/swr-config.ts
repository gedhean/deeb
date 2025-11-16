import type { SWRConfiguration } from 'swr';
import { APIError } from './api/types';

/**
 * Global SWR Configuration
 *
 * This configuration provides:
 * - Automatic request deduplication (dedupingInterval: 2000ms)
 * - Background revalidation on window focus and reconnect
 * - Retry logic for failed requests
 * - Error handling compatible with APIError type
 */
export const swrConfig: SWRConfiguration = {
  // Deduplication interval - requests with the same key within this window are deduplicated
  dedupingInterval: 2000,

  // Revalidation settings
  revalidateOnFocus: true, // Revalidate when window regains focus
  revalidateOnReconnect: true, // Revalidate when network reconnects
  revalidateIfStale: true, // Revalidate if data is stale

  // Cache settings
  focusThrottleInterval: 5000, // Throttle revalidation on focus to once per 5 seconds

  // Error retry settings
  errorRetryInterval: 5000, // Wait 5 seconds between retries
  errorRetryCount: 3, // Maximum 3 retry attempts

  // Conditional retry - don't retry on client errors (4xx) except 408 (timeout)
  onErrorRetry: (error, _key, _config, revalidate, { retryCount }) => {
    // Never retry on 404, 429 (rate limit), or other client errors except timeout
    if (error instanceof APIError) {
      if (error.statusCode === 404) return; // Resource not found - no point retrying
      if (error.statusCode === 429) return; // Rate limited - retrying makes it worse
      if (error.statusCode >= 400 && error.statusCode < 500 && error.statusCode !== 408) {
        return; // Client errors (except timeout) - don't retry
      }
    }

    // Stop after max retries
    if (retryCount >= 3) return;

    // Retry after 5 seconds with exponential backoff
    setTimeout(() => revalidate({ retryCount }), 5000 * Math.pow(2, retryCount));
  },

  // Global error handler for logging
  onError: (error) => {
    if (error instanceof APIError) {
      console.error('SWR Error:', {
        message: error.message,
        statusCode: error.statusCode,
        response: error.response,
      });
    } else {
      console.error('SWR Error:', error);
    }
  },

  // Success handler for debugging (can be removed in production)
  onSuccess: (data, key) => {
    if (import.meta.env.DEV) {
      console.debug('SWR Success:', { key, data });
    }
  },
};
