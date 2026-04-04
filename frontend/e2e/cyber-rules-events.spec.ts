import { test, expect } from '@playwright/test';

test.describe('Detection Rules', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/rules');
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Detection Rules', { timeout: 20_000 });
  });

  test('loads the rules workspace and template gallery', async ({ page }) => {
    await expect(page.getByPlaceholder('Search rules by name or description')).toBeVisible({ timeout: 10_000 });

    const templatesResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/rules/templates') && resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.getByRole('button', { name: /Templates/i }).click();
    await expect(page.getByText('Rule Template Gallery')).toBeVisible({ timeout: 10_000 });

    const response = await templatesResponse;
    const body = await response.json();
    expect(body).toHaveProperty('data');
    expect(Array.isArray(body.data)).toBe(true);
  });

  test('sends sortable rule list queries that the API accepts', async ({ page }) => {
    const sortResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/rules') &&
        resp.url().includes('sort=name') &&
        resp.url().includes('order=asc') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.getByRole('button', { name: /Rule Name/i }).click();

    const response = await sortResponse;
    const body = await response.json();
    expect(body).toHaveProperty('data');
    expect(body).toHaveProperty('meta');
    expect(body.meta).toHaveProperty('total');
  });
});

test.describe('Event Explorer', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/events');
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Event Explorer', { timeout: 20_000 });
  });

  test('loads event stats and list endpoints', async ({ page }) => {
    const statsResponse = page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/cyber/events/stats') && resp.status() === 200,
      { timeout: 15_000 },
    );
    const listResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/events') &&
        !resp.url().includes('/stats') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await expect(page.getByPlaceholder('Search events (IP, process, command, text)…')).toBeVisible({ timeout: 10_000 });

    const [stats, list] = await Promise.all([statsResponse, listResponse]);
    const statsBody = await stats.json();
    const listBody = await list.json();

    expect(statsBody).toHaveProperty('data');
    expect(statsBody.data).toHaveProperty('total');
    expect(listBody).toHaveProperty('data');
    expect(listBody).toHaveProperty('meta');
  });

  test('sends sortable event explorer queries that the API accepts', async ({ page }) => {
    const sortResponse = page.waitForResponse(
      (resp) =>
        resp.url().includes('/api/v1/cyber/events') &&
        !resp.url().includes('/stats') &&
        resp.url().includes('sort=source') &&
        resp.url().includes('order=asc') &&
        resp.status() === 200,
      { timeout: 15_000 },
    );

    await page.getByRole('button', { name: /Source/i }).click();

    const response = await sortResponse;
    const body = await response.json();
    expect(body).toHaveProperty('data');
    expect(body).toHaveProperty('meta');
  });
});
