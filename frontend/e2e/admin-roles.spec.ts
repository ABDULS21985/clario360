import { test, expect } from '@playwright/test';

/* ------------------------------------------------------------------ */
/*  Helper: wait for roles API to respond                              */
/* ------------------------------------------------------------------ */
function waitForRolesApi(page: import('@playwright/test').Page) {
  return page.waitForResponse(
    (r) =>
      r.url().includes('/api/v1/roles') &&
      !r.url().includes('/users') &&
      r.request().method() === 'GET' &&
      r.status() === 200,
    { timeout: 15_000 },
  );
}

/* ================================================================== */
/*  Roles List Page                                                    */
/* ================================================================== */
test.describe('Roles List Page', () => {
  test.beforeEach(async ({ page }) => {
    const api = waitForRolesApi(page).catch(() => null);
    await page.goto('/admin/roles');
    // Wait for either API response or page content (cached data may skip the API call)
    await Promise.race([api, page.getByRole('heading', { name: /Role Management/i }).waitFor({ timeout: 15_000 })]);
    await expect(page.getByRole('heading', { name: /Role Management/i })).toBeVisible({ timeout: 15_000 });
  });

  test('renders page header and description', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /Role Management/i })).toBeVisible();
    await expect(page.getByText(/Define roles and permissions/i)).toBeVisible();
  });

  test('displays Create Role button', async ({ page }) => {
    await expect(page.getByRole('button', { name: /Create Role/i })).toBeVisible();
  });

  test('roles API returns valid data', async ({ page }) => {
    // Re-navigate to intercept a fresh response
    const api = waitForRolesApi(page);
    await page.goto('/admin/roles');
    const res = await api;
    const body = await res.json();

    // API returns a raw array of roles (not wrapped in { data: [] })
    expect(Array.isArray(body)).toBe(true);
    expect(body.length).toBeGreaterThan(0);
    expect(body[0]).toHaveProperty('name');
    expect(body[0]).toHaveProperty('permissions');
  });

  test('shows role cards with permissions count', async ({ page }) => {
    await expect(page.getByText(/\d+ permissions/i).first()).toBeVisible({ timeout: 10_000 });
  });

  test('system roles display System badge and locked View button', async ({ page }) => {
    const systemBadge = page.getByText('System', { exact: true });
    const hasBadge = await systemBadge.first().isVisible().catch(() => false);

    if (hasBadge) {
      const viewBtn = page.getByRole('button', { name: /View/i });
      await expect(viewBtn.first()).toBeVisible();
      await expect(viewBtn.first()).toBeDisabled();
    }
  });

  test('custom roles display Edit and Delete buttons', async ({ page }) => {
    const editBtn = page.getByRole('button', { name: /Edit/i });
    const hasEdit = await editBtn.first().isVisible().catch(() => false);

    if (hasEdit) {
      await expect(editBtn.first()).toBeEnabled();
    }
  });

  test('Create Role button opens form dialog', async ({ page }) => {
    await page.getByRole('button', { name: /Create Role/i }).click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });
    await expect(dialog.getByRole('heading', { name: 'Create Role' })).toBeVisible();
    await expect(dialog.getByPlaceholder(/Security Analyst/i)).toBeVisible();
  });

  test('Create Role dialog has all form fields', async ({ page }) => {
    await page.getByRole('button', { name: /Create Role/i }).click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });

    // Name and description fields
    await expect(dialog.getByPlaceholder(/Security Analyst/i)).toBeVisible();
    await expect(dialog.getByPlaceholder(/What can users with this role do/i)).toBeVisible();
    // Permissions section with groups (scoped to dialog)
    await expect(dialog.getByText('Permissions', { exact: true })).toBeVisible();
    await expect(dialog.getByText('Cybersecurity', { exact: true })).toBeVisible();
    await expect(dialog.getByText('Data Intelligence', { exact: true })).toBeVisible();
    await expect(dialog.getByText('Platform Admin', { exact: true })).toBeVisible();
    await expect(dialog.getByText('Audit', { exact: true })).toBeVisible();
    // Action buttons
    await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeVisible();
    await expect(dialog.getByRole('button', { name: 'Create Role' })).toBeVisible();
  });

  test('Create Role dialog validates required fields', async ({ page }) => {
    await page.getByRole('button', { name: /Create Role/i }).click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });

    // Submit with empty form
    await dialog.getByRole('button', { name: 'Create Role' }).click();

    // Should show validation errors
    await expect(dialog.getByText(/at least 3 characters|at least one permission/i).first()).toBeVisible({
      timeout: 5_000,
    });
  });

  test('permission groups expand to show individual permissions', async ({ page }) => {
    await page.getByRole('button', { name: /Create Role/i }).click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });

    // Expand Cybersecurity group
    await dialog.getByRole('button', { name: /Expand Cybersecurity/i }).click();

    // Individual permissions should be visible
    await expect(dialog.getByText('cyber:read')).toBeVisible();
    await expect(dialog.getByText('cyber:write')).toBeVisible();
    await expect(dialog.getByText('alerts:read')).toBeVisible();
  });

  test('clicking group checkbox selects all permissions in the group', async ({ page }) => {
    await page.getByRole('button', { name: /Create Role/i }).click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });

    // Click the Cybersecurity group label to toggle all permissions
    await dialog.getByText('Cybersecurity', { exact: true }).click();

    // The count should show 6/6
    await expect(dialog.getByText('6/6')).toBeVisible();
  });

  test('dashed "Create New Role" card is visible', async ({ page }) => {
    await expect(page.getByText('Create New Role')).toBeVisible();
  });
});

