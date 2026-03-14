import { test, expect } from '@playwright/test';

const uniqueControlName = `pw-control-${Date.now()}`;

test.describe('Workflows Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/vciso/workflows');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(
      page.getByText(/Manage control ownership assignments and approval workflows/),
    ).toBeVisible();
  });

  test('displays both tabs', async ({ page }) => {
    await expect(page.getByRole('tab', { name: /Control Ownership/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /Approval Queue/i })).toBeVisible();
  });

  test('shows existing data in Control Ownership tab', async ({ page }) => {
    // Control Ownership is the default tab
    await expect(page.locator('table')).toBeVisible({ timeout: 10_000 });
  });

  test('switches to Approval Queue tab and displays KPI cards', async ({ page }) => {
    await page.getByRole('tab', { name: /Approval Queue/i }).click();
    await expect(page.getByText('Pending Approvals').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Overdue').first()).toBeVisible();
    await expect(page.getByText('Approved This Month').first()).toBeVisible();
    await expect(page.getByText('Rejected This Month').first()).toBeVisible();
  });

  test('opens Assign Ownership dialog', async ({ page }) => {
    await page.getByRole('button', { name: /Assign Ownership/i }).click();
    await expect(page.getByRole('heading', { name: 'Assign Control Ownership' })).toBeVisible();
    await expect(page.locator('#ownership-control-id')).toBeVisible();
    await expect(page.locator('#ownership-framework')).toBeVisible();
    await expect(page.locator('#ownership-control-name')).toBeVisible();
    await expect(page.locator('#ownership-owner-id')).toBeVisible();
    await expect(page.locator('#ownership-owner-name')).toBeVisible();
    await expect(page.locator('#ownership-review-date')).toBeVisible();
  });

  test('fills and submits Assign Ownership form', async ({ page }) => {
    await page.getByRole('button', { name: /Assign Ownership/i }).click();
    await expect(page.getByRole('heading', { name: 'Assign Control Ownership' })).toBeVisible();

    await page.fill('#ownership-control-id', 'AC-001');
    await page.fill('#ownership-framework', 'NIST 800-53');
    await page.fill('#ownership-control-name', uniqueControlName);
    await page.fill('#ownership-owner-id', 'user-001');
    await page.fill('#ownership-owner-name', 'Test Owner');
    await page.fill('#ownership-review-date', '2026-09-30');

    await page.getByRole('button', { name: /^Assign Ownership$/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 15_000 });

    // Verify new ownership assignment appears
    await expect(page.getByText(uniqueControlName).first()).toBeVisible({ timeout: 10_000 });
  });

  test('cancel closes dialog without saving', async ({ page }) => {
    await page.getByRole('button', { name: /Assign Ownership/i }).click();
    await expect(page.getByRole('heading', { name: 'Assign Control Ownership' })).toBeVisible();

    await page.fill('#ownership-control-name', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByRole('dialog')).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });
});
