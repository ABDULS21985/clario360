import { describe, it, expect } from 'vitest';
import { createRoleSchema, editRoleSchema } from './role-validators';

describe('createRoleSchema', () => {
  it('test_createRole_validInput: valid data passes', () => {
    const result = createRoleSchema.safeParse({
      name: 'Custom Reviewer',
      description: 'Reviews submitted reports',
      permissions: ['reports:read'],
    });
    expect(result.success).toBe(true);
  });

  it('test_createRole_shortName: name too short → error', () => {
    const result = createRoleSchema.safeParse({
      name: 'ab',
      permissions: ['reports:read'],
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const fields = result.error.issues.map((i) => i.path.join('.'));
      expect(fields).toContain('name');
    }
  });

  it('test_createRole_noPermissions: empty permissions → error', () => {
    const result = createRoleSchema.safeParse({
      name: 'Valid Name',
      permissions: [],
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const fields = result.error.issues.map((i) => i.path.join('.'));
      expect(fields).toContain('permissions');
    }
  });

  it('test_createRole_optionalDescription: description is optional', () => {
    const result = createRoleSchema.safeParse({
      name: 'Valid Name',
      permissions: ['users:read'],
    });
    expect(result.success).toBe(true);
  });

  it('test_createRole_longDescription: description > 500 chars → error', () => {
    const result = createRoleSchema.safeParse({
      name: 'Valid Name',
      description: 'x'.repeat(501),
      permissions: ['users:read'],
    });
    expect(result.success).toBe(false);
  });
});

describe('editRoleSchema', () => {
  it('test_editRole_sameAsCreate: edit schema accepts same fields', () => {
    const result = editRoleSchema.safeParse({
      name: 'Updated Name',
      description: 'Updated description',
      permissions: ['cyber:read', 'alerts:read'],
    });
    expect(result.success).toBe(true);
  });
});
