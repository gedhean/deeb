import { useState, useCallback, useEffect } from 'react';
import { changePlanService } from '@/lib/api/services/changePlans';
import type { ExecutionLogHistoryResponse } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';

interface UseExecutionOptions {
  enabled?: boolean;
}

interface UseExecutionResult {
  executionLogs: ExecutionLogHistoryResponse['logs'];
  total: number;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching execution log history for a change plan
 */
export function useExecution(
  changePlanId: string | undefined,
  options: UseExecutionOptions = {},
): UseExecutionResult {
  const { enabled = true } = options;
  const [executionLogs, setExecutionLogs] = useState<ExecutionLogHistoryResponse['logs']>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchExecutionLogs = useCallback(async () => {
    if (!enabled || !changePlanId) return;

    try {
      setIsLoading(true);
      setError(null);
      const response = await changePlanService.getExecutionLogHistory(changePlanId);
      setExecutionLogs(response.logs || []);
      setTotal(response.total || 0);
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to load execution logs';
      setError(errorMsg);
      console.error('Failed to load execution logs:', err);
    } finally {
      setIsLoading(false);
    }
  }, [enabled, changePlanId]);

  useEffect(() => {
    fetchExecutionLogs();
  }, [fetchExecutionLogs]);

  return {
    executionLogs,
    total,
    isLoading,
    error,
    refetch: fetchExecutionLogs,
  };
}
