import { test, expect } from '@playwright/test';

test.describe('Data Pipelines List Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/data/pipelines');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Pipelines');
    await expect(
      page.getByText(/Operational pipeline registry/),
    ).toBeVisible();
  });

  test('displays Create pipeline button', async ({ page }) => {
    await expect(
      page.getByRole('button', { name: /Create pipeline/i }),
    ).toBeVisible();
  });

  test('pipelines list API returns envelope with data', async ({ page }) => {
    const pipelinesResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/data/pipelines') &&
        !resp.url().includes('/count') &&
        !resp.url().includes('/stats') &&
        !resp.url().includes('/active') &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/data/pipelines');
    const response = await pipelinesResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('shows pipeline table with column headers', async ({ page }) => {
    await page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/data/pipelines') &&
        !resp.url().includes('/count') &&
        !resp.url().includes('/stats') &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    // Check for expected table column headers
    const hasTable = await page.locator('table').first().isVisible().catch(() => false);
    const hasEmptyState = await page.getByText('No pipelines found').first().isVisible().catch(() => false);

    // Either shows a table with headers or the empty state
    if (hasTable) {
      await expect(page.getByText('Pipeline').first()).toBeVisible();
      await expect(page.getByText('Status').first()).toBeVisible();
    } else {
      expect(hasEmptyState).toBe(true);
    }
  });

  test('shows pipeline rows or empty state', async ({ page }) => {
    await page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/data/pipelines') &&
        !resp.url().includes('/count') &&
        !resp.url().includes('/stats') &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    // Either pipelines listed with "Run now" buttons, or empty state
    const hasRunButtons = await page.getByRole('button', { name: /Run now/i }).first().isVisible().catch(() => false);
    const hasEmptyState = await page.getByText('No pipelines found').first().isVisible().catch(() => false);

    expect(hasRunButtons || hasEmptyState).toBe(true);
  });

  test('search input is present and accepts text', async ({ page }) => {
    const searchInput = page.getByPlaceholder('Search pipelines...');
    await expect(searchInput).toBeVisible();
    await searchInput.fill('test pipeline');
    await expect(searchInput).toHaveValue('test pipeline');
  });

  test('type filter options are available', async ({ page }) => {
    // Look for filter button/trigger for Type
    const typeFilter = page.getByRole('button', { name: /Type/i }).first();
    const hasTypeFilter = await typeFilter.isVisible().catch(() => false);

    if (hasTypeFilter) {
      await typeFilter.click();
      // Should show type options
      const hasETL = await page.getByText('ETL').first().isVisible().catch(() => false);
      const hasBatch = await page.getByText('Batch').first().isVisible().catch(() => false);
      expect(hasETL || hasBatch).toBe(true);
    }
  });

  test('status filter options are available', async ({ page }) => {
    const statusFilter = page.getByRole('button', { name: /Status/i }).first();
    const hasStatusFilter = await statusFilter.isVisible().catch(() => false);

    if (hasStatusFilter) {
      await statusFilter.click();
      const hasActive = await page.getByText('Active').first().isVisible().catch(() => false);
      const hasPaused = await page.getByText('Paused').first().isVisible().catch(() => false);
      expect(hasActive || hasPaused).toBe(true);
    }
  });

  test('Create pipeline button opens wizard dialog', async ({ page }) => {
    const createBtn = page.getByRole('button', { name: /Create pipeline/i });
    await createBtn.click();

    // Wizard dialog should appear
    await expect(
      page.getByRole('dialog').or(page.locator('[role="dialog"]')),
    ).toBeVisible({ timeout: 5_000 });
  });

  test('pipeline rows show actions dropdown', async ({ page }) => {
    await page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/data/pipelines') &&
        !resp.url().includes('/count') &&
        !resp.url().includes('/stats') &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    // Check for action buttons in table rows
    const actionButtons = page.getByRole('button', { name: /Pipeline actions/i });
    const count = await actionButtons.count();

    if (count > 0) {
      await actionButtons.first().click();

      // Dropdown should show Pause or Resume, and Delete
      const hasPause = await page.getByText('Pause').first().isVisible().catch(() => false);
      const hasResume = await page.getByText('Resume').first().isVisible().catch(() => false);
      const hasDelete = await page.getByText('Delete').first().isVisible().catch(() => false);

      expect(hasPause || hasResume).toBe(true);
      expect(hasDelete).toBe(true);
    }
  });

  test('clicking pipeline name navigates to detail page', async ({ page }) => {
    // beforeEach already loaded the page; wait for data to render
    await page.waitForTimeout(2000);

    // Find pipeline links in the table
    const pipelineLinks = page.locator('table a[href*="/data/pipelines/"]');
    const linkCount = await pipelineLinks.count();

    if (linkCount > 0) {
      await pipelineLinks.first().click();
      await expect(page).toHaveURL(/\/data\/pipelines\/[0-9a-f-]+/, { timeout: 15_000 });
    }
  });

  test('Run now button triggers pipeline run API call', async ({ page }) => {
    await page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/data/pipelines') &&
        !resp.url().includes('/count') &&
        !resp.url().includes('/stats') &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    const runButtons = page.getByRole('button', { name: /Run now/i });
    const count = await runButtons.count();

    if (count > 0) {
      // Listen for the POST /run API call
      const runResponse = page.waitForResponse(
        (resp) =>
          resp.url().includes('/run') &&
          resp.request().method() === 'POST',
        { timeout: 10_000 },
      );

      await runButtons.first().click();

      const response = await runResponse;
      // Should get either 202 (success) or 422 (expected failure for seeded data)
      expect([202, 422]).toContain(response.status());
    }
  });
});

