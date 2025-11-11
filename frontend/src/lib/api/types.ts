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
  display_name?: string;
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

export interface CreateUserRequest {
  email: string;
  password: string;
  display_name?: string;
  organization_role: 'OrganizationAdmin' | 'OrganizationDBA' | 'OrganizationDeveloper';
}

export interface UpdateUserRequest {
  password?: string;
  display_name?: string;
  organization_role?: 'OrganizationAdmin' | 'OrganizationDBA' | 'OrganizationDeveloper';
  is_active?: boolean;
  two_factor_enabled?: boolean;
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

// Project Member Types
export type ProjectRole = 'ProjectOwner' | 'ProjectDeveloper' | 'ProjectApprover' | 'ProjectViewer';

export interface ProjectMember {
  id: string;
  project_id: string;
  user_id: string;
  role: ProjectRole;
  added_by_id: string;
  added_at: string;
}

export interface ProjectMemberWithUser {
  member: ProjectMember;
  user: {
    id: string;
    email: string;
    display_name?: string;
  };
}

export interface AddProjectMemberRequest {
  user_id: string;
  project_role: ProjectRole;
}

export interface UpdateProjectMemberRoleRequest {
  project_role: ProjectRole;
}

export interface ProjectMembersListResponse {
  members: ProjectMemberWithUser[];
  count: number;
}

// Database Instance Types
export interface DatabaseInstance {
  id: string;
  name: string;
  description?: string;
  database_type: 'mysql' | 'postgresql';
  host: string;
  port: number;
  username: string;
  use_rds_iam_auth: boolean;
  require_ssl: boolean;
  status: 'Active' | 'Inactive' | 'Maintenance';
  environment_id?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateDatabaseInstanceRequest {
  name: string;
  description?: string;
  database_type: 'mysql' | 'postgresql';
  host: string;
  port: number;
  username: string;
  password: string;
  use_rds_iam_auth?: boolean;
  require_ssl?: boolean;
  environment_id?: string;
}

export interface UpdateDatabaseInstanceRequest {
  name?: string;
  description?: string;
  database_type?: 'mysql' | 'postgresql';
  host?: string;
  port?: number;
  username?: string;
  password?: string;
  use_rds_iam_auth?: boolean;
  require_ssl?: boolean;
  status?: 'Active' | 'Inactive' | 'Maintenance';
  environment_id?: string;
}

// Database Types
export interface Database {
  id: string;
  project_id: string;
  database_instance_id: string;
  environment_id: string;
  database_name: string;
  description?: string;
  status: 'Active' | 'Inactive';
  created_at: string;
  updated_at: string;
  project?: Project;
  database_instance?: DatabaseInstance;
  environment?: Environment;
}

export interface CreateDatabaseRequest {
  project_id: string;
  database_instance_id: string;
  environment_id: string;
  database_name: string;
  description?: string;
}

export interface UpdateDatabaseRequest {
  database_name?: string;
  description?: string;
  status?: 'Active' | 'Inactive';
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
  policy_id?: string | null;
}

// Policy Types
export type SQLReviewRuleLevel = 'error' | 'warning';

export type DatabaseType = 'mysql' | 'postgresql';

export type RuleCategory =
  | 'Engine'
  | 'Naming'
  | 'Table'
  | 'Column'
  | 'System'
  | 'Index'
  | 'Schema'
  | 'Statement';

export interface SQLReviewRule {
  type: string;
  level: SQLReviewRuleLevel;
  payload?: Record<string, unknown>;
  engine: DatabaseType;
  comment?: string;
}

export interface SQLReviewPolicyConfig {
  rules: Record<string, SQLReviewRule>;
}

export interface RuleParameter {
  name: string;
  type: 'STRING' | 'NUMBER' | 'BOOLEAN' | 'STRING_ARRAY' | 'TEMPLATE';
  description: string;
  required: boolean;
  default?: unknown;
  templateList?: string[];
}

export interface RuleMetadata {
  type: string;
  name: string;
  description: string;
  category: RuleCategory;
  engine: DatabaseType;
  parameters?: RuleParameter[];
}

export interface Policy {
  id: string;
  name: string;
  description?: string;
  type: 'SQL_REVIEW';
  policy_config: SQLReviewPolicyConfig;
  created_at: string;
  updated_at: string;
}

export interface CreatePolicyRequest {
  name: string;
  description?: string;
  type: 'SQL_REVIEW';
  policy_config: SQLReviewPolicyConfig;
}

export interface UpdatePolicyRequest {
  name?: string;
  description?: string;
  policy_config?: SQLReviewPolicyConfig;
}

export interface SQLReviewRulesResponse {
  mysql?: Record<RuleCategory, RuleMetadata[]>;
  postgresql?: Record<RuleCategory, RuleMetadata[]>;
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

export interface CreateChangePlanRequest {
  project_id: string;
  target_database_id: string;
  title: string;
  description: string;
  change_type: ChangeType;
  sql_content: string;
  rollback_sql?: string;
}

export interface ApproveChangePlanRequest {
  comments?: string;
}

export interface CloseChangePlanRequest {
  comments?: string;
}

export interface UpdateChangePlanRequest {
  title?: string;
  description?: string;
  sql_content?: string;
  rollback_sql?: string;
  status?: ChangePlanStatus;
  risk_level?: RiskLevel;
  assigned_reviewer_id?: string;
}

export interface Approval {
  id: string;
  change_plan_id: string;
  approver_id: string;
  status: 'Approved' | 'Rejected';
  comments?: string;
  approved_at?: string;
  created_at: string;
}

export interface ApprovalHistoryResponse {
  change_plan_id: string;
  approvals: Approval[];
}

export interface ExecutionLog {
  id: string;
  change_plan_id: string;
  executor_id: string;
  status: 'Completed' | 'Failed';
  execution_duration_ms: number;
  rows_affected?: number;
  error_message?: string;
  sql_content?: string;
  database_id?: string;
  database_name?: string;
  database_instance_name?: string;
  database_type?: string;
  started_at: string;
  completed_at: string;
  created_at: string;
}

export interface ExecutionLogHistoryResponse {
  change_plan_id: string;
  total: number;
  logs: ExecutionLog[];
}

// Audit Log Types
export interface AuditLog {
  id: string;
  user_id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  details?: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
  timestamp: string;
}

export interface ListAuditLogsParams {
  user_id?: string;
  action?: string;
  target_type?: string;
  target_id?: string;
  start_date?: string;
  end_date?: string;
  limit?: number;
  offset?: number;
}

export interface AuditLogListResponse {
  audit_logs: AuditLog[];
  total: number;
  limit: number;
  offset: number;
  count: number;
}

// Review Findings Filtering
export interface ReviewFindingsFilterParams {
  severity?: string;
  rule_name?: string;
  validator_type?: string;
}

export interface ReviewFindingsResponse {
  change_plan_id: string;
  findings: PolicyFindingItem[];
  summary: {
    total: number;
    errors: number;
    warnings: number;
    infos: number;
  };
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
