import { test, expect } from '@playwright/test';

const uniquePolicyTitle = `pw-policy-${Date.now()}`;

test.describe('Policy Management Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/vciso/policies');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(
      page.getByText(/Manage security policies/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Total Policies').first()).toBeVisible();
    await expect(page.getByText('Published', { exact: true }).first()).toBeVisible();
    await expect(page.getByText('In Review').first()).toBeVisible();
    await expect(page.getByText('Overdue Reviews').first()).toBeVisible();
    await expect(page.getByText('Active Exceptions').first()).toBeVisible();
  });

  test('shows existing data in Policies tab', async ({ page }) => {
    // Wait for data table to load
    await expect(page.locator('table')).toBeVisible({ timeout: 10_000 });
  });

  test('switches to Exceptions tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Exceptions/i }).click();
    await expect(page.getByText(/exception/i).first()).toBeVisible({ timeout: 10_000 });
  });

  test('switches to AI Draft tab', async ({ page }) => {
    await page.getByRole('tab', { name: /AI Draft/i }).click();
    // AI Draft tab should have the policy draft generator content
    await expect(page.getByRole('tabpanel')).toBeVisible({ timeout: 10_000 });
  });

  test('opens create dialog', async ({ page }) => {
    await page.getByRole('button', { name: /Create Policy/i }).click();
    await expect(page.getByRole('heading', { name: 'Create Policy' })).toBeVisible();
    await expect(page.locator('#policy-title')).toBeVisible();
    await expect(page.locator('#policy-content')).toBeVisible();
  });

  test('fills and submits create form', async ({ page }) => {
    await page.getByRole('button', { name: /Create Policy/i }).click();
    await expect(page.getByRole('heading', { name: 'Create Policy' })).toBeVisible();

    await page.fill('#policy-title', uniquePolicyTitle);

    // Select domain via the select dropdown
    await page.locator('#policy-domain').click();
    await page.getByRole('option', { name: 'Access Control' }).click();

    await page.fill('#policy-content', 'E2E test policy content for automated testing.');

    await page.getByRole('button', { name: /^Create Policy$/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 15_000 });

    // Verify new policy appears
    await expect(page.getByText(uniquePolicyTitle).first()).toBeVisible({ timeout: 10_000 });
  });

  test('cancel closes dialog without saving', async ({ page }) => {
    await page.getByRole('button', { name: /Create Policy/i }).click();
    await expect(page.getByRole('heading', { name: 'Create Policy' })).toBeVisible();

    await page.fill('#policy-title', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByRole('dialog')).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });
});
