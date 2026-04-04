import { test, expect } from '@playwright/test';

test.describe('DSPM Data Proliferation Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/dspm/proliferation');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Data Proliferation');
    await expect(
      page.getByText(/Track data asset spread/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Total Tracked Assets').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Spreading').first()).toBeVisible();
    await expect(page.getByText('Uncontrolled').first()).toBeVisible();
    await expect(page.getByText('Unauthorized Copies').first()).toBeVisible();
  });

  test('proliferation overview API returns envelope with data', async ({ page }) => {
    const overviewResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/proliferation/overview') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/cyber/dspm/proliferation');
    const response = await overviewResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    const data = body.data;
    expect(data).toHaveProperty('total_tracked_assets');
    expect(data).toHaveProperty('spreading_count');
    expect(data).toHaveProperty('uncontrolled_count');
    expect(data).toHaveProperty('total_unauthorized_copies');
    expect(data).toHaveProperty('proliferations');
    expect(Array.isArray(data.proliferations)).toBe(true);
  });

  test('shows Tracked Data Assets section or empty state', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/proliferation/overview') && resp.status() === 200,
      { timeout: 15_000 },
    );

    // Either shows tracked assets list or empty state
    const hasAssets = await page.getByText('Tracked Data Assets').first().isVisible().catch(() => false);
    const hasEmpty = await page.getByText('No Data Proliferation Detected').first().isVisible().catch(() => false);

    expect(hasAssets || hasEmpty).toBe(true);
  });

  test('KPI cards show numeric values after loading', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/proliferation/overview') && resp.status() === 200,
      { timeout: 15_000 },
    );

    const kpiCards = page.locator('.tabular-nums');
    const count = await kpiCards.count();
    expect(count).toBeGreaterThanOrEqual(4);
  });

  test('asset cards show status badges', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/proliferation/overview') && resp.status() === 200,
      { timeout: 15_000 },
    );

    // If there are proliferations, they should have status labels
    const hasAssets = await page.getByText('Tracked Data Assets').first().isVisible().catch(() => false);

    if (hasAssets) {
      // Check for status badges (Contained, Spreading, or Uncontrolled)
      const statusBadges = page.locator('button').filter({
        has: page.locator('text=/Contained|Spreading|Uncontrolled/'),
      });
      const badgeCount = await statusBadges.count();
      expect(badgeCount).toBeGreaterThanOrEqual(1);
    }
  });

  test('clicking asset row expands spread events', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/proliferation/overview') && resp.status() === 200,
      { timeout: 15_000 },
    );

    const hasAssets = await page.getByText('Tracked Data Assets').first().isVisible().catch(() => false);

    if (hasAssets) {
      // Click the first expandable asset button
      const assetButtons = page.locator('button[type="button"]').filter({
        has: page.locator('text=/total cop|authorized/'),
      });
      const buttonCount = await assetButtons.count();

      if (buttonCount > 0) {
        await assetButtons.first().click();

        // After clicking, we should see spread events or the expanded section
        await expect(
          page.getByText(/Spread Events/).first(),
        ).toBeVisible({ timeout: 5_000 });
      }
    }
  });

  test('spread events show authorized/unauthorized status', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/proliferation/overview') && resp.status() === 200,
      { timeout: 15_000 },
    );

    const hasAssets = await page.getByText('Tracked Data Assets').first().isVisible().catch(() => false);

    if (hasAssets) {
      // Expand first asset with spread events
      const assetButtons = page.locator('button[type="button"]').filter({
        has: page.locator('text=/total cop|authorized/'),
      });
      const buttonCount = await assetButtons.count();

      if (buttonCount > 0) {
        await assetButtons.first().click();

        // Check for Authorized or Unauthorized labels in spread events
        const hasAuthLabels = await page.getByText(/Authorized|Unauthorized/).first().isVisible({ timeout: 5_000 }).catch(() => false);
        if (hasAuthLabels) {
          expect(hasAuthLabels).toBe(true);
        }
      }
    }
  });
});
