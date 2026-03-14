import { test, expect } from '@playwright/test';

const uniqueIntegrationName = `pw-integration-${Date.now()}`;

test.describe('Integrations Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/vciso/integrations');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(
      page.getByText(/Manage connections to external security tools/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Total Integrations').first()).toBeVisible();
    await expect(page.getByText('Connected').first()).toBeVisible();
    await expect(page.getByText('Errors').first()).toBeVisible();
    await expect(page.getByText('Total Items Synced').first()).toBeVisible();
  });

  test('shows existing data as integration cards', async ({ page }) => {
    // Integrations page uses a card grid layout rather than a data table
    await expect(page.locator('[class*="grid"]').first()).toBeVisible({ timeout: 10_000 });
  });

  test('opens Add Integration dialog', async ({ page }) => {
    await page.getByRole('button', { name: /Add Integration/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Integration' })).toBeVisible();
    await expect(page.locator('#integration-name')).toBeVisible();
    await expect(page.locator('#integration-provider')).toBeVisible();
    await expect(page.locator('#integration-config')).toBeVisible();
  });

  test('fills and submits Add Integration form', async ({ page }) => {
    await page.getByRole('button', { name: /Add Integration/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Integration' })).toBeVisible();

    await page.fill('#integration-name', uniqueIntegrationName);
    await page.fill('#integration-provider', 'CrowdStrike');
    await page.fill('#integration-config', '{"api_key": "test-key"}');

    await page.getByRole('button', { name: /^Add Integration$/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 15_000 });

    // Verify new integration appears
    await expect(page.getByText(uniqueIntegrationName).first()).toBeVisible({ timeout: 10_000 });
  });

  test('cancel closes dialog without saving', async ({ page }) => {
    await page.getByRole('button', { name: /Add Integration/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Integration' })).toBeVisible();

    await page.fill('#integration-name', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByRole('dialog')).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });
});
