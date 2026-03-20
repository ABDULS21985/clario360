'use client';

import { PieChart } from '@/components/shared/charts/pie-chart';

interface ClassificationChartProps {
  data: Record<string, number>;
}

const CLASSIFICATION_COLORS: Record<string, string> = {
  public: '#22c55e',
  internal: '#3b82f6',
  confidential: '#f59e0b',
  restricted: '#ef4444',
  top_secret: '#7c3aed',
};

export function ClassificationChart({ data }: ClassificationChartProps) {
  if (!data || typeof data !== 'object') {
    return (
      <div>
        <h3 className="mb-3 text-sm font-semibold">Classification Breakdown</h3>
        <p className="text-xs text-muted-foreground">No classification data available.</p>
      </div>
    );
  }
  const chartData = Object.entries(data).map(([key, count]) => ({
    name: key.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase()),
    value: count,
    color: CLASSIFICATION_COLORS[key] ?? '#94a3b8',
  }));

  return (
    <div>
      <h3 className="mb-3 text-sm font-semibold">Classification Breakdown</h3>
      <PieChart
        data={chartData}
        height={220}
        showLegend={false}
      />
      <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
        {chartData.map(({ name, value, color }) => (
          <div key={name} className="flex items-center gap-2 text-xs">
            <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: color }} />
            <span className="text-muted-foreground">{name}</span>
            <span className="ml-auto font-semibold">{value}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
