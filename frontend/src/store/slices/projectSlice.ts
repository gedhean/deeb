import type { StateCreator } from 'zustand';
import type { Project } from '@/lib/api/types';

export interface ProjectSlice {
  currentProject: Project | null;
  setCurrentProject: (project: Project | null) => void;
  clearCurrentProject: () => void;
}

export const createProjectSlice: StateCreator<ProjectSlice, [], [], ProjectSlice> = (set) => ({
  currentProject: null,
  setCurrentProject: (project) =>
    set({
      currentProject: project,
    }),
  clearCurrentProject: () =>
    set({
      currentProject: null,
    }),
});
