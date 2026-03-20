import {
  Bell,
  Shield,
  ShieldAlert,
  Workflow,
  Database,
  Settings,
  Scale,
  Building2,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import type { Notification } from '@/types/models';
import { isToday, isYesterday, isThisWeek, isThisMonth, parseISO } from 'date-fns';

export function isNotificationRead(notification: Notification): boolean {
  return notification.read || Boolean(notification.read_at);
}

export function getNotificationIcon(notification: Notification): LucideIcon {
  const { category, priority } = notification;
  if (category === 'security') {
    return priority === 'critical' ? ShieldAlert : Shield;
  }
  if (category === 'workflow') return Workflow;
  if (category === 'data') return Database;
  if (category === 'legal') return Scale;
  if (category === 'governance') return Building2;
  if (category === 'system') return Settings;
  return Bell;
}

export function getNotificationIconColor(notification: Notification): string {
  const { priority } = notification;
  if (priority === 'critical') return 'text-red-500';
  if (priority === 'high') return 'text-orange-500';
  if (priority === 'medium') return 'text-blue-500';
  return 'text-gray-500';
}

export function groupNotificationsByDate(
  notifications: Notification[],
): Map<string, Notification[]> {
  const groups = new Map<string, Notification[]>();
  const order = ['Today', 'Yesterday', 'This Week', 'This Month', 'Older'];
  order.forEach((key) => groups.set(key, []));

  for (const notif of notifications) {
    const date = parseISO(notif.created_at);
    if (isToday(date)) {
      groups.get('Today')!.push(notif);
    } else if (isYesterday(date)) {
      groups.get('Yesterday')!.push(notif);
    } else if (isThisWeek(date)) {
      groups.get('This Week')!.push(notif);
    } else if (isThisMonth(date)) {
      groups.get('This Month')!.push(notif);
    } else {
      groups.get('Older')!.push(notif);
    }
  }

  // Remove empty groups
  for (const [key, val] of groups) {
    if (val.length === 0) groups.delete(key);
  }

  return groups;
}

export function getNotificationCategoryLabel(category: string): string {
  const map: Record<string, string> = {
    security: 'Security',
    workflow: 'Workflows',
    data: 'Data',
    governance: 'Governance',
    legal: 'Legal',
    system: 'System',
  };
  return map[category] ?? category;
}

const NOTIFICATION_TYPE_LABELS: Record<string, string> = {
  'alert.created': 'Alert Created',
  'alert.escalated': 'Alert Escalated',
  'remediation.approval_required': 'Remediation Approval',
  'remediation.completed': 'Remediation Completed',
  'remediation.failed': 'Remediation Failed',
  'task.assigned': 'Task Assigned',
  'task.overdue': 'Task Overdue',
  'task.escalated': 'Task Escalated',
  'pipeline.failed': 'Pipeline Failed',
  'pipeline.completed': 'Pipeline Completed',
  'data_quality.issue_detected': 'Data Quality Issue',
  'contradiction.detected': 'Contradiction Detected',
  'contract.expiring': 'Contract Expiring',
  'contract.created': 'Contract Created',
  'meeting.scheduled': 'Meeting Scheduled',
  'meeting.reminder': 'Meeting Reminder',
  'action_item.assigned': 'Action Item Assigned',
  'action_item.overdue': 'Action Item Overdue',
  'minutes.approved': 'Minutes Approved',
  'kpi.threshold_breached': 'KPI Threshold Breached',
  'system.maintenance': 'System Maintenance',
  'security.incident': 'Security Incident',
  'password.expiring': 'Password Expiring',
  'login.anomaly': 'Login Anomaly',
  'analysis.ready': 'Analysis Ready',
  'clause.risk_flagged': 'Clause Risk Flagged',
  'workflow.failed': 'Workflow Failed',
  'workflow.completed': 'Workflow Completed',
  'welcome': 'Welcome',
  'malware.detected': 'Malware Detected',
};

export function getNotificationTypeLabel(type: string | undefined): string {
  if (!type) return 'Notification';
  return (
    NOTIFICATION_TYPE_LABELS[type] ??
    type
      .replace(/[._]/g, ' ')
      .replace(/\b\w/g, (c) => c.toUpperCase())
  );
}
