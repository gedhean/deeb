import { useState, useCallback, useEffect } from 'react';
import { policiesService } from '@/lib/api/services/policies';
import type { Policy, CreatePolicyRequest, UpdatePolicyRequest } from '@/lib/api/types';
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

/**
 * Hook for fetching policies list with optional type filter
 */
export function usePolicies(options: UsePoliciesOptions = {}): UsePoliciesResult {
  const { enabled = true, limit, offset, type } = options;
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchPolicies = useCallback(async () => {
    if (!enabled) return;

    try {
      setIsLoading(true);
      setError(null);
      const response = await policiesService.listPolicies({ limit, offset, type });
      setPolicies(response.items);
      setTotal(response.total);
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to load policies';
      setError(errorMsg);
      console.error('Failed to load policies:', err);
    } finally {
      setIsLoading(false);
    }
  }, [enabled, limit, offset, type]);

  useEffect(() => {
    fetchPolicies();
  }, [fetchPolicies]);

  return {
    policies,
    total,
    isLoading,
    error,
    refetch: fetchPolicies,
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
 * Hook for creating a policy
 */
export function useCreatePolicy(): UseCreatePolicyResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createPolicy = useCallback(async (data: CreatePolicyRequest): Promise<Policy | null> => {
    try {
      setIsLoading(true);
      setError(null);
      const result = await policiesService.createPolicy(data);
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
 * Hook for updating a policy
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
 * Hook for deleting a policy
 */
export function useDeletePolicy(): UseDeletePolicyResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const deletePolicy = useCallback(async (id: string): Promise<boolean> => {
    try {
      setIsLoading(true);
      setError(null);
      await policiesService.deletePolicy(id);
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