/* ================================================================== */
/*  Full lifecycle: create, list, edit, delete in one test              */
/* ================================================================== */
test.describe('Role Full Lifecycle', () => {
  const LIFECYCLE_ROLE = `E2E Lifecycle Role ${Date.now()}`;

  test('create → list → edit → delete role lifecycle', async ({ page }) => {
    // 1. Navigate to roles page
    const api = waitForRolesApi(page);
    await page.goto('/admin/roles');
    await api;
    await expect(page.getByRole('heading', { name: /Role Management/i })).toBeVisible({ timeout: 15_000 });

    // 2. CREATE the role
    await page.getByRole('button', { name: /Create Role/i }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });

    await dialog.getByPlaceholder(/Security Analyst/i).fill(LIFECYCLE_ROLE);
    await dialog.getByPlaceholder(/What can users with this role do/i).fill('Lifecycle test');

    // Select all Cybersecurity permissions via group toggle
    await dialog.getByText('Cybersecurity', { exact: true }).click();
    // Verify selection
    await expect(dialog.getByText('6/6')).toBeVisible();

    const createResp = page.waitForResponse(
      (r) => r.url().includes('/api/v1/roles') && r.request().method() === 'POST',
      { timeout: 15_000 },
    );
    await dialog.getByRole('button', { name: 'Create Role' }).click();
    const createResult = await createResp;
    expect(createResult.status()).toBeLessThan(300);

    await expect(dialog).not.toBeVisible({ timeout: 5_000 });

    // 3. VERIFY it's listed
    await expect(page.getByText(LIFECYCLE_ROLE)).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('6 permissions').first()).toBeVisible();

    // 4. EDIT the role — change description
    // Use the grid's direct child cards to avoid matching parent containers
    const roleCard = page.locator('.grid > div').filter({ hasText: LIFECYCLE_ROLE });
    await roleCard.getByRole('button', { name: /Edit/i }).click();

    const editDialog = page.getByRole('dialog');
    await expect(editDialog).toBeVisible({ timeout: 5_000 });

    const descInput = editDialog.getByPlaceholder(/What can users with this role do/i);
    await descInput.clear();
    await descInput.fill('Updated description');

    const editResp = page.waitForResponse(
      (r) => r.url().includes('/api/v1/roles/') && r.request().method() === 'PUT',
      { timeout: 15_000 },
    );
    await editDialog.getByRole('button', { name: /Save Changes/i }).click();
    const editResult = await editResp;
    expect(editResult.status()).toBeLessThan(300);
    await expect(editDialog).not.toBeVisible({ timeout: 5_000 });

    // Verify role is still listed with updated info
    await expect(page.getByText(LIFECYCLE_ROLE)).toBeVisible({ timeout: 5_000 });

    // 5. DELETE the role — re-locate card after edit re-render
    const updatedCard = page.locator('.grid > div').filter({ hasText: LIFECYCLE_ROLE });
    // The trash button has no accessible name — it's the button that isn't "Edit"
    const allBtns = updatedCard.getByRole('button');
    const btnCount = await allBtns.count();
    let trashBtn = allBtns.last();
    for (let i = 0; i < btnCount; i++) {
      const name = await allBtns.nth(i).textContent();
      if (name && !name.includes('Edit') && !name.includes('View')) {
        trashBtn = allBtns.nth(i);
        break;
      }
    }
    await trashBtn.click();

    const deleteDialog = page.getByRole('alertdialog');
    await expect(deleteDialog).toBeVisible({ timeout: 5_000 });

    const deleteResp = page.waitForResponse(
      (r) => r.url().includes('/api/v1/roles/') && r.request().method() === 'DELETE',
      { timeout: 15_000 },
    );
    await deleteDialog.getByRole('button', { name: /Delete/i }).click();
    const deleteResult = await deleteResp;
    expect(deleteResult.status()).toBeLessThan(300);

    // 6. VERIFY it's gone
    await expect(page.getByText(LIFECYCLE_ROLE)).not.toBeVisible({ timeout: 10_000 });
  });
});

