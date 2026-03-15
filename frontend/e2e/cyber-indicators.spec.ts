import { test, expect } from '@playwright/test';

test.describe('Cyber Indicators (IOC Management) Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/indicators');
    await expect(page.getByRole('heading', { name: 'IOC Management' })).toBeVisible({ timeout: 30_000 });
  });

  // ---- Page structure ----

  test('renders page header with title and description', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'IOC Management' })).toBeVisible();
    await expect(
      page.getByText(/indicators/i).first(),
    ).toBeVisible();
  });

  test('renders action buttons in header', async ({ page }) => {
    await expect(page.getByRole('button', { name: /Check Indicators/i })).toBeVisible({ timeout: 10_000 });
    await expect(page.getByRole('button', { name: /Bulk Import/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Add Indicator/i })).toBeVisible();
  });

  // ---- Stats API ----

  test('indicator stats API returns valid data', async ({ request }) => {
    // Login to get token
    const loginResp = await request.post('http://localhost:8081/api/v1/auth/login', {
      data: { email: 'admin@clario.dev', password: 'Cl@rio360Dev!' },
    });
    const { access_token } = await loginResp.json();

    const response = await request.get('http://localhost:8080/api/v1/cyber/indicators/stats', {
      headers: { Authorization: `Bearer ${access_token}` },
    });
    expect(response.ok()).toBe(true);

    const body = await response.json();
    expect(body).toHaveProperty('data');
    expect(body.data).toHaveProperty('total');
    expect(body.data).toHaveProperty('active');
    expect(body.data).toHaveProperty('expiring_soon');
    expect(body.data).toHaveProperty('by_source');
    expect(typeof body.data.total).toBe('number');
    expect(body.data.total).toBeGreaterThan(0);
    expect(Array.isArray(body.data.by_source)).toBe(true);
  });

  // ---- KPI cards ----

  test('displays KPI cards with stats', async ({ page }) => {
    await expect(page.getByText('Total IOCs')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Active IOCs')).toBeVisible();
    await expect(page.getByText('Expiring Soon')).toBeVisible();
    await expect(page.getByText('Source Mix')).toBeVisible();
  });

  // ---- Indicator list API ----

  test('indicators list API returns paginated data', async ({ request }) => {
    // Login to get token
    const loginResp = await request.post('http://localhost:8081/api/v1/auth/login', {
      data: { email: 'admin@clario.dev', password: 'Cl@rio360Dev!' },
    });
    const { access_token } = await loginResp.json();

    const response = await request.get('http://localhost:8080/api/v1/cyber/indicators?page=1&per_page=25&sort=last_seen_at&order=desc', {
      headers: { Authorization: `Bearer ${access_token}` },
    });
    expect(response.ok()).toBe(true);

    const body = await response.json();
    expect(body).toHaveProperty('data');
    expect(body).toHaveProperty('meta');
    expect(Array.isArray(body.data)).toBe(true);
    expect(body.data.length).toBeGreaterThan(0);
    expect(body.meta).toHaveProperty('total');
    expect(body.meta.total).toBeGreaterThan(0);

    // Verify indicator shape
    const indicator = body.data[0];
    expect(indicator).toHaveProperty('id');
    expect(indicator).toHaveProperty('type');
    expect(indicator).toHaveProperty('value');
    expect(indicator).toHaveProperty('severity');
    expect(indicator).toHaveProperty('source');
    expect(indicator).toHaveProperty('confidence');
    expect(indicator).toHaveProperty('active');
  });

  // ---- DataTable rendering ----

  test('displays indicator rows in the table', async ({ page }) => {
    // Table should render multiple rows with indicator data
    const tableBody = page.locator('tbody');
    await expect(tableBody).toBeVisible({ timeout: 15_000 });

    const rows = tableBody.locator('tr');
    await expect(rows.first()).toBeVisible({ timeout: 10_000 });
    const rowCount = await rows.count();
    expect(rowCount).toBeGreaterThan(0);
  });

  test('table displays indicator type badges', async ({ page }) => {
    // Wait for table to populate
    const tableBody = page.locator('tbody');
    await expect(tableBody).toBeVisible({ timeout: 15_000 });

    // Verify some type badges are visible (ip, domain, url, etc.)
    const typeBadges = page.locator('tbody td:first-child');
    await expect(typeBadges.first()).toBeVisible({ timeout: 10_000 });
  });

  test('table shows severity indicators', async ({ page }) => {
    const tableBody = page.locator('tbody');
    await expect(tableBody).toBeVisible({ timeout: 15_000 });

    // At least one severity badge should be visible
    const severityText = page.locator('tbody').getByText(/critical|high|medium|low/i).first();
    await expect(severityText).toBeVisible({ timeout: 10_000 });
  });

  // ---- Search ----

  test('search placeholder is visible', async ({ page }) => {
    await expect(
      page.getByPlaceholder(/Search IOC/i),
    ).toBeVisible({ timeout: 10_000 });
  });

  // ---- Filters ----

  test('filter buttons are available', async ({ page }) => {
    // DataTable filter controls should be visible
    await expect(page.getByText('Type').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Source').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Severity').first()).toBeVisible({ timeout: 10_000 });
  });

  // ---- Add Indicator dialog ----

  test('add indicator dialog opens and closes', async ({ page }) => {
    const addButton = page.getByRole('button', { name: /Add Indicator/i });
    await expect(addButton).toBeVisible({ timeout: 10_000 });
    await addButton.click();

    // Dialog should appear
    await expect(page.getByText(/Add Indicator|New Indicator/i).first()).toBeVisible({ timeout: 5_000 });

    // Close dialog by pressing Escape
    await page.keyboard.press('Escape');
  });

  // ---- Source Mix breakdown ----

  test('source mix card shows breakdown bars', async ({ page }) => {
    // Source Mix card should show source labels
    await expect(page.getByText('Source Mix')).toBeVisible({ timeout: 10_000 });

    // Should show at least some source names (OSINT, STIX Feed, Manual, Internal, Vendor)
    const sourceLabels = ['OSINT', 'STIX', 'Manual', 'Internal', 'Vendor'];
    let foundCount = 0;
    for (const label of sourceLabels) {
      const el = page.getByText(label, { exact: false }).first();
      if (await el.isVisible().catch(() => false)) {
        foundCount++;
      }
    }
    expect(foundCount).toBeGreaterThanOrEqual(3);
  });

  // ---- Row interaction ----

  test('clicking a row opens the detail panel', async ({ page }) => {
    // Wait for table rows to be present
    const firstRow = page.locator('tbody tr').first();
    await expect(firstRow).toBeVisible({ timeout: 15_000 });
    await firstRow.click();

    // Detail panel should open showing indicator info
    await expect(
      page.getByText(/First Seen|Last Seen|Confidence|Enrichment/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });

  // ---- Total row count ----

  test('table shows total row count', async ({ page }) => {
    const tableBody = page.locator('tbody');
    await expect(tableBody).toBeVisible({ timeout: 15_000 });

    // DataTable should show a total row count somewhere on the page
    // Look for "21 total" or "21 rows" or "Showing X of 21" patterns
    await expect(
      page.getByText(/\d+\s*(total|rows|results|items)/i).first(),
    ).toBeVisible({ timeout: 10_000 });
  });
});
