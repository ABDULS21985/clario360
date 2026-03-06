"use client";
import { useState } from "react";
import {
  PieChart as RechartsPieChart,
  Pie,
  Cell,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { ChartContainer } from "./chart-container";

interface PieDataItem {
  name: string;
  value: number;
  color: string;
}

interface PieChartProps {
  data: PieDataItem[];
  innerRadius?: number;
  outerRadius?: number;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  height?: number;
  showLegend?: boolean;
  centerLabel?: string;
  centerValue?: string;
  title?: string;
  className?: string;
}

export function PieChart({
  data, innerRadius = 60, outerRadius = 100,
  loading = false, error, onRetry, height = 300,
  showLegend = true, centerLabel, centerValue, title, className,
}: PieChartProps) {
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const [hiddenKeys, setHiddenKeys] = useState<Set<string>>(new Set());

  const visibleData = data.filter((d) => !hiddenKeys.has(d.name));
  const total = visibleData.reduce((sum, d) => sum + d.value, 0);

  const toggleKey = (name: string) => {
    setHiddenKeys((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  return (
    <ChartContainer loading={loading} error={error} onRetry={onRetry} empty={data.length === 0} height={height} title={title} className={className}>
      <div className="flex flex-col sm:flex-row items-center gap-4 h-full">
        <ResponsiveContainer width="100%" height={height} className="shrink-0" style={{ maxWidth: height }}>
          <RechartsPieChart>
            <Pie
              data={visibleData}
              cx="50%"
              cy="50%"
              innerRadius={innerRadius}
              outerRadius={outerRadius}
              paddingAngle={2}
              dataKey="value"
              onMouseEnter={(_, i) => setActiveIndex(i)}
              onMouseLeave={() => setActiveIndex(null)}
            >
              {visibleData.map((entry, idx) => (
                <Cell
                  key={`cell-${idx}`}
                  fill={entry.color}
                  opacity={activeIndex === null || activeIndex === idx ? 1 : 0.6}
                  strokeWidth={0}
                />
              ))}
            </Pie>
            {(centerLabel || centerValue) && innerRadius > 0 && (
              <text x="50%" y="50%" textAnchor="middle" dominantBaseline="middle">
                {centerValue && <tspan x="50%" dy="-0.2em" className="text-xl font-bold fill-foreground" style={{ fontSize: "1.25rem", fontWeight: 700 }}>{centerValue}</tspan>}
                {centerLabel && <tspan x="50%" dy="1.4em" style={{ fontSize: "0.75rem", fill: "hsl(var(--muted-foreground))" }}>{centerLabel}</tspan>}
              </text>
            )}
            <Tooltip
              content={({ active, payload }) => {
                if (!active || !payload?.[0]) return null;
                const item = payload[0].payload as PieDataItem;
                const pct = total > 0 ? ((item.value / total) * 100).toFixed(1) : "0";
                return (
                  <div className="rounded-lg border border-border bg-background shadow-md p-3 text-sm">
                    <div className="flex items-center gap-2">
                      <div className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: item.color }} />
                      <span className="font-medium">{item.name}</span>
                    </div>
                    <p className="mt-1 text-muted-foreground">{item.value.toLocaleString()} ({pct}%)</p>
                  </div>
                );
              }}
            />
          </RechartsPieChart>
        </ResponsiveContainer>

        {showLegend && (
          <div className="flex flex-col gap-2 text-sm min-w-0">
            {data.map((item) => {
              const pct = total > 0 ? ((item.value / total) * 100).toFixed(1) : "0";
              const isHidden = hiddenKeys.has(item.name);
              return (
                <button
                  key={item.name}
                  className="flex items-center gap-2 text-left hover:opacity-80 transition-opacity"
                  onClick={() => toggleKey(item.name)}
                  aria-pressed={!isHidden}
                >
                  <div className={`h-2.5 w-2.5 rounded-full shrink-0 transition-opacity ${isHidden ? "opacity-30" : ""}`} style={{ backgroundColor: item.color }} />
                  <span className={`truncate ${isHidden ? "line-through text-muted-foreground" : ""}`}>{item.name}</span>
                  <span className="ml-auto text-muted-foreground shrink-0">{pct}%</span>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </ChartContainer>
  );
}
