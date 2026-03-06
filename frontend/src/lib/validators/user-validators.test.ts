import { describe, it, expect } from 'vitest';
import { createUserSchema, editUserSchema } from './user-validators';

const validCreate = {
  first_name: 'John',
  last_name: 'Doe',
  email: 'john@example.com',
  password: 'Password123!',
  confirm_password: 'Password123!',
  roles: ['role-1'],
  status: 'active' as const,
  send_welcome_email: true,
};

describe('createUserSchema', () => {
  it('test_createUser_validInput: valid data passes', () => {
    const result = createUserSchema.safeParse(validCreate);
    expect(result.success).toBe(true);
  });

  it('test_createUser_passwordMismatch: passwords differ → error on confirm_password', () => {
    const result = createUserSchema.safeParse({
      ...validCreate,
      confirm_password: 'Different123!',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const fields = result.error.issues.map((i) => i.path.join('.'));
      expect(fields).toContain('confirm_password');
    }
  });

  it('test_createUser_weakPassword: short password → multiple errors', () => {
    const result = createUserSchema.safeParse({
      ...validCreate,
      password: 'abc',
      confirm_password: 'abc',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues.length).toBeGreaterThan(1);
    }
  });

  it('test_createUser_noRoles: empty roles array → error', () => {
    const result = createUserSchema.safeParse({ ...validCreate, roles: [] });
    expect(result.success).toBe(false);
    if (!result.success) {
      const fields = result.error.issues.map((i) => i.path.join('.'));
      expect(fields).toContain('roles');
    }
  });

  it('test_createUser_invalidEmail: bad email → error', () => {
    const result = createUserSchema.safeParse({ ...validCreate, email: 'notanemail' });
    expect(result.success).toBe(false);
    if (!result.success) {
      const fields = result.error.issues.map((i) => i.path.join('.'));
      expect(fields).toContain('email');
    }
  });

  it('test_createUser_shortName: first_name too short → error', () => {
    const result = createUserSchema.safeParse({ ...validCreate, first_name: 'J' });
    expect(result.success).toBe(false);
    if (!result.success) {
      const fields = result.error.issues.map((i) => i.path.join('.'));
      expect(fields).toContain('first_name');
    }
  });
});

describe('editUserSchema', () => {
  it('test_editUser_validInput: valid data passes', () => {
    const result = editUserSchema.safeParse({
      first_name: 'Jane',
      last_name: 'Smith',
      status: 'suspended',
    });
    expect(result.success).toBe(true);
  });

  it('test_editUser_invalidStatus: unknown status → error', () => {
    const result = editUserSchema.safeParse({
      first_name: 'Jane',
      last_name: 'Smith',
      status: 'unknown',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const fields = result.error.issues.map((i) => i.path.join('.'));
      expect(fields).toContain('status');
    }
  });
});
