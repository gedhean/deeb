import { apiClient } from '../client';
import type {
  CreatePolicyRequest,
  ListResponse,
  Policy,
  SQLReviewRulesResponse,
  UpdatePolicyRequest,
} from '../types';

export const policiesService = {
  /**
   * List all policies
   * Requires Organization Admin role
   * @param type Optional filter by type (e.g., 'SQL_REVIEW')
   */
  async listPolicies(
    params?: Record<string, string | number | undefined>,
  ): Promise<ListResponse<Policy>> {
    const queryParams = new URLSearchParams();
    for (const [key, value] of Object.entries(params ?? {})) {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value));
      }
    }

    const queryString = queryParams.toString();
    const url = `/api/v1/policies${queryString ? `?${queryString}` : ''}`;

    const response = await apiClient.get<{
      policies: Policy[];
      total: number;
      limit: number;
      offset: number;
    }>(url);

    return {
      items: response.policies,
      total: response.total,
      page: Math.floor((response.offset || 0) / (response.limit || 100)) + 1,
      page_size: response.limit || 100,
    };
  },

  /**
   * Get a policy by ID
   */
  async getPolicy(id: string): Promise<Policy> {
    const response = await apiClient.get<Policy>(`/api/v1/policies/${id}`);
    return response;
  },

  /**
   * Create a new policy
   * Requires Organization Admin role
   */
  async createPolicy(data: CreatePolicyRequest): Promise<Policy> {
    const response = await apiClient.post<Policy>('/api/v1/policies', data);
    return response;
  },

  /**
   * Update a policy
   * Requires Organization Admin role
   */
  async updatePolicy(id: string, data: UpdatePolicyRequest): Promise<Policy> {
    const response = await apiClient.put<Policy>(`/api/v1/policies/${id}`, data);
    return response;
  },

  /**
   * Delete a policy
   * Requires Organization Admin role
   */
  async deletePolicy(id: string): Promise<void> {
    await apiClient.delete(`/api/v1/policies/${id}`);
  },

  /**
   * Get available SQL review rules
   * Returns rules organized by engine and category
   */
  async getSQLReviewRules(): Promise<SQLReviewRulesResponse> {
    const response = await apiClient.get<SQLReviewRulesResponse>(
      '/api/v1/policies/sql-review/rules',
    );
    return response;
  },
};
