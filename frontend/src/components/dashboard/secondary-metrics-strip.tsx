'use client';

import { motion } from 'framer-motion';
import {
  Clock,
  Timer,
  Shield,
  Activity,
  Users,
  FileCheck,
  type LucideIcon,
} from 'lucide-react';
import { useAuth } from '@/hooks/use-auth';
import { useRealtimeData } from '@/hooks/use-realtime-data';

interface DashboardMetrics {
  mttr_minutes?: number;
  mtta_minutes?: number;
  sla_compliance_pct?: number;
  active_incidents?: number;
  active_users_today?: number;
  pending_reviews?: number;
}

const STATUS_COLORS = {
  green: '#10B981',
  amber: '#F59E0B',
  red: '#EF4444',
} as const;

type StatusColor = (typeof STATUS_COLORS)[keyof typeof STATUS_COLORS];

function getStatusColor(
  value: number | undefined | null,
  evaluate: (v: number) => StatusColor,
): StatusColor {
  if (value == null) return STATUS_COLORS.green;
  return evaluate(value);
}

interface MetricConfig {
  key: string;
  label: string;
  icon: LucideIcon;
  value: number | undefined;
  suffix: string;
  colorFn: (v: number) => StatusColor;
  permission?: string;
}

function formatMetricValue(value: number, suffix: string): string {
  if (suffix === 'min') {
    if (value >= 1440) {
      return `${(value / 1440).toFixed(1)}d`;
    }
    if (value >= 60) {
      return `${(value / 60).toFixed(1)}h`;
    }
    return `${Math.round(value)}min`;
  }
  if (suffix === '%') {
    return `${Math.round(value)}%`;
  }
  return value.toLocaleString();
}

export function SecondaryMetricsStrip() {
  const { hasPermission } = useAuth();
  const hasCyber = hasPermission('cyber:read');

  const { data: envelope, isLoading, error } = useRealtimeData<{ data: DashboardMetrics }>(
    '/api/v1/cyber/dashboard/metrics',
    {
      wsTopics: ['dashboard.metrics.updated'],
      enabled: hasCyber,
    },
  );
  const data = envelope?.data;

  // Hide entire strip if the endpoint doesn't exist or returned an error
  if (error && !isLoading) return null;
  // Also hide if we got a response but all values are undefined (no data)
  if (!isLoading && data && Object.values(data).every((v) => v === undefined || v === null)) return null;

  const metrics: MetricConfig[] = [
    {
      key: 'mttr',
      label: 'MTTR',
      icon: Clock,
      value: data?.mttr_minutes,
      suffix: 'min',
      colorFn: (v: number) =>
        v <= 60 ? STATUS_COLORS.green : v <= 120 ? STATUS_COLORS.amber : STATUS_COLORS.red,
      permission: 'cyber:read',
    },
    {
      key: 'mtta',
      label: 'MTTA',
      icon: Timer,
      value: data?.mtta_minutes,
      suffix: 'min',
      colorFn: (v: number) =>
        v <= 15 ? STATUS_COLORS.green : v <= 30 ? STATUS_COLORS.amber : STATUS_COLORS.red,
      permission: 'cyber:read',
    },
    {
      key: 'sla',
      label: 'SLA Compliance',
      icon: Shield,
      value: data?.sla_compliance_pct,
      suffix: '%',
      colorFn: (v: number) =>
        v >= 95 ? STATUS_COLORS.green : v >= 85 ? STATUS_COLORS.amber : STATUS_COLORS.red,
    },
    {
      key: 'incidents',
      label: 'Active Incidents',
      icon: Activity,
      value: data?.active_incidents,
      suffix: '',
      colorFn: (v: number) =>
        v === 0 ? STATUS_COLORS.green : v <= 3 ? STATUS_COLORS.amber : STATUS_COLORS.red,
      permission: 'cyber:read',
    },
    {
      key: 'users',
      label: 'Active Users',
      icon: Users,
      value: data?.active_users_today,
      suffix: '',
      colorFn: (_v: number) => STATUS_COLORS.green,
    },
    {
      key: 'reviews',
      label: 'Pending Reviews',
      icon: FileCheck,
      value: data?.pending_reviews,
      suffix: '',
      colorFn: (v: number) =>
        v === 0 ? STATUS_COLORS.green : v <= 5 ? STATUS_COLORS.amber : STATUS_COLORS.red,
    },
  ].filter((m) => {
    if (m.permission === 'cyber:read' && !hasCyber) return false;
    return true;
  });

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, delay: 0.4 }}
      className="overflow-x-auto rounded-2xl border border-border/60"
      style={{
        background: 'rgba(255, 255, 255, 0.55)',
        backdropFilter: 'blur(16px)',
        WebkitBackdropFilter: 'blur(16px)',
      }}
    >
      <div className="flex flex-nowrap">
        {metrics.map((metric, index) => {
          const Icon = metric.icon;
          const color = getStatusColor(metric.value, metric.colorFn);
          const isLast = index === metrics.length - 1;

          return (
            <div
              key={metric.key}
              className={`flex flex-1 flex-col items-center justify-center gap-1 px-4 py-3 min-w-[100px] ${!isLast ? 'border-r border-border/40' : ''}`}
            >
              <Icon className="h-3 w-3 text-muted-foreground" />

              {isLoading ? (
                <div className="h-4 w-8 animate-pulse rounded bg-muted/50" />
              ) : (
                <span
                  className="text-sm font-semibold tabular-nums"
                  style={{ color }}
                >
                  {metric.value != null
                    ? formatMetricValue(metric.value, metric.suffix)
                    : '—'}
                </span>
              )}

              <span className="whitespace-nowrap text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                {metric.label}
              </span>
            </div>
          );
        })}
      </div>
    </motion.div>
  );
}
