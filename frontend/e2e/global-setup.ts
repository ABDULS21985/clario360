import { test as setup, expect } from '@playwright/test';

const AUTH_FILE = './e2e/.auth/user.json';

setup('authenticate', async ({ page }) => {
  // Navigate to login page
  await page.goto('/login');

  // Wait for the login form to be ready
  await page.waitForSelector('#email', { timeout: 15_000 });

  // Fill in credentials
  await page.fill('#email', 'admin@clario.dev');
  await page.fill('#password', 'Cl@rio360Dev!');

  // Click the sign-in button
  await page.getByRole('button', { name: /sign in/i }).click();

  // Wait for redirect to dashboard (successful login)
  await expect(page).toHaveURL(/\/(dashboard|admin)/, { timeout: 30_000 });

  // Save the authenticated state (cookies + localStorage)
  await page.context().storageState({ path: AUTH_FILE });
});