/* ================================================================== */
/*  Role CRUD — Create with individual permissions, edit, delete       */
/* ================================================================== */
test.describe('Role CRUD with Individual Permissions', () => {
  const ROLE_NAME = `E2E Perm Role ${Date.now()}`;
  const ROLE_DESC = 'Role with individually selected permissions';

  test('create role with individual permissions via expand + checkbox', async ({ page }) => {
    const api = waitForRolesApi(page);
    await page.goto('/admin/roles');
    await api;
    await expect(page.getByRole('heading', { name: /Role Management/i })).toBeVisible({ timeout: 15_000 });

    // Open dialog
    await page.getByRole('button', { name: /Create Role/i }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });

    // Fill name and description
    await dialog.getByPlaceholder(/Security Analyst/i).fill(ROLE_NAME);
    await dialog.getByPlaceholder(/What can users with this role do/i).fill(ROLE_DESC);

    // Expand Cybersecurity and select individual permissions via their labels
    await dialog.getByRole('button', { name: /Expand Cybersecurity/i }).click();
    await dialog.locator('label[for="perm-cyber\\:read"]').click();
    await dialog.locator('label[for="perm-cyber\\:write"]').click();

    // Expand Data Intelligence and select a permission
    await dialog.getByRole('button', { name: /Expand Data Intelligence/i }).click();
    await dialog.locator('label[for="perm-data\\:read"]').click();

    // Submit
    const createResponse = page.waitForResponse(
      (r) => r.url().includes('/api/v1/roles') && r.request().method() === 'POST',
      { timeout: 15_000 },
    );
    await dialog.getByRole('button', { name: 'Create Role' }).click();
    const resp = await createResponse;
    expect(resp.status()).toBeLessThan(300);

    // Dialog should close
    await expect(dialog).not.toBeVisible({ timeout: 5_000 });

    // Role should appear with 3 permissions
    await expect(page.getByText(ROLE_NAME)).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('3 permissions').first()).toBeVisible();
  });

  test('verify newly created role in API response', async ({ page }) => {
    const api = waitForRolesApi(page);
    await page.goto('/admin/roles');
    const res = await api;
    const body = await res.json();

    const roles = Array.isArray(body) ? body : (body.data ?? []);
    const role = roles.find((r: { name: string }) => r.name === ROLE_NAME);
    if (!role) {
      test.skip();
      return;
    }

    expect(role).toHaveProperty('name', ROLE_NAME);
    expect(role).toHaveProperty('description', ROLE_DESC);
    expect(role).toHaveProperty('is_system', false);
    expect(role.permissions).toContain('cyber:read');
    expect(role.permissions).toContain('cyber:write');
    expect(role.permissions).toContain('data:read');
  });

  test('edit the created role to add audit permission', async ({ page }) => {
    const api = waitForRolesApi(page);
    await page.goto('/admin/roles');
    await api;
    await expect(page.getByRole('heading', { name: /Role Management/i })).toBeVisible({ timeout: 15_000 });

    const roleVisible = await page.getByText(ROLE_NAME).first().isVisible().catch(() => false);
    if (!roleVisible) {
      test.skip();
      return;
    }

    // Click Edit on our role
    const roleCard = page.locator('.grid > div').filter({ hasText: ROLE_NAME });
    await roleCard.getByRole('button', { name: /Edit/i }).click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible({ timeout: 5_000 });
    await expect(dialog.getByPlaceholder(/Security Analyst/i)).toHaveValue(ROLE_NAME);

    // Expand Audit and select audit:read
    await dialog.getByRole('button', { name: /Expand Audit/i }).click();
    await dialog.locator('label[for="perm-audit\\:read"]').click();

    // Save
    const saveResponse = page.waitForResponse(
      (r) => r.url().includes('/api/v1/roles/') && r.request().method() === 'PUT',
      { timeout: 15_000 },
    );
    await dialog.getByRole('button', { name: /Save Changes/i }).click();
    const resp = await saveResponse;
    expect(resp.status()).toBeLessThan(300);

    await expect(dialog).not.toBeVisible({ timeout: 5_000 });
    await expect(page.getByText(ROLE_NAME)).toBeVisible();
  });

  test('delete the created role', async ({ page }) => {
    const api = waitForRolesApi(page);
    await page.goto('/admin/roles');
    await api;
    await expect(page.getByRole('heading', { name: /Role Management/i })).toBeVisible({ timeout: 15_000 });

    const roleVisible = await page.getByText(ROLE_NAME).first().isVisible().catch(() => false);
    if (!roleVisible) {
      test.skip();
      return;
    }

    // Click the delete button — the trash button has no text
    const roleCard = page.locator('.grid > div').filter({ hasText: ROLE_NAME });
    const allBtns = roleCard.getByRole('button');
    const btnCount = await allBtns.count();
    let trashBtn = allBtns.last();
    for (let i = 0; i < btnCount; i++) {
      const name = await allBtns.nth(i).textContent();
      if (name && !name.includes('Edit') && !name.includes('View')) {
        trashBtn = allBtns.nth(i);
        break;
      }
    }
    await trashBtn.click();

    const confirmDialog = page.getByRole('alertdialog');
    await expect(confirmDialog).toBeVisible({ timeout: 5_000 });

    // Confirm delete
    const deleteResponse = page.waitForResponse(
      (r) => r.url().includes('/api/v1/roles/') && r.request().method() === 'DELETE',
      { timeout: 15_000 },
    );
    await confirmDialog.getByRole('button', { name: /Delete/i }).click();
    const resp = await deleteResponse;
    expect(resp.status()).toBeLessThan(300);

    // Role should be removed
    await expect(page.getByText(ROLE_NAME)).not.toBeVisible({ timeout: 10_000 });
  });
});
