import type { Changelog, ChangelogDetail } from '../types';
import { apiClient } from '../client';

/**
 * Service for managing database changelog operations
 */
export const changelogService = {
  /**
   * List all changelog entries for a specific database
   * @param databaseId - The database ID
   * @returns Promise with array of changelog entries
   */
  listDatabaseChangelog: (databaseId: string): Promise<Changelog[]> =>
    apiClient.get(`/api/v1/databases/${databaseId}/changelog`),

  /**
   * Get detailed information about a specific changelog entry
   * @param changelogId - The changelog ID
   * @returns Promise with changelog details including change plan and schema history
   */
  getChangelogDetails: (changelogId: string): Promise<ChangelogDetail> =>
    apiClient.get(`/api/v1/changelog/${changelogId}`),
};
