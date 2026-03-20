'use client';

import { useMemo } from 'react';
import {
  Clock,
  LogIn,
  Upload,
  AlertTriangle,
  CheckSquare,
  Settings,
  FileText,
  Activity,
  Inbox,
} from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { useQuery } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { timeAgo } from '@/lib/utils';
import { subDays, formatISO } from 'date-fns';
import { useAuth } from '@/hooks/use-auth';
import type { PaginatedResponse } from '@/types/api';
import type { AuditLog } from '@/types/models';

/* Severity color mapping based on action type */
const SEVERITY_RED = '#EF4444';
const SEVERITY_AMBER = '#F59E0B';
const SEVERITY_GREEN = '#22C55E';
const SEVERITY_BLUE = '#3B82F6';

function getActionSeverityColor(action: string): string {
  if (action.includes('alert') || action.includes('escalat') || action.includes('breach')) return SEVERITY_RED;
  if (action.includes('fail') || action.includes('error') || action.includes('reject')) return SEVERITY_RED;
  if (action.includes('task') || action.includes('workflow') || action.includes('create')) return SEVERITY_AMBER;
  if (action.includes('update') || action.includes('settings') || action.includes('login')) return SEVERITY_BLUE;
  if (action.includes('complet') || action.includes('resolv') || action.includes('approv')) return SEVERITY_GREEN;
  return SEVERITY_BLUE;
}

function getActionIcon(action: string) {
  if (action.includes('login')) return LogIn;
  if (action.includes('upload') || action.includes('file')) return Upload;
  if (action.includes('alert')) return AlertTriangle;
  if (action.includes('task') || action.includes('workflow')) return CheckSquare;
  if (action.includes('settings') || action.includes('update')) return Settings;
  if (action.includes('document') || action.includes('contract')) return FileText;
  return Clock;
}

function formatAction(log: AuditLog): string {
  const action = log.action.replace(/_/g, ' ').replace(/\./g, ' ');
  if (log.resource_id) {
    return `${action}: ${log.resource_type} ${log.resource_id.slice(0, 8)}`;
  }
  return action;
}

/* Timeline dot with severity glow */
function TimelineDot({ action }: { action: string }) {
  const color = getActionSeverityColor(action);
  const isRed = color === SEVERITY_RED;

  return (
    <div className="relative flex h-3 w-3 flex-shrink-0 items-center justify-center">
      <div
        className="absolute inset-0 rounded-full"
        style={{ boxShadow: `0 0 8px ${color}66` }}
      />
      <div
        className="relative z-[1] h-3 w-3 rounded-full"
        style={{ backgroundColor: color }}
      />
    </div>
  );
}

/* Live pulsing dot */
function LiveDot() {
  return (
    <span className="relative flex h-2.5 w-2.5">
      <span
        className="absolute inline-flex h-full w-full rounded-full opacity-75"
        style={{
          backgroundColor: '#22C55E',
          animation: 'live-dot-ping 1.5s cubic-bezier(0, 0, 0.2, 1) infinite',
        }}
      />
      <span
        className="relative inline-flex h-2.5 w-2.5 rounded-full"
        style={{ backgroundColor: '#22C55E' }}
      />
    </span>
  );
}

/* Skeleton loading item */
function SkeletonItem({ index }: { index: number }) {
  return (
    <div
      className="flex items-start gap-3 px-4 py-3"
      style={{ animationDelay: `${index * 80}ms` }}
    >
      <div className="mt-1 flex items-center gap-2.5">
        <div className="h-3 w-3 rounded-full bg-border skeleton-shimmer" />
        <div className="h-7 w-7 rounded-full bg-border skeleton-shimmer" />
      </div>
      <div className="flex-1 space-y-2">
        <div
          className="h-3.5 rounded skeleton-shimmer bg-border"
          style={{ width: `${65 + (index % 3) * 10}%` }}
        />
        <div
          className="h-2.5 w-[30%] rounded skeleton-shimmer bg-border"
        />
      </div>
    </div>
  );
}

