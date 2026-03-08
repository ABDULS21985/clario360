'use client';

import { create } from 'zustand';
import { apiPost, apiGet } from '@/lib/api';
import {
  getAccessToken,
  setAccessToken,
  clearAccessToken,
  getTokenPayload,
} from '@/lib/auth';
import type { User, Tenant, SuiteName } from '@/types/models';
import type { LoginApiResponse, isMFARequired as IsMFARequired } from '@/types/auth';
import { isMFARequired } from '@/types/auth';
import { API_ENDPOINTS } from '@/lib/constants';

interface LoginResult {
  requiresMFA: boolean;
  mfaToken?: string;
}

interface AuthState {
  user: User | null;
  tenant: Tenant | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  isHydrated: boolean;
  sessionExpired: boolean;
  error: string | null;

  // Actions
  login: (email: string, password: string) => Promise<LoginResult>;
  verifyMFA: (mfaToken: string, code: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshSession: () => Promise<void>;
  updateProfile: (
    data: Partial<Pick<User, 'first_name' | 'last_name' | 'email'>>,
  ) => Promise<void>;
  hasPermission: (permission: string) => boolean;
  hasAnyPermission: (permissions: string[]) => boolean;
  hasAllPermissions: (permissions: string[]) => boolean;
  hasSuiteAccess: (suite: SuiteName) => boolean;
  clearError: () => void;
  setSessionExpired: (value: boolean) => void;
}

const SUITE_PERMISSIONS: Record<SuiteName, string> = {
  cyber: 'cyber:read',
  data: 'data:read',
  acta: 'acta:read',
  lex: 'lex:read',
  visus: 'visus:read',
};

/**
 * Resolve permissions with wildcard matching, consistent with backend logic.
 * Supports: exact match, resource wildcard (resource:*), action wildcard (*:action), super (*).
 */
function checkPermission(userPermissions: string[], required: string): boolean {
  if (userPermissions.includes('*')) return true;
  if (userPermissions.includes(required)) return true;

  const [resource, action] = required.split(':');
  if (resource && action) {
    if (userPermissions.includes(`${resource}:*`)) return true;
    if (userPermissions.includes(`*:${action}`)) return true;
  }

  return false;
}

function getPermissionsFromToken(): string[] {
  const token = getAccessToken();
  if (!token) return [];
  const payload = getTokenPayload(token);
  return payload?.permissions ?? [];
}

async function hydrateSessionFromBFF(): Promise<{
  user: User;
  tenant: Tenant;
  accessToken: string;
} | null> {
  try {
    // GET /api/auth/session is a Next.js BFF route — must use a relative URL
    // so it resolves to localhost:3000, not the backend gateway.
    const resp = await fetch(API_ENDPOINTS.BFF_SESSION, { credentials: 'include' });
    if (!resp.ok) return null;
    const sessionData = (await resp.json()) as {
      user: User;
      tenant: Tenant;
      access_token: string;
      expires_at: string;
    };
    return {
      user: sessionData.user,
      tenant: sessionData.tenant,
      accessToken: sessionData.access_token,
    };
  } catch {
    return null;
  }
}

async function storeSessionInBFF(
  accessToken: string,
  refreshToken: string,
): Promise<void> {
  // POST /api/auth/session is a Next.js BFF route — must use a relative URL.
  const resp = await fetch(API_ENDPOINTS.BFF_SESSION, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ access_token: accessToken, refresh_token: refreshToken }),
  });
  if (!resp.ok) {
    throw new Error(`Failed to store session: ${resp.status}`);
  }
}

