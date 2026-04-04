import { test, expect } from '@playwright/test';

const uniqueBudgetTitle = `pw-budget-${Date.now()}`;

test.describe('Maturity & Budget Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/vciso/maturity');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(
      page.getByText(/Assess your security maturity, benchmark against industry peers/),
    ).toBeVisible();
  });

  test('displays all three tabs', async ({ page }) => {
    await expect(page.getByRole('tab', { name: /Maturity Assessment/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /Benchmarking/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /Security Budget/i })).toBeVisible();
  });

  test('shows Maturity Assessment tab content by default', async ({ page }) => {
    // Maturity Assessment is the default tab
    await expect(page.getByRole('tabpanel')).toBeVisible({ timeout: 10_000 });
  });

  test('switches to Benchmarking tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Benchmarking/i }).click();
    await expect(page.getByRole('tabpanel')).toBeVisible({ timeout: 10_000 });
  });

  test('switches to Security Budget tab and displays KPI cards', async ({ page }) => {
    await page.getByRole('tab', { name: /Security Budget/i }).click();
    await expect(page.getByText('Total Proposed').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Total Approved').first()).toBeVisible();
    await expect(page.getByText('Total Spent').first()).toBeVisible();
    await expect(page.getByText('Risk Reduction').first()).toBeVisible();
  });

  test('shows existing data in Security Budget tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Security Budget/i }).click();
    await expect(page.locator('table')).toBeVisible({ timeout: 10_000 });
  });

  test('opens Add Budget Item dialog', async ({ page }) => {
    await page.getByRole('tab', { name: /Security Budget/i }).click();
    await expect(page.getByText('Total Proposed').first()).toBeVisible({ timeout: 10_000 });

    await page.getByRole('button', { name: /Add Budget Item/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Budget Item' })).toBeVisible();
    await expect(page.locator('#budget-title')).toBeVisible();
    await expect(page.locator('#budget-amount')).toBeVisible();
    await expect(page.locator('#budget-justification')).toBeVisible();
  });

  test('fills and submits Add Budget Item form', async ({ page }) => {
    await page.getByRole('tab', { name: /Security Budget/i }).click();
    await expect(page.getByText('Total Proposed').first()).toBeVisible({ timeout: 10_000 });

    await page.getByRole('button', { name: /Add Budget Item/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Budget Item' })).toBeVisible();

    await page.fill('#budget-title', uniqueBudgetTitle);
    await page.fill('#budget-amount', '50000');
    await page.fill('#budget-justification', 'E2E test budget justification for security investment');

    // Select category
    await page.locator('button:has-text("Select category")').click();
    await page.getByRole('option', { name: 'Endpoint Security' }).click();

    await page.getByRole('button', { name: /^Create Budget Item$/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 15_000 });

    // Verify new budget item appears
    await expect(page.getByText(uniqueBudgetTitle).first()).toBeVisible({ timeout: 10_000 });
  });

  test('cancel closes dialog without saving', async ({ page }) => {
    await page.getByRole('tab', { name: /Security Budget/i }).click();
    await expect(page.getByText('Total Proposed').first()).toBeVisible({ timeout: 10_000 });

    await page.getByRole('button', { name: /Add Budget Item/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Budget Item' })).toBeVisible();

    await page.fill('#budget-title', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByRole('dialog')).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });
});
