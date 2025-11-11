import { describe, it, expect, beforeEach } from 'vitest';
import { create, type StoreApi, type UseBoundStore } from 'zustand';
import { createAuthSlice, type AuthSlice } from '@/store/slices/authSlice';

const mockUser = {
  id: '1',
  email: 'test@example.com',
  organization_role: 'OrganizationAdmin' as const,
  is_active: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

describe('authSlice', () => {
  let useStore: UseBoundStore<StoreApi<AuthSlice>>;

  beforeEach(() => {
    useStore = create(createAuthSlice);
  });

  it('should initialize with null user and false isAuthenticated', () => {
    const state = useStore.getState();
    expect(state.user).toBeNull();
    expect(state.isAuthenticated).toBe(false);
  });

  it('should login user', () => {
    const { login } = useStore.getState();
    login(mockUser);

    const state = useStore.getState();
    expect(state.user).toEqual(mockUser);
    expect(state.isAuthenticated).toBe(true);
  });

  it('should logout user', () => {
    const { login, logout } = useStore.getState();
    login(mockUser);
    expect(useStore.getState().isAuthenticated).toBe(true);

    logout();
    const state = useStore.getState();
    expect(state.user).toBeNull();
    expect(state.isAuthenticated).toBe(false);
  });

  it('should set user', () => {
    const { setUser } = useStore.getState();
    setUser(mockUser);

    const state = useStore.getState();
    expect(state.user).toEqual(mockUser);
    expect(state.isAuthenticated).toBe(true);
  });

  it('should set user to null', () => {
    const { setUser } = useStore.getState();
    setUser(mockUser);
    setUser(null);

    const state = useStore.getState();
    expect(state.user).toBeNull();
    expect(state.isAuthenticated).toBe(false);
  });
});
