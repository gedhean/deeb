import { apiClient } from '../client';
import type { CreateProjectRequest, ListResponse, Project, UpdateProjectRequest } from '../types';

export const projectsService = {
  /**
   * List all projects
   * Requires Organization Admin role
   */
  async listProjects(params?: { limit?: number; offset?: number }): Promise<ListResponse<Project>> {
    const queryParams = new URLSearchParams();
    if (params?.limit) {
      queryParams.append('limit', params.limit.toString());
    }
    if (params?.offset) {
      queryParams.append('offset', params.offset.toString());
    }

    const queryString = queryParams.toString();
    const url = `/api/v1/projects${queryString ? `?${queryString}` : ''}`;

    const response = await apiClient.get<{
      projects: Project[];
      count: number;
      limit: number;
      offset: number;
    }>(url);

    return {
      items: response.projects,
      total: response.count,
      page: Math.floor((response.offset || 0) / (response.limit || 100)) + 1,
      page_size: response.limit || 100,
    };
  },

  /**
   * Get a project by ID
   */
  async getProject(id: string): Promise<Project> {
    const response = await apiClient.get<Project>(`/api/v1/projects/${id}`);
    return response;
  },

  /**
   * Create a new project
   * Requires Organization Admin or DBA role
   */
  async createProject(data: CreateProjectRequest): Promise<Project> {
    const response = await apiClient.post<Project>('/api/v1/projects', data);
    return response;
  },

  /**
   * Update a project
   * Requires Organization Admin role
   */
  async updateProject(id: string, data: UpdateProjectRequest): Promise<Project> {
    const response = await apiClient.put<Project>(`/api/v1/projects/${id}`, data);
    return response;
  },

  /**
   * Delete a project
   * Requires Organization Admin role
   */
  async deleteProject(id: string): Promise<void> {
    await apiClient.delete(`/api/v1/projects/${id}`);
  },
};
