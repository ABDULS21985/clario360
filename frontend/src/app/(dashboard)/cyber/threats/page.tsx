'use client';

import { useState } from 'react';
import { Search, Shield } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { DataTable } from '@/components/shared/data-table/data-table';
import { useDataTable } from '@/hooks/use-data-table';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { PaginatedResponse } from '@/types/api';
import type { FetchParams, FilterConfig } from '@/types/table';
import type { Threat } from '@/types/cyber';

import { getThreatColumns } from './_components/threat-columns';
import { ThreatDetailPanel } from './_components/threat-detail-panel';
import { IndicatorCheckDialog } from './_components/indicator-check-dialog';

const THREAT_FILTERS: FilterConfig[] = [
  {
    key: 'severity',
    label: 'Severity',
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
      { label: 'Contained', value: 'contained' },
      { label: 'Eradicated', value: 'eradicated' },
      { label: 'Monitoring', value: 'monitoring' },
      { label: 'Closed', value: 'closed' },
    ],
  },
];

function fetchThreats(params: FetchParams): Promise<PaginatedResponse<Threat>> {
  return apiGet<PaginatedResponse<Threat>>(API_ENDPOINTS.CYBER_THREATS, params as unknown as Record<string, unknown>);
}

export default function CyberThreatsPage() {
  const [selectedThreat, setSelectedThreat] = useState<Threat | null>(null);
  const [indicatorCheckOpen, setIndicatorCheckOpen] = useState(false);

  const { tableProps } = useDataTable<Threat>({
    fetchFn: fetchThreats,
    queryKey: 'cyber-threats',
    defaultPageSize: 25,
    defaultSort: { column: 'last_seen', direction: 'desc' },
    wsTopics: ['threat.detected', 'threat.updated'],
  });

  const columns = getThreatColumns({ onViewDetail: setSelectedThreat });

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Threat Intelligence"
          description="Track active threats and hunt for malicious indicators"
          actions={
            <Button variant="outline" size="sm" onClick={() => setIndicatorCheckOpen(true)}>
              <Search className="mr-1.5 h-3.5 w-3.5" />
              Check Indicators
            </Button>
          }
        />

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className={selectedThreat ? 'lg:col-span-2' : 'lg:col-span-3'}>
            <DataTable
              columns={columns}
              filters={THREAT_FILTERS}
              searchPlaceholder="Search threats…"
              emptyState={{
                icon: Shield,
                title: 'No threats found',
                description: 'No active threats match the current filter.',
              }}
              getRowId={(row) => row.id}
              onRowClick={setSelectedThreat}
              {...tableProps}
            />
          </div>

          {selectedThreat && (
            <div className="lg:col-span-1">
              <ThreatDetailPanel
                threat={selectedThreat}
                onClose={() => setSelectedThreat(null)}
              />
            </div>
          )}
        </div>
      </div>

      <IndicatorCheckDialog
        open={indicatorCheckOpen}
        onOpenChange={setIndicatorCheckOpen}
      />
    </PermissionRedirect>
  );
}
