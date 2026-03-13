import { test, expect } from '@playwright/test';

const uniqueServerName = `pw-server-${Date.now()}`;

test.describe('Compute Infrastructure Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/admin/ai-governance/compute');
    // Wait for page content to load
    await expect(page.getByText('Compute Infrastructure')).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByText('Compute Infrastructure')).toBeVisible();
    await expect(
      page.getByText(/Manage inference servers for CPU and GPU model serving/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Total Servers')).toBeVisible();
    await expect(page.getByText('Healthy', { exact: true })).toBeVisible();
    await expect(page.getByText('CPU Backends')).toBeVisible();
    await expect(page.getByText('GPU Backends')).toBeVisible();
  });

  test('shows existing inference servers in table', async ({ page }) => {
    await expect(page.getByText('llamacpp-cpu-dev-01')).toBeVisible({ timeout: 10_000 });
  });

  test('shows server details in table row', async ({ page }) => {
    const row = page.locator('tr').filter({ hasText: 'llamacpp-cpu-dev-01' });
    await expect(row).toBeVisible({ timeout: 10_000 });
    // Check backend type badge
    await expect(row.getByText('llama.cpp CPU')).toBeVisible();
    // Check model name
    await expect(row.getByText('llama-3.1-8b-instruct-q4_0')).toBeVisible();
    // Check status badge
    await expect(row.getByText('healthy', { exact: true })).toBeVisible();
  });

  test('opens Register Server dialog', async ({ page }) => {
    await page.getByRole('button', { name: /Add Server/i }).click();

    await expect(page.getByText('Register Inference Server')).toBeVisible();
    await expect(page.locator('#server-name')).toBeVisible();
    await expect(page.locator('#base-url')).toBeVisible();
    await expect(page.locator('#health-endpoint')).toBeVisible();
  });

  test('fills and submits Register Server form', async ({ page }) => {
    await page.getByRole('button', { name: /Add Server/i }).click();
    await expect(page.getByText('Register Inference Server')).toBeVisible();

    await page.fill('#server-name', uniqueServerName);
    await page.fill('#base-url', 'http://localhost:9999/v1');
    await page.fill('#health-endpoint', '/health');
    await page.fill('#model-name', 'test-model');
    await page.fill('#quantization', 'Q8_0');
    await page.fill('#cpu-cores', '16');
    await page.fill('#memory-mb', '32768');
    await page.fill('#max-concurrent', '8');

    await page.getByRole('button', { name: /Register Server/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByText('Register Inference Server')).not.toBeVisible({ timeout: 10_000 });

    // Verify new server appears in the table
    await expect(page.getByText(uniqueServerName).first()).toBeVisible({ timeout: 10_000 });
  });

  test('decommissions a server', async ({ page }) => {
    // Wait for the unique server to appear (created by previous test)
    await expect(page.getByText(uniqueServerName).first()).toBeVisible({ timeout: 10_000 });

    // Find the row and click the delete (trash) button
    const row = page.locator('tr').filter({ hasText: uniqueServerName }).first();
    await row.getByRole('button').filter({ has: page.locator('.text-destructive') }).click();

    // Confirm decommission dialog
    await expect(page.getByText('Decommission Server')).toBeVisible();
    await page.getByRole('button', { name: /Decommission/i }).click();

    // Wait for dialog to close
    await expect(page.getByText('Decommission Server')).not.toBeVisible({ timeout: 10_000 });
  });

  test('Refresh button reloads data', async ({ page }) => {
    await page.getByRole('button', { name: /Refresh/i }).click();
    await expect(page.getByRole('heading', { name: 'Inference Servers' })).toBeVisible();
  });

  test('cancel closes dialog without saving', async ({ page }) => {
    await page.getByRole('button', { name: /Add Server/i }).click();
    await expect(page.getByText('Register Inference Server')).toBeVisible();

    await page.fill('#server-name', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByText('Register Inference Server')).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });
});
