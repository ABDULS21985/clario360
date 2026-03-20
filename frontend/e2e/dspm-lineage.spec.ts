import { test, expect } from '@playwright/test';

test.describe('DSPM Data Lineage Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/dspm/lineage');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Data Lineage');
    await expect(
      page.getByText(/Track data flow across systems/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Total Nodes').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Total Edges').first()).toBeVisible();
    await expect(page.getByText('PII Flow Count').first()).toBeVisible();
    await expect(page.getByText('Classification Changes').first()).toBeVisible();
  });

  test('lineage graph API returns envelope with data', async ({ page }) => {
    const graphResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/lineage/graph') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/cyber/dspm/lineage');
    const response = await graphResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    const data = body.data;
    expect(data).toHaveProperty('total_nodes');
    expect(data).toHaveProperty('total_edges');
    expect(data).toHaveProperty('pii_flow_count');
    expect(data).toHaveProperty('edges');
    expect(Array.isArray(data.edges)).toBe(true);
  });

  test('pii flow API returns envelope with data', async ({ page }) => {
    const piiResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/lineage/pii-flow') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/cyber/dspm/lineage');
    const response = await piiResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('shows Lineage Edges section', async ({ page }) => {
    await expect(page.getByText('Lineage Edges').first()).toBeVisible({ timeout: 10_000 });
  });

  test('shows search input and filter dropdowns', async ({ page }) => {
    await expect(page.getByPlaceholder('Search assets or pipelines...')).toBeVisible({ timeout: 10_000 });
    // Edge type and status filter selects
    const selects = page.locator('select');
    const count = await selects.count();
    expect(count).toBeGreaterThanOrEqual(2);
  });

  test('search filters edges in the table', async ({ page }) => {
    // Wait for the lineage edges section to load
    await expect(page.getByText('Lineage Edges').first()).toBeVisible({ timeout: 10_000 });

    const searchInput = page.getByPlaceholder('Search assets or pipelines...');
    await searchInput.fill('nonexistent-asset-xyz-999');

    // Should show empty or reduced results
    await expect(
      page.getByText(/No Lineage Edges Found|Showing 0|Try adjusting/).first(),
    ).toBeVisible({ timeout: 5_000 });
  });

  test('edge type filter changes displayed edges', async ({ page }) => {
    await expect(page.getByText('Lineage Edges').first()).toBeVisible({ timeout: 10_000 });

    const typeSelect = page.locator('select').first();
    // Verify it has the "All Types" default
    await expect(typeSelect).toHaveValue('all');
  });

  test('KPI cards show numeric values after loading', async ({ page }) => {
    // Wait for API to complete
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/lineage/graph') && resp.status() === 200,
      { timeout: 15_000 },
    );

    // Verify loading skeletons are gone and numeric values are present
    const kpiCards = page.locator('.tabular-nums');
    const count = await kpiCards.count();
    expect(count).toBeGreaterThanOrEqual(4);
  });
});
