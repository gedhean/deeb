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
    environment_id?: string;
  }): Promise<ListResponse<DatabaseInstance>> {
    const queryParams = new URLSearchParams();
    if (params?.limit) {
      queryParams.append('limit', params.limit.toString());
    }
    if (params?.offset) {
      queryParams.append('offset', params.offset.toString());
    }
    if (params?.environment_id) {
      queryParams.append('environment_id', params.environment_id);
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
   * Test database connection for an existing instance
   * Uses the dedicated test connection endpoint
   */
  async testConnection(
    instanceId: string,
    databaseName?: string,
  ): Promise<{
    success: boolean;
    message?: string;
    error?: string;
    host?: string;
    port?: number;
    database?: string;
    engine?: string;
    username?: string;
  }> {
    try {
      const response = await apiClient.post<{
        success: boolean;
        message: string;
        error?: string;
        host: string;
        port: number;
        database: string;
        engine: string;
      }>(`/api/v1/database-instances/${instanceId}/test-connection`, {
        database_name: databaseName,
      });
      return response;
    } catch (error: unknown) {
      if (error && typeof error === 'object' && 'response' in error) {
        const apiError = error as { response?: { data?: { error?: string } } };
        if (apiError.response?.data) {
          return {
            success: false,
            error: apiError.response.data.error || 'Connection test failed',
          };
        }
      }
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error occurred',
      };
    }
  },

  /**
   * Test database connection with provided credentials (for new instances before creation)
   * Uses the test connection endpoint without requiring an existing instance
   */
  async testConnectionWithCredentials(credentials: CreateDatabaseInstanceRequest): Promise<{
    success: boolean;
    message?: string;
    error?: string;
    host?: string;
    port?: number;
    database?: string;
    engine?: string;
  }> {
    try {
      const response = await apiClient.post<{
        success: boolean;
        message: string;
        error?: string;
        host: string;
        port: number;
        database: string;
        engine: string;
      }>('/api/v1/database-instances/test-connection', credentials);
      return response;
    } catch (error: unknown) {
      if (error && typeof error === 'object' && 'response' in error) {
        const apiError = error as { response?: { data?: { error?: string } } };
        if (apiError.response?.data) {
          return {
            success: false,
            error: apiError.response.data.error || 'Connection test failed',
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
