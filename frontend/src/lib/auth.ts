import type { TokenPayload } from '@/types/auth';

// Access tokens are stored IN MEMORY ONLY — never localStorage or sessionStorage.
// Refresh tokens are in httpOnly cookies managed by the BFF layer.
// XSS cannot steal httpOnly cookies; in-memory tokens are lost on page refresh,
// at which point the BFF /api/auth/session endpoint rehydrates from the cookie.

let accessToken: string | null = null;

export function getAccessToken(): string | null {
  return accessToken;
}

export function setAccessToken(token: string): void {
  accessToken = token;
}

export function clearAccessToken(): void {
  accessToken = null;
}

/**
 * Decode the JWT payload section without signature verification.
 * We trust our own tokens; the backend already verified the signature.
 * Returns null on any error (malformed token).
 */
export function decodeJWTPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = parts[1];
    // base64url → base64 → decode
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(base64.length + (4 - (base64.length % 4)) % 4, '=');
    const decoded = atob(padded);
    return JSON.parse(decoded) as Record<string, unknown>;
  } catch {
    return null;
  }
}

export function getTokenPayload(token: string): TokenPayload | null {
  const raw = decodeJWTPayload(token);
  if (!raw) return null;
  // Validate required claims exist
  if (
    typeof raw['sub'] !== 'string' ||
    typeof raw['email'] !== 'string' ||
    typeof raw['tenant_id'] !== 'string' ||
    typeof raw['exp'] !== 'number' ||
    typeof raw['iat'] !== 'number' ||
    typeof raw['jti'] !== 'string' ||
    !Array.isArray(raw['roles']) ||
    !Array.isArray(raw['permissions'])
  ) {
    return null;
  }
  return raw as unknown as TokenPayload;
}

/**
 * Returns true if the token is expired or will expire within 30 seconds.
 * The 30-second buffer prevents edge-case failures on slow networks.
 */
export function isTokenExpired(token: string): boolean {
  const payload = decodeJWTPayload(token);
  if (!payload || typeof payload['exp'] !== 'number') return true;
  const nowPlusBuffer = Math.floor(Date.now() / 1000) + 30;
  return payload['exp'] < nowPlusBuffer;
}
