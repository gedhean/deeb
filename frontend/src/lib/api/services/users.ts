import { apiClient } from '../client';
import type { CreateUserRequest, ListResponse, User, UpdateUserRequest } from '../types';

export const usersService = {
  /**
   * List all users
   * Requires Organization Admin role
   */
  async listUsers(params?: { limit?: number; offset?: number }): Promise<ListResponse<User>> {
    const queryParams = new URLSearchParams();
    if (params?.limit) {
      queryParams.append('limit', params.limit.toString());
    }
    if (params?.offset) {
      queryParams.append('offset', params.offset.toString());
    }

    const queryString = queryParams.toString();
    const url = `/api/v1/users${queryString ? `?${queryString}` : ''}`;

    const response = await apiClient.get<{
      users: User[];
      count: number;
      limit: number;
      offset: number;
    }>(url);

    return {
      items: response.users,
      total: response.count,
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
