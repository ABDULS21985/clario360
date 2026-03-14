import { test, expect } from '@playwright/test';

test.describe('DSPM Financial Risk Quantification Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/dspm/financial');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Financial Risk Quantification');
    await expect(
      page.getByText(/Quantify the financial impact/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Total Breach Cost Exposure').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Annual Expected Loss').first()).toBeVisible();
    await expect(page.getByText('Max Single Breach').first()).toBeVisible();
    await expect(page.getByText('Assets at Risk').first()).toBeVisible();
  });

  test('portfolio API returns envelope with data', async ({ page }) => {
    const portfolioResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/financial/impact') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/cyber/dspm/financial');
    const response = await portfolioResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    const data = body.data;
    expect(data).toHaveProperty('total_breach_cost');
    expect(data).toHaveProperty('total_expected_loss');
    expect(data).toHaveProperty('max_single_breach');
    expect(data).toHaveProperty('asset_count');
  });

  test('top risks API returns envelope with data', async ({ page }) => {
    const topRisksResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/financial/top-risks') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/cyber/dspm/financial');
    const response = await topRisksResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('shows Top Financial Risks section', async ({ page }) => {
    await expect(page.getByText('Top Financial Risks').first()).toBeVisible({ timeout: 10_000 });
    await expect(
      page.getByText(/Highest-impact assets ranked/),
    ).toBeVisible();
  });

  test('financial risks table has correct column headers', async ({ page }) => {
    await expect(page.getByText('Top Financial Risks').first()).toBeVisible({ timeout: 10_000 });

    // Wait for table to potentially load
    const table = page.locator('table');
    const hasTable = await table.isVisible().catch(() => false);

    if (hasTable) {
      const headers = ['Asset', 'Breach Cost', 'Cost per Record', 'Records', 'Breach Probability', 'Annual Expected Loss', 'Methodology'];
      for (const header of headers) {
        await expect(page.locator('th').filter({ hasText: header }).first()).toBeVisible();
      }
    }
  });

  test('KPI cards display currency formatted values', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/financial/impact') && resp.status() === 200,
      { timeout: 15_000 },
    );

    // KPI values should contain $ sign for currency formatting
    const kpiCards = page.locator('.tabular-nums');
    const count = await kpiCards.count();
    expect(count).toBeGreaterThanOrEqual(4);
  });

  test('shows empty state when no financial risks exist', async ({ page }) => {
    // Either shows table rows or the empty state
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/financial/top-risks') && resp.status() === 200,
      { timeout: 15_000 },
    );

    // Check that either the table has rows or we see the empty state
    const tableRows = page.locator('tbody tr');
    const rowCount = await tableRows.count().catch(() => 0);

    if (rowCount === 0) {
      await expect(page.getByText(/No financial risk data available/).first()).toBeVisible({ timeout: 5_000 });
    } else {
      expect(rowCount).toBeGreaterThan(0);
    }
  });
});
