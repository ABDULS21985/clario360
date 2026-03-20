'use client';

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { AreaChart } from '@/components/shared/charts/area-chart';

interface BurndownDataPoint {
  date: string;
  open: number;
  closed: number;
}

interface RemediationBurndownChartProps {
  data: BurndownDataPoint[];
}

function formatDate(value: string | number): string {
  const d = new Date(String(value));
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

export function RemediationBurndownChart({ data }: RemediationBurndownChartProps) {
  if (!data || data.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Remediation Burndown (30 Days)</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-12 text-center">
            <p className="text-sm text-muted-foreground">No burndown data available yet</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Remediation Burndown (30 Days)</CardTitle>
      </CardHeader>
      <CardContent>
        <AreaChart
          data={data as unknown as Array<Record<string, unknown>>}
          xKey="date"
          yKeys={[
            { key: 'open', label: 'Open', color: '#f97316' },
            { key: 'closed', label: 'Closed', color: '#22c55e' },
          ]}
          height={300}
          showLegend={true}
          xFormatter={formatDate}
        />
      </CardContent>
    </Card>
  );
}
