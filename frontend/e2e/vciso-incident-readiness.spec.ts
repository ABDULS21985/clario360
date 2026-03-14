import { test, expect } from '@playwright/test';

const uniqueRuleName = `pw-rule-${Date.now()}`;
const uniquePlaybookName = `pw-playbook-${Date.now()}`;

test.describe('Incident Readiness Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cyber/vciso/incident-readiness');
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
    await expect(
      page.getByText(/Manage escalation rules and crisis playbooks/),
    ).toBeVisible();
  });

  test('displays KPI cards', async ({ page }) => {
    await expect(page.getByText('Escalation Rules').first()).toBeVisible();
    await expect(page.getByText('Total Triggers').first()).toBeVisible();
    await expect(page.getByText('Tested Playbooks').first()).toBeVisible();
    await expect(page.getByText('Overdue Tests').first()).toBeVisible();
  });

  test('shows existing data in Escalation Rules tab', async ({ page }) => {
    // Escalation Rules is the default tab
    await expect(page.locator('table')).toBeVisible({ timeout: 10_000 });
  });

  test('switches to Crisis Playbooks tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Crisis Playbooks/i }).click();
    await expect(page.getByRole('tabpanel')).toBeVisible({ timeout: 10_000 });
  });

  test('opens Add Escalation Rule dialog', async ({ page }) => {
    await page.getByRole('button', { name: /Add Rule/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Escalation Rule' })).toBeVisible();
    await expect(page.locator('#rule-name')).toBeVisible();
    await expect(page.locator('#rule-description')).toBeVisible();
    await expect(page.locator('#rule-condition')).toBeVisible();
  });

  test('fills and submits Add Escalation Rule form', async ({ page }) => {
    await page.getByRole('button', { name: /Add Rule/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Escalation Rule' })).toBeVisible();

    await page.fill('#rule-name', uniqueRuleName);
    await page.fill('#rule-description', 'E2E test escalation rule description');
    await page.fill('#rule-condition', 'severity >= critical AND response_time > 30m');

    await page.getByRole('button', { name: /^Create Rule$/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 15_000 });

    // Verify new rule appears
    await expect(page.getByText(uniqueRuleName).first()).toBeVisible({ timeout: 10_000 });
  });

  test('cancel closes Add Rule dialog without saving', async ({ page }) => {
    await page.getByRole('button', { name: /Add Rule/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Escalation Rule' })).toBeVisible();

    await page.fill('#rule-name', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByRole('dialog')).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });

  test('opens Add Playbook dialog from Crisis Playbooks tab', async ({ page }) => {
    await page.getByRole('tab', { name: /Crisis Playbooks/i }).click();
    await expect(page.getByRole('tabpanel')).toBeVisible({ timeout: 10_000 });

    await page.getByRole('button', { name: /Add Playbook/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Crisis Playbook' })).toBeVisible();
    await expect(page.locator('#playbook-name')).toBeVisible();
    await expect(page.locator('#playbook-scenario')).toBeVisible();
    await expect(page.locator('#playbook-next-test')).toBeVisible();
  });

  test('fills and submits Add Playbook form', async ({ page }) => {
    await page.getByRole('tab', { name: /Crisis Playbooks/i }).click();
    await expect(page.getByRole('tabpanel')).toBeVisible({ timeout: 10_000 });

    await page.getByRole('button', { name: /Add Playbook/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Crisis Playbook' })).toBeVisible();

    await page.fill('#playbook-name', uniquePlaybookName);
    await page.fill('#playbook-scenario', 'E2E test crisis scenario for ransomware attack');
    await page.fill('#playbook-next-test', '2026-06-15');

    await page.getByRole('button', { name: /^Create Playbook$/i }).click();

    // Wait for dialog to close (success)
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 15_000 });

    // Verify new playbook appears
    await expect(page.getByText(uniquePlaybookName).first()).toBeVisible({ timeout: 10_000 });
  });

  test('cancel closes Add Playbook dialog without saving', async ({ page }) => {
    await page.getByRole('tab', { name: /Crisis Playbooks/i }).click();
    await expect(page.getByRole('tabpanel')).toBeVisible({ timeout: 10_000 });

    await page.getByRole('button', { name: /Add Playbook/i }).click();
    await expect(page.getByRole('heading', { name: 'Add Crisis Playbook' })).toBeVisible();

    await page.fill('#playbook-name', 'should-not-save');
    await page.getByRole('button', { name: /Cancel/i }).click();

    await expect(page.getByRole('dialog')).not.toBeVisible();
    await expect(page.getByText('should-not-save')).not.toBeVisible();
  });
});
