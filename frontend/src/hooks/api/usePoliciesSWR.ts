import useSWR, { mutate } from 'swr';
import { useState, useCallback, useMemo, useEffect } from 'react';
import { policiesService } from '@/lib/api/services/policies';
import type {
  Policy,
  CreatePolicyRequest,
  UpdatePolicyRequest,
  SQLReviewRulesResponse,
} from '@/lib/api/types';
import { APIError } from '@/lib/api/types';

interface UsePoliciesOptions {
  enabled?: boolean;
  limit?: number;
  offset?: number;
  type?: 'SQL_REVIEW';
}

interface UsePoliciesResult {
  policies: Policy[];
  total: number;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

interface ListPoliciesResponse {
  items: Policy[];
  total: number;
}

/**
 * SWR fetcher function for policies list
 */
const fetchPolicies = async (params: {
  limit?: number;
  offset?: number;
  type?: 'SQL_REVIEW';
}): Promise<ListPoliciesResponse> => {
  return await policiesService.listPolicies(params);
};

/**
 * Hook for fetching policies list with optional type filter
 * Uses SWR for automatic request deduplication and caching
 */
export function usePolicies(options: UsePoliciesOptions = {}): UsePoliciesResult {
  const { enabled = true, limit, offset, type } = options;

  // Create a stable cache key from params
  const cacheKey = useMemo(() => {
    if (!enabled) return null;
    const params = { limit, offset, type };
    // Sort keys to ensure consistent cache key
    const sortedParams = Object.keys(params)
      .sort()
      .reduce(
        (acc, key) => {
          const value = params[key as keyof typeof params];
          if (value !== undefined) {
            acc[key] = value;
          }
          return acc;
        },
        {} as Record<string, unknown>,
      );
    return `/api/policies?${JSON.stringify(sortedParams)}`;
  }, [enabled, limit, offset, type]);

  const {
    data,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<ListPoliciesResponse>(
    cacheKey,
    cacheKey ? () => fetchPolicies({ limit, offset, type }) : null,
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
      : 'Failed to load policies'
    : null;

  return {
    policies: data?.items || [],
    total: data?.total || 0,
    isLoading,
    error,
    refetch,
  };
}

interface UsePolicyOptions {
  enabled?: boolean;
}

interface UsePolicyResult {
  policy: Policy | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching a single policy by ID
 */
export function usePolicy(id: string | undefined, options: UsePolicyOptions = {}): UsePolicyResult {
  const { enabled = true } = options;
  const [policy, setPolicy] = useState<Policy | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchPolicy = useCallback(async () => {
    if (!enabled || !id) return;

    try {
      setIsLoading(true);
      setError(null);
      const data = await policiesService.getPolicy(id);
      setPolicy(data);
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to load policy';
      setError(errorMsg);
      console.error('Failed to load policy:', err);
    } finally {
      setIsLoading(false);
    }
  }, [enabled, id]);

  useEffect(() => {
    fetchPolicy();
  }, [fetchPolicy]);

  return {
    policy,
    isLoading,
    error,
    refetch: fetchPolicy,
  };
}

interface UseCreatePolicyResult {
  createPolicy: (data: CreatePolicyRequest) => Promise<Policy | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for creating a policy with automatic cache invalidation
 */
export function useCreatePolicy(): UseCreatePolicyResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createPolicy = useCallback(async (data: CreatePolicyRequest): Promise<Policy | null> => {
    try {
      setIsLoading(true);
      setError(null);
      const result = await policiesService.createPolicy(data);

      // Invalidate all policies caches using global mutate with filter function
      await mutate(
        (key) => typeof key === 'string' && key.startsWith('/api/policies?'),
        undefined,
        { revalidate: true },
      );

      return result;
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to create policy';
      setError(errorMsg);
      console.error('Failed to create policy:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return {
    createPolicy,
    isLoading,
    error,
  };
}

interface UseUpdatePolicyResult {
  updatePolicy: (id: string, data: UpdatePolicyRequest) => Promise<Policy | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for updating a policy with automatic cache invalidation
 */
export function useUpdatePolicy(): UseUpdatePolicyResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updatePolicy = useCallback(
    async (id: string, data: UpdatePolicyRequest): Promise<Policy | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await policiesService.updatePolicy(id, data);

        // Invalidate all policies caches using global mutate with filter function
        await mutate(
          (key) => typeof key === 'string' && key.startsWith('/api/policies?'),
          undefined,
          { revalidate: true },
        );

        return result;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Failed to update policy';
        setError(errorMsg);
        console.error('Failed to update policy:', err);
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  return {
    updatePolicy,
    isLoading,
    error,
  };
}

interface UseDeletePolicyResult {
  deletePolicy: (id: string) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for deleting a policy with automatic cache invalidation
 */
export function useDeletePolicy(): UseDeletePolicyResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const deletePolicy = useCallback(async (id: string): Promise<boolean> => {
    try {
      setIsLoading(true);
      setError(null);
      await policiesService.deletePolicy(id);

      // Invalidate all policies caches using global mutate with filter function
      await mutate(
        (key) => typeof key === 'string' && key.startsWith('/api/policies?'),
        undefined,
        { revalidate: true },
      );

      return true;
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to delete policy';
      setError(errorMsg);
      console.error('Failed to delete policy:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return {
    deletePolicy,
    isLoading,
    error,
  };
}

interface UseSQLReviewRulesResult {
  rules: SQLReviewRulesResponse | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching SQL review rules with SWR caching
 * Returns available rules organized by engine and category
 */
export function useSQLReviewRules(): UseSQLReviewRulesResult {
  const {
    data: rules,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<SQLReviewRulesResponse>(
    '/api/v1/policies/sql-review/rules',
    () => policiesService.getSQLReviewRules(),
    {
      revalidateIfStale: false,
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
    },
  );

  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  return {
    rules: rules || null,
    isLoading,
    error: swrError
      ? swrError instanceof APIError
        ? swrError.message
        : 'Failed to load SQL review rules'
      : null,
    refetch,
  };
}
