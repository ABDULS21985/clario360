import type { FilterConfig } from '@/types/table';
import type { Role } from '@/types/models';

export function getUserFilters(roles: Role[]): FilterConfig[] {
  return [
    {
      key: 'status',
      label: 'Status',
      type: 'multi-select',
      options: [
        { label: 'Active', value: 'active' },
        { label: 'Suspended', value: 'suspended' },
        { label: 'Deactivated', value: 'deactivated' },
        { label: 'Pending Verification', value: 'pending_verification' },
      ],
    },
    {
      key: 'role',
      label: 'Role',
      type: 'multi-select',
      options: roles.map((r) => ({ label: r.name, value: r.slug })),
    },
    {
      key: 'mfa_enabled',
      label: 'MFA',
      type: 'select',
      options: [
        { label: 'Enabled', value: 'true' },
        { label: 'Disabled', value: 'false' },
      ],
    },
  ];
}
