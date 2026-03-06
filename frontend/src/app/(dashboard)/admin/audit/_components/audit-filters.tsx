import type { FilterConfig } from '@/types/table';

export const auditFilters: FilterConfig[] = [
  {
    key: 'service',
    label: 'Service',
    type: 'multi-select',
    options: [
      { label: 'IAM Service', value: 'iam-service' },
      { label: 'Audit Service', value: 'audit-service' },
      { label: 'Workflow Engine', value: 'workflow-engine' },
      { label: 'Notification Service', value: 'notification-service' },
      { label: 'File Service', value: 'file-service' },
      { label: 'Cyber Service', value: 'cyber-service' },
      { label: 'Data Service', value: 'data-service' },
      { label: 'Acta Service', value: 'acta-service' },
      { label: 'Lex Service', value: 'lex-service' },
      { label: 'Visus Service', value: 'visus-service' },
    ],
  },
  {
    key: 'severity',
    label: 'Severity',
    type: 'multi-select',
    options: [
      { label: 'Info', value: 'info' },
      { label: 'Warning', value: 'warning' },
      { label: 'High', value: 'high' },
      { label: 'Critical', value: 'critical' },
    ],
  },
];
