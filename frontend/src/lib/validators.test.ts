import { describe, it, expect } from 'vitest';
import {
  loginSchema,
  registerSchema,
  resetPasswordSchema,
  changePasswordSchema,
} from './validators';

describe('loginSchema', () => {
  it('test_loginSchema_validInput: valid email + password → passes', () => {
    const result = loginSchema.safeParse({
      email: 'user@example.com',
      password: 'anypassword',
    });
    expect(result.success).toBe(true);
  });

  it('test_loginSchema_invalidEmail: "not-an-email" → fails with message', () => {
    const result = loginSchema.safeParse({
      email: 'not-an-email',
      password: 'anypassword',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const emailErrors = result.error.flatten().fieldErrors['email'];
      expect(emailErrors?.length).toBeGreaterThan(0);
    }
  });

  it('test_loginSchema_emptyPassword: "" → fails', () => {
    const result = loginSchema.safeParse({
      email: 'user@example.com',
      password: '',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const passwordErrors = result.error.flatten().fieldErrors['password'];
      expect(passwordErrors?.length).toBeGreaterThan(0);
    }
  });
});

describe('registerSchema', () => {
  const validInput = {
    organization_name: 'Acme Corp',
    industry: 'financial',
    country: 'SA',
    first_name: 'John',
    last_name: 'Doe',
    email: 'john@example.com',
    password: 'Str0ng!Pass#word',
    confirm_password: 'Str0ng!Pass#word',
  };

  it('test_registerSchema_validInput: all fields valid → passes', () => {
    const result = registerSchema.safeParse(validInput);
    expect(result.success).toBe(true);
  });

  it('test_registerSchema_passwordMismatch: password ≠ confirm → fails on confirm_password', () => {
    const result = registerSchema.safeParse({
      ...validInput,
      confirm_password: 'DifferentPass1!',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const errors = result.error.flatten().fieldErrors;
      expect(errors['confirm_password']).toBeDefined();
    }
  });

  it('test_registerSchema_weakPassword: "abc123" → fails (too short, missing requirements)', () => {
    const result = registerSchema.safeParse({
      ...validInput,
      password: 'abc123',
      confirm_password: 'abc123',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const errors = result.error.flatten().fieldErrors;
      expect(errors['password']).toBeDefined();
    }
  });

  it('test_registerSchema_missingOrganization: organization name missing → fails', () => {
    const result = registerSchema.safeParse({
      ...validInput,
      organization_name: '',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const errors = result.error.flatten().fieldErrors;
      expect(errors['organization_name']).toBeDefined();
    }
  });

  it('accepts a valid 2-letter country code', () => {
    const result = registerSchema.safeParse({
      ...validInput,
      country: 'NG',
    });
    expect(result.success).toBe(true);
  });
});

describe('resetPasswordSchema', () => {
  it('valid matching passwords → passes', () => {
    const result = resetPasswordSchema.safeParse({
      password: 'NewStr0ng!Pass#word',
      confirm_password: 'NewStr0ng!Pass#word',
    });
    expect(result.success).toBe(true);
  });

  it('mismatched passwords → fails on confirm_password', () => {
    const result = resetPasswordSchema.safeParse({
      password: 'NewStr0ng!Pass#word',
      confirm_password: 'DifferentPass1!',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.flatten().fieldErrors['confirm_password']).toBeDefined();
    }
  });
});

describe('changePasswordSchema', () => {
  it('test_resetPasswordSchema_sameAsCurrent: same password → fails', () => {
    const result = changePasswordSchema.safeParse({
      current_password: 'OldStr0ng!Pass#word',
      new_password: 'OldStr0ng!Pass#word',
      confirm_password: 'OldStr0ng!Pass#word',
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.flatten().fieldErrors['new_password']).toBeDefined();
    }
  });

  it('different new password → passes', () => {
    const result = changePasswordSchema.safeParse({
      current_password: 'OldStr0ng!Pass#word',
      new_password: 'NewStr0ng!Pass#word',
      confirm_password: 'NewStr0ng!Pass#word',
    });
    expect(result.success).toBe(true);
  });
});
