import useSWR, { mutate } from 'swr';
import { useCallback, useState } from 'react';
import { usersService } from '@/lib/api/services/users';
import type { User, CreateUserRequest, UpdateUserRequest } from '@/lib/api/types';
import { APIError } from '@/lib/api/types';
import {
  useUsers as useUsersStore,
  useUsersLoading,
  useUsersError,
  useGetUserById as useGetUserByIdStore,
} from '@/store';

/**
 * SWR fetcher function for users list
 */
const fetchUsers = async () => {
  const response = await usersService.listUsers({ limit: 100 });
  return response.items;
};

interface UseUsersResult {
  users: User[];
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
  getUserById: (userId: string) => User | undefined;
}

/**
 * Hook for fetching users list with SWR caching and Zustand store integration
 *
 * This hook combines SWR for automatic request deduplication and background revalidation
 * with Zustand store for immediate cache availability across components.
 *
 * Features:
 * - Automatic request deduplication (multiple components calling this won't duplicate requests)
 * - Background revalidation on window focus and network reconnect
 * - Fallback to Zustand store while SWR is fetching
 * - Persistent cache across page reloads via Zustand persist middleware
 */
export function useUsers(): UseUsersResult {
  const storeUsers = useUsersStore();
  const storeLoading = useUsersLoading();
  const storeError = useUsersError();
  const getUserByIdFromStore = useGetUserByIdStore();

  // SWR hook for automatic caching and revalidation
  const {
    data: swrUsers,
    error: swrError,
    isLoading: swrLoading,
    mutate,
  } = useSWR<User[]>('/api/users', fetchUsers, {
    // Use store data as fallback while fetching
    fallbackData: storeUsers.length > 0 ? storeUsers : undefined,
    // Revalidate on mount if data is older than 5 minutes
    revalidateIfStale: true,
    // Keep previous data while revalidating
    keepPreviousData: true,
  });

  // Manual refetch function
  const refetch = useCallback(async () => {
    await mutate();
  }, [mutate]);

  // Helper to get user by ID
  const getUserById = useCallback(
    (userId: string) => {
      // Try SWR data first, fall back to store
      const users = swrUsers || storeUsers;
      return users.find((user) => user.id === userId) || getUserByIdFromStore(userId);
    },
    [swrUsers, storeUsers, getUserByIdFromStore],
  );

  // Use SWR data if available, otherwise fall back to store
  const users = swrUsers || storeUsers;
  const isLoading = swrLoading || storeLoading;
  const error = swrError
    ? swrError instanceof Error
      ? swrError.message
      : 'Failed to load users'
    : storeError;

  return {
    users,
    isLoading,
    error,
    refetch,
    getUserById,
  };
}

/**
 * Hook for getting a single user by ID
 *
 * This hook uses the cached users list instead of making a separate API call
 */
export function useUserById(userId: string | undefined): {
  user: User | undefined;
  isLoading: boolean;
  error: string | null;
} {
  const { isLoading, error, getUserById } = useUsers();

  const user = userId ? getUserById(userId) : undefined;

  return {
    user,
    isLoading,
    error,
  };
}

interface UseCreateUserResult {
  createUser: (data: CreateUserRequest) => Promise<User | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for creating a user with automatic cache invalidation
 */
export function useCreateUser(): UseCreateUserResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const createUser = useCallback(async (data: CreateUserRequest): Promise<User | null> => {
    try {
      setIsLoading(true);
      setError(null);
      const result = await usersService.createUser(data);

      // Invalidate users cache using global mutate
      await mutate('/api/users', undefined, { revalidate: true });

      return result;
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to create user';
      setError(errorMsg);
      console.error('Failed to create user:', err);
      return null;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return {
    createUser,
    isLoading,
    error,
  };
}

interface UseUpdateUserResult {
  updateUser: (id: string, data: UpdateUserRequest) => Promise<User | null>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for updating a user with automatic cache invalidation
 */
export function useUpdateUser(): UseUpdateUserResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateUser = useCallback(
    async (id: string, data: UpdateUserRequest): Promise<User | null> => {
      try {
        setIsLoading(true);
        setError(null);
        const result = await usersService.updateUser(id, data);

        // Invalidate users cache using global mutate
        await mutate('/api/users', undefined, { revalidate: true });

        return result;
      } catch (err) {
        const errorMsg = err instanceof APIError ? err.message : 'Failed to update user';
        setError(errorMsg);
        console.error('Failed to update user:', err);
        return null;
      } finally {
        setIsLoading(false);
      }
    },
    [],
  );

  return {
    updateUser,
    isLoading,
    error,
  };
}

interface UseDeleteUserResult {
  deleteUser: (id: string) => Promise<boolean>;
  isLoading: boolean;
  error: string | null;
}

/**
 * Hook for deleting (deactivating) a user with automatic cache invalidation
 */
export function useDeleteUser(): UseDeleteUserResult {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const deleteUser = useCallback(async (id: string): Promise<boolean> => {
    try {
      setIsLoading(true);
      setError(null);
      await usersService.deleteUser(id);

      // Invalidate users cache using global mutate
      await mutate('/api/users', undefined, { revalidate: true });

      return true;
    } catch (err) {
      const errorMsg = err instanceof APIError ? err.message : 'Failed to delete user';
      setError(errorMsg);
      console.error('Failed to delete user:', err);
      return false;
    } finally {
      setIsLoading(false);
    }
  }, []);

  return {
    deleteUser,
    isLoading,
    error,
  };
}
