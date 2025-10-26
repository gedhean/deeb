import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import { createOrganizationSlice, type OrganizationSlice } from './slices/organizationSlice';
import { createProjectSlice, type ProjectSlice } from './slices/projectSlice';

// Combined store type
export type AppStore = OrganizationSlice & ProjectSlice;

// Create the store with combined slices
// Following Zustand best practices: https://github.com/pmndrs/zustand/blob/main/docs/guides/slices-pattern.md
export const useAppStore = create<AppStore>()(
  devtools(
    persist(
      (...a) => ({
        ...createOrganizationSlice(...a),
        ...createProjectSlice(...a),
      }),
      {
        name: 'deeb-app-store',
        // Only persist specific fields to avoid storing sensitive data
        partialize: (state) => ({
          currentProject: state.currentProject,
          organization: state.organization,
        }),
      },
    ),
    {
      name: 'DeebAppStore',
    },
  ),
);

// Export individual selectors for better performance
// Following Flux-inspired practices: https://github.com/pmndrs/zustand/blob/main/docs/guides/flux-inspired-practice.md
export const useOrganization = () => useAppStore((state) => state.organization);
export const useSetOrganization = () => useAppStore((state) => state.setOrganization);
export const useClearOrganization = () => useAppStore((state) => state.clearOrganization);

export const useCurrentProject = () => useAppStore((state) => state.currentProject);
export const useSetCurrentProject = () => useAppStore((state) => state.setCurrentProject);
export const useClearCurrentProject = () => useAppStore((state) => state.clearCurrentProject);

// Export a function to clear all state (useful for logout)
export const useClearAllState = () => {
  const clearOrganization = useClearOrganization();
  const clearCurrentProject = useClearCurrentProject();

  return () => {
    clearOrganization();
    clearCurrentProject();
  };
};
