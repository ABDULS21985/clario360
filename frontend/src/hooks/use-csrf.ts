'use client';

import { useCallback, useEffect, useState } from 'react';
import { getCSRFToken, CSRF_COOKIE } from '@/lib/csrf';

/**
 * React hook for CSRF token management.
 *
 * Reads the CSRF token from the cookie and provides it to components.
 * Automatically detects when the token changes (e.g., after login or refresh).
 *
 * Usage:
 *   const { csrfToken, isReady } = useCSRF();
 *   // csrfToken is the current CSRF token or null
 *   // isReady is true once we've attempted to read the token
 */
export function useCSRF() {
  const [csrfToken, setCSRFToken] = useState<string | null>(null);
  const [isReady, setIsReady] = useState(false);

  const refreshToken = useCallback(() => {
    const token = getCSRFToken();
    setCSRFToken(token);
    setIsReady(true);
    return token;
  }, []);

  // Read token on mount
  useEffect(() => {
    refreshToken();
  }, [refreshToken]);

  // Poll for token changes (handles login/refresh setting new cookies)
  useEffect(() => {
    const interval = setInterval(() => {
      const current = getCSRFToken();
      if (current !== csrfToken) {
        setCSRFToken(current);
      }
    }, 5000); // Check every 5 seconds

    return () => clearInterval(interval);
  }, [csrfToken]);

  return {
    /** The current CSRF token value, or null if not available. */
    csrfToken,
    /** True once the hook has attempted to read the token. */
    isReady,
    /** Manually refresh the token from the cookie. */
    refreshToken,
    /** The cookie name for the CSRF token. */
    cookieName: CSRF_COOKIE,
  };
}
