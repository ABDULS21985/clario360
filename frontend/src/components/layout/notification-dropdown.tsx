'use client';

import { useState, useRef, useEffect } from 'react';
import { Bell, Shield, Database, Workflow, Settings, Gavel, AlertTriangle } from 'lucide-react';
import Link from 'next/link';
import { useNotificationStore } from '@/stores/notification-store';
import { timeAgo } from '@/lib/utils';
import { cn } from '@/lib/utils';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import type { Notification, NotificationCategory } from '@/types/models';
import { useNotificationActions } from '@/hooks/use-notification-actions';
import { LiveIndicator } from '@/components/realtime/live-indicator';
import { isNotificationRead } from '@/lib/notification-utils';

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
  onRead: (id: string, url?: string | null) => void;
}) {
  const Icon = getCategoryIcon(notification.category);
  const iconColor = getPriorityColor(notification.priority);
  const unread = !isNotificationRead(notification);

  return (
    <button
      onClick={() => onRead(notification.id, notification.action_url)}
      className={cn(
        'flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-accent/50',
        unread && 'bg-primary/5',
      )}
      type="button"
    >
      <div className={cn('mt-0.5 shrink-0', iconColor)}>
        <Icon className="h-4 w-4" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          {unread && (
            <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-primary" aria-hidden="true" />
          )}
          <p className={cn('truncate text-sm', unread && 'font-medium')}>
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
  const {
    unreadCount,
    recentNotifications,
    isInitialized,
    connectionStatus,
    reconnectAttempt,
    requestReconnect,
  } = useNotificationStore();
  const { markAsRead, markAllAsRead } = useNotificationActions();

  useEffect(() => {
    if (!open) return;
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false);
    };
    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [open]);

  const handleRead = async (id: string, url?: string | null) => {
    await markAsRead(id);
    setOpen(false);
    if (url) {
      window.location.href = url;
    }
  };

  const handleMarkAllRead = async () => {
    setMarkingAll(true);
    try {
      await markAllAsRead();
    } finally {
      setMarkingAll(false);
    }
  };

  return (
    <div ref={dropdownRef} className="relative">
      <button
        onClick={() => setOpen((value) => !value)}
        aria-label={`Notifications${unreadCount > 0 ? ` (${unreadCount} unread)` : ''}`}
        aria-expanded={open}
        className="relative inline-flex h-11 w-11 items-center justify-center rounded-2xl border border-border/70 bg-white/80 text-muted-foreground shadow-sm transition-all hover:border-primary/20 hover:bg-white hover:text-foreground"
        type="button"
      >
        <Bell className="h-4 w-4" />
        <span className="absolute -right-0.5 -top-0.5">
          <LiveIndicator
            status={connectionStatus}
            attempt={reconnectAttempt}
            onReconnect={requestReconnect}
          />
        </span>
        {unreadCount > 0 && (
          <span className="absolute right-1 top-1 flex h-4 w-4 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-destructive-foreground">
            {unreadCount > 9 ? '9+' : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 top-full z-50 mt-2 flex max-h-[72vh] w-[22rem] flex-col overflow-hidden rounded-[24px] border border-[color:var(--card-border)] bg-[var(--card-bg)] shadow-[var(--card-shadow)] backdrop-blur-xl">
          <div className="flex items-center justify-between border-b border-border/70 px-4 py-3">
            <h3 className="text-sm font-semibold">Notifications</h3>
            {unreadCount > 0 && (
              <button
                onClick={handleMarkAllRead}
                disabled={markingAll}
                className="text-xs font-medium text-primary hover:underline disabled:opacity-50"
                type="button"
              >
                Mark all read
              </button>
            )}
          </div>

          <div className="flex-1 overflow-y-auto">
            {!isInitialized ? (
              <div className="p-4">
                <LoadingSkeleton variant="list-item" count={3} />
              </div>
            ) : recentNotifications.length === 0 ? (
              <div className="flex flex-col items-center px-4 py-10 text-center">
                <Bell className="mb-2 h-8 w-8 text-muted-foreground" />
                <p className="text-sm text-muted-foreground">
                  {"You're all caught up! No new notifications."}
                </p>
              </div>
            ) : (
              <div className="divide-y divide-border/60">
                {recentNotifications.map((notification) => (
                  <NotificationItem
                    key={notification.id}
                    notification={notification}
                    onRead={handleRead}
                  />
                ))}
              </div>
            )}
          </div>

          <div className="border-t border-border/70 px-4 py-3">
            <Link
              href="/notifications"
              onClick={() => setOpen(false)}
              className="block text-center text-xs font-medium text-primary hover:underline"
            >
              View all notifications
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}
