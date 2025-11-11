import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useChangePlans } from '@/hooks/api/useChangePlans';
import { changePlanService } from '@/lib/api/services/changePlans';
import type { ChangePlan } from '@/lib/api/types';

// Mock the service
vi.mock('@/lib/api/services/changePlans', () => ({
  changePlanService: {
    listChangePlans: vi.fn(),
  },
}));

const mockChangePlans: ChangePlan[] = [
  {
    id: '1',
    title: 'Test Change Plan',
    description: 'Test description',
    status: 'Draft',
    project_id: '1',
    target_database_id: '1',
    author_id: '1',
    change_type: 'DDL',
    sql_content: 'ALTER TABLE test ADD COLUMN new_column VARCHAR(255);',
    risk_level: 'Low',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

describe('useChangePlans', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should fetch change plans on mount', async () => {
    vi.mocked(changePlanService.listChangePlans).mockResolvedValue({
      change_plans: mockChangePlans,
      total: 1,
    });

    const { result } = renderHook(() => useChangePlans());

    expect(result.current.isLoading).toBe(true);
    expect(result.current.changePlans).toEqual([]);

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.changePlans).toEqual(mockChangePlans);
    expect(result.current.total).toBe(1);
    expect(changePlanService.listChangePlans).toHaveBeenCalledTimes(1);
  });

  it('should handle errors', async () => {
    const error = new Error('Failed to fetch');
    vi.mocked(changePlanService.listChangePlans).mockRejectedValue(error);

    const { result } = renderHook(() => useChangePlans());

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.error).toBe('Failed to load change plans');
    expect(result.current.changePlans).toEqual([]);
  });

  it('should support refetch', async () => {
    vi.mocked(changePlanService.listChangePlans).mockResolvedValue({
      change_plans: mockChangePlans,
      total: 1,
    });

    const { result } = renderHook(() => useChangePlans());

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    vi.mocked(changePlanService.listChangePlans).mockResolvedValue({
      change_plans: [...mockChangePlans, { ...mockChangePlans[0], id: '2' }],
      total: 2,
    });

    await waitFor(async () => {
      await result.current.refetch();
    });

    await waitFor(() => {
      expect(result.current.changePlans).toHaveLength(2);
    });

    expect(changePlanService.listChangePlans).toHaveBeenCalledTimes(2);
  });

  it('should not fetch when enabled is false', () => {
    renderHook(() => useChangePlans({ enabled: false }));

    expect(changePlanService.listChangePlans).not.toHaveBeenCalled();
  });
});
