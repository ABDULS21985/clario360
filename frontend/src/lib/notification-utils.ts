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
