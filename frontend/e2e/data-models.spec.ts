import { test, expect, type Page } from '@playwright/test';

/** Navigate to the first model detail page. Returns true if a model exists, false otherwise. */
async function goToFirstModel(page: Page): Promise<boolean> {
  await page.goto('/data/models');
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

  // Wait for either a table with rows or the empty state
  const hasTable = await page.locator('table tbody tr').first().isVisible({ timeout: 15_000 }).catch(() => false);

  if (!hasTable) {
    return false;
  }

  await page.locator('table tbody tr').first().locator('a').first().click();
  await page.waitForURL(/\/data\/models\/[a-f0-9-]+/, { timeout: 15_000 });
  // Wait for detail page to load
  await expect(page.getByRole('link', { name: /Back to models/i })).toBeVisible({ timeout: 15_000 });
  return true;
}

test.describe('Data Models — List Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/data/models');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Data Models');
    await expect(
      page.getByText(/Governed semantic models derived from discovered sources/),
    ).toBeVisible();
  });

  test('models list API returns paginated envelope', async ({ page }) => {
    // Intercept the API call that beforeEach already triggered
    // Navigate again with a unique query param to force a fresh API call
    const modelsResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/data/models') &&
        !resp.url().includes('/versions') &&
        !resp.url().includes('/lineage') &&
        !resp.url().includes('/derive') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    // Force fresh search query to trigger new API call
    await page.getByPlaceholder('Search models...').fill('__e2e_api_check__');

    const response = await modelsResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(body).toHaveProperty('meta');
    expect(body.meta).toHaveProperty('total');
    expect(body.meta).toHaveProperty('page');
    expect(body.meta).toHaveProperty('per_page');
    expect(Array.isArray(body.data)).toBe(true);

    // Clear the search
    await page.getByPlaceholder('Search models...').clear();
  });

  test('shows data table with column headers or empty state', async ({ page }) => {
    // Column headers are always rendered (even when table body is empty)
    // Model column header should always be visible after loading
    await expect(page.locator('th', { hasText: 'Model' })).toBeVisible({ timeout: 15_000 });
  });

  test('data table has search input', async ({ page }) => {
    await expect(page.getByPlaceholder('Search models...')).toBeVisible({ timeout: 10_000 });
  });

  test('data table has correct column headers', async ({ page }) => {
    // Column headers appear even when table is empty
    await expect(page.locator('th', { hasText: 'Model' })).toBeVisible({ timeout: 15_000 });
    await expect(page.locator('th', { hasText: 'Status' }).first()).toBeVisible();
    await expect(page.locator('th', { hasText: 'Classification' }).first()).toBeVisible();
    await expect(page.locator('th', { hasText: 'Fields' })).toBeVisible();
    await expect(page.locator('th', { hasText: 'PII' })).toBeVisible();
    await expect(page.locator('th', { hasText: 'Updated' })).toBeVisible();
  });

  test('data table supports status filter', async ({ page }) => {
    const statusFilter = page.locator('button.border-dashed', { hasText: 'Status' });
    await expect(statusFilter).toBeVisible({ timeout: 10_000 });

    await statusFilter.click();
    await expect(page.getByRole('button', { name: /^Active$/ })).toBeVisible({ timeout: 5_000 });
    await expect(page.getByRole('button', { name: /^Draft$/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /^Deprecated$/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /^Archived$/ })).toBeVisible();
    await page.keyboard.press('Escape');
  });

  test('data table supports classification filter', async ({ page }) => {
    const classFilter = page.locator('button.border-dashed', { hasText: 'Classification' });
    await expect(classFilter).toBeVisible({ timeout: 10_000 });

    await classFilter.click();
    await expect(page.getByRole('button', { name: /^Public$/ })).toBeVisible({ timeout: 5_000 });
    await expect(page.getByRole('button', { name: /^Internal$/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /^Confidential$/ })).toBeVisible();
    await expect(page.getByRole('button', { name: /^Restricted$/ })).toBeVisible();
    await page.keyboard.press('Escape');
  });

  test('empty state message appears when no data', async ({ page }) => {
    // Check if empty state is visible (it may or may not be, depending on seeded data)
    const hasEmpty = await page.getByText('No models found').isVisible({ timeout: 10_000 }).catch(() => false);

    if (hasEmpty) {
      await expect(page.getByText('No data models matched the current filters.')).toBeVisible();
    }
  });

  test('table rows display model links when data exists', async ({ page }) => {
    const firstRow = page.locator('table tbody tr').first();
    const hasRow = await firstRow.isVisible({ timeout: 15_000 }).catch(() => false);

    if (hasRow) {
      // First row should have a link (model name) and version text
      await expect(firstRow.locator('a')).toBeVisible();
      await expect(firstRow.getByText(/v\d+/)).toBeVisible();
    }
  });

  test('clicking model name navigates to detail page', async ({ page }) => {
    const firstRow = page.locator('table tbody tr').first();
    const hasRow = await firstRow.isVisible({ timeout: 15_000 }).catch(() => false);

    if (hasRow) {
      await firstRow.locator('a').first().click();
      await page.waitForURL(/\/data\/models\/[a-f0-9-]+/, { timeout: 15_000 });
    }
  });

  test('pagination info is visible', async ({ page }) => {
    const firstRow = page.locator('table tbody tr').first();
    const hasRow = await firstRow.isVisible({ timeout: 15_000 }).catch(() => false);

    if (hasRow) {
      await expect(page.getByText(/Showing \d+–\d+ of \d+ results/)).toBeVisible({ timeout: 10_000 });
    }
  });
});

