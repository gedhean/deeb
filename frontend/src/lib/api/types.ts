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

// List Response with Pagination
export interface ListResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

// API Error
export class APIError extends Error {
  constructor(
    message: string,
    public statusCode: number,
    public response?: ErrorResponse
  ) {
    super(message);
    this.name = 'APIError';
  }
}
