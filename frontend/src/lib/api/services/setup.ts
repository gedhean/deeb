import { apiClient } from '../client';

export interface SetupStatusResponse {
  requires_setup: boolean;
  user_count: number;
}

export const setupService = {
  /**
   * Check if the system requires initial setup (first user registration)
   */
  async getSetupStatus(): Promise<SetupStatusResponse> {
    const response = await apiClient.get<SetupStatusResponse>('/api/v1/setup/status');
    return response;
  },
};
