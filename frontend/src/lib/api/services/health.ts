import { apiClient } from '../client';
import type { HealthCheckResponse } from '../types';

export const healthService = {
  /**
   * Check API health status
   */
  async check(): Promise<HealthCheckResponse> {
    return apiClient.get<HealthCheckResponse>('/health');
  },
};
