'use client';

import { useEffect, useMemo } from 'react';
import { fetchThreatEvents, flattenThreatEventFetchParams } from '@/lib/cti-api';
import { useDataTable } from '@/hooks/use-data-table';
import { useCTIStore } from '@/stores/cti-store';
import type { CTIThreatEvent, CTIThreatEventFilters } from '@/types/cti';
import type { FilterConfig, FetchParams } from '@/types/table';
import type { PaginatedResponse } from '@/types/api';

function fetchEvents(params: FetchParams): Promise<PaginatedResponse<CTIThreatEvent>> {
  return fetchThreatEvents(flattenThreatEventFetchParams(params));
}

export function useCTIThreatEvents(initialFilters?: Partial<CTIThreatEventFilters>) {
  const { categories, sectors, loadReferenceData } = useCTIStore();

  useEffect(() => {
    void loadReferenceData();
  }, [loadReferenceData]);

  const table = useDataTable<CTIThreatEvent>({
    fetchFn: fetchEvents,
    queryKey: 'cti-threat-events',
    defaultPageSize: initialFilters?.per_page ?? 25,
    defaultSort: {
      column: initialFilters?.sort ?? initialFilters?.sort_by ?? 'first_seen_at',
      direction: initialFilters?.order ?? initialFilters?.sort_dir ?? 'desc',
    },
    wsTopics: [
      'com.clario360.cyber.cti.threat-event.created',
      'com.clario360.cyber.cti.threat-event.updated',
      'cyber.cti.threat-event.created',
      'cyber.cti.threat-event.updated',
    ],
  });

  const filterConfig = useMemo<FilterConfig[]>(
    () => [
      {
        key: 'severity',
        label: 'Severity',
        type: 'multi-select',
        options: [
          { label: 'Critical', value: 'critical' },
          { label: 'High', value: 'high' },
          { label: 'Medium', value: 'medium' },
          { label: 'Low', value: 'low' },
          { label: 'Informational', value: 'informational' },
        ],
      },
      {
        key: 'category',
        label: 'Category',
        type: 'multi-select',
        options: categories.map((category) => ({
          label: category.label,
          value: category.code,
        })),
      },
      {
        key: 'event_type',
        label: 'Event Type',
        type: 'multi-select',
        options: [
          { label: 'Indicator Sighting', value: 'indicator_sighting' },
          { label: 'Attack Attempt', value: 'attack_attempt' },
          { label: 'Vulnerability Exploit', value: 'vulnerability_exploit' },
          { label: 'Malware Detection', value: 'malware_detection' },
          { label: 'Anomaly', value: 'anomaly' },
          { label: 'Policy Violation', value: 'policy_violation' },
        ],
      },
      {
        key: 'target_sector',
        label: 'Target Sector',
        type: 'multi-select',
        options: sectors.map((sector) => ({
          label: sector.label,
          value: sector.code,
        })),
      },
      {
        key: 'origin_country',
        label: 'Origin Country',
        type: 'text',
        placeholder: 'ISO country code',
      },
      {
        key: 'ioc_type',
        label: 'IOC Type',
        type: 'multi-select',
        options: [
          { label: 'IP', value: 'ip' },
          { label: 'Domain', value: 'domain' },
          { label: 'URL', value: 'url' },
          { label: 'SHA256', value: 'hash_sha256' },
          { label: 'MD5', value: 'hash_md5' },
          { label: 'Email', value: 'email' },
          { label: 'CVE', value: 'cve' },
        ],
      },
      {
        key: 'first_seen',
        label: 'Date Range',
        type: 'date-range',
      },
      {
        key: 'confidence',
        label: 'Confidence',
        type: 'range',
        min: 0,
        max: 100,
        step: 5,
        valueSuffix: '%',
      },
      {
        key: 'is_false_positive',
        label: 'False Positive',
        type: 'select',
        options: [
          { label: 'Yes', value: 'true' },
          { label: 'No', value: 'false' },
        ],
      },
    ],
    [categories, sectors],
  );

  return {
    ...table,
    filters: filterConfig,
  };
}
