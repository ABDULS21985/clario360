import { describe, it, expect, beforeAll, afterEach, afterAll, vi } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { User, Role } from '@/types/models';

const API_URL = 'http://localhost:8080';

// Mock next/navigation
const pushMock = vi.fn();
vi.mock('next/navigation', () => ({
  useRouter: () => ({ push: pushMock, replace: vi.fn() }),
  usePathname: () => '/admin/users',
  useSearchParams: () => new URLSearchParams(),
}));

// Mock auth
vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({
    user: { id: 'current-user-id', first_name: 'Admin', last_name: 'User', email: 'admin@test.com' },
    hasPermission: () => true,
    tenant: { id: 'tenant-1' },
  }),
}));

const mockRoles: Role[] = [
  {
    id: 'role-1',
    tenant_id: 'tenant-1',
    name: 'Tenant Admin',
    slug: 'tenant-admin',
    description: '',
    permissions: ['*'],
    is_system: true,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

const mockUsers: User[] = [
  {
    id: 'user-1',
    tenant_id: 'tenant-1',
    email: 'alice@example.com',
    first_name: 'Alice',
    last_name: 'Smith',
    status: 'active',
    mfa_enabled: true,
    last_login_at: '2024-03-01T10:00:00Z',
    roles: [mockRoles[0]],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'user-2',
    tenant_id: 'tenant-1',
    email: 'bob@example.com',
    first_name: 'Bob',
    last_name: 'Jones',
    status: 'suspended',
    mfa_enabled: false,
    last_login_at: null,
    roles: [],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

const server = setupServer(
  http.get(`${API_URL}/api/v1/users`, () =>
    HttpResponse.json({
      data: mockUsers,
      meta: { page: 1, per_page: 25, total: 2, total_pages: 1 },
    })
  ),
  http.get(`${API_URL}/api/v1/roles`, () =>
    HttpResponse.json({
      data: mockRoles,
      meta: { page: 1, per_page: 100, total: 1, total_pages: 1 },
    })
  )
);

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

function makeQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

async function renderUsersPage() {
  const { default: AdminUsersPage } = await import(
    '@/app/(dashboard)/admin/users/page'
  );
  const queryClient = makeQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <AdminUsersPage />
    </QueryClientProvider>
  );
}

describe('User Management Page', () => {
  it('test_loadAndDisplayUsers: renders user rows after load', async () => {
    await renderUsersPage();
    await waitFor(() => {
      expect(screen.getByText('Alice Smith')).toBeInTheDocument();
      expect(screen.getByText('Bob Jones')).toBeInTheDocument();
    });
  });

  it('test_showsUserEmail: displays email in name column', async () => {
    await renderUsersPage();
    await waitFor(() => {
      expect(screen.getByText('alice@example.com')).toBeInTheDocument();
    });
  });

  it('test_pageHeaderVisible: page header is present', async () => {
    await renderUsersPage();
    expect(screen.getByText('User Management')).toBeInTheDocument();
  });

  it('test_addUserButtonVisible: add user button shown for users:write', async () => {
    await renderUsersPage();
    expect(screen.getByText('Add User')).toBeInTheDocument();
  });

  it('test_createUserDialogOpens: clicking Add User opens dialog', async () => {
    const user = userEvent.setup();
    await renderUsersPage();
    const btn = screen.getByText('Add User');
    await user.click(btn);
    await waitFor(() => {
      expect(screen.getByText('Add New User')).toBeInTheDocument();
    });
  });

  it('test_createUser_formInputsWork: form fields accept input', async () => {
    const user = userEvent.setup();
    await renderUsersPage();
    await user.click(screen.getByText('Add User'));
    await waitFor(() => screen.getByText('Add New User'));

    const firstNameInput = screen.getByPlaceholderText('John');
    const lastNameInput = screen.getByPlaceholderText('Doe');
    const emailInput = screen.getByPlaceholderText('john@company.com');

    await user.type(firstNameInput, 'Test');
    await user.type(lastNameInput, 'User');
    await user.type(emailInput, 'test@example.com');

    expect(firstNameInput).toHaveValue('Test');
    expect(lastNameInput).toHaveValue('User');
    expect(emailInput).toHaveValue('test@example.com');
  });

  it('test_errorState: shows error state on API failure', async () => {
    server.use(
      http.get(`${API_URL}/api/v1/users`, () =>
        HttpResponse.json({ message: 'Server error' }, { status: 500 })
      )
    );
    await renderUsersPage();
    await waitFor(() => {
      expect(
        screen.getByText(/failed to load/i) ||
        screen.getByText(/error/i) ||
        screen.queryByRole('alert')
      ).toBeTruthy();
    });
  });
});
