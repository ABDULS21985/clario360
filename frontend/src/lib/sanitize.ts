/**
 * Client-side sanitization utilities.
 *
 * IMPORTANT: These are for DISPLAY purposes only and are NOT a security boundary.
 * The backend is the source of truth for input validation and output encoding.
 * These functions provide defence-in-depth on the client side.
 */

/**
 * HTML-encodes the 5 dangerous characters for XSS prevention.
 * Used when rendering user-provided content outside of React's JSX
 * (which auto-escapes by default).
 */
export function escapeHTML(input: string): string {
  const map: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  };
  return input.replace(/[&<>"']/g, (char) => map[char] ?? char);
}

/**
 * Strips all HTML tags from a string.
 * Used for search inputs, filter values, and other plain-text contexts.
 */
export function stripHTML(input: string): string {
  return input.replace(/<[^>]*>/g, '');
}

/**
 * Removes null bytes and control characters from a string.
 */
export function sanitizeString(input: string): string {
  // Remove null bytes
  let result = input.replace(/\0/g, '');
  // Remove control characters except tab, newline, carriage return
  result = result.replace(/[\x01-\x08\x0B\x0C\x0E-\x1F\x7F]/g, '');
  return result;
}

/**
 * Validates and sanitizes a URL for safe use in href attributes.
 * Blocks javascript:, data:, and vbscript: URIs.
 */
export function sanitizeURL(url: string): string | null {
  const trimmed = url.trim();

  // Block dangerous protocols
  const lower = trimmed.toLowerCase();
  if (
    lower.startsWith('javascript:') ||
    lower.startsWith('data:') ||
    lower.startsWith('vbscript:')
  ) {
    return null;
  }

  // Allow http, https, mailto, tel, and relative URLs
  if (
    lower.startsWith('http://') ||
    lower.startsWith('https://') ||
    lower.startsWith('mailto:') ||
    lower.startsWith('tel:') ||
    lower.startsWith('/') ||
    lower.startsWith('#') ||
    !lower.includes(':')
  ) {
    return trimmed;
  }

  return null;
}

/**
 * Truncates a string to a maximum length with ellipsis.
 * Used for display to prevent UI overflow with malicious long strings.
 */
export function truncate(input: string, maxLength: number): string {
  if (input.length <= maxLength) return input;
  return input.slice(0, maxLength - 3) + '...';
}

/**
 * Validates that a string looks like a valid email (basic check).
 * This is NOT a security boundary — full validation is server-side.
 */
export function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

/**
 * Validates that a string is a valid UUID v4.
 */
export function isValidUUID(input: string): boolean {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(input);
}
