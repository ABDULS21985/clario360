'use client';

import { useState } from 'react';
import Link from 'next/link';
import { motion, AnimatePresence } from 'framer-motion';
import { AlertTriangle, X, ArrowRight } from 'lucide-react';
import { useAuth } from '@/hooks/use-auth';
import { API_ENDPOINTS } from '@/lib/constants';
import { useRealtimeData } from '@/hooks/use-realtime-data';

interface AlertStats {
  by_severity: Array<{ name: string; count: number }>;
  open_count: number;
}

export function CriticalAlertsBanner() {
  const [dismissed, setDismissed] = useState(false);
  const { hasPermission } = useAuth();
  const hasCyber = hasPermission('cyber:read');

  const { data: envelope, isLoading } = useRealtimeData<{ data: AlertStats }>(
    API_ENDPOINTS.CYBER_ALERTS_STATS,
    {
      wsTopics: ['alert.created', 'alert.escalated', 'alert.resolved'],
      enabled: hasCyber,
    },
  );

  const stats = envelope?.data;
  const severityMap = Object.fromEntries(
    (stats?.by_severity ?? []).map((s) => [s.name, s.count]),
  );
  const criticalCount = severityMap['critical'] ?? 0;
  const highCount = severityMap['high'] ?? 0;
  const totalCritical = criticalCount + highCount;
  const isVisible = hasCyber && !isLoading && totalCritical > 0 && !dismissed;

  const pills = [
    criticalCount > 0 && {
      label: 'Critical Alerts',
      count: criticalCount,
      href: '/cyber/alerts?severity=critical',
    },
    highCount > 0 && {
      label: 'High Severity',
      count: highCount,
      href: '/cyber/alerts?severity=high',
    },
  ].filter(Boolean) as { label: string; count: number; href: string }[];

  return (
    <AnimatePresence>
      {isVisible && (
        <motion.div
          initial={{ opacity: 0, height: 0, marginBottom: 0 }}
          animate={{ opacity: 1, height: 'auto', marginBottom: 0 }}
          exit={{ opacity: 0, height: 0, marginBottom: 0 }}
          transition={{ duration: 0.35, ease: 'easeInOut' }}
          style={{ overflow: 'hidden' }}
        >
          <div
            className="rounded-2xl px-5 py-3"
            style={{
              background: 'linear-gradient(135deg, #DC2626, #991B1B)',
              animation: 'critical-pulse 2.5s ease-in-out infinite',
            }}
          >
            <div className="flex items-center justify-between gap-4 flex-wrap">
              {/* Left: icon + message */}
              <div className="flex items-center gap-3 min-w-0">
                <div
                  className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg"
                  style={{ backgroundColor: 'rgba(255, 255, 255, 0.15)' }}
                >
                  <AlertTriangle className="h-[18px] w-[18px] text-white" />
                </div>
                <span className="text-sm font-semibold text-white whitespace-nowrap">
                  {totalCritical} Critical Item{totalCritical !== 1 ? 's' : ''} Require{totalCritical === 1 ? 's' : ''} Attention
                </span>
              </div>

              {/* Center: quick-action pills */}
              <div className="flex items-center gap-2 flex-wrap">
                {pills.map((pill) => (
                  <Link key={pill.href} href={pill.href}>
                    <span className="inline-flex items-center gap-1.5 rounded-full bg-white/[0.18] px-3 py-1 text-xs font-medium text-white transition-colors hover:bg-white/30">
                      {pill.count} {pill.label}
                      <ArrowRight className="h-3 w-3" />
                    </span>
                  </Link>
                ))}
              </div>

              {/* Right: dismiss button */}
              <button
                onClick={() => setDismissed(true)}
                className="flex-shrink-0 rounded-lg p-1.5 transition-colors hover:bg-white/15"
                aria-label="Dismiss critical alerts"
              >
                <X className="h-[18px] w-[18px] text-white" />
              </button>
            </div>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
