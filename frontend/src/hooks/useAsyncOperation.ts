import { useState, useCallback } from 'react';
import { showSuccessToast, showErrorToast } from '@/lib/errorHandling';

interface UseAsyncOperationOptions {
  successMessage?: string;
  errorMessage?: string;
  showSuccessToast?: boolean;
  showErrorToast?: boolean;
}

/**
 * Hook for handling async operations with loading states and toast notifications
 */
export function useAsyncOperation<T = void>(options: UseAsyncOperationOptions = {}) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const execute = useCallback(
    async (
      operation: () => Promise<T>,
      customOptions?: Partial<UseAsyncOperationOptions>,
    ): Promise<T | null> => {
      setIsLoading(true);
      setError(null);

      const finalOptions = { ...options, ...customOptions };

      try {
        const result = await operation();

        if (finalOptions.showSuccessToast && finalOptions.successMessage) {
          showSuccessToast(finalOptions.successMessage);
        }

        setIsLoading(false);
        return result;
      } catch (err) {
        const errorMsg =
          finalOptions.errorMessage || (err instanceof Error ? err.message : 'An error occurred');

        setError(errorMsg);

        if (finalOptions.showErrorToast) {
          showErrorToast(err, finalOptions.errorMessage);
        }

        setIsLoading(false);
        return null;
      }
    },
    [options],
  );

  const reset = useCallback(() => {
    setError(null);
    setIsLoading(false);
  }, []);

  return {
    execute,
    isLoading,
    error,
    reset,
  };
}
