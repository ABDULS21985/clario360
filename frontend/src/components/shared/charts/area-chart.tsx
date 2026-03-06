"use client";
import {
  AreaChart as RechartsAreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import { ChartContainer } from "./chart-container";
import { ChartTooltip } from "./chart-tooltip";

interface AreaSeriesConfig {
  key: string;
  label: string;
  color: string;
}

interface AreaChartProps {
  data: Array<Record<string, unknown>>;
  xKey: string;
  yKeys: AreaSeriesConfig[];
  stacked?: boolean;
  xFormatter?: (value: string | number) => string;
  yFormatter?: (value: number) => string;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  height?: number;
  showGrid?: boolean;
  showLegend?: boolean;
  title?: string;
  className?: string;
}

export function AreaChart({
  data, xKey, yKeys, stacked = false,
  xFormatter, yFormatter, loading = false, error, onRetry,
  height = 300, showGrid = true, showLegend = true, title, className,
}: AreaChartProps) {
  return (
    <ChartContainer loading={loading} error={error} onRetry={onRetry} empty={data.length === 0} height={height} title={title} className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <RechartsAreaChart data={data} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
          {showGrid && <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />}
          <XAxis dataKey={xKey} tickFormatter={xFormatter} tick={{ fontSize: 12, fill: "hsl(var(--muted-foreground))" }} axisLine={false} tickLine={false} />
          <YAxis tickFormatter={yFormatter} tick={{ fontSize: 12, fill: "hsl(var(--muted-foreground))" }} axisLine={false} tickLine={false} />
          <Tooltip content={<ChartTooltip labelFormatter={xFormatter} valueFormatter={yFormatter} />} />
          {showLegend && <Legend iconType="circle" iconSize={8} wrapperStyle={{ fontSize: 12 }} />}
          {yKeys.map((series) => (
            <Area
              key={series.key}
              type="monotone"
              dataKey={series.key}
              name={series.label}
              stroke={series.color}
              fill={series.color}
              fillOpacity={0.15}
              stackId={stacked ? "stack" : undefined}
              strokeWidth={2}
            />
          ))}
        </RechartsAreaChart>
      </ResponsiveContainer>
    </ChartContainer>
  );
}