test.describe('Data Models — Detail Page', () => {
  test('loads detail page with summary cards', async ({ page }) => {
    const hasModel = await goToFirstModel(page);
    if (!hasModel) {
      test.skip();
      return;
    }

    // Summary cards
    await expect(page.getByText('Status').first()).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText('Fields').first()).toBeVisible();
    await expect(page.getByText('PII Columns')).toBeVisible();
    await expect(page.getByText('Updated').first()).toBeVisible();
  });

  test('shows classification badge', async ({ page }) => {
    const hasModel = await goToFirstModel(page);
    if (!hasModel) {
      test.skip();
      return;
    }

    await expect(page.getByText('Classification')).toBeVisible({ timeout: 15_000 });
    const hasClassification = await page
      .getByText(/Public|Internal|Confidential|Restricted/)
      .first()
      .isVisible({ timeout: 10_000 })
      .catch(() => false);
    expect(hasClassification).toBe(true);
  });

  test('displays all 4 tabs', async ({ page }) => {
    const hasModel = await goToFirstModel(page);
    if (!hasModel) {
      test.skip();
      return;
    }

    await expect(page.getByRole('tab', { name: /Schema/i })).toBeVisible({ timeout: 15_000 });
    await expect(page.getByRole('tab', { name: /Quality Rules/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /Lineage/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: /Versions/i })).toBeVisible();
  });

  test('Schema tab shows field table', async ({ page }) => {
    const hasModel = await goToFirstModel(page);
    if (!hasModel) {
      test.skip();
      return;
    }

    // Schema tab is default — schema table headers should be visible
    await expect(page.locator('th', { hasText: 'Field' })).toBeVisible({ timeout: 10_000 });
    await expect(page.locator('th', { hasText: 'Type' })).toBeVisible();
    await expect(page.locator('th', { hasText: 'Nullable' })).toBeVisible();
    await expect(page.locator('th', { hasText: 'Description' })).toBeVisible();
  });

  test('Quality Rules tab shows rules or empty state', async ({ page }) => {
    const hasModel = await goToFirstModel(page);
    if (!hasModel) {
      test.skip();
      return;
    }

    await page.getByRole('tab', { name: /Quality Rules/i }).click();

    const hasRules = await page.getByText(/passed|failed|warning|never run/).first().isVisible({ timeout: 10_000 }).catch(() => false);
    const hasEmpty = await page.getByText('No quality rules are attached to this model.').isVisible().catch(() => false);

    expect(hasRules || hasEmpty).toBe(true);
  });

  test('Lineage tab shows source info', async ({ page }) => {
    const hasModel = await goToFirstModel(page);
    if (!hasModel) {
      test.skip();
      return;
    }

    await page.getByRole('tab', { name: /Lineage/i }).click();

    await expect(page.getByText(/Upstream source:/)).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText(/Source table:/)).toBeVisible();
    await expect(page.getByText(/Consumers:/)).toBeVisible();
    await expect(page.getByRole('link', { name: /Open full lineage/i })).toBeVisible();
  });

  test('Versions tab shows version history or empty state', async ({ page }) => {
    const hasModel = await goToFirstModel(page);
    if (!hasModel) {
      test.skip();
      return;
    }

    await page.getByRole('tab', { name: /Versions/i }).click();

    const hasVersions = await page.getByText(/Version \d+/).first().isVisible({ timeout: 10_000 }).catch(() => false);
    const hasEmpty = await page.getByText('No historical versions are available for this model.').isVisible().catch(() => false);

    expect(hasVersions || hasEmpty).toBe(true);
  });

  test('back button navigates to models list', async ({ page }) => {
    const hasModel = await goToFirstModel(page);
    if (!hasModel) {
      test.skip();
      return;
    }

    await page.getByRole('link', { name: /Back to models/i }).click();
    await page.waitForURL(/\/data\/models$/, { timeout: 15_000 });
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Data Models');
  });

  test('detail API returns model with schema', async ({ page }) => {
    await page.goto('/data/models');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

    const firstRow = page.locator('table tbody tr').first();
    const hasRow = await firstRow.isVisible({ timeout: 15_000 }).catch(() => false);

    if (!hasRow) {
      test.skip();
      return;
    }

    const detailResponse = page.waitForResponse(
      (resp) =>
        /\/api\/v1\/data\/models\/[a-f0-9-]+$/.test(resp.url()) &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await firstRow.locator('a').first().click();
    const response = await detailResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(body.data).toHaveProperty('name');
    expect(body.data).toHaveProperty('status');
    expect(body.data).toHaveProperty('schema_definition');
    expect(body.data).toHaveProperty('data_classification');
    expect(body.data).toHaveProperty('field_count');
    expect(Array.isArray(body.data.schema_definition)).toBe(true);
  });
});
