// API Response Types

// Health Check
export interface HealthCheckResponse {
  status: string;
  version: string;
  timestamp: string;
  database: string;
}

// Error Response
export interface ErrorResponse {
  error: string;
}

// User Types
export interface User {
  id: string;
  email: string;
  organization_role: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
}

// Authentication Types
export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface RegisterRequest {
  email: string;
  password: string;
  organization_role?: string;
}

export interface RegisterResponse {
  user: User;
}

// Project Types
export interface Project {
  id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CreateProjectRequest {
  name: string;
  description: string;
}

export interface UpdateProjectRequest {
  name?: string;
  description?: string;
}

// Database Instance Types
export interface DatabaseInstance {
  id: string;
  name: string;
  host: string;
  port: number;
  engine: 'mysql' | 'postgresql';
  environment_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreateDatabaseInstanceRequest {
  name: string;
  host: string;
  port: number;
  engine: 'mysql' | 'postgresql';
  environment_id: string;
}

export interface UpdateDatabaseInstanceRequest {
  name?: string;
  host?: string;
  port?: number;
}

// Database Types
export interface Database {
  id: string;
  name: string;
  database_instance_id: string;
  project_id: string;
  environment_id: string;
  created_at: string;
  updated_at: string;
}

export interface CreateDatabaseRequest {
  name: string;
  database_instance_id: string;
  project_id: string;
  environment_id: string;
}

export interface UpdateDatabaseRequest {
  name?: string;
}

// Environment Types
export interface Environment {
  id: string;
  name: string;
  description?: string;
  policy_id?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateEnvironmentRequest {
  name: string;
  description?: string;
  policy_id?: string;
}

export interface UpdateEnvironmentRequest {
  name?: string;
  description?: string;
  policy_id?: string;
}

// Policy Types
export interface Policy {
  id: string;
  name: string;
  description?: string;
  type: 'SQL_REVIEW';
  policy_config: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CreatePolicyRequest {
  name: string;
  description?: string;
  type: 'SQL_REVIEW';
  policy_config: Record<string, unknown>;
}

export interface UpdatePolicyRequest {
  name?: string;
  description?: string;
  policy_config?: Record<string, unknown>;
}

// List Response with Pagination
export interface ListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

// Change Plan Types
export type ChangePlanStatus =
  | 'Draft'
  | 'Open'
  | 'Approved'
  | 'Closed'
  | 'InProgress'
  | 'Completed'
  | 'Failed'
  | 'Cancelled';

export type ChangeType = 'DDL' | 'DML';

export type RiskLevel = 'Low' | 'Medium' | 'High';

export interface PolicyFindingItem {
  validator_type: string;
  rule_name: string;
  message: string;
  line_number?: number;
  column_number?: number;
}

export interface PolicyFindingCounts {
  errors: number;
  warnings: number;
  infos: number;
}

export interface PolicyFindingsSummary {
  errors: PolicyFindingItem[];
  warnings: PolicyFindingItem[];
  infos: PolicyFindingItem[];
  counts: PolicyFindingCounts;
}

export interface ChangePlan {
  id: string;
  project_id: string;
  target_database_id: string;
  title: string;
  description: string;
  change_type: ChangeType;
  sql_content: string;
  rollback_sql?: string;
  status: ChangePlanStatus;
  risk_level: RiskLevel;
  author_id: string;
  assigned_reviewer_id?: string;
  scheduled_for?: string;
  executed_at?: string;
  execution_duration_ms?: number;
  affected_rows?: number;
  error_message?: string;
  created_at: string;
  updated_at: string;
  policy_findings?: PolicyFindingsSummary;
  has_errors?: boolean;
  has_warnings?: boolean;
}

export interface ChangePlanListResponse {
  change_plans: ChangePlan[];
  total: number;
}

export interface ListChangePlansParams {
  status?: ChangePlanStatus;
  project_id?: string;
  database_id?: string;
  author_id?: string;
  limit?: number;
  offset?: number;
}

// API Error
export class APIError extends Error {
  constructor(
    message: string,
    public statusCode: number,
    public response?: ErrorResponse,
  ) {
    super(message);
    this.name = 'APIError';
  }
}
