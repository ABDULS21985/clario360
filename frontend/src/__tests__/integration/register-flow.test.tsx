import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { RegisterForm } from '@/components/auth/register-form';

const pushMock = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: pushMock }),
  usePathname: () => '/',
  useSearchParams: () => ({ get: () => null }),
}));

const API_URL = 'http://localhost:8080';

const server = setupServer(
  http.post(`${API_URL}/api/v1/auth/register`, () =>
    HttpResponse.json({ message: 'created' }, { status: 201 }),
  ),
  http.get(`${API_URL}/api/v1/auth/check-email`, () =>
    HttpResponse.json({ available: true }),
  ),
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => {
  server.resetHandlers();
  pushMock.mockClear();
});
afterAll(() => server.close());

async function fillRegisterForm() {
  const user = userEvent.setup();
  render(<RegisterForm />);
  await user.type(screen.getByLabelText('First name'), 'John');
  await user.type(screen.getByLabelText('Last name'), 'Doe');
  await user.type(screen.getByLabelText('Email'), 'john@example.com');
  await user.type(screen.getByLabelText('Password'), 'Str0ng!Pass#word');
  await user.type(screen.getByLabelText('Confirm password'), 'Str0ng!Pass#word');
  await user.type(screen.getByLabelText('Organization name'), 'Acme Corp');
  return user;
}

describe('Register flow integration', () => {
  it('test_registerSuccess: valid form → redirect to /login?registered=true', async () => {
    await fillRegisterForm();
    const submitButton = screen.getByRole('button', { name: /create account/i });
    await userEvent.setup().click(submitButton);
    await waitFor(() => {
      expect(pushMock).toHaveBeenCalledWith('/login?registered=true');
    });
  });

  it('test_registerDuplicateEmail: 409 → email field shows error', async () => {
    server.use(
      http.post(`${API_URL}/api/v1/auth/register`, () =>
        HttpResponse.json(
          { code: 'EMAIL_TAKEN', message: 'Email already registered' },
          { status: 409 },
        ),
      ),
    );
    await fillRegisterForm();
    const submitButton = screen.getByRole('button', { name: /create account/i });
    await userEvent.setup().click(submitButton);
    await waitFor(() => {
      expect(screen.getByText(/already registered/i)).toBeInTheDocument();
    });
  });

  it('test_passwordStrengthUpdates: strength meter updates as user types', async () => {
    const user = userEvent.setup();
    render(<RegisterForm />);
    const passwordInput = screen.getByLabelText('Password');
    // Initially no meter
    expect(screen.queryByText(/weak|fair|good|strong/i)).toBeNull();
    // Type a weak password
    await user.type(passwordInput, 'abc');
    expect(screen.getByText('Weak')).toBeInTheDocument();
    // Type a strong password
    await user.clear(passwordInput);
    await user.type(passwordInput, 'C0mpl3x!P@ssw0rd#2026');
    expect(screen.getByText('Strong')).toBeInTheDocument();
  });
});
