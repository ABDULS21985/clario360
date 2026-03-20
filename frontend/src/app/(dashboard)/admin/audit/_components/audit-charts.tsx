"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { AreaChart } from "@/components/shared/charts/area-chart";
import { BarChart } from "@/components/shared/charts/bar-chart";
import { PieChart } from "@/components/shared/charts/pie-chart";
import { formatDate } from "@/lib/format";
import type { AuditLogStats } from "@/types/audit";

interface AuditChartsProps {
  stats: AuditLogStats | undefined;
  loading: boolean;
  error?: string;
  onRetry?: () => void;
}

const CHART_COLORS = [
  "hsl(220, 70%, 55%)",
  "hsl(160, 60%, 45%)",
  "hsl(280, 65%, 55%)",
  "hsl(35, 85%, 55%)",
  "hsl(350, 65%, 55%)",
  "hsl(190, 70%, 45%)",
  "hsl(50, 75%, 50%)",
  "hsl(310, 55%, 50%)",
];

const SEVERITY_COLORS: Record<string, string> = {
  critical: "hsl(0, 72%, 51%)",
  high: "hsl(25, 95%, 53%)",
  warning: "hsl(45, 93%, 47%)",
  medium: "hsl(45, 93%, 47%)",
  low: "hsl(142, 71%, 45%)",
  info: "hsl(217, 91%, 60%)",
};

export function AuditCharts({ stats, loading, error, onRetry }: AuditChartsProps) {
  const [timeRange, setTimeRange] = useState<"24h" | "30d">("24h");

  const timeseriesData =
    timeRange === "24h"
      ? (stats?.by_hour ?? []).map((item) => ({
          timestamp: formatDate(item.timestamp, "HH:mm"),
          count: item.count,
        }))
      : (stats?.by_day ?? []).map((item) => ({
          timestamp: formatDate(item.timestamp, "MMM d"),
          count: item.count,
        }));

  const serviceData = (stats?.by_service ?? []).map((item) => ({
    service: item.key,
    count: item.count,
  }));

  const actionData = (stats?.by_action ?? []).map((item, i) => ({
    name: item.key,
    value: item.count,
    color: CHART_COLORS[i % CHART_COLORS.length],
  }));

  const severityData = (stats?.by_severity ?? []).map((item) => ({
    severity: item.key,
    count: item.count,
  }));

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <Card className="lg:col-span-2">
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-semibold">
              Events Over Time
            </CardTitle>
            <div className="flex gap-1">
              <Button
                variant={timeRange === "24h" ? "default" : "outline"}
                size="sm"
                onClick={() => setTimeRange("24h")}
              >
                24h
              </Button>
              <Button
                variant={timeRange === "30d" ? "default" : "outline"}
                size="sm"
                onClick={() => setTimeRange("30d")}
              >
                30d
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <AreaChart
            data={timeseriesData}
            xKey="timestamp"
            yKeys={[
              {
                key: "count",
                label: "Events",
                color: "hsl(220, 70%, 55%)",
              },
            ]}
            loading={loading}
            error={error}
            onRetry={onRetry}
            height={280}
            showLegend={false}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-semibold">
            Events by Service
          </CardTitle>
        </CardHeader>
        <CardContent>
          <BarChart
            data={serviceData}
            xKey="service"
            yKeys={[
              {
                key: "count",
                label: "Events",
                color: "hsl(220, 70%, 55%)",
              },
            ]}
            layout="horizontal"
            loading={loading}
            error={error}
            onRetry={onRetry}
            height={280}
            showLegend={false}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-semibold">
            Events by Action
          </CardTitle>
        </CardHeader>
        <CardContent>
          <PieChart
            data={actionData}
            loading={loading}
            error={error}
            onRetry={onRetry}
            height={280}
            innerRadius={50}
            outerRadius={90}
            centerValue={
              stats
                ? new Intl.NumberFormat("en-US").format(stats.total_events)
                : undefined
            }
            centerLabel="total"
          />
        </CardContent>
      </Card>

      <Card className="lg:col-span-2">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-semibold">
            Events by Severity
          </CardTitle>
        </CardHeader>
        <CardContent>
          <BarChart
            data={severityData}
            xKey="severity"
            yKeys={severityData.map((item) => ({
              key: "count",
              label: item.severity,
              color: SEVERITY_COLORS[item.severity] ?? "hsl(220, 70%, 55%)",
            })).slice(0, 1).concat([])}
            loading={loading}
            error={error}
            onRetry={onRetry}
            height={200}
            showLegend={false}
          />
        </CardContent>
      </Card>
    </div>
  );
}
