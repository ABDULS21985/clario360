import {
  AlertCircle,
  ArrowUpCircle,
  CheckCircle2,
  Clock3,
  Eye,
  Search,
  ShieldAlert,
  XCircle,
} from 'lucide-react';
import type { StatusConfig } from '@/lib/status-configs';
import type { AlertStatus } from '@/types/cyber';

export const ALERT_STATUS_OPTIONS: Array<{ label: string; value: AlertStatus }> = [
  { label: 'New', value: 'new' },
  { label: 'Acknowledged', value: 'acknowledged' },
  { label: 'Investigating', value: 'investigating' },
  { label: 'In Progress', value: 'in_progress' },
  { label: 'Resolved', value: 'resolved' },
  { label: 'Closed', value: 'closed' },
  { label: 'False Positive', value: 'false_positive' },
  { label: 'Escalated', value: 'escalated' },
  { label: 'Merged', value: 'merged' },
];

export const ALERT_STATUS_TRANSITIONS: Record<AlertStatus, AlertStatus[]> = {
  new: ['acknowledged', 'escalated', 'false_positive'],
  acknowledged: ['investigating', 'escalated', 'false_positive'],
  investigating: ['in_progress', 'resolved', 'escalated', 'false_positive'],
  in_progress: ['resolved', 'escalated', 'false_positive'],
  resolved: ['closed', 'investigating'],
  closed: ['investigating'],
  false_positive: ['investigating'],
  escalated: ['investigating', 'in_progress', 'resolved', 'false_positive', 'closed'],
  merged: [],
};

export const ALERT_STATUS_CONFIG: StatusConfig = {
  new: { label: 'New', color: 'red', icon: AlertCircle },
  acknowledged: { label: 'Acknowledged', color: 'blue', icon: Eye },
  investigating: { label: 'Investigating', color: 'yellow', icon: Search },
  in_progress: { label: 'In Progress', color: 'orange', icon: Clock3 },
  resolved: { label: 'Resolved', color: 'green', icon: CheckCircle2 },
  closed: { label: 'Closed', color: 'gray', icon: XCircle },
  false_positive: { label: 'False Positive', color: 'purple', icon: ShieldAlert },
  escalated: { label: 'Escalated', color: 'red', icon: ArrowUpCircle },
  merged: { label: 'Merged', color: 'gray', icon: XCircle },
};

export const ALERT_RULE_TYPE_OPTIONS = [
  { label: 'Sigma', value: 'sigma' },
  { label: 'Threshold', value: 'threshold' },
  { label: 'Correlation', value: 'correlation' },
  { label: 'Anomaly', value: 'anomaly' },
] as const;

export function getAlertStatusLabel(status: AlertStatus): string {
  return ALERT_STATUS_OPTIONS.find((option) => option.value === status)?.label ?? status;
}

export function getAlertStatusVariant(status: AlertStatus): 'default' | 'outline' {
  if (status === 'escalated' || status === 'merged') {
    return 'outline';
  }
  return 'default';
}

export function alertConfidencePercent(score: number | null | undefined): number {
  if (typeof score !== 'number' || Number.isNaN(score)) {
    return 0;
  }
  const normalized = score <= 1 ? score * 100 : score;
  return Math.max(0, Math.min(100, Math.round(normalized)));
}
