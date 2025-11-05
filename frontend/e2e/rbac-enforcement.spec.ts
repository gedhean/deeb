import { test, expect } from '@playwright/test';
import { mockData } from './utils/mocks';

test.describe('RBAC Enforcement E2E', () => {
  test('Admin-only features accessible to admin', async ({ page }) => {
    // Mock admin user
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: mockData.users.admin,
        }),
      });
    });

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

    // Login as admin
    await page.goto('/login');
    await page.fill('input[type="email"]', 'admin@example.com');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/dashboard', { timeout: 10000 });

    // Mock projects API
    await page.route('**/api/v1/projects*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: mockData.projects,
          total: mockData.projects.length,
          page: 1,
          page_size: 100,
        }),
      });
    });

    // Verify admin can access Projects page
    await page.goto('/projects');
    await expect(page).toHaveURL(/\/projects/);
    await expect(page.locator('h1').first()).toContainText(/Projects/i, { timeout: 5000 });

    // Verify admin can see Create Project button
    const createButton = page.locator('button:has-text("Create"), a:has-text("Create")').first();
    await expect(createButton).toBeVisible({ timeout: 5000 });

    // Verify admin can access Users page
    await page.goto('/users');
    await expect(page).toHaveURL(/\/users/);
    await expect(page.locator('h1').first()).toContainText(/Users/i, { timeout: 5000 });

    // Verify admin can access Audit Logs page
    await page.goto('/audit');
    await expect(page).toHaveURL(/\/audit/);
    await expect(page.locator('h1').first()).toContainText(/Audit Log/i, { timeout: 5000 });
  });

  test('Admin-only features blocked for non-admin', async ({ page }) => {
    // Mock developer user (non-admin)
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: mockData.users.user, // OrganizationDeveloper
        }),
      });
    });

    // Mock login
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: mockData.users.user,
          message: 'Login successful',
        }),
      });
    });

    // Login as developer
    await page.goto('/login');
    await page.fill('input[type="email"]', 'user@example.com');
    await page.fill('input[type="password"]', 'user123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/dashboard', { timeout: 10000 });

    // Mock API responses for admin-only pages (should return 403 or show access denied)
    await page.route('**/api/v1/projects*', async (route) => {
      await route.fulfill({
        status: 403,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Access denied',
        }),
      });
    });

    await page.route('**/api/v1/users*', async (route) => {
      await route.fulfill({
        status: 403,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Access denied',
        }),
      });
    });

    await page.route('**/api/v1/audit-logs*', async (route) => {
      await route.fulfill({
        status: 403,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Access denied',
        }),
      });
    });

    // Try to access Projects page - should show access denied
    await page.goto('/projects');
    await expect(page).toHaveURL(/\/projects/);
    // Check for access denied message
    await expect(
      page.locator('text=/Access denied|do not have permission|required/i').first(),
    ).toBeVisible({ timeout: 5000 });

    // Try to access Users page - should show access denied
    await page.goto('/users');
    await expect(page).toHaveURL(/\/users/);
    await expect(
      page.locator('text=/Access denied|do not have permission|required/i').first(),
    ).toBeVisible({ timeout: 5000 });

    // Try to access Audit Logs page - should show access denied or redirect
    await page.goto('/audit');
    // May redirect or show access denied
    const currentURL = page.url();
    if (currentURL.includes('/audit')) {
      await expect(
        page.locator('text=/Access denied|do not have permission|required/i').first(),
      ).toBeVisible({ timeout: 5000 });
    }
  });

  test('Project Developer can create change plans but not approve', async ({ page }) => {
    // Mock developer user
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: mockData.users.user, // OrganizationDeveloper
        }),
      });
    });

    // Mock login
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: mockData.users.user,
          message: 'Login successful',
        }),
      });
    });

    // Login as developer
    await page.goto('/login');
    await page.fill('input[type="email"]', 'user@example.com');
    await page.fill('input[type="password"]', 'user123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/dashboard', { timeout: 10000 });

    // Mock projects and databases APIs
    await page.route('**/api/v1/projects*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: mockData.projects,
          total: mockData.projects.length,
          page: 1,
          page_size: 100,
        }),
      });
    });

    await page.route('**/api/v1/databases*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: [
            {
              id: '1',
              name: 'Test Database',
              project_id: '1',
              instance_id: '1',
              environment_id: '1',
              created_at: '2024-01-01T00:00:00Z',
              updated_at: '2024-01-01T00:00:00Z',
            },
          ],
          total: 1,
          page: 1,
          page_size: 100,
        }),
      });
    });

    // Verify developer can access Change Plans create page
    await page.goto('/change-plans/create');
    await expect(page).toHaveURL(/\/change-plans\/create/);
    await expect(page.locator('h1').first()).toContainText(/Create Change Plan/i, {
      timeout: 5000,
    });

    // Mock creating a change plan
    await page.route('**/api/v1/change-plans', async (route) => {
      if (route.request().method() === 'POST') {
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'test-change-plan-1',
            title: 'Test Change Plan',
            status: 'Open',
            project_id: '1',
            target_database_id: '1',
            author_id: mockData.users.user.id,
            has_errors: false,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          }),
        });
      }
    });

    // Fill form and create change plan
    await page.waitForSelector('select[name="project_id"], button:has-text("Select project")', {
      timeout: 5000,
    });

    const projectSelect = page.locator('select[name="project_id"]').first();
    const projectButton = page.locator('button:has-text("Select project")').first();

    if (await projectSelect.isVisible()) {
      await projectSelect.selectOption('1');
    } else if (await projectButton.isVisible()) {
      await projectButton.click();
      await page.waitForTimeout(500);
      await page.locator('text=Test Project').first().click();
    }

    await page.waitForTimeout(1000);

    const databaseSelect = page.locator('select[name="target_database_id"]').first();
    const databaseButton = page.locator('button:has-text("Select database")').first();

    if (await databaseSelect.isVisible()) {
      await databaseSelect.selectOption('1');
    } else if (await databaseButton.isVisible()) {
      await databaseButton.click();
      await page.waitForTimeout(500);
      await page.locator('text=Test Database').first().click();
    }

    await page.fill('input[name="title"]', 'Test Change Plan');
    await page.fill('textarea[name="description"]', 'Test description');
    await page.fill('textarea[name="sql_content"]', 'CREATE TABLE test (id INT);');

    await page.click('button[type="submit"]');
    await page.waitForURL(/\/change-plans\/.+/, { timeout: 10000 });

    // Mock change plan detail
    await page.route('**/api/v1/change-plans/test-change-plan-1', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'test-change-plan-1',
          title: 'Test Change Plan',
          status: 'Open',
          project_id: '1',
          target_database_id: '1',
          author_id: mockData.users.user.id,
          has_errors: false,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }),
      });
    });

    // Verify developer cannot see Approve button (only admins/DBA can approve)
    const approveButton = page.locator('button:has-text("Approve")').first();
    await expect(approveButton).not.toBeVisible({ timeout: 3000 });
  });

  test('Project Approver can approve change plans', async ({ page }) => {
    // Mock DBA user (has approve permissions)
    await page.route('**/api/v1/auth/me', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: {
            ...mockData.users.admin,
            organization_role: 'OrganizationDBA', // DBA can approve
          },
        }),
      });
    });

    // Mock login
    await page.route('**/api/v1/auth/login', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          user: {
            ...mockData.users.admin,
            organization_role: 'OrganizationDBA',
          },
          message: 'Login successful',
        }),
      });
    });

    // Login as DBA
    await page.goto('/login');
    await page.fill('input[type="email"]', 'dba@example.com');
    await page.fill('input[type="password"]', 'dba123');
    await page.click('button[type="submit"]');
    await page.waitForURL('/dashboard', { timeout: 10000 });

    // Mock change plan detail
    await page.route('**/api/v1/change-plans/test-change-plan-1', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'test-change-plan-1',
          title: 'Test Change Plan',
          status: 'Open',
          project_id: '1',
          target_database_id: '1',
          author_id: mockData.users.user.id,
          has_errors: false,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }),
      });
    });

    // Navigate to change plan detail
    await page.goto('/change-plans/test-change-plan-1');

    // Verify DBA can see Approve button
    const approveButton = page.locator('button:has-text("Approve")').first();
    await expect(approveButton).toBeVisible({ timeout: 5000 });
  });

  test('Unauthorized access blocked', async ({ page }) => {
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

    // Try to access protected route without authentication
    await page.goto('/dashboard');

    // Should redirect to login
    await page.waitForURL(/\/login/, { timeout: 5000 });
    await expect(page).toHaveURL(/\/login/);

    // Try to access Projects page
    await page.goto('/projects');
    await page.waitForURL(/\/login/, { timeout: 5000 });

    // Try to access Users page
    await page.goto('/users');
    await page.waitForURL(/\/login/, { timeout: 5000 });
  });
});
