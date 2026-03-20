"use client";

import { useAuditStats } from "@/hooks/use-audit";
import { AuditStatsCards } from "./audit-stats-cards";
import { AuditCharts } from "./audit-charts";
import { AuditTopTables } from "./audit-top-tables";
import type { AuditStatsParams } from "@/types/audit";

interface AuditDashboardProps {
  params?: AuditStatsParams;
}

export function AuditDashboard({ params }: AuditDashboardProps) {
  const { data: stats, isLoading, error, refetch } = useAuditStats(params);

  const errorMessage = error
    ? error instanceof Error
      ? error.message
      : "Failed to load audit statistics"
    : undefined;

  return (
    <div className="space-y-6">
      <AuditStatsCards stats={stats} loading={isLoading} />
      <AuditCharts
        stats={stats}
        loading={isLoading}
        error={errorMessage}
        onRetry={() => refetch()}
      />
      <AuditTopTables stats={stats} loading={isLoading} />
    </div>
  );
}
