'use client';

import { BarChart } from '@/components/shared/charts/bar-chart';
import { LineChart } from '@/components/shared/charts/line-chart';
import { PieChart } from '@/components/shared/charts/pie-chart';
import { KpiCard } from '@/components/shared/kpi-card';
import type { DetectionRule, DetectionRulePerformance } from '@/types/cyber';

interface RulePerformanceProps {
  rule: DetectionRule;
  performance?: DetectionRulePerformance;
  loading?: boolean;
}

export function RulePerformance({ rule, performance, loading = false }: RulePerformanceProps) {
  const alertTrend = (performance?.alert_trend ?? []).map((point) => ({
    date: new Date(point.date).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
    alerts: point.count,
  }));

  const severityData = (performance?.severity_distribution ?? []).map((item) => ({
    name: item.name,
    value: item.count,
    color:
      item.name === 'critical'
        ? '#dc2626'
        : item.name === 'high'
        ? '#f97316'
        : item.name === 'medium'
        ? '#f59e0b'
        : '#2563eb',
  }));

  const topAssets = (performance?.top_assets ?? []).map((item) => ({
    asset: item.asset_name,
    alerts: item.alert_count,
  }));

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <KpiCard title="Triggers" value={rule.trigger_count.toLocaleString()} loading={loading} />
        <KpiCard title="Alerts 30d" value={performance?.alerts_last_30_days ?? 0} loading={loading} />
        <KpiCard title="True Positive Rate" value={`${((performance?.true_positive_rate ?? 0) * 100).toFixed(1)}%`} loading={loading} />
        <KpiCard title="False Positive Rate" value={`${((performance?.false_positive_rate ?? 0) * 100).toFixed(1)}%`} loading={loading} />
      </div>

      <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
        <LineChart
          data={alertTrend}
          xKey="date"
          yKeys={[{ key: 'alerts', label: 'Alerts', color: '#0f766e' }]}
          title="Alert Trend (90 days)"
          loading={loading}
          height={320}
        />
        <PieChart
          data={severityData}
          title="Alert Severity Distribution"
          loading={loading}
          centerLabel="Alerts"
          centerValue={String(performance?.alerts_last_90_days ?? 0)}
          height={320}
        />
      </div>

      <BarChart
        data={topAssets}
        xKey="asset"
        yKeys={[{ key: 'alerts', label: 'Alerts', color: '#1d4ed8' }]}
        layout="horizontal"
        title="Top Triggered Assets"
        loading={loading}
        height={320}
      />
    </div>
  );
}
