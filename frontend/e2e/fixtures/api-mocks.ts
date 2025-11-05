import { test as base, Page } from '@playwright/test';
import { mockAPIResponse } from '../utils/helpers';
import { mockData } from '../utils/mocks';

type MockAPIFixtures = {
  mockAPIPage: Page;
};

/**
 * API mock fixtures for E2E tests
 * Provides pages with mocked API responses
 */
export const testWithMocks = base.extend<MockAPIFixtures>({
  mockAPIPage: async ({ page }, use) => {
    // Setup common API mocks
    await mockAPIResponse(page, /\/api\/v1\/projects/, {
      items: mockData.projects,
      total: mockData.projects.length,
      page: 1,
      page_size: 100,
    });

    await mockAPIResponse(page, /\/api\/v1\/change-plans/, {
      change_plans: mockData.changePlans,
      total: mockData.changePlans.length,
      limit: 100,
      offset: 0,
    });

    await mockAPIResponse(page, /\/api\/v1\/policies/, {
      items: mockData.policies,
      total: mockData.policies.length,
      page: 1,
      page_size: 100,
    });

    await use(page);
  },
});

export { expect } from '@playwright/test';
