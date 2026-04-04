'use client';

import { BarChart } from '@/components/shared/charts/bar-chart';
import { CTI_SEVERITY_COLORS, type CTISectorThreatSummary } from '@/types/cti';

interface SectorThreatChartProps {
  sectors: CTISectorThreatSummary[];
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
}

export function SectorThreatChart({
  sectors,
  loading = false,
  error,
  onRetry,
}: SectorThreatChartProps) {
  const data = [...sectors]
    .sort((left, right) => right.total_count - left.total_count)
    .slice(0, 10)
    .map((sector) => ({
      name: sector.sector_label,
      critical: sector.severity_critical_count,
      high: sector.severity_high_count,
      medium: sector.severity_medium_count,
      low: sector.severity_low_count,
    }));

  return (
    <BarChart
      title="Threat Pressure By Sector"
      data={data}
      xKey="name"
      layout="horizontal"
      stacked
      loading={loading}
      error={error}
      onRetry={onRetry}
      height={360}
      yKeys={[
        { key: 'critical', label: 'Critical', color: CTI_SEVERITY_COLORS.critical },
        { key: 'high', label: 'High', color: CTI_SEVERITY_COLORS.high },
        { key: 'medium', label: 'Medium', color: CTI_SEVERITY_COLORS.medium },
        { key: 'low', label: 'Low', color: CTI_SEVERITY_COLORS.low },
      ]}
    />
  );
}