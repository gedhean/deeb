import { describe, it, expect, beforeEach } from 'vitest';
import { create } from 'zustand';
import { createProjectSlice, type ProjectSlice } from '@/store/slices/projectSlice';

const mockProject = {
  id: '1',
  name: 'Test Project',
  description: 'Test Description',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

describe('projectSlice', () => {
  let useStore: ReturnType<typeof create<ProjectSlice>>;

  beforeEach(() => {
    useStore = create<ProjectSlice>()(createProjectSlice);
  });

  it('should initialize with null current project', () => {
    const state = useStore.getState();
    expect(state.currentProject).toBeNull();
  });

  it('should set current project', () => {
    const { setCurrentProject } = useStore.getState();
    setCurrentProject(mockProject);

    expect(useStore.getState().currentProject).toEqual(mockProject);
  });

  it('should clear current project', () => {
    const { setCurrentProject, clearCurrentProject } = useStore.getState();
    setCurrentProject(mockProject);
    expect(useStore.getState().currentProject).toEqual(mockProject);

    clearCurrentProject();
    expect(useStore.getState().currentProject).toBeNull();
  });

  it('should set current project to null', () => {
    const { setCurrentProject } = useStore.getState();
    setCurrentProject(mockProject);
    setCurrentProject(null);

    expect(useStore.getState().currentProject).toBeNull();
  });
});
