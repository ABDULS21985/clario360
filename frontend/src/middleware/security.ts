/**
 * Security middleware utilities for Next.js.
 *
 * These functions can be composed with the existing auth middleware
 * in src/middleware.ts to add security headers to all responses.
 *
 * Usage in middleware.ts:
 *   import { addSecurityHeaders } from '@/middleware/security';
 *   // After auth logic:
 *   const response = NextResponse.next();
 *   addSecurityHeaders(response);
 *   return response;
 */

import { NextResponse } from 'next/server';

const IS_PRODUCTION = process.env.NODE_ENV === 'production';
const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';

/**
 * Adds security headers to a Next.js response.
 * Call this in your middleware after auth checks.
 */
export function addSecurityHeaders(response: NextResponse): void {
  const headers = response.headers;

  // Prevent MIME sniffing
  headers.set('X-Content-Type-Options', 'nosniff');

  // Prevent clickjacking
  headers.set('X-Frame-Options', 'DENY');

  // Disable legacy XSS filter (CSP is the modern protection)
  headers.set('X-XSS-Protection', '0');

  // Referrer policy
  headers.set('Referrer-Policy', 'strict-origin-when-cross-origin');

  // Permissions policy
  headers.set(
    'Permissions-Policy',
    'camera=(), microphone=(), geolocation=(), payment=(), usb=(), interest-cohort=()',
  );

  // Content Security Policy
  headers.set('Content-Security-Policy', buildCSP());

  // Production-only headers
  if (IS_PRODUCTION) {
    headers.set('Strict-Transport-Security', 'max-age=31536000; includeSubDomains; preload');
    headers.set('Cross-Origin-Embedder-Policy', 'require-corp');
    headers.set('Cross-Origin-Opener-Policy', 'same-origin');
    headers.set('Cross-Origin-Resource-Policy', 'same-origin');
  }

  // Remove server identification
  headers.delete('Server');
  headers.delete('X-Powered-By');
}

/**
 * Builds the CSP for the frontend application.
 */
function buildCSP(): string {
  const isDev = !IS_PRODUCTION;

  const directives: string[] = [
    "default-src 'self'",
    isDev
      ? "script-src 'self' 'unsafe-eval' 'unsafe-inline'"
      : "script-src 'self'",
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data: https:",
    "font-src 'self'",
    `connect-src 'self' ${API_URL} wss:`,
    "worker-src 'self' blob:",
    "frame-ancestors 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "object-src 'none'",
  ];

  if (!isDev) {
    directives.push('upgrade-insecure-requests');
  }

  return directives.join('; ');
}
