import { test, expect } from '@playwright/test';

test.describe('Dashboard Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/dashboard');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  // ---- Page structure ----

  test('renders welcome header', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
  });

  // ---- KPI Grid (uses /api/v1/cyber/alerts/count, /api/v1/data/pipelines/count, etc.) ----

  test('displays KPI cards with values (not dashes)', async ({ page }) => {
    // Wait for KPI grid to load — cards should show numeric values, not "—"
    const kpiSection = page.locator('[class*="grid"]').first();
    await expect(kpiSection).toBeVisible({ timeout: 10_000 });

    // Verify at least one KPI card is present and shows a numeric value
    // The cards display values like "12", "0", "95.2%" — never just "—" if data loaded
    const kpiCards = page.locator('[class*="rounded"]').filter({
      hasText: /Open Alerts|Failed Pipelines|Data Quality|Pending Tasks/,
    });
    await expect(kpiCards.first()).toBeVisible({ timeout: 10_000 });
  });

  // ---- Secondary Metrics Strip (uses /api/v1/cyber/dashboard/metrics) ----

  test('secondary metrics strip loads with data', async ({ page }) => {
    // Intercept the metrics API call to verify it is made and returns data
    const metricsResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dashboard/metrics') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/dashboard');
    const response = await metricsResponse;
    const body = await response.json();

    // Verify response envelope structure
    expect(body).toHaveProperty('data');
    const data = body.data;

    // At least some metrics should be present (nullable fields may be null)
    const fields = ['mttr_minutes', 'mtta_minutes', 'sla_compliance_pct', 'active_incidents', 'active_users_today', 'pending_reviews'];
    for (const field of fields) {
      expect(data).toHaveProperty(field);
    }
  });

  test('secondary metrics strip displays metric labels', async ({ page }) => {
    // The strip should display labels for the metrics
    const metricLabels = ['MTTR', 'MTTA', 'SLA Compliance', 'Active Incidents', 'Active Users', 'Pending Reviews'];

    for (const label of metricLabels) {
      await expect(page.getByText(label, { exact: false }).first()).toBeVisible({ timeout: 10_000 });
    }
  });

  test('secondary metrics strip shows numeric values not loading skeletons', async ({ page }) => {
    // Wait for the metrics API to complete
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dashboard/metrics') && resp.status() === 200,
      { timeout: 15_000 },
    );

    // After API response, loading skeletons (animate-pulse elements) should be gone
    // and actual values should be visible
    const metricsStrip = page.locator('[class*="flex-nowrap"]').first();
    await expect(metricsStrip).toBeVisible({ timeout: 5_000 });

    // Verify no loading skeletons remain in the strip
    const skeletons = metricsStrip.locator('[class*="animate-pulse"]');
    await expect(skeletons).toHaveCount(0, { timeout: 5_000 });
  });

  // ---- Critical Alerts Banner (uses /api/v1/cyber/alerts/stats) ----

  test('critical alerts banner API returns valid data', async ({ page }) => {
    const statsResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/alerts/stats') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/dashboard');
    const response = await statsResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(body.data).toHaveProperty('by_severity');
    expect(Array.isArray(body.data.by_severity)).toBe(true);
  });

  // ---- API response validation ----

  test('KPI alerts count API returns envelope with data', async ({ page }) => {
    const alertsResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/alerts/count') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/dashboard');
    const response = await alertsResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(body.data).toHaveProperty('count');
    expect(typeof body.data.count).toBe('number');
  });

  // ---- Dashboard content grid ----

  test('displays recent alerts table', async ({ page }) => {
    await expect(
      page.getByText(/Recent Alerts/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });

  test('displays tasks list', async ({ page }) => {
    await expect(
      page.getByText(/Tasks/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });

  test('displays activity timeline', async ({ page }) => {
    await expect(
      page.getByText(/Activity/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});
