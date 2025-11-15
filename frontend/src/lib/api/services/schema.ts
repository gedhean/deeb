import { apiClient } from '../client';
import type { DatabaseSchemaMetadata, TableDetail } from '../types';

export const schemaService = {
  /**
   * Get schema metadata for a database
   * Returns list of tables with basic statistics
   * Requires authentication and database read permission
   */
  async getDatabaseSchema(databaseId: string): Promise<DatabaseSchemaMetadata> {
    const response = await apiClient.get<DatabaseSchemaMetadata>(
      `/api/v1/databases/${databaseId}/schema`,
    );
    return response;
  },

  /**
   * Get detailed information for a specific table
   * Returns table metadata, columns, indexes, foreign keys, and constraints
   * Requires authentication and database read permission
   */
  async getTableDetail(databaseId: string, tableName: string): Promise<TableDetail> {
    const response = await apiClient.get<TableDetail>(
      `/api/v1/databases/${databaseId}/tables/${encodeURIComponent(tableName)}`,
    );
    return response;
  },

  /**
   * Manually refresh schema metadata for a database
   * Forces immediate extraction bypassing cache
   * Requires authentication and database read permission
   */
  async refreshDatabaseSchema(databaseId: string): Promise<DatabaseSchemaMetadata> {
    const response = await apiClient.post<DatabaseSchemaMetadata>(
      `/api/v1/databases/${databaseId}/schema/refresh`,
    );
    return response;
  },
};
