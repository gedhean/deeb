import { describe, it, expect, beforeEach } from 'vitest';
import { create, type StoreApi, type UseBoundStore } from 'zustand';
import { createOrganizationSlice, type OrganizationSlice } from '@/store/slices/organizationSlice';

const mockOrganization = {
  id: '1',
  name: 'Test Organization',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

describe('organizationSlice', () => {
  let useStore: UseBoundStore<StoreApi<OrganizationSlice>>;

  beforeEach(() => {
    useStore = create(createOrganizationSlice);
  });

  it('should initialize with null organization', () => {
    const state = useStore.getState();
    expect(state.organization).toBeNull();
  });

  it('should set organization', () => {
    const { setOrganization } = useStore.getState();
    setOrganization(mockOrganization);

    expect(useStore.getState().organization).toEqual(mockOrganization);
  });

  it('should clear organization', () => {
    const { setOrganization, clearOrganization } = useStore.getState();
    setOrganization(mockOrganization);
    expect(useStore.getState().organization).toEqual(mockOrganization);

    clearOrganization();
    expect(useStore.getState().organization).toBeNull();
  });

  it('should set organization to null', () => {
    const { setOrganization } = useStore.getState();
    setOrganization(mockOrganization);
    setOrganization(null);

    expect(useStore.getState().organization).toBeNull();
  });
});
