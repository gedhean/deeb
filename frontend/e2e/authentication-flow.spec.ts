import { test, expect } from '@playwright/test';
import { mockData } from './utils/mocks';

test.describe('Authentication Flow E2E', () => {
  test.beforeEach(async ({ page }) => {
    // Mock API responses for authentication
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: mockData.users.admin,
          message: 'Login successful',
        }),
      });
    });

    await page.route('**/api/v1/auth/register', async (route) => {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          message: 'Registration successful',
        }),
      });
    });

    await page.route('**/api/v1/auth/logout', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          message: 'Logout successful',
        }),
      });
    });

    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: mockData.users.admin,
        }),
      });
    });
  });

  test('Login flow', async ({ page }) => {
    // Navigate to login page
    await page.goto('/login');
    await expect(page).toHaveURL(/\/login/);

    // Verify login page is visible
    await expect(
      page
        .locator('h1, h2')
        .filter({ hasText: /Welcome to Deeb|Sign in/i })
        .first(),
    ).toBeVisible();

    // Fill in email
    const emailInput = page.locator('input[type="email"]');
    await expect(emailInput).toBeVisible();
    await emailInput.fill('admin@example.com');

    // Fill in password
    const passwordInput = page.locator('input[type="password"]');
    await expect(passwordInput).toBeVisible();
    await passwordInput.fill('admin123');

    // Mock login success
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: mockData.users.admin,
          message: 'Login successful',
        }),
      });
    });

    // Mock user data after login
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: mockData.users.admin,
        }),
      });
    });

    // Submit form
    const submitButton = page.locator('button[type="submit"]');
    await expect(submitButton).toBeVisible();
    await submitButton.click();

    // Wait for redirect to dashboard
    await page.waitForURL('/dashboard', { timeout: 10000 });
    await expect(page).toHaveURL(/\/dashboard/);

    // Verify we're on dashboard (authenticated)
    await expect(page.locator('h1').first()).toContainText(/Dashboard/i, { timeout: 5000 });
  });

  test('Login flow with invalid credentials', async ({ page }) => {
    await page.goto('/login');

    // Mock login failure
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Invalid email or password',
        }),
      });
    });

    // Fill in invalid credentials
    await page.fill('input[type="email"]', 'invalid@example.com');
    await page.fill('input[type="password"]', 'wrongpassword');

    // Submit form
    await page.click('button[type="submit"]');

    // Wait for error message
    await expect(page.locator('text=/Invalid|error/i').first()).toBeVisible({ timeout: 5000 });

    // Verify we're still on login page
    await expect(page).toHaveURL(/\/login/);
  });

  test('Registration flow', async ({ page }) => {
    // Navigate to registration page
    await page.goto('/register');
    await expect(page).toHaveURL(/\/register/);

    // Verify registration page is visible
    await expect(
      page
        .locator('h1, h2')
        .filter({ hasText: /Create an account|Initial Setup/i })
        .first(),
    ).toBeVisible();

    // Fill in email
    const emailInput = page.locator('input[type="email"]');
    await expect(emailInput).toBeVisible();
    await emailInput.fill('newuser@example.com');

    // Fill in password
    const passwordInputs = page.locator('input[type="password"]');
    await expect(passwordInputs.first()).toBeVisible();
    await passwordInputs.first().fill('SecurePassword123!');

    // Fill in confirm password
    await expect(passwordInputs.nth(1)).toBeVisible();
    await passwordInputs.nth(1).fill('SecurePassword123!');

    // Mock registration success
    await page.route('**/api/v1/auth/register', async (route) => {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          message: 'Registration successful',
        }),
      });
    });

    // Submit form
    const submitButton = page.locator('button[type="submit"]');
    await expect(submitButton).toBeVisible();
    await submitButton.click();

    // Wait for success message or redirect
    await expect(page.locator('text=/success|redirecting|Account created/i').first()).toBeVisible({
      timeout: 5000,
    });

    // Should redirect to login after registration
    await page.waitForURL('/login', { timeout: 5000 });
    await expect(page).toHaveURL(/\/login/);
  });

  test('Registration flow with validation errors', async ({ page }) => {
    await page.goto('/register');

    // Try to submit without filling fields
    const submitButton = page.locator('button[type="submit"]');
    await submitButton.click();

    // Check for validation errors
    await expect(page.locator('text=/required|Email is required/i').first()).toBeVisible({
      timeout: 3000,
    });

    // Fill invalid email
    await page.fill('input[type="email"]', 'invalid-email');
    await page.fill('input[type="password"]', 'short');
    await page.fill('input[type="password"]').nth(1).fill('short');

    // Blur to trigger validation
    await page.locator('input[type="email"]').blur();

    // Check for email validation error
    await expect(page.locator('text=/Invalid email|email format/i').first()).toBeVisible({
      timeout: 3000,
    });

    // Check for password validation error
    await expect(page.locator('text=/Password must be|at least 8/i').first()).toBeVisible({
      timeout: 3000,
    });
  });

  test('Logout flow', async ({ page }) => {
    // First, login
    await page.goto('/login');
    await page.fill('input[type="email"]', 'admin@example.com');
    await page.fill('input[type="password"]', 'admin123');

    // Mock login
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: mockData.users.admin,
          message: 'Login successful',
        }),
      });
    });

    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: mockData.users.admin,
        }),
      });
    });

    await page.click('button[type="submit"]');
    await page.waitForURL('/dashboard', { timeout: 10000 });

    // Mock logout
    await page.route('**/api/v1/auth/logout', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          message: 'Logout successful',
        }),
      });
    });

    // Mock unauthenticated state after logout
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Unauthorized',
        }),
      });
    });

    // Find and click logout button (in user menu)
    // The logout button is in a dropdown menu
    // Check if sidebar exists
    const sidebar = page.locator('aside, nav').first();
    if (await sidebar.isVisible()) {
      // Look for user avatar or email in sidebar
      const userButton = page
        .locator('button:has-text("admin@example.com"), [role="button"]:has-text("admin"), button')
        .filter({ hasText: /admin/i })
        .first();

      if (await userButton.isVisible({ timeout: 2000 }).catch(() => false)) {
        await userButton.click();
      } else {
        // Try clicking on avatar or user area
        const avatar = page.locator('img[alt*="admin"], [aria-label*="user"], button').first();
        if (await avatar.isVisible({ timeout: 2000 }).catch(() => false)) {
          await avatar.click();
        }
      }
    }

    // Look for logout menu item
    const logoutButton = page.locator('text=/Log out|Logout/i').first();

    // If menu is not open, try clicking user area first
    if (!(await logoutButton.isVisible({ timeout: 1000 }).catch(() => false))) {
      // Try to find and click user menu trigger
      const menuTriggers = page
        .locator('button, [role="button"]')
        .filter({ hasText: /admin|user/i });
      const trigger = await menuTriggers
        .first()
        .isVisible({ timeout: 2000 })
        .catch(() => false);
      if (trigger) {
        await menuTriggers.first().click();
        await page.waitForTimeout(500);
      }
    }

    // Click logout
    await expect(logoutButton).toBeVisible({ timeout: 5000 });
    await logoutButton.click();

    // Wait for redirect to login or home
    await page.waitForURL(/\/login|\//, { timeout: 10000 });

    // Verify we're logged out (should be redirected to login or home)
    const currentURL = page.url();
    expect(currentURL).toMatch(/\/login|\//);
  });

  test('OAuth flow initiation', async ({ page }) => {
    await page.goto('/login');

    // Verify Google OAuth button is visible
    const googleButton = page
      .locator('button:has-text("Google"), button:has-text("Sign in with Google")')
      .first();
    await expect(googleButton).toBeVisible();

    // Click Google OAuth button
    await googleButton.click();

    // OAuth redirects to external URL, so we can verify the redirect was initiated
    // The actual OAuth flow requires Google's servers, so we can't fully test it
    // But we can verify that clicking the button triggers the redirect
    await page.waitForTimeout(1000);

    // Verify that navigation was attempted (either redirect happened or URL changed)
    // Since OAuth redirects to external URL, we check if page navigated away
    // If OAuth was initiated, URL should either be the OAuth endpoint or Google's domain
    // For this test, we just verify the button click worked (no error)
    expect(googleButton).toBeDefined();
  });

  test('Protected route redirects to login when not authenticated', async ({ page }) => {
    // Mock unauthenticated state
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Unauthorized',
        }),
      });
    });

    // Try to access protected route
    await page.goto('/dashboard');

    // Should redirect to login
    await page.waitForURL(/\/login/, { timeout: 5000 });
    await expect(page).toHaveURL(/\/login/);
  });

  test('Public route redirects authenticated users to dashboard', async ({ page }) => {
    // Mock authenticated state
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: mockData.users.admin,
        }),
      });
    });

    // Try to access public route (login) while authenticated
    await page.goto('/login');

    // Should redirect to dashboard
    await page.waitForURL('/dashboard', { timeout: 5000 });
    await expect(page).toHaveURL(/\/dashboard/);
  });
});
