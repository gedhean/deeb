import { apiClient } from '../client';
import type {
  CreateDatabaseInstanceRequest,
  ListResponse,
  DatabaseInstance,
  UpdateDatabaseInstanceRequest,
} from '../types';

export const databaseInstancesService = {
  /**
   * List all database instances
   * Requires authentication and database read permission
   */
  async listDatabaseInstances(params?: {
    limit?: number;
    offset?: number;
  }): Promise<ListResponse<DatabaseInstance>> {
    const queryParams = new URLSearchParams();
    if (params?.limit) {
      queryParams.append('limit', params.limit.toString());
    }
    if (params?.offset) {
      queryParams.append('offset', params.offset.toString());
    }

    const queryString = queryParams.toString();
    const url = `/api/v1/database-instances${queryString ? `?${queryString}` : ''}`;

    const response = await apiClient.get<{
      database_instances: DatabaseInstance[];
      count: number;
      limit: number;
      offset: number;
    }>(url);

    return {
      items: response.database_instances,
      total: response.count,
      page: Math.floor((response.offset || 0) / (response.limit || 100)) + 1,
      page_size: response.limit || 100,
    };
  },

  /**
   * Get a database instance by ID
   */
  async getDatabaseInstance(id: string): Promise<DatabaseInstance> {
    const response = await apiClient.get<DatabaseInstance>(`/api/v1/database-instances/${id}`);
    return response;
  },

  /**
   * Create a new database instance
   * Requires Organization Admin role
   * Automatically tests connection before saving
   */
  async createDatabaseInstance(data: CreateDatabaseInstanceRequest): Promise<DatabaseInstance> {
    const response = await apiClient.post<DatabaseInstance>('/api/v1/database-instances', data);
    return response;
  },

  /**
   * Update a database instance
   * Requires Organization Admin role
   * Automatically tests connection if credentials changed
   */
  async updateDatabaseInstance(
    id: string,
    data: UpdateDatabaseInstanceRequest,
  ): Promise<DatabaseInstance> {
    const response = await apiClient.put<DatabaseInstance>(
      `/api/v1/database-instances/${id}`,
      data,
    );
    return response;
  },

  /**
   * Delete a database instance
   * Requires Organization Admin role
   */
  async deleteDatabaseInstance(id: string): Promise<void> {
    await apiClient.delete(`/api/v1/database-instances/${id}`);
  },

  /**
   * Test database connection
   * Note: Since backend doesn't have a dedicated test endpoint,
   * this will attempt to create an instance (which tests connection).
   * The frontend should handle this appropriately - if successful,
   * a test instance will be created that should be deleted.
   */
  async testConnection(data: CreateDatabaseInstanceRequest): Promise<{
    success: boolean;
    error?: string;
    details?: string;
    instance?: DatabaseInstance;
  }> {
    try {
      // Make a test request - backend will validate connection
      // If connection fails, backend returns 400 with error details
      const instance = await this.createDatabaseInstance(data);
      // If we get here, connection was successful
      return { success: true, instance };
    } catch (error: unknown) {
      if (error && typeof error === 'object' && 'response' in error) {
        const apiError = error as { response?: { data?: { error?: string; details?: string } } };
        if (apiError.response?.data) {
          return {
            success: false,
            error: apiError.response.data.error || 'Connection test failed',
            details: apiError.response.data.details,
          };
        }
      }
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error occurred',
      };
    }
  },
};
