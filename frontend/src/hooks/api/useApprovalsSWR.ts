import useSWR from 'swr';
import { useCallback } from 'react';
import { changePlanService } from '@/lib/api/services/changePlans';
import type { ApprovalHistoryResponse } from '@/lib/api/types';

/**
 * SWR fetcher function for approvals
 */
const fetchApprovals = async (changePlanId: string) => {
  return await changePlanService.getApprovalHistory(changePlanId);
};

interface UseApprovalsResult {
  approvals: ApprovalHistoryResponse['approvals'];
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching approval history with SWR caching
 *
 * This hook uses SWR for automatic request deduplication and background revalidation.
 */
export function useApprovalsSWR(changePlanId: string | undefined): UseApprovalsResult {
  const shouldFetch = !!changePlanId;

  const {
    data: approvalHistory,
    error: swrError,
    isLoading,
    mutate,
  } = useSWR<ApprovalHistoryResponse>(
    shouldFetch ? `/api/change-plans/${changePlanId}/approvals` : null,
    shouldFetch && changePlanId ? () => fetchApprovals(changePlanId) : null,
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
      : 'Failed to load approvals'
    : null;

  return {
    approvals: approvalHistory?.approvals || [],
    isLoading,
    error,
    refetch,
  };
}
