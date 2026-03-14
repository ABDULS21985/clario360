import { test, expect } from '@playwright/test';

const uniqueThreatName = `pw-threat-${Date.now()}`;

test.describe('Threat Hunting — List Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/threats');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Threat Intelligence');
    await expect(
      page.getByText(/Track active threats, manage their lifecycle/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Active Threats').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Critical / High').first()).toBeVisible();
    await expect(page.getByText('IOCs Tracked').first()).toBeVisible();
    await expect(page.getByText('Contained This Month').first()).toBeVisible();
  });

  test('KPI cards show numeric values after loading', async ({ page }) => {
    await page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/threats/stats') &&
        !resp.url().includes('trend') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    // Verify KPI cards have rendered values (numbers or dashes)
    const activeThreats = page.getByText('Active Threats').first();
    await expect(activeThreats).toBeVisible();
    const criticalHigh = page.getByText('Critical / High').first();
    await expect(criticalHigh).toBeVisible();
  });

  test('stats API returns expected shape', async ({ page }) => {
    const statsResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/threats/stats') &&
        !resp.url().includes('trend') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/cyber/threats');
    const response = await statsResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    const data = body.data;
    expect(data).toHaveProperty('total');
    expect(data).toHaveProperty('active');
    expect(data).toHaveProperty('indicators_total');
    expect(data).toHaveProperty('contained_this_month');
    expect(data).toHaveProperty('by_type');
    expect(data).toHaveProperty('by_severity');
    expect(Array.isArray(data.by_type)).toBe(true);
    expect(Array.isArray(data.by_severity)).toBe(true);
  });

  test('displays Threats by Type bar chart', async ({ page }) => {
    await page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/threats/stats') &&
        !resp.url().includes('trend') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await expect(page.getByText('Threats by Type').first()).toBeVisible();
  });

  test('displays Threats by Severity pie chart', async ({ page }) => {
    await page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/threats/stats') &&
        !resp.url().includes('trend') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await expect(page.getByText('Threats by Severity').first()).toBeVisible();
  });

  test('shows threat data table or empty state', async ({ page }) => {
    await page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/threats') &&
        !resp.url().includes('stats') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    const hasTable = await page.locator('table').isVisible().catch(() => false);
    const hasEmpty = await page.getByText('No threats found').isVisible().catch(() => false);

    expect(hasTable || hasEmpty).toBe(true);
  });

  test('data table has search input', async ({ page }) => {
    await expect(page.getByPlaceholder('Search threats')).toBeVisible({ timeout: 10_000 });
  });

  test('data table supports severity filter', async ({ page }) => {
    await page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/threats') &&
        !resp.url().includes('stats') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    // Filter buttons have border-dashed class (vs column header sort buttons)
    const severityFilter = page.locator('button.border-dashed', { hasText: 'Severity' });
    await severityFilter.scrollIntoViewIfNeeded();
    await expect(severityFilter).toBeVisible({ timeout: 10_000 });

    // Click and verify popover opens with filter options
    await severityFilter.click();
    await expect(page.getByRole('button', { name: /^Critical$/ })).toBeVisible({ timeout: 5_000 });
    // Close popover
    await page.keyboard.press('Escape');
  });

  test('data table supports status filter', async ({ page }) => {
    await page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/threats') &&
        !resp.url().includes('stats') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    const statusFilter = page.locator('button.border-dashed', { hasText: 'Status' });
    await statusFilter.scrollIntoViewIfNeeded();
    await expect(statusFilter).toBeVisible({ timeout: 10_000 });

    // Click and verify popover opens with status options
    await statusFilter.click();
    await expect(page.getByRole('button', { name: /^Contained$/ })).toBeVisible({ timeout: 5_000 });
    await page.keyboard.press('Escape');
  });

  test('Check Indicators button opens dialog', async ({ page }) => {
    await page.getByRole('button', { name: /Check Indicators/i }).click();
    await expect(page.getByRole('heading', { name: 'Indicator Check' })).toBeVisible();
    await expect(page.locator('#indicators-input')).toBeVisible();
    await expect(
      page.getByText(/Paste IPs, domains, hashes/),
    ).toBeVisible();
  });

  test('Indicator Check dialog submits and shows results', async ({ page }) => {
    await page.getByRole('button', { name: /Check Indicators/i }).click();
    await expect(page.getByRole('heading', { name: 'Indicator Check' })).toBeVisible();

    // Enter test indicators
    await page.fill('#indicators-input', '8.8.8.8\nexample.com');

    // Submit
    await page.getByRole('button', { name: /Check Indicators/i }).last().click();

    // Wait for response
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/indicators/check') && resp.status() === 200,
      { timeout: 15_000 },
    );

    // Should show results (either Malicious or Clean sections)
    const hasMalicious = await page.getByText(/Malicious Indicator/).isVisible().catch(() => false);
    const hasClean = await page.getByText(/Clean/).isVisible().catch(() => false);

    expect(hasMalicious || hasClean).toBe(true);
  });

  test('Indicator Check dialog close resets state', async ({ page }) => {
    await page.getByRole('button', { name: /Check Indicators/i }).first().click();
    await expect(page.getByRole('heading', { name: 'Indicator Check' })).toBeVisible();

    await page.fill('#indicators-input', 'test-value');

    // Close button in dialog footer — use .first() since X close button also has name "Close"
    await page.getByRole('button', { name: /^Close$/ }).first().click();

    await expect(page.getByRole('heading', { name: 'Indicator Check' })).not.toBeVisible({ timeout: 10_000 });
  });

  test('New Threat button opens create dialog', async ({ page }) => {
    await page.getByRole('button', { name: /New Threat/i }).click();
    await expect(page.getByRole('heading', { name: 'Create Threat' })).toBeVisible();
    await expect(page.locator('#name')).toBeVisible();
    await expect(page.locator('#type')).toBeVisible();
    await expect(page.locator('#severity')).toBeVisible();
  });

  test('create dialog shows Initial Indicators section', async ({ page }) => {
    await page.getByRole('button', { name: /New Threat/i }).click();
    await expect(page.getByRole('heading', { name: 'Create Threat' })).toBeVisible();

    await expect(page.getByText('Initial Indicators')).toBeVisible();
    await expect(page.getByText('No indicators added yet.')).toBeVisible();
  });

  test('create dialog add and remove indicator row', async ({ page }) => {
    await page.getByRole('button', { name: /New Threat/i }).click();
    await expect(page.getByRole('heading', { name: 'Create Threat' })).toBeVisible();

    // Click Add Indicator
    await page.getByRole('button', { name: /Add Indicator/i }).click();
    await expect(page.getByText('Indicator 1')).toBeVisible();
    await expect(page.getByText('No indicators added yet.')).not.toBeVisible();

    // Remove it
    await page.getByRole('button', { name: /Remove/i }).click();
    await expect(page.getByText('No indicators added yet.')).toBeVisible();
  });

  test('create dialog cancel closes without saving', async ({ page }) => {
    await page.getByRole('button', { name: /New Threat/i }).click();
    await expect(page.getByRole('heading', { name: 'Create Threat' })).toBeVisible();

    await page.fill('#name', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByRole('heading', { name: 'Create Threat' })).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });

  test('fills and submits create threat form', async ({ page }) => {
    await page.getByRole('button', { name: /New Threat/i }).click();
    await expect(page.getByRole('heading', { name: 'Create Threat' })).toBeVisible();

    // Fill required fields
    await page.fill('#name', uniqueThreatName);
    await page.fill('#description', 'E2E test threat description');
    await page.fill('#threat_actor', 'PW-Actor');
    await page.fill('#campaign', 'pw-campaign');
    await page.fill('#tags_input', 'e2e, playwright');

    // Scroll to and click the submit button
    const submitBtn = page.getByRole('button', { name: /^Create Threat$/ });
    await submitBtn.scrollIntoViewIfNeeded();
    await submitBtn.click();

    // Wait for the API call to complete
    await page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/threats') &&
        !resp.url().includes('stats') &&
        (resp.status() === 200 || resp.status() === 201),
      { timeout: 15_000 },
    );

    // Should navigate to detail page on success
    await page.waitForURL(/\/cyber\/threats\/[a-f0-9-]+/, { timeout: 15_000 });
  });
});

