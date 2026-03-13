import { test, expect } from '@playwright/test';

const uniqueSuiteName = `pw-suite-${Date.now()}`;

test.describe('Benchmarks Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/admin/ai-governance/benchmarks');
    await expect(page.getByText('Inference Benchmarks')).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByText('Inference Benchmarks')).toBeVisible();
    await expect(
      page.getByText(/Measure and compare CPU vs GPU inference latency/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    // Use exact match or role to avoid conflicts with table headings
    await expect(page.getByText('Benchmark Suites', { exact: true }).first()).toBeVisible();
    await expect(page.getByText('Total Runs')).toBeVisible();
    await expect(page.getByText('Completed', { exact: true })).toBeVisible();
    await expect(page.getByText('Avg Latency')).toBeVisible();
  });

  test('shows Suites tab with existing data', async ({ page }) => {
    await expect(page.getByText('threat-scorer-cpu-bench')).toBeVisible({ timeout: 10_000 });
  });

  test('shows suite configuration in table row', async ({ page }) => {
    const row = page.locator('tr').filter({ hasText: 'threat-scorer-cpu-bench' });
    await expect(row).toBeVisible({ timeout: 10_000 });
    // Check target model badge within the row
    await expect(row.getByText('threat-scorer', { exact: true })).toBeVisible();
    // Check configuration display
    await expect(row.getByText(/iter/)).toBeVisible();
  });

  test('switches to Run Results tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Run Results/i }).click();
    await expect(page.getByRole('heading', { name: 'Run Results' })).toBeVisible();
  });

  test('shows run data in Run Results tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Run Results/i }).click();
    await expect(page.getByText('llamacpp_cpu').first()).toBeVisible({ timeout: 10_000 });
  });

  test('opens Create Suite dialog', async ({ page }) => {
    await page.getByRole('button', { name: /New Suite/i }).click();
    await expect(page.getByText('Create Benchmark Suite')).toBeVisible();
    await expect(page.locator('#suite-name')).toBeVisible();
    await expect(page.locator('#model-slug')).toBeVisible();
  });

  test('fills and submits Create Suite form', async ({ page }) => {
    await page.getByRole('button', { name: /New Suite/i }).click();
    await expect(page.getByText('Create Benchmark Suite')).toBeVisible();

    await page.fill('#suite-name', uniqueSuiteName);
    await page.fill('#suite-desc', 'E2E test benchmark suite');
    await page.fill('#model-slug', 'test-model-slug');
    await page.fill('#warmup', '2');
    await page.fill('#iterations', '10');
    await page.fill('#concurrency', '2');
    await page.fill('#timeout', '15');

    await page.getByRole('button', { name: /Create Suite/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByText('Create Benchmark Suite')).not.toBeVisible({ timeout: 15_000 });

    // Verify new suite appears
    await expect(page.getByText(uniqueSuiteName).first()).toBeVisible({ timeout: 10_000 });
  });

  test('opens Run Benchmark dialog from suite row', async ({ page }) => {
    await expect(page.getByText('threat-scorer-cpu-bench')).toBeVisible({ timeout: 10_000 });

    const row = page.locator('tr').filter({ hasText: 'threat-scorer-cpu-bench' });
    await row.getByRole('button', { name: /Run/i }).click();

    await expect(page.getByText('Run Benchmark')).toBeVisible();
    await expect(page.getByText('Suite Configuration')).toBeVisible();
  });

  test('cancel closes Create Suite dialog', async ({ page }) => {
    await page.getByRole('button', { name: /New Suite/i }).click();
    await expect(page.getByText('Create Benchmark Suite')).toBeVisible();

    await page.fill('#suite-name', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByText('Create Benchmark Suite')).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });

  test('navigates to suite detail page on suite name click', async ({ page }) => {
    await expect(page.getByText('threat-scorer-cpu-bench')).toBeVisible({ timeout: 10_000 });
    await page.getByText('threat-scorer-cpu-bench').click();
    await expect(page).toHaveURL(/\/admin\/ai-governance\/benchmarks\/[a-f0-9-]+/, { timeout: 10_000 });
  });
});

test.describe('Benchmark Suite Detail Page', () => {
  test('loads suite detail page and shows configuration', async ({ page }) => {
    await page.goto('/admin/ai-governance/benchmarks');
    await expect(page.getByText('threat-scorer-cpu-bench')).toBeVisible({ timeout: 15_000 });
    await page.getByText('threat-scorer-cpu-bench').click();
    await expect(page).toHaveURL(/\/admin\/ai-governance\/benchmarks\/[a-f0-9-]+/, { timeout: 10_000 });
    await expect(page.getByText('threat-scorer-cpu-bench')).toBeVisible();
  });

  test('shows benchmark runs on detail page', async ({ page }) => {
    await page.goto('/admin/ai-governance/benchmarks');
    await expect(page.getByText('threat-scorer-cpu-bench')).toBeVisible({ timeout: 15_000 });
    await page.getByText('threat-scorer-cpu-bench').click();
    await expect(page).toHaveURL(/\/admin\/ai-governance\/benchmarks\/[a-f0-9-]+/, { timeout: 10_000 });
    await expect(page.getByText('llamacpp_cpu').first()).toBeVisible({ timeout: 10_000 });
  });
});
