"use client";
import {
  BarChart as RechartsBarChart,
  Bar,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import { ChartContainer } from "./chart-container";
import { ChartTooltip } from "./chart-tooltip";

interface BarSeriesConfig {
  key: string;
  label: string;
  color: string;
}

interface BarChartProps {
  data: Array<Record<string, unknown>>;
  xKey: string;
  yKeys: BarSeriesConfig[];
  /** Per-bar colors applied to the first yKey series. Length must match data. */
  cellColors?: string[];
  layout?: "vertical" | "horizontal";
  stacked?: boolean;
  xFormatter?: (value: string | number) => string;
  yFormatter?: (value: number) => string;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  height?: number;
  showGrid?: boolean;
  showLegend?: boolean;
  barRadius?: number;
  title?: string;
  className?: string;
}

export function BarChart({
  data, xKey, yKeys, cellColors, layout = "vertical", stacked = false,
  xFormatter, yFormatter, loading = false, error, onRetry,
  height = 300, showGrid = true, showLegend = true, barRadius = 4, title, className,
}: BarChartProps) {
  return (
    <ChartContainer loading={loading} error={error} onRetry={onRetry} empty={data.length === 0} height={height} title={title} className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <RechartsBarChart data={data} layout={layout === "horizontal" ? "vertical" : "horizontal"} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
          {showGrid && <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />}
          <XAxis dataKey={layout === "vertical" ? xKey : undefined} type={layout === "horizontal" ? "number" : "category"} tickFormatter={xFormatter} tick={{ fontSize: 12, fill: "hsl(var(--muted-foreground))" }} axisLine={false} tickLine={false} />
          <YAxis dataKey={layout === "horizontal" ? xKey : undefined} type={layout === "horizontal" ? "category" : "number"} tickFormatter={yFormatter} tick={{ fontSize: 12, fill: "hsl(var(--muted-foreground))" }} axisLine={false} tickLine={false} width={layout === "horizontal" ? 100 : 40} />
          <Tooltip content={<ChartTooltip valueFormatter={yFormatter} />} cursor={{ fill: "hsl(var(--muted))" }} />
          {showLegend && <Legend iconType="circle" iconSize={8} wrapperStyle={{ fontSize: 12 }} />}
          {yKeys.map((series, seriesIdx) => (
            <Bar key={series.key} dataKey={series.key} name={series.label} fill={series.color} stackId={stacked ? "stack" : undefined} radius={[barRadius, barRadius, 0, 0]}>
              {seriesIdx === 0 && cellColors && cellColors.length === data.length &&
                data.map((_, i) => <Cell key={i} fill={cellColors[i]} />)}
            </Bar>
          ))}
        </RechartsBarChart>
      </ResponsiveContainer>
    </ChartContainer>
  );
}