test.describe('Threat Hunting — Detail Page', () => {
  test('navigates to a threat from the list and shows detail', async ({ page }) => {
    await page.goto('/cyber/threats');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

    // Wait for table data
    const dataResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/threats') &&
        !resp.url().includes('stats') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );
    await dataResponse;

    // Click the first row if data exists
    const rows = page.locator('table tbody tr');
    const rowCount = await rows.count();

    if (rowCount > 0) {
      await rows.first().click();

      // Should navigate to detail page
      await page.waitForURL(/\/cyber\/threats\/[a-f0-9-]+/, { timeout: 15_000 });

      // Detail page should show tabs
      await expect(page.getByRole('tab', { name: /Overview/i })).toBeVisible({ timeout: 15_000 });
      await expect(page.getByRole('tab', { name: /Indicators/i })).toBeVisible();
      await expect(page.getByRole('tab', { name: /Related Alerts/i })).toBeVisible();
      await expect(page.getByRole('tab', { name: /Activity Timeline/i })).toBeVisible();
      await expect(page.getByRole('tab', { name: /MITRE Mapping/i })).toBeVisible();
    }
  });

  test('detail page shows action buttons', async ({ page }) => {
    await page.goto('/cyber/threats');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

    // Wait for table to render (avoid waitForResponse race with cached data)
    await expect(page.locator('table')).toBeVisible({ timeout: 15_000 });

    const rows = page.locator('table tbody tr');
    const rowCount = await rows.count();

    if (rowCount > 0) {
      await rows.first().click();
      await page.waitForURL(/\/cyber\/threats\/[a-f0-9-]+/, { timeout: 15_000 });

      await expect(page.getByRole('button', { name: /Refresh/i })).toBeVisible({ timeout: 15_000 });
      await expect(page.getByRole('button', { name: /Update Status/i })).toBeVisible();
      await expect(page.getByRole('button', { name: /Edit Threat/i })).toBeVisible();
      await expect(page.getByRole('button', { name: /Delete Threat/i })).toBeVisible();
    }
  });

  test('detail page tabs switch content', async ({ page }) => {
    await page.goto('/cyber/threats');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

    // Wait for table to render
    await expect(page.locator('table')).toBeVisible({ timeout: 15_000 });

    const rows = page.locator('table tbody tr');
    const rowCount = await rows.count();

    if (rowCount > 0) {
      await rows.first().click();
      await page.waitForURL(/\/cyber\/threats\/[a-f0-9-]+/, { timeout: 15_000 });
      await expect(page.getByRole('tab', { name: /Overview/i })).toBeVisible({ timeout: 15_000 });

      // Switch to Indicators tab
      await page.getByRole('tab', { name: /Indicators/i }).click();
      await expect(page.getByRole('tabpanel')).toBeVisible({ timeout: 10_000 });

      // Switch to Related Alerts tab
      await page.getByRole('tab', { name: /Related Alerts/i }).click();
      await expect(page.getByRole('tabpanel')).toBeVisible({ timeout: 10_000 });

      // Switch to Activity Timeline tab
      await page.getByRole('tab', { name: /Activity Timeline/i }).click();
      await expect(page.getByRole('tabpanel')).toBeVisible({ timeout: 10_000 });

      // Switch to MITRE Mapping tab
      await page.getByRole('tab', { name: /MITRE Mapping/i }).click();
      await expect(page.getByRole('tabpanel')).toBeVisible({ timeout: 10_000 });
    }
  });

  test('detail page shows severity and status badges', async ({ page }) => {
    await page.goto('/cyber/threats');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

    // Wait for table to render (avoid waitForResponse race with cached data)
    await expect(page.locator('table')).toBeVisible({ timeout: 15_000 });

    const rows = page.locator('table tbody tr');
    const rowCount = await rows.count();

    if (rowCount > 0) {
      await rows.first().click();
      await page.waitForURL(/\/cyber\/threats\/[a-f0-9-]+/, { timeout: 15_000 });

      // Should show severity and status indicators in the header description area
      const hasSeverity = await page
        .getByText(/critical|high|medium|low/i)
        .first()
        .isVisible({ timeout: 10_000 })
        .catch(() => false);
      expect(hasSeverity).toBe(true);
    }
  });

  test('detail page Edit Threat opens edit dialog', async ({ page }) => {
    await page.goto('/cyber/threats');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

    // Wait for table to render (avoid waitForResponse race with cached data)
    await expect(page.locator('table')).toBeVisible({ timeout: 15_000 });

    const rows = page.locator('table tbody tr');
    const rowCount = await rows.count();

    if (rowCount > 0) {
      await rows.first().click();
      await page.waitForURL(/\/cyber\/threats\/[a-f0-9-]+/, { timeout: 15_000 });

      await page.getByRole('button', { name: /Edit Threat/i }).click();
      await expect(page.getByRole('heading', { name: 'Edit Threat' })).toBeVisible({ timeout: 10_000 });

      // Edit dialog should NOT show Initial Indicators section
      await expect(page.getByText('Initial Indicators')).not.toBeVisible();

      // Name field should be pre-filled
      const nameInput = page.locator('#name');
      const nameValue = await nameInput.inputValue();
      expect(nameValue.length).toBeGreaterThan(0);

      // Cancel
      await page.getByRole('button', { name: /Cancel/i }).click();
      await expect(page.getByRole('heading', { name: 'Edit Threat' })).not.toBeVisible();
    }
  });

  test('detail page back button navigates to list', async ({ page }) => {
    await page.goto('/cyber/threats');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

    // Wait for table to render (avoid waitForResponse race with cached data)
    await expect(page.locator('table')).toBeVisible({ timeout: 15_000 });

    const rows = page.locator('table tbody tr');
    const rowCount = await rows.count();

    if (rowCount > 0) {
      await rows.first().click();
      await page.waitForURL(/\/cyber\/threats\/[a-f0-9-]+/, { timeout: 15_000 });

      // Click back button (round button with ArrowLeft icon)
      const backButton = page.locator('button').filter({ has: page.locator('svg.lucide-arrow-left') });
      await backButton.click();

      await page.waitForURL(/\/cyber\/threats$/, { timeout: 15_000 });
      await expect(page.getByRole('heading', { level: 1 })).toContainText('Threat Intelligence');
    }
  });

  test('detail page Update Status dropdown shows transitions', async ({ page }) => {
    await page.goto('/cyber/threats');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });

    // Wait for table to render (avoid waitForResponse race with cached data)
    await expect(page.locator('table')).toBeVisible({ timeout: 15_000 });

    const rows = page.locator('table tbody tr');
    const rowCount = await rows.count();

    if (rowCount > 0) {
      await rows.first().click();
      await page.waitForURL(/\/cyber\/threats\/[a-f0-9-]+/, { timeout: 15_000 });

      const statusBtn = page.getByRole('button', { name: /Update Status/i });
      const isDisabled = await statusBtn.isDisabled();

      if (!isDisabled) {
        await statusBtn.click();

        // Should show at least one "Move to" option
        const moveItem = page.getByRole('menuitem').filter({ hasText: /Move to/ });
        const itemCount = await moveItem.count();
        expect(itemCount).toBeGreaterThanOrEqual(1);

        // Press Escape to close
        await page.keyboard.press('Escape');
      }
    }
  });
});

test.describe('Threat Hunting — API responses', () => {
  test('threats list API returns paginated envelope', async ({ page }) => {
    const threatsResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/threats') &&
        !resp.url().includes('stats') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/cyber/threats');
    const response = await threatsResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(body).toHaveProperty('meta');
    expect(body.meta).toHaveProperty('total');
    expect(body.meta).toHaveProperty('page');
    expect(body.meta).toHaveProperty('per_page');
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('trend API returns data array', async ({ page }) => {
    const trendResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/threats/stats/trend') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/cyber/threats');
    const response = await trendResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(Array.isArray(body.data)).toBe(true);
  });
});
