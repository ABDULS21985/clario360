import { NextRequest, NextResponse } from 'next/server';
import { COOKIES } from '@/lib/constants';

// Routes that do NOT require authentication
const PUBLIC_PATH_PREFIXES = [
  '/login',
  '/register',
  '/verify',
  '/verify-email',
  '/invite',
  '/forgot-password',
  '/reset-password',
  '/api/',
  '/_next/',
  '/favicon.ico',
];

function isPublicPath(pathname: string): boolean {
  return PUBLIC_PATH_PREFIXES.some((prefix) => pathname.startsWith(prefix));
}

function decodeJWTExpiry(token: string): number | null {
  // Edge middleware: no Node.js crypto, base64 decode only (no sig verification)
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(
      base64.length + (4 - (base64.length % 4)) % 4,
      '=',
    );
    const decoded = atob(padded);
    const payload = JSON.parse(decoded) as Record<string, unknown>;
    return typeof payload['exp'] === 'number' ? payload['exp'] : null;
  } catch {
    return null;
  }
}

export async function middleware(req: NextRequest): Promise<NextResponse> {
  const { pathname } = req.nextUrl;

  // Always allow public paths
  if (isPublicPath(pathname)) {
    // If authenticated user visits /login → redirect to /dashboard
    if (pathname === '/login') {
      const accessCookie = req.cookies.get(COOKIES.ACCESS);
      if (accessCookie?.value) {
        const exp = decodeJWTExpiry(accessCookie.value);
        if (exp && exp > Math.floor(Date.now() / 1000)) {
          return NextResponse.redirect(new URL('/dashboard', req.url));
        }
      }
    }
    return NextResponse.next();
  }

  // Protected routes — require valid session
  const accessCookie = req.cookies.get(COOKIES.ACCESS);

  if (!accessCookie?.value) {
    const loginUrl = new URL('/login', req.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }

  const exp = decodeJWTExpiry(accessCookie.value);
  const nowPlusBuffer = Math.floor(Date.now() / 1000) + 30;

  // Token valid — proceed
  if (exp && exp > nowPlusBuffer) {
    return NextResponse.next();
  }

  // Token expired — attempt silent refresh via BFF
  const refreshCookie = req.cookies.get(COOKIES.REFRESH);
  if (!refreshCookie?.value) {
    const loginUrl = new URL('/login', req.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }

  try {
    const refreshUrl = new URL('/api/auth/refresh', req.url);
    const refreshResp = await fetch(refreshUrl.toString(), {
      method: 'POST',
      headers: {
        cookie: `${COOKIES.REFRESH}=${refreshCookie.value}`,
      },
    });

    if (refreshResp.ok) {
      // Forward cookies from refresh response to the original request's response
      const nextResp = NextResponse.next();
      const setCookieHeader = refreshResp.headers.get('set-cookie');
      if (setCookieHeader) {
        nextResp.headers.set('set-cookie', setCookieHeader);
      }
      return nextResp;
    }
  } catch {
    // Fall through to redirect
  }

  const loginUrl = new URL('/login', req.url);
  loginUrl.searchParams.set('redirect', pathname);
  return NextResponse.redirect(loginUrl);
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico|api/).*)'],
};
