import { test, expect, type Page } from '@playwright/test';
import { mockData } from './utils/mocks';

/**
 * E2E coverage for the "Preserve Path on Login Redirect" feature.
 *
 * Proves the full deep-link -> login -> land-back experience for both login
 * methods, plus the no-bounce fallback. All network is mocked so the tests are
 * hermetic (no real backend, no real Google OAuth).
 *
 * The sessionStorage key below mirrors the internal `deeb.returnTo` key used by
 * `src/lib/auth/return-to.ts`. The helper does not export it, so the OAuth test
 * seeds it directly to simulate the state left behind by a pre-login bounce.
 */
const RETURN_TO_KEY = 'deeb.returnTo';
const DEEP_LINK = '/projects/42';

/**
 * Installs hermetic auth mocks with a mutable authentication state.
 *
 * - `/api/v1/auth/me` reflects the current `authenticated` flag (401 -> 200).
 * - `/api/v1/auth/login` flips the flag on and succeeds.
 * - A catch-all keeps every other API call from hitting the network so the
 *   protected deep-link page renders without a real backend.
 *
 * Playwright matches route handlers in reverse registration order, so the
 * catch-all is registered first (fallback) and the specific auth routes last
 * (higher priority).
 */
async function setupAuthMocks(page: Page, { startAuthenticated = false } = {}) {
  const state = { authenticated: startAuthenticated };

  // Fallback: benign 200 for any other API call the app makes while rendering
  // a protected page. We only assert on the URL, so the exact shape is inert.
  await page.route('**/api/v1/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ items: [], total: 0 }),
    });
  });

  // Deep-link target: return a concrete project so the detail page renders.
  await page.route('**/api/v1/projects/42', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ...mockData.projects[0], id: '42', name: 'Deep Link Project' }),
    });
  });

  await page.route('**/api/v1/auth/me', async (route) => {
    if (state.authenticated) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ user: mockData.users.admin }),
      });
    } else {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Unauthorized' }),
      });
    }
  });

  await page.route('**/api/v1/auth/login', async (route) => {
    state.authenticated = true;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ user: mockData.users.admin, message: 'Login successful' }),
    });
  });

  return state;
}

async function submitLogin(page: Page) {
  await page.fill('input[type="email"]', 'admin@example.com');
  await page.fill('input[type="password"]', 'admin123');
  await page.click('button[type="submit"]');
}

test.describe('Preserve path on login redirect (E2E)', () => {
  test('password path: deep-link -> login -> lands back on original path', async ({ page }) => {
    await setupAuthMocks(page, { startAuthenticated: false });

    // Logged-out visit to a protected deep-link bounces to /login.
    await page.goto(DEEP_LINK);
    await page.waitForURL(/\/login/, { timeout: 10000 });
    await expect(page).toHaveURL(/\/login/);

    // Complete email/password login.
    await submitLogin(page);

    // Lands back on the originally-requested deep-link, not /dashboard.
    await page.waitForURL(`**${DEEP_LINK}`, { timeout: 10000 });
    await expect(page).toHaveURL(new RegExp(`${DEEP_LINK}$`));
  });

  test('oauth path: /auth/callback forwards to the pre-seeded returnTo path', async ({ page }) => {
    // The auth cookie is already set by the time the OAuth roundtrip lands, so
    // this flow is authenticated from the start.
    await setupAuthMocks(page, { startAuthenticated: true });

    // Seed returnTo the way a pre-login bounce would have, before the callback
    // route reads it. Navigate to a same-origin page first to get an origin.
    await page.goto('/login');
    await page.evaluate(
      ([key, value]) => sessionStorage.setItem(key, value),
      [RETURN_TO_KEY, DEEP_LINK],
    );

    // Simulate the backend redirect landing the SPA on /auth/callback.
    await page.goto('/auth/callback');

    // The callback forwards to the original path and clears returnTo.
    await page.waitForURL(`**${DEEP_LINK}`, { timeout: 10000 });
    await expect(page).toHaveURL(new RegExp(`${DEEP_LINK}$`));
    expect(await page.evaluate((key) => sessionStorage.getItem(key), RETURN_TO_KEY)).toBeNull();
  });

  test('fallback: direct /login with no prior bounce lands on /dashboard', async ({ page }) => {
    await setupAuthMocks(page, { startAuthenticated: false });

    // Direct visit to /login (no bounce -> no RR state, no returnTo).
    await page.goto('/login');
    await expect(page).toHaveURL(/\/login/);
    expect(await page.evaluate((key) => sessionStorage.getItem(key), RETURN_TO_KEY)).toBeNull();

    await submitLogin(page);

    // Falls back to /dashboard, with no stale returnTo to send us elsewhere.
    await page.waitForURL('**/dashboard', { timeout: 10000 });
    await expect(page).toHaveURL(/\/dashboard/);
  });
});
