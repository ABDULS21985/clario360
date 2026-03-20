import { test as setup } from '@playwright/test';

const AUTH_FILE = './e2e/.auth/user.json';

setup('authenticate', async ({ page }) => {
  // 1. Login via IAM API to get tokens
  const loginResp = await page.request.post('http://localhost:8081/api/v1/auth/login', {
    data: { email: 'admin@clario.dev', password: 'Cl@rio360Dev!' },
  });

  if (!loginResp.ok()) {
    throw new Error(`IAM login failed: ${loginResp.status()} ${await loginResp.text()}`);
  }

  const { access_token, refresh_token } = await loginResp.json();

  // 2. Store tokens via BFF session endpoint (sets httpOnly cookies on localhost:3000)
  const sessionResp = await page.request.post('http://localhost:3000/api/auth/session', {
    data: { access_token, refresh_token },
  });

  if (!sessionResp.ok()) {
    throw new Error(`BFF session store failed: ${sessionResp.status()} ${await sessionResp.text()}`);
  }

  // 3. Navigate briefly to let browser context register the cookies
  await page.goto('/login', { waitUntil: 'commit' });

  // 4. Save the authenticated state (cookies)
  await page.context().storageState({ path: AUTH_FILE });
});
