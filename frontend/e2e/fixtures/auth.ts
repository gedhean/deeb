import { test as base, Page } from '@playwright/test';
import { loginAsAdmin, loginAsUser } from '../utils/helpers';

type AuthFixtures = {
  authenticatedPage: Page;
  adminPage: Page;
  userPage: Page;
};

/**
 * Auth fixtures for E2E tests
 * Provides authenticated page instances for tests
 */
export const test = base.extend<AuthFixtures>({
  authenticatedPage: async ({ page }, use) => {
    // Default: login as admin
    await loginAsAdmin(page);
    await use(page);
  },

  adminPage: async ({ page }, use) => {
    await loginAsAdmin(page);
    await use(page);
  },

  userPage: async ({ page }, use) => {
    await loginAsUser(page);
    await use(page);
  },
});

export { expect } from '@playwright/test';
