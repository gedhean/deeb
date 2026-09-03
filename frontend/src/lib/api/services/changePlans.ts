import { apiClient } from '../client';
import type {
  ChangePlan,
  ChangePlanListResponse,
  CreateChangePlanRequest,
  UpdateChangePlanRequest,
  ApproveChangePlanRequest,
  CloseChangePlanRequest,
  GhostPreflightResponse,
  ApprovalHistoryResponse,
  ExecutionLogHistoryResponse,
  StatementExecutionResultsResponse,
} from '../types';

export const changePlanService = {
  /**
   * List change plans with optional filters
   */
  async listChangePlans(
    params?: Record<string, string | number | undefined>,
  ): Promise<ChangePlanListResponse> {
    const queryParams = new URLSearchParams();
    for (const [key, value] of Object.entries(params ?? {})) {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.append(key, String(value));
      }
    }

    const queryString = queryParams.toString();
    const url = `/api/v1/change-plans${queryString ? `?${queryString}` : ''}`;

    const response = await apiClient.get<ChangePlanListResponse>(url);
    return response;
  },

  /**
   * Get a specific change plan by ID
   */
  async getChangePlan(id: string): Promise<ChangePlan> {
    const response = await apiClient.get<ChangePlan>(`/api/v1/change-plans/${id}`);
    return response;
  },

  /**
   * Create a new change plan
   */
  async createChangePlan(data: CreateChangePlanRequest): Promise<ChangePlan> {
    const response = await apiClient.post<ChangePlan>('/api/v1/change-plans', data);
    return response;
  },

  /**
   * Trigger SQL Review manually for a change plan
   */
  async triggerSQLReview(id: string): Promise<void> {
    await apiClient.post(`/api/v1/change-plans/${id}/review`, {});
  },

  /**
   * Approve a change plan
   */
  async approveChangePlan(id: string, data?: ApproveChangePlanRequest): Promise<void> {
    await apiClient.post(`/api/v1/change-plans/${id}/approve`, data || {});
  },

  /**
   * Close/reject a change plan
   */
  async closeChangePlan(id: string, data?: CloseChangePlanRequest): Promise<void> {
    await apiClient.post(`/api/v1/change-plans/${id}/close`, data || {});
  },

  /**
   * Get approval history for a change plan
   */
  async getApprovalHistory(id: string): Promise<ApprovalHistoryResponse> {
    const response = await apiClient.get<ApprovalHistoryResponse>(
      `/api/v1/change-plans/${id}/approvals`,
    );
    return response;
  },

  /**
   * Execute a change plan
   */
  async executeChangePlan(id: string): Promise<void> {
    await apiClient.post(`/api/v1/change-plans/${id}/execute`, {});
  },

  /**
   * Approve ghost cutover (ghost executor plans in AwaitingCutover)
   */
  async approveGhostCutover(id: string): Promise<void> {
    await apiClient.post(`/api/v1/change-plans/${id}/cutover-approve`, {});
  },

  /**
   * Abort ghost execution (creates the panic flag pre-cutover)
   */
  async abortGhostExecution(id: string): Promise<void> {
    await apiClient.post(`/api/v1/change-plans/${id}/abort`, {});
  },

  /**
   * Run ghost executor pre-flight checks against the target database
   */
  async getGhostPreflight(id: string): Promise<GhostPreflightResponse> {
    const response = await apiClient.get<GhostPreflightResponse>(
      `/api/v1/change-plans/${id}/ghost-preflight`,
    );
    return response.data;
  },

  /**
   * Get execution log history for a change plan
   */
  async getExecutionLogHistory(id: string): Promise<ExecutionLogHistoryResponse> {
    const response = await apiClient.get<ExecutionLogHistoryResponse>(
      `/api/v1/change-plans/${id}/execution-logs`,
    );
    return response;
  },

  /**
   * Update a change plan
   */
  async updateChangePlan(id: string, data: UpdateChangePlanRequest): Promise<ChangePlan> {
    const response = await apiClient.put<ChangePlan>(`/api/v1/change-plans/${id}`, data);
    return response;
  },

  /**
   * Get statement execution results for an execution log
   */
  async getStatementResults(
    changePlanId: string,
    executionLogId: string,
  ): Promise<StatementExecutionResultsResponse> {
    const response = await apiClient.get<StatementExecutionResultsResponse>(
      `/api/v1/change-plans/${changePlanId}/execution-logs/${executionLogId}/statements`,
    );
    return response;
  },
};
