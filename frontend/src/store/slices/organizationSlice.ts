import type { StateCreator } from 'zustand';

// Organization type - to be extended when backend Organization entity is added
export interface Organization {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface OrganizationSlice {
  organization: Organization | null;
  setOrganization: (organization: Organization | null) => void;
  clearOrganization: () => void;
}

export const createOrganizationSlice: StateCreator<OrganizationSlice, [], [], OrganizationSlice> = (
  set,
) => ({
  organization: null,
  setOrganization: (organization) =>
    set({
      organization,
    }),
  clearOrganization: () =>
    set({
      organization: null,
    }),
});
