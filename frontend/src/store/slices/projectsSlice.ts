import type { StateCreator } from 'zustand';
import type { Project } from '@/lib/api/types';
import { projectsService } from '@/lib/api/services/projects';

export interface ProjectsSlice {
  // State
  projects: Project[];
  projectsLoading: boolean;
  projectsError: string | null;
  projectsLastFetchedAt: number | null;

  // Actions
  fetchProjects: () => Promise<void>;
  setProjects: (projects: Project[]) => void;
  addProject: (project: Project) => void;
  updateProject: (projectId: string, updates: Partial<Project>) => void;
  removeProject: (projectId: string) => void;
  clearProjects: () => void;
  getProjectById: (projectId: string) => Project | undefined;
}

const CACHE_TTL = 5 * 60 * 1000; // 5 minutes cache TTL

export const createProjectsSlice: StateCreator<ProjectsSlice, [], [], ProjectsSlice> = (
  set,
  get,
) => ({
  // Initial state
  projects: [],
  projectsLoading: false,
  projectsError: null,
  projectsLastFetchedAt: null,

  // Fetch projects from API with caching
  fetchProjects: async () => {
    const state = get();

    // Check if data is still fresh
    if (
      state.projectsLastFetchedAt &&
      Date.now() - state.projectsLastFetchedAt < CACHE_TTL &&
      state.projects.length > 0
    ) {
      // Data is fresh, no need to fetch
      return;
    }

    // Prevent duplicate requests
    if (state.projectsLoading) {
      return;
    }

    set({ projectsLoading: true, projectsError: null });

    try {
      const response = await projectsService.listProjects({ limit: 100 }); // Standardized limit for cache consistency
      set({
        projects: response.items,
        projectsLoading: false,
        projectsError: null,
        projectsLastFetchedAt: Date.now(),
      });
    } catch (error) {
      set({
        projectsLoading: false,
        projectsError: error instanceof Error ? error.message : 'Failed to fetch projects',
      });
    }
  },

  // Set projects directly (useful for initial load or refresh)
  setProjects: (projects) =>
    set({
      projects,
      projectsLastFetchedAt: Date.now(),
    }),

  // Add a new project to the cache
  addProject: (project) =>
    set((state) => ({
      projects: [...state.projects, project],
      projectsLastFetchedAt: Date.now(),
    })),

  // Update an existing project in the cache
  updateProject: (projectId, updates) =>
    set((state) => ({
      projects: state.projects.map((project) =>
        project.id === projectId ? { ...project, ...updates } : project,
      ),
      projectsLastFetchedAt: Date.now(),
    })),

  // Remove a project from the cache
  removeProject: (projectId) =>
    set((state) => ({
      projects: state.projects.filter((project) => project.id !== projectId),
      projectsLastFetchedAt: Date.now(),
    })),

  // Clear all projects
  clearProjects: () =>
    set({
      projects: [],
      projectsLoading: false,
      projectsError: null,
      projectsLastFetchedAt: null,
    }),

  // Helper to get project by ID
  getProjectById: (projectId) => {
    const state = get();
    return state.projects.find((project) => project.id === projectId);
  },
});
