import { describe, it, expect, beforeEach, vi } from 'vitest';
import { useAuthStore } from './auth-store';
import * as authLib from '@/lib/auth';

// We test hasPermission in isolation by mocking getAccessToken + getTokenPayload
vi.mock('@/lib/auth', async (importOriginal) => {
  const actual = await importOriginal<typeof authLib>();
  return {
    ...actual,
    getAccessToken: vi.fn(() => 'mock-token'),
    getTokenPayload: vi.fn(() => ({
      sub: 'u1',
      email: 'u@example.com',
      tenant_id: 't1',
      roles: ['analyst'],
      permissions: [],
      exp: Math.floor(Date.now() / 1000) + 900,
      iat: Math.floor(Date.now() / 1000),
      jti: 'jti1',
    })),
    setAccessToken: vi.fn(),
    clearAccessToken: vi.fn(),
    isTokenExpired: vi.fn(() => false),
  };
});

function setPermissions(permissions: string[]) {
  vi.mocked(authLib.getTokenPayload).mockReturnValue({
    sub: 'u1',
    email: 'u@example.com',
    tenant_id: 't1',
    roles: ['analyst'],
    permissions,
    exp: Math.floor(Date.now() / 1000) + 900,
    iat: Math.floor(Date.now() / 1000),
    jti: 'jti1',
  });
}

describe('auth-store permissions', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: {
        id: 'u1',
        tenant_id: 't1',
        email: 'u@example.com',
        first_name: 'Test',
        last_name: 'User',
        status: 'active',
        mfa_enabled: false,
        last_login_at: null,
        password_changed_at: '',
        roles: [],
        created_at: '',
        updated_at: '',
      },
      isAuthenticated: true,
    });
  });

  it('test_hasPermission_exactMatch: "cyber:read" in ["cyber:read"] → true', () => {
    setPermissions(['cyber:read']);
    expect(useAuthStore.getState().hasPermission('cyber:read')).toBe(true);
  });

  it('test_hasPermission_resourceWildcard: "alerts:write" in ["alerts:*"] → true', () => {
    setPermissions(['alerts:*']);
    expect(useAuthStore.getState().hasPermission('alerts:write')).toBe(true);
  });

  it('test_hasPermission_superWildcard: "anything" in ["*"] → true', () => {
    setPermissions(['*']);
    expect(useAuthStore.getState().hasPermission('anything:whatever')).toBe(true);
  });

  it('test_hasPermission_actionWildcard: "cyber:read" in ["*:read"] → true', () => {
    setPermissions(['*:read']);
    expect(useAuthStore.getState().hasPermission('cyber:read')).toBe(true);
  });

  it('test_hasPermission_noMatch: "cyber:write" in ["cyber:read"] → false', () => {
    setPermissions(['cyber:read']);
    expect(useAuthStore.getState().hasPermission('cyber:write')).toBe(false);
  });

  it('test_hasSuiteAccess_cyber: has "cyber:read" → true', () => {
    setPermissions(['cyber:read']);
    expect(useAuthStore.getState().hasSuiteAccess('cyber')).toBe(true);
  });

  it('test_hasSuiteAccess_noAccess: has only "data:read" → cyber = false', () => {
    setPermissions(['data:read']);
    expect(useAuthStore.getState().hasSuiteAccess('cyber')).toBe(false);
  });

  it('hasAnyPermission returns true when at least one matches', () => {
    setPermissions(['data:read']);
    expect(
      useAuthStore.getState().hasAnyPermission(['cyber:read', 'data:read']),
    ).toBe(true);
  });

  it('hasAllPermissions returns false when any is missing', () => {
    setPermissions(['cyber:read']);
    expect(
      useAuthStore.getState().hasAllPermissions(['cyber:read', 'data:read']),
    ).toBe(false);
  });
});
