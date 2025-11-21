// API Response Types

import type { OrganizationRole, ProjectRole } from '../rbac.generated';

// Re-export RBAC types for convenience
export type { OrganizationRole, ProjectRole };

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
  organization_role: OrganizationRole;
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
  organization_role?: OrganizationRole;
}

export interface RegisterResponse {
  user: User;
}

export interface CreateUserRequest {
  email: string;
  password: string;
  display_name?: string;
  organization_role: OrganizationRole;
}

export interface UpdateUserRequest {
  password?: string;
  display_name?: string;
  organization_role?: OrganizationRole;
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
// Note: ProjectRole is imported from rbac.generated (YAML-based)

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
export interface DatabaseSyncResult {
  total_found: number;
  synced: number;
  failed: number;
  skipped: number;
  failed_databases?: string[];
  skipped_databases?: string[];
}

export interface DatabaseInstance {
  id: string;
  name: string;
  description?: string;
  database_engine: 'mysql' | 'postgres';
  host: string;
  port: number;
  username: string;
  use_rds_iam_auth: boolean;
  require_ssl: boolean;
  status: DatabaseInstanceStatus;
  environment_id?: string;
  created_at: string;
  updated_at: string;
  sync_results?: DatabaseSyncResult;
}

export interface CreateDatabaseInstanceRequest {
  name: string;
  description?: string;
  database_engine: 'mysql' | 'postgres';
  host: string;
  port: number;
  username: string;
  password: string;
  use_rds_iam_auth?: boolean;
  require_ssl?: boolean;
  environment_id?: string;
  sync_all_databases?: boolean;
}

export interface UpdateDatabaseInstanceRequest {
  name?: string;
  description?: string;
  database_engine?: 'mysql' | 'postgres';
  host?: string;
  port?: number;
  username?: string;
  password?: string;
  use_rds_iam_auth?: boolean;
  require_ssl?: boolean;
  status?: DatabaseInstanceStatus;
  environment_id?: string;
}

// Database Status Types
export type DatabaseStatus = 'Active' | 'Inactive' | 'Pending';
export type DatabaseInstanceStatus = 'Active' | 'Inactive' | 'Maintenance';

// Database Types
export interface Database {
  id: string;
  project_id?: string | null;
  database_instance_id: string;
  environment_id?: string | null;
  database_name: string;
  description?: string;
  status: DatabaseStatus;
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
  project_id?: string;
  environment_id?: string;
  description?: string;
  status?: DatabaseStatus;
}

// Environment Types
export interface Environment {
  id: string;
  name: string;
  description?: string;
  policy_id?: string;
  allow_self_approval: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateEnvironmentRequest {
  name: string;
  description?: string;
  policy_id?: string;
  allow_self_approval?: boolean;
}

export interface UpdateEnvironmentRequest {
  name?: string;
  description?: string;
  policy_id?: string | null;
  allow_self_approval?: boolean;
}

// Policy Types
export type SQLReviewRuleLevel = 'error' | 'warning';

export type DatabaseEngine = 'mysql' | 'postgres';

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
  engine: DatabaseEngine;
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
  engine: DatabaseEngine;
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
  postgres?: Record<RuleCategory, RuleMetadata[]>;
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
  database_engine?: string;
  started_at: string;
  completed_at: string;
  created_at: string;
}

export interface ExecutionLogHistoryResponse {
  change_plan_id: string;
  total: number;
  logs: ExecutionLog[];
}

// Changelog Types

// Snapshot status constants
export const SNAPSHOT_STATUS = {
  SUCCESS: 'SUCCESS',
  FAILED: 'FAILED',
} as const;

export type SnapshotStatus = (typeof SNAPSHOT_STATUS)[keyof typeof SNAPSHOT_STATUS];

export interface ChangelogMetadata {
  snapshot_status?: SnapshotStatus;
  snapshot_error?: string;
}

export interface Changelog {
  id: string;
  database_instance_id: string;
  database_id: string;
  change_plan_id: string;
  change_plan_title?: string; // Included in list view
  status: 'pending' | 'done' | 'failed';
  prev_schema_history_id: string | null;
  schema_history_id: string | null;
  metadata?: ChangelogMetadata | null;
  created_at: string;
}

export interface SchemaHistory {
  id: string;
  database_instance_id: string;
  database_id: string;
  metadata: Record<string, unknown>;
  raw_dump: string; // The SQL dump
  created_at: string;
}

export interface ChangePlanSummary {
  id: string;
  title: string;
  description: string;
}

export interface ChangelogDetail {
  changelog: Changelog;
  change_plan: ChangePlanSummary;
  schema_history: SchemaHistory | null;
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

// Schema Metadata Types
export interface TableListItem {
  name: string;
  row_count: number;
  data_size: number;
  index_size: number;
  total_size: number;
  engine: string;
  comment: string;
  created_at?: string;
  updated_at?: string;
}

export interface Column {
  name: string;
  position: number;
  type: string;
  nullable: boolean;
  default?: string;
  comment: string;
  character_set: string;
  collation: string;
  on_update: string;
  is_generated: boolean;
  generated_as: string;
  is_identity: boolean;
}

export interface Index {
  name: string;
  type: string;
  columns: string[];
  primary: boolean;
  unique: boolean;
  visible: boolean;
  comment: string;
  where_clause: string;
}

export interface ForeignKey {
  name: string;
  columns: string[];
  referenced_table: string;
  referenced_columns: string[];
  on_delete: string;
  on_update: string;
}

export interface CheckConstraint {
  name: string;
  expression: string;
}

export interface ViewListItem {
  name: string;
  definition: string;
  comment: string;
}

export interface MaterializedViewListItem {
  name: string;
  definition: string;
  comment: string;
}

export interface ExtensionListItem {
  name: string;
  schema: string;
  version: string;
  description: string;
}

export interface FunctionListItem {
  name: string;
  definition: string;
  comment: string;
}

export interface SequenceListItem {
  name: string;
  data_type: string;
  start: number;
  increment: number;
  min: number;
  max: number;
  cache: number;
  cycle: boolean;
  comment: string;
}

export interface TableDetail {
  name: string;
  comment: string;
  engine: string;
  collation: string;
  charset: string;
  row_count: number;
  data_size: number;
  index_size: number;
  total_size: number;
  created_at?: string;
  updated_at?: string;
  columns: Column[];
  indexes: Index[];
  foreign_keys: ForeignKey[];
  check_constraints: CheckConstraint[];
}

export interface DatabaseSchemaMetadata {
  database_name: string;
  database_type: string;
  tables: TableListItem[];
  views: ViewListItem[];
  materialized_views?: MaterializedViewListItem[];
  extensions?: ExtensionListItem[];
  functions?: FunctionListItem[];
  sequences?: SequenceListItem[];
  last_synced_at?: string;
}

// API Error
export class APIError extends Error {
  statusCode: number;
  response?: ErrorResponse;

  constructor(message: string, statusCode: number, response?: ErrorResponse) {
    super(message);
    this.name = 'APIError';
    this.statusCode = statusCode;
    this.response = response;
  }
}
