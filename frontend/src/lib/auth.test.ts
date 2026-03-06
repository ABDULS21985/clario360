import { describe, it, expect, beforeEach } from 'vitest';
import {
  getAccessToken,
  setAccessToken,
  clearAccessToken,
  isTokenExpired,
  getTokenPayload,
  decodeJWTPayload,
} from './auth';

/**
 * Create a minimal JWT with the given payload.
 * We only care about the payload for these tests — signature is fake.
 */
function makeJWT(payload: Record<string, unknown>): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const body = btoa(JSON.stringify(payload))
    .replace(/=/g, '')
    .replace(/\+/g, '-')
    .replace(/\//g, '_');
  return `${header}.${body}.fakesignature`;
}

const NOW_SECONDS = Math.floor(Date.now() / 1000);

describe('Auth token utilities', () => {
  beforeEach(() => {
    clearAccessToken();
  });

  describe('getAccessToken / setAccessToken / clearAccessToken', () => {
    it('returns null before any token is set', () => {
      expect(getAccessToken()).toBeNull();
    });

    it('returns the token after setAccessToken', () => {
      setAccessToken('test-token');
      expect(getAccessToken()).toBe('test-token');
    });

    it('returns null after clearAccessToken', () => {
      setAccessToken('test-token');
      clearAccessToken();
      expect(getAccessToken()).toBeNull();
    });
  });

  describe('isTokenExpired', () => {
    it('test_isTokenExpired_validToken: token with exp far in future → false', () => {
      const token = makeJWT({ exp: NOW_SECONDS + 900, sub: 'u1' });
      expect(isTokenExpired(token)).toBe(false);
    });

    it('test_isTokenExpired_expiredToken: token with exp in past → true', () => {
      const token = makeJWT({ exp: NOW_SECONDS - 60, sub: 'u1' });
      expect(isTokenExpired(token)).toBe(true);
    });

    it('test_isTokenExpired_30sBuffer: token expiring in 20s → true (30s buffer)', () => {
      const token = makeJWT({ exp: NOW_SECONDS + 20, sub: 'u1' });
      expect(isTokenExpired(token)).toBe(true);
    });

    it('returns true for malformed token', () => {
      expect(isTokenExpired('not.a.token')).toBe(true);
    });

    it('returns true for empty string', () => {
      expect(isTokenExpired('')).toBe(true);
    });
  });

  describe('getTokenPayload', () => {
    it('test_getTokenPayload_validJWT: returns correct claims', () => {
      const payload = {
        sub: 'user-123',
        email: 'user@example.com',
        tenant_id: 'tenant-abc',
        roles: ['analyst'],
        permissions: ['cyber:read'],
        exp: NOW_SECONDS + 900,
        iat: NOW_SECONDS,
        jti: 'jti-xyz',
      };
      const token = makeJWT(payload);
      const result = getTokenPayload(token);
      expect(result).not.toBeNull();
      expect(result?.sub).toBe('user-123');
      expect(result?.email).toBe('user@example.com');
      expect(result?.tenant_id).toBe('tenant-abc');
      expect(result?.roles).toEqual(['analyst']);
      expect(result?.permissions).toEqual(['cyber:read']);
    });

    it('test_getTokenPayload_malformedToken: returns null (no crash)', () => {
      expect(getTokenPayload('not-a-jwt')).toBeNull();
    });

    it('test_getTokenPayload_emptyString: returns null', () => {
      expect(getTokenPayload('')).toBeNull();
    });

    it('returns null when required claims are missing', () => {
      const token = makeJWT({ sub: 'u1' }); // missing email, tenant_id, etc.
      expect(getTokenPayload(token)).toBeNull();
    });
  });
});
