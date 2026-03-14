import { test, expect } from '@playwright/test';

const uniqueEvidenceTitle = `pw-evidence-${Date.now()}`;

test.describe('Audit Evidence Repository Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/vciso/evidence');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(
      page.getByText(/Manage evidence collection/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Total Evidence').first()).toBeVisible();
    await expect(page.getByText('Needs Attention').first()).toBeVisible();
    await expect(page.getByText('Frameworks Covered').first()).toBeVisible();
    await expect(page.getByText('Controls with Evidence').first()).toBeVisible();
  });

  test('shows existing data in Evidence Repository tab', async ({ page }) => {
    // Wait for data table to load
    await expect(page.locator('table')).toBeVisible({ timeout: 10_000 });
  });

  test('switches to Collection Status tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Collection Status/i }).click();
    await expect(page.getByText('Manual Collection').first()).toBeVisible({ timeout: 10_000 });
  });

  test('opens Upload Evidence dialog', async ({ page }) => {
    await page.getByRole('button', { name: /Upload Evidence/i }).click();
    await expect(page.getByRole('heading', { name: 'Upload Evidence' })).toBeVisible();
    await expect(page.locator('#evidence-title')).toBeVisible();
    await expect(page.locator('#evidence-description')).toBeVisible();
    await expect(page.locator('#evidence-frameworks')).toBeVisible();
  });

  test('fills and submits Upload Evidence form', async ({ page }) => {
    await page.getByRole('button', { name: /Upload Evidence/i }).click();
    await expect(page.getByRole('heading', { name: 'Upload Evidence' })).toBeVisible();

    await page.fill('#evidence-title', uniqueEvidenceTitle);
    await page.fill('#evidence-description', 'E2E test evidence description');
    await page.fill('#evidence-frameworks', 'SOC 2, ISO 27001');

    await page.getByRole('button', { name: /^Upload Evidence$/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 15_000 });

    // Verify new evidence appears
    await expect(page.getByText(uniqueEvidenceTitle).first()).toBeVisible({ timeout: 10_000 });
  });

  test('cancel closes dialog without saving', async ({ page }) => {
    await page.getByRole('button', { name: /Upload Evidence/i }).click();
    await expect(page.getByRole('heading', { name: 'Upload Evidence' })).toBeVisible();

    await page.fill('#evidence-title', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByRole('dialog')).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });
});
