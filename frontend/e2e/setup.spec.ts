import { test, expect } from '@playwright/test';

/**
 * Smoke test to verify E2E test environment setup is working
 */
test('E2E test environment setup', async ({ page }) => {
  // Navigate to home page
  await page.goto('/');

  // Verify page loads
  await expect(page).toHaveTitle(/Deeb/i);

  // Verify page content is visible
  const heading = page.locator('h1').first();
  await expect(heading).toBeVisible();
});
