"use client";
import {
  LineChart as RechartsLineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from "recharts";
import { ChartContainer } from "./chart-container";
import { ChartTooltip } from "./chart-tooltip";
import { cn } from "@/lib/utils";

interface LineSeriesConfig {
  key: string;
  label: string;
  color: string;
  dashed?: boolean;
}

interface LineChartProps {
  data: Array<Record<string, unknown>>;
  xKey: string;
  yKeys: LineSeriesConfig[];
  xFormatter?: (value: string | number) => string;
  yFormatter?: (value: number) => string;
  tooltipFormatter?: (value: number, name: string) => string;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  height?: number;
  showGrid?: boolean;
  showLegend?: boolean;
  animate?: boolean;
  title?: string;
  className?: string;
}

export function LineChart({
  data,
  xKey,
  yKeys,
  xFormatter,
  yFormatter,
  tooltipFormatter,
  loading = false,
  error,
  onRetry,
  height = 300,
  showGrid = true,
  showLegend = true,
  animate = true,
  title,
  className,
}: LineChartProps) {
  return (
    <ChartContainer loading={loading} error={error} onRetry={onRetry} empty={data.length === 0} height={height} title={title} className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <RechartsLineChart data={data} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
          {showGrid && <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />}
          <XAxis dataKey={xKey} tickFormatter={xFormatter} tick={{ fontSize: 12, fill: "hsl(var(--muted-foreground))" }} axisLine={false} tickLine={false} />
          <YAxis tickFormatter={yFormatter} tick={{ fontSize: 12, fill: "hsl(var(--muted-foreground))" }} axisLine={false} tickLine={false} />
          <Tooltip
            content={<ChartTooltip labelFormatter={xFormatter} valueFormatter={tooltipFormatter} />}
            cursor={{ stroke: "hsl(var(--border))", strokeWidth: 1 }}
          />
          {showLegend && <Legend iconType="circle" iconSize={8} wrapperStyle={{ fontSize: 12 }} />}
          {yKeys.map((series) => (
            <Line
              key={series.key}
              type="monotone"
              dataKey={series.key}
              name={series.label}
              stroke={series.color}
              strokeWidth={2}
              dot={false}
              strokeDasharray={series.dashed ? "5 5" : undefined}
              isAnimationActive={animate}
              animationDuration={300}
            />
          ))}
        </RechartsLineChart>
      </ResponsiveContainer>
    </ChartContainer>
  );
}
