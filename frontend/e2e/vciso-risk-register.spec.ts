import { test, expect } from '@playwright/test';

const uniqueRiskTitle = `pw-risk-${Date.now()}`;

test.describe('Risk Register Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/vciso/risk-register');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(
      page.getByText(/Identify, assess, and manage organizational risks/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Total Risks').first()).toBeVisible();
    await expect(page.getByText('Avg Residual Score').first()).toBeVisible();
    await expect(page.getByText('Overdue Reviews').first()).toBeVisible();
    await expect(page.getByText('Accepted Risks').first()).toBeVisible();
  });

  test('shows existing data in Risk Register tab', async ({ page }) => {
    // Wait for data table to load with seeded data
    await expect(page.locator('table')).toBeVisible({ timeout: 10_000 });
  });

  test('switches to Risk Acceptance tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Risk Acceptance/i }).click();
    await expect(page.getByText(/accepted risks/i).first()).toBeVisible({ timeout: 10_000 });
  });

  test('switches to Business Impact tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Business Impact/i }).click();
    await expect(page.getByText('Risk Heat Matrix').first()).toBeVisible({ timeout: 10_000 });
  });

  test('opens create dialog', async ({ page }) => {
    await page.getByRole('button', { name: /Add Risk/i }).click();
    await expect(page.getByRole('heading', { name: 'Add New Risk' })).toBeVisible();
    await expect(page.locator('#risk-title')).toBeVisible();
    await expect(page.locator('#risk-description')).toBeVisible();
    await expect(page.locator('#risk-category')).toBeVisible();
    await expect(page.locator('#risk-department')).toBeVisible();
  });

  test('fills and submits create form', async ({ page }) => {
    await page.getByRole('button', { name: /Add Risk/i }).click();
    await expect(page.getByRole('heading', { name: 'Add New Risk' })).toBeVisible();

    await page.fill('#risk-title', uniqueRiskTitle);
    await page.fill('#risk-description', 'E2E test risk description');
    await page.fill('#risk-category', 'Operational');
    await page.fill('#risk-department', 'Engineering');

    await page.getByRole('button', { name: /Create Risk/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByRole('heading', { name: 'Add New Risk' })).not.toBeVisible({ timeout: 15_000 });

    // Verify new risk appears
    await expect(page.getByText(uniqueRiskTitle).first()).toBeVisible({ timeout: 10_000 });
  });

  test('cancel closes dialog without saving', async ({ page }) => {
    await page.getByRole('button', { name: /Add Risk/i }).click();
    await expect(page.getByRole('heading', { name: 'Add New Risk' })).toBeVisible();

    await page.fill('#risk-title', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByRole('heading', { name: 'Add New Risk' })).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });
});
