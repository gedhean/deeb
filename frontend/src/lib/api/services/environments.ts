import { apiClient } from '../client';
import type {
  CreateEnvironmentRequest,
  ListResponse,
  Environment,
  UpdateEnvironmentRequest,
} from '../types';

export const environmentsService = {
  /**
   * List all environments
   * Requires Organization Admin role
   */
  async listEnvironments(params?: {
    limit?: number;
    offset?: number;
  }): Promise<ListResponse<Environment>> {
    const queryParams = new URLSearchParams();
    if (params?.limit) {
      queryParams.append('limit', params.limit.toString());
    }
    if (params?.offset) {
      queryParams.append('offset', params.offset.toString());
    }

    const queryString = queryParams.toString();
    const url = `/api/v1/environments${queryString ? `?${queryString}` : ''}`;

    const response = await apiClient.get<{
      environments: Environment[];
      count: number;
      limit: number;
      offset: number;
    }>(url);

    return {
      items: response.environments,
      total: response.count,
      page: Math.floor((response.offset || 0) / (response.limit || 100)) + 1,
      page_size: response.limit || 100,
    };
  },

  /**
   * Get an environment by ID
   */
  async getEnvironment(id: string): Promise<Environment> {
    const response = await apiClient.get<Environment>(`/api/v1/environments/${id}`);
    return response;
  },

  /**
   * Create a new environment
   * Requires Organization Admin role
   */
  async createEnvironment(data: CreateEnvironmentRequest): Promise<Environment> {
    const response = await apiClient.post<Environment>('/api/v1/environments', data);
    return response;
  },

  /**
   * Update an environment
   * Requires Organization Admin role
   */
  async updateEnvironment(id: string, data: UpdateEnvironmentRequest): Promise<Environment> {
    const response = await apiClient.put<Environment>(`/api/v1/environments/${id}`, data);
    return response;
  },

  /**
   * Delete an environment
   * Requires Organization Admin role
   */
  async deleteEnvironment(id: string): Promise<void> {
    await apiClient.delete(`/api/v1/environments/${id}`);
  },
};
