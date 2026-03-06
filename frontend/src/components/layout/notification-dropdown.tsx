'use client';

import { useState, useRef, useEffect } from 'react';
import { Bell, Shield, Database, Workflow, Settings, Gavel, AlertTriangle } from 'lucide-react';
import Link from 'next/link';
import { useNotificationStore } from '@/stores/notification-store';
import { apiPut } from '@/lib/api';
import { timeAgo } from '@/lib/utils';
import { cn } from '@/lib/utils';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import type { Notification, NotificationCategory } from '@/types/models';

function getCategoryIcon(category: NotificationCategory) {
  const icons: Record<NotificationCategory, React.ElementType> = {
    security: Shield,
    data: Database,
    workflow: Workflow,
    system: Settings,
    governance: Gavel,
    legal: AlertTriangle,
  };
  return icons[category] ?? Bell;
}

function getPriorityColor(priority: string): string {
  const colors: Record<string, string> = {
    critical: 'text-red-500',
    high: 'text-orange-500',
    medium: 'text-blue-500',
    low: 'text-muted-foreground',
  };
  return colors[priority] ?? 'text-muted-foreground';
}

function NotificationItem({
  notification,
  onRead,
}: {
  notification: Notification;
  onRead: (id: string, url?: string) => void;
}) {
  const Icon = getCategoryIcon(notification.category);
  const iconColor = getPriorityColor(notification.priority);

  return (
    <button
      onClick={() => onRead(notification.id, notification.action_url)}
      className={cn(
        'flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-accent/50',
        !notification.read && 'bg-primary/5',
      )}
    >
      <div className={cn('mt-0.5 shrink-0', iconColor)}>
        <Icon className="h-4 w-4" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          {!notification.read && (
            <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-primary" aria-hidden="true" />
          )}
          <p className={cn('truncate text-sm', !notification.read && 'font-medium')}>
            {notification.title}
          </p>
        </div>
        <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">{notification.body}</p>
        <p className="mt-1 text-xs text-muted-foreground">{timeAgo(notification.created_at)}</p>
      </div>
    </button>
  );
}

export function NotificationDropdown() {
  const [open, setOpen] = useState(false);
  const [markingAll, setMarkingAll] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const { unreadCount, recentNotifications, isInitialized, markAsRead, markAllAsRead } =
    useNotificationStore();

  useEffect(() => {
    if (!open) return;
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [open]);

  const handleRead = async (id: string, url?: string) => {
    try {
      await apiPut(`/api/v1/notifications/${id}/read`);
      markAsRead(id);
    } catch {
      markAsRead(id);
    }
    setOpen(false);
    if (url) window.location.href = url;
  };

  const handleMarkAllRead = async () => {
    setMarkingAll(true);
    try {
      await apiPut('/api/v1/notifications/read-all');
      markAllAsRead();
    } catch {
      markAllAsRead();
    } finally {
      setMarkingAll(false);
    }
  };

  return (
    <div ref={dropdownRef} className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        aria-label={`Notifications${unreadCount > 0 ? ` (${unreadCount} unread)` : ''}`}
        aria-expanded={open}
        className="relative rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
      >
        <Bell className="h-4 w-4" />
        {unreadCount > 0 && (
          <span className="absolute right-1 top-1 flex h-4 w-4 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-destructive-foreground">
            {unreadCount > 9 ? '9+' : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 top-full mt-1 w-80 max-h-[70vh] overflow-hidden rounded-lg border bg-popover shadow-lg flex flex-col z-50">
          {/* Header */}
          <div className="flex items-center justify-between border-b px-4 py-3">
            <h3 className="text-sm font-semibold">Notifications</h3>
            {unreadCount > 0 && (
              <button
                onClick={handleMarkAllRead}
                disabled={markingAll}
                className="text-xs text-primary hover:underline disabled:opacity-50"
              >
                Mark all read
              </button>
            )}
          </div>

          {/* Body */}
          <div className="overflow-y-auto flex-1">
            {!isInitialized ? (
              <div className="p-4">
                <LoadingSkeleton variant="list-item" count={3} />
              </div>
            ) : recentNotifications.length === 0 ? (
              <div className="flex flex-col items-center py-10 text-center px-4">
                <Bell className="h-8 w-8 text-muted-foreground mb-2" />
                <p className="text-sm text-muted-foreground">
                  {"You're all caught up! No new notifications."}
                </p>
              </div>
            ) : (
              <div className="divide-y">
                {recentNotifications.map((n) => (
                  <NotificationItem key={n.id} notification={n} onRead={handleRead} />
                ))}
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="border-t px-4 py-2">
            <Link
              href="/notifications"
              onClick={() => setOpen(false)}
              className="block text-center text-xs text-primary hover:underline"
            >
              View all notifications
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
