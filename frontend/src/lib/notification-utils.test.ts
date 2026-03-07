import { subDays, subMonths } from 'date-fns';
import { describe, expect, it } from 'vitest';
import {
  getNotificationIcon,
  getNotificationIconColor,
  groupNotificationsByDate,
} from './notification-utils';
import type { Notification } from '@/types/models';
import { ShieldAlert, Workflow } from 'lucide-react';

function makeNotification(overrides: Partial<Notification>): Notification {
  return {
    id: overrides.id ?? crypto.randomUUID(),
    title: overrides.title ?? 'Notification',
    body: overrides.body ?? 'Body',
    category: overrides.category ?? 'system',
    priority: overrides.priority ?? 'low',
    read: overrides.read ?? false,
    read_at: overrides.read_at ?? null,
    action_url: overrides.action_url ?? null,
    data: overrides.data ?? null,
    type: overrides.type,
    created_at: overrides.created_at ?? new Date().toISOString(),
  };
}

describe('notification-utils', () => {
  it('groups notifications by Today and Yesterday', () => {
    const notifications = [
      makeNotification({ created_at: new Date().toISOString() }),
      makeNotification({ created_at: subDays(new Date(), 1).toISOString() }),
    ];

    const groups = groupNotificationsByDate(notifications);

    expect(Array.from(groups.keys())).toEqual(['Today', 'Yesterday']);
  });

  it('orders groups from newest to oldest', () => {
    const notifications = [
      makeNotification({ created_at: subMonths(new Date(), 2).toISOString() }),
      makeNotification({ created_at: new Date().toISOString() }),
      makeNotification({ created_at: subDays(new Date(), 10).toISOString() }),
    ];

    const groups = groupNotificationsByDate(notifications);

    expect(Array.from(groups.keys())[0]).toBe('Today');
    expect(Array.from(groups.keys()).at(-1)).toBe('Older');
  });

  it('maps security critical notifications to ShieldAlert', () => {
    const notification = makeNotification({
      category: 'security',
      priority: 'critical',
    });

    expect(getNotificationIcon(notification)).toBe(ShieldAlert);
  });

  it('maps workflow notifications to Workflow icon', () => {
    const notification = makeNotification({
      category: 'workflow',
      priority: 'medium',
    });

    expect(getNotificationIcon(notification)).toBe(Workflow);
  });

  it('maps critical priority to red icon color', () => {
    const notification = makeNotification({
      priority: 'critical',
    });

    expect(getNotificationIconColor(notification)).toBe('text-red-500');
  });
});
