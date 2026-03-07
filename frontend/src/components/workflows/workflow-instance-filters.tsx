import type { FilterConfig } from '@/types/table';

export const workflowInstanceFilters: FilterConfig[] = [
  {
    key: 'status',
    label: 'Status',
    type: 'multi-select',
    options: [
      { label: 'Running', value: 'running' },
      { label: 'Completed', value: 'completed' },
      { label: 'Failed', value: 'failed' },
      { label: 'Cancelled', value: 'cancelled' },
      { label: 'Suspended', value: 'suspended' },
    ],
  },
  {
    key: 'started_at',
    label: 'Started',
    type: 'date-range',
  },
];
