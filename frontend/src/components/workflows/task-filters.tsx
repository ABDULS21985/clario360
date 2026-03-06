import { apiGet } from '@/lib/api';
import type { FilterConfig } from '@/types/table';
import type { Role } from '@/types/models';

export const taskFilters: FilterConfig[] = [
  {
    key: 'priority',
    label: 'Priority',
    type: 'multi-select',
    options: [
      { label: 'Critical', value: '2' },
      { label: 'High', value: '1' },
      { label: 'Normal', value: '0' },
    ],
  },
  {
    key: 'assignee_role',
    label: 'Role',
    type: 'select',
    options: [],
  },
  {
    key: 'sla_breached',
    label: 'SLA',
    type: 'select',
    options: [
      { label: 'Overdue', value: 'true' },
      { label: 'On Time', value: 'false' },
    ],
  },
];

export async function fetchRoleFilterOptions(): Promise<{ label: string; value: string }[]> {
  const roles = await apiGet<Role[]>('/api/v1/roles');
  return roles.map((r) => ({ label: r.name, value: r.slug }));
}
