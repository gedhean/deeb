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
  async listDatabases(
    params?: Record<string, string | number | undefined>,
  ): Promise<ListResponse<Database>> {
    const queryParams = new URLSearchParams();
    for (const [key, value] of Object.entries(params ?? {})) {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value));
      }
    }

    const queryString = queryParams.toString();
    const url = `/api/v1/databases${queryString ? `?${queryString}` : ''}`;

    const response = await apiClient.get<{
      databases: Database[];
      total: number;
      limit: number;
      offset: number;
    }>(url);

    return {
      items: response.databases,
      total: response.total,
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
