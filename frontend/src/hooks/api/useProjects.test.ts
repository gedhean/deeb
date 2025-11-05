import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useProjects } from '@/hooks/api/useProjects';
import { projectsService } from '@/lib/api/services/projects';
import type { Project } from '@/lib/api/types';

// Mock the service
vi.mock('@/lib/api/services/projects', () => ({
  projectsService: {
    listProjects: vi.fn(),
  },
}));

const mockProjects: Project[] = [
  {
    id: '1',
    name: 'Test Project',
    description: 'Test Description',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

describe('useProjects', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should fetch projects on mount', async () => {
    vi.mocked(projectsService.listProjects).mockResolvedValue({
      items: mockProjects,
      total: 1,
      page: 1,
      page_size: 100,
    });

    const { result } = renderHook(() => useProjects());

    expect(result.current.isLoading).toBe(true);
    expect(result.current.projects).toEqual([]);

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.projects).toEqual(mockProjects);
    expect(result.current.total).toBe(1);
    expect(projectsService.listProjects).toHaveBeenCalledTimes(1);
  });

  it('should handle errors', async () => {
    const error = new Error('Failed to fetch');
    vi.mocked(projectsService.listProjects).mockRejectedValue(error);

    const { result } = renderHook(() => useProjects());

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.error).toBe('Failed to load projects');
    expect(result.current.projects).toEqual([]);
  });

  it('should support pagination parameters', async () => {
    vi.mocked(projectsService.listProjects).mockResolvedValue({
      items: mockProjects,
      total: 1,
      page: 1,
      page_size: 10,
    });

    renderHook(() => useProjects({ limit: 10, offset: 0 }));

    await waitFor(() => {
      expect(projectsService.listProjects).toHaveBeenCalledWith({ limit: 10, offset: 0 });
    });
  });

  it('should support refetch', async () => {
    vi.mocked(projectsService.listProjects).mockResolvedValue({
      items: mockProjects,
      total: 1,
      page: 1,
      page_size: 100,
    });

    const { result } = renderHook(() => useProjects());

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    vi.mocked(projectsService.listProjects).mockResolvedValue({
      items: [...mockProjects, { ...mockProjects[0], id: '2' }],
      total: 2,
      page: 1,
      page_size: 100,
    });

    await result.current.refetch();

    await waitFor(() => {
      expect(result.current.projects).toHaveLength(2);
    });

    expect(projectsService.listProjects).toHaveBeenCalledTimes(2);
  });
});
