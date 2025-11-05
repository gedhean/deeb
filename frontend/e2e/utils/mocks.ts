import { APIRequestContext, expect } from '@playwright/test';

/**
 * API mock data for E2E tests
 */
export const mockData = {
  users: {
    admin: {
      id: '1',
      email: 'admin@example.com',
      organization_role: 'OrganizationAdmin',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
    user: {
      id: '2',
      email: 'user@example.com',
      organization_role: 'OrganizationDeveloper',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  },
  projects: [
    {
      id: '1',
      name: 'Test Project',
      description: 'Test Description',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ],
  changePlans: [
    {
      id: '1',
      title: 'Test Change Plan',
      status: 'draft',
      project_id: '1',
      database_id: '1',
      author_id: '1',
      sql_statements: [],
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ],
  policies: [
    {
      id: '1',
      name: 'Test Policy',
      type: 'SQL_REVIEW',
      enabled: true,
      config: {},
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ],
};

/**
 * Setup API mocks for common endpoints
 */
export async function setupAPIMocks(context: APIRequestContext) {
  // This is a placeholder - actual implementation would use Playwright's route handling
  // or a test server that intercepts API calls
  return context;
}

/**
 * Create mock API response
 */
export function createMockResponse<T>(data: T, status = 200) {
  return {
    status,
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  };
}

/**
 * Verify API response structure
 */
export function expectAPIResponse(response: unknown, expectedFields: string[]) {
  if (typeof response !== 'object' || response === null) {
    throw new Error('Response is not an object');
  }

  for (const field of expectedFields) {
    expect(response).toHaveProperty(field);
  }
}
