import type { FetchParams, FilterConfig } from '@/types/table';

export function flattenAssetFetchParams(params: FetchParams): Record<string, unknown> {
  const flat: Record<string, unknown> = {
    page: params.page,
    per_page: params.per_page,
    sort: params.sort,
    order: params.order,
    search: params.search,
  };

  for (const [key, value] of Object.entries(params.filters ?? {})) {
    if (!value || (typeof value === 'string' && value.length === 0)) {
      continue;
    }
    flat[key] = value;
  }

  return flat;
}

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
  {
    key: 'has_vulnerabilities',
    label: 'Has Vulnerabilities',
    type: 'select',
    options: [
      { label: 'Yes', value: 'true' },
      { label: 'No', value: 'false' },
    ],
  },
  {
    key: 'owner',
    label: 'Owner',
    type: 'text',
    placeholder: 'Filter by owner...',
  },
  {
    key: 'department',
    label: 'Department',
    type: 'text',
    placeholder: 'Filter by department...',
  },
  {
    key: 'tag',
    label: 'Tag',
    type: 'text',
    placeholder: 'Filter by tag...',
  },
  {
    key: 'discovered_after',
    label: 'Discovered After',
    type: 'date-range',
  },
];
