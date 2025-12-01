import useSWR, { mutate as globalMutate } from 'swr';
import { useCallback, useState } from 'react';
import { changePlanService } from '@/lib/api/services/changePlans';
import type {
  ChangePlan,
  ChangePlanListResponse,
  ListChangePlansParams,
  CreateChangePlanRequest,
  UpdateChangePlanRequest,
  ApproveChangePlanRequest,
  CloseChangePlanRequest,
  ExecutionLogHistoryResponse,
} from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import { showSuccessToast } from '@/lib/errorHandling';

/**
 * SWR fetcher function for change plans list
 */
const fetchChangePlans = async (params?: ListChangePlansParams) => {
  return await changePlanService.listChangePlans(params);
};

/**
 * SWR fetcher function for single change plan
 */
const fetchChangePlan = async (id: string) => {
  return await changePlanService.getChangePlan(id);
};

interface UseChangePlansResult {
  changePlans: ChangePlan[];
  total: number;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching change plans list with SWR caching
 *
 * This hook provides automatic request deduplication and background revalidation
 * for change plans lists.
 */
export function useChangePlansSWR(params?: ListChangePlansParams): UseChangePlansResult {
  // Create a stable cache key that includes params
  const cacheKey = params
    ? `/api/change-plans?${new URLSearchParams(
        Object.entries(params).reduce(
          (acc, [key, value]) => {
            if (value !== undefined) {
              acc[key] = String(value);
            }
            return acc;
          },
          {} as Record<string, string>,
        ),
      ).toString()}`
    : '/api/change-plans';

  const {
    data,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<ChangePlanListResponse>(cacheKey, () => fetchChangePlans(params), {
    revalidateIfStale: true,
    keepPreviousData: true,
  });

  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  const error = swrError
    ? swrError instanceof Error
      ? swrError.message
      : 'Failed to load change plans'
    : null;

  return {
    changePlans: data?.change_plans || [],
    total: data?.total || 0,
    isLoading,
    error,
    refetch,
  };
}

interface UseChangePlanOptions {
  enabled?: boolean;
}

interface UseChangePlanResult {
  changePlan: ChangePlan | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching a single change plan by ID with SWR
 */
export function useChangePlanSWR(
  id: string | undefined,
  options: UseChangePlanOptions = {},
): UseChangePlanResult {
  const { enabled = true } = options;

  const shouldFetch = enabled && id;
  const {
    data: changePlan,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<ChangePlan>(
    shouldFetch ? `/api/change-plans/${id}` : null,
    shouldFetch && id ? () => fetchChangePlan(id) : null,
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
      : 'Failed to load change plan'
    : null;

  return {
    changePlan: changePlan || null,
    isLoading,
    error,
    refetch,
  };
}

interface UseCreateChangePlanResult {
  createChangePlan: (data: CreateChangePlanRequest) => Promise<ChangePlan | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for creating a change plan with automatic cache update
 */
export function useCreateChangePlanSWR(): UseCreateChangePlanResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createChangePlan = useCallback(
    async (data: CreateChangePlanRequest): Promise<ChangePlan | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await changePlanService.createChangePlan(data);

        // Invalidate all change plans caches using global mutate with filter function
        await globalMutate(
          (key) => typeof key === 'string' && key.startsWith('/api/change-plans'),
          undefined,
          { revalidate: true },
        );

        return result;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Failed to create change plan';
        setError(errorMsg);
        console.error('Failed to create change plan:', err);
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  return {
    createChangePlan,
    isLoading,
    error,
  };
}

interface UseUpdateChangePlanResult {
  updateChangePlan: (id: string, data: UpdateChangePlanRequest) => Promise<ChangePlan | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for updating a change plan with automatic cache update
 */
export function useUpdateChangePlanSWR(): UseUpdateChangePlanResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateChangePlan = useCallback(
    async (id: string, data: UpdateChangePlanRequest): Promise<ChangePlan | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await changePlanService.updateChangePlan(id, data);

        // Invalidate all change plans caches using global mutate with filter function
        await globalMutate(
          (key) => typeof key === 'string' && key.startsWith('/api/change-plans'),
          undefined,
          { revalidate: true },
        );

        return result;
      } catch (err) {
        let errorMsg = 'Failed to update change plan';
        if (err instanceof APIError) {
          // Use details.message if available, otherwise fall back to error message
          errorMsg = err.response?.details?.message || err.response?.error || err.message;
        }
        setError(errorMsg);
        console.error('Failed to update change plan:', err);
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  return {
    updateChangePlan,
    isLoading,
    error,
  };
}

interface UseTriggerSQLReviewResult {
  triggerSQLReview: (id: string) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for triggering SQL review on a change plan with automatic cache update
 */
export function useTriggerSQLReview(): UseTriggerSQLReviewResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const triggerSQLReview = useCallback(async (id: string): Promise<boolean> => {
    try {
      setIsLoading(true);
      setError(null);
      await changePlanService.triggerSQLReview(id);

      // Invalidate all change plans caches using global mutate with filter function
      await globalMutate(
        (key) => typeof key === 'string' && key.startsWith('/api/change-plans'),
        undefined,
        { revalidate: true },
      );

      showSuccessToast('SQL review triggered successfully');
      return true;
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to trigger SQL review';
      setError(errorMsg);
      console.error('Failed to trigger SQL review:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return {
    triggerSQLReview,
    isLoading,
    error,
  };
}

interface UseApproveChangePlanResult {
  approveChangePlan: (id: string, data?: ApproveChangePlanRequest) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for approving a change plan with automatic cache update
 */
export function useApproveChangePlan(): UseApproveChangePlanResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const approveChangePlan = useCallback(
    async (id: string, data?: ApproveChangePlanRequest): Promise<boolean> => {
      try {
        setIsLoading(true);
        setError(null);
        await changePlanService.approveChangePlan(id, data);

        // Invalidate all change plans caches using global mutate with filter function
        await globalMutate(
          (key) => typeof key === 'string' && key.startsWith('/api/change-plans'),
          undefined,
          { revalidate: true },
        );

        showSuccessToast('Change plan approved successfully');
        return true;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Failed to approve change plan';
        setError(errorMsg);
        console.error('Failed to approve change plan:', err);
        return false;
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  return {
    approveChangePlan,
    isLoading,
    error,
  };
}

interface UseCloseChangePlanResult {
  closeChangePlan: (id: string, data?: CloseChangePlanRequest) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for closing/rejecting a change plan with automatic cache update
 */
export function useCloseChangePlan(): UseCloseChangePlanResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const closeChangePlan = useCallback(
    async (id: string, data?: CloseChangePlanRequest): Promise<boolean> => {
      try {
        setIsLoading(true);
        setError(null);
        await changePlanService.closeChangePlan(id, data);

        // Invalidate all change plans caches using global mutate with filter function
        await globalMutate(
          (key) => typeof key === 'string' && key.startsWith('/api/change-plans'),
          undefined,
          { revalidate: true },
        );

        showSuccessToast('Change plan closed successfully');
        return true;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Failed to close change plan';
        setError(errorMsg);
        console.error('Failed to close change plan:', err);
        return false;
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  return {
    closeChangePlan,
    isLoading,
    error,
  };
}

interface UseExecuteChangePlanResult {
  executeChangePlan: (id: string) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for executing a change plan with automatic cache update
 */
export function useExecuteChangePlan(): UseExecuteChangePlanResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const executeChangePlan = useCallback(async (id: string): Promise<boolean> => {
    try {
      setIsLoading(true);
      setError(null);
      await changePlanService.executeChangePlan(id);

      // Invalidate all change plans caches using global mutate with filter function
      await globalMutate(
        (key) => typeof key === 'string' && key.startsWith('/api/change-plans'),
        undefined,
        { revalidate: true },
      );

      showSuccessToast('Change plan executed successfully');
      return true;
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to execute change plan';
      setError(errorMsg);
      console.error('Failed to execute change plan:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return {
    executeChangePlan,
    isLoading,
    error,
  };
}

/**
 * SWR fetcher function for execution log history
 */
const fetchExecutionLogHistory = async (id: string) => {
  return await changePlanService.getExecutionLogHistory(id);
};

interface UseExecutionLogsSWROptions {
  enabled?: boolean;
}

interface UseExecutionLogsSWRResult {
  executionLogs: ExecutionLogHistoryResponse | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching execution log history for a change plan with SWR
 */
export function useExecutionLogsSWR(
  changePlanId: string | undefined,
  options: UseExecutionLogsSWROptions = {},
): UseExecutionLogsSWRResult {
  const { enabled = true } = options;

  const shouldFetch = enabled && changePlanId;
  const {
    data: executionLogs,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<ExecutionLogHistoryResponse>(
    shouldFetch ? `/api/change-plans/${changePlanId}/execution-logs` : null,
    shouldFetch && changePlanId ? () => fetchExecutionLogHistory(changePlanId) : null,
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
      : 'Failed to load execution logs'
    : null;

  return {
    executionLogs: executionLogs || null,
    isLoading,
    error,
    refetch,
  };
}
