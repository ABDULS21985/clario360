import { test, expect } from '@playwright/test';

const uniqueObligationName = `pw-obligation-${Date.now()}`;

test.describe('Compliance Management Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/vciso/compliance');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(
      page.getByText(/Track regulatory obligations/),
    ).toBeVisible();
  });

  test('shows existing data in Regulatory Obligations tab', async ({ page }) => {
    // Wait for data table to load
    await expect(page.locator('table')).toBeVisible({ timeout: 10_000 });
  });

  test('switches to Control Testing tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Control Testing/i }).click();
    await expect(page.getByText(/control test/i).first()).toBeVisible({ timeout: 10_000 });
  });

  test('switches to Control Dependencies tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Control Dependencies/i }).click();
    await expect(page.getByText(/dependenc/i).first()).toBeVisible({ timeout: 10_000 });
  });

  test('opens Add Obligation dialog', async ({ page }) => {
    await page.getByRole('button', { name: /Add Obligation/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Obligation' })).toBeVisible();
    await expect(page.locator('#obligation-name')).toBeVisible();
    await expect(page.locator('#obligation-jurisdiction')).toBeVisible();
    await expect(page.locator('#obligation-description')).toBeVisible();
  });

  test('fills and submits Add Obligation form', async ({ page }) => {
    await page.getByRole('button', { name: /Add Obligation/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Obligation' })).toBeVisible();

    await page.fill('#obligation-name', uniqueObligationName);
    await page.fill('#obligation-jurisdiction', 'United States');
    await page.fill('#obligation-description', 'E2E test regulatory obligation');

    // Select type via dropdown
    await page.locator('#obligation-type').click();
    await page.getByRole('option', { name: 'Regulatory' }).click();

    await page.getByRole('button', { name: /^Add Obligation$/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 15_000 });

    // Verify new obligation appears
    await expect(page.getByText(uniqueObligationName).first()).toBeVisible({ timeout: 10_000 });
  });

  test('cancel closes Add Obligation dialog without saving', async ({ page }) => {
    await page.getByRole('button', { name: /Add Obligation/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Obligation' })).toBeVisible();

    await page.fill('#obligation-name', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByRole('dialog')).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });

  test('opens Record Control Test dialog from Control Testing tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Control Testing/i }).click();

    // Wait for the tab content to load
    await expect(page.getByRole('tabpanel')).toBeVisible({ timeout: 10_000 });

    await page.getByRole('button', { name: /Record Test/i }).click();
    await expect(page.getByRole('heading', { name: 'Record Control Test' })).toBeVisible();
    await expect(page.locator('#ct-control-name')).toBeVisible();
    await expect(page.locator('#ct-framework')).toBeVisible();
  });
});
