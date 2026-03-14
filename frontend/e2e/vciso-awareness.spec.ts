import { test, expect } from '@playwright/test';

const uniqueProgramName = `pw-program-${Date.now()}`;

test.describe('Awareness & IAM Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/vciso/awareness');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(
      page.getByText(/Track security awareness programs and manage identity/),
    ).toBeVisible();
  });

  test('displays both tabs', async ({ page }) => {
    await expect(page.getByRole('tab', { name: /Security Awareness/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /Identity & Access Governance/i })).toBeVisible();
  });

  test('shows existing data in Security Awareness tab', async ({ page }) => {
    // Security Awareness is the default tab; check for table content
    await expect(page.locator('table')).toBeVisible({ timeout: 10_000 });
  });

  test('switches to Identity & Access Governance tab and displays KPI cards', async ({ page }) => {
    await page.getByRole('tab', { name: /Identity & Access Governance/i }).click();
    await expect(page.getByText('Privileged Accounts').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Orphaned Accounts').first()).toBeVisible();
    await expect(page.getByText('Stale Access').first()).toBeVisible();
  });

  test('opens Create Program dialog', async ({ page }) => {
    await page.getByRole('button', { name: /Create Program/i }).click();
    await expect(page.getByRole('heading', { name: 'Create Awareness Program' })).toBeVisible();
    await expect(page.locator('#program-name')).toBeVisible();
    await expect(page.locator('#program-total-users')).toBeVisible();
    await expect(page.locator('#program-start-date')).toBeVisible();
    await expect(page.locator('#program-end-date')).toBeVisible();
  });

  test('fills and submits Create Program form', async ({ page }) => {
    await page.getByRole('button', { name: /Create Program/i }).click();
    await expect(page.getByRole('heading', { name: 'Create Awareness Program' })).toBeVisible();

    await page.fill('#program-name', uniqueProgramName);
    await page.fill('#program-total-users', '500');
    await page.fill('#program-start-date', '2026-04-01');
    await page.fill('#program-end-date', '2026-12-31');

    await page.getByRole('button', { name: /^Create Program$/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 15_000 });

    // Verify new program appears
    await expect(page.getByText(uniqueProgramName).first()).toBeVisible({ timeout: 10_000 });
  });

  test('cancel closes dialog without saving', async ({ page }) => {
    await page.getByRole('button', { name: /Create Program/i }).click();
    await expect(page.getByRole('heading', { name: 'Create Awareness Program' })).toBeVisible();

    await page.fill('#program-name', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByRole('dialog')).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });
});
