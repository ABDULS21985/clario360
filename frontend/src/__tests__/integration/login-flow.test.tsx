import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { LoginForm } from '@/components/auth/login-form';
import { useAuthStore } from '@/stores/auth-store';

// Mock Next.js router
const pushMock = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: pushMock, replace: vi.fn() }),
  usePathname: () => '/',
  useSearchParams: () => ({ get: () => null }),
}));

// Mock BFF session endpoint
vi.mock('@/lib/auth', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/auth')>();
  return {
    ...actual,
    setAccessToken: vi.fn(),
    clearAccessToken: vi.fn(),
    getAccessToken: vi.fn(() => null),
  };
});

const API_URL = 'http://localhost:8080';

const server = setupServer(
  // Default: successful login
  http.post(`${API_URL}/api/v1/auth/login`, () =>
    HttpResponse.json({
      access_token: 'mock-access-token',
      refresh_token: 'mock-refresh-token',
      user: {
        id: 'u1',
        tenant_id: 't1',
        email: 'user@example.com',
        first_name: 'John',
        last_name: 'Doe',
        status: 'active',
        mfa_enabled: false,
        last_login_at: null,
        roles: [],
        created_at: '',
        updated_at: '',
      },
    }),
  ),
  // BFF session endpoint (Next.js API route — relative path becomes http://localhost:8080 in tests)
  http.post('/api/auth/session', () => HttpResponse.json({ success: true })),
  http.post(`${API_URL}/api/auth/session`, () => HttpResponse.json({ success: true })),
  http.get('/api/v1/users/me', () =>
    HttpResponse.json({
      id: 'u1',
      email: 'user@example.com',
      first_name: 'John',
      last_name: 'Doe',
      tenant_id: 't1',
      status: 'active',
      mfa_enabled: false,
      last_login_at: null,
      roles: [],
      created_at: '',
      updated_at: '',
    }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => {
  server.resetHandlers();
  pushMock.mockClear();
  useAuthStore.setState({ user: null, isAuthenticated: false, isHydrated: true });
});
afterAll(() => server.close());

async function fillAndSubmitLogin(email = 'user@example.com', password = 'password123') {
  const user = userEvent.setup();
  render(<LoginForm />);
  await user.type(screen.getByLabelText('Email'), email);
  await user.type(screen.getByLabelText('Password'), password);
  await user.click(screen.getByRole('button', { name: /sign in/i }));
}

describe('Login flow integration', () => {
  it('test_loginSuccess: successful login → redirect to /dashboard', async () => {
    await fillAndSubmitLogin();
    await waitFor(() => {
      expect(pushMock).toHaveBeenCalledWith('/dashboard');
    });
  });

  it('test_loginFailure: 401 → error banner shown', async () => {
    server.use(
      http.post(`${API_URL}/api/v1/auth/login`, () =>
        HttpResponse.json({ code: 'INVALID_CREDENTIALS', message: 'Invalid credentials' }, { status: 401 }),
      ),
    );
    await fillAndSubmitLogin();
    await waitFor(() => {
      expect(screen.getByRole('alert')).toBeInTheDocument();
    });
    expect(screen.getByText(/invalid email or password/i)).toBeInTheDocument();
  });

  it('test_loginMFA: mfa_required response → MFA input shown', async () => {
    server.use(
      http.post(`${API_URL}/api/v1/auth/login`, () =>
        HttpResponse.json({ mfa_required: true, mfa_token: 'mfa-tok-123' }),
      ),
    );
    await fillAndSubmitLogin();
    await waitFor(() => {
      expect(screen.getByText(/two-factor authentication/i)).toBeInTheDocument();
    });
  });

  it('test_loginAccountLocked: 403 ACCOUNT_LOCKED → locked message shown', async () => {
    server.use(
      http.post(`${API_URL}/api/v1/auth/login`, () =>
        HttpResponse.json(
          { code: 'ACCOUNT_LOCKED', message: 'Account locked', details: { lock_minutes: ['30'] } },
          { status: 403 },
        ),
      ),
    );
    await fillAndSubmitLogin();
    await waitFor(() => {
      expect(screen.getByText(/account is locked/i)).toBeInTheDocument();
    });
  });

  it('test_loginRateLimited: 429 → countdown shown, button disabled', async () => {
    server.use(
      http.post(`${API_URL}/api/v1/auth/login`, () =>
        HttpResponse.json(
          {
            code: 'RATE_LIMITED',
            message: 'Too many attempts',
            details: { retry_after: ['60'] },
          },
          { status: 429 },
        ),
      ),
    );
    await fillAndSubmitLogin();
    await waitFor(() => {
      const button = screen.getByRole('button', { name: /sign in/i });
      expect(button).toBeDisabled();
    });
  });
});
