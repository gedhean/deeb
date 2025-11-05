import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { usePolicies } from '@/hooks/api/usePolicies';
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
    enabled: true,
    config: {},
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

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

    const { result } = renderHook(() => usePolicies());

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

    renderHook(() => usePolicies({ type: 'SQL_REVIEW' }));

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

    const { result } = renderHook(() => usePolicies());

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.error).toBe('Failed to load policies');
    expect(result.current.policies).toEqual([]);
  });

  it('should support refetch', async () => {
    vi.mocked(policiesService.listPolicies).mockResolvedValue({
      items: mockPolicies,
      total: 1,
      page: 1,
      page_size: 100,
    });

    const { result } = renderHook(() => usePolicies());

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    vi.mocked(policiesService.listPolicies).mockResolvedValue({
      items: [...mockPolicies, { ...mockPolicies[0], id: '2' }],
      total: 2,
      page: 1,
      page_size: 100,
    });

    await result.current.refetch();

    await waitFor(() => {
      expect(result.current.policies).toHaveLength(2);
    });

    expect(policiesService.listPolicies).toHaveBeenCalledTimes(2);
  });
});
