import { test, expect, type Page } from '@playwright/test';

/**
 * E2E regression for the first-user registration redirect.
 *
 * Bug: after the first user was created, RegisterPage announced "Redirecting to
 * login..." and called navigate('/login'), but SetupCheck (which reads a cached
 * /api/setup/status with revalidateIfStale disabled) still saw requires_setup
 * = true and bounced the user straight back to /register.
 *
 * Fix: RegisterPage refreshes the cached setup status to requires_setup = false
 * after a successful registration, so SetupCheck lets the /login navigation
 * through. This test proves the user lands on /login, not /register.
 *
 * All network is mocked so the test is hermetic (no real backend).
 */
async function setupFirstUserMocks(page: Page) {
  // Mutable setup state: no users exist until registration succeeds.
  const state = { requiresSetup: true, userCount: 0 };

  // Fallback: benign 200 for any other API call made while rendering.
  await page.route('**/api/v1/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  // Setup status reflects the current state (drives SetupCheck).
  await page.route('**/api/v1/setup/status', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ requires_setup: state.requiresSetup, user_count: state.userCount }),
    });
  });

  // Not authenticated throughout: registration does not log the user in.
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'Unauthorized' }),
    });
  });

  // Registration succeeds and the system no longer requires setup.
  await page.route('**/api/v1/auth/register', async (route) => {
    state.requiresSetup = false;
    state.userCount = 1;
    await route.fulfill({
      status: 201,
      contentType: 'application/json',
      body: JSON.stringify({ message: 'User created successfully' }),
    });
  });

  return state;
}

test.describe('First-user registration redirect (E2E)', () => {
  test('after creating the first user, lands on /login (not back on /register)', async ({
    page,
  }) => {
    await setupFirstUserMocks(page);

    // With no users, SetupCheck forces the app to the register page.
    await page.goto('/login');
    await page.waitForURL(/\/register/, { timeout: 10000 });
    await expect(page).toHaveURL(/\/register/);

    // Create the first user.
    await page.fill('#email', 'first.admin@example.com');
    await page.fill('#password', 'StrongPassw0rd!');
    await page.fill('#confirmPassword', 'StrongPassw0rd!');
    await page.click('button[type="submit"]');

    // The success message confirms intent...
    await expect(page.getByText(/Redirecting to login/i)).toBeVisible({ timeout: 10000 });

    // ...and the app actually lands on /login rather than being bounced back
    // to /register by a stale setup-status cache.
    await page.waitForURL(/\/login/, { timeout: 10000 });
    await expect(page).toHaveURL(/\/login$/);
  });
});
