import { Page, expect } from '@playwright/test';

/**
 * Login helper function
 * Navigates to login page and performs login with provided credentials
 */
export async function loginAs(page: Page, email: string, password: string) {
  await page.goto('/login');
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', password);
  await page.click('button[type="submit"]');
  // Wait for redirect to dashboard after successful login
  await page.waitForURL('/dashboard', { timeout: 5000 });
}

/**
 * Login as admin user (for tests that need admin privileges)
 */
export async function loginAsAdmin(page: Page) {
  const adminEmail = process.env.E2E_ADMIN_EMAIL || 'admin@example.com';
  const adminPassword = process.env.E2E_ADMIN_PASSWORD || 'admin123';
  await loginAs(page, adminEmail, adminPassword);
}

/**
 * Login as regular user (for tests that need non-admin user)
 */
export async function loginAsUser(page: Page) {
  const userEmail = process.env.E2E_USER_EMAIL || 'user@example.com';
  const userPassword = process.env.E2E_USER_PASSWORD || 'user123';
  await loginAs(page, userEmail, userPassword);
}

/**
 * Logout helper function
 */
export async function logout(page: Page) {
  // Assuming logout button is in navigation/user menu
  // This will need to be adjusted based on actual UI implementation
  await page.click('button:has-text("Logout")');
  await page.waitForURL('/login', { timeout: 5000 });
}

/**
 * Wait for API call to complete
 * Useful for waiting for async operations to finish
 */
export async function waitForAPICall(page: Page, urlPattern: string | RegExp) {
  await page.waitForResponse((response) => {
    const url = response.url();
    if (typeof urlPattern === 'string') {
      return url.includes(urlPattern);
    }
    return urlPattern.test(url);
  });
}

/**
 * Mock API response
 * Sets up route interception for API mocking
 */
export async function mockAPIResponse(
  page: Page,
  urlPattern: string | RegExp,
  mockResponse: unknown,
  status = 200,
) {
  await page.route(urlPattern, async (route) => {
    await route.fulfill({
      status,
      contentType: 'application/json',
      body: JSON.stringify(mockResponse),
    });
  });
}

/**
 * Verify toast notification appears
 */
export async function expectToast(page: Page, message: string) {
  const toast = page.locator(`text=${message}`).first();
  await expect(toast).toBeVisible({ timeout: 5000 });
}

/**
 * Wait for loading spinner to disappear
 */
export async function waitForLoadingToComplete(page: Page) {
  await page.waitForSelector('[data-testid="loading-spinner"]', { state: 'hidden' }).catch(() => {
    // If spinner doesn't exist, continue
  });
}

/**
 * Fill form fields helper
 */
export async function fillForm(
  page: Page,
  fields: Record<string, string>,
  options?: { submit?: boolean },
) {
  for (const [name, value] of Object.entries(fields)) {
    const input = page.locator(`input[name="${name}"], textarea[name="${name}"]`).first();
    await input.fill(value);
  }

  if (options?.submit) {
    await page.click('button[type="submit"]');
  }
}

/**
 * Select dropdown option helper
 */
export async function selectOption(page: Page, selectName: string, optionValue: string) {
  const select = page.locator(`select[name="${selectName}"]`).first();
  await select.selectOption(optionValue);
}

/**
 * Check if element is visible helper
 */
export async function expectVisible(page: Page, selector: string) {
  await expect(page.locator(selector).first()).toBeVisible();
}

/**
 * Check if element is not visible helper
 */
export async function expectNotVisible(page: Page, selector: string) {
  await expect(page.locator(selector).first()).not.toBeVisible();
}
