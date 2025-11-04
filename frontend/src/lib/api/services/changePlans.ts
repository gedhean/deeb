import { apiClient } from '../client';
import type {
  ChangePlan,
  ChangePlanListResponse,
  CreateChangePlanRequest,
  ListChangePlansParams,
} from '../types';

export const changePlanService = {
  /**
   * List change plans with optional filters
   */
  async listChangePlans(params?: ListChangePlansParams): Promise<ChangePlanListResponse> {
    const queryParams = new URLSearchParams();

    if (params?.status) {
      queryParams.append('status', params.status);
    }
    if (params?.project_id) {
      queryParams.append('project_id', params.project_id);
    }
    if (params?.database_id) {
      queryParams.append('database_id', params.database_id);
    }
    if (params?.author_id) {
      queryParams.append('author_id', params.author_id);
    }
    if (params?.limit) {
      queryParams.append('limit', params.limit.toString());
    }
    if (params?.offset) {
      queryParams.append('offset', params.offset.toString());
    }

    const queryString = queryParams.toString();
    const url = `/api/v1/change-plans${queryString ? `?${queryString}` : ''}`;

    const response = await apiClient.get<ChangePlanListResponse>(url);
    return response;
  },

  /**
   * Get a specific change plan by ID
   */
  async getChangePlan(id: string): Promise<ChangePlan> {
    const response = await apiClient.get<ChangePlan>(`/api/v1/change-plans/${id}`);
    return response;
  },

  /**
   * Create a new change plan
   */
  async createChangePlan(data: CreateChangePlanRequest): Promise<ChangePlan> {
    const response = await apiClient.post<ChangePlan>('/api/v1/change-plans', data);
    return response;
  },
};
