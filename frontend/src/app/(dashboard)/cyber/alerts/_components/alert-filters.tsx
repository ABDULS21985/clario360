import type { FilterConfig } from '@/types/table';

export const ALERT_FILTERS: FilterConfig[] = [
  {
    key: 'severity',
    label: 'Severity',
    type: 'multi-select',
    options: [
      { label: 'Critical', value: 'critical' },
      { label: 'High', value: 'high' },
      { label: 'Medium', value: 'medium' },
      { label: 'Low', value: 'low' },
      { label: 'Info', value: 'info' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'multi-select',
    options: [
      { label: 'New', value: 'new' },
      { label: 'Acknowledged', value: 'acknowledged' },
      { label: 'Investigating', value: 'investigating' },
      { label: 'In Progress', value: 'in_progress' },
      { label: 'Resolved', value: 'resolved' },
      { label: 'Closed', value: 'closed' },
      { label: 'False Positive', value: 'false_positive' },
      { label: 'Escalated', value: 'escalated' },
    ],
  },
  {
    key: 'source',
    label: 'Source',
    type: 'select',
    options: [
      { label: 'SIEM', value: 'siem' },
      { label: 'EDR', value: 'edr' },
      { label: 'Network', value: 'network' },
      { label: 'Manual', value: 'manual' },
    ],
  },
];
