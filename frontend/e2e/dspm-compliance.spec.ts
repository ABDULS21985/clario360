import { test, expect } from '@playwright/test';

test.describe('DSPM Compliance Posture Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/dspm/compliance');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Compliance Posture');
    await expect(
      page.getByText(/Monitor data security compliance/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Total Violations').first()).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Frameworks Covered').first()).toBeVisible();
    await expect(page.getByText('Critical Violations').first()).toBeVisible();
  });

  test('policy violations API returns envelope with data', async ({ page }) => {
    const violationsResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/policies/violations') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/cyber/dspm/compliance');
    const response = await violationsResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('data policies API returns envelope with data', async ({ page }) => {
    const policiesResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/policies') && !resp.url().includes('violations') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.goto('/cyber/dspm/compliance');
    const response = await policiesResponse;
    const body = await response.json();

    expect(body).toHaveProperty('data');
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('shows framework cards or empty state', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/policies/violations') && resp.status() === 200,
      { timeout: 15_000 },
    );

    // Either shows framework cards or the "No Compliance Violations" message
    const hasFrameworks = await page.getByText(/GDPR|HIPAA|SOC 2|PCI-DSS|Saudi PDPL|ISO/).first().isVisible().catch(() => false);
    const hasEmpty = await page.getByText('No Compliance Violations').first().isVisible().catch(() => false);

    expect(hasFrameworks || hasEmpty).toBe(true);
  });

  test('KPI cards show numeric values after loading', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/policies/violations') && resp.status() === 200,
      { timeout: 15_000 },
    );

    const kpiCards = page.locator('.tabular-nums');
    const count = await kpiCards.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test('clicking a framework card shows violation details', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/policies/violations') && resp.status() === 200,
      { timeout: 15_000 },
    );

    // Check if there are framework cards with violations
    const frameworkCards = page.locator('[class*="cursor-pointer"]').filter({
      has: page.getByText('Violations'),
    });
    const cardCount = await frameworkCards.count();

    if (cardCount > 0) {
      // Click the first framework card
      await frameworkCards.first().click();

      // Should show violation details below
      await expect(
        page.getByText(/violation[s]? detected/).first(),
      ).toBeVisible({ timeout: 5_000 });
    }
  });

  test('clicking a framework card toggles selection', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/policies/violations') && resp.status() === 200,
      { timeout: 15_000 },
    );

    const frameworkCards = page.locator('[class*="cursor-pointer"]').filter({
      has: page.getByText('Violations'),
    });
    const cardCount = await frameworkCards.count();

    if (cardCount > 0) {
      // Click to select
      await frameworkCards.first().click();

      // Click again to deselect
      await frameworkCards.first().click();

      // Violation details should be hidden after deselecting
      await expect(
        page.getByText(/violation[s]? detected/).first(),
      ).not.toBeVisible({ timeout: 3_000 });
    }
  });

  test('framework cards show severity breakdown', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/policies/violations') && resp.status() === 200,
      { timeout: 15_000 },
    );

    // Check for severity labels on framework cards
    const hasSeverity = await page.getByText(/Critical|High|Medium|Low|Compliant/).first().isVisible().catch(() => false);
    expect(hasSeverity).toBe(true);
  });

  test('framework cards show top violations list', async ({ page }) => {
    await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/dspm/policies/violations') && resp.status() === 200,
      { timeout: 15_000 },
    );

    // If frameworks have violations, they should show "Top Violations" section
    const hasTopViolations = await page.getByText('Top Violations').first().isVisible().catch(() => false);
    const hasEmpty = await page.getByText('No Compliance Violations').first().isVisible().catch(() => false);

    // Either we see top violations or the empty state
    expect(hasTopViolations || hasEmpty).toBe(true);
  });
});
