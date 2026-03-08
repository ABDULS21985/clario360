/**
 * Security headers configuration for Next.js.
 *
 * These headers are applied via next.config.mjs or the Next.js middleware.
 * They mirror the backend security headers for the frontend routes.
 */

export interface SecurityHeadersConfig {
  environment: 'development' | 'staging' | 'production';
  apiUrl: string;
  cspReportUri?: string;
}

/**
 * Builds the Content-Security-Policy header value for the frontend.
 */
export function buildCSP(config: SecurityHeadersConfig): string {
  const isDev = config.environment === 'development';

  const directives: string[] = [
    "default-src 'self'",
    isDev
      ? "script-src 'self' 'unsafe-eval' 'unsafe-inline'" // HMR requires eval + inline
      : "script-src 'self'",
    "style-src 'self' 'unsafe-inline'", // Tailwind JIT requires unsafe-inline
    "img-src 'self' data: https:",
    "font-src 'self'",
    `connect-src 'self' ${config.apiUrl} wss:`,
    "worker-src 'self' blob:",
    "frame-ancestors 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "object-src 'none'",
  ];

  if (!isDev) {
    directives.push('upgrade-insecure-requests');
  }

  if (config.cspReportUri && !isDev) {
    directives.push(`report-uri ${config.cspReportUri}`);
  }

  return directives.join('; ');
}

/**
 * Returns all security headers for Next.js configuration.
 */
export function getSecurityHeaders(config: SecurityHeadersConfig): Array<{ key: string; value: string }> {
  const isDev = config.environment === 'development';

  const headers: Array<{ key: string; value: string }> = [
    { key: 'X-Content-Type-Options', value: 'nosniff' },
    { key: 'X-Frame-Options', value: 'DENY' },
    { key: 'X-XSS-Protection', value: '0' },
    { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
    {
      key: 'Permissions-Policy',
      value: 'camera=(), microphone=(), geolocation=(), payment=(), usb=(), interest-cohort=()',
    },
    { key: 'Content-Security-Policy', value: buildCSP(config) },
  ];

  if (!isDev) {
    headers.push({
      key: 'Strict-Transport-Security',
      value: 'max-age=31536000; includeSubDomains; preload',
    });
    headers.push({ key: 'Cross-Origin-Embedder-Policy', value: 'require-corp' });
    headers.push({ key: 'Cross-Origin-Opener-Policy', value: 'same-origin' });
    headers.push({ key: 'Cross-Origin-Resource-Policy', value: 'same-origin' });
  }

  return headers;
}
