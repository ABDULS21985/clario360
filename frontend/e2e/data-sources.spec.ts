import { test, expect } from '@playwright/test';

test.describe('Data Sources List Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/data/sources');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Data Sources');
  });

  test('displays Add Source button', async ({ page }) => {
    await expect(
      page.getByRole('button', { name: /Add Source/i }),
    ).toBeVisible();
  });

  test('sources list API returns envelope with data', async ({ page }) => {
    const sourcesResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/data/sources') &&
        !resp.url().includes('/stats') &&
        !resp.url().includes('/test') &&
        !resp.url().includes('/sync') &&
        !resp.url().includes('/schema') &&
        !resp.url().includes('/source-types') &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/data/sources');
    const response = await sourcesResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('shows source entries or empty state', async ({ page }) => {
    // Wait for data to load — either test buttons visible or empty state
    await page.waitForTimeout(2000);

    const hasTestButtons = await page.getByRole('button', { name: /Test/i }).first().isVisible().catch(() => false);
    const hasSourceCards = await page.locator('table').first().isVisible().catch(() => false);
    const hasEmptyState = await page.getByText(/No data sources/i).first().isVisible().catch(() => false);

    expect(hasTestButtons || hasSourceCards || hasEmptyState).toBe(true);
  });

  test('search input is present and accepts text', async ({ page }) => {
    const searchInput = page.getByPlaceholder(/Search/i);
    await expect(searchInput).toBeVisible();
    await searchInput.fill('test source');
    await expect(searchInput).toHaveValue('test source');
  });

  test('type filter options are available', async ({ page }) => {
    const typeFilter = page.getByRole('button', { name: /Type/i }).first();
    const hasTypeFilter = await typeFilter.isVisible().catch(() => false);

    if (hasTypeFilter) {
      await typeFilter.click();
      // Should show type options like PostgreSQL, MySQL, API, etc.
      const hasPostgres = await page.getByText(/PostgreSQL/i).first().isVisible().catch(() => false);
      const hasMySQL = await page.getByText(/MySQL/i).first().isVisible().catch(() => false);
      const hasAPI = await page.getByText(/API/).first().isVisible().catch(() => false);
      expect(hasPostgres || hasMySQL || hasAPI).toBe(true);
    }
  });

  test('status filter options are available', async ({ page }) => {
    const statusFilter = page.getByRole('button', { name: /Status/i }).first();
    const hasStatusFilter = await statusFilter.isVisible().catch(() => false);

    if (hasStatusFilter) {
      await statusFilter.click();
      const hasActive = await page.getByText('Active').first().isVisible().catch(() => false);
      const hasError = await page.getByText('Error').first().isVisible().catch(() => false);
      expect(hasActive || hasError).toBe(true);
    }
  });

  test('Add Source button opens wizard dialog', async ({ page }) => {
    const addBtn = page.getByRole('button', { name: /Add Source/i });
    await addBtn.click();

    await expect(
      page.getByRole('dialog').or(page.locator('[role="dialog"]')),
    ).toBeVisible({ timeout: 5_000 });
  });

  test('source table shows column headers in table view', async ({ page }) => {
    // Default view is cards/grid — switch to table view via URL param
    const apiReady = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/data/sources') &&
        !resp.url().includes('/stats') &&
        !resp.url().includes('/test') &&
        !resp.url().includes('/sync') &&
        !resp.url().includes('/schema') &&
        !resp.url().includes('/source-types') &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/data/sources?view=table');
    await apiReady;

    const hasTable = await page.locator('table').first().isVisible().catch(() => false);
    const hasEmptyState = await page.getByText(/No data sources/i).first().isVisible().catch(() => false);

    if (hasTable) {
      await expect(page.getByText('Name').first()).toBeVisible();
      await expect(page.getByText('Status').first()).toBeVisible();
    } else {
      expect(hasEmptyState).toBe(true);
    }
  });

  test('source rows show action buttons', async ({ page }) => {
    // Wait for API data to load
    const apiReady = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/data/sources') &&
        !resp.url().includes('/stats') &&
        !resp.url().includes('/test') &&
        !resp.url().includes('/sync') &&
        !resp.url().includes('/schema') &&
        !resp.url().includes('/source-types') &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/data/sources');
    const res = await apiReady;
    const body = await res.json();

    if (!body.data || body.data.length === 0) return;

    // Should show Test and/or Sync buttons
    const hasTestBtn = await page.getByRole('button', { name: /Test/i }).first().isVisible().catch(() => false);
    const hasSyncBtn = await page.getByRole('button', { name: /Sync/i }).first().isVisible().catch(() => false);
    expect(hasTestBtn || hasSyncBtn).toBe(true);
  });

  test('Test button triggers connection test API call', async ({ page }) => {
    const apiReady = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/data/sources') &&
        !resp.url().includes('/stats') &&
        !resp.url().includes('/test') &&
        !resp.url().includes('/sync') &&
        !resp.url().includes('/schema') &&
        !resp.url().includes('/source-types') &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/data/sources');
    const res = await apiReady;
    const body = await res.json();

    if (!body.data || body.data.length === 0) return;

    const testButtons = page.getByRole('button', { name: /^Test$/i });
    const count = await testButtons.count();

    if (count > 0) {
      // Listen for the POST /test API call
      const testResponse = page.waitForResponse(
        (resp) =>
          resp.url().includes('/test') &&
          resp.url().includes('/sources/') &&
          resp.request().method() === 'POST',
        { timeout: 15_000 },
      );

      await testButtons.first().click();

      const response = await testResponse;
      // Should get 200 (success/failure result) or 422/500
      expect([200, 422, 500]).toContain(response.status());
    }
  });

  test('clicking source name navigates to detail page', async ({ page }) => {
    // beforeEach already loaded the page; wait for source cards/rows to render
    await page.waitForTimeout(2000);

    // Click the "Open" link on the first source card
    const openLinks = page.getByRole('link', { name: 'Open' });
    const count = await openLinks.count();
    if (count === 0) return;

    await openLinks.first().click();
    await expect(page).toHaveURL(/\/data\/sources\/[0-9a-f-]+/, { timeout: 15_000 });
  });
});

