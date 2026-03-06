import { NextRequest, NextResponse } from 'next/server';
import { COOKIES, SESSION } from '@/lib/constants';

// POST /api/auth/refresh
// Reads the httpOnly refresh cookie and exchanges it for new tokens.
// The frontend JS never sees the refresh token — cookie-to-cookie only.
export async function POST(req: NextRequest): Promise<NextResponse> {
  const refreshCookie = req.cookies.get(COOKIES.REFRESH);

  if (!refreshCookie?.value) {
    return NextResponse.json({ error: 'no refresh token' }, { status: 401 });
  }

  const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';

  try {
    const backendResp = await fetch(`${apiUrl}/api/v1/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshCookie.value }),
    });

    if (!backendResp.ok) {
      // Refresh rejected — clear cookies
      const response = NextResponse.json({ error: 'refresh failed' }, { status: 401 });
      response.cookies.set(COOKIES.ACCESS, '', { maxAge: 0, path: '/' });
      response.cookies.set(COOKIES.REFRESH, '', { maxAge: 0, path: '/api/auth' });
      return response;
    }

    const tokens = (await backendResp.json()) as {
      access_token: string;
      refresh_token: string;
    };

    const cookieSecure = SESSION.COOKIE_SECURE;
    const cookieSameSite = SESSION.COOKIE_SAMESITE;

    const response = NextResponse.json({ access_token: tokens.access_token });

    response.cookies.set(COOKIES.ACCESS, tokens.access_token, {
      httpOnly: true,
      secure: cookieSecure,
      sameSite: cookieSameSite,
      path: '/',
      maxAge: SESSION.ACCESS_TOKEN_MAX_AGE,
    });

    response.cookies.set(COOKIES.REFRESH, tokens.refresh_token, {
      httpOnly: true,
      secure: cookieSecure,
      sameSite: cookieSameSite,
      path: '/api/auth',
      maxAge: SESSION.REFRESH_TOKEN_MAX_AGE,
    });

    return response;
  } catch {
    return NextResponse.json({ error: 'refresh error' }, { status: 500 });
  }
}
