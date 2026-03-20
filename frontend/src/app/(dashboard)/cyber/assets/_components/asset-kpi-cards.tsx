'use client';

import { useRouter } from 'next/navigation';
import { Server, ShieldAlert, Bug, Compass } from 'lucide-react';
import { KpiCard } from '@/components/shared/kpi-card';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { API_ENDPOINTS } from '@/lib/constants';
import type { AssetStats } from '@/types/cyber';

export function AssetKpiCards() {
  const router = useRouter();
  const { data: envelope, isLoading } = useRealtimeData<{ data: AssetStats }>(
    API_ENDPOINTS.CYBER_ASSETS_STATS,
    { pollInterval: 120000 },
  );
  const stats = envelope?.data;

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <div className="cursor-pointer" onClick={() => router.push('/cyber/assets')}>
        <KpiCard
          title="Total Assets"
          value={stats?.total ?? 0}
          icon={Server}
          loading={isLoading}
        />
      </div>
      <div className="cursor-pointer" onClick={() => router.push('/cyber/assets?criticality=critical')}>
        <KpiCard
          title="Critical Assets"
          value={stats?.by_criticality?.['critical'] ?? 0}
          icon={ShieldAlert}
          iconColor="text-red-600"
          loading={isLoading}
        />
      </div>
      <div className="cursor-pointer" onClick={() => router.push('/cyber/assets?has_vulnerabilities=true')}>
        <KpiCard
          title="With Open Vulns"
          value={stats?.assets_with_vulns ?? 0}
          icon={Bug}
          iconColor="text-orange-500"
          loading={isLoading}
        />
      </div>
      <div className="cursor-pointer" onClick={() => router.push('/cyber/assets?discovery_source=network_scan')}>
        <KpiCard
          title="Discovered This Week"
          value={stats?.assets_discovered_this_week ?? 0}
          icon={Compass}
          loading={isLoading}
        />
      </div>
    </div>
  );
}