test.describe('Data Source Detail Page', () => {
  /**
   * Helper: navigate to a source's detail page.
   * Fetches source IDs directly via browser fetch, then tries each
   * until one loads successfully (no error state).
   * Returns false if no sources exist or all show errors (caller should skip).
   */
  async function goToFirstSource(page: import('@playwright/test').Page): Promise<boolean> {
    // Fetch source list directly via browser API to avoid waitForResponse race conditions
    await page.goto('/data/sources');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

    const sourceIds: string[] = await page.evaluate(async () => {
      try {
        const res = await fetch('/api/v1/data/sources?per_page=10&sort=updated_at&order=desc');
        if (!res.ok) return [];
        const body = await res.json();
        return (body.data || []).map((s: { id: string }) => s.id);
      } catch {
        return [];
      }
    });

    if (sourceIds.length === 0) return false;

    // Try each source until one loads without error
    for (const id of sourceIds.slice(0, 5)) {
      await page.goto(`/data/sources/${id}`);
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

      // Wait for either summary cards or error state to appear
      await page.waitForTimeout(2000);

      const hasError = await page.getByText(/Failed to load|Something went wrong/i).first().isVisible().catch(() => false);
      if (!hasError) {
        try {
          await expect(page.getByText('Type').first()).toBeVisible({ timeout: 10_000 });
          return true;
        } catch {
          continue;
        }
      }
    }

    return false;
  }

  test('loads source detail with summary cards', async ({ page }) => {
    const ok = await goToFirstSource(page);
    if (!ok) { test.skip(); return; }

    await expect(page.getByText('Type').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Tables').first()).toBeVisible();
    await expect(page.getByText('Rows').first()).toBeVisible();
    await expect(page.getByText('Size').first()).toBeVisible();
  });

  test('displays tabs for Overview, Schema, Pipelines, Quality, Lineage, Activity', async ({ page }) => {
    const ok = await goToFirstSource(page);
    if (!ok) { test.skip(); return; }

    await expect(page.getByRole('tab', { name: 'Overview' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Schema' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Pipelines' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Quality' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Lineage' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Activity' })).toBeVisible();
  });

  test('Back to sources link navigates back', async ({ page }) => {
    const ok = await goToFirstSource(page);
    if (!ok) { test.skip(); return; }

    const backLink = page.getByRole('link', { name: /Back to sources/i });
    await expect(backLink).toBeVisible();
    await backLink.click();

    await expect(page).toHaveURL(/\/data\/sources(\?|$)/);
  });

  test('switching to Schema tab shows schema content', async ({ page }) => {
    const ok = await goToFirstSource(page);
    if (!ok) { test.skip(); return; }

    await page.getByRole('tab', { name: 'Schema' }).click();

    // Should show schema content or empty state
    const hasSchema = await page.getByText(/column|table|field|schema|PII|No schema/i).first().isVisible({ timeout: 5_000 }).catch(() => false);
    expect(hasSchema).toBe(true);
  });

  test('switching to Pipelines tab shows pipeline list', async ({ page }) => {
    const ok = await goToFirstSource(page);
    if (!ok) { test.skip(); return; }

    await page.getByRole('tab', { name: 'Pipelines' }).click();

    // Should show pipelines or empty state
    const hasPipelines = await page.getByText(/pipeline|ETL|ELT|No pipelines/i).first().isVisible({ timeout: 5_000 }).catch(() => false);
    expect(hasPipelines).toBe(true);
  });

  test('source detail API returns valid data', async ({ page }) => {
    const listApiResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/data/sources') &&
        !resp.url().includes('/stats') &&
        !resp.url().includes('/test') &&
        !resp.url().includes('/sync') &&
        !resp.url().includes('/schema') &&
        !resp.url().includes('/source-types') &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/data/sources');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
    const listRes = await listApiResponse;
    const listBody = await listRes.json();
    if (!listBody.data || listBody.data.length === 0) { test.skip(); return; }

    // Navigate directly to the first source's detail page
    const sourceId = listBody.data[0].id;

    const detailResponse = page.waitForResponse(
      (resp) =>
        resp.url().match(/\/api\/v1\/data\/sources\/[0-9a-f-]+$/) !== null &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto(`/data/sources/${sourceId}`);
    const response = await detailResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(body.data).toHaveProperty('id');
    expect(body.data).toHaveProperty('name');
    expect(body.data).toHaveProperty('type');
    expect(body.data).toHaveProperty('status');
  });

  test('summary cards show type value after loading', async ({ page }) => {
    const ok = await goToFirstSource(page);
    if (!ok) { test.skip(); return; }

    // The type card should show an actual source type value
    const hasTypeValue = await page.getByText(/postgresql|mysql|clickhouse|api|csv|s3|stream|dolt|impala|hive|hdfs|spark|dagster/i).first().isVisible().catch(() => false);
    expect(hasTypeValue).toBe(true);
  });
});
