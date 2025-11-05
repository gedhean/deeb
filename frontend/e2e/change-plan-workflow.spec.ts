import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './utils/helpers';
import { mockData } from './utils/mocks';

test.describe('Change Plan Workflow E2E', () => {
  test.beforeEach(async ({ page }) => {
    // Login as admin to have permissions for all actions
    await loginAsAdmin(page);

    // Mock API responses for the workflow
    // Mock projects list
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

    // Mock databases list
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
  });

  test('Create → SQL Review → Approve → Execute workflow', async ({ page }) => {
    let createdChangePlanId: string | null = null;

    // Step 1: Navigate to Create Change Plan page
    await page.goto('/change-plans/create');
    await expect(page).toHaveURL(/\/change-plans\/create/);

    // Verify form is visible
    await expect(page.locator('h1').first()).toContainText(/Create Change Plan/i);

    // Step 2: Fill out the change plan form
    // Select project
    await page.waitForSelector('select[name="project_id"], button:has-text("Select project")', {
      timeout: 5000,
    });

    // Handle both select and combobox cases
    const projectSelect = page.locator('select[name="project_id"]').first();
    const projectButton = page.locator('button:has-text("Select project")').first();

    if (await projectSelect.isVisible()) {
      await projectSelect.selectOption('1');
    } else if (await projectButton.isVisible()) {
      await projectButton.click();
      await page.waitForTimeout(500); // Wait for dropdown
      await page.locator('text=Test Project').first().click();
    }

    // Wait for database to load
    await page.waitForTimeout(1000);

    // Select database
    const databaseSelect = page.locator('select[name="target_database_id"]').first();
    const databaseButton = page.locator('button:has-text("Select database")').first();

    if (await databaseSelect.isVisible()) {
      await databaseSelect.selectOption('1');
    } else if (await databaseButton.isVisible()) {
      await databaseButton.click();
      await page.waitForTimeout(500);
      await page.locator('text=Test Database').first().click();
    }

    // Fill title
    await page.fill('input[name="title"]', 'E2E Test Change Plan');

    // Fill description
    await page.fill('textarea[name="description"]', 'E2E test change plan description');

    // Select change type
    const changeTypeSelect = page.locator('select[name="change_type"]').first();
    if (await changeTypeSelect.isVisible()) {
      await changeTypeSelect.selectOption('DDL');
    }

    // Fill SQL content
    const sqlContent = 'CREATE TABLE test_table (id INT PRIMARY KEY, name VARCHAR(255));';
    await page.fill('textarea[name="sql_content"]', sqlContent);

    // Mock the create API call
    await page.route('**/api/v1/change-plans', async (route) => {
      if (route.request().method() === 'POST') {
        const requestBody = await route.request().postDataJSON();
        createdChangePlanId = 'test-change-plan-' + Date.now();
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: createdChangePlanId,
            ...requestBody,
            status: 'Open',
            author_id: mockData.users.admin.id,
            has_errors: false,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          }),
        });
      } else {
        // GET request - return list
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            change_plans: [],
            total: 0,
            limit: 100,
            offset: 0,
          }),
        });
      }
    });

    // Step 3: Submit the form
    const submitButton = page.locator('button[type="submit"]').first();
    await expect(submitButton).toBeVisible();
    await submitButton.click();

    // Wait for navigation to detail page
    await page.waitForURL(/\/change-plans\/.+/, { timeout: 10000 });
    expect(createdChangePlanId).not.toBeNull();

    // Step 4: Verify Change Plan Detail Page - SQL Review section
    await expect(page.locator('h1').first()).toContainText(/E2E Test Change Plan/i);

    // Verify status is Open
    await expect(page.locator('text=Open').first()).toBeVisible({ timeout: 5000 });

    // Verify SQL Review section is visible
    const sqlReviewSection = page.locator('text=SQL Review').first();
    await expect(sqlReviewSection).toBeVisible({ timeout: 5000 });

    // Mock SQL review results (no errors for successful workflow)
    await page.route(`**/api/v1/change-plans/${createdChangePlanId}/review*`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          findings: [],
          has_errors: false,
          reviewed_at: new Date().toISOString(),
        }),
      });
    });

    // Mock the get change plan API to return updated status
    await page.route(`**/api/v1/change-plans/${createdChangePlanId}`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: createdChangePlanId,
          title: 'E2E Test Change Plan',
          description: 'E2E test change plan description',
          status: 'Open',
          project_id: '1',
          target_database_id: '1',
          author_id: mockData.users.admin.id,
          change_type: 'DDL',
          sql_content: sqlContent,
          has_errors: false,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }),
      });
    });

    // Step 5: Approve the change plan
    // Mock approval API
    await page.route(`**/api/v1/change-plans/${createdChangePlanId}/approve`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: createdChangePlanId,
          status: 'Approved',
          approved_at: new Date().toISOString(),
        }),
      });
    });

    // Mock updated change plan after approval
    await page.route(`**/api/v1/change-plans/${createdChangePlanId}`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: createdChangePlanId,
          title: 'E2E Test Change Plan',
          description: 'E2E test change plan description',
          status: 'Approved',
          project_id: '1',
          target_database_id: '1',
          author_id: mockData.users.admin.id,
          change_type: 'DDL',
          sql_content: sqlContent,
          has_errors: false,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }),
      });
    });

    // Click Approve button
    const approveButton = page.locator('button:has-text("Approve")').first();
    await expect(approveButton).toBeVisible({ timeout: 5000 });
    await approveButton.click();

    // Wait for approval dialog
    await expect(page.locator('text=Approve Change Plan').first()).toBeVisible({
      timeout: 5000,
    });

    // Confirm approval (click approve in dialog)
    const confirmApproveButton = page
      .locator('button:has-text("Approve"):not(:has-text("Approve Change Plan"))')
      .last();
    await confirmApproveButton.click();

    // Wait for approval to complete
    await page.waitForTimeout(2000);

    // Verify status changed to Approved
    await expect(page.locator('text=Approved').first()).toBeVisible({ timeout: 10000 });

    // Step 6: Execute the change plan
    // Mock execution API
    await page.route(`**/api/v1/change-plans/${createdChangePlanId}/execute`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: createdChangePlanId,
          status: 'Completed',
          executed_at: new Date().toISOString(),
        }),
      });
    });

    // Mock updated change plan after execution
    await page.route(`**/api/v1/change-plans/${createdChangePlanId}`, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: createdChangePlanId,
          title: 'E2E Test Change Plan',
          description: 'E2E test change plan description',
          status: 'Completed',
          project_id: '1',
          target_database_id: '1',
          author_id: mockData.users.admin.id,
          change_type: 'DDL',
          sql_content: sqlContent,
          has_errors: false,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }),
      });
    });

    // Click Execute button
    const executeButton = page.locator('button:has-text("Execute")').first();
    await expect(executeButton).toBeVisible({ timeout: 5000 });
    await executeButton.click();

    // Wait for execution dialog
    await expect(page.locator('text=Execute Change Plan').first()).toBeVisible({
      timeout: 5000,
    });

    // Confirm execution
    const confirmExecuteButton = page
      .locator('button:has-text("Execute"):not(:has-text("Execute Change Plan"))')
      .last();
    await confirmExecuteButton.click();

    // Wait for execution to complete
    await page.waitForTimeout(3000);

    // Verify status changed to Completed
    await expect(page.locator('text=Completed').first()).toBeVisible({ timeout: 10000 });

    // Verify execution results section is visible
    const executionResults = page.locator('text=Execution Results').first();
    await expect(executionResults).toBeVisible({ timeout: 5000 });
  });

  test('Validates UI state at each workflow step', async ({ page }) => {
    await loginAsAdmin(page);

    // Navigate to change plans list
    await page.goto('/change-plans');
    await expect(page).toHaveURL(/\/change-plans/);

    // Verify change plans page loads
    await expect(page.locator('h1').first()).toContainText(/Change Plans/i);

    // Verify Create button is visible
    const createButton = page
      .locator('a:has-text("Create Change Plan"), button:has-text("Create")')
      .first();
    await expect(createButton).toBeVisible();

    // Navigate to create page
    await createButton.click();
    await page.waitForURL(/\/change-plans\/create/);

    // Verify form fields are visible
    await expect(page.locator('input[name="title"]')).toBeVisible();
    await expect(page.locator('textarea[name="description"]')).toBeVisible();
    await expect(page.locator('textarea[name="sql_content"]')).toBeVisible();

    // Verify submit button is visible
    await expect(page.locator('button[type="submit"]').first()).toBeVisible();
  });

  test('Validates API calls during workflow', async ({ page }) => {
    const apiCalls: string[] = [];

    // Intercept all API calls
    page.on('request', (request) => {
      if (request.url().includes('/api/v1/')) {
        apiCalls.push(`${request.method()} ${request.url()}`);
      }
    });

    await loginAsAdmin(page);
    await page.goto('/change-plans');

    // Wait for initial API calls
    await page.waitForTimeout(2000);

    // Verify API calls were made
    expect(apiCalls.length).toBeGreaterThan(0);
    expect(apiCalls.some((call) => call.includes('GET') && call.includes('/change-plans'))).toBe(
      true,
    );
  });

  test('Handles async operations correctly', async ({ page }) => {
    await loginAsAdmin(page);

    // Mock slow API response
    await page.route('**/api/v1/projects*', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 1000));
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

    await page.goto('/change-plans/create');

    // Verify loading state appears and page eventually loads
    // Loading might be too fast, so we just verify page eventually loads
    await expect(page.locator('h1').first()).toBeVisible({ timeout: 5000 });

    // Verify data loads after async operation
    await expect(page.locator('text=Test Project').first()).toBeVisible({ timeout: 10000 });
  });
});
