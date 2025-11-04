import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import { createOrganizationSlice, type OrganizationSlice } from './slices/organizationSlice';
import { createProjectSlice, type ProjectSlice } from './slices/projectSlice';
import { createAuthSlice, type AuthSlice } from './slices/authSlice';

// Combined store type
export type AppStore = OrganizationSlice & ProjectSlice & AuthSlice;

// Create the store with combined slices
// Following Zustand best practices: https://github.com/pmndrs/zustand/blob/main/docs/guides/slices-pattern.md
export const useAppStore = create<AppStore>()(
  devtools(
    persist(
      (...a) => ({
        ...createOrganizationSlice(...a),
        ...createProjectSlice(...a),
        ...createAuthSlice(...a),
      }),
      {
        name: 'deeb-app-store',
        // Only persist specific fields to avoid storing sensitive data
        // Note: Auth state (user) is NOT persisted - tokens are in HttpOnly cookies
        partialize: (state) => ({
          currentProject: state.currentProject,
          organization: state.organization,
          // Explicitly exclude auth state from persistence
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

// Auth selectors
export const useUser = () => useAppStore((state) => state.user);
export const useIsAuthenticated = () => useAppStore((state) => state.isAuthenticated);
export const useLogin = () => useAppStore((state) => state.login);
export const useLogout = () => useAppStore((state) => state.logout);
export const useSetUser = () => useAppStore((state) => state.setUser);

// Export a function to clear all state (useful for logout)
export const useClearAllState = () => {
  const clearOrganization = useClearOrganization();
  const clearCurrentProject = useClearCurrentProject();
  const logout = useLogout();

  return () => {
    clearOrganization();
    clearCurrentProject();
    logout();
  };
};
