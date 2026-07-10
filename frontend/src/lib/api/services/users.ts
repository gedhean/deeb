import { apiClient } from '../client';
import type { CreateUserRequest, ListResponse, User, UpdateUserRequest } from '../types';

export const usersService = {
  /**
   * List all users
   * Requires Organization Admin role
   */
  async listUsers(
    params?: Record<string, string | number | undefined>,
  ): Promise<ListResponse<User>> {
    const queryParams = new URLSearchParams();
    for (const [key, value] of Object.entries(params ?? {})) {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value));
      }
    }

    const queryString = queryParams.toString();
    const url = `/api/v1/users${queryString ? `?${queryString}` : ''}`;

    const response = await apiClient.get<{
      users: User[];
      total: number;
      limit: number;
      offset: number;
    }>(url);

    return {
      items: response.users,
      total: response.total,
      page: Math.floor((response.offset || 0) / (response.limit || 100)) + 1,
      page_size: response.limit || 100,
    };
  },

  /**
   * Get a user by ID
   * Requires Organization Admin role
   */
  async getUser(id: string): Promise<User> {
    const response = await apiClient.get<User>(`/api/v1/users/${id}`);
    return response;
  },

  /**
   * Create a new user
   * Requires Organization Admin role
   */
  async createUser(data: CreateUserRequest): Promise<User> {
    const response = await apiClient.post<User>('/api/v1/users', data);
    return response;
  },

  /**
   * Update a user
   * Requires Organization Admin role
   */
  async updateUser(id: string, data: UpdateUserRequest): Promise<User> {
    const response = await apiClient.put<User>(`/api/v1/users/${id}`, data);
    return response;
  },

  /**
   * Delete (deactivate) a user
   * Requires Organization Admin role
   */
  async deleteUser(id: string): Promise<void> {
    await apiClient.delete(`/api/v1/users/${id}`);
  },
};
