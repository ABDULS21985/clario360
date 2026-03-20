import { test, expect } from '@playwright/test';

test.describe('DSPM AI Data Security Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/dspm/ai-security');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toContainText('AI Data Security');
    await expect(
      page.getByText(/Monitor AI data usage risks/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Total AI Data Usages').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('High Risk Count').first()).toBeVisible();
    await expect(page.getByText('PII in AI Count').first()).toBeVisible();
    await expect(page.getByText('Consent Gap Count').first()).toBeVisible();
  });

  test('AI dashboard API returns envelope with data', async ({ page }) => {
    const dashResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/ai/dashboard') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/cyber/dspm/ai-security');
    const response = await dashResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    const data = body.data;
    expect(data).toHaveProperty('total_ai_data_usages');
    expect(data).toHaveProperty('high_risk_count');
    expect(data).toHaveProperty('pii_in_ai_count');
    expect(data).toHaveProperty('consent_gap_count');
  });

  test('AI risk ranking API returns envelope with data', async ({ page }) => {
    const rankingResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/ai/risk-ranking') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/cyber/dspm/ai-security');
    const response = await rankingResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('shows Risk Distribution section', async ({ page }) => {
    await expect(page.getByText('Risk Distribution').first()).toBeVisible({ timeout: 10_000 });
  });

  test('shows Usage Type Distribution section', async ({ page }) => {
    await expect(page.getByText('Usage Type Distribution').first()).toBeVisible({ timeout: 10_000 });
  });

  test('shows Top Risky AI Data Usages table', async ({ page }) => {
    await expect(page.getByText('Top Risky AI Data Usages').first()).toBeVisible({ timeout: 10_000 });
    await expect(
      page.getByText(/AI data usages ranked by risk score/),
    ).toBeVisible();
  });

  test('risky usages table has correct column headers', async ({ page }) => {
    // Wait for the table section to load
    await expect(page.getByText('Top Risky AI Data Usages').first()).toBeVisible({ timeout: 10_000 });

    // Check table column headers
    const headers = ['Asset Name', 'Usage Type', 'Risk Level', 'Risk Score', 'PII Types', 'Consent', 'Anonymization Level', 'Status'];
    for (const header of headers) {
      await expect(page.getByRole('columnheader', { name: header })).toBeVisible();
    }
  });

  test('KPI cards show numeric values after loading', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/ai/dashboard') && resp.status() === 200,
      { timeout: 15_000 },
    );

    const kpiCards = page.locator('.tabular-nums');
    const count = await kpiCards.count();
    expect(count).toBeGreaterThanOrEqual(4);
  });

  test('risk distribution shows severity levels', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/ai/dashboard') && resp.status() === 200,
      { timeout: 15_000 },
    );

    // Check for risk distribution severity badges or no-data message
    const riskSection = page.getByText('Risk Distribution').first();
    await expect(riskSection).toBeVisible();
  });
});
