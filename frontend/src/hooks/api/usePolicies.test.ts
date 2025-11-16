import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { SWRConfig } from 'swr';
import { usePolicies } from '@/hooks/api/usePoliciesSWR';
import { policiesService } from '@/lib/api/services/policies';
import type { Policy } from '@/lib/api/types';

// Mock the service
vi.mock('@/lib/api/services/policies', () => ({
  policiesService: {
    listPolicies: vi.fn(),
  },
}));

const mockPolicies: Policy[] = [
  {
    id: '1',
    name: 'Test Policy',
    type: 'SQL_REVIEW',
    policy_config: { rules: {} },
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

// Helper to wrap hooks with SWRConfig for testing
const wrapper = ({ children }: { children: React.ReactNode }) =>
  React.createElement(
    SWRConfig,
    {
      value: {
        dedupingInterval: 0, // Disable deduplication in tests
        provider: () => new Map(), // Use isolated cache for each test
        shouldRetryOnError: false, // Disable retries in tests
      },
    },
    children,
  );

describe('usePolicies', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should fetch policies on mount', async () => {
    vi.mocked(policiesService.listPolicies).mockResolvedValue({
      items: mockPolicies,
      total: 1,
      page: 1,
      page_size: 100,
    });

    const { result } = renderHook(() => usePolicies(), { wrapper });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.policies).toEqual([]);

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.policies).toEqual(mockPolicies);
    expect(result.current.total).toBe(1);
    expect(policiesService.listPolicies).toHaveBeenCalledTimes(1);
  });

  it('should support type filter', async () => {
    vi.mocked(policiesService.listPolicies).mockResolvedValue({
      items: mockPolicies,
      total: 1,
      page: 1,
      page_size: 100,
    });

    renderHook(() => usePolicies({ type: 'SQL_REVIEW' }), { wrapper });

    await waitFor(() => {
      expect(policiesService.listPolicies).toHaveBeenCalledWith({
        type: 'SQL_REVIEW',
        limit: undefined,
        offset: undefined,
      });
    });
  });

  it('should handle errors', async () => {
    const error = new Error('Failed to fetch');
    vi.mocked(policiesService.listPolicies).mockRejectedValue(error);

    const { result } = renderHook(() => usePolicies(), { wrapper });

    await waitFor(
      () => {
        expect(result.current.isLoading).toBe(false);
      },
      { timeout: 3000 },
    );

    // Wait for error to be set
    await waitFor(
      () => {
        expect(result.current.error).not.toBeNull();
      },
      { timeout: 3000 },
    );

    // The error message comes from the Error instance
    expect(result.current.error).toBe('Failed to fetch');
    expect(result.current.policies).toEqual([]);
  });

  it('should support refetch', async () => {
    vi.mocked(policiesService.listPolicies).mockResolvedValue({
      items: mockPolicies,
      total: 1,
      page: 1,
      page_size: 100,
    });

    const { result } = renderHook(() => usePolicies(), { wrapper });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    vi.mocked(policiesService.listPolicies).mockResolvedValue({
      items: [...mockPolicies, { ...mockPolicies[0], id: '2' }],
      total: 2,
      page: 1,
      page_size: 100,
    });

    await waitFor(async () => {
      await result.current.refetch();
    });

    await waitFor(() => {
      expect(result.current.policies).toHaveLength(2);
    });

    expect(policiesService.listPolicies).toHaveBeenCalledTimes(2);
  });
});
