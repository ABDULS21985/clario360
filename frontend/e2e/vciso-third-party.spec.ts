import { test, expect } from '@playwright/test';

const uniqueVendorName = `pw-vendor-${Date.now()}`;

test.describe('Third-Party Risk Management Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/vciso/third-party');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(
      page.getByText(/Monitor vendor risk/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Total Vendors').first()).toBeVisible();
    await expect(page.getByText('Critical Vendors').first()).toBeVisible();
    await expect(page.getByText('Pending Reviews').first()).toBeVisible();
    await expect(page.getByText('Open Questionnaires').first()).toBeVisible();
  });

  test('shows existing data in Vendors tab', async ({ page }) => {
    // Wait for data table to load
    await expect(page.locator('table')).toBeVisible({ timeout: 10_000 });
  });

  test('switches to Questionnaires tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Questionnaires/i }).click();
    await expect(page.getByText(/questionnaire/i).first()).toBeVisible({ timeout: 10_000 });
  });

  test('opens Add Vendor dialog', async ({ page }) => {
    await page.getByRole('button', { name: /Add Vendor/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Vendor' })).toBeVisible();
    await expect(page.locator('#vendor-name')).toBeVisible();
    await expect(page.locator('#vendor-contact-name')).toBeVisible();
    await expect(page.locator('#vendor-contact-email')).toBeVisible();
    await expect(page.locator('#vendor-next-review')).toBeVisible();
  });

  test('fills and submits Add Vendor form', async ({ page }) => {
    await page.getByRole('button', { name: /Add Vendor/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Vendor' })).toBeVisible();

    await page.fill('#vendor-name', uniqueVendorName);
    await page.fill('#vendor-contact-name', 'Test Contact');
    await page.fill('#vendor-contact-email', 'test@example.com');
    await page.fill('#vendor-next-review', '2027-01-15');

    // Select category via dropdown
    await page.locator('#vendor-category').click();
    await page.getByRole('option', { name: 'Cloud Infrastructure' }).click();

    await page.getByRole('button', { name: /^Add Vendor$/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 15_000 });

    // Verify new vendor appears
    await expect(page.getByText(uniqueVendorName).first()).toBeVisible({ timeout: 10_000 });
  });

  test('cancel closes dialog without saving', async ({ page }) => {
    await page.getByRole('button', { name: /Add Vendor/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Vendor' })).toBeVisible();

    await page.fill('#vendor-name', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByRole('dialog')).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });
});
