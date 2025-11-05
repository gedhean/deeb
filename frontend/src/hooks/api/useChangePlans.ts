import { useState, useCallback, useEffect } from 'react';
import { changePlanService } from '@/lib/api/services/changePlans';
import type { ChangePlan, CreateChangePlanRequest, ListChangePlansParams } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';

interface UseChangePlansOptions {
  enabled?: boolean;
  params?: ListChangePlansParams;
}

interface UseChangePlansResult {
  changePlans: ChangePlan[];
  total: number;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching change plans list
 */
export function useChangePlans(options: UseChangePlansOptions = {}): UseChangePlansResult {
  const { enabled = true, params } = options;
  const [changePlans, setChangePlans] = useState<ChangePlan[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchChangePlans = useCallback(async () => {
    if (!enabled) return;

    try {
      setIsLoading(true);
      setError(null);
      const response = await changePlanService.listChangePlans(params);
      setChangePlans(response.change_plans || []);
      setTotal(response.total || 0);
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to load change plans';
      setError(errorMsg);
      console.error('Failed to load change plans:', err);
    } finally {
      setIsLoading(false);
    }
  }, [enabled, params]);

  useEffect(() => {
    fetchChangePlans();
  }, [fetchChangePlans]);

  return {
    changePlans,
    total,
    isLoading,
    error,
    refetch: fetchChangePlans,
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
 * Hook for fetching a single change plan by ID
 */
export function useChangePlan(
  id: string | undefined,
  options: UseChangePlanOptions = {},
): UseChangePlanResult {
  const { enabled = true } = options;
  const [changePlan, setChangePlan] = useState<ChangePlan | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchChangePlan = useCallback(async () => {
    if (!enabled || !id) return;

    try {
      setIsLoading(true);
      setError(null);
      const data = await changePlanService.getChangePlan(id);
      setChangePlan(data);
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to load change plan';
      setError(errorMsg);
      console.error('Failed to load change plan:', err);
    } finally {
      setIsLoading(false);
    }
  }, [enabled, id]);

  useEffect(() => {
    fetchChangePlan();
  }, [fetchChangePlan]);

  return {
    changePlan,
    isLoading,
    error,
    refetch: fetchChangePlan,
  };
}

interface UseCreateChangePlanResult {
  createChangePlan: (data: CreateChangePlanRequest) => Promise<ChangePlan | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for creating a change plan
 */
export function useCreateChangePlan(): UseCreateChangePlanResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createChangePlan = useCallback(
    async (data: CreateChangePlanRequest): Promise<ChangePlan | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await changePlanService.createChangePlan(data);
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

interface UseApproveChangePlanResult {
  approveChangePlan: (id: string, data?: { comments?: string }) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for approving a change plan
 */
export function useApproveChangePlan(): UseApproveChangePlanResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const approveChangePlan = useCallback(
    async (id: string, data?: { comments?: string }): Promise<boolean> => {
      try {
        setIsLoading(true);
        setError(null);
        await changePlanService.approveChangePlan(id, data);
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
  closeChangePlan: (id: string, data?: { comments?: string }) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for closing/rejecting a change plan
 */
export function useCloseChangePlan(): UseCloseChangePlanResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const closeChangePlan = useCallback(
    async (id: string, data?: { comments?: string }): Promise<boolean> => {
      try {
        setIsLoading(true);
        setError(null);
        await changePlanService.closeChangePlan(id, data);
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
 * Hook for executing a change plan
 */
export function useExecuteChangePlan(): UseExecuteChangePlanResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const executeChangePlan = useCallback(async (id: string): Promise<boolean> => {
    try {
      setIsLoading(true);
      setError(null);
      await changePlanService.executeChangePlan(id);
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