async function clearSessionInBFF(): Promise<void> {
  try {
    // DELETE /api/auth/session clears httpOnly cookies
    const response = await fetch(API_ENDPOINTS.BFF_SESSION, { method: 'DELETE' });
    if (!response.ok) {
      // Non-fatal — continue logout even if cookie clearing fails
    }
  } catch {
    // Non-fatal
  }
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  tenant: null,
  isAuthenticated: false,
  isLoading: false,
  isHydrated: false,
  sessionExpired: false,
  error: null,

  clearError: () => set({ error: null }),

  setSessionExpired: (value: boolean) => set({ sessionExpired: value }),

  login: async (email: string, password: string): Promise<LoginResult> => {
    set({ isLoading: true, error: null });
    try {
      const resp = await apiPost<LoginApiResponse>(API_ENDPOINTS.AUTH_LOGIN, {
        email,
        password,
      });

      if (isMFARequired(resp)) {
        set({ isLoading: false });
        return { requiresMFA: true, mfaToken: resp.mfa_token };
      }

      // Full login — store tokens
      await storeSessionInBFF(resp.access_token, resp.refresh_token);
      setAccessToken(resp.access_token);

      // Fetch full user profile if not included in response
      let user = resp.user;
      if (!user) {
        user = await apiGet<User>(API_ENDPOINTS.USERS_ME);
      }

      set({
        user,
        isAuthenticated: true,
        isLoading: false,
        error: null,
      });

      return { requiresMFA: false };
    } catch (err) {
      const msg = extractErrorMessage(err);
      set({ isLoading: false, error: msg });
      throw err;
    }
  },

  verifyMFA: async (mfaToken: string, code: string): Promise<void> => {
    set({ isLoading: true, error: null });
    try {
      const resp = await apiPost<{
        access_token: string;
        refresh_token: string;
        user: User;
      }>(API_ENDPOINTS.AUTH_VERIFY_MFA, { mfa_token: mfaToken, code });

      await storeSessionInBFF(resp.access_token, resp.refresh_token);
      setAccessToken(resp.access_token);

      let user = resp.user;
      if (!user) {
        user = await apiGet<User>(API_ENDPOINTS.USERS_ME);
      }

      set({
        user,
        isAuthenticated: true,
        isLoading: false,
        error: null,
      });
    } catch (err) {
      const msg = extractErrorMessage(err);
      set({ isLoading: false, error: msg });
      throw err;
    }
  },

  logout: async (): Promise<void> => {
    try {
      await apiPost(API_ENDPOINTS.AUTH_LOGOUT);
    } catch {
      // Continue logout even if server-side revocation fails
    }
    await clearSessionInBFF();
    clearAccessToken();
    set({
      user: null,
      tenant: null,
      isAuthenticated: false,
      sessionExpired: false,
      error: null,
    });
    if (typeof window !== 'undefined') {
      window.location.href = '/login';
    }
  },

  refreshSession: async (): Promise<void> => {
    set({ isLoading: true });
    try {
      const session = await hydrateSessionFromBFF();
      if (session) {
        setAccessToken(session.accessToken);
        set({
          user: session.user,
          tenant: session.tenant,
          isAuthenticated: true,
          isHydrated: true,
          isLoading: false,
          error: null,
        });
      } else {
        set({
          user: null,
          tenant: null,
          isAuthenticated: false,
          isHydrated: true,
          isLoading: false,
        });
      }
    } catch {
      set({
        user: null,
        tenant: null,
        isAuthenticated: false,
        isHydrated: true,
        isLoading: false,
      });
    }
  },

  updateProfile: async (
    data: Partial<Pick<User, 'first_name' | 'last_name' | 'email'>>,
  ): Promise<void> => {
    set({ isLoading: true, error: null });
    try {
      const updated = await apiPatch<User>(API_ENDPOINTS.USERS_ME, data);
      set({ user: updated, isLoading: false });
    } catch (err) {
      const msg = extractErrorMessage(err);
      set({ isLoading: false, error: msg });
      throw err;
    }
  },

  hasPermission: (permission: string): boolean => {
    const perms = getPermissionsFromToken();
    if (perms.length === 0) {
      // Fall back to user roles if token not yet loaded
      const { user } = get();
      if (!user) return false;
      const rolePerms = user.roles.flatMap((r) => r.permissions);
      return checkPermission(rolePerms, permission);
    }
    return checkPermission(perms, permission);
  },

  hasAnyPermission: (permissions: string[]): boolean => {
    return permissions.some((p) => get().hasPermission(p));
  },

  hasAllPermissions: (permissions: string[]): boolean => {
    return permissions.every((p) => get().hasPermission(p));
  },

  hasSuiteAccess: (suite: SuiteName): boolean => {
    return get().hasPermission(SUITE_PERMISSIONS[suite]);
  },
}));

// Lazy import to avoid circular dependency (apiPatch lives in api.ts which imports auth-store indirectly)
async function apiPatch<T>(url: string, data?: unknown): Promise<T> {
  const { apiPatch: patch } = await import('@/lib/api');
  return patch<T>(url, data);
}

function extractErrorMessage(err: unknown): string {
  if (err && typeof err === 'object' && 'message' in err) {
    return (err as { message: string }).message;
  }
  return 'An unexpected error occurred';
}