export function ActivityTimeline() {
  const { user } = useAuth();
  const sevenDaysAgo = useMemo(() => formatISO(subDays(new Date(), 7)), []);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['dashboard', 'activity'],
    queryFn: () =>
      apiGet<PaginatedResponse<AuditLog>>(API_ENDPOINTS.AUDIT_LOGS, {
        user_id: user?.id,
        per_page: 20,
        date_from: sevenDaysAgo,
      }),
    enabled: !!user?.id,
    refetchInterval: 120000,
  });

  const logs = data?.data ?? [];
  const total = data?.meta?.total ?? 0;

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, delay: 0.5 }}
      className="flex flex-col rounded-2xl border border-border/60"
      style={{
        background: 'rgba(255, 255, 255, 0.6)',
        backdropFilter: 'blur(24px)',
        WebkitBackdropFilter: 'blur(24px)',
      }}
    >
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border/60 px-5 py-4">
        <div className="flex items-center gap-2.5">
          <Activity className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-semibold">Live Activity</h3>
          <LiveDot />
        </div>
        {!isLoading && total > 0 && (
          <span className="rounded-full bg-secondary/60 px-2 py-0.5 text-xs font-medium text-muted-foreground">
            {total} event{total !== 1 ? 's' : ''}
          </span>
        )}
      </div>

      {/* Scrollable content */}
      <div className="relative overflow-y-auto" style={{ maxHeight: 380 }}>
        {/* Vertical timeline line */}
        {!isLoading && logs.length > 0 && (
          <div
            className="absolute top-3 bottom-3"
            style={{
              left: 21,
              width: 1,
              backgroundColor: 'hsl(var(--border))',
            }}
          />
        )}

        {/* Loading */}
        {isLoading && (
          <div>
            {Array.from({ length: 5 }).map((_, i) => (
              <SkeletonItem key={i} index={i} />
            ))}
          </div>
        )}

        {/* Error */}
        {isError && !isLoading && (
          <div className="flex flex-col items-center gap-2 px-4 py-8">
            <p className="text-sm text-muted-foreground">Failed to load activity</p>
            <button
              onClick={() => refetch()}
              className="rounded-md bg-primary/10 px-3 py-1 text-xs font-medium text-primary hover:bg-primary/20"
            >
              Retry
            </button>
          </div>
        )}

        {/* Empty */}
        {!isLoading && !isError && logs.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-3 px-4 py-10">
            <div
              className="flex h-12 w-12 items-center justify-center rounded-full"
              style={{ backgroundColor: 'rgba(34, 197, 94, 0.08)' }}
            >
              <Inbox className="h-[22px] w-[22px]" style={{ color: '#22C55E' }} />
            </div>
            <div className="text-center">
              <p className="text-sm font-medium text-muted-foreground">All quiet</p>
              <p className="mt-0.5 text-xs text-muted-foreground/70">
                No recent activity in the last 7 days.
              </p>
            </div>
          </div>
        )}

        {/* Event list */}
        {!isLoading && !isError && logs.length > 0 && (
          <AnimatePresence initial={false}>
            {logs.map((log) => {
              const Icon = getActionIcon(log.action);
              return (
                <motion.div
                  key={log.id}
                  layout
                  initial={{ opacity: 0, x: -20, height: 0 }}
                  animate={{ opacity: 1, x: 0, height: 'auto' }}
                  exit={{ opacity: 0, x: 20, height: 0 }}
                  transition={{ type: 'spring', damping: 25, stiffness: 300 }}
                  className="overflow-hidden"
                >
                  <div className="relative flex items-start gap-3 px-4 py-3">
                    <div className="mt-1 flex flex-shrink-0 items-center gap-2.5">
                      <TimelineDot action={log.action} />
                      <div className="flex h-7 w-7 items-center justify-center rounded-full bg-muted/60">
                        <Icon className="h-3.5 w-3.5 text-muted-foreground" />
                      </div>
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm capitalize">{formatAction(log)}</p>
                      <p className="mt-0.5 text-xs text-muted-foreground">{timeAgo(log.created_at)}</p>
                    </div>
                  </div>
                </motion.div>
              );
            })}
          </AnimatePresence>
        )}
      </div>
    </motion.div>
  );
}
