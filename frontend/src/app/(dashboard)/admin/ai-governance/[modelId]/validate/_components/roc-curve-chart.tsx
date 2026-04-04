'use client';

import {
  Area,
  AreaChart,
  CartesianGrid,
  Legend,
  Line,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { formatPercentage } from '@/lib/format';
import type { AIValidationResult } from '@/types/ai-governance';

interface ROCCurveChartProps {
  result: AIValidationResult;
}

export function ROCCurveChart({ result }: ROCCurveChartProps) {
  const data = result.roc_curve
    .slice()
    .sort((left, right) => left.fpr - right.fpr)
    .map((point) => ({
      ...point,
      baseline: point.fpr,
    }));

  return (
    <Card className="border-border/70">
      <CardHeader className="flex-row items-center justify-between">
        <CardTitle>ROC Curve</CardTitle>
        <div className="rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-emerald-800">
          AUC {formatPercentage(result.auc, 1)}
        </div>
      </CardHeader>
      <CardContent>
        <div className="h-[220px] sm:h-[320px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data} margin={{ top: 8, right: 12, left: 0, bottom: 8 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
              <XAxis
                dataKey="fpr"
                type="number"
                domain={[0, 1]}
                tickFormatter={(value) => formatPercentage(value, 0)}
                tick={{ fontSize: 12, fill: 'hsl(var(--muted-foreground))' }}
                axisLine={false}
                tickLine={false}
              />
              <YAxis
                type="number"
                domain={[0, 1]}
                tickFormatter={(value) => formatPercentage(value, 0)}
                tick={{ fontSize: 12, fill: 'hsl(var(--muted-foreground))' }}
                axisLine={false}
                tickLine={false}
              />
              <Tooltip
                formatter={(value: number, name: string) => [formatPercentage(value, 1), name]}
                labelFormatter={(value: number) => `FPR ${formatPercentage(value, 1)}`}
              />
              <Legend iconType="circle" iconSize={8} wrapperStyle={{ fontSize: 12 }} />
              <ReferenceLine segment={[{ x: 0, y: 0 }, { x: 1, y: 1 }]} stroke="#94a3b8" strokeDasharray="6 6" />
              <Area
                dataKey="tpr"
                type="monotone"
                name="ROC"
                stroke="#0f766e"
                fill="#34d399"
                fillOpacity={0.2}
                strokeWidth={2.5}
              />
              <Line
                dataKey="baseline"
                type="linear"
                name="Random"
                stroke="#64748b"
                strokeDasharray="5 5"
                dot={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}
