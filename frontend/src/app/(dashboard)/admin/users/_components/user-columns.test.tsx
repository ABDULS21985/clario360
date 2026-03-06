import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { getUserColumns } from './user-columns';
import type { User, Role } from '@/types/models';

// Mock hooks
vi.mock('@/hooks/use-auth', () => ({
  useAuth: () => ({ hasPermission: () => true, user: { id: 'current-user-id' } }),
}));

const mockRole: Role = {
  id: 'role-1',
  tenant_id: 'tenant-1',
  name: 'Tenant Admin',
  slug: 'tenant-admin',
  description: 'Full access',
  permissions: ['*'],
  is_system: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const makeUser = (overrides: Partial<User> = {}): User => ({
  id: 'user-1',
  tenant_id: 'tenant-1',
  email: 'john@example.com',
  first_name: 'John',
  last_name: 'Doe',
  status: 'active',
  mfa_enabled: false,
  last_login_at: null,
  password_changed_at: '2024-01-01T00:00:00Z',
  roles: [],
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  ...overrides,
});

describe('getUserColumns', () => {
  const options = {
    onEdit: vi.fn(),
    onAssignRoles: vi.fn(),
    onResetPassword: vi.fn(),
    onChangeStatus: vi.fn(),
    onDelete: vi.fn(),
    onRowClick: vi.fn(),
    currentUserId: 'current-user-id',
    hasPermission: () => true,
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('test_columnsCount: returns expected column count', () => {
    const cols = getUserColumns(options);
    // select, name, roles, status, mfa, last_login, created_at, actions = 8
    expect(cols.length).toBe(8);
  });

  it('test_rendersStatusBadge: status column renders status', () => {
    const cols = getUserColumns(options);
    const statusCol = cols.find((c) => (c as { id?: string }).id === 'status');
    expect(statusCol).toBeDefined();
  });

  it('test_rendersMfaIcons: mfa column defined', () => {
    const cols = getUserColumns(options);
    const mfaCol = cols.find((c) => (c as { id?: string }).id === 'mfa_enabled');
    expect(mfaCol).toBeDefined();
  });

  it('test_rendersNeverLogin: last_login column defined', () => {
    const cols = getUserColumns(options);
    const loginCol = cols.find((c) => (c as { id?: string }).id === 'last_login_at');
    expect(loginCol).toBeDefined();
  });

  it('test_actionsColumnDefined: actions column is last', () => {
    const cols = getUserColumns(options);
    const actionsCol = cols[cols.length - 1];
    expect((actionsCol as { id?: string }).id).toBe('actions');
  });

  it('test_nameColumnSortable: name column has sorting enabled', () => {
    const cols = getUserColumns(options);
    const nameCol = cols.find((c) => (c as { id?: string }).id === 'name');
    expect(nameCol?.enableSorting).toBe(true);
  });

  it('test_rolesColumnNotSortable: roles column has sorting disabled', () => {
    const cols = getUserColumns(options);
    const rolesCol = cols.find((c) => (c as { id?: string }).id === 'roles');
    expect(rolesCol?.enableSorting).toBe(false);
  });
});