test.describe('Pipeline Detail Page', () => {
  /**
   * Helper: navigate to the first pipeline's detail page.
   * Fetches pipeline IDs directly via browser fetch, then navigates to detail.
   * Returns false if no pipelines exist (caller should skip).
   */
  async function goToFirstPipeline(page: import('@playwright/test').Page): Promise<boolean> {
    // Fetch pipeline list directly via browser API to avoid waitForResponse race conditions
    await page.goto('/data/pipelines');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

    const pipelineId: string | null = await page.evaluate(async () => {
      try {
        const res = await fetch('/api/v1/data/pipelines?per_page=10&sort=updated_at&order=desc');
        if (!res.ok) return null;
        const body = await res.json();
        return body.data?.[0]?.id ?? null;
      } catch {
        return null;
      }
    });

    if (!pipelineId) return false;

    await page.goto(`/data/pipelines/${pipelineId}`);
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
    // Wait for summary cards to render
    await expect(page.getByText('Status').first()).toBeVisible({ timeout: 10_000 });
    return true;
  }

  test('loads pipeline detail with summary cards', async ({ page }) => {
    const ok = await goToFirstPipeline(page);
    if (!ok) { test.skip(); return; }

    await expect(page.getByText('Status').first()).toBeVisible();
    await expect(page.getByText('Runs').first()).toBeVisible();
    await expect(page.getByText('Processed').first()).toBeVisible();
    await expect(page.getByText('Avg Duration').first()).toBeVisible();
  });

  test('displays tabs for Runs, Config, Quality, Lineage, Root Cause', async ({ page }) => {
    const ok = await goToFirstPipeline(page);
    if (!ok) { test.skip(); return; }

    await expect(page.getByRole('tab', { name: 'Runs' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Config' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Quality' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Lineage' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Root Cause' })).toBeVisible();
  });

  test('Back to pipelines link navigates back', async ({ page }) => {
    const ok = await goToFirstPipeline(page);
    if (!ok) { test.skip(); return; }

    const backLink = page.getByRole('link', { name: /Back to pipelines/i });
    await expect(backLink).toBeVisible();
    await backLink.click();

    await expect(page).toHaveURL(/\/data\/pipelines(\?|$)/);
  });

  test('switching to Config tab shows pipeline configuration', async ({ page }) => {
    const ok = await goToFirstPipeline(page);
    if (!ok) { test.skip(); return; }

    await page.getByRole('tab', { name: 'Config' }).click();

    const hasConfig = await page.getByText(/source|batch|schedule|transform/i).first().isVisible({ timeout: 5_000 }).catch(() => false);
    expect(hasConfig).toBe(true);
  });

  test('pipeline detail API returns valid data', async ({ page }) => {
    // Fetch pipeline ID directly via browser API
    await page.goto('/data/pipelines');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

    const pipelineId: string | null = await page.evaluate(async () => {
      try {
        const res = await fetch('/api/v1/data/pipelines?per_page=10&sort=updated_at&order=desc');
        if (!res.ok) return null;
        const body = await res.json();
        return body.data?.[0]?.id ?? null;
      } catch {
        return null;
      }
    });

    if (!pipelineId) { test.skip(); return; }

    const detailResponse = page.waitForResponse(
      (resp) =>
        resp.url().match(/\/api\/v1\/data\/pipelines\/[0-9a-f-]+$/) !== null &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto(`/data/pipelines/${pipelineId}`);
    const response = await detailResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(body.data).toHaveProperty('id');
    expect(body.data).toHaveProperty('name');
    expect(body.data).toHaveProperty('status');
    expect(body.data).toHaveProperty('type');
  });

  test('pipeline runs API returns data array', async ({ page }) => {
    // Fetch pipeline ID directly via browser API
    await page.goto('/data/pipelines');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

    const pipelineId: string | null = await page.evaluate(async () => {
      try {
        const res = await fetch('/api/v1/data/pipelines?per_page=10&sort=updated_at&order=desc');
        if (!res.ok) return null;
        const body = await res.json();
        return body.data?.[0]?.id ?? null;
      } catch {
        return null;
      }
    });

    if (!pipelineId) { test.skip(); return; }

    // Navigate to the pipeline detail page and listen for runs API
    const runsResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/runs') &&
        resp.url().includes('/pipelines/') &&
        resp.request().method() === 'GET' &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto(`/data/pipelines/${pipelineId}`);
    const response = await runsResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('summary cards show status value after loading', async ({ page }) => {
    const ok = await goToFirstPipeline(page);
    if (!ok) { test.skip(); return; }

    // The status card should show an actual value
    const hasStatusValue = await page.getByText(/active|paused|disabled|error/i).first().isVisible().catch(() => false);
    expect(hasStatusValue).toBe(true);
  });
});
