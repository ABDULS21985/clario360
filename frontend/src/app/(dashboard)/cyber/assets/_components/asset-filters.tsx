import type { FilterConfig } from '@/types/table';

export const ASSET_FILTERS: FilterConfig[] = [
  {
    key: 'type',
    label: 'Type',
    type: 'multi-select',
    options: [
      { label: 'Server', value: 'server' },
      { label: 'Endpoint', value: 'endpoint' },
      { label: 'Cloud Resource', value: 'cloud_resource' },
      { label: 'Network Device', value: 'network_device' },
      { label: 'IoT Device', value: 'iot_device' },
      { label: 'Application', value: 'application' },
      { label: 'Database', value: 'database' },
      { label: 'Container', value: 'container' },
    ],
  },
  {
    key: 'criticality',
    label: 'Criticality',
    type: 'multi-select',
    options: [
      { label: 'Critical', value: 'critical' },
      { label: 'High', value: 'high' },
      { label: 'Medium', value: 'medium' },
      { label: 'Low', value: 'low' },
    ],
  },
  {
    key: 'status',
    label: 'Status',
    type: 'multi-select',
    options: [
      { label: 'Active', value: 'active' },
      { label: 'Inactive', value: 'inactive' },
      { label: 'Decommissioned', value: 'decommissioned' },
      { label: 'Unknown', value: 'unknown' },
    ],
  },
  {
    key: 'discovery_source',
    label: 'Discovery Source',
    type: 'multi-select',
    options: [
      { label: 'Manual', value: 'manual' },
      { label: 'Network Scan', value: 'network_scan' },
      { label: 'Cloud Scan', value: 'cloud_scan' },
      { label: 'Agent', value: 'agent' },
      { label: 'Import', value: 'import' },
    ],
  },
];
