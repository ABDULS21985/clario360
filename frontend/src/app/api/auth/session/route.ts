import { NextRequest, NextResponse } from 'next/server';
import { COOKIES, SESSION } from '@/lib/constants';

const cookieSecure = SESSION.COOKIE_SECURE;
const accessMaxAge = SESSION.ACCESS_TOKEN_MAX_AGE;
const refreshMaxAge = SESSION.REFRESH_TOKEN_MAX_AGE;

function cookieOptions(maxAge: number, path: string) {
  return {
    httpOnly: true,
    secure: cookieSecure,
    sameSite: SESSION.COOKIE_SAMESITE,
    path,
    maxAge,
  } as const;
}

function decodeJWTPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(base64.length + (4 - (base64.length % 4)) % 4, '=');
    return JSON.parse(Buffer.from(padded, 'base64').toString('utf8')) as Record<
      string,
      unknown
    >;
  } catch {
    return null;
  }
}

// POST /api/auth/session
// Body: { access_token, refresh_token }
// Sets httpOnly cookies for both tokens.
export async function POST(req: NextRequest): Promise<NextResponse> {
  try {
    const body = (await req.json()) as {
      access_token?: string;
      refresh_token?: string;
    };

    if (!body.access_token || !body.refresh_token) {
      return NextResponse.json(
        { error: 'access_token and refresh_token are required' },
        { status: 400 },
      );
    }

    const response = NextResponse.json({ success: true });

    response.cookies.set(
      COOKIES.ACCESS,
      body.access_token,
      cookieOptions(accessMaxAge, '/'),
    );

    // Restrict refresh cookie path to /api/auth/* to limit exposure
    response.cookies.set(
      COOKIES.REFRESH,
      body.refresh_token,
      cookieOptions(refreshMaxAge, '/api/auth'),
    );

    return response;
  } catch {
    return NextResponse.json({ error: 'Invalid request body' }, { status: 400 });
  }
}

// GET /api/auth/session
// Reads the access cookie and returns decoded session info + a fresh access_token
// for the in-memory store to use. Also attempts to refresh if expired.
export async function GET(req: NextRequest): Promise<NextResponse> {
  const accessCookie = req.cookies.get(COOKIES.ACCESS);

  if (!accessCookie?.value) {
    return NextResponse.json({ error: 'no session' }, { status: 401 });
  }

  const payload = decodeJWTPayload(accessCookie.value);
  if (!payload) {
    return NextResponse.json({ error: 'invalid token' }, { status: 401 });
  }

  const exp = typeof payload['exp'] === 'number' ? payload['exp'] : 0;
  const nowPlusBuffer = Math.floor(Date.now() / 1000) + 30;

  // If token is still valid (not within 30s of expiry), return it directly
  if (exp > nowPlusBuffer) {
    const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
    // Fetch full user + tenant from backend
    let user = null;
    let tenant = null;
    try {
      const meResp = await fetch(`${apiUrl}/api/v1/users/me`, {
        headers: {
          Authorization: `Bearer ${accessCookie.value}`,
          'Content-Type': 'application/json',
        },
      });
      if (meResp.ok) {
        const meData = (await meResp.json()) as { user?: unknown; data?: unknown };
        user = meData.user ?? meData.data ?? meData;
      }
    } catch {
      // Non-fatal: return basic info from token
    }

    return NextResponse.json({
      user: user ?? {
        id: payload['sub'],
        email: payload['email'],
        tenant_id: payload['tenant_id'],
        roles: payload['roles'] ?? [],
        permissions: payload['permissions'] ?? [],
      },
      tenant,
      access_token: accessCookie.value,
      expires_at: new Date(exp * 1000).toISOString(),
    });
  }

  // Token expired — attempt refresh using the refresh cookie
  const refreshCookie = req.cookies.get(COOKIES.REFRESH);
  if (!refreshCookie?.value) {
    return NextResponse.json({ error: 'session expired' }, { status: 401 });
  }

  const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080';
  try {
    const refreshResp = await fetch(`${apiUrl}/api/v1/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshCookie.value }),
    });

    if (!refreshResp.ok) {
      const response = NextResponse.json({ error: 'session expired' }, { status: 401 });
      response.cookies.set(COOKIES.ACCESS, '', { maxAge: 0, path: '/' });
      response.cookies.set(COOKIES.REFRESH, '', { maxAge: 0, path: '/api/auth' });
      return response;
    }

    const tokens = (await refreshResp.json()) as {
      access_token: string;
      refresh_token: string;
    };

    // Fetch user with new token
    let user = null;
    try {
      const meResp = await fetch(`${apiUrl}/api/v1/users/me`, {
        headers: { Authorization: `Bearer ${tokens.access_token}` },
      });
      if (meResp.ok) {
        const meData = (await meResp.json()) as { user?: unknown; data?: unknown };
        user = meData.user ?? meData.data ?? meData;
      }
    } catch {
      // Non-fatal
    }

    const newPayload = decodeJWTPayload(tokens.access_token);
    const newExp =
      newPayload && typeof newPayload['exp'] === 'number' ? newPayload['exp'] : 0;

    const response = NextResponse.json({
      user,
      tenant: null,
      access_token: tokens.access_token,
      expires_at: new Date(newExp * 1000).toISOString(),
    });

    response.cookies.set(
      COOKIES.ACCESS,
      tokens.access_token,
      cookieOptions(accessMaxAge, '/'),
    );
    response.cookies.set(
      COOKIES.REFRESH,
      tokens.refresh_token,
      cookieOptions(refreshMaxAge, '/api/auth'),
    );

    return response;
  } catch {
    return NextResponse.json({ error: 'refresh failed' }, { status: 401 });
  }
}

// DELETE /api/auth/session — clear cookies (logout)
export async function DELETE(): Promise<NextResponse> {
  const response = NextResponse.json({ success: true });
  response.cookies.set(COOKIES.ACCESS, '', { maxAge: 0, path: '/' });
  response.cookies.set(COOKIES.REFRESH, '', { maxAge: 0, path: '/api/auth' });
  return response;
}
