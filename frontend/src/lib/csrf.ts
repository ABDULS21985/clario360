/**
 * CSRF Token Management
 *
 * Implements the client side of the Double-Submit Cookie pattern:
 * 1. Backend sets a non-httpOnly cookie "clario360_csrf" on auth.
 * 2. Frontend reads that cookie and sends it back as X-CSRF-Token header.
 * 3. Backend validates cookie value === header value (constant-time).
 *
 * This module is imported by the Axios request interceptor (lib/api.ts).
 */

const CSRF_COOKIE_NAME = 'clario360_csrf';
const CSRF_HEADER_NAME = 'X-CSRF-Token';

/**
 * Reads the CSRF token from the cookie.
 * The cookie is NOT httpOnly so JavaScript can read it.
 */
export function getCSRFToken(): string | null {
  if (typeof document === 'undefined') return null;

  const cookies = document.cookie.split(';');
  for (const cookie of cookies) {
    const [name, ...valueParts] = cookie.trim().split('=');
    if (name === CSRF_COOKIE_NAME) {
      return decodeURIComponent(valueParts.join('='));
    }
  }
  return null;
}

/**
 * Returns the CSRF header name and value for use in request headers.
 * Returns null if no CSRF token is available.
 */
export function getCSRFHeaders(): Record<string, string> | null {
  const token = getCSRFToken();
  if (!token) return null;
  return { [CSRF_HEADER_NAME]: token };
}

/**
 * Checks if a request method requires CSRF protection.
 * Safe methods (GET, HEAD, OPTIONS) are exempt per HTTP spec.
 */
export function requiresCSRF(method: string): boolean {
  const safeMethods = ['GET', 'HEAD', 'OPTIONS'];
  return !safeMethods.includes(method.toUpperCase());
}

/**
 * CSRF header name constant for external use.
 */
export const CSRF_HEADER = CSRF_HEADER_NAME;
export const CSRF_COOKIE = CSRF_COOKIE_NAME;
