import { useState, useCallback, useEffect } from 'react';
import { changePlanService } from '@/lib/api/services/changePlans';
import type { ApprovalHistoryResponse } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';

interface UseApprovalsOptions {
  enabled?: boolean;
}

interface UseApprovalsResult {
  approvals: ApprovalHistoryResponse['approvals'];
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching approval history for a change plan
 */
export function useApprovals(
  changePlanId: string | undefined,
  options: UseApprovalsOptions = {},
): UseApprovalsResult {
  const { enabled = true } = options;
  const [approvals, setApprovals] = useState<ApprovalHistoryResponse['approvals']>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchApprovals = useCallback(async () => {
    if (!enabled || !changePlanId) return;

    try {
      setIsLoading(true);
      setError(null);
      const response = await changePlanService.getApprovalHistory(changePlanId);
      setApprovals(response.approvals || []);
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to load approval history';
      setError(errorMsg);
      console.error('Failed to load approval history:', err);
    } finally {
      setIsLoading(false);
    }
  }, [enabled, changePlanId]);

  useEffect(() => {
    fetchApprovals();
  }, [fetchApprovals]);

  return {
    approvals,
    isLoading,
    error,
    refetch: fetchApprovals,
  };
}
