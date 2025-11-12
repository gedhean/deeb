import { apiClient } from '../client';
import type {
  CreateDatabaseRequest,
  ListResponse,
  Database,
  UpdateDatabaseRequest,
  ChangePlan,
} from '../types';

export const databasesService = {
  /**
   * List all databases
   * Requires authentication and database read permission
   */
  async listDatabases(params?: {
    limit?: number;
    offset?: number;
    project_id?: string;
    environment_id?: string;
    database_instance_id?: string;
  }): Promise<ListResponse<Database>> {
    const queryParams = new URLSearchParams();
    if (params?.limit) {
      queryParams.append('limit', params.limit.toString());
    }
    if (params?.offset) {
      queryParams.append('offset', params.offset.toString());
    }
    if (params?.project_id) {
      queryParams.append('project_id', params.project_id);
    }
    if (params?.environment_id) {
      queryParams.append('environment_id', params.environment_id);
    }
    if (params?.database_instance_id) {
      queryParams.append('database_instance_id', params.database_instance_id);
    }

    const queryString = queryParams.toString();
    const url = `/api/v1/databases${queryString ? `?${queryString}` : ''}`;

    const response = await apiClient.get<{
      databases: Database[];
      count: number;
      limit: number;
      offset: number;
    }>(url);

    return {
      items: response.databases,
      total: response.count,
      page: Math.floor((response.offset || 0) / (response.limit || 100)) + 1,
      page_size: response.limit || 100,
    };
  },

  /**
   * Get a database by ID
   */
  async getDatabase(id: string): Promise<Database> {
    const response = await apiClient.get<Database>(`/api/v1/databases/${id}`);
    return response;
  },

  /**
   * Create a new database
   * Requires Organization Admin role
   */
  async createDatabase(data: CreateDatabaseRequest): Promise<Database> {
    const response = await apiClient.post<Database>('/api/v1/databases', data);
    return response;
  },

  /**
   * Update a database
   * Requires Organization Admin role
   */
  async updateDatabase(id: string, data: UpdateDatabaseRequest): Promise<Database> {
    const response = await apiClient.put<Database>(`/api/v1/databases/${id}`, data);
    return response;
  },

  /**
   * Delete a database
   * Requires Organization Admin role
   */
  async deleteDatabase(id: string): Promise<void> {
    await apiClient.delete(`/api/v1/databases/${id}`);
  },

  /**
   * Create a database for a specific database instance
   * If the database exists physically on the instance, creates an Active record
   * If the database doesn't exist, creates a Pending record and a Change Plan
   * Requires Organization Admin role
   */
  async createDatabaseForInstance(
    instanceId: string,
    data: CreateDatabaseRequest,
  ): Promise<Database | ChangePlan> {
    const response = await apiClient.post<Database | ChangePlan>(
      `/api/v1/database-instances/${instanceId}/databases`,
      data,
    );
    return response;
  },
};
