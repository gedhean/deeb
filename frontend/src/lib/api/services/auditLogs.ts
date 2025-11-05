import { apiClient } from '../client';
import type { AuditLogListResponse, ListAuditLogsParams } from '../types';

export const auditLogService = {
  /**
   * List audit logs with optional filters
   */
  async listAuditLogs(params?: ListAuditLogsParams): Promise<AuditLogListResponse> {
    const queryParams = new URLSearchParams();

    if (params?.user_id) {
      queryParams.append('user_id', params.user_id);
    }
    if (params?.action) {
      queryParams.append('action', params.action);
    }
    if (params?.target_type) {
      queryParams.append('target_type', params.target_type);
    }
    if (params?.target_id) {
      queryParams.append('target_id', params.target_id);
    }
    if (params?.start_date) {
      queryParams.append('start_date', params.start_date);
    }
    if (params?.end_date) {
      queryParams.append('end_date', params.end_date);
    }
    if (params?.limit) {
      queryParams.append('limit', params.limit.toString());
    }
    if (params?.offset) {
      queryParams.append('offset', params.offset.toString());
    }

    const queryString = queryParams.toString();
    const url = queryString ? `/api/v1/audit-logs?${queryString}` : '/api/v1/audit-logs';

    const response = await apiClient.get<AuditLogListResponse>(url);
    return response;
  },
};
