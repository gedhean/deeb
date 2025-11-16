import type { StateCreator } from 'zustand';
import type { User } from '@/lib/api/types';
import { usersService } from '@/lib/api/services/users';

export interface UsersSlice {
  // State
  users: User[];
  usersLoading: boolean;
  usersError: string | null;
  usersLastFetchedAt: number | null;

  // Actions
  fetchUsers: () => Promise<void>;
  setUsers: (users: User[]) => void;
  addUser: (user: User) => void;
  updateUser: (userId: string, updates: Partial<User>) => void;
  removeUser: (userId: string) => void;
  clearUsers: () => void;
  getUserById: (userId: string) => User | undefined;
}

const CACHE_TTL = 5 * 60 * 1000; // 5 minutes cache TTL

export const createUsersSlice: StateCreator<UsersSlice, [], [], UsersSlice> = (set, get) => ({
  // Initial state
  users: [],
  usersLoading: false,
  usersError: null,
  usersLastFetchedAt: null,

  // Fetch users from API with caching
  fetchUsers: async () => {
    const state = get();

    // Check if data is still fresh
    if (
      state.usersLastFetchedAt &&
      Date.now() - state.usersLastFetchedAt < CACHE_TTL &&
      state.users.length > 0
    ) {
      // Data is fresh, no need to fetch
      return;
    }

    // Prevent duplicate requests
    if (state.usersLoading) {
      return;
    }

    set({ usersLoading: true, usersError: null });

    try {
      const response = await usersService.listUsers({ limit: 1000 }); // Fetch all users
      set({
        users: response.items,
        usersLoading: false,
        usersError: null,
        usersLastFetchedAt: Date.now(),
      });
    } catch (error) {
      set({
        usersLoading: false,
        usersError: error instanceof Error ? error.message : 'Failed to fetch users',
      });
    }
  },

  // Set users directly (useful for initial load or refresh)
  setUsers: (users) =>
    set({
      users,
      usersLastFetchedAt: Date.now(),
    }),

  // Add a new user to the cache
  addUser: (user) =>
    set((state) => ({
      users: [...state.users, user],
      usersLastFetchedAt: Date.now(),
    })),

  // Update an existing user in the cache
  updateUser: (userId, updates) =>
    set((state) => ({
      users: state.users.map((user) => (user.id === userId ? { ...user, ...updates } : user)),
      usersLastFetchedAt: Date.now(),
    })),

  // Remove a user from the cache
  removeUser: (userId) =>
    set((state) => ({
      users: state.users.filter((user) => user.id !== userId),
      usersLastFetchedAt: Date.now(),
    })),

  // Clear all users
  clearUsers: () =>
    set({
      users: [],
      usersLoading: false,
      usersError: null,
      usersLastFetchedAt: null,
    }),

  // Helper to get user by ID
  getUserById: (userId) => {
    const state = get();
    return state.users.find((user) => user.id === userId);
  },
});
